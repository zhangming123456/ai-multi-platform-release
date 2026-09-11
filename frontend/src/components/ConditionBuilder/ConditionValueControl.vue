<template>
  <span v-if="!needsValue" class="cvc-empty">无需填写值</span>
  <template v-else-if="useMulti">
    <a-select
      v-if="hasFixedOptions"
      class="cvc-control"
      multiple
      allow-clear
      :allow-create="creatable"
      :model-value="multiSelected"
      :options="resolvedOptions"
      :disabled="disabled"
      placeholder="选择值（可多选）"
      @update:model-value="onMultiValue"
    />
    <a-input
      v-else
      class="cvc-control"
      allow-clear
      :model-value="modelValue"
      :disabled="disabled"
      placeholder="多个值用逗号分隔"
      @update:model-value="onTextValue"
    />
  </template>
  <a-select
    v-else-if="fieldType === 'select' || fieldType === 'boolean'"
    class="cvc-control"
    allow-clear
    :model-value="modelValue"
    :options="resolvedOptions"
    :disabled="disabled"
    placeholder="选择值"
    @update:model-value="onTextValue"
  />
  <a-input-number
    v-else-if="fieldType === 'number'"
    class="cvc-control"
    :model-value="numberValue"
    :disabled="disabled"
    placeholder="输入数值"
    @update:model-value="onTextValue"
  />
  <a-date-picker
    v-else-if="fieldType === 'date'"
    class="cvc-control"
    :model-value="modelValue"
    value-format="YYYY-MM-DD"
    :disabled="disabled"
    placeholder="选择日期"
    @update:model-value="onTextValue"
  />
  <a-date-picker
    v-else-if="fieldType === 'datetime'"
    class="cvc-control"
    show-time
    :model-value="modelValue"
    value-format="YYYY-MM-DD HH:mm:ss"
    :disabled="disabled"
    placeholder="选择日期时间"
    @update:model-value="onTextValue"
  />
  <a-time-picker
    v-else-if="fieldType === 'time'"
    class="cvc-control"
    :model-value="modelValue"
    format="HH:mm:ss"
    :disabled="disabled"
    placeholder="选择时间"
    @update:model-value="onTextValue"
  />
  <a-input
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
import type { SelectOptionData } from '@arco-design/web-vue'
import type {
  ConditionValueControlEmits,
  ConditionValueControlProps,
} from './ConditionBuilder.types'
import {
  CONDITION_BOOLEAN_OPTIONS,
  conditionFieldHasOptions,
  conditionOperatorIsMultiValue,
  conditionOperatorNeedsValue,
  resolveConditionFieldType,
  splitConditionValues,
} from './conditionOperator'

const props = withDefaults(defineProps<ConditionValueControlProps>(), {
  field: undefined,
  operator: undefined,
  multiple: undefined,
  creatable: false,
  disabled: false,
  disabledValues: () => [],
})

const emit = defineEmits<ConditionValueControlEmits>()

const fieldType = computed(() => resolveConditionFieldType(props.field))

const needsValue = computed(() =>
  props.operator ? conditionOperatorNeedsValue(props.operator) : true,
)

const hasFixedOptions = computed(() => conditionFieldHasOptions(props.field))

const allowsMultiple = computed(() => fieldType.value === 'string' || hasFixedOptions.value)

const useMulti = computed(() => {
  if (!allowsMultiple.value) return false
  if (props.multiple !== undefined) return props.multiple
  return props.operator ? conditionOperatorIsMultiValue(props.operator) : false
})

const resolvedOptions = computed<SelectOptionData[]>(() => {
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
