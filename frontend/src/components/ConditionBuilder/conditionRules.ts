import type {
  ConditionFieldOption,
  ConditionGroup,
  ConditionItem,
  ConditionOperator,
  ConditionRule,
  ConditionRuleBase,
  ConditionRuleContext,
  ConditionRuleDraft,
  ConditionRuleEffect,
  ConditionRuleFieldEffect,
  ConditionRuleFieldLock,
  ConditionRuleSelector,
  ConditionRuleState,
  ConditionRuleType,
  ConditionRuleValueLimit,
  ConditionRuleViolation,
} from './ConditionBuilder.types'
import {
  collectConditionItems,
  conditionFieldLabel,
  evaluateConditionItem,
  getConditionOperator,
  splitConditionValues,
} from './conditionOperator'

export const CONDITION_RULE_TYPE_LABELS: Record<ConditionRuleType, string> = {
  mutual_exclusive: '互斥',
  prerequisite: '先决',
  linkage: '联动',
}

export const CONDITION_RULE_STATE_LABELS: Record<ConditionRuleState, string> = {
  idle: '未触发',
  active: '命中',
  partial: '部分未生效',
  ineffective: '未生效',
  violation: '冲突',
}

export const CONDITION_RULE_WARNING_LABELS: Record<
  Exclude<ConditionRuleEffect, 'effective'>,
  string
> = {
  partial: '激活部分未生效',
  ineffective: '激活未生效',
}

export function conditionRuleWarning(effect: ConditionRuleEffect): string {
  return effect === 'effective' ? '' : CONDITION_RULE_WARNING_LABELS[effect]
}

let conditionRuleSeed = 0

export function createConditionRuleId(prefix = 'rule'): string {
  conditionRuleSeed += 1
  return `${prefix}_${Date.now().toString(36)}_${conditionRuleSeed}`
}

function selectorValues(selector: ConditionRuleSelector): string[] {
  return selector.value ? splitConditionValues(selector.value) : []
}

function selectorMatchesItem(selector: ConditionRuleSelector, item: ConditionItem): boolean {
  if (item.field !== selector.field) return false
  if (selector.operator && item.operator !== selector.operator) return false
  const expected = selectorValues(selector)
  if (expected.length === 0) return true
  const actual = splitConditionValues(item.value)
  return expected.some((value) => actual.includes(value))
}

function matchingItems(items: ConditionItem[], selector: ConditionRuleSelector): ConditionItem[] {
  return items.filter((item) => selectorMatchesItem(selector, item))
}

function matchedItemIds(items: ConditionItem[], selector: ConditionRuleSelector): string[] {
  return matchingItems(items, selector).map((item) => item.id)
}

