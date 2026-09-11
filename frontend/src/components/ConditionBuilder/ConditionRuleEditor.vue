<template>
  <div class="cre">
    <div class="cre-toolbar">
      <a-button
        type="text"
        size="mini"
        class="cre-toolbar__add"
        :disabled="disabled"
        @click="openCreate"
      >
        <template #icon><IconPlus :size="13" /></template>
        新增规则
      </a-button>
    </div>

    <div v-if="rules.length" class="cre-list">
      <div
        v-for="rule in rules"
        :key="rule.id"
        class="cre-item"
        :class="{ 'cre-item--off': rule.isActive === false }"
      >
        <div class="cre-item__head">
          <a-tag :color="typeColor(rule.type)" size="small" class="!m-0">
            {{ typeLabel(rule.type) }}
          </a-tag>
          <span class="cre-item__name">{{ rule.name }}</span>
          <span
            v-if="ruleStates[rule.id]"
            class="cre-item__state"
            :class="`cre-item__state--${ruleStates[rule.id]}`"
          >
            {{ stateLabel(ruleStates[rule.id]) }}
          </span>
          <div class="cre-item__spacer" />
          <a-switch
            :model-value="rule.isActive !== false"
            size="small"
            :disabled="disabled"
            @change="(value: boolean | string | number) => toggleActive(rule, Boolean(value))"
          />
          <a-button
            type="text"
            size="mini"
            class="cre-item__action"
            :disabled="disabled"
            @click="openEdit(rule)"
          >
            <template #icon><IconEdit :size="13" /></template>
          </a-button>
          <a-button
            type="text"
            size="mini"
            class="cre-item__action"
            status="danger"
            :disabled="disabled"
            @click="removeRule(rule)"
          >
            <template #icon><IconDelete :size="13" /></template>
          </a-button>
        </div>
        <div class="cre-item__summary">{{ summaryOf(rule) }}</div>
        <div v-if="rule.description" class="cre-item__desc">{{ rule.description }}</div>
      </div>
    </div>
    <div v-else class="cre-empty">暂无规则，点击「新增规则」创建</div>

    <a-modal
      v-model:visible="formVisible"
      :title="draft.id ? '编辑规则' : '新增规则'"
      :width="660"
      :on-before-ok="handleBeforeOk"
      ok-text="保存"
      cancel-text="取消"
      unmount-on-close
    >
      <div class="cre-form">
        <div class="cre-form__row">
          <span class="cre-form__label">规则类型</span>
          <a-select
            :model-value="draft.type"
            :options="typeOptions"
            class="cre-form__control"
            @update:model-value="setType"
          />
        </div>
        <div class="cre-form__row">
          <span class="cre-form__label">规则名称</span>
          <a-input v-model="draft.name" class="cre-form__control" placeholder="如：平台取值互斥" />
        </div>
        <div class="cre-form__row">
          <span class="cre-form__label">规则描述</span>
          <a-input
            v-model="draft.description"
            class="cre-form__control"
            placeholder="选填，用于列表展示"
          />
        </div>

        <div class="cre-form__section">触发条件（当满足）</div>
        <div class="cre-form__row">
          <span class="cre-form__label">变量</span>
          <a-select
            :model-value="draft.whenField"
            :options="fieldSelectOptions"
            class="cre-form__control"
            placeholder="选择变量"
            @update:model-value="(value: unknown) => setTextField('whenField', value)"
          />
        </div>
        <div class="cre-form__row">
          <span class="cre-form__label">运算符</span>
          <a-select
            :model-value="draft.whenOperator"
            :options="operatorOptionsFor(draft.whenField)"
            class="cre-form__control"
            @update:model-value="(value: unknown) => setTextOperator('whenOperator', value)"
          />
        </div>
        <div class="cre-form__row">
          <span class="cre-form__label">取值</span>
          <div class="cre-form__control">
            <ConditionValueControl
              v-model="draft.whenValue"
              :field="fieldOf(draft.whenField)"
              :operator="draft.whenOperator || undefined"
            />
          </div>
        </div>

        <template v-if="draft.type === 'linkage'">
          <div class="cre-form__section">联动约束（当命中时）</div>
          <div class="cre-form__row">
            <span class="cre-form__label">目标变量</span>
            <a-select
              :model-value="draft.linkageField"
              :options="fieldSelectOptions"
              class="cre-form__control"
              placeholder="选择被约束的变量"
              @update:model-value="(value: unknown) => setTextField('linkageField', value)"
            />
          </div>
          <div class="cre-form__row">
            <span class="cre-form__label">运算符</span>
            <a-select
              :model-value="draft.linkageOperator"
              :options="linkageOperatorOptions"
              class="cre-form__control"
              placeholder="选择运算符"
              @update:model-value="setLinkageOperator"
            />
          </div>
          <div v-if="linkageNeedsValue" class="cre-form__row">
            <span class="cre-form__label">约束取值</span>
            <div class="cre-form__control">
              <ConditionValueControl
                :model-value="linkageValueText"
                :field="fieldOf(draft.linkageField)"
                :operator="draft.linkageOperator || undefined"
                multiple
                creatable
                @update:model-value="setLinkageValueText"
              />
            </div>
          </div>
        </template>

        <template v-else>
          <div class="cre-form__section">
            {{
              draft.type === 'prerequisite' ? '先决条件（需先存在）' : '互斥目标（不可同时存在）'
            }}
          </div>
          <div v-for="(target, index) in draft.targets" :key="index" class="cre-form__target">
            <a-select
              :model-value="target.field"
              :options="fieldSelectOptions"
              class="cre-form__control"
              placeholder="变量"
              @update:model-value="(value: unknown) => setTargetField(target, value)"
            />
            <a-select
              :model-value="target.operator"
              :options="operatorOptionsFor(target.field)"
              class="cre-form__control"
              @update:model-value="(value: unknown) => setTargetOperator(target, value)"
            />
            <div class="cre-form__control">
              <ConditionValueControl
                v-model="target.value"
                :field="fieldOf(target.field)"
                :operator="target.operator || undefined"
              />
            </div>
            <a-button
              type="text"
              size="mini"
              status="danger"
              :disabled="draft.targets.length <= 1"
              @click="removeTarget(index)"
            >
              <template #icon><IconDelete :size="13" /></template>
            </a-button>
          </div>
          <a-button type="text" size="mini" class="cre-form__add-target" @click="addTarget">
            <template #icon><IconPlus :size="13" /></template>
            添加目标
          </a-button>
        </template>

        <div class="cre-form__row">
          <span class="cre-form__label">冲突提示</span>
          <a-input
            v-model="draft.message"
            class="cre-form__control"
            placeholder="选填；命中冲突时的提示文案"
          />
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import type { SelectOptionData } from '@arco-design/web-vue'
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-vue/es/icon'
import type {
  ConditionFieldOption,
  ConditionOperator,
  ConditionRule,
  ConditionRuleDraft,
  ConditionRuleDraftTarget,
  ConditionRuleEditorEmits,
  ConditionRuleEditorProps,
  ConditionRuleState,
  ConditionRuleType,
} from './ConditionBuilder.types'
import ConditionValueControl from './ConditionValueControl.vue'
import {
  conditionOperatorNeedsValue,
  conditionOperatorsForType,
  groupConditionFields,
  resolveConditionFieldType,
  splitConditionValues,
} from './conditionOperator'
import {
  CONDITION_RULE_STATE_LABELS,
  CONDITION_RULE_TYPE_LABELS,
  CONDITION_RULE_TYPE_OPTIONS,
  CONDITION_RULE_WARNING_LABELS,
  conditionRuleFieldEffectIndex,
  conditionRuleFromDraft,
  conditionRuleSummary,
  conditionRuleToDraft,
  conditionRuleWarning,
  createConditionRuleDraft,
} from './conditionRules'

