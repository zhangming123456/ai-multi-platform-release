import type {
  ConditionEvaluation,
  ConditionFieldOption,
  ConditionFieldOptionValue,
  ConditionGroup,
  ConditionItem,
  ConditionItemResult,
  ConditionLogic,
  ConditionNode,
  ConditionOperator,
  ConditionOperatorMeta,
  ConditionValueType,
} from './ConditionBuilder.types'

const ALL_VALUE_TYPES: ConditionValueType[] = [
  'string',
  'number',
  'boolean',
  'date',
  'datetime',
  'time',
  'select',
]

const ORDERED_VALUE_TYPES: ConditionValueType[] = ['number', 'date', 'datetime', 'time']

export const CONDITION_OPERATORS: ConditionOperatorMeta[] = [
  { value: 'eq', label: '等于', symbol: '=', types: ALL_VALUE_TYPES, needsValue: true },
  { value: 'ne', label: '不等于', symbol: '≠', types: ALL_VALUE_TYPES, needsValue: true },
  { value: 'gt', label: '大于', symbol: '>', types: ORDERED_VALUE_TYPES, needsValue: true },
  { value: 'gte', label: '大于等于', symbol: '≥', types: ORDERED_VALUE_TYPES, needsValue: true },
  { value: 'lt', label: '小于', symbol: '<', types: ORDERED_VALUE_TYPES, needsValue: true },
  { value: 'lte', label: '小于等于', symbol: '≤', types: ORDERED_VALUE_TYPES, needsValue: true },
  {
    value: 'contains',
    label: '包含',
    symbol: 'IN',
    types: ALL_VALUE_TYPES,
    needsValue: true,
  },
  {
    value: 'not_contains',
    label: '不包含',
    symbol: 'NOT IN',
    types: ALL_VALUE_TYPES,
    needsValue: true,
  },
  { value: 'is_null', label: '为空', symbol: '为空', types: ALL_VALUE_TYPES, needsValue: false },
]

export const CONDITION_LOGIC_SYMBOL: Record<ConditionLogic, string> = { and: '且', or: '或' }

export const CONDITION_VALUE_TYPE_LABELS: Record<ConditionValueType, string> = {
  string: '文本',
  number: '数值',
  boolean: '布尔',
  date: '日期',
  datetime: '日期时间',
  time: '时间',
  select: '枚举',
}

export const CONDITION_BOOLEAN_OPTIONS: ConditionFieldOptionValue[] = [
  { label: '是', value: 'true' },
  { label: '否', value: 'false' },
]

export const CONDITION_MAX_DEPTH = 3

const OPERATOR_MAP = new Map<ConditionOperator, ConditionOperatorMeta>(
  CONDITION_OPERATORS.map((operator) => [operator.value, operator]),
)

export function getConditionOperator(
  operator: ConditionOperator,
): ConditionOperatorMeta | undefined {
  return OPERATOR_MAP.get(operator)
}

export function isConditionGroup(node: ConditionNode): node is ConditionGroup {
  return node.nodeType === 'group'
}

export function conditionFieldLabel(field: ConditionFieldOption): string {
  return field.label ?? field.value
}

export function resolveConditionFieldType(field?: ConditionFieldOption): ConditionValueType {
  return field?.type ?? 'string'
}

export function conditionOperatorsForType(type: ConditionValueType): ConditionOperatorMeta[] {
  return CONDITION_OPERATORS.filter((operator) => operator.types.includes(type))
}

export function conditionOperatorNeedsValue(operator: ConditionOperator): boolean {
  return OPERATOR_MAP.get(operator)?.needsValue ?? true
}

export function resolveConditionOperator(
  item: ConditionItem,
  field?: ConditionFieldOption,
): ConditionOperator {
  const type = resolveConditionFieldType(field)
  const supported = conditionOperatorsForType(type).some(
    (operator) => operator.value === item.operator,
  )
  return supported ? item.operator : 'eq'
}

export function conditionValueAllowedForField(
  value: string,
  field?: ConditionFieldOption,
): boolean {
  if (!value || !field) return true
  const parts = splitConditionValues(value)
  if (parts.length === 0) return true
  if (field.type === 'select') {
    return parts.every((part) => (field.options ?? []).some((option) => option.value === part))
  }
  if (field.type === 'boolean') {
    return parts.every((part) => part === 'true' || part === 'false')
  }
  return true
}

