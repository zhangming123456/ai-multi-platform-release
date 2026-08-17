<template>
  <a-modal
    :visible="visible"
    :footer="false"
    :closable="true"
    :width="'auto'"
    :body-style="{ padding: '0', overflow: 'hidden', borderRadius: '16px' }"
    wrap-class-name="image-viewer-modal"
    @cancel="close"
  >
    <div class="viewer-wrap">
      <div
        class="viewer-stage"
        @mousedown="onDragStart"
        @mousemove="onDragMove"
        @mouseup="onDragEnd"
        @mouseleave="onDragEnd"
        @wheel.prevent="onWheel"
      >
        <img
          v-if="currentItem"
          :src="currentItem.url"
          :style="imgStyle"
          class="viewer-img"
          draggable="false"
        />
      </div>

      <div class="viewer-topbar">
        <span class="viewer-title">{{ currentItem?.name || '' }}</span>
        <div class="viewer-topbar-actions">
          <button type="button" class="viewer-btn" title="缩小" @click="zoomBy(-0.25)">
            <IconZoomOut :size="18" />
          </button>
          <span class="viewer-percent">{{ Math.round(scale * 100) }}%</span>
          <button type="button" class="viewer-btn" title="放大" @click="zoomBy(0.25)">
            <IconZoomIn :size="18" />
          </button>
          <button type="button" class="viewer-btn" title="重置" @click="reset">
            <IconRefresh :size="18" />
          </button>
        </div>
      </div>

      <div class="viewer-bottombar">
        <button
          v-if="items.length > 1"
          type="button"
          class="viewer-btn"
          title="上一张"
          :disabled="index === 0"
          @click="emit('prev')"
        >
          <IconLeft :size="18" />
        </button>
        <span v-if="items.length > 1" class="viewer-count"
          >{{ index + 1 }} / {{ items.length }}</span
        >
        <button
          v-if="items.length > 1"
          type="button"
          class="viewer-btn"
          title="下一张"
          :disabled="index === items.length - 1"
          @click="emit('next')"
        >
          <IconRight :size="18" />
        </button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  IconZoomIn,
  IconZoomOut,
  IconRefresh,
  IconLeft,
  IconRight,
} from '@arco-design/web-vue/es/icon'

export interface ViewerItem {
  url: string
  name: string
}

const props = defineProps<{
  visible: boolean
  items: ViewerItem[]
  index: number
}>()

const emit = defineEmits<{
  close: []
  prev: []
  next: []
}>()

const scale = ref(1)
const offset = ref({ x: 0, y: 0 })
const isDragging = ref(false)
let startPos = { x: 0, y: 0 }

watch(
  () => [props.visible, props.index] as const,
  () => {
    if (props.visible) {
      reset()
    }
  },
)

const currentItem = computed(() => props.items[props.index] || null)

const imgStyle = computed(() => ({
  transform: `translate(${offset.value.x}px, ${offset.value.y}px) scale(${scale.value})`,
  transition: isDragging.value ? 'none' : 'transform 0.15s ease',
}))

function zoomBy(delta: number) {
  scale.value = Math.min(4, Math.max(0.25, scale.value + delta))
}

function reset() {
  scale.value = 1
  offset.value = { x: 0, y: 0 }
}

function onWheel(e: WheelEvent) {
  const delta = e.deltaY > 0 ? -0.12 : 0.12
  zoomBy(delta)
}

function onDragStart(e: MouseEvent) {
  isDragging.value = true
  startPos = { x: e.clientX, y: e.clientY }
}

function onDragMove(e: MouseEvent) {
  if (!isDragging.value) return
  offset.value = {
    x: offset.value.x + (e.clientX - startPos.x),
    y: offset.value.y + (e.clientY - startPos.y),
  }
  startPos = { x: e.clientX, y: e.clientY }
}

function onDragEnd() {
  isDragging.value = false
}

function close() {
  emit('close')
}
</script>

<style scoped>
.viewer-wrap {
  position: relative;
  width: min(80vw, 900px);
  height: min(78vh, 640px);
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0b0b0f;
}

.viewer-stage {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  cursor: grab;
  user-select: none;
}

.viewer-stage:active {
  cursor: grabbing;
}

.viewer-img {
  max-width: 92%;
  max-height: 88%;
  object-fit: contain;
  border-radius: 8px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  transform-origin: center center;
  will-change: transform;
}

.viewer-topbar {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.55), transparent);
}

.viewer-title {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.92);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 55%;
}

.viewer-topbar-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.viewer-percent {
  min-width: 44px;
  text-align: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.85);
  font-variant-numeric: tabular-nums;
}

.viewer-bottombar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 12px 16px;
  background: linear-gradient(0deg, rgba(0, 0, 0, 0.55), transparent);
}

.viewer-count {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.85);
  font-variant-numeric: tabular-nums;
}

.viewer-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.12);
  color: #ffffff;
  cursor: pointer;
  transition: background 0.15s;
}

.viewer-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.24);
}

.viewer-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
</style>
