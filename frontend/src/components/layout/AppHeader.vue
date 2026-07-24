<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { IconMenuFold, IconMenuUnfold } from '@arco-design/web-vue/es/icon'
import RegionSwitcher from './RegionSwitcher.vue'
import NotificationBell from '@/components/NotificationBell.vue'

const props = defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  toggleSidebar: []
}>()

const route = useRoute()

const breadcrumbMap: Record<string, string> = {
  '/': '仪表盘',
  '/profile': '个人资料',
  '/platforms': '平台管理',
  '/accounts': '账号管理',
  '/content': '内容工坊',
  '/content/create': '创建内容',
  '/publish': '发布管理',
  '/review': '审核管理',
  '/templates': '模板中心',
  '/settings/token-plan': 'Token 配置',
  '/developer/docs': 'API 文档',
}

const breadcrumbs = computed(() => {
  const path = route.path
  const items: { label: string; path: string }[] = [{ label: 'Matrix', path: '/' }]
  if (path !== '/') {
    items.push({ label: breadcrumbMap[path] || '页面', path })
  }
  return items
})
</script>

<template>
  <header
    class="h-[56px] bg-white/60 backdrop-blur-xl border-b border-black/[0.04] flex items-center justify-between px-4 md:px-8 sticky top-0 z-20"
  >
    <div class="flex items-center gap-4">
      <!-- 汉堡菜单 / 折叠切换 -->
      <a-button type="text" size="small" class="!rounded-[10px]" @click="emit('toggleSidebar')">
        <template #icon>
          <component :is="collapsed ? IconMenuUnfold : IconMenuFold" :size="20" />
        </template>
      </a-button>

      <!-- 面包屑 -->
      <a-breadcrumb class="hidden sm:block">
        <a-breadcrumb-item
          v-for="(item, index) in breadcrumbs"
          :key="item.path"
          :class="
            index === breadcrumbs.length - 1 ? '!font-semibold !text-[#1D1D1F]' : '!text-[#86868B]'
          "
        >
          {{ item.label }}
        </a-breadcrumb-item>
      </a-breadcrumb>
    </div>

    <div class="flex items-center gap-3">
      <RegionSwitcher />

      <!-- 搜索框 -->
      <a-input-search
        placeholder="搜索..."
        class="hidden md:block"
        :style="{ width: '224px' }"
        allow-clear
      />

      <!-- 通知铃铛 -->
      <NotificationBell />
    </div>
  </header>
</template>

<style scoped>
@media (max-width: 248px) {
  header {
    height: 44px !important;
    padding-left: 10px !important;
    padding-right: 10px !important;
  }
}
</style>
