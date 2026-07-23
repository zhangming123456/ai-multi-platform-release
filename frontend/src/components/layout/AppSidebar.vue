<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  IconHome,
  IconUser,
  IconFile,
  IconSend,
  IconApps,
  IconLeft,
  IconRight,
  IconExport,
  IconSettings,
  IconCode,
} from '@arco-design/web-vue/es/icon'

const props = defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  toggle: []
  closeMobile: []
}>()

const route = useRoute()
const router = useRouter()

const menuItems = [
  { path: '/', name: '仪表盘', key: 'dashboard', icon: IconHome },
  { path: '/accounts', name: '账号管理', key: 'accounts', icon: IconUser },
  { path: '/content', name: '内容工坊', key: 'content', icon: IconFile },
  { path: '/publish', name: '发布管理', key: 'publish', icon: IconSend },
  { path: '/templates', name: '模板中心', key: 'templates', icon: IconApps },
]

const settingsItems = [
  { path: '/settings/token-plan', name: 'Token 配置', key: 'token-plan', icon: IconSettings },
  { path: '/developer/docs', name: 'API 文档', key: 'api-docs', icon: IconCode },
]

const selectedKey = computed(() => {
  const match = [...menuItems, ...settingsItems].find((item) => {
    if (item.path === '/') return route.path === '/'
    return route.path.startsWith(item.path)
  })
  return match ? match.key : 'dashboard'
})

function onMenuItemClick(key: string) {
  const item = [...menuItems, ...settingsItems].find((m) => m.key === key)
  if (item) {
    router.push(item.path)
    emit('closeMobile')
  }
}
</script>

<template>
  <div class="flex flex-col h-full">
    <div :class="['px-5 pt-7 pb-6', { 'px-3': collapsed }]">
      <div class="flex items-center gap-3">
        <div
          class="w-10 h-10 rounded-[12px] flex items-center justify-center shrink-0"
          style="
            background: linear-gradient(135deg, #007aff 0%, #0055d4 100%);
            box-shadow:
              0 4px 14px rgba(0, 122, 255, 0.35),
              0 1px 3px rgba(0, 122, 255, 0.2);
          "
        >
          <span class="text-white font-bold text-[16px] tracking-tight">M</span>
        </div>
        <div v-if="!collapsed" class="flex-1 min-w-0">
          <p class="text-[16px] font-bold text-[#1D1D1F] tracking-[-0.01em] leading-tight">
            Matrix
          </p>
          <p class="text-[11px] text-[#86868B] leading-tight mt-0.5">Studio</p>
        </div>
      </div>
    </div>

    <p
      v-if="!collapsed"
      class="px-7 pb-3 text-[10px] font-semibold uppercase tracking-[0.1em] text-[#AEAEB2]"
    >
      Workspace
    </p>

    <a-menu
      :selected-keys="[selectedKey]"
      :collapsed="collapsed"
      :auto-open-selected="false"
      class="!bg-transparent !px-2 flex-1 overflow-y-auto"
      @menu-item-click="onMenuItemClick"
    >
      <a-menu-item v-for="item in menuItems" :key="item.key" class="!rounded-[10px] !mb-1">
        <template #icon>
          <component :is="item.icon" />
        </template>
        {{ item.name }}
      </a-menu-item>
      <a-menu-divider v-if="!collapsed" />
      <a-menu-item v-for="item in settingsItems" :key="item.key" class="!rounded-[10px] !mb-1">
        <template #icon>
          <component :is="item.icon" />
        </template>
        {{ item.name }}
      </a-menu-item>
    </a-menu>

    <div :class="['p-3 border-t border-black/[0.04]', { 'px-2': collapsed }]">
      <a-button
        v-if="!collapsed"
        type="text"
        long
        class="!rounded-[10px] !text-[#636366]"
        @click="emit('toggle')"
      >
        <template #icon>
          <IconLeft />
        </template>
        收起菜单
      </a-button>
      <a-button
        v-else
        type="text"
        long
        class="!rounded-[10px] !text-[#636366]"
        @click="emit('toggle')"
      >
        <template #icon>
          <IconRight />
        </template>
      </a-button>
    </div>

    <div v-if="!collapsed" class="p-3 border-t border-black/[0.04]">
      <div
        class="flex items-center gap-3 px-3 py-2.5 rounded-[12px] hover:bg-black/[0.03] cursor-pointer transition-all duration-250 group"
      >
        <a-avatar
          :size="36"
          class="shrink-0"
          style="background: linear-gradient(135deg, #30d158 0%, #007aff 100%)"
        >
          A
        </a-avatar>
        <div class="flex-1 min-w-0">
          <p class="text-[14px] font-semibold text-[#1D1D1F] truncate leading-tight">Admin</p>
          <p class="text-[11px] text-[#86868B] truncate leading-tight mt-0.5">admin@matrix.com</p>
        </div>
        <div
          class="p-1.5 rounded-[8px] hover:bg-[#FF3B30]/10 text-[#86868B] hover:text-[#FF3B30] transition-all duration-250 shrink-0 opacity-0 group-hover:opacity-100"
        >
          <IconExport :size="15" />
        </div>
      </div>
    </div>
  </div>
</template>
