<template>
  <div class="panel">
    <div class="panel__row">
      <span class="panel__label">颜色</span>
      <div class="swatches">
        <button
          v-for="c in swatches"
          :key="c"
          class="swatch"
          :class="{ 'swatch--active': color === c }"
          :style="{ background: c }"
          @click="selectColor(c)"
        />
      </div>
    </div>
    <div class="panel__row">
      <span class="panel__label">粗细</span>
      <a-slider v-model="width" :min="1" :max="24" style="flex: 1" @change="emitStyle" />
      <span class="panel__value">{{ width }}px</span>
    </div>
    <div class="panel__actions">
      <a-button size="small" status="danger" @click="emit('clear')">清除涂鸦</a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  'style-change': [color: string, width: number]
  clear: []
}>()

const swatches = [
  '#1d1d1f',
  '#ff4d4f',
  '#ff7a45',
  '#fadb14',
  '#52c41a',
  '#13c2c2',
  '#007aff',
  '#722ed1',
  '#ffffff',
]

const color = ref('#007aff')
const width = ref(6)

function selectColor(c: string) {
  color.value = c
  emitStyle()
}

function emitStyle() {
  emit('style-change', color.value, width.value)
}
</script>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.panel__row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.panel__label {
  flex-shrink: 0;
  width: 36px;
  font-size: 12px;
  color: #86868b;
}

.panel__value {
  flex-shrink: 0;
  min-width: 40px;
  font-size: 12px;
  color: #1d1d1f;
  text-align: right;
}

.panel__actions {
  display: flex;
  justify-content: flex-end;
}

.swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.swatch {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid rgba(0, 0, 0, 0.12);
  cursor: pointer;
  transition: transform 0.12s ease;
}

.swatch:hover {
  transform: scale(1.15);
}

.swatch--active {
  outline: 2px solid #007aff;
  outline-offset: 2px;
}
</style>
