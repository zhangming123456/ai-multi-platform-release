import type {
  ConditionFieldOption,
  ConditionGroup,
  ConditionItem,
  ConditionNode,
  ConditionOperator,
  ConditionSqlResult,
  ConditionValueType,
} from './ConditionBuilder.types'
import {
  isConditionGroup,
  resolveConditionFieldType,
  splitConditionValues,
} from './conditionOperator'

export const CONDITION_SQL_PLACEHOLDER = '?'

const COMPARISON_SQL: Partial<Record<ConditionOperator, string>> = {
  eq: '=',
  ne: '!=',
  gt: '>',
  gte: '>=',
  lt: '<',
  lte: '<=',
}

interface SqlFragment {
  text: string
  params: unknown[]
}

function normalizeBooleanValue(value: string): number {
  const text = value.trim().toLowerCase()
  return text === 'true' || text === '1' || text === 'yes' || text === '是' ? 1 : 0
}

function toSqlParam(value: string, type: ConditionValueType): unknown {
  if (type === 'number') {
    const parsed = Number(value)
    return Number.isNaN(parsed) ? value : parsed
  }
  if (type === 'boolean') return normalizeBooleanValue(value)
  return value
}

function itemSqlFragment(
  item: ConditionItem,
  fieldMap: Map<string, ConditionFieldOption>,
): SqlFragment | null {
  if (!item.field) return null
  const column = item.field
  const type = resolveConditionFieldType(fieldMap.get(item.field))

  if (item.operator === 'is_null') {
    return { text: `(${column} IS NULL OR ${column} = '')`, params: [] }
  }

  const comparison = COMPARISON_SQL[item.operator]
  if (comparison) {
    return {
      text: `${column} ${comparison} ${CONDITION_SQL_PLACEHOLDER}`,
      params: [toSqlParam(item.value, type)],
    }
  }

  if (item.operator === 'contains' || item.operator === 'not_contains') {
    const values = splitConditionValues(item.value)
    if (values.length === 0) return null
    const isContains = item.operator === 'contains'

    if (type === 'string') {
      const keyword = isContains ? 'LIKE' : 'NOT LIKE'
      const joiner = isContains ? ' OR ' : ' AND '
      const parts = values.map(() => `${column} ${keyword} ${CONDITION_SQL_PLACEHOLDER}`)
      return {
        text: parts.length === 1 ? parts[0] : `(${parts.join(joiner)})`,
        params: values.map((value) => `%${value}%`),
      }
    }

    const keyword = isContains ? 'IN' : 'NOT IN'
    const placeholders = values.map(() => CONDITION_SQL_PLACEHOLDER).join(', ')
    return {
      text: `${column} ${keyword} (${placeholders})`,
      params: values.map((value) => toSqlParam(value, type)),
    }
  }

  return null
}

function mergeSqlFragments(fragments: SqlFragment[], joiner: string): SqlFragment {
  return {
    text: fragments.map((fragment) => fragment.text).join(joiner),
    params: fragments.flatMap((fragment) => fragment.params),
  }
}

function nodeSqlFragment(
  node: ConditionNode,
  fieldMap: Map<string, ConditionFieldOption>,
): SqlFragment | null {
  if (!isConditionGroup(node)) return itemSqlFragment(node, fieldMap)
  const inner = groupSqlFragment(node, fieldMap)
  return inner ? { text: `(${inner.text})`, params: inner.params } : null
}

function groupSqlFragment(
  group: ConditionGroup,
  fieldMap: Map<string, ConditionFieldOption>,
): SqlFragment | null {
  const segments: SqlFragment[] = []
  let current: SqlFragment[] = []

  group.children.forEach((child, index) => {
    if (index > 0 && child.logic === 'or') {
      if (current.length) segments.push(mergeSqlFragments(current, ' AND '))
      current = []
    }
    const fragment = nodeSqlFragment(child, fieldMap)
    if (fragment) current.push(fragment)
  })

  if (current.length) segments.push(mergeSqlFragments(current, ' AND '))
  if (segments.length === 0) return null
  return mergeSqlFragments(segments, ' OR ')
}

export function buildConditionWhere(
  group: ConditionGroup,
  fieldOptions: ConditionFieldOption[] = [],
): ConditionSqlResult {
  const fieldMap = new Map(fieldOptions.map((field) => [field.value, field]))
  const fragment = groupSqlFragment(group, fieldMap)

  if (!fragment) return { hasConditions: false, where: '', params: [] }
  return { hasConditions: true, where: fragment.text, params: fragment.params }
}

export function formatConditionSql(result: ConditionSqlResult): string {
  return result.hasConditions ? `WHERE ${result.where}` : ''
}
