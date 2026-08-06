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

    <a-menu
      :selected-keys="[selectedKey]"
      v-model:open-keys="openKeys"
      :collapsed="collapsed"
      class="!bg-transparent !px-2 flex-1 overflow-y-auto"
      @menu-item-click="onMenuItemClick"
    >
      <template v-for="item in menuItems" :key="item.key">
        <a-sub-menu v-if="isGroup(item)" :key="item.key">
          <template #icon>
            <component :is="item.icon" />
          </template>
          <template #title>{{ item.name }}</template>
          <a-menu-item
            v-for="child in item.children"
            :key="child.key"
            class="!rounded-[10px] !mb-1"
          >
            <template #icon>
              <component :is="child.icon" />
            </template>
            {{ child.name }}
          </a-menu-item>
        </a-sub-menu>
        <a-menu-item v-else :key="item.key" class="!rounded-[10px] !mb-1">
          <template #icon>
            <component :is="item.icon" />
          </template>
          {{ item.name }}
        </a-menu-item>
      </template>
    </a-menu>

    <div v-if="!collapsed" class="p-3 border-t border-black/[0.04]">
      <div
        class="flex items-center gap-3 px-3 py-2.5 rounded-[12px] hover:bg-black/[0.03] cursor-pointer transition-all duration-250 group"
        @click="router.push('/profile')"
      >
        <a-avatar
          :size="36"
          class="shrink-0"
          :image-url="userStore.userInfo?.avatar_url || undefined"
          :style="
            !userStore.userInfo?.avatar_url
              ? { background: 'linear-gradient(135deg, #30d158 0%, #007aff 100%)' }
              : {}
          "
        >
          {{
            (!userStore.userInfo?.avatar_url &&
              userStore.userInfo?.nickname?.charAt(0).toUpperCase()) ||
            'U'
          }}
        </a-avatar>
        <div class="flex-1 min-w-0">
          <p class="text-[14px] font-semibold text-[#1D1D1F] truncate leading-tight">
            {{ userStore.userInfo?.nickname || '用户' }}
          </p>
          <p class="text-[11px] text-[#86868B] truncate leading-tight mt-0.5">
            {{ userStore.userInfo?.username || '' }}
          </p>
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

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import type { Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { usePermissionStore } from '@/stores/permission'
import {
  IconHome,
  IconFile,
  IconSend,
  IconApps,
  IconExport,
  IconSettings,
  IconCode,
  IconSafe,
  IconStorage,
  IconTool,
  IconCheckCircle,
  IconUser,
  IconEdit,
} from '@arco-design/web-vue/es/icon'

const iconRegistry: Record<string, Component> = {
  home: IconHome,
  file: IconFile,
  send: IconSend,
  apps: IconApps,
  settings: IconSettings,
  code: IconCode,
  safe: IconSafe,
  storage: IconStorage,
  tool: IconTool,
  check: IconCheckCircle,
  user: IconUser,
  edit: IconEdit,
}

interface SidebarGroupConfig {
  name: string
  icon: string
  order: number
  wrapGroup: boolean
}

const SIDEBAR_GROUPS: Record<string, SidebarGroupConfig> = {
  top: { name: '', icon: '', order: 0, wrapGroup: false },
  content: { name: '内容管理', icon: 'file', order: 1, wrapGroup: true },
  review: { name: '审核管理', icon: 'check', order: 2, wrapGroup: true },
  platforms: { name: '平台管理', icon: 'apps', order: 3, wrapGroup: false },
  rbac: { name: '权限管理', icon: 'safe', order: 4, wrapGroup: true },
  system: { name: '系统管理', icon: 'tool', order: 5, wrapGroup: true },
}

interface MenuEntry {
  key: string
  name: string
  path: string
  icon?: Component
  permKey?: string
}

interface MenuGroup {
  key: string
  name: string
  icon?: Component
  children?: MenuEntry[]
}

type MenuItem = MenuEntry | MenuGroup

function isGroup(item: MenuItem): item is MenuGroup {
  return 'children' in item
}

const props = defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  toggle: []
  closeMobile: []
}>()

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const permStore = usePermissionStore()

