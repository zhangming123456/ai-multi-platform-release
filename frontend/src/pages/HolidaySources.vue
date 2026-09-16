<template>
  <div class="page-main">
    <PageHeader
      title="日历订阅源"
      subtitle="管理节假日日历的 ICS 订阅源，支持多源合并、自动刷新与手动拉取，数据落库并记录更新时间"
    />

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <HolidaySourceManager />
    </div>
  </div>
</template>

<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import HolidaySourceManager from '@/components/HolidaySourceManager.vue'
</script>
