import type {
  ConditionEvaluation,
  ConditionFieldGroup,
  ConditionFieldOption,
  ConditionFieldOptionValue,
  ConditionGroup,
  ConditionItem,
  ConditionItemResult,
  ConditionLogic,
  ConditionNode,
  ConditionOperator,
  ConditionOperatorMeta,
  ConditionValueGranularity,
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

export const CONDITION_GRANULARITY_LABELS: Record<ConditionValueGranularity, string> = {
  datetime: '日期时间',
  date: '仅日期',
  time: '仅时间',
}

const CONDITION_DATETIME_GRANULARITIES: ConditionValueGranularity[] = ['datetime', 'date', 'time']

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

export function flattenFieldGroups(groups: ConditionFieldGroup[] = []): ConditionFieldOption[] {
  const seen = new Set<string>()
  const fields: ConditionFieldOption[] = []
  for (const group of groups) {
    for (const field of group.fields) {
      if (seen.has(field.value)) continue
      seen.add(field.value)
      fields.push(field)
    }
  }
  return fields
}

export interface ConditionFieldGroupView {
  key: string
  label: string
  description: string
  fields: ConditionFieldOption[]
}

export function groupConditionFields(
  fieldOptions: ConditionFieldOption[] = [],
  fieldGroups: ConditionFieldGroup[] = [],
): ConditionFieldGroupView[] {
  if (!fieldGroups.length) {
    return fieldOptions.length
      ? [{ key: '__all__', label: '', description: '', fields: fieldOptions }]
      : []
  }

  const fieldMap = new Map(fieldOptions.map((field) => [field.value, field]))
  const used = new Set<string>()
  const views: ConditionFieldGroupView[] = []

  for (const group of fieldGroups) {
    const fields: ConditionFieldOption[] = []
    for (const field of group.fields) {
      if (used.has(field.value)) continue
      used.add(field.value)
      fields.push(fieldMap.get(field.value) ?? field)
    }
    if (fields.length) {
      views.push({
        key: group.key,
        label: group.label,
        description: group.description ?? '',
        fields,
      })
    }
  }

  const rest = fieldOptions.filter((field) => !used.has(field.value))
  if (rest.length) {
    views.push({ key: '__rest__', label: '未分组变量', description: '', fields: rest })
  }

  return views
}

export function conditionFieldGroupMap(
  fieldGroups: ConditionFieldGroup[] = [],
): Map<string, ConditionFieldGroup> {
  const map = new Map<string, ConditionFieldGroup>()
  for (const group of fieldGroups) {
    for (const field of group.fields) {
      if (!map.has(field.value)) map.set(field.value, group)
    }
  }
  return map
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

export function conditionGranularityOptions(
  field?: ConditionFieldOption,
): ConditionValueGranularity[] {
  return resolveConditionFieldType(field) === 'datetime' ? CONDITION_DATETIME_GRANULARITIES : []
}

export function resolveGranularityForType(
  type: ConditionValueType,
  granularity?: ConditionValueGranularity,
): ConditionValueGranularity {
  if (type === 'date') return 'date'
  if (type === 'time') return 'time'
  if (type === 'datetime' && (granularity === 'date' || granularity === 'time')) return granularity
  return 'datetime'
}

export function resolveConditionGranularity(
  item: ConditionItem,
  field?: ConditionFieldOption,
): ConditionValueGranularity {
  return resolveGranularityForType(resolveConditionFieldType(field), item.granularity)
}

const CONDITION_LOCKABLE_TYPES: ConditionValueType[] = ['number', 'date', 'datetime', 'time']

export function isConditionFieldLockable(field?: ConditionFieldOption): boolean {
  return CONDITION_LOCKABLE_TYPES.includes(resolveConditionFieldType(field))
}

export function resolveGroupFieldLock(
  group: ConditionGroup,
  fieldOptions: ConditionFieldOption[] = [],
): string {
  const counts = new Map<string, number>()
  for (const child of group.children) {
    if (isConditionGroup(child) || !child.field) continue
    counts.set(child.field, (counts.get(child.field) ?? 0) + 1)
  }
  for (const [field, count] of counts) {
    if (count < 2) continue
    if (!isConditionFieldLockable(fieldOptions.find((entry) => entry.value === field))) continue
    return field
  }
  return ''
}

interface ConditionLockScope {
  active: Map<string, ConditionFieldOption[]>
  known: Set<string>
}

function createConditionLockScope(scopedGroups: ConditionFieldGroup[] = []): ConditionLockScope {
  const active = new Map<string, ConditionFieldOption[]>()
  const known = new Set<string>()
  for (const entry of scopedGroups) {
    known.add(entry.key)
    if (entry.active !== false) active.set(entry.key, entry.fields)
  }
  return { active, known }
}

function regroupRun(items: ConditionItem[]): ConditionNode {
  const [first] = items
  return { ...createConditionGroup(items), logic: first?.logic ?? 'and', groupedByLock: true }
}

function flattenUnlockedLockGroups(
  children: ConditionNode[],
  fieldOptions: ConditionFieldOption[],
): ConditionNode[] | null {
  let changed = false
  const result: ConditionNode[] = []
  for (const child of children) {
    if (
      isConditionGroup(child) &&
      child.groupedByLock &&
      !resolveGroupFieldLock(child, fieldOptions)
    ) {
      changed = true
      child.children.forEach((node, index) => {
        result.push(index === 0 ? { ...node, logic: child.logic } : node)
      })
      continue
    }
    result.push(child)
  }
  return changed ? result : null
}

function conditionNodeHasForeignField(node: ConditionNode, lockedField: string): boolean {
  if (!isConditionGroup(node)) return Boolean(node.field) && node.field !== lockedField
  return node.children.some((child) => conditionNodeHasForeignField(child, lockedField))
}

function regroupLockedChildren(
  children: ConditionNode[],
  lockedField: string,
): ConditionNode[] | null {
  const needsRegroup = children.some((child) =>
    isConditionGroup(child)
      ? conditionNodeHasForeignField(child, lockedField)
      : !child.field || child.field !== lockedField,
  )
  if (!needsRegroup) return null

  const lockedItems = children.filter(
    (child): child is ConditionItem => !isConditionGroup(child) && child.field === lockedField,
  )
  const result: ConditionNode[] = []
  let emittedLock = false
  let run: ConditionItem[] = []
  let runField = ''

  const flushRun = (): void => {
    if (!run.length) return
    if (run.length > 1) {
      result.push(regroupRun(run))
    } else {
      const [single] = run
      if (single) result.push(single)
    }
    run = []
    runField = ''
  }

  for (const child of children) {
    if (isConditionGroup(child)) {
      flushRun()
      result.push(child)
      continue
    }
    if (!child.field) {
      flushRun()
      result.push(child)
      continue
    }
    if (child.field === lockedField) {
      flushRun()
      if (emittedLock) continue
      emittedLock = true
      result.push(regroupRun(lockedItems))
      continue
    }
    if (run.length && runField !== child.field) flushRun()
    runField = child.field
    run.push(child)
  }
  flushRun()
  return result
}

function regroupLockedGroup(
  group: ConditionGroup,
  fieldOptions: ConditionFieldOption[],
  scope: ConditionLockScope,
  depth: number,
  maxDepth: number,
): ConditionGroup {
  let changed = false
  const children: ConditionNode[] = []
  for (const child of group.children) {
    if (!isConditionGroup(child)) {
      children.push(child)
      continue
    }
    const scopeKey = depth === 0 ? child.scope : undefined
    if (scopeKey && scope.known.has(scopeKey) && !scope.active.has(scopeKey)) {
      children.push(child)
      continue
    }
    const childFields = scopeKey ? (scope.active.get(scopeKey) ?? fieldOptions) : fieldOptions
    const next = regroupLockedGroup(child, childFields, scope, depth + 1, maxDepth)
    if (next !== child) changed = true
    children.push(next)
  }

  const flattened = flattenUnlockedLockGroups(children, fieldOptions)
  const base = flattened ?? children
  if (flattened) changed = true

  if (depth < maxDepth) {
    const lockedField = resolveGroupFieldLock({ ...group, children: base }, fieldOptions)
    if (lockedField) {
      const regrouped = regroupLockedChildren(base, lockedField)
      if (regrouped) return { ...group, children: regrouped }
    }
  }

  return changed ? { ...group, children: base } : group
}

export function regroupLockedConditions(
  group: ConditionGroup,
  fieldOptions: ConditionFieldOption[] = [],
  scopedGroups: ConditionFieldGroup[] = [],
  maxDepth: number = CONDITION_MAX_DEPTH,
): ConditionGroup {
  if (!fieldOptions.length) return group
  const scope = createConditionLockScope(scopedGroups)
  return regroupLockedGroup(group, fieldOptions, scope, 0, maxDepth)
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

const CONDITION_DATE_PART = /^(\d{4})-(\d{2})-(\d{2})/
const CONDITION_TIME_PART = /(\d{1,2}):(\d{2})(?::(\d{2}))?/

export const CONDITION_RANGE_SEPARATOR = '~'

export interface ConditionRangeValue {
  start: string
  end: string
}

export function parseConditionRange(value: string): ConditionRangeValue {
  const [rawStart, rawEnd] = String(value ?? '').split(CONDITION_RANGE_SEPARATOR)
  return { start: (rawStart ?? '').trim(), end: (rawEnd ?? '').trim() }
}

export function formatConditionRange(range: ConditionRangeValue): string {
  const start = range.start.trim()
  const end = range.end.trim()
  if (!start && !end) return ''
  return `${start}${CONDITION_RANGE_SEPARATOR}${end}`
}

function toValueText(value: unknown): string {
  if (value === undefined || value === null) return ''
  return String(value).trim()
}

export function conditionDatePart(value: unknown): string {
  const match = CONDITION_DATE_PART.exec(toValueText(value))
  return match ? `${match[1]}-${match[2]}-${match[3]}` : ''
}

export function conditionTimePart(value: unknown): string {
  const match = CONDITION_TIME_PART.exec(toValueText(value))
  if (!match) return ''
  return `${match[1].padStart(2, '0')}:${match[2]}:${match[3] ?? '00'}`
}

function convertValuePart(value: string, granularity: ConditionValueGranularity): string {
  if (!value) return ''
  if (granularity === 'date') return conditionDatePart(value)
  if (granularity === 'time') return conditionTimePart(value)
  return value
}

export function convertConditionValue(
  value: string,
  granularity: ConditionValueGranularity,
): string {
  if (!value) return ''
  if (String(value).includes(CONDITION_RANGE_SEPARATOR)) {
    const range = parseConditionRange(value)
    return formatConditionRange({
      start: convertValuePart(range.start, granularity),
      end: convertValuePart(range.end, granularity),
    })
  }
  return convertValuePart(value, granularity)
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
    if (String(item.value ?? '').includes(CONDITION_RANGE_SEPARATOR)) {
      const range = parseConditionRange(item.value)
      return Boolean(range.start || range.end)
    }
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

export function createScopedConditionGroup(
  scope: string,
  children: ConditionNode[] = [],
): ConditionGroup {
  return {
    id: createConditionId('scope'),
    nodeType: 'group',
    logic: 'and',
    scope,
    children,
  }
}

export function pruneInactiveScopedGroups(
  group: ConditionGroup,
  isActive: (scope: string) => boolean,
): ConditionGroup {
  let changed = false
  const children: ConditionNode[] = []
  for (const child of group.children) {
    if (isConditionGroup(child)) {
      if (child.scope && !isActive(child.scope)) {
        changed = true
        continue
      }
      const next = pruneInactiveScopedGroups(child, isActive)
      if (next !== child) changed = true
      children.push(next)
      continue
    }
    children.push(child)
  }
  return changed ? { ...group, children } : group
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

function collapseSingleChildGroup(
  group: ConditionGroup,
  collapsible: (target: ConditionGroup) => boolean,
): ConditionGroup | ConditionNode {
  if (group.children.length !== 1) return group
  if (!collapsible(group)) return group
  const child = group.children[0]
  return { ...child, logic: group.logic }
}

function removeNodeCollapsing(
  group: ConditionGroup,
  path: number[],
  index: number,
  collapsible: (target: ConditionGroup) => boolean,
  isRootNode: boolean,
): ConditionGroup | ConditionNode {
  if (path.length === 0) {
    const children = group.children.filter((_, current) => current !== index)
    const next: ConditionGroup = { ...group, children }
    return isRootNode ? next : collapseSingleChildGroup(next, collapsible)
  }

  const childIndex = path[0]
  const child = group.children[childIndex]
  if (!child || !isConditionGroup(child)) return group

  const replaced = removeNodeCollapsing(child, path.slice(1), index, collapsible, false)
  const children = group.children.slice()
  children[childIndex] = replaced
  const next: ConditionGroup = { ...group, children }
  return isRootNode ? next : collapseSingleChildGroup(next, collapsible)
}

export function removeNodeWithCollapse(
  group: ConditionGroup,
  path: number[],
  index: number,
  collapsible: (target: ConditionGroup) => boolean,
): ConditionGroup {
  if (path.length === 0) {
    return { ...group, children: group.children.filter((_, current) => current !== index) }
  }
  const next = removeNodeCollapsing(group, path, index, collapsible, true)
  return isConditionGroup(next) ? next : group
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

export function setGroupLevelLogic(group: ConditionGroup, logic: ConditionLogic): ConditionGroup {
  if (group.children.every((child) => child.logic === logic)) return group
  return {
    ...group,
    children: group.children.map((child) => (child.logic === logic ? child : { ...child, logic })),
  }
}

export function setGroupTreeLevelLogicFromFirst(group: ConditionGroup): ConditionGroup {
  const logic = group.children[0]?.logic ?? group.logic
  let changed = false
  const children = group.children.map((child) => {
    let nextChild = isConditionGroup(child) ? setGroupTreeLevelLogicFromFirst(child) : child
    if (nextChild !== child) changed = true
    if (nextChild.logic === logic) return nextChild
    changed = true
    nextChild = { ...nextChild, logic }
    return nextChild
  })
  return changed ? { ...group, children } : group
}

function granularitySuffix(item: ConditionItem, field?: ConditionFieldOption): string {
  if (resolveConditionFieldType(field) !== 'datetime') return ''
  const granularity = resolveConditionGranularity(item, field)
  return granularity === 'date' || granularity === 'time'
    ? `（${CONDITION_GRANULARITY_LABELS[granularity]}）`
    : ''
}

export function conditionItemFieldLabel(item: ConditionItem, field?: ConditionFieldOption): string {
  const base = field ? conditionFieldLabel(field) : item.field
  return `${base}${granularitySuffix(item, field)}`
}

function rangeExpression(name: string, value: string, negated: boolean): string {
  const range = parseConditionRange(value)
  if (range.start && range.end) {
    return `${name} ${negated ? '不在' : '在'} (${range.start} ~ ${range.end}) 内`
  }
  if (range.start) return `${name} ${negated ? '<' : '≥'} ${range.start}`
  if (range.end) return `${name} ${negated ? '>' : '≤'} ${range.end}`
  return ''
}

function itemExpression(item: ConditionItem, fieldMap: Map<string, ConditionFieldOption>): string {
  if (!isConditionItemEffective(item)) return ''
  const operator = OPERATOR_MAP.get(item.operator)
  const field = fieldMap.get(item.field)
  const name = conditionItemFieldLabel(item, field)
  const symbol = operator?.symbol ?? item.operator
  if (conditionOperatorIsMultiValue(item.operator)) {
    if (isConditionTemporalType(resolveConditionFieldType(field))) {
      return rangeExpression(name, item.value, item.operator === 'not_contains')
    }
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

export function isConditionTemporalType(type: ConditionValueType): boolean {
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

function toDayNumber(value: unknown): number {
  const date = conditionDatePart(value)
  if (!date) return Number.NaN
  const [year, month, day] = date.split('-').map(Number)
  return Math.round(Date.UTC(year, month - 1, day) / 86400000)
}

function toSecondsOfDay(value: unknown): number {
  const time = conditionTimePart(value)
  if (!time) return Number.NaN
  const [hours, minutes, seconds] = time.split(':').map(Number)
  return hours * 3600 + minutes * 60 + seconds
}

function toComparableGranular(granularity: ConditionValueGranularity, value: unknown): number {
  if (granularity === 'date') return toDayNumber(value)
  if (granularity === 'time') return toSecondsOfDay(value)
  return toComparableTime(value)
}

function conditionRangeMatch(
  value: string,
  actual: unknown,
  granularity: ConditionValueGranularity,
  negated: boolean,
): boolean {
  const range = parseConditionRange(value)
  const target = toComparableGranular(granularity, actual)
  if (Number.isNaN(target)) return false
  const start = range.start ? toComparableGranular(granularity, range.start) : Number.NaN
  const end = range.end ? toComparableGranular(granularity, range.end) : Number.NaN
  const hasStart = !Number.isNaN(start)
  const hasEnd = !Number.isNaN(end)
  if (!hasStart && !hasEnd) return false
  const inside = (!hasStart || target >= start) && (!hasEnd || target <= end)
  return negated ? !inside : inside
}

function normalizeBoolean(value: unknown): boolean {
  if (typeof value === 'boolean') return value
  const text = String(value).trim().toLowerCase()
  return text === 'true' || text === '1' || text === 'yes' || text === '是'
}

function isBlankValue(value: unknown): boolean {
  return value === undefined || value === null || String(value).trim() === ''
}

function compareEquality(
  type: ConditionValueType,
  actual: unknown,
  expected: string,
  granularity: ConditionValueGranularity,
): boolean {
  if (type === 'number') return toComparableNumber(actual) === toComparableNumber(expected)
  if (type === 'boolean') return normalizeBoolean(actual) === normalizeBoolean(expected)
  if (isConditionTemporalType(type)) {
    const left = toComparableGranular(granularity, actual)
    const right = toComparableGranular(granularity, expected)
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
  const granularity = resolveConditionGranularity(item, field)
  const actual = data[item.field]

  if (item.operator === 'is_null') return isBlankValue(actual)
  if (actual === undefined || actual === null) return false

  switch (item.operator) {
    case 'eq':
      return compareEquality(type, actual, item.value, granularity)
    case 'ne':
      return !compareEquality(type, actual, item.value, granularity)
    case 'contains': {
      if (isConditionTemporalType(type)) {
        return conditionRangeMatch(item.value, actual, granularity, false)
      }
      const values = splitConditionValues(item.value)
      if (values.length === 0) return false
      if (conditionOperatorUsesLike(item.operator, field)) {
        const text = String(actual).toLowerCase()
        return values.some((value) => text.includes(value.toLowerCase()))
      }
      return values.some((value) => compareEquality(type, actual, value, granularity))
    }
    case 'not_contains': {
      if (isConditionTemporalType(type)) {
        return conditionRangeMatch(item.value, actual, granularity, true)
      }
      const values = splitConditionValues(item.value)
      if (values.length === 0) return false
      if (conditionOperatorUsesLike(item.operator, field)) {
        const text = String(actual).toLowerCase()
        return !values.some((value) => text.includes(value.toLowerCase()))
      }
      return !values.some((value) => compareEquality(type, actual, value, granularity))
    }
    case 'gt':
    case 'gte':
    case 'lt':
    case 'lte': {
      const temporal = isConditionTemporalType(type)
      const left = temporal ? toComparableGranular(granularity, actual) : toComparableNumber(actual)
      const right = temporal
        ? toComparableGranular(granularity, item.value)
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

function formatExpectedValue(item: ConditionItem, needsValue?: boolean): string {
  if (needsValue === false) return '—'
  return item.value
}

function collectItemResults(
  group: ConditionGroup,
  data: Record<string, unknown>,
  fieldMap: Map<string, ConditionFieldOption>,
  depth: number,
): ConditionItemResult[] {
  const results: ConditionItemResult[] = []

  group.children.forEach((child, index) => {
    if (isConditionGroup(child)) {
      results.push(...collectItemResults(child, data, fieldMap, depth + 1))
      return
    }
    if (!isConditionItemEffective(child)) return
    const field = fieldMap.get(child.field)
    const operator = OPERATOR_MAP.get(child.operator)
    results.push({
      item: child,
      fieldLabel: conditionItemFieldLabel(child, field),
      operatorLabel: operator?.label ?? child.operator,
      expected: formatExpectedValue(child, operator?.needsValue),
      actual: formatActualValue(data[child.field]),
      passed: evaluateConditionItem(child, data, field),
      depth,
      isFirst: index === 0,
    })
  })

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