const props = withDefaults(defineProps<ConditionRuleEditorProps>(), {
  rules: () => [],
  fieldOptions: () => [],
  fieldGroups: () => [],
  fieldEffects: () => [],
  ruleStates: () => ({}),
  disabled: false,
})

const emit = defineEmits<ConditionRuleEditorEmits>()

const typeOptions = CONDITION_RULE_TYPE_OPTIONS

const formVisible = ref(false)

const draft = reactive<ConditionRuleDraft>(createConditionRuleDraft())

const fieldEffects = computed(() => conditionRuleFieldEffectIndex(props.fieldEffects))

function fieldLabel(field: ConditionFieldOption): string {
  const label = field.label ?? field.value
  const effect = fieldEffects.value.get(field.value)?.effect ?? 'effective'
  const warning = conditionRuleWarning(effect)
  return warning ? `${label} · ${warning}` : label
}

const fieldSelectOptions = computed<SelectOptionData[]>(() => {
  if (!props.fieldGroups.length) {
    return props.fieldOptions.map((field) => ({
      value: field.value,
      label: fieldLabel(field),
    }))
  }
  return groupConditionFields(props.fieldOptions, props.fieldGroups).map((group) => ({
    isGroup: true,
    label: group.label,
    options: group.fields.map((field) => ({
      value: field.value,
      label: fieldLabel(field),
    })),
  }))
})

