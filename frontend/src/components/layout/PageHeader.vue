<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<template>
  <a-space
    direction="horizontal"
    :align="'end'"
    :size="16"
    fill
    class="w-full animate-fade-up mb-7"
  >
    <a-space direction="vertical" :size="4" class="flex-1">
      <h1 class="text-[28px] font-bold tracking-[-0.025em] text-[#1D1D1F] leading-[1.15] m-0">
        {{ title }}
      </h1>
      <p v-if="subtitle" class="text-[13px] text-[#86868B] m-0">{{ subtitle }}</p>
      <slot name="below-title"></slot>
    </a-space>
    <div class="flex items-center gap-2.5 shrink-0">
      <slot name="actions"></slot>
    </div>
  </a-space>
</template>
