<template>
  <div class="crop-overlay" @pointerdown="startImagePan" @wheel.prevent="handleImageWheel">
    <div class="crop-box" :style="boxStyle" @pointerdown.stop="startMove">
      <span
        v-for="h in handles"
        :key="h"
        class="crop-handle"
        :class="`crop-handle--${h}`"
        @pointerdown.stop="startResize($event, h)"
      />
      <div class="crop-rule crop-rule--v1" />
      <div class="crop-rule crop-rule--v2" />
      <div class="crop-rule crop-rule--h1" />
      <div class="crop-rule crop-rule--h2" />
      <div class="crop-size">{{ Math.round(box.w) }} × {{ Math.round(box.h) }}</div>
    </div>

    <div class="crop-controls" @pointerdown.stop>
      <a-radio-group v-model="ratio" type="button" size="small">
        <a-radio value="free">自由</a-radio>
        <a-radio value="1:1">1:1</a-radio>
        <a-radio value="4:3">4:3</a-radio>
        <a-radio value="16:9">16:9</a-radio>
      </a-radio-group>
      <div class="crop-controls__actions">
        <a-button size="small" @click="emit('cancel')">取消</a-button>
        <a-button type="primary" size="small" @click="handleApply">应用</a-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import type { CropRect } from '../utils/image'
import type { Viewport } from '../ImageEditor.types'

const props = defineProps<{
  canvasWidth: number
  canvasHeight: number
  canvasEl: HTMLCanvasElement
  viewport: Viewport
}>()

const emit = defineEmits<{
  apply: [rect: CropRect]
  cancel: []
  panImage: [dx: number, dy: number]
  zoomImage: [factor: number]
}>()

const handles = ['nw', 'n', 'ne', 'w', 'e', 'sw', 's', 'se'] as const
const box = reactive({ x: 0, y: 0, w: 0, h: 0 })
const ratio = ref('free')
const canvasRect = reactive({ left: 0, top: 0, width: 0, height: 0 })

const aspectRatio = computed(() => {
  switch (ratio.value) {
    case '1:1':
      return 1
    case '4:3':
      return 4 / 3
    case '16:9':
      return 16 / 9
    default:
      return 0
  }
})

const contentRect = computed(() => {
  const vp = props.viewport
  const scale = vp.scale > 0 ? vp.scale : 1
  return {
    left: canvasRect.left + vp.offsetX,
    top: canvasRect.top + vp.offsetY,
    width: Math.min(props.canvasWidth * scale, canvasRect.width),
    height: Math.min(props.canvasHeight * scale, canvasRect.height),
  }
})

const boxStyle = computed(() => ({
  left: `${box.x}px`,
  top: `${box.y}px`,
  width: `${box.w}px`,
  height: `${box.h}px`,
}))

let dragState: {
  type: 'move' | 'resize'
  handle?: string
  startX: number
  startY: number
  orig: { x: number; y: number; w: number; h: number }
} | null = null

let imagePanState: { startX: number; startY: number } | null = null

onMounted(() => {
  updateCanvasRect()
  const rect = contentRect.value
  box.x = rect.left + rect.width * 0.05
  box.y = rect.top + rect.height * 0.05
  box.w = rect.width * 0.9
  box.h = rect.height * 0.9
})

watch(ratio, () => {
  const r = aspectRatio.value
  if (r <= 0) return
  const rect = contentRect.value
  const cx = box.x + box.w / 2
  const cy = box.y + box.h / 2
  const maxW = rect.width
  const maxH = rect.height
  let w = box.w
  let h = box.h
  if (w / h > r) {
    h = w / r
  } else {
    w = h * r
  }
  if (w > maxW) {
    w = maxW
    h = w / r
  }
  if (h > maxH) {
    h = maxH
    w = h * r
  }
  box.w = w
  box.h = h
  box.x = clamp(cx - w / 2, rect.left, rect.left + maxW - w)
  box.y = clamp(cy - h / 2, rect.top, rect.top + maxH - h)
})

