<template>
  <div class="panel">
    <div class="panel__row">
      <span class="panel__label">格式</span>
      <a-radio-group v-model="format" type="button" size="small">
        <a-radio value="jpeg">JPEG</a-radio>
        <a-radio value="png">PNG</a-radio>
      </a-radio-group>
    </div>
    <div class="panel__row">
      <span class="panel__label">质量</span>
      <a-slider
        v-model="quality"
        :min="10"
        :max="100"
        :step="5"
        :disabled="format === 'png'"
        style="flex: 1"
      />
      <span class="panel__value">{{ format === 'png' ? '无损' : quality + '%' }}</span>
    </div>
    <div class="panel__row">
      <span class="panel__label">尺寸</span>
      <span class="panel__hint">导出原尺寸图片</span>
    </div>
    <div class="panel__actions">
      <a-button type="primary" size="small" :loading="exporting" @click="handleExport">
        导出压缩图
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  export: [format: 'jpeg' | 'png', quality: number]
}>()

const format = ref<'jpeg' | 'png'>('jpeg')
const quality = ref(80)
const exporting = ref(false)

async function handleExport() {
  if (exporting.value) return
  exporting.value = true
  try {
    emit('export', format.value, format.value === 'png' ? 1 : quality.value / 100)
  } finally {
    exporting.value = false
  }
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

.panel__hint {
  font-size: 12px;
  color: #86868b;
}

.panel__actions {
  display: flex;
  justify-content: flex-end;
}
</style>