export function splitConditionValues(value: string): string[] {
  return value
    .split(/[,，]/)
    .map((part) => part.trim())
    .filter((part) => part.length > 0)
}

export function conditionOperatorIsMultiValue(operator: ConditionOperator): boolean {
  return operator === 'contains' || operator === 'not_contains'
}

export function conditionFieldHasOptions(field?: ConditionFieldOption): boolean {
  if (!field) return false
  if (field.type === 'boolean') return true
  return field.type === 'select' && (field.options ?? []).length > 0
}

export function conditionOperatorUsesLike(
  operator: ConditionOperator,
  field?: ConditionFieldOption,
): boolean {
  return conditionOperatorIsMultiValue(operator) && resolveConditionFieldType(field) === 'string'
}

export function isConditionItemEffective(item: ConditionItem): boolean {
  if (!item.field) return false
  if (conditionOperatorIsMultiValue(item.operator)) {
    return splitConditionValues(item.value).length > 0
  }
  return true
}

let conditionIdSeed = 0

export function createConditionId(prefix = 'cond'): string {
  conditionIdSeed += 1
  return `${prefix}_${Date.now().toString(36)}_${conditionIdSeed}`
}

export function createConditionItem(field = ''): ConditionItem {
  return {
    id: createConditionId('item'),
    nodeType: 'item',
    logic: 'and',
    field,
    operator: 'eq',
    value: '',
  }
}

export function createConditionGroup(children?: ConditionNode[]): ConditionGroup {
  return {
    id: createConditionId('group'),
    nodeType: 'group',
    logic: 'and',
    children: children ?? [createConditionItem()],
  }
}

export function cloneConditionGroup(group: ConditionGroup): ConditionGroup {
  return {
    ...group,
    children: group.children.map((child) =>
      isConditionGroup(child) ? cloneConditionGroup(child) : { ...child },
    ),
  }
}

export function collectConditionItems(group: ConditionGroup): ConditionItem[] {
  const items: ConditionItem[] = []
  for (const child of group.children) {
    if (isConditionGroup(child)) items.push(...collectConditionItems(child))
    else items.push(child)
  }
  return items
}

export function collectConditionGroups(group: ConditionGroup): ConditionGroup[] {
  const groups: ConditionGroup[] = []
  for (const child of group.children) {
    if (isConditionGroup(child)) {
      groups.push(child, ...collectConditionGroups(child))
    }
  }
  return groups
}

export function updateGroupAt(
  group: ConditionGroup,
  path: number[],
  updater: (target: ConditionGroup) => ConditionGroup,
): ConditionGroup {
  if (path.length === 0) return updater(group)
  const index = path[0]
  const child = group.children[index]
  if (!child || !isConditionGroup(child)) return group
  const next = updateGroupAt(child, path.slice(1), updater)
  if (next === child) return group
  const children = group.children.slice()
  children[index] = next
  return { ...group, children }
}

export function removeNodeAt(group: ConditionGroup, path: number[]): ConditionGroup {
  if (path.length === 0) return group
  const index = path[path.length - 1]
  return updateGroupAt(group, path.slice(0, -1), (target) => ({
    ...target,
    children: target.children.filter((_, current) => current !== index),
  }))
}

export function ungroupNodeAt(group: ConditionGroup, path: number[]): ConditionGroup {
  if (path.length === 0) return group
  const index = path[path.length - 1]
  return updateGroupAt(group, path.slice(0, -1), (target) => {
    const child = target.children[index]
    if (!child || !isConditionGroup(child)) return target
    const children = target.children.slice()
    children.splice(index, 1, ...child.children)
    return { ...target, children }
  })
}

export function patchChildItem(
  group: ConditionGroup,
  index: number,
  patch: Partial<ConditionItem>,
): ConditionGroup {
  const child = group.children[index]
  if (!child || isConditionGroup(child)) return group
  const children = group.children.slice()
  children[index] = { ...child, ...patch }
  return { ...group, children }
}

