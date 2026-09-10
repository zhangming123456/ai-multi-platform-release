<template>
  <div class="image-editor" :style="imageEditorStyle">
    <div class="image-editor__toolbar">
      <div class="image-editor__toolbar-left">
        <button
          v-if="capabilities.freeTransform"
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Select }"
          @click="setMode(EditorMode.Select)"
        >
          <IconPen :size="14" />
          <span>选择</span>
        </button>
        <button
          class="editor-btn"
          :class="{ 'editor-btn--active': state.mode === EditorMode.Crop }"
          @click="setMode(EditorMode.Crop)"
        >
          <IconScissor :size="14" />
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
          <IconFontColors :size="14" />
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
          <IconFilter :size="14" />
          <span>滤镜</span>
        </button>
        <button class="editor-btn" @click="handleRotate">
          <IconSync :size="14" />
          <span>旋转</span>
        </button>
        <div class="editor-divider" />
        <button class="editor-btn" :disabled="!canUndo()" @click="undoAction">
          <IconUndo :size="14" />
          <span>撤销</span>
        </button>
        <button class="editor-btn" :disabled="!canRedo()" @click="redoAction">
          <IconRedo :size="14" />
          <span>重做</span>
        </button>
      </div>
      <div class="image-editor__toolbar-right">
        <div class="engine-switch" title="引擎切换">
          <button
            class="engine-switch__btn"
            :class="{ 'engine-switch__btn--active': engineType === 'lightweight' }"
            @click="handleEngineSwitch('lightweight')"
          >
            <IconCode :size="12" />
            <span>Canvas</span>
          </button>
          <button
            class="engine-switch__btn"
            :class="{ 'engine-switch__btn--active': engineType === 'powerful' }"
            @click="handleEngineSwitch('powerful')"
          >
            <IconThunderbolt :size="12" />
            <span>Fabric</span>
          </button>
        </div>
        <button
          class="editor-btn"
          :class="{ 'editor-btn--active': compressVisible }"
          @click="toggleCompress"
        >
          <IconDownload :size="14" />
          <span>压缩</span>
        </button>
        <button class="editor-btn" @click="handleReset">
          <IconRefresh :size="14" />
          <span>重置</span>
        </button>
        <button class="editor-btn" :disabled="!canDeleteLayer" @click="handleDeleteLayer">
          <IconDelete :size="14" />
          <span>删除</span>
        </button>
        <button class="editor-btn" @click="clearAll">
          <IconDelete :size="14" />
          <span>清空</span>
        </button>
      </div>
    </div>

    <div ref="canvasEditRef" class="image-editor__canvas">
      <canvas ref="canvasRef"></canvas>
    </div>

    <div v-if="showBottomPanel" class="image-editor__panel">
      <button type="button" class="image-editor__panel-close" title="关闭面板" @click="closePanel">
        <IconClose :size="12" />
      </button>
      <PanelDraw
        v-if="state.mode === EditorMode.Draw"
        @style-change="handleDrawStyle"
        @clear="clearDrawings"
      />
      <PanelText
        v-else-if="state.mode === EditorMode.Text"
        :edit-layer="editingTextLayer"
        @add="handleAddText"
        @update="handleUpdateText"
      />
      <PanelSticker v-else-if="state.mode === EditorMode.Sticker" @add="handleAddSticker" />
      <PanelFilter v-else-if="state.mode === EditorMode.Filter" @apply="handleApplyFilter" />
      <PanelCompress v-else-if="compressVisible" @export="handleCompressExport" />
    </div>

    <PanelCrop
      v-if="state.mode === EditorMode.Crop && canvasRef"
      :canvas-width="state.canvasWidth"
      :canvas-height="state.canvasHeight"
      :canvas-el="canvasRef"
      :viewport="state.viewport"
      @apply="handleCropApply"
      @cancel="handleCropCancel"
      @pan-image="moveBaseImage"
      @zoom-image="scaleBaseImage"
    />

    <div class="image-editor__footer">
      <span class="image-editor__info">
        画布: {{ state.canvasWidth }} x {{ state.canvasHeight }}
      </span>
      <span class="image-editor__info">图层: {{ layers.length }}</span>
      <div class="image-editor__footer-actions">
        <a-button size="small" @click="emit('cancel')">取消</a-button>
        <a-button type="primary" size="small" @click="handleExport">保存</a-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, unref, type CSSProperties, nextTick } from 'vue'
