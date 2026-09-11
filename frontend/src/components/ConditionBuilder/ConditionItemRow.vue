<template>
  <div class="cir-row">
    <ConditionConnector
      :index="index"
      :logic="item.logic"
      :disabled="disabled"
      :editable="logicEditable"
      @update:logic="onLogicChange"
    />

    <div class="cir-row__fields">
      <a-auto-complete
        class="cre-form__control cir-field"
        :model-value="item.field"
        :data="fieldSuggestData"
        :filter-option="filterFieldOption"
        :strict="false"
        :disabled="disabled"
        :trigger-props="{
          contentStyle: {
            minWidth: 'max-content',
          },
        }"
        allow-clear
        placeholder="输入变量名"
        @update:model-value="onFieldChange"
      >
        <template #option="{ data }">
          <div
            class="cir-suggest"
            :class="{ 'cir-suggest--disabled': fieldDisabled(optionValue(data)) }"
          >
            <div class="cir-suggest__head">
              <span class="cir-suggest__name">{{ suggestLabel(optionValue(data)) }}</span>
              <code class="cir-suggest__code">{{ optionValue(data) }}</code>
              <span class="cir-suggest__type">{{
                valueTypeLabel(suggestType(optionValue(data)))
              }}</span>
              <span v-if="fieldWarning(optionValue(data))" class="cir-suggest__warn">
                {{ fieldWarning(optionValue(data)) }}
              </span>
              <span v-if="groupLabelOf(optionValue(data))" class="cir-suggest__group">
                {{ groupLabelOf(optionValue(data)) }}
              </span>
              <span v-if="fieldQueryable(optionValue(data)) === false" class="cir-suggest__demo">
                不可查询
              </span>
              <span v-if="fieldDisabled(optionValue(data))" class="cir-suggest__lock">禁用</span>
            </div>
            <div v-if="fieldHint(optionValue(data))" class="cir-suggest__desc">
              {{ fieldHint(optionValue(data)) }}
            </div>
          </div>
        </template>
      </a-auto-complete>

      <a-tooltip
        v-if="fieldDemo"
        content="该变量暂无对应真实列，生成的 SQL 会跳过此条件"
        position="tr"
      >
        <span class="cir-demo">演示</span>
      </a-tooltip>

      <a-tooltip
        v-if="fieldInactive"
        content="该变量不属于当前可用变量组，已保留原配置"
        position="tr"
      >
        <span class="cir-stale">
          <IconExclamationCircle :size="14" />
        </span>
      </a-tooltip>

      <a-select
        class="cre-form__control cir-operator"
        :model-value="item.operator"
        :options="operatorOptions"
        :disabled="disabled"
        :trigger-props="{
          contentStyle: {
            minWidth: 'max-content',
          },
        }"
        @update:model-value="onOperatorChange"
      />

      <div class="cre-form__control cir-value">
        <ConditionValueControl
          :model-value="item.value"
          :field="field"
          :operator="item.operator"
          :disabled="disabled"
          :disabled-values="disabledValues"
          @update:model-value="onValueChange"
        />
      </div>

      <a-tooltip v-if="rowViolations.length" :content="rowViolationText" position="tr">
        <span class="cir-warn">
          <IconExclamationCircle :size="14" />
        </span>
      </a-tooltip>

      <a-tooltip content="把当前条件变成条件组，并自动追加一个空条件（且）" position="tr">
        <a-button
          class="cir-wrap"
          type="text"
          size="mini"
          :disabled="disabled"
          @click="wrapToGroup"
        >
          + 并且满足
        </a-button>
      </a-tooltip>

      <a-button
        class="cir-remove"
        type="text"
        status="danger"
        size="small"
        :disabled="disabled"
        @click="removeSelf"
      >
        <template #icon><IconDelete /></template>
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import type { SelectOptionData } from '@arco-design/web-vue'
import { IconDelete, IconExclamationCircle } from '@arco-design/web-vue/es/icon'
import type {
  ConditionFieldGroup,
  ConditionFieldOption,
  ConditionItem,
  ConditionItemRowEmits,
  ConditionItemRowProps,
  ConditionLogic,
  ConditionOperator,
  ConditionValueType,
} from './ConditionBuilder.types'
import ConditionConnector from './ConditionConnector.vue'
import ConditionValueControl from './ConditionValueControl.vue'
import {
  CONDITION_BOOLEAN_OPTIONS,
  CONDITION_OPERATORS,
  CONDITION_VALUE_TYPE_LABELS,
  conditionFieldGroupMap,
  conditionFieldLabel,
  conditionOperatorsForType,
  conditionOperatorNeedsValue,
  conditionValueAllowedForField,
  resolveConditionFieldType,
  resolveConditionOperator,
} from './conditionOperator'
import {
  conditionRuleFieldEffectMap,
  conditionRuleFieldLockMap,
  conditionRuleValueDisabled,
  conditionRuleValueLimits,
  conditionRuleWarning,
} from './conditionRules'

