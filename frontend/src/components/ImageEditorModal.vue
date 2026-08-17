<template>
  <a-modal
    :visible="visible"
    :footer="false"
    :closable="true"
    :width="880"
    wrap-class-name="image-editor-modal"
    @cancel="close"
  >
    <div class="editor-wrap">
      <div class="editor-toolbar">
        <div class="editor-toolbar-left">
          <template v-if="!isCropMode">
            <button type="button" class="editor-btn" :disabled="!image" @click="startCrop">
              <IconScissor :size="16" />
              <span>裁剪</span>
            </button>
            <button type="button" class="editor-btn" :disabled="!image" @click="rotate(-90)">
              <IconRotateLeft :size="16" />
              <span>左转</span>
            </button>
            <button type="button" class="editor-btn" :disabled="!image" @click="rotate(90)">
              <IconRotateRight :size="16" />
              <span>右转</span>
            </button>
            <button type="button" class="editor-btn" :disabled="!image" @click="reset">
              <IconRefresh :size="16" />
              <span>重置</span>
            </button>
            <div class="editor-quality">
              <span class="editor-quality-label">质量</span>
              <a-slider
                v-model="quality"
                :min="0.3"
                :max="1"
                :step="0.05"
                :style="{ width: '110px' }"
                size="small"
              />
              <span class="editor-quality-value">{{ Math.round(quality * 100) }}%</span>
            </div>
          </template>
          <template v-else>
            <span class="editor-crop-hint">拖动选框选择裁剪区域</span>
          </template>
        </div>
        <div class="editor-toolbar-right">
          <template v-if="isCropMode">
            <a-button size="small" @click="cancelCrop">取消</a-button>
            <a-button size="small" type="primary" @click="applyCrop">应用裁剪</a-button>
          </template>
          <template v-else>
            <a-button size="small" @click="close">取消</a-button>
            <a-button size="small" type="primary" :loading="isExporting" @click="confirm">
              <template #icon><IconCheck :size="14" /></template>
              确定
            </a-button>
          </template>
        </div>
      </div>

      <div class="editor-stage">
        <canvas ref="canvasRef" class="editor-canvas"></canvas>
      </div>

      <div class="editor-footbar">
        <span class="editor-foot-text"
          >编辑后将以 JPEG 格式导出（质量 {{ Math.round(quality * 100) }}%）</span
        >
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount } from 'vue'
import { Message } from '@arco-design/web-vue'
import { Canvas, FabricImage, Rect } from 'fabric'
import {
  IconScissor,
  IconRotateLeft,
  IconRotateRight,
  IconRefresh,
  IconCheck,
} from '@arco-design/web-vue/es/icon'

interface EditorResult {
  blob: Blob
  name: string
}

const props = defineProps<{
  visible: boolean
  url: string
  name: string
}>()

const emit = defineEmits<{
  close: []
  confirm: [result: EditorResult]
}>()

const CANVAS_W = 820
const CANVAS_H = 540

const canvasRef = ref<HTMLCanvasElement | null>(null)
const isCropMode = ref(false)
const isExporting = ref(false)
const quality = ref(0.85)

let canvas: Canvas | null = null
let image: FabricImage | null = null
let cropRect: Rect | null = null

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    nextTick(async () => {
      if (!canvasRef.value) return
      if (!canvas) {
        canvas = new Canvas(canvasRef.value, {
          width: CANVAS_W,
          height: CANVAS_H,
          selection: false,
          preserveObjectStacking: true,
        })
      }
      await loadImage(props.url)
    })
  },
)

onBeforeUnmount(() => {
  canvas?.dispose()
  canvas = null
})

async function loadImage(url: string) {
  if (!canvas) return
  canvas.clear()
  image = null
  cropRect = null
  isCropMode.value = false
  try {
    const img = await FabricImage.fromURL(url)
    const width = img.width || 1
    const height = img.height || 1
    const scale = Math.min((CANVAS_W - 32) / width, (CANVAS_H - 32) / height, 1)
    img.scale(scale)
    img.set({
      originX: 'center',
      originY: 'center',
      left: CANVAS_W / 2,
      top: CANVAS_H / 2,
    })
    canvas.add(img)
    canvas.setActiveObject(img)
    image = img
    canvas.renderAll()
    console.log('[ImageEditor] image loaded:', url.slice(0, 30), 'w/h:', width, height)
  } catch (err) {
    console.error('[ImageEditor] load failed:', url.slice(0, 30), err)
    Message.error('图片加载失败，无法进入编辑模式')
  }
}