import {
  IconPen,
  IconScissor,
  IconFontColors,
  IconFaceSmileFill,
  IconFilter,
  IconSync,
  IconUndo,
  IconRedo,
  IconDownload,
  IconRefresh,
  IconDelete,
  IconClose,
  IconCode,
  IconThunderbolt,
} from '@arco-design/web-vue/es/icon'
import { useEditor } from './composables/useEditor'
import { createEngine } from './engine'
import { ENGINE_TYPE } from './config/engine.config'
import type { EngineType } from './config/engine.config.types'
import { EditorMode } from './ImageEditor.types'
import type { BaseLayer, ImageEditorEmits, ImageEditorProps } from './ImageEditor.types'
import type { CropRect, FilterType } from './utils/image.types'
import { loadImage } from './utils/image'
import PanelDraw from './components/panel-draw.vue'
import PanelText from './components/panel-text.vue'
import PanelSticker from './components/panel-sticker.vue'
import PanelFilter from './components/panel-filter.vue'
import PanelCompress from './components/panel-compress.vue'
import PanelCrop from './components/panel-crop.vue'
import { unrefElement } from '@vueuse/core'

const props = defineProps<ImageEditorProps>()

const emit = defineEmits<ImageEditorEmits>()

const canvasEditRef = ref<HTMLElement>()
const canvasRef = ref<HTMLCanvasElement>()
const compressVisible = ref(false)
const engineType = ref<EngineType>(ENGINE_TYPE)
const drawColor = ref('#007aff')
const drawWidth = ref(6)
const canvasWidth = ref(0)
const canvasHeight = ref(0)

/**
 * 判断是否带小数的数字
 * @param {*} val
 * @returns {boolean}
 */
function isDecimalNumber(val: unknown): boolean {
  // 先转字符串判断，匹配数字.数字，支持正负
  const reg = /^-?\d+\.\d+$/
  return reg.test(String(val ?? ''))
}

const imageEditorStyle = computed(() => {
  const style: CSSProperties = {
    width: isDecimalNumber(props.width) ? `${props.width}px` : props.width,
    height: isDecimalNumber(props.height) ? `${props.height}px` : props.height,
  }
  return style
})

const {
  engine,
  state,
  layers,
  capabilities,
  setEngine,
  setMode,
  addLayer,
  removeLayer,
  updateLayer,
  selectLayer,
  undoAction,
  redoAction,
  canUndo,
  canRedo,
  exportToBlob,
  clearAll,
  setCanvasSize,
  setOriginalImage,
  rotate90,
  applyCrop,
  applyFilter,
  addTextLayer,
  addStickerLayer,
  resetEditor,
  moveBaseImage,
  scaleBaseImage,
  syncLayersToEngine,
} = useEditor()

const editingTextLayer = computed(() => {
  const id = state.value.activeLayerId
  if (!id) return null
  const layer = layers.value.find((l) => l.id === id)
  if (!layer || layer.type !== 'text') return null
  return layer
})

const canDeleteLayer = computed(() => {
  const id = state.value.activeLayerId
  if (!id) return false
  const layer = layers.value.find((l) => l.id === id)
  return !!layer && layer.type !== 'image'
})

const showBottomPanel = computed(() => {
  return (
    state.value.mode === EditorMode.Draw ||
    state.value.mode === EditorMode.Text ||
    state.value.mode === EditorMode.Sticker ||
    state.value.mode === EditorMode.Filter ||
    compressVisible.value
  )
})

