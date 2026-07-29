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
} from '@arco-design/web-vue/es/icon'

interface MenuEntry {
  key: string
  name: string
  path: string
  icon: Component
  permKey: string
}

interface MenuGroup {
  key: string
  name: string
  icon: Component
  children: MenuEntry[]
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

function hasPerm(key: string): boolean {
  return permStore.hasPermission(key, 'read')
}

const contentChildren = computed<MenuEntry[]>(() => {
  const items: MenuEntry[] = []
  if (hasPerm('content:read')) {
    items.push({ key: 'content', name: '内容列表', path: '/content', icon: IconFile, permKey: 'content:read' })
  }
  if (hasPerm('content:create')) {
    items.push({ key: 'content-create', name: '创作内容', path: '/content/create', icon: IconFile, permKey: 'content:create' })
  }
  if (hasPerm('publish:read')) {
    items.push({ key: 'publish', name: '发布管理', path: '/publish', icon: IconSend, permKey: 'publish:read' })
  }
  if (hasPerm('templates:read')) {
    items.push({ key: 'templates', name: '模板管理', path: '/templates', icon: IconApps, permKey: 'templates:read' })
  }
  return items
})

const reviewChildren = computed<MenuEntry[]>(() => {
  const items: MenuEntry[] = []
  if (hasPerm('review:read')) {
    items.push({ key: 'review', name: '内容审核', path: '/review', icon: IconCheckCircle, permKey: 'review:read' })
  }
  if (hasPerm('sql_review:read')) {
    items.push({ key: 'sql-review', name: 'SQL审核', path: '/sql-review', icon: IconStorage, permKey: 'sql_review:read' })
  }
  if (hasPerm('review:read')) {
    items.push({ key: 'user-creation-review', name: '用户注册审核', path: '/settings/user-creation-review', icon: IconUser, permKey: 'review:read' })
  }
  return items
})

const rbacChildren = computed<MenuEntry[]>(() => {
  const items: MenuEntry[] = []
  if (hasPerm('users:read')) {
    items.push({ key: 'rbac-users', name: '用户管理', path: '/rbac/users', icon: IconSafe, permKey: 'users:read' })
  }
  if (hasPerm('roles:read')) {
    items.push({ key: 'rbac-roles', name: '角色管理', path: '/rbac/roles', icon: IconUser, permKey: 'roles:read' })
  }
  if (hasPerm('permissions:read')) {
    items.push({ key: 'rbac-permissions', name: '权限管理', path: '/rbac/permissions', icon: IconSafe, permKey: 'permissions:read' })
  }
  if (hasPerm('constraints:read')) {
    items.push({ key: 'rbac-constraints', name: '约束管理', path: '/rbac/constraints', icon: IconSafe, permKey: 'constraints:read' })
  }
  return items
})

const sysChildren = computed<MenuEntry[]>(() => {
  const items: MenuEntry[] = []
  if (hasPerm('token_plan:read')) {
    items.push({ key: 'token-plan', name: 'Token方案', path: '/settings/token-plan', icon: IconSettings, permKey: 'token_plan:read' })
  }
  if (hasPerm('api_docs:read')) {
    items.push({ key: 'api-docs', name: 'API文档', path: '/developer/docs', icon: IconCode, permKey: 'api_docs:read' })
  }
  if (hasPerm('db:read')) {
    items.push({ key: 'database', name: '数据库控制台', path: '/developer/database', icon: IconStorage, permKey: 'db:read' })
  }
  return items
})

const menuItems = computed<MenuItem[]>(() => {
  const items: MenuItem[] = []

  if (hasPerm('dashboard:read')) {
    items.push({ key: 'dashboard', name: '仪表盘', path: '/', icon: IconHome, permKey: 'dashboard:read' })
  }

  if (contentChildren.value.length > 0) {
    items.push({
      key: 'content-group',
      name: '内容管理',
      icon: IconFile,
      children: contentChildren.value,
    })
  }

  if (reviewChildren.value.length > 0) {
    items.push({
      key: 'review-group',
      name: '审核管理',
      icon: IconCheckCircle,
      children: reviewChildren.value,
    })
  }

  if (hasPerm('platforms:read')) {
    items.push({ key: 'platforms', name: '平台管理', path: '/platforms', icon: IconApps, permKey: 'platforms:read' })
  }

  if (rbacChildren.value.length > 0) {
    items.push({
      key: 'rbac-group',
      name: '权限管理',
      icon: IconSafe,
      children: rbacChildren.value,
    })
  }

  if (sysChildren.value.length > 0) {
    items.push({
      key: 'settings-group',
      name: '系统管理',
      icon: IconTool,
      children: sysChildren.value,
    })
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
  const match = allEntries.value.find((item) => {
    if (item.path === '/') return route.path === '/'
    return route.path.startsWith(item.path)
  })
  return match ? match.key : 'dashboard'
})

const openKeys = ref<string[]>([])

function updateOpenKeys() {
  const open: string[] = []
  menuItems.value.forEach((item) => {
    if (isGroup(item)) {
      const matched = item.children.some((child) => {
        if (child.path === '/') return route.path === '/'
        return route.path.startsWith(child.path)
      })
      if (matched) open.push(item.key)
    }
  })
  // Defer openKeys update to the next tick to avoid Arco Menu DOM mutations
  // colliding with the current route transition.
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
          :style="!userStore.userInfo?.avatar_url ? { background: 'linear-gradient(135deg, #30d158 0%, #007aff 100%)' } : {}"
        >
          {{ (!userStore.userInfo?.avatar_url && userStore.userInfo?.nickname?.charAt(0).toUpperCase()) || 'U' }}
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

<style scoped>
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