const linkageValueText = computed(() => draft.linkageValues.join(', '))

const linkageOperatorOptions = computed<SelectOptionData[]>(() =>
  operatorOptionsFor(draft.linkageField).filter((option) => option.value !== ''),
)

function fieldOf(fieldValue: string): ConditionFieldOption | undefined {
  return props.fieldOptions.find((option) => option.value === fieldValue)
}

function setLinkageValueText(value: string): void {
  draft.linkageValues = splitConditionValues(value)
}

const linkageNeedsValue = computed(
  () => !draft.linkageOperator || conditionOperatorNeedsValue(draft.linkageOperator),
)

function operatorValuesFor(fieldValue: string): ConditionOperator[] {
  const field = props.fieldOptions.find((option) => option.value === fieldValue)
  return conditionOperatorsForType(resolveConditionFieldType(field)).map(
    (operator) => operator.value,
  )
}

function operatorSupported(fieldValue: string, operator: ConditionOperator | ''): boolean {
  if (!operator) return true
  return operatorValuesFor(fieldValue).includes(operator)
}

function toText(value: unknown): string {
  if (value === undefined || value === null) return ''
  return String(value)
}

function operatorOptionsFor(fieldValue: string): SelectOptionData[] {
  const field = props.fieldOptions.find((option) => option.value === fieldValue)
  const operators = conditionOperatorsForType(resolveConditionFieldType(field))
  return [
    { value: '', label: '不限' },
    ...operators.map((operator) => ({ value: operator.value, label: operator.label })),
  ]
}

function typeLabel(type: ConditionRuleType): string {
  return CONDITION_RULE_TYPE_LABELS[type]
}

function typeColor(type: ConditionRuleType): string {
  if (type === 'mutual_exclusive') return 'red'
  if (type === 'prerequisite') return 'orange'
  return 'purple'
}

function stateLabel(state: ConditionRuleState): string {
  if (state === 'ineffective') return CONDITION_RULE_WARNING_LABELS.ineffective
  if (state === 'partial') return CONDITION_RULE_WARNING_LABELS.partial
  return CONDITION_RULE_STATE_LABELS[state]
}

function summaryOf(rule: ConditionRule): string {
  return conditionRuleSummary(rule, props.fieldOptions)
}

function setType(value: unknown): void {
  draft.type = (toText(value) || 'mutual_exclusive') as ConditionRuleType
}

function setTextField(key: 'whenField' | 'linkageField', value: unknown): void {
  draft[key] = toText(value)
  if (key === 'whenField' && !operatorSupported(draft.whenField, draft.whenOperator)) {
    draft.whenOperator = ''
  }
  if (key === 'linkageField' && !operatorSupported(draft.linkageField, draft.linkageOperator)) {
    draft.linkageOperator = ''
  }
}

function setLinkageOperator(value: unknown): void {
  draft.linkageOperator = toText(value) as ConditionOperator | ''
}

function setTextOperator(key: 'whenOperator', value: unknown): void {
  draft[key] = toText(value) as ConditionOperator | ''
}

function setTargetField(target: ConditionRuleDraftTarget, value: unknown): void {
  target.field = toText(value)
  if (!operatorSupported(target.field, target.operator)) {
    target.operator = ''
  }
}

function setTargetOperator(target: ConditionRuleDraftTarget, value: unknown): void {
  target.operator = toText(value) as ConditionOperator | ''
}

