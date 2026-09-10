<template>
  <div class="panel">
    <div class="panel__row">
      <span class="panel__label">内容</span>
      <a-input
        v-model="content"
        placeholder="请输入文字"
        size="small"
        @press-enter="handleSubmit"
      />
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
      <a-button type="primary" size="small" :disabled="!content.trim()" @click="handleSubmit">{{
        editingId ? '更新' : '添加'
      }}</a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { PanelTextEmits, PanelTextProps } from './panel-text.types'

const props = defineProps<PanelTextProps>()

const emit = defineEmits<PanelTextEmits>()

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
const editingId = ref<string | null>(null)

watch(
  () => props.editLayer,
  (val) => {
    if (val && val.type === 'text' && val.text) {
      editingId.value = val.id
      content.value = val.text.content
      fontSize.value = val.text.fontSize
      color.value = val.text.color
      bold.value = val.text.bold
    }
  },
  { immediate: true },
)

function handleSubmit() {
  const text = content.value.trim()
  if (!text) return
  const options = {
    content: text,
    fontSize: fontSize.value,
    color: color.value,
    bold: bold.value,
  }
  if (editingId.value) {
    emit('update', editingId.value, options)
  } else {
    emit('add', options)
  }
  content.value = ''
  editingId.value = null
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
