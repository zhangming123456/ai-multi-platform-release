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

<script setup lang="ts">
import type { ModalProps, ModalEmits } from './Modal.types'

defineProps<ModalProps>()

const emit = defineEmits<ModalEmits>()

function handleCancel() {
  emit('update:visible', false)
  emit('close')
}
</script>