const props = withDefaults(defineProps<ConditionItemRowProps>(), {
  fieldGroups: () => [],
  disabled: false,
  logicEditable: true,
})

const emit = defineEmits<ConditionItemRowEmits>()

const field = computed<ConditionFieldOption | undefined>(() =>
  props.fieldOptions.find((option) => option.value === props.item.field),
)

const fieldInactive = computed(() => Boolean(props.item.field) && !field.value)

const fieldDemo = computed(() => field.value?.queryable === false)

const fieldGroupMap = computed<Map<string, ConditionFieldGroup>>(() =>
  conditionFieldGroupMap(props.fieldGroups),
)

const fieldType = computed<ConditionValueType>(() => resolveConditionFieldType(field.value))

const optionValues = computed<string[]>(() => {
  if (fieldType.value === 'boolean') {
    return CONDITION_BOOLEAN_OPTIONS.map((option) => option.value)
  }
  return (field.value?.options ?? []).map((option) => option.value)
})

const fieldLocks = computed(() => conditionRuleFieldLockMap(props.ruleContext))

const fieldEffects = computed(() => conditionRuleFieldEffectMap(props.ruleContext))

function fieldWarning(fieldValue: string): string {
  const effect = fieldEffects.value.get(fieldValue)
  return effect ? conditionRuleWarning(effect.effect) : ''
}

const valueLimits = computed(() => conditionRuleValueLimits(props.ruleContext, props.item.field))

const rowViolations = computed(() =>
  (props.ruleContext?.violations ?? []).filter((violation) =>
    violation.itemIds.includes(props.item.id),
  ),
)

const rowViolationText = computed(() =>
  rowViolations.value.map((violation) => violation.message).join('；'),
)

function fieldDisabled(fieldValue: string): boolean {
  return fieldLocks.value.has(fieldValue)
}

function fieldHint(fieldValue: string): string {
  const lock = fieldLocks.value.get(fieldValue)
  if (lock) return lock.reason
  return fieldOf(fieldValue)?.description ?? ''
}

function valueDisabled(value: string): boolean {
  return conditionRuleValueDisabled(valueLimits.value, value, fieldOf(props.item.field))
}

const operatorOptions = computed<SelectOptionData[]>(() => {
  const operators = fieldInactive.value
    ? CONDITION_OPERATORS
    : conditionOperatorsForType(fieldType.value)
  return operators.map((operator) => ({ value: operator.value, label: operator.label }))
})

const disabledValues = computed<string[]>(() =>
  optionValues.value.filter((value) => valueDisabled(value)),
)

const fieldSuggestData = computed<SelectOptionData[]>(() =>
  props.fieldOptions.map((option) => ({
    value: option.value,
    label: option.value,
    description: option.description ?? '',
    disabled: fieldDisabled(option.value),
  })),
)

function toValueText(value: unknown): string {
  if (value === undefined || value === null) return ''
  return String(value)
}

function send(patch: Partial<ConditionItem>): void {
  emit('command', { type: 'update-item', path: props.path, index: props.index, patch })
}

function onLogicChange(logic: ConditionLogic): void {
  emit('command', { type: 'set-logic', path: [...props.path, props.index], logic })
}

function onFieldChange(value: string): void {
  const lock = fieldLocks.value.get(value)
  if (lock) {
    Message.warning(lock.reason)
    return
  }
  const nextField = props.fieldOptions.find((option) => option.value === value)
  const operator = resolveConditionOperator({ ...props.item, field: value }, nextField)
  const nextValue = conditionValueAllowedForField(props.item.value, nextField)
    ? props.item.value
    : ''
  send({ field: value, operator, value: nextValue })
}

