<template>
  <div class="crop-overlay">
    <div class="crop-box" :style="boxStyle" @pointerdown="startMove">
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

    <div class="crop-controls">
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
import { reactive, ref, computed, onMounted, onBeforeUnmount } from 'vue'
import type { CropRect } from '../utils/image'
import type { Viewport } from '../types'

const props = defineProps<{
  canvasWidth: number
  canvasHeight: number
  canvasEl: HTMLCanvasElement
  viewport: Viewport
}>()

const emit = defineEmits<{
  apply: [rect: CropRect]
  cancel: []
}>()

const handles = ['nw', 'ne', 'sw', 'se'] as const
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

onMounted(() => {
  updateCanvasRect()
  const rect = contentRect.value
  box.x = rect.left + rect.width * 0.05
  box.y = rect.top + rect.height * 0.05
  box.w = rect.width * 0.9
  box.h = rect.height * 0.9
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

function resizeBox(handle: string, _dx: number, dy: number) {
  const orig = dragState!.orig
  const ratioVal = aspectRatio.value
  const rect = contentRect.value
  let { x, y, w } = orig
  let h: number

  if (handle.startsWith('n')) {
    h = clamp(orig.h - dy, 10, rect.height)
    y = orig.y + (orig.h - h)
  } else {
    h = clamp(orig.h + dy, 10, rect.height)
  }

  if (ratioVal > 0) {
    w = h * ratioVal
  }

  if (handle.endsWith('w')) {
    x = orig.x + (orig.w - w)
    w = clamp(w, 10, rect.width)
    if (x < rect.left) {
      x = rect.left
    }
  } else {
    w = clamp(w, 10, rect.width)
  }

  box.x = clamp(x, rect.left, rect.left + rect.width - w)
  box.y = clamp(y, rect.top, rect.top + rect.height - h)
  box.w = w
  box.h = h
}

function endDrag() {
  dragState = null
  window.removeEventListener('pointermove', handleDrag)
  window.removeEventListener('pointerup', endDrag)
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

.crop-handle--nw {
  top: -5px;
  left: -5px;
  cursor: nwse-resize;
}

.crop-handle--ne {
  top: -5px;
  right: -5px;
  cursor: nesw-resize;
}

.crop-handle--sw {
  bottom: -5px;
  left: -5px;
  cursor: nesw-resize;
}

.crop-handle--se {
  bottom: -5px;
  right: -5px;
  cursor: nwse-resize;
}

.crop-rule {
  position: absolute;
  background: rgba(255, 255, 255, 0.5);
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
