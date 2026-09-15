<template>
  <span v-if="!needsValue" class="cvc-empty">无需填写值</span>
  <template v-else-if="useMulti">
    <el-select
      v-if="supportsOptionControl"
      class="cvc-control"
      multiple
      clearable
      :filterable="isEnum || creatable"
      :allow-create="creatable"
      :remote="hasRemoteOptions"
      :remote-method="loadRemoteOptions"
      :loading="hasRemoteOptions && optionsLoading"
      :no-data-text="noDataText"
      default-first-option
      :model-value="multiSelected"
      :disabled="disabled"
      placeholder="选择值（可多选）"
      @visible-change="onSelectVisibleChange"
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
  <div v-else-if="isNumberRange" class="cvc-number-range">
    <el-input-number
      class="cvc-number-range__input"
      :model-value="numberRangeStart"
      :controls="false"
      :disabled="disabled"
      placeholder="最小值"
      @update:model-value="onNumberRangeStart"
    />
    <span class="cvc-number-range__separator">至</span>
    <el-input-number
      class="cvc-number-range__input"
      :model-value="numberRangeEnd"
      :controls="false"
      :disabled="disabled"
      placeholder="最大值"
      @update:model-value="onNumberRangeEnd"
    />
  </div>
  <el-select
    v-else-if="fieldType === 'select' || fieldType === 'boolean'"
    class="cvc-control"
    clearable
    :filterable="isEnum"
    :remote="hasRemoteOptions"
    :remote-method="loadRemoteOptions"
    :loading="hasRemoteOptions && optionsLoading"
    :no-data-text="noDataText"
    :model-value="modelValue"
    :disabled="disabled"
    placeholder="选择值"
    @visible-change="onSelectVisibleChange"
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
    controls-position="right"
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
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type {
  ConditionFieldOptionValue,
  ConditionValueControlEmits,
  ConditionValueControlProps,
  ConditionValueGranularity,
} from './ConditionBuilder.types'
import {
  CONDITION_BOOLEAN_OPTIONS,
  conditionFieldHasOptions,
  conditionOperatorAllowsMultipleValues,
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

const REMOTE_OPTIONS_DEBOUNCE = 250

const fieldType = computed(() => resolveConditionFieldType(props.field))
const isEnum = computed(() => fieldType.value === 'select')
const fieldLoader = computed(() => props.field?.loadOptions)
const hasRemoteOptions = computed(() => isEnum.value && typeof fieldLoader.value === 'function')

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

const isNumberRange = computed(
  () =>
    fieldType.value === 'number' &&
    Boolean(props.operator && conditionOperatorIsMultiValue(props.operator)),
)

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

function parseNumberRangeValue(value: string): number | undefined {
  if (!value) return undefined
  const parsed = Number(value)
  return Number.isNaN(parsed) ? undefined : parsed
}

const numberRange = computed(() => parseConditionRange(props.modelValue))
const numberRangeStart = computed(() => parseNumberRangeValue(numberRange.value.start))
const numberRangeEnd = computed(() => parseNumberRangeValue(numberRange.value.end))

const supportsOptionControl = computed(() => conditionFieldHasOptions(props.field))

const allowsMultiple = computed(() => fieldType.value === 'string' || supportsOptionControl.value)

const useMulti = computed(() => {
  if (!allowsMultiple.value) return false
  if (props.multiple !== undefined) return props.multiple
  return props.operator ? conditionOperatorAllowsMultipleValues(props.operator, props.field) : false
})

const baseOptions = computed<ConditionFieldOptionValue[]>(() =>
  fieldType.value === 'boolean' ? CONDITION_BOOLEAN_OPTIONS : (props.field?.options ?? []),
)

const remoteOptions = ref<ConditionFieldOptionValue[]>([])
const optionsLoading = ref(false)
const optionsError = ref(false)
const selectedOptionLabels = ref<Record<string, string>>({})

let optionsTimer: ReturnType<typeof setTimeout> | undefined
let optionsController: AbortController | undefined
let optionsRequestId = 0

function mergeOptions(
  base: ConditionFieldOptionValue[],
  remote: ConditionFieldOptionValue[],
): ConditionFieldOptionValue[] {
  const optionMap = new Map<string, ConditionFieldOptionValue>()
  for (const option of base) optionMap.set(option.value, option)
  for (const option of remote) optionMap.set(option.value, option)
  return [...optionMap.values()]
}

const availableOptions = computed(() => mergeOptions(baseOptions.value, remoteOptions.value))

const multiSelected = computed<string[]>(() => splitConditionValues(props.modelValue))

const selectedValues = computed<string[]>(() => {
  if (!supportsOptionControl.value) return []
  return useMulti.value ? multiSelected.value : props.modelValue ? [props.modelValue] : []
})

const resolvedOptions = computed<ValueSelectOption[]>(() => {
  const options = availableOptions.value.map((option) => ({ ...option }))
  const knownValues = new Set(options.map((option) => option.value))
  for (const value of selectedValues.value) {
    if (!value || knownValues.has(value)) continue
    options.push({
      value,
      label: selectedOptionLabels.value[value] ?? value,
    })
  }
  return options.map((option) => ({
    value: option.value,
    label: option.label,
    disabled: props.disabledValues.includes(option.value),
  }))
})

watch(
  availableOptions,
  (options) => {
    const labels = { ...selectedOptionLabels.value }
    for (const option of options) labels[option.value] = option.label
    selectedOptionLabels.value = labels
  },
  { immediate: true },
)

watch(
  () => props.field?.value,
  () => {
    cancelRemoteOptions()
    remoteOptions.value = []
    optionsError.value = false
    selectedOptionLabels.value = {}
  },
)

const noDataText = computed(() => {
  if (!hasRemoteOptions.value) return '暂无数据'
  if (optionsLoading.value) return '正在加载...'
  if (optionsError.value) return '加载失败，请重试'
  return '暂无数据'
})

function cancelRemoteOptions(): void {
  optionsRequestId += 1
  if (optionsTimer) {
    clearTimeout(optionsTimer)
    optionsTimer = undefined
  }
  optionsController?.abort()
  optionsController = undefined
  optionsLoading.value = false
}

function executeRemoteLoad(query: string, requestId: number): void {
  const loader = fieldLoader.value
  if (!loader) return
  const controller = new AbortController()
  optionsController = controller
  optionsLoading.value = true
  optionsError.value = false

  void loader(query, controller.signal)
    .then((options) => {
      if (requestId !== optionsRequestId || controller.signal.aborted) return
      remoteOptions.value = Array.isArray(options)
        ? options.map((option) => ({
            value: String(option.value),
            label: String(option.label),
          }))
        : []
    })
    .catch(() => {
      if (requestId !== optionsRequestId || controller.signal.aborted) return
      optionsError.value = true
    })
    .finally(() => {
      if (requestId !== optionsRequestId) return
      optionsLoading.value = false
      optionsController = undefined
    })
}

function loadRemoteOptions(query = ''): void {
  if (!hasRemoteOptions.value) return
  cancelRemoteOptions()
  const requestId = optionsRequestId
  const normalizedQuery = query.trim()
  optionsTimer = setTimeout(() => {
    optionsTimer = undefined
    executeRemoteLoad(normalizedQuery, requestId)
  }, REMOTE_OPTIONS_DEBOUNCE)
}

function onSelectVisibleChange(visible: boolean): void {
  if (!visible || !hasRemoteOptions.value) return
  cancelRemoteOptions()
  executeRemoteLoad('', optionsRequestId)
}

onBeforeUnmount(cancelRemoteOptions)

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

function updateNumberRange(part: 'start' | 'end', value: unknown): void {
  emit(
    'update:modelValue',
    formatConditionRange({
      ...numberRange.value,
      [part]: toText(value).trim(),
    }),
  )
}

function onNumberRangeStart(value: unknown): void {
  updateNumberRange('start', value)
}

function onNumberRangeEnd(value: unknown): void {
  updateNumberRange('end', value)
}
</script>

<style scoped lang="scss">
:deep(.cvc-control) {
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

.cvc-number-range {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  width: 100%;
}

.cvc-number-range__input {
  width: 100%;
}

.cvc-number-range__separator {
  font-size: 12px;
  color: #86868b;
}
</style>
