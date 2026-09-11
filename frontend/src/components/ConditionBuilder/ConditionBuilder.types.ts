export type ConditionLogic = 'and' | 'or'

export type ConditionValueType =
  'string' | 'number' | 'boolean' | 'date' | 'datetime' | 'time' | 'select'

export type ConditionOperator =
  'eq' | 'ne' | 'gt' | 'gte' | 'lt' | 'lte' | 'contains' | 'not_contains' | 'is_null'

export type ConditionNodeType = 'item' | 'group'

export interface ConditionFieldOptionValue {
  label: string
  value: string
}

export interface ConditionFieldOption {
  value: string
  label?: string
  type?: ConditionValueType
  description?: string
  options?: ConditionFieldOptionValue[]
  queryable?: boolean
}

export interface ConditionFieldGroup {
  key: string
  label: string
  description?: string
  active?: boolean
  fields: ConditionFieldOption[]
}

export interface ConditionItem {
  id: string
  nodeType: 'item'
  logic: ConditionLogic
  field: string
  operator: ConditionOperator
  value: string
}

export interface ConditionGroup {
  id: string
  nodeType: 'group'
  logic: ConditionLogic
  scope?: string
  children: ConditionNode[]
}

export type ConditionNode = ConditionItem | ConditionGroup

export interface ConditionOperatorMeta {
  value: ConditionOperator
  label: string
  symbol: string
  types: ConditionValueType[]
  needsValue: boolean
}

export interface ConditionItemResult {
  item: ConditionItem
  fieldLabel: string
  operatorLabel: string
  expected: string
  actual: string
  passed: boolean
  depth: number
}

export interface ConditionEvaluation {
  hasConditions: boolean
  passed: boolean
  results: ConditionItemResult[]
}

export interface ConditionSqlResult {
  hasConditions: boolean
  where: string
  params: unknown[]
  skipped: string[]
}

export interface ConditionSqlColumnMap {
  variable: string
  table: string
  column: string
  queryable: boolean
}

export interface ConditionSqlOptions {
  table?: string
  columns?: string[]
  orderBy?: string
  limit?: number
  columnMap?: ConditionSqlColumnMap[]
}

export interface ConditionSqlStatement {
  hasConditions: boolean
  sql: string
  params: unknown[]
  skipped: string[]
}

export type ConditionCommand =
  | { type: 'set-logic'; path: number[]; logic: ConditionLogic }
  | { type: 'add-item'; path: number[] }
  | { type: 'add-group'; path: number[] }
  | { type: 'clear-group'; path: number[] }
  | { type: 'remove-group'; path: number[] }
  | { type: 'ungroup-group'; path: number[] }
  | { type: 'update-item'; path: number[]; index: number; patch: Partial<ConditionItem> }
  | { type: 'remove-item'; path: number[]; index: number }
  | { type: 'wrap-item'; path: number[]; index: number }

export interface ConditionBuilderProps {
  modelValue?: ConditionGroup
  fieldOptions?: ConditionFieldOption[]
  fieldGroups?: ConditionFieldGroup[]
  scopedGroups?: ConditionFieldGroup[]
  ruleFieldOptions?: ConditionFieldOption[]
  rules?: ConditionRule[]
  disabled?: boolean
  maxItems?: number
  maxDepth?: number
  logicEditable?: boolean
  addText?: string
  addGroupText?: string
  clearText?: string
  emptyText?: string
  ungroupText?: string
}

export type ConditionBuilderEmits = {
  (e: 'update:modelValue', value: ConditionGroup): void
  (e: 'change', value: ConditionGroup): void
}

export interface ConditionGroupEditorProps {
  group: ConditionGroup
  path: number[]
  depth: number
  fieldOptions: ConditionFieldOption[]
  fieldGroups?: ConditionFieldGroup[]
  scopedGroups?: ConditionFieldGroup[]
  groupTitle?: string
  lockGroup?: boolean
  flat?: boolean
  ruleContext?: ConditionRuleContext
  disabled?: boolean
  logicEditable?: boolean
  maxDepth?: number
  maxItems?: number
  isRoot?: boolean
  addText?: string
  addGroupText?: string
  clearText?: string
  emptyText?: string
  ungroupText?: string
}

export type ConditionGroupEditorEmits = {
  (e: 'command', command: ConditionCommand): void
}

export interface ConditionItemRowProps {
  item: ConditionItem
  path: number[]
  index: number
  fieldOptions: ConditionFieldOption[]
  fieldGroups?: ConditionFieldGroup[]
  ruleContext?: ConditionRuleContext
  disabled?: boolean
  logicEditable?: boolean
}

export type ConditionItemRowEmits = {
  (e: 'command', command: ConditionCommand): void
}