export function patchChildLogic(
  group: ConditionGroup,
  index: number,
  logic: ConditionLogic,
): ConditionGroup {
  const child = group.children[index]
  if (!child) return group
  const children = group.children.slice()
  children[index] = isConditionGroup(child) ? { ...child, logic } : { ...child, logic }
  return { ...group, children }
}

function itemExpression(item: ConditionItem, fieldMap: Map<string, ConditionFieldOption>): string {
  if (!isConditionItemEffective(item)) return ''
  const operator = OPERATOR_MAP.get(item.operator)
  const field = fieldMap.get(item.field)
  const name = field?.label ?? item.field
  const symbol = operator?.symbol ?? item.operator
  if (conditionOperatorIsMultiValue(item.operator)) {
    const values = splitConditionValues(item.value)
    if (conditionOperatorUsesLike(item.operator, field)) {
      const keyword = item.operator === 'contains' ? '包含' : '不包含'
      return `${name} ${keyword} (${values.join(', ')})`
    }
    return `${name} ${symbol} (${values.join(', ')})`
  }
  return operator?.needsValue === false ? `${name} ${symbol}` : `${name} ${symbol} ${item.value}`
}

function nodeExpression(node: ConditionNode, fieldMap: Map<string, ConditionFieldOption>): string {
  if (!isConditionGroup(node)) return itemExpression(node, fieldMap)
  const inner = groupExpression(node, fieldMap)
  return inner ? `(${inner})` : ''
}

function groupExpression(
  group: ConditionGroup,
  fieldMap: Map<string, ConditionFieldOption>,
): string {
  const segments: string[] = []
  let current: string[] = []

  group.children.forEach((child, index) => {
    if (index > 0 && child.logic === 'or') {
      if (current.length) segments.push(current.join(` ${CONDITION_LOGIC_SYMBOL.and} `))
      current = []
    }
    const text = nodeExpression(child, fieldMap)
    if (text) current.push(text)
  })

  if (current.length) segments.push(current.join(` ${CONDITION_LOGIC_SYMBOL.and} `))
  return segments.join(` ${CONDITION_LOGIC_SYMBOL.or} `)
}

export function buildConditionExpression(
  group: ConditionGroup,
  fieldOptions: ConditionFieldOption[] = [],
): string {
  const fieldMap = new Map(fieldOptions.map((field) => [field.value, field]))
  return groupExpression(group, fieldMap)
}

function toComparableNumber(value: unknown): number {
  return Number(value)
}

function isTemporalType(type: ConditionValueType): boolean {
  return type === 'date' || type === 'datetime' || type === 'time'
}

function toComparableTime(value: unknown): number {
  const match = /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?/.exec(
    String(value).trim(),
  )
  if (!match) return Number.NaN
  return new Date(
    Number(match[1]),
    Number(match[2]) - 1,
    Number(match[3]),
    Number(match[4] ?? 0),
    Number(match[5] ?? 0),
    Number(match[6] ?? 0),
  ).getTime()
}

function toComparableSeconds(value: unknown): number {
  const match = /^(\d{1,2}):(\d{2})(?::(\d{2}))?$/.exec(String(value).trim())
  if (!match) return Number.NaN
  return Number(match[1]) * 3600 + Number(match[2]) * 60 + Number(match[3] ?? 0)
}

function toComparableTemporal(type: ConditionValueType, value: unknown): number {
  return type === 'time' ? toComparableSeconds(value) : toComparableTime(value)
}

function normalizeBoolean(value: unknown): boolean {
  if (typeof value === 'boolean') return value
  const text = String(value).trim().toLowerCase()
  return text === 'true' || text === '1' || text === 'yes' || text === '是'
}

function isBlankValue(value: unknown): boolean {
  return value === undefined || value === null || String(value).trim() === ''
}

function compareEquality(type: ConditionValueType, actual: unknown, expected: string): boolean {
  if (type === 'number') return toComparableNumber(actual) === toComparableNumber(expected)
  if (type === 'boolean') return normalizeBoolean(actual) === normalizeBoolean(expected)
  if (isTemporalType(type)) {
    const left = toComparableTemporal(type, actual)
    const right = toComparableTemporal(type, expected)
    return !Number.isNaN(left) && !Number.isNaN(right) && left === right
  }
  return String(actual) === expected
}

