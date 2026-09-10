<template>
  <a-radio-group type="button" :model-value="modelValue" @change="handleChange">
    <a-radio v-for="option in options" :key="option.key" :value="option.key">
      {{ option.label }}
    </a-radio>
  </a-radio-group>
</template>

<script setup lang="ts">
import type { SegmentedControlProps, SegmentedControlEmits } from './SegmentedControl.types'

defineProps<SegmentedControlProps>()

const emit = defineEmits<SegmentedControlEmits>()

function handleChange(value: string | number | boolean) {
  emit('update:modelValue', value as string)
}
</script>
