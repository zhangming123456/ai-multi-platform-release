<script setup lang="ts">
const props = defineProps<{
  options: { key: string; label: string }[]
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function handleChange(value: string | number | boolean) {
  emit('update:modelValue', value as string)
}
</script>

<template>
  <a-radio-group type="button" :model-value="modelValue" @change="handleChange">
    <a-radio v-for="option in options" :key="option.key" :value="option.key">
      {{ option.label }}
    </a-radio>
  </a-radio-group>
</template>