export function evaluateConditionItem(
  item: ConditionItem,
  data: Record<string, unknown>,
  field?: ConditionFieldOption,
): boolean {
  const type = resolveConditionFieldType(field)
  const actual = data[item.field]

  if (item.operator === 'is_null') return isBlankValue(actual)
  if (actual === undefined || actual === null) return false

  switch (item.operator) {
    case 'eq':
      return compareEquality(type, actual, item.value)
    case 'ne':
      return !compareEquality(type, actual, item.value)
    case 'contains': {
      const values = splitConditionValues(item.value)
      if (values.length === 0) return false
      if (conditionOperatorUsesLike(item.operator, field)) {
        const text = String(actual).toLowerCase()
        return values.some((value) => text.includes(value.toLowerCase()))
      }
      return values.some((value) => compareEquality(type, actual, value))
    }
    case 'not_contains': {
      const values = splitConditionValues(item.value)
      if (values.length === 0) return false
      if (conditionOperatorUsesLike(item.operator, field)) {
        const text = String(actual).toLowerCase()
        return !values.some((value) => text.includes(value.toLowerCase()))
      }
      return !values.some((value) => compareEquality(type, actual, value))
    }
    case 'gt':
    case 'gte':
    case 'lt':
    case 'lte': {
      const temporal = isTemporalType(type)
      const left = temporal ? toComparableTemporal(type, actual) : toComparableNumber(actual)
      const right = temporal
        ? toComparableTemporal(type, item.value)
        : toComparableNumber(item.value)
      if (Number.isNaN(left) || Number.isNaN(right)) return false
      if (item.operator === 'gt') return left > right
      if (item.operator === 'gte') return left >= right
      if (item.operator === 'lt') return left < right
      return left <= right
    }
    default:
      return false
  }
}

function formatActualValue(value: unknown): string {
  if (value === undefined) return '未定义'
  if (value === null) return 'null'
  return String(value)
}

function evaluateGroupNode(
  group: ConditionGroup,
  data: Record<string, unknown>,
  fieldMap: Map<string, ConditionFieldOption>,
): boolean | null {
  const segments: boolean[][] = []
  let current: boolean[] = []

  group.children.forEach((child, index) => {
    if (index > 0 && child.logic === 'or') {
      if (current.length) segments.push(current)
      current = []
    }

    let value: boolean | null = null
    if (isConditionGroup(child)) {
      value = evaluateGroupNode(child, data, fieldMap)
    } else if (isConditionItemEffective(child)) {
      value = evaluateConditionItem(child, data, fieldMap.get(child.field))
    }
    if (value !== null) current.push(value)
  })

  if (current.length) segments.push(current)
  if (segments.length === 0) return null
  return segments.some((segment) => segment.every(Boolean))
}

function collectItemResults(
  group: ConditionGroup,
  data: Record<string, unknown>,
  fieldMap: Map<string, ConditionFieldOption>,
  depth: number,
): ConditionItemResult[] {
  const results: ConditionItemResult[] = []

  for (const child of group.children) {
    if (isConditionGroup(child)) {
      results.push(...collectItemResults(child, data, fieldMap, depth + 1))
      continue
    }
    if (!isConditionItemEffective(child)) continue
    const field = fieldMap.get(child.field)
    const operator = OPERATOR_MAP.get(child.operator)
    results.push({
      item: child,
      fieldLabel: field ? conditionFieldLabel(field) : child.field,
      operatorLabel: operator?.label ?? child.operator,
      expected: operator?.needsValue === false ? '—' : child.value,
      actual: formatActualValue(data[child.field]),
      passed: evaluateConditionItem(child, data, field),
      depth,
    })
  }

  return results
}

export function evaluateConditionGroup(
  group: ConditionGroup,
  data: Record<string, unknown>,
  fieldOptions: ConditionFieldOption[] = [],
): ConditionEvaluation {
  const fieldMap = new Map(fieldOptions.map((field) => [field.value, field]))
  const results = collectItemResults(group, data, fieldMap, 1)
  const overall = evaluateGroupNode(group, data, fieldMap)

  return { hasConditions: results.length > 0, passed: overall ?? true, results }
}
