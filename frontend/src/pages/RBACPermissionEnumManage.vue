<template>
  <div class="page-main">
    <PageHeader
      title="权限字典管理"
      subtitle="编辑权限字典的名称、Key 和描述，或自定义添加新的权限字典。仅超级管理员可操作。"
    >
      <template #actions>
        <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="goCreate">
          <template #icon><IconPlus :size="13" /></template>
          新增权限字典
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <div class="space-y-4">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-2">
                <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">页面权限</h3>
                <span class="text-[12px] text-[#86868B] font-medium">{{
                  pagePermissions.length
                }}</span>
              </div>
            </div>

            <div v-if="pagePermissions.length === 0" class="py-8">
              <a-empty description="暂无页面权限字典" />
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div
                v-for="item in pagePermissions"
                :key="item.id"
                class="flex items-start justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
                :class="{ 'opacity-50': !item.is_active }"
              >
                <div class="min-w-0 mr-3 flex-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">
                      {{ item.resource.name }}
                    </p>
                    <a-tag size="small" color="blue" class="!m-0 shrink-0 font-mono">
                      {{ item.key }}
                    </a-tag>
                    <a-tag v-if="!item.is_active" size="small" color="gray" class="!m-0 shrink-0">
                      已禁用
                    </a-tag>
                  </div>
                  <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                    &#123;{{ _parseKey(item.resource.key).keyName }}&#125;:read
                  </p>
                </div>
                <div class="flex items-center gap-1 shrink-0">
                  <a-tooltip content="编辑">
                    <a-button size="mini" type="text" @click="goEdit(item)">
                      <template #icon><IconEdit :size="13" /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip :content="item.is_active ? '禁用' : '启用'">
                    <a-button
                      size="mini"
                      type="text"
                      :status="item.is_active ? 'normal' : 'warning'"
                      @click="toggleActive(item.id, item.is_active)"
                    >
                      <template #icon>
                        <component :is="item.is_active ? IconEye : IconEyeInvisible" :size="13" />
                      </template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="删除">
                    <a-button
                      size="mini"
                      type="text"
                      status="danger"
                      @click="deletePermission(item)"
                    >
                      <template #icon><IconDelete :size="13" /></template>
                    </a-button>
                  </a-tooltip>
                </div>
              </div>
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-2">
                <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">操作权限</h3>
                <span class="text-[12px] text-[#86868B] font-medium">{{
                  actionPermissions.length
                }}</span>
              </div>
            </div>

            <div v-if="actionPermissions.length === 0" class="py-8">
              <a-empty description="暂无操作权限字典" />
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div
                v-for="item in actionPermissions"
                :key="item.id"
                class="flex items-start justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
                :class="{ 'opacity-50': !item.is_active }"
              >
                <div class="min-w-0 mr-3 flex-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">
                      {{ item.resource.name }}
                    </p>
                    <a-tag size="small" color="arcoblue" class="!m-0 shrink-0 font-mono">
                      {{ item.key }}
                    </a-tag>
                    <a-tag v-if="!item.is_active" size="small" color="gray" class="!m-0 shrink-0">
                      已禁用
                    </a-tag>
                  </div>
                  <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                    &#123;{{ _parseKey(item.resource.key).keyName }}&#125;:&#123;{{
                      _parseKey(item.resource.key).operation
                    }}&#125;:{{ _parseKey(item.resource.key).mode }}
                  </p>
                </div>
                <div class="flex items-center gap-1 shrink-0">
                  <a-tooltip content="编辑">
                    <a-button size="mini" type="text" @click="goEdit(item)">
                      <template #icon><IconEdit :size="13" /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip :content="item.is_active ? '禁用' : '启用'">
                    <a-button
                      size="mini"
                      type="text"
                      :status="item.is_active ? 'normal' : 'warning'"
                      @click="toggleActive(item.id, item.is_active)"
                    >
                      <template #icon>
                        <component :is="item.is_active ? IconEye : IconEyeInvisible" :size="13" />
                      </template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="删除">
                    <a-button
                      size="mini"
                      type="text"
                      status="danger"
                      @click="deletePermission(item)"
                    >
                      <template #icon><IconDelete :size="13" /></template>
                    </a-button>
                  </a-tooltip>
                </div>
              </div>
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center gap-2 mb-4">
              <IconSafe :size="18" class="text-[#86868B]" />
              <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">权限 Key 格式说明</h3>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div class="p-3 rounded-xl bg-[#007AFF]/[0.04] border border-[#007AFF]/[0.1]">
                <p class="text-[12px] font-semibold text-[#1D1D1F] m-0 mb-1">页面权限 (page)</p>
                <code class="text-[12px] text-[#007AFF]">&#123;name&#125;:read</code>
                <p class="text-[11px] text-[#86868B] m-0 mt-1">
                  2段格式，name 不能含 read/write。结尾固定为 read。根据 key 格式自动推断类型。
                </p>
              </div>
              <div class="p-3 rounded-xl bg-[#34C759]/[0.04] border border-[#34C759]/[0.1]">
                <p class="text-[12px] font-semibold text-[#1D1D1F] m-0 mb-1">操作权限 (action)</p>
                <code class="text-[12px] text-[#34C759]"
                  >&#123;name&#125;:&#123;operation&#125;:read | write</code
                >
                <p class="text-[11px] text-[#86868B] m-0 mt-1">
                  3段格式，name 和 operation 不能含 read/write。结尾为 read 或 write。根据 key
                  格式自动推断类型。
                </p>
              </div>
            </div>
          </div>
        </div>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconSafe,
  IconEdit,
  IconDelete,
  IconPlus,
  IconEye,
  IconEyeInvisible,
} from '@arco-design/web-vue/es/icon'
import { orderBy } from 'lodash-es'
import PageHeader from '@/components/layout/PageHeader.vue'
import api, { getApiErrorDetail } from '@/utils/api'