function updateCanvasRect() {
  const rect = props.canvasEl.getBoundingClientRect()
  canvasRect.left = rect.left
  canvasRect.top = rect.top
  canvasRect.width = rect.width
  canvasRect.height = rect.height
}

function startMove(e: PointerEvent) {
  dragState = {
    type: 'move',
    startX: e.clientX,
    startY: e.clientY,
    orig: { ...box },
  }
  window.addEventListener('pointermove', handleDrag)
  window.addEventListener('pointerup', endDrag)
}

function startResize(e: PointerEvent, handle: string) {
  dragState = {
    type: 'resize',
    handle,
    startX: e.clientX,
    startY: e.clientY,
    orig: { ...box },
  }
  window.addEventListener('pointermove', handleDrag)
  window.addEventListener('pointerup', endDrag)
}

function handleDrag(e: PointerEvent) {
  if (!dragState) return
  const dx = e.clientX - dragState.startX
  const dy = e.clientY - dragState.startY
  const orig = dragState.orig
  const rect = contentRect.value

  if (dragState.type === 'move') {
    box.x = clamp(orig.x + dx, rect.left, rect.left + rect.width - box.w)
    box.y = clamp(orig.y + dy, rect.top, rect.top + rect.height - box.h)
  } else {
    resizeBox(dragState.handle!, dx, dy)
  }
}

function resizeBox(handle: string, dx: number, dy: number) {
  const orig = dragState!.orig
  const ratioVal = aspectRatio.value
  const rect = contentRect.value
  const minSize = 20
  const maxW = rect.width
  const maxH = rect.height

  let left = orig.x
  let top = orig.y
  let right = orig.x + orig.w
  let bottom = orig.y + orig.h

  if (handle.includes('w')) left = orig.x + dx
  if (handle.includes('e')) right = orig.x + orig.w + dx
  if (handle.includes('n')) top = orig.y + dy
  if (handle.includes('s')) bottom = orig.y + orig.h + dy

  let w = right - left
  let h = bottom - top

  if (ratioVal > 0) {
    const isWidthSide = handle.includes('w') || handle.includes('e')
    if (isWidthSide) {
      h = w / ratioVal
      if (handle.includes('n')) top = bottom - h
      else bottom = top + h
    } else {
      w = h * ratioVal
      if (handle.includes('w')) left = right - w
      else right = left + w
    }
  }

  if (w < minSize) {
    if (handle.includes('w')) left = right - minSize
    else right = left + minSize
    w = right - left
  }
  if (h < minSize) {
    if (handle.includes('n')) top = bottom - minSize
    else bottom = top + minSize
  }

  const cl = rect.left
  const ct = rect.top
  const cr = rect.left + maxW
  const cb = rect.top + maxH

  if (ratioVal > 0) {
    if (left < cl) left = cl
    if (top < ct) top = ct
    const rightBound = Math.min(right, cr)
    const bottomBound = Math.min(bottom, cb)
    const targetW = Math.min(w, rightBound - left, (bottomBound - top) * ratioVal)
    w = targetW
    h = targetW / ratioVal
    if (w < minSize || h < minSize) {
      left = cl + maxW * 0.05
      top = ct + maxH * 0.05
      w = maxW * 0.9
      h = w / ratioVal
      if (h > maxH * 0.9) {
        h = maxH * 0.9
        w = h * ratioVal
      }
    }
  } else {
    if (left < cl) left = cl
    if (top < ct) top = ct
    if (right > cr) right = cr
    if (bottom > cb) bottom = cb
    w = right - left
    h = bottom - top
  }

  box.x = left
  box.y = top
  box.w = w
  box.h = h
}

function endDrag() {
  dragState = null
  window.removeEventListener('pointermove', handleDrag)
  window.removeEventListener('pointerup', endDrag)
}