export function resolveConditionRules(
  group: ConditionGroup,
  rules: ConditionRule[] = [],
  fieldOptions: ConditionFieldOption[] = [],
  availableFields?: string[],
): ConditionRuleContext {
  const items = collectConditionItems(group)
  const fieldMap = new Map(fieldOptions.map((field) => [field.value, field]))
  const available = new Set(availableFields ?? fieldOptions.map((field) => field.value))
  const activeRuleIds: string[] = []
  const ruleEffects: Record<string, ConditionRuleEffect> = {}
  const fieldEffects: ConditionRuleFieldEffect[] = []
  const fieldLocks: ConditionRuleFieldLock[] = []
  const valueLimits: ConditionRuleValueLimit[] = []
  const violations: ConditionRuleViolation[] = []

  const markEffect = (rule: ConditionRule, fields: string[]): void => {
    const targets = Array.from(new Set(fields.filter(Boolean)))
    const missed = targets.filter((field) => !available.has(field))
    const effect: ConditionRuleEffect =
      targets.length === 0 || missed.length === 0
        ? 'effective'
        : missed.length === targets.length
          ? 'ineffective'
          : 'partial'
    ruleEffects[rule.id] = effect
    if (effect === 'effective') return
    for (const field of targets) {
      fieldEffects.push({ field, ruleId: rule.id, ruleName: rule.name, effect })
    }
  }

  for (const rule of rules) {
    if (rule.isActive === false) continue

    if (rule.type === 'mutual_exclusive') {
      const members: ConditionRuleSelector[] = [rule.when, ...rule.targets]
      const matched = members.filter((selector) => matchingItems(items, selector).length > 0)
      if (matched.length === 0) continue

      activeRuleIds.push(rule.id)
      markEffect(
        rule,
        members.filter((selector) => !matched.includes(selector)).map((selector) => selector.field),
      )

      if (matched.length > 1) {
        violations.push({
          ruleId: rule.id,
          ruleName: rule.name,
          message: rule.message ?? `「${rule.name}」中的互斥条件不可同时存在`,
          itemIds: matched.flatMap((selector) => matchedItemIds(items, selector)),
        })
      }

      for (const selector of members) {
        if (matched.includes(selector)) continue
        const values = selectorValues(selector)
        if (values.length > 0) {
          valueLimits.push({
            field: selector.field,
            operator: 'ne',
            value: values.join(', '),
            ruleId: rule.id,
            ruleName: rule.name,
            reason: rule.message ?? `与「${rule.name}」互斥`,
          })
        } else {
          fieldLocks.push({
            field: selector.field,
            ruleId: rule.id,
            ruleName: rule.name,
            reason: rule.message ?? `与「${rule.name}」互斥`,
          })
        }
      }
      continue
    }

    if (rule.type === 'prerequisite') {
      const satisfied = matchingItems(items, rule.when).length > 0
      if (satisfied) {
        activeRuleIds.push(rule.id)
        markEffect(
          rule,
          rule.targets.map((selector) => selector.field),
        )
        continue
      }

      const matchedTargets = rule.targets.filter(
        (selector) => matchingItems(items, selector).length > 0,
      )

      if (matchedTargets.length === 0) {
        for (const selector of rule.targets) {
          fieldLocks.push({
            field: selector.field,
            ruleId: rule.id,
            ruleName: rule.name,
            reason: rule.message ?? `需先满足「${rule.name}」的前置条件`,
          })
        }
        continue
      }

      activeRuleIds.push(rule.id)
      violations.push({
        ruleId: rule.id,
        ruleName: rule.name,
        message: rule.message ?? `「${rule.name}」缺少前置条件`,
        itemIds: matchedTargets.flatMap((selector) => matchedItemIds(items, selector)),
      })
      continue
    }

    const triggered = matchingItems(items, rule.when).length > 0
    if (!triggered) continue

    activeRuleIds.push(rule.id)
    markEffect(rule, [rule.field])

    const limit: ConditionRuleValueLimit = {
      field: rule.field,
      operator: rule.operator,
      value: rule.allowedValues.join(', '),
      ruleId: rule.id,
      ruleName: rule.name,
      reason: rule.message ?? `受「${rule.name}」联动限制`,
    }
    valueLimits.push(limit)

    const targetField = fieldMap.get(rule.field)
    const conflicted = items.filter(
      (item) =>
        item.field === rule.field && !conditionRuleValueSatisfied(limit, item.value, targetField),
    )
    if (conflicted.length > 0) {
      violations.push({
        ruleId: rule.id,
        ruleName: rule.name,
        message: rule.message ?? `「${rule.name}」限定的约束不满足当前值`,
        itemIds: conflicted.map((item) => item.id),
      })
    }
  }

  return { activeRuleIds, ruleEffects, fieldEffects, fieldLocks, valueLimits, violations }
}

