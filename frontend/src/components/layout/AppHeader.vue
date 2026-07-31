<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { IconMenuFold, IconMenuUnfold, IconMore, IconLanguage } from '@arco-design/web-vue/es/icon'
import { useRegionStore } from '@/stores/region'
import NotificationBell from '@/components/NotificationBell.vue'

const props = defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  toggleSidebar: []
}>()

const route = useRoute()
const { selectedTz, regions, switchRegion } = useRegionStore()

const breadcrumbMap: Record<string, string> = {
  '/': '仪表盘',
  '/profile': '个人资料',
  '/platforms': '平台管理',
  '/content': '内容工坊',
  '/content/create': '创建内容',
  '/publish': '发布管理',
  '/review': '审核管理',
  '/templates': '模板中心',
  '/settings/token-plan': 'Token 配置',
  '/developer/docs': 'API 文档',
  '/rbac/users': '用户设置',
  '/rbac/roles': '角色设置',
  '/rbac/permissions': '权限设置',
  '/rbac/permissions/enum': '权限字典管理',
  '/rbac/constraints': '职责分离',
}

const breadcrumbs = computed(() => {
  const path = route.path
  const items: { label: string; path: string }[] = [{ label: 'Matrix', path: '/' }]
  if (path !== '/') {
    items.push({ label: breadcrumbMap[path] || '页面', path })
  }
  return items
})

const mobileDrawerVisible = ref(false)

function regionShortLabel(value: string): string {
  const r = regions.find((r) => r.value === value)
  if (!r) return 'CN'
  const label = r.label
  if (label.includes('大陆')) return 'CN'
  if (label.includes('台湾')) return 'TW'
  if (label.includes('香港')) return 'HK'
  if (label.includes('澳门')) return 'MO'
  if (label.includes('日本')) return 'JP'
  if (label.includes('韩国')) return 'KR'
  if (label.includes('新加坡')) return 'SG'
  if (label.includes('美国东部')) return 'US-E'
  if (label.includes('美国西部')) return 'US-W'
  if (label.includes('英国')) return 'UK'
  if (label.includes('法国')) return 'FR'
  return label.slice(0, 2)
}
</script>

<template>
  <header
    class="h-[56px] bg-white/60 backdrop-blur-xl border-b border-black/[0.04] flex items-center justify-between px-4 md:px-8 sticky top-0 z-20 whitespace-nowrap shrink-0"
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

    <!-- 右侧区域：全部向右排列 -->
    <div class="flex items-center gap-2">
      <!-- 区域：icon + 简称 -->
      <a-popover
        trigger="click"
        position="br"
        :content-style="{ padding: '4px', minWidth: '180px' }"
      >
        <a-button type="text" size="small" class="!rounded-[10px]">
          <template #icon>
            <IconLanguage :size="18" />
          </template>
          <span class="text-[12px] font-medium text-[#636366] ml-0.5">{{
            regionShortLabel(selectedTz)
          }}</span>
        </a-button>
        <template #content>
          <div class="region-list">
            <div
              v-for="region in regions"
              :key="region.value"
              :class="['region-list-item', { active: region.value === selectedTz }]"
              @click="switchRegion(region.value)"
            >
              <span class="region-list-item__label">{{ region.label }}</span>
              <span class="region-list-item__offset">{{ region.offset }}</span>
            </div>
          </div>
        </template>
      </a-popover>

      <!-- 搜索框 -->
      <a-input-search
        placeholder="搜索..."
        class="hidden md:block"
        :style="{ width: '200px' }"
        allow-clear
      />

      <!-- 通知 -->
      <NotificationBell />

      <!-- 更多工具（移动端） -->
      <a-button
        type="text"
        size="small"
        class="!rounded-[10px] md:hidden"
        @click="mobileDrawerVisible = true"
      >
        <template #icon>
          <IconMore :size="20" />
        </template>
      </a-button>
    </div>

    <!-- 移动端 Drawer -->
    <a-drawer
      v-model:visible="mobileDrawerVisible"
      :width="300"
      placement="right"
      :footer="false"
      title="工具"
    >
      <div class="flex flex-col gap-4">
        <!-- 搜索框 -->
        <a-input-search placeholder="搜索..." allow-clear />

        <!-- 区域切换 -->
        <div>
          <p class="text-[12px] text-[#86868B] mb-2">区域</p>
          <a-select
            :model-value="selectedTz"
            size="small"
            :bordered="false"
            class="w-full"
            @change="
              (value: string | number | boolean | Record<string, any> | undefined) => {
                if (typeof value === 'string') switchRegion(value)
              }
            "
          >
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
        </div>
      </div>
    </a-drawer>
  </header>
</template>

<style scoped>
.region-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.region-list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s ease;
}

.region-list-item:hover {
  background: rgba(0, 0, 0, 0.04);
}

.region-list-item.active {
  background: rgba(0, 122, 255, 0.08);
}

.region-list-item__label {
  font-size: 13px;
  font-weight: 500;
  color: #1d1d1f;
}

.region-list-item__offset {
  font-size: 11px;
  color: #86868b;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 248px) {
  header {
    height: 44px !important;
    padding-left: 10px !important;
    padding-right: 10px !important;
  }
}
</style>
