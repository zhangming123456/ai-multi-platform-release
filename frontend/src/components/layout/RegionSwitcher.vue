<script setup lang="ts">
import { IconLanguage } from '@arco-design/web-vue/es/icon'
import { useRegionStore } from '@/stores/region'

const { selectedTz, regions, switchRegion, currentRegion } = useRegionStore()
</script>

<template>
  <a-select
    :model-value="selectedTz"
    size="small"
    :bordered="false"
    class="region-select"
    :trigger-props="{ autoFitPopupWidth: true }"
    @change="(value: string | number | boolean | Record<string, any> | undefined) => {
      if (typeof value === 'string') switchRegion(value)
    }"
  >
    <template #prefix>
      <IconLanguage :size="16" />
    </template>
    <a-option
      v-for="region in regions"
      :key="region.value"
      :value="region.value"
      :label="region.label"
    >
      <div class="region-option">
        <span class="region-option__label">{{ region.label }}</span>
        <span class="region-option__offset">{{ region.offset }}</span>
      </div>
    </a-option>
  </a-select>
</template>

<style scoped>
.region-select {
  width: auto;
  min-width: 90px;
}

.region-select :deep(.arco-select-view) {
  background: transparent !important;
  color: #636366;
  font-size: 12px;
  border-radius: 10px;
  height: 32px;
  padding: 0 8px;
}

.region-select :deep(.arco-select-view:hover) {
  background: rgba(0, 0, 0, 0.04) !important;
  color: #1d1d1f;
}

.region-select :deep(.arco-select-prefix) {
  padding-right: 4px;
}

.region-select :deep(.arco-select-view-value) {
  font-weight: 500;
}

.region-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 140px;
  gap: 12px;
}

.region-option__label {
  font-size: 13px;
  font-weight: 500;
  color: #1d1d1f;
}

.region-option__offset {
  font-size: 11px;
  color: #86868b;
  font-variant-numeric: tabular-nums;
}
</style>