async function initEngine(type: EngineType) {
  if (!unref(canvasRef)) return
  handleCanvasRect()

  const oldEngine = engine.value
  if (oldEngine) {
    oldEngine.destroy()
  }

  const eng = await createEngine(unref(canvasRef)!, unref(canvasWidth), unref(canvasHeight), type)
  engineType.value = type
  setEngine(eng)

  bindDrawComplete(eng)
  bindDoubleClick(eng)
  eng.setDrawStyle?.(drawColor.value, drawWidth.value)
  setMode(state.value.mode)
  if (oldEngine) {
    eng.setSize(state.value.canvasWidth, state.value.canvasHeight)
    await syncLayersToEngine()
    eng.fitToViewport?.()
    return
  }

  const image = await loadImage(props.src)
  const imgWidth = image.naturalWidth || unref(canvasWidth)
  const imgHeight = image.naturalHeight || unref(canvasHeight)
  const bgLayer: BaseLayer = {
    id: 'bg-0',
    type: 'image',
    visible: true,
    zIndex: 0,
    locked: true,
    x: 0,
    y: 0,
    width: imgWidth,
    height: imgHeight,
    rotate: 0,
    scaleX: 1,
    scaleY: 1,
    image: {
      src: props.src,
      naturalWidth: imgWidth,
      naturalHeight: imgHeight,
    },
  }
  setCanvasSize(imgWidth, imgHeight)
  setOriginalImage(props.src, imgWidth, imgHeight)
  addLayer(bgLayer)
}

function bindDrawComplete(eng: NonNullable<typeof engine.value>) {
  eng.onDrawComplete?.((points, color, strokeWidth) => {
    const layer: BaseLayer = {
      id: `draw-${Date.now()}`,
      type: 'draw',
      visible: true,
      zIndex: layers.value.length + 1,
      locked: false,
      x: 0,
      y: 0,
      width: state.value.canvasWidth,
      height: state.value.canvasHeight,
      rotate: 0,
      scaleX: 1,
      scaleY: 1,
      draw: {
        points,
        strokeColor: color,
        strokeWidth,
      },
    }
    addLayer(layer)
  })
}

function bindDoubleClick(eng: NonNullable<typeof engine.value>) {
  eng.onDoubleClickEdit?.((id) => {
    const layer = layers.value.find((l) => l.id === id)
    if (!layer || layer.type !== 'text') return
    selectLayer(id)
    setMode(EditorMode.Text)
  })
}

async function handleEngineSwitch(type: EngineType) {
  if (type === engineType.value || !canvasRef.value) return
  handleCanvasRect()
  const oldEngine = engine.value
  const eng = await createEngine(unref(canvasRef)!, unref(canvasWidth), unref(canvasHeight), type)
  engineType.value = type
  setEngine(eng)
  bindDrawComplete(eng)
  bindDoubleClick(eng)
  eng.setDrawStyle?.(drawColor.value, drawWidth.value)
  eng.setSize(state.value.canvasWidth, state.value.canvasHeight)
  setMode(state.value.mode)
  await syncLayersToEngine()
  eng.fitToViewport?.()
  if (oldEngine) {
    oldEngine.destroy()
  }
}

function toggleCompress() {
  compressVisible.value = !compressVisible.value
  if (compressVisible.value) {
    setMode(EditorMode.Select)
  }
}

function closePanel() {
  compressVisible.value = false
  setMode(EditorMode.Select)
}

function handleDrawStyle(color: string, widthValue: number) {
  drawColor.value = color
  drawWidth.value = widthValue
  engine.value?.setDrawStyle?.(color, widthValue)
}

function clearDrawings() {
  const drawLayers = layers.value.filter((l) => l.type === 'draw')
  drawLayers.forEach((l) => removeLayer(l.id))
}

function handleAddText(options: {
  content: string
  fontSize: number
  color: string
  bold: boolean
}) {
  addTextLayer(options)
}

function handleUpdateText(
  id: string,
  options: { content: string; fontSize: number; color: string; bold: boolean },
) {
  const layer = layers.value.find((l) => l.id === id)
  if (!layer || layer.type !== 'text') return
  layer.text = { ...layer.text!, ...options }
  updateLayer(id, { text: layer.text })
}

function handleDeleteLayer() {
  const id = state.value.activeLayerId
  if (!id) return
  const layer = layers.value.find((l) => l.id === id)
  if (!layer || layer.type === 'image') return
  removeLayer(id)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key !== 'Delete' && e.key !== 'Backspace') return
  const target = e.target as HTMLElement | null
  if (
    target &&
    (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
  ) {
    return
  }
  handleDeleteLayer()
}