function resetDraft(next: ConditionRuleDraft): void {
  Object.assign(draft, next)
}

function openCreate(): void {
  resetDraft(createConditionRuleDraft())
  formVisible.value = true
}

function openEdit(rule: ConditionRule): void {
  resetDraft(conditionRuleToDraft(rule))
  formVisible.value = true
}

function addTarget(): void {
  draft.targets.push({ field: '', operator: '', value: '' })
}

function removeTarget(index: number): void {
  if (draft.targets.length <= 1) return
  draft.targets.splice(index, 1)
}

function validate(): string {
  if (!draft.name.trim()) return '请填写规则名称'
  if (!draft.whenField) return '请选择触发条件的变量'

  if (draft.type === 'linkage') {
    if (!draft.linkageField) return '请选择联动约束的目标变量'
    if (!draft.linkageOperator) return '请选择联动约束的运算符'
    if (conditionOperatorNeedsValue(draft.linkageOperator) && !draft.linkageValues.length) {
      return '请至少填写一个约束取值'
    }
    return ''
  }

  if (!draft.targets.some((target) => target.field)) {
    return draft.type === 'prerequisite' ? '请至少添加一个先决条件' : '请至少添加一个互斥目标'
  }
  return ''
}

async function handleBeforeOk(): Promise<boolean> {
  const error = validate()
  if (error) {
    Message.warning(error)
    return false
  }

  const next = conditionRuleFromDraft(draft)
  const list = props.rules.slice()
  const index = list.findIndex((rule) => rule.id === next.id)
  if (index >= 0) list.splice(index, 1, next)
  else list.push(next)
  emit('update:rules', list)
  return true
}

function removeRule(rule: ConditionRule): void {
  emit(
    'update:rules',
    props.rules.filter((entry) => entry.id !== rule.id),
  )
}

function toggleActive(rule: ConditionRule, value: boolean): void {
  emit(
    'update:rules',
    props.rules.map((entry) =>
      entry.id === rule.id ? ({ ...entry, isActive: value } as ConditionRule) : entry,
    ),
  )
}
</script>

<style scoped lang="scss">
.cre-toolbar {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.cre-toolbar__add {
  color: #007aff;
  padding: 0;
}

.cre-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cre-item {
  padding: 8px 10px;
  border: 1px solid #e5e5ea;
  border-radius: 10px;
  background: #fafafc;
}

.cre-item--off {
  opacity: 0.6;
}

.cre-item__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cre-item__name {
  font-size: 13px;
  font-weight: 600;
  color: #1d1d1f;
}

.cre-item__state {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 4px;
}

.cre-item__state--idle {
  color: #86868b;
  background: #f2f2f4;
}

.cre-item__state--active {
  color: #007aff;
  background: rgba(0, 122, 255, 0.1);
}

.cre-item__state--violation {
  color: #ff3b30;
  background: rgba(255, 59, 48, 0.1);
}

.cre-item__state--partial,
.cre-item__state--ineffective {
  color: #d46b08;
  background: rgba(255, 149, 0, 0.16);
}

.cre-item__spacer {
  flex: 1;
}

.cre-item__action {
  color: #86868b;
  padding: 0 4px;
}

.cre-item__summary {
  margin-top: 4px;
  font-size: 12px;
  color: #1d1d1f;
}

.cre-item__desc {
  margin-top: 2px;
  font-size: 11px;
  color: #86868b;
}

.cre-empty {
  padding: 12px 0;
  text-align: center;
  font-size: 12px;
  color: #86868b;
}

.cre-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.cre-form__row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cre-form__label {
  flex: 0 0 72px;
  font-size: 12px;
  color: #86868b;
}

.cre-form__control {
  flex: 1;
  width: max-content;
  min-width: 120px;
}

.cre-form__section {
  margin-top: 4px;
  font-size: 12px;
  font-weight: 600;
  color: #1d1d1f;
  padding-bottom: 4px;
  border-bottom: 1px dashed #e5e5ea;
}

.cre-form__target {
  display: flex;
  align-items: center;
  gap: 8px;
  :deep(.cre-form__control) {
    flex: 1;
    width: max-content;
    min-width: 120px;
  }
}

.cre-form__add-target {
  align-self: flex-start;
  color: #007aff;
  padding: 0;
}
</style>
