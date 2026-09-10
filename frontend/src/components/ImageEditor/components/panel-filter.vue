<template>
  <div class="panel">
    <div class="filter-list">
      <button
        v-for="f in filters"
        :key="f.type"
        type="button"
        class="filter-item"
        :class="{ 'filter-item--active': active === f.type }"
        @click="handleApply(f.type)"
      >
        <span class="filter-item__dot" :style="{ background: f.color }" />
        <span>{{ f.label }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { FilterType } from '../utils/image.types'
import type { PanelFilterEmits } from './panel-filter.types'

const emit = defineEmits<PanelFilterEmits>()

const active = ref<FilterType>('none')

const filters: { type: FilterType; label: string; color: string }[] = [
  { type: 'none', label: '原图', color: 'linear-gradient(135deg, #ff9a9e, #fad0c4)' },
  { type: 'grayscale', label: '灰度', color: 'linear-gradient(135deg, #86868b, #3a3a3c)' },
  { type: 'sepia', label: '怀旧', color: 'linear-gradient(135deg, #b5a06d, #6e5a3a)' },
  { type: 'invert', label: '反色', color: 'linear-gradient(135deg, #36d1dc, #5b86e5)' },
  { type: 'pixelate', label: '像素', color: 'linear-gradient(135deg, #f7971e, #ffd200)' },
  { type: 'emboss', label: '浮雕', color: 'linear-gradient(135deg, #8e9eab, #eef2f3)' },
]

function handleApply(type: FilterType) {
  active.value = type
  emit('apply', type)
}
</script>

<style scoped>
.panel {
  display: flex;
}

.filter-list {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 8px;
  width: 100%;
}

.filter-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 4px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: #f5f5f7;
  font-size: 11px;
  color: #1d1d1f;
  cursor: pointer;
  transition: all 0.12s ease;
}

.filter-item:hover {
  border-color: #007aff;
}

.filter-item--active {
  background: rgba(0, 122, 255, 0.1);
  border-color: #007aff;
  color: #007aff;
}

.filter-item__dot {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.08);
}
</style>
