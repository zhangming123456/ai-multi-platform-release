<script setup lang="ts">
import type { Component } from 'vue'
import { IconArrowRise, IconArrowFall } from '@arco-design/web-vue/es/icon'

defineProps<{
  title: string
  value: number
  trend: string
  icon: Component
  color: string
  delay?: number
}>()
</script>

<template>
  <a-card class="stat-card" :bordered="false" :style="{ animationDelay: `${delay || 0}ms` }">
    <div class="flex items-center justify-between mb-4">
      <div
        class="w-10 h-10 rounded-[10px] flex items-center justify-center"
        :style="{ background: color, boxShadow: `0 2px 8px ${color}44` }"
      >
        <component :is="icon" :size="19" :stroke-width="2" class="text-white" />
      </div>
      <a-space :size="4" align="center">
        <IconArrowRise
          v-if="trend.startsWith('+')"
          :size="14"
          style="color: var(--arco-theme-success)"
        />
        <IconArrowFall v-else :size="14" style="color: var(--arco-theme-danger)" />
        <a-tag :color="trend.startsWith('+') ? 'green' : 'red'" size="small">
          {{ trend }}
        </a-tag>
      </a-space>
    </div>
    <a-statistic :value="value" :title="title" />
  </a-card>
</template>

<style scoped>
.stat-card {
  animation: fade-up 0.5s ease both;
  transition:
    box-shadow 0.2s ease,
    transform 0.2s ease;
}
.stat-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}
@keyframes fade-up {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