function conditionRuleValueSatisfied(
  limit: ConditionRuleValueLimit,
  value: string,
  field?: ConditionFieldOption,
): boolean {
  const dataset: Record<string, unknown> = { [limit.field]: value }
  const evaluate = (operator: ConditionOperator, expected: string): boolean =>
    evaluateConditionItem(
      {
        id: 'condition-rule-limit',
        nodeType: 'item',
        logic: 'and',
        field: limit.field,
        operator,
        value: expected,
      },
      dataset,
      field,
    )

  if (limit.operator === 'is_null') return evaluate('is_null', '')

  const values = splitConditionValues(limit.value)
  if (values.length === 0) return true
  if (limit.operator === 'eq') return values.some((entry) => evaluate('eq', entry))
  if (limit.operator === 'ne') return !values.some((entry) => evaluate('eq', entry))
  if (
    limit.operator === 'gt' ||
    limit.operator === 'gte' ||
    limit.operator === 'lt' ||
    limit.operator === 'lte'
  ) {
    return evaluate(limit.operator, values[0])
  }
  return evaluate(limit.operator, limit.value)
}

export function conditionRuleFieldLockMap(
  context?: ConditionRuleContext,
): Map<string, ConditionRuleFieldLock> {
  const map = new Map<string, ConditionRuleFieldLock>()
  for (const lock of context?.fieldLocks ?? []) {
    if (!map.has(lock.field)) map.set(lock.field, lock)
  }
  return map
}

export function conditionRuleValueLimits(
  context: ConditionRuleContext | undefined,
  field: string,
): ConditionRuleValueLimit[] {
  return (context?.valueLimits ?? []).filter((limit) => limit.field === field)
}

export function conditionRuleValueDisabled(
  limits: ConditionRuleValueLimit[],
  value: string,
  field?: ConditionFieldOption,
): boolean {
  return limits.some((limit) => !conditionRuleValueSatisfied(limit, value, field))
}

export function conditionRuleValueReason(
  limits: ConditionRuleValueLimit[],
  value: string,
  field?: ConditionFieldOption,
): string {
  return limits.find((limit) => !conditionRuleValueSatisfied(limit, value, field))?.reason ?? ''
}

export const CONDITION_RULE_TYPE_OPTIONS: { value: ConditionRuleType; label: string }[] = [
  { value: 'mutual_exclusive', label: CONDITION_RULE_TYPE_LABELS.mutual_exclusive },
  { value: 'prerequisite', label: CONDITION_RULE_TYPE_LABELS.prerequisite },
  { value: 'linkage', label: CONDITION_RULE_TYPE_LABELS.linkage },
]

function conditionFieldValueText(value: string, field?: ConditionFieldOption): string {
  const values = splitConditionValues(value)
  if (values.length === 0) return value
  return values
    .map((entry) => field?.options?.find((option) => option.value === entry)?.label ?? entry)
    .join(', ')
}

export function conditionRuleSelectorText(
  selector: ConditionRuleSelector,
  fieldOptions: ConditionFieldOption[] = [],
): string {
  const field = fieldOptions.find((option) => option.value === selector.field)
  const parts = [field ? conditionFieldLabel(field) : selector.field]
  const operator = selector.operator ? getConditionOperator(selector.operator) : undefined
  if (operator) parts.push(operator.label)
  if (selector.value) parts.push(conditionFieldValueText(selector.value, field))
  return parts.join(' ')
}

export function conditionRuleSummary(
  rule: ConditionRule,
  fieldOptions: ConditionFieldOption[] = [],
): string {
  const when = conditionRuleSelectorText(rule.when, fieldOptions)

  if (rule.type === 'linkage') {
    const field = fieldOptions.find((option) => option.value === rule.field)
    const name = field ? conditionFieldLabel(field) : rule.field
    const values = rule.allowedValues
      .map((value) => field?.options?.find((option) => option.value === value)?.label ?? value)
      .join(' / ')
    const operator = getConditionOperator(rule.operator)
    const hasTrigger = Boolean(rule.when.operator || rule.when.value)
    const parts = [hasTrigger ? `当 ${when} 时，${name}` : name]
    if (operator) parts.push(operator.label)
    if ((operator?.needsValue ?? true) && values) parts.push(values)
    return parts.join(' ')
  }

  const targets = rule.targets
    .map((selector) => conditionRuleSelectorText(selector, fieldOptions))
    .join(' / ')

  if (rule.type === 'prerequisite') return `配置 ${targets} 前需先有 ${when}`
  return `当 ${when} 时，不可同时使用 ${targets}`
}

