<template>
  <div class="image-editor">
    <div class="image-editor__toolbar">
      <div class="image-editor__toolbar-left">
        <button
          v-if="capabilities.freeTransform"
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Select }"
          @click="setMode(EditorMode.Select)"
        >
          <IconCursor :size="14" />
          <span>选择</span>
        </button>
        <button
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Crop }"
          @click="setMode(EditorMode.Crop)"
        >
          <IconEdit :size="14" />
          <span>裁剪</span>
        </button>
        <button
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Draw }"
          @click="setMode(EditorMode.Draw)"
        >
          <IconPen :size="14" />
          <span>画笔</span>
        </button>
        <button
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Text }"
          @click="setMode(EditorMode.Text)"
        >
          <IconType :size="14" />
          <span>文字</span>
        </button>
        <button
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Sticker }"
          @click="setMode(EditorMode.Sticker)"
        >
          <IconFaceSmileFill :size="14" />
          <span>贴纸</span>
        </button>
        <button
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Filter }"
          @click="setMode(EditorMode.Filter)"
        >
          <IconWand :size="14" />
          <span>滤镜</span>
        </button>
        <button class="editor-btn" :disabled="!canUndo()" @click="undoAction">
          <IconUndo :size="14" />
        </button>
        <button class="editor-btn" :disabled="!canRedo()" @click="redoAction">
          <IconRedo :size="14" />
        </button>
      </div>
      <div class="image-editor__toolbar-right">
        <button class="editor-btn" @click="clearAll">
          <IconDelete :size="14" />
          <span>清空</span>
        </button>
      </div>
    </div>

    <div class="image-editor__canvas">
      <canvas ref="canvasRef"></canvas>
    </div>

    <div class="image-editor__footer">
      <span class="image-editor__info">
        画布: {{ state.canvasWidth }} x {{ state.canvasHeight }}
      </span>
      <span class="image-editor__info"> 图层: {{ layers.length }} </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import {
  IconCursor,
  IconScissor,
  IconPen,
  IconType,
  IconFaceSmileFill,
  IconWand,
  IconUndo,
  IconRedo,
  IconDelete,
} from '@arco-design/web-vue/es/icon'
import { useEditor } from './composables/useEditor'
import { createEngine } from './engine'
import { EditorMode } from './types'
import type { BaseLayer, ExportOptions } from './types'

const props = defineProps<{
  src: string
  width?: number
  height?: number
}>()

const emit = defineEmits<{
  export: [blob: Blob]
  cancel: []
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const {
  engine,
  state,
  layers,
  capabilities,
  setEngine,
  setMode,
  addLayer,
  updateLayer,
  removeLayer,
  undoAction,
  redoAction,
  canUndo,
  canRedo,
  exportToBlob,
  clearAll,
} = useEditor()

const width = props.width || 800
const height = props.height || 600

onMounted(async () => {
  if (!canvasRef.value) return

  const eng = await createEngine(canvasRef.value, width, height)
  setEngine(eng)

  const bgLayer: BaseLayer = {
    id: 'bg-0',
    type: 'image',
    visible: true,
    zIndex: 0,
    locked: true,
    x: 0,
    y: 0,
    width,
    height,
    rotate: 0,
    scaleX: 1,
    scaleY: 1,
    image: {
      src: props.src,
      naturalWidth: width,
      naturalHeight: height,
    },
  }
  addLayer(bgLayer)
})

onBeforeUnmount(() => {
  if (engine.value) {
    engine.value.destroy()
  }
})

async function handleExport() {
  try {
    const blob = await exportToBlob({ format: 'jpeg', quality: 0.85 })
    emit('export', blob)
  } catch (error) {
    console.error('Export failed:', error)
  }
}

defineExpose({ handleExport })
</script>

<style scoped>
.image-editor {
  display: flex;
  flex-direction: column;
  background: #ffffff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.image-editor__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid #e5e5ea;
  background: #fafafa;
}

.image-editor__toolbar-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.image-editor__toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.editor-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: #1d1d1f;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.editor-btn:hover:not(:disabled) {
  background: rgba(0, 122, 255, 0.06);
  border-color: #007aff;
  color: #007aff;
}

.editor-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.editor-btn--active {
  background: #007aff;
  color: #ffffff;
  border-color: #007aff;
}

.editor-btn--active:hover {
  background: #0066cc;
  color: #ffffff;
}

.image-editor__canvas {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: #f5f5f7;
  background-image: repeating-conic-gradient(#e8e8ed 0% 25%, #f5f5f7 0% 50%);
  background-size: 20px 20px;
}

.image-editor__canvas canvas {
  max-width: 100%;
  max-height: 100%;
  border-radius: 6px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
}

.image-editor__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  padding: 8px 16px;
  border-top: 1px solid #e5e5ea;
  background: #fafafa;
}

.image-editor__info {
  font-size: 11px;
  color: #86868b;
}
</style>
