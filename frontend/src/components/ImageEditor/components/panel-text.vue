<template>
  <div class="panel">
    <div class="panel__row">
      <span class="panel__label">内容</span>
      <a-input v-model="content" placeholder="请输入文字" size="small" @press-enter="handleAdd" />
    </div>
    <div class="panel__row">
      <span class="panel__label">字号</span>
      <a-slider v-model="fontSize" :min="12" :max="96" style="flex: 1" />
      <span class="panel__value">{{ fontSize }}px</span>
    </div>
    <div class="panel__row">
      <span class="panel__label">颜色</span>
      <div class="swatches">
        <button
          v-for="c in swatches"
          :key="c"
          class="swatch"
          :class="{ 'swatch--active': color === c }"
          :style="{ background: c }"
          @click="color = c"
        />
      </div>
    </div>
    <div class="panel__row">
      <span class="panel__label">加粗</span>
      <a-switch v-model="bold" size="small" />
    </div>
    <div class="panel__actions">
      <a-button type="primary" size="small" :disabled="!content.trim()" @click="handleAdd"
        >添加</a-button
      >
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  add: [options: { content: string; fontSize: number; color: string; bold: boolean }]
}>()

const swatches = [
  '#1d1d1f',
  '#ffffff',
  '#ff4d4f',
  '#ff7a45',
  '#fadb14',
  '#52c41a',
  '#007aff',
  '#722ed1',
]

const content = ref('')
const fontSize = ref(36)
const color = ref('#1d1d1f')
const bold = ref(false)

function handleAdd() {
  const text = content.value.trim()
  if (!text) return
  emit('add', {
    content: text,
    fontSize: fontSize.value,
    color: color.value,
    bold: bold.value,
  })
  content.value = ''
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