function handleAddSticker(url: string) {
  addStickerLayer(url)
}

function handleApplyFilter(type: FilterType) {
  void applyFilter(type)
}

async function handleCropApply(rect: CropRect) {
  await applyCrop(rect)
  setMode(EditorMode.Select)
}

function handleCropCancel() {
  setMode(EditorMode.Select)
}

async function handleRotate() {
  await rotate90()
  setMode(EditorMode.Select)
}

function handleReset() {
  resetEditor()
}

async function handleExport() {
  try {
    const blob = await exportToBlob({ format: 'jpeg', quality: 0.85 })
    emit('export', blob)
  } catch (error) {
    console.error('Export failed:', error)
  }
}

async function handleCompressExport(format: 'jpeg' | 'png', quality: number) {
  try {
    const blob = await exportToBlob({ format, quality })
    emit('export', blob)
  } catch (error) {
    console.error('Compress export failed:', error)
  }
}

function getCanvasContentSize() {
  const element = canvasEditRef.value ? unrefElement(canvasEditRef) : null
  if (!element) return { width: 0, height: 0 }
  const style = getComputedStyle(element)
  const width = Math.max(
    1,
    Math.floor(
      element.clientWidth - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight),
    ),
  )
  const height = Math.max(
    1,
    Math.floor(
      element.clientHeight - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom),
    ),
  )
  return { width, height }
}

function handleCanvasRect() {
  const { width, height } = getCanvasContentSize()
  canvasWidth.value = width
  canvasHeight.value = height
}

function fitCanvasToContainer() {
  const eng = engine.value
  if (!eng) return
  const { width, height } = getCanvasContentSize()
  eng.resizeBuffer?.(width, height)
  eng.fitToViewport?.()
}

let resizeObserver: ResizeObserver | null = null

function setupResizeObserver() {
  const element = canvasEditRef.value ? unrefElement(canvasEditRef) : null
  if (!element || resizeObserver) return
  resizeObserver = new ResizeObserver(() => {
    handleCanvasRect()
    fitCanvasToContainer()
  })
  resizeObserver.observe(element)
}

async function handleInit() {
  await nextTick()
  if (!canvasRef.value) return
  await initEngine(engineType.value)
}

function handledDestroy() {
  resizeObserver?.disconnect()
  resizeObserver = null
  if (engine.value) {
    engine.value.destroy()
  }
}

onMounted(async () => {
  await handleInit()
  setupResizeObserver()
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  handledDestroy()
})

defineExpose({
  handleExport,
  reset: handleInit,
  destroy: handledDestroy,
})
</script>

<style scoped>
.image-editor {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  max-height: 100%;
  background: #ffffff;
  border-radius: 0;
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
  flex-wrap: wrap;
  gap: 6px;
}

.image-editor__toolbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.engine-switch {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-right: 4px;
  padding: 2px;
  border: 1px solid #e5e5ea;
  border-radius: 8px;
  background: #f5f5f7;
}

.engine-switch__btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #86868b;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.engine-switch__btn:hover {
  color: #1d1d1f;
}

.engine-switch__btn--active {
  background: #ffffff;
  color: #007aff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.editor-divider {
  width: 1px;
  height: 18px;
  margin: 0 4px;
  background: #e5e5ea;
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
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: #f5f5f7;
  background-image: repeating-conic-gradient(#e8e8ed 0% 25%, #f5f5f7 0% 50%);
  background-size: 20px 20px;
}

.image-editor__canvas canvas {
  border-radius: 6px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
}

.image-editor__panel {
  position: relative;
  padding: 14px 16px;
  border-top: 1px solid #e5e5ea;
  background: #ffffff;
}

.image-editor__panel-close {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #86868b;
  cursor: pointer;
  transition: all 0.12s ease;
}

.image-editor__panel-close:hover {
  background: rgba(0, 0, 0, 0.06);
  color: #1d1d1f;
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

.image-editor__footer-actions {
  display: flex;
  gap: 8px;
}
</style>