interface ResourceRef {
  id: string
  key: string
  name: string
  description?: string | null
}

interface PermissionItem {
  id: string
  key: string
  operation: string
  is_active: boolean
  resource: ResourceRef
}

type PermMode = 'read' | 'write'

function _isPageKey(key: string): boolean {
  const parts = key.split(':')
  return parts.length === 2 && (parts[1] === 'read' || parts[1] === 'write')
}

function _parseKey(key: string): { keyName: string; operation: string; mode: PermMode } {
  const parts = key.split(':')
  if (_isPageKey(key)) {
    return { keyName: parts[0], operation: parts[1], mode: parts[1] as PermMode }
  }
  return { keyName: parts[0], operation: parts[1] || '', mode: (parts[2] || 'write') as PermMode }
}

const router = useRouter()

const loading = ref(false)
const saving = ref(false)
const permissions = ref<PermissionItem[]>([])

const pagePermissions = computed(() =>
  orderBy(
    permissions.value.filter((p) => _isPageKey(p.key)),
    ['key'],
    ['asc'],
  ),
)

const actionPermissions = computed(() =>
  orderBy(
    permissions.value.filter((p) => !_isPageKey(p.key)),
    ['key'],
    ['asc'],
  ),
)

async function fetchPermissions() {
  loading.value = true
  try {
    const res = await api.get<PermissionItem[]>('/v2/permissions')
    permissions.value = Array.isArray(res.data) ? res.data : []
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载权限列表失败')
  } finally {
    loading.value = false
  }
}

function goEdit(item: PermissionItem) {
  router.push({ name: 'RBACPermissionEnumEdit', params: { resourceId: item.resource.id } })
}

function goCreate() {
  router.push({ name: 'RBACPermissionEnumCreate' })
}

async function toggleActive(permissionId: string, currentActive: boolean) {
  saving.value = true
  try {
    await api.put(`/v2/permissions/${permissionId}`, { is_active: !currentActive })
    Message.success(currentActive ? '权限已禁用' : '权限已启用')
    await fetchPermissions()
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '操作失败')
  } finally {
    saving.value = false
  }
}

async function deletePermission(item: PermissionItem) {
  Modal.warning({
    title: '确认删除',
    content: `确定要删除权限 "${item.resource.name}" (${item.key}) 吗？此操作不可恢复。`,
    hideCancel: false,
    onOk: async () => {
      saving.value = true
      try {
        await api.delete(`/v2/resources/${item.resource.id}`)
        Message.success('权限已删除')
        await fetchPermissions()
      } catch (e) {
        Message.error(getApiErrorDetail(e) || '删除失败')
      } finally {
        saving.value = false
      }
    },
  })
}

onMounted(fetchPermissions)
</script>