export function conditionRuleState(
  context: ConditionRuleContext,
  ruleId: string,
): ConditionRuleState {
  if (context.violations.some((violation) => violation.ruleId === ruleId)) return 'violation'
  const effect = context.ruleEffects[ruleId]
  if (effect === 'ineffective') return 'ineffective'
  if (effect === 'partial') return 'partial'
  if (context.activeRuleIds.includes(ruleId)) return 'active'
  return 'idle'
}

export function conditionRuleFieldEffectIndex(
  effects: ConditionRuleFieldEffect[] = [],
): Map<string, ConditionRuleFieldEffect> {
  const map = new Map<string, ConditionRuleFieldEffect>()
  for (const effect of effects) {
    if (!map.has(effect.field)) map.set(effect.field, effect)
  }
  return map
}

export function conditionRuleFieldEffectMap(
  context?: ConditionRuleContext,
): Map<string, ConditionRuleFieldEffect> {
  return conditionRuleFieldEffectIndex(context?.fieldEffects ?? [])
}

export function createConditionRuleDraft(
  type: ConditionRuleType = 'mutual_exclusive',
): ConditionRuleDraft {
  return {
    id: '',
    type,
    name: '',
    description: '',
    message: '',
    isActive: true,
    whenField: '',
    whenOperator: '',
    whenValue: '',
    targets: [{ field: '', operator: '', value: '' }],
    linkageField: '',
    linkageOperator: '',
    linkageValues: [],
  }
}

export function conditionRuleToDraft(rule: ConditionRule): ConditionRuleDraft {
  const draft: ConditionRuleDraft = {
    ...createConditionRuleDraft(rule.type),
    id: rule.id,
    name: rule.name,
    description: rule.description ?? '',
    message: rule.message ?? '',
    isActive: rule.isActive !== false,
    whenField: rule.when.field,
    whenOperator: rule.when.operator ?? '',
    whenValue: rule.when.value ?? '',
  }

  if (rule.type === 'linkage') {
    return {
      ...draft,
      linkageField: rule.field,
      linkageOperator: rule.operator,
      linkageValues: [...rule.allowedValues],
    }
  }

  return {
    ...draft,
    targets: rule.targets.length
      ? rule.targets.map((selector) => ({
          field: selector.field,
          operator: selector.operator ?? '',
          value: selector.value ?? '',
        }))
      : [{ field: '', operator: '', value: '' }],
  }
}

function draftSelector(
  field: string,
  operator: ConditionOperator | '',
  value: string,
): ConditionRuleSelector {
  const selector: ConditionRuleSelector = { field }
  if (operator) selector.operator = operator
  if (value) selector.value = value
  return selector
}

export function conditionRuleFromDraft(draft: ConditionRuleDraft): ConditionRule {
  const base: ConditionRuleBase = {
    id: draft.id || createConditionRuleId(),
    name: draft.name.trim(),
    isActive: draft.isActive,
  }
  if (draft.description.trim()) base.description = draft.description.trim()
  if (draft.message.trim()) base.message = draft.message.trim()

  const when = draftSelector(draft.whenField, draft.whenOperator, draft.whenValue.trim())

  if (draft.type === 'linkage') {
    return {
      ...base,
      type: 'linkage',
      when,
      field: draft.linkageField,
      operator: draft.linkageOperator || 'eq',
      allowedValues: [...draft.linkageValues],
    }
  }

  const targets = draft.targets.map((target) =>
    draftSelector(target.field, target.operator, target.value.trim()),
  )

  if (draft.type === 'prerequisite') {
    return { ...base, type: 'prerequisite', when, targets }
  }

  return { ...base, type: 'mutual_exclusive', when, targets }
}
