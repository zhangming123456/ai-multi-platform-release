<template>
  <span v-if="!needsValue" class="cvc-empty">无需填写值</span>
  <template v-else-if="useMulti">
    <el-select
      v-if="hasFixedOptions"
      class="cvc-control"
      multiple
      clearable
      :filterable="creatable"
      :allow-create="creatable"
      default-first-option
      :model-value="multiSelected"
      :disabled="disabled"
      placeholder="选择值（可多选）"
      @update:model-value="onMultiValue"
    >
      <el-option
        v-for="option in resolvedOptions"
        :key="option.value"
        :label="option.label"
        :value="option.value"
        :disabled="option.disabled"
      />
    </el-select>
    <el-input
      v-else
      class="cvc-control"
      clearable
      :model-value="modelValue"
      :disabled="disabled"
      placeholder="多个值用逗号分隔"
      @update:model-value="onTextValue"
    />
  </template>
  <el-select
    v-else-if="fieldType === 'select' || fieldType === 'boolean'"
    class="cvc-control"
    clearable
    :model-value="modelValue"
    :disabled="disabled"
    placeholder="选择值"
    @update:model-value="onTextValue"
  >
    <el-option
      v-for="option in resolvedOptions"
      :key="option.value"
      :label="option.label"
      :value="option.value"
      :disabled="option.disabled"
    />
  </el-select>
  <el-input-number
    v-else-if="fieldType === 'number'"
    class="cvc-control"
    :model-value="numberValue"
    :disabled="disabled"
    placeholder="输入数值"
    @update:model-value="onTextValue"
  />
  <el-time-picker
    v-else-if="temporalRange && temporalGranularity === 'time'"
    class="cvc-control"
    is-range
    :model-value="rangePickerValue"
    format="HH:mm:ss"
    value-format="HH:mm:ss"
    clearable
    :disabled="disabled"
    start-placeholder="开始时间"
    end-placeholder="结束时间"
    range-separator="至"
    @update:model-value="onRangeValue"
  />
  <el-date-picker
    v-else-if="temporalRange"
    class="cvc-control"
    :type="rangePickerType"
    :model-value="rangePickerValue"
    :value-format="rangeValueFormat"
    clearable
    :disabled="disabled"
    :start-placeholder="rangePlaceholder[0]"
    :end-placeholder="rangePlaceholder[1]"
    range-separator="至"
    @update:model-value="onRangeValue"
  />
  <el-date-picker
    v-else-if="temporalGranularity === 'date'"
    class="cvc-control"
    type="date"
    :model-value="modelValue"
    value-format="YYYY-MM-DD"
    :disabled="disabled"
    placeholder="选择日期"
    @update:model-value="onTextValue"
  />
  <el-date-picker
    v-else-if="temporalGranularity === 'datetime'"
    class="cvc-control"
    type="datetime"
    :model-value="modelValue"
    value-format="YYYY-MM-DD HH:mm:ss"
    :disabled="disabled"
    placeholder="选择日期时间"
    @update:model-value="onTextValue"
  />
  <el-time-picker
    v-else-if="temporalGranularity === 'time'"
    class="cvc-control"
    :model-value="modelValue"
    format="HH:mm:ss"
    value-format="HH:mm:ss"
    :disabled="disabled"
    placeholder="选择时间"
    @update:model-value="onTextValue"
  />
  <el-input
    v-else
    class="cvc-control"
    :model-value="modelValue"
    :disabled="disabled"
    placeholder="输入值"
    @update:model-value="onTextValue"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  ConditionValueControlEmits,
  ConditionValueControlProps,
  ConditionValueGranularity,
} from './ConditionBuilder.types'
import {
  CONDITION_BOOLEAN_OPTIONS,
  conditionFieldHasOptions,
  conditionOperatorIsMultiValue,
  conditionOperatorNeedsValue,
  formatConditionRange,
  parseConditionRange,
  resolveConditionFieldType,
  resolveGranularityForType,
  splitConditionValues,
} from './conditionOperator'

const props = withDefaults(defineProps<ConditionValueControlProps>(), {
  field: undefined,
  operator: undefined,
  granularity: undefined,
  multiple: undefined,
  creatable: false,
  disabled: false,
  disabledValues: () => [],
})

const emit = defineEmits<ConditionValueControlEmits>()

interface ValueSelectOption {
  value: string
  label: string
  disabled: boolean
}

const fieldType = computed(() => resolveConditionFieldType(props.field))

const temporalGranularity = computed<ConditionValueGranularity | ''>(() => {
  const type = fieldType.value
  if (type !== 'date' && type !== 'datetime' && type !== 'time') return ''
  return resolveGranularityForType(type, props.granularity)
})

const needsValue = computed(() =>
  props.operator ? conditionOperatorNeedsValue(props.operator) : true,
)

const temporalRange = computed(() => {
  if (!temporalGranularity.value || !props.operator) return false
  return conditionOperatorIsMultiValue(props.operator)
})

const rangeValueFormat = computed(() =>
  temporalGranularity.value === 'date' ? 'YYYY-MM-DD' : 'YYYY-MM-DD HH:mm:ss',
)

const rangePickerType = computed<'daterange' | 'datetimerange'>(() =>
  temporalGranularity.value === 'datetime' ? 'datetimerange' : 'daterange',
)

const rangePlaceholder = computed(() =>
  temporalGranularity.value === 'date' ? ['开始日期', '结束日期'] : ['开始时间', '结束时间'],
)

function rangeEnds(): (string | undefined)[] {
  const range = parseConditionRange(props.modelValue)
  if (!range.start && !range.end) return []
  return [range.start || undefined, range.end || undefined]
}

const rangePickerValue = computed(() => rangeEnds() as unknown as (string | number | Date)[])

const hasFixedOptions = computed(() => conditionFieldHasOptions(props.field))

const allowsMultiple = computed(() => fieldType.value === 'string' || hasFixedOptions.value)

const useMulti = computed(() => {
  if (!allowsMultiple.value) return false
  if (props.multiple !== undefined) return props.multiple
  return props.operator ? conditionOperatorIsMultiValue(props.operator) : false
})

const resolvedOptions = computed<ValueSelectOption[]>(() => {
  const base =
    fieldType.value === 'boolean' ? CONDITION_BOOLEAN_OPTIONS : (props.field?.options ?? [])
  return base.map((option) => ({
    value: option.value,
    label: option.label,
    disabled: props.disabledValues.includes(option.value),
  }))
})

const multiSelected = computed<string[]>(() => splitConditionValues(props.modelValue))

const numberValue = computed<number | undefined>(() => {
  if (props.modelValue === '') return undefined
  const parsed = Number(props.modelValue)
  return Number.isNaN(parsed) ? undefined : parsed
})

function toText(value: unknown): string {
  if (value === undefined || value === null) return ''
  return String(value)
}

function onTextValue(value: unknown): void {
  emit('update:modelValue', toText(value))
}

function onMultiValue(value: unknown): void {
  emit(
    'update:modelValue',
    Array.isArray(value) ? value.map((entry) => toText(entry)).join(', ') : '',
  )
}

function onRangeValue(value: unknown): void {
  const ends = Array.isArray(value) ? value : []
  emit(
    'update:modelValue',
    formatConditionRange({ start: toText(ends[0]).trim(), end: toText(ends[1]).trim() }),
  )
}
</script>

<style scoped>
.cvc-control {
  width: 100%;
}

.cvc-empty {
  display: inline-flex;
  align-items: center;
  height: 30px;
  padding: 0 12px;
  font-size: 12px;
  color: #86868b;
  background: #f5f5f7;
  border-radius: 8px;
}
</style>