function onOperatorChange(value: unknown): void {
  const operator = String(value ?? 'eq') as ConditionOperator
  send({ operator, value: conditionOperatorNeedsValue(operator) ? props.item.value : '' })
}

function onValueChange(value: unknown): void {
  send({ value: toValueText(value) })
}

function removeSelf(): void {
  emit('command', { type: 'remove-item', path: props.path, index: props.index })
}

function wrapToGroup(): void {
  emit('command', { type: 'wrap-item', path: props.path, index: props.index })
}

function fieldOf(fieldValue: string): ConditionFieldOption | undefined {
  return props.fieldOptions.find((option) => option.value === fieldValue)
}

function valueTypeLabel(type?: ConditionValueType): string {
  return CONDITION_VALUE_TYPE_LABELS[type ?? 'string']
}

function suggestLabel(fieldValue: string): string {
  const option = fieldOf(fieldValue)
  return option ? conditionFieldLabel(option) : fieldValue
}

function groupLabelOf(fieldValue: string): string {
  return fieldGroupMap.value.get(fieldValue)?.label ?? ''
}

function fieldQueryable(fieldValue: string): boolean | undefined {
  return fieldOf(fieldValue)?.queryable
}

function suggestType(fieldValue: string): ConditionValueType | undefined {
  return fieldOf(fieldValue)?.type
}

function optionValue(option: unknown): string {
  if (option && typeof option === 'object' && 'value' in option) {
    return toValueText((option as { value?: unknown }).value)
  }
  return ''
}

function filterFieldOption(inputValue: string, option: SelectOptionData): boolean {
  const query = inputValue.trim().toLowerCase()
  if (!query) return true
  const candidates = [option.value, option.label, option.description]
  return candidates.some((candidate) => toValueText(candidate).toLowerCase().includes(query))
}
</script>

<style scoped lang="scss">
.cir-row {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 10px;
}

.cir-row__fields {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  :deep(.cre-form__control) {
    flex: 1;
    width: max-content;
    min-width: 120px;
  }
}

.cir-remove {
  flex: 0 0 auto;
}

.cir-wrap {
  flex: 0 0 auto;
  padding: 0 6px;
  font-size: 12px;
  color: #5856d6;
}

.cir-wrap:hover {
  color: #007aff;
}

.cir-suggest {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cir-suggest__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cir-suggest__name {
  font-size: 13px;
  color: #1d1d1f;
}

.cir-suggest__code {
  font-size: 12px;
  color: #86868b;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.cir-suggest__type {
  font-size: 11px;
  color: #007aff;
  background: rgba(0, 122, 255, 0.08);
  border-radius: 4px;
  padding: 1px 6px;
}

.cir-suggest__group {
  font-size: 11px;
  color: #5856d6;
  background: rgba(88, 86, 214, 0.1);
  border-radius: 4px;
  padding: 1px 6px;
}

.cir-suggest__warn {
  font-size: 11px;
  color: #d46b08;
  background: rgba(255, 149, 0, 0.16);
  border-radius: 4px;
  padding: 1px 6px;
}

.cir-suggest__demo {
  font-size: 11px;
  color: #b7791f;
  background: rgba(255, 193, 7, 0.16);
  border-radius: 4px;
  padding: 1px 6px;
}

.cir-suggest__desc {
  font-size: 11px;
  color: #86868b;
}

.cir-suggest--disabled {
  opacity: 0.55;
}

.cir-suggest__lock {
  font-size: 10px;
  color: #ff3b30;
  background: rgba(255, 59, 48, 0.1);
  border-radius: 4px;
  padding: 1px 5px;
}

.cir-warn {
  display: inline-flex;
  align-items: center;
  color: #ff9500;
  cursor: help;
}

.cir-stale {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  color: #ff9500;
  cursor: help;
}

.cir-demo {
  flex: 0 0 auto;
  font-size: 11px;
  color: #b7791f;
  background: rgba(255, 193, 7, 0.16);
  border-radius: 4px;
  padding: 1px 6px;
  cursor: help;
}
</style>
