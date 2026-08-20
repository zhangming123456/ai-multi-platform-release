<template>
  <a-modal
    :visible="visible"
    :footer="false"
    :closable="true"
    :width="880"
    wrap-class-name="image-editor-modal"
    @cancel="handleClose"
  >
    <div class="editor-modal-wrap">
      <ImageEditor
        v-if="visible"
        ref="editorRef"
        :src="src"
        :width="820"
        :height="540"
        @export="handleExport"
      />
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ImageEditor from './ImageEditor.vue'

const props = defineProps<{
  visible: boolean
  src: string
}>()

const emit = defineEmits<{
  close: []
  confirm: [blob: Blob]
}>()

const editorRef = ref<InstanceType<typeof ImageEditor> | null>(null)

function handleClose() {
  emit('close')
}

async function handleExport(blob: Blob) {
  emit('confirm', blob)
  emit('close')
}

defineExpose({
  handleExport: () => editorRef.value?.handleExport(),
})
</script>

<style scoped>
.editor-modal-wrap {
  padding: 0;
}

:deep(.image-editor-modal .arco-modal-content) {
  padding: 0;
  background: #ffffff;
}

:deep(.image-editor-modal .arco-modal-header) {
  display: none;
}

:deep(.image-editor-modal .arco-modal-close-icon) {
  top: 12px;
  right: 12px;
  z-index: 10;
}
</style>
