<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

function goHome() {
  router.push({ name: 'Dashboard' })
}
</script>

<template>
  <div class="flex items-center justify-center py-20">
    <div class="text-center px-6">
      <div class="text-[80px] font-bold text-[#E5E5EA] leading-none select-none">403</div>
      <h1 class="text-[24px] font-bold text-[#1D1D1F] mt-4 mb-2">无权访问</h1>
      <p class="text-[14px] text-[#86868B] mb-8 max-w-[360px] mx-auto">
        抱歉，你没有访问此页面的权限。如需获取权限，请联系管理员。
      </p>
      <a-button type="primary" size="large" @click="goHome">返回首页</a-button>
    </div>
  </div>
</template>
