<script setup lang="ts">
defineProps<{
  visible: boolean
  title: string
  width?: string | number
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  close: []
}>()

function handleCancel() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <a-modal
    :visible="visible"
    :title="title"
    :width="width || 520"
    :mask-closable="true"
    @cancel="handleCancel"
  >
    <slot></slot>
    <template #footer>
      <slot name="footer"></slot>
    </template>
  </a-modal>
</template>