export interface ConditionConnectorProps {
  index: number
  logic: ConditionLogic
  disabled?: boolean
  editable?: boolean
}

export type ConditionConnectorEmits = {
  (e: 'update:logic', value: ConditionLogic): void
}

export type ConditionRuleType = 'mutual_exclusive' | 'prerequisite' | 'linkage'

export interface ConditionRuleSelector {
  field: string
  operator?: ConditionOperator
  value?: string
}

export interface ConditionRuleBase {
  id: string
  name: string
  description?: string
  message?: string
  isActive?: boolean
}

export interface ConditionMutualExclusiveRule extends ConditionRuleBase {
  type: 'mutual_exclusive'
  when: ConditionRuleSelector
  targets: ConditionRuleSelector[]
}

export interface ConditionPrerequisiteRule extends ConditionRuleBase {
  type: 'prerequisite'
  when: ConditionRuleSelector
  targets: ConditionRuleSelector[]
}

export interface ConditionLinkageRule extends ConditionRuleBase {
  type: 'linkage'
  when: ConditionRuleSelector
  field: string
  operator: ConditionOperator
  allowedValues: string[]
}

export type ConditionRule =
  ConditionMutualExclusiveRule | ConditionPrerequisiteRule | ConditionLinkageRule

export interface ConditionRuleFieldLock {
  field: string
  ruleId: string
  ruleName: string
  reason: string
}

export interface ConditionRuleValueLimit {
  field: string
  operator: ConditionOperator
  value: string
  ruleId: string
  ruleName: string
  reason: string
}

export interface ConditionRuleViolation {
  ruleId: string
  ruleName: string
  message: string
  itemIds: string[]
}

export type ConditionRuleEffect = 'effective' | 'partial' | 'ineffective'

export interface ConditionRuleFieldEffect {
  field: string
  ruleId: string
  ruleName: string
  effect: ConditionRuleEffect
}

export interface ConditionRuleContext {
  activeRuleIds: string[]
  ruleEffects: Record<string, ConditionRuleEffect>
  fieldEffects: ConditionRuleFieldEffect[]
  fieldLocks: ConditionRuleFieldLock[]
  valueLimits: ConditionRuleValueLimit[]
  violations: ConditionRuleViolation[]
}

export type ConditionRuleState = 'idle' | 'active' | 'partial' | 'ineffective' | 'violation'

export interface ConditionRuleDraftTarget {
  field: string
  operator: ConditionOperator | ''
  value: string
}

export interface ConditionRuleDraft {
  id: string
  type: ConditionRuleType
  name: string
  description: string
  message: string
  isActive: boolean
  whenField: string
  whenOperator: ConditionOperator | ''
  whenValue: string
  targets: ConditionRuleDraftTarget[]
  linkageField: string
  linkageOperator: ConditionOperator | ''
  linkageValues: string[]
}

export interface ConditionRuleEditorProps {
  rules?: ConditionRule[]
  fieldOptions?: ConditionFieldOption[]
  fieldGroups?: ConditionFieldGroup[]
  fieldEffects?: ConditionRuleFieldEffect[]
  ruleStates?: Record<string, ConditionRuleState>
  disabled?: boolean
}

export type ConditionRuleEditorEmits = {
  (e: 'update:rules', value: ConditionRule[]): void
}

export interface ConditionValueControlProps {
  modelValue: string
  field?: ConditionFieldOption
  operator?: ConditionOperator
  multiple?: boolean
  creatable?: boolean
  disabled?: boolean
  disabledValues?: string[]
}

export type ConditionValueControlEmits = {
  (e: 'update:modelValue', value: string): void
}

export interface ConditionSchemaField {
  name: string
  type: string
  desc: string
}

export interface ConditionOperatorDoc {
  value: string
  label: string
  symbol: string
  types: string
  needsValue: boolean
  sql: string
}

export type ConditionDocTabKey = 'schema' | 'operator' | 'condition' | 'rule' | 'sql'

export interface ConditionDocDrawerProps {
  visible: boolean
  activeTab: ConditionDocTabKey
  schemaFields?: ConditionSchemaField[]
  operatorDocs?: ConditionOperatorDoc[]
}

export type ConditionDocDrawerEmits = {
  (e: 'update:visible', value: boolean): void
  (e: 'update:activeTab', value: ConditionDocTabKey): void
}

export interface ConditionJsonInputProps {
  modelValue: string
  fieldOptions?: ConditionFieldOption[]
  fieldGroups?: ConditionFieldGroup[]
  placeholder?: string
  height?: number
  error?: boolean
}

export type ConditionJsonInputEmits = {
  (e: 'update:modelValue', value: string): void
}