function hasPerm(key: string, ctx?: any): boolean {
  return permStore.hasPermission(key, ctx)
}

const menuItems = computed<MenuItem[]>(() => {
  const sidebarRoutes = router.getRoutes().filter((r) => r.meta.sidebarType && r.meta.title)

  const byType = new Map<string, { route: (typeof sidebarRoutes)[number]; order: number }[]>()

  for (let i = 0; i < sidebarRoutes.length; i++) {
    const sideRoute = sidebarRoutes[i]
    const type = sideRoute.meta.sidebarType!
    if (!byType.has(type)) byType.set(type, [])
    byType.get(type)!.push({ route: sideRoute, order: sideRoute.meta.sidebarOrder ?? i })
  }

  for (const entries of byType.values()) {
    entries.sort((a, b) => a.order - b.order)
  }

  const groupOrder = Object.entries(SIDEBAR_GROUPS).sort((a, b) => a[1].order - b[1].order)

  const items: MenuItem[] = []

  for (const [type, config] of groupOrder) {
    const entries = byType.get(type)
    if (!entries || entries.length === 0) continue

    const visibleEntries: MenuEntry[] = []
    for (const entry of entries) {
      const permKey = entry.route.meta.permKey as string | undefined
      if (permKey && !hasPerm(permKey, entry.route.meta.ctx)) continue

      const icon = iconRegistry[entry.route.meta.icon ?? ''] ?? IconFile
      const resolved = router.resolve({ name: entry.route.name as string })
      visibleEntries.push({
        key: entry.route.name as string,
        name: entry.route.meta.title as string,
        path: resolved.path,
        icon,
        permKey: permKey,
      })
    }

    if (visibleEntries.length === 0) continue

    if (config.wrapGroup) {
      items.push({
        key: `${type}-group`,
        name: config.name,
        icon: iconRegistry[config.icon] ?? IconFile,
        children: visibleEntries,
      })
    } else {
      items.push(...visibleEntries)
    }
  }

  return items
})

const allEntries = computed<MenuEntry[]>(() => {
  const entries: MenuEntry[] = []
  menuItems.value.forEach((item) => {
    if (isGroup(item)) {
      entries.push(...item.children)
    } else {
      entries.push(item)
    }
  })
  return entries
})

const selectedKey = computed(() => {
  // 优先精确匹配，其次按路径长度降序匹配（最长优先）
  const sorted = [...allEntries.value].sort((a, b) => b.path.length - a.path.length)
  for (const item of sorted) {
    if (route.path === item.path) return item.key
  }
  for (const item of sorted) {
    if (item.path !== '/' && route.path.startsWith(item.path + '/')) return item.key
  }
  // 根路径特殊处理
  if (route.path === '/') {
    const root = allEntries.value.find((item) => item.path === '/')
    return root ? root.key : 'dashboard'
  }
  return 'dashboard'
})

const openKeys = ref<string[]>([])

function updateOpenKeys() {
  const open: string[] = []
  menuItems.value.forEach((item) => {
    if (isGroup(item)) {
      const matched = item.children.some((child) => {
        if (route.path === child.path) return true
        if (child.path !== '/' && route.path.startsWith(child.path + '/')) return true
        return false
      })
      if (matched) open.push(item.key)
    }
  })
  nextTick(() => {
    openKeys.value = open
  })
}

watch(() => route.path, updateOpenKeys, { immediate: true })

function onMenuItemClick(key: string) {
  const item = allEntries.value.find((m) => m.key === key)
  if (item) {
    router.push(item.path)
    emit('closeMobile')
  }
}
</script>

<style scoped lang="scss">
@media (max-width: 248px) {
  :deep(.arco-menu-item) {
    height: 32px !important;
    line-height: 32px !important;
    font-size: 12px !important;
    padding-left: 10px !important;
    padding-right: 10px !important;
  }
  :deep(.arco-menu-icon) {
    font-size: 14px !important;
  }
}
</style>
