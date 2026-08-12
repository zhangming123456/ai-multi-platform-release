<template>
  <a-space
    direction="horizontal"
    :align="'end'"
    :size="16"
    fill
    class="page-header w-full animate-fade-up mb-7 px-4 md:px-6 lg:px-8 pt-4 md:pt-6 lg:pt-8"
  >
    <a-space direction="vertical" :size="4" class="flex-1 page-header__left">
      <h1
        class="page-header__title text-[28px] font-bold tracking-[-0.025em] text-[#1D1D1F] leading-[1.15] m-0"
      >
        {{ title }}
      </h1>
      <p v-if="subtitle" class="page-header__subtitle text-[13px] text-[#86868B] m-0">
        {{ subtitle }}
      </p>
      <div class="page-header__below">
        <slot name="below-title"></slot>
      </div>
    </a-space>
    <div class="flex items-center gap-2.5 shrink-0 page-header__actions">
      <slot name="actions"></slot>
    </div>
  </a-space>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<style scoped lang="scss">
@media (max-width: 768px) {
  .page-header {
    margin-bottom: 20px !important;
    flex-direction: column !important;
    align-items: flex-start !important;
    gap: 12px !important;
  }
  .page-header__left {
    width: 100%;
  }
  .page-header__title {
    font-size: 22px !important;
  }
  .page-header__subtitle {
    font-size: 12px !important;
  }
  .page-header__actions {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}

@media (max-width: 480px) {
  .page-header__title {
    font-size: 20px !important;
  }
}

@media (max-width: 248px) {
  .page-header {
    margin-bottom: 12px !important;
    gap: 8px !important;
  }
  .page-header__title {
    font-size: 16px !important;
  }
  .page-header__subtitle {
    font-size: 11px !important;
  }
  .page-header__below {
    flex-wrap: wrap;
  }
}

.page-header__below {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
</style>
