<template>
  <a-modal
    :visible="visible"
    :footer="false"
    :closable="false"
    :mask-closable="false"
    :fullscreen="true"
    class="image-editor-container"
    @cancel="handleClose"
    @before-open="handleBeforeOpen"
  >
    <ImageEditor
      v-if="visible"
      ref="editorRef"
      :src="src"
      width="100%"
      height="100%"
      @export="handleExport"
      @cancel="handleClose"
    />
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount, unref } from 'vue'
import ImageEditor from './ImageEditor.vue'
import type { ImageEditorModalEmits, ImageEditorModalProps } from './ImageEditorModal.types'

const props = defineProps<ImageEditorModalProps>()

const emit = defineEmits<ImageEditorModalEmits>()

const editorRef = ref<InstanceType<typeof ImageEditor>>()

watch(
  () => props.visible,
  (val) => {
    document.body.style.overflow = val ? 'hidden' : ''
  },
)

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})

function handleClose() {
  emit('close')
}

function handleBeforeOpen() {
  unref(editorRef)?.reset()
}

async function handleExport(blob: Blob) {
  emit('confirm', blob)
  emit('close')
}

defineExpose({
  handleExport: () => editorRef.value?.handleExport(),
})
</script>

<style lang="scss">
.image-editor-container {
  .arco-modal-wrapper {
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    .arco-modal.arco-modal-fullscreen {
      display: block;
      width: 100vw;
      height: 100vh;
      max-height: 100%;
      max-width: 100%;
      overflow: hidden;
      .arco-modal-body {
        max-height: 100%;
        max-width: 100%;
        width: 100%;
        height: 100%;
      }
    }
  }
}
</style>