function startImagePan(e: PointerEvent) {
  if (e.target !== e.currentTarget) return
  imagePanState = { startX: e.clientX, startY: e.clientY }
  window.addEventListener('pointermove', handleImagePan)
  window.addEventListener('pointerup', endImagePan)
}

function handleImagePan(e: PointerEvent) {
  if (!imagePanState) return
  const dx = e.clientX - imagePanState.startX
  const dy = e.clientY - imagePanState.startY
  imagePanState.startX = e.clientX
  imagePanState.startY = e.clientY
  emit('panImage', dx, dy)
}

function endImagePan() {
  imagePanState = null
  window.removeEventListener('pointermove', handleImagePan)
  window.removeEventListener('pointerup', endImagePan)
}

function handleImageWheel(e: WheelEvent) {
  const factor = e.deltaY < 0 ? 1.1 : 0.9
  emit('zoomImage', factor)
}

function handleApply() {
  updateCanvasRect()
  const vp = props.viewport
  const scale = vp.scale > 0 ? vp.scale : 1
  const rect = contentRect.value
  const x = (box.x - rect.left) / scale
  const y = (box.y - rect.top) / scale
  const width = box.w / scale
  const height = box.h / scale
  emit('apply', {
    x: Math.round(Math.max(0, x)),
    y: Math.round(Math.max(0, y)),
    width: Math.round(Math.min(width, props.canvasWidth - x)),
    height: Math.round(Math.min(height, props.canvasHeight - y)),
  })
}

onBeforeUnmount(() => {
  endDrag()
  endImagePan()
})

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max)
}
</script>

<style scoped>
.crop-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
}

.crop-box {
  position: fixed;
  border: 1.5px solid #ffffff;
  box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.45);
  cursor: move;
}

.crop-handle {
  position: absolute;
  width: 10px;
  height: 10px;
  border: 1.5px solid #ffffff;
  background: #007aff;
}

.crop-handle::after {
  content: '';
  position: absolute;
  inset: -8px;
}

.crop-handle--nw {
  top: -5px;
  left: -5px;
  cursor: nwse-resize;
}

.crop-handle--n {
  top: -5px;
  left: 50%;
  transform: translateX(-50%);
  cursor: ns-resize;
}

.crop-handle--ne {
  top: -5px;
  right: -5px;
  cursor: nesw-resize;
}

.crop-handle--w {
  top: 50%;
  left: -5px;
  transform: translateY(-50%);
  cursor: ew-resize;
}

.crop-handle--e {
  top: 50%;
  right: -5px;
  transform: translateY(-50%);
  cursor: ew-resize;
}

.crop-handle--sw {
  bottom: -5px;
  left: -5px;
  cursor: nesw-resize;
}

.crop-handle--s {
  bottom: -5px;
  left: 50%;
  transform: translateX(-50%);
  cursor: ns-resize;
}

.crop-handle--se {
  bottom: -5px;
  right: -5px;
  cursor: nwse-resize;
}

.crop-rule {
  position: absolute;
  background: rgba(255, 255, 255, 0.5);
  pointer-events: none;
}

.crop-rule--v1 {
  left: 33.33%;
  top: 0;
  width: 1px;
  height: 100%;
}

.crop-rule--v2 {
  left: 66.66%;
  top: 0;
  width: 1px;
  height: 100%;
}

.crop-rule--h1 {
  top: 33.33%;
  left: 0;
  height: 1px;
  width: 100%;
}

.crop-rule--h2 {
  top: 66.66%;
  left: 0;
  height: 1px;
  width: 100%;
}

.crop-size {
  position: absolute;
  top: -24px;
  left: 50%;
  transform: translateX(-50%);
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.6);
  color: #ffffff;
  font-size: 11px;
  white-space: nowrap;
  pointer-events: none;
}

.crop-controls {
  position: fixed;
  left: 50%;
  bottom: 24px;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 16px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.16);
}

.crop-controls__actions {
  display: flex;
  gap: 8px;
}
</style>