function startCrop() {
  console.log('[ImageEditor] startCrop:', {
    canvas: !!canvas,
    image: !!image,
    cropRect: !!cropRect,
  })
  if (!canvas || !image || cropRect) return
  const b = image.getBoundingRect()
  cropRect = new Rect({
    left: b.left,
    top: b.top,
    width: b.width,
    height: b.height,
    fill: 'rgba(0, 122, 255, 0.06)',
    stroke: '#007AFF',
    strokeWidth: 1.5,
    strokeUniform: true,
    strokeDashArray: [6, 4],
    cornerColor: '#007AFF',
    cornerSize: 12,
    transparentCorners: false,
    lockRotation: true,
    borderColor: '#007AFF',
  })
  canvas.add(cropRect)
  canvas.setActiveObject(cropRect)
  isCropMode.value = true
  canvas.renderAll()
}

function cancelCrop() {
  if (!canvas || !cropRect) return
  canvas.remove(cropRect)
  cropRect = null
  isCropMode.value = false
  canvas.renderAll()
}

async function applyCrop() {
  if (!canvas || !cropRect) return
  const b = cropRect.getBoundingRect()
  canvas.remove(cropRect)
  cropRect = null
  isCropMode.value = false
  const dataUrl = exportDataUrl({
    left: b.left,
    top: b.top,
    width: b.width,
    height: b.height,
  })
  if (dataUrl) {
    await loadImage(dataUrl)
  }
}

function rotate(degrees: number) {
  if (!image) return
  image.rotate((image.angle || 0) + degrees)
  canvas?.renderAll()
}

function reset() {
  loadImage(props.url)
}

function exportDataUrl(region?: { left: number; top: number; width: number; height: number }) {
  if (!canvas) return ''
  const prevBg = canvas.backgroundColor
  canvas.backgroundColor = '#ffffff'
  canvas.renderAll()
  const dataUrl = canvas.toDataURL({
    format: 'jpeg',
    quality: quality.value,
    multiplier: 1,
    ...(region ? { ...region } : {}),
  })
  canvas.backgroundColor = prevBg
  canvas.renderAll()
  return dataUrl
}

async function confirm() {
  if (!canvas) return
  if (cropRect) {
    await applyCrop()
  }
  isExporting.value = true
  try {
    const prevBg = canvas.backgroundColor
    canvas.backgroundColor = '#ffffff'
    canvas.renderAll()
    const blob = await canvas.toBlob({
      format: 'jpeg',
      quality: quality.value,
      multiplier: 1,
    })
    canvas.backgroundColor = prevBg
    canvas.renderAll()
    if (!blob) {
      Message.error('导出图片失败，请重试')
      return
    }
    const baseName = props.name.replace(/\.[^.]+$/, '') || 'image'
    emit('confirm', { blob, name: `${baseName}_edited.jpg` })
    close()
  } catch {
    Message.error('导出图片失败，请重试')
  } finally {
    isExporting.value = false
  }
}

function close() {
  emit('close')
}
</script>

<style scoped>
.editor-wrap {
  padding: 0;
}

.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.editor-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.editor-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.editor-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  background: #ffffff;
  color: #1d1d1f;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.editor-btn:hover:not(:disabled) {
  border-color: #007aff;
  color: #007aff;
  background: rgba(0, 122, 255, 0.04);
}

.editor-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.editor-quality {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 8px;
  padding-left: 12px;
  border-left: 1px solid rgba(0, 0, 0, 0.08);
}

.editor-quality-label {
  font-size: 12px;
  color: #86868b;
  white-space: nowrap;
}

.editor-quality-value {
  font-size: 12px;
  color: #1d1d1f;
  font-variant-numeric: tabular-nums;
  min-width: 34px;
  text-align: right;
}

.editor-crop-hint {
  font-size: 12px;
  color: #007aff;
}

.editor-stage {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: #f5f5f7;
  background-image: repeating-conic-gradient(#e8e8ed 0% 25%, #f5f5f7 0% 50%);
  background-size: 20px 20px;
  border-radius: 0;
}

.editor-canvas {
  max-width: 100%;
  border-radius: 6px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.editor-footbar {
  padding: 8px 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.editor-foot-text {
  font-size: 11px;
  color: #86868b;
}
</style>
