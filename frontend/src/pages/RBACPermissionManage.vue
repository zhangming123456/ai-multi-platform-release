<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconFolder,
  IconFile,
  IconLock,
  IconSafe,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore, type RBACRole, type RBACResource, type RBACPermission } from '@/stores/permission'
import api from '@/utils/api'

const permStore = usePermissionStore()

const loading = ref(false)
const saving = ref(false)
const selectedRole = ref<RBACRole | null>(null)
const selectedResource = ref<RBACResource | null>(null)
const categoryFilter = ref<'all' | 'business' | 'admin' | 'database'>('all')

const DATABASE_RESOURCE_KEYS = new Set(['database', 'db', 'db_change', 'sql_review'])
const ADMIN_RESOURCE_KEYS = new Set([
  'accounts',
  'users',
  'roles',
  'permissions',
  'constraints',
  'sessions',
  'audit_logs',
  'permission_manage',
  'user_perm_manage',
])

function getResourceCategory(key: string): 'business' | 'admin' | 'database' {
  if (DATABASE_RESOURCE_KEYS.has(key)) return 'database'
  if (ADMIN_RESOURCE_KEYS.has(key)) return 'admin'
  return 'business'
}

const rolePermissions = ref<Record<string, { read: boolean; write: boolean; grant_type: 'direct' | 'inherited' }>>({})

const BUILTIN_COLORS: Record<string, string> = {
  super_admin: 'red',
  admin: 'orangered',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
  auditor: 'cyan',
}

function roleColor(role: RBACRole): string {
  return BUILTIN_COLORS[role.name] || 'arcoblue'
}

const sortedRoles = computed(() => {
  return [...permStore.roles].sort((a, b) => {
    if (a.is_super_admin) return -1
    if (b.is_super_admin) return 1
    if (a.is_system && !b.is_system) return -1
    if (!a.is_system && b.is_system) return 1
    return a.display_name.localeCompare(b.display_name)
  })
})

const resourceTree = computed(() => permStore.getResourceTree())

const filteredResourceTree = computed(() => {
  if (categoryFilter.value === 'all') return resourceTree.value
  return resourceTree.value.filter(
    r => getResourceCategory(r.key) === categoryFilter.value
  )
})

interface ResourceTreeNode {
  title: string
  key: string
  resource: RBACResource
  children?: ResourceTreeNode[]
}

const resourceTreeData = computed<ResourceTreeNode[]>(() => {
  function build(list: (RBACResource & { children?: RBACResource[] })[]): ResourceTreeNode[] {
    return list.map(r => ({
      title: r.name,
      key: r.id,
      resource: r,
      children: r.children && r.children.length > 0 ? build(r.children) : undefined,
    }))
  }
  return build(filteredResourceTree.value)
})

const selectedResourceKeys = computed(() =>
  selectedResource.value ? [selectedResource.value.id] : []
)

function handleTreeSelect(selectedKeys: (string | number)[]) {
  if (selectedKeys.length === 0) return
  const key = String(selectedKeys[0])
  function find(list: ResourceTreeNode[]): RBACResource | null {
    for (const node of list) {
      if (node.key === key) return node.resource
      if (node.children) {
        const found = find(node.children)
        if (found) return found
      }
    }
    return null
  }
  const resource = find(resourceTreeData.value)
  if (resource) selectedResource.value = resource
}

async function loadAllData() {
  loading.value = true
  try {
    await permStore.loadAllPermissionsData()
    if (sortedRoles.value.length > 0 && !selectedRole.value) {
      const editableRole = sortedRoles.value.find(r => !r.is_super_admin)
      if (editableRole) {
        await selectRole(editableRole)
      }
    }
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载权限数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadAllData()
})

watch(categoryFilter, () => {
  if (
    selectedResource.value &&
    categoryFilter.value !== 'all' &&
    getResourceCategory(selectedResource.value.key) !== categoryFilter.value
  ) {
    selectedResource.value = null
  }
})

async function selectRole(role: RBACRole) {
  selectedRole.value = role
  if (role.is_super_admin) {
    rolePermissions.value = {}
    return
  }
  const roleId = role.id
  try {
    const res = await api.get<{ role_id: string; permissions: Record<string, { read: boolean; write: boolean; grant_type: 'direct' | 'inherited' }> }>(
      `/v2/roles/${role.id}/permissions/detail`
    )
    if (selectedRole.value?.id !== roleId) return
    rolePermissions.value = res.data.permissions
  } catch (e: any) {
    if (selectedRole.value?.id === roleId) {
      Message.error(e.response?.data?.detail || '加载角色权限失败')
    }
  }
}

function isInheritedPerm(permKey: string): boolean {
  return rolePermissions.value[permKey]?.grant_type === 'inherited'
}

function handleRoleChange(roleId: unknown) {
  const role = sortedRoles.value.find(r => r.id === String(roleId))
  if (role) selectRole(role)
}

const currentResourcePermissions = computed(() => {
  if (!selectedResource.value) return []
  return permStore.getPermissionsByResource(selectedResource.value.id)
})

const pagePermissions = computed(() =>
  currentResourcePermissions.value.filter(p => p.operation === 'read')
)

const actionPermissions = computed(() =>
  currentResourcePermissions.value.filter(p => p.operation !== 'read')
)

function hasPermRead(permKey: string): boolean {
  return rolePermissions.value[permKey]?.read ?? false
}

function hasPermWrite(permKey: string): boolean {
  return rolePermissions.value[permKey]?.write ?? false
}

function togglePermRead(permKey: string) {
  if (!selectedRole.value || selectedRole.value.is_super_admin || isInheritedPerm(permKey)) return
  if (!rolePermissions.value[permKey]) {
    rolePermissions.value[permKey] = { read: false, write: false, grant_type: 'direct' }
  }
  rolePermissions.value[permKey].read = !rolePermissions.value[permKey].read
}

function togglePermWrite(permKey: string) {
  if (!selectedRole.value || selectedRole.value.is_super_admin || isInheritedPerm(permKey)) return
  if (!rolePermissions.value[permKey]) {
    rolePermissions.value[permKey] = { read: false, write: false, grant_type: 'direct' }
  }
  rolePermissions.value[permKey].write = !rolePermissions.value[permKey].write
}

function isGroupAllReadSelected(perms: RBACPermission[]): boolean {
  if (perms.length === 0) return false
  return perms.every(p => hasPermRead(p.key))
}

function isGroupAllWriteSelected(perms: RBACPermission[]): boolean {
  if (perms.length === 0) return false
  return perms.every(p => hasPermWrite(p.key))
}

function isGroupPartialReadSelected(perms: RBACPermission[]): boolean {
  const selectedCount = perms.filter(p => hasPermRead(p.key)).length
  return selectedCount > 0 && selectedCount < perms.length
}

function isGroupPartialWriteSelected(perms: RBACPermission[]): boolean {
  const selectedCount = perms.filter(p => hasPermWrite(p.key)).length
  return selectedCount > 0 && selectedCount < perms.length
}

function toggleGroupAllRead(perms: RBACPermission[]) {
  if (!selectedRole.value || selectedRole.value.is_super_admin) return
  const configurablePerms = perms.filter(p => !isInheritedPerm(p.key))
  if (configurablePerms.length === 0) return
  const allSelected = configurablePerms.every(p => hasPermRead(p.key))
  const newVal = !allSelected
  for (const p of configurablePerms) {
    if (!rolePermissions.value[p.key]) {
      rolePermissions.value[p.key] = { read: false, write: false, grant_type: 'direct' }
    }
    rolePermissions.value[p.key].read = newVal
  }
}

function toggleGroupAllWrite(perms: RBACPermission[]) {
  if (!selectedRole.value || selectedRole.value.is_super_admin) return
  const configurablePerms = perms.filter(p => !isInheritedPerm(p.key))
  if (configurablePerms.length === 0) return
  const allSelected = configurablePerms.every(p => hasPermWrite(p.key))
  const newVal = !allSelected
  for (const p of configurablePerms) {
    if (!rolePermissions.value[p.key]) {
      rolePermissions.value[p.key] = { read: false, write: false, grant_type: 'direct' }
    }
    rolePermissions.value[p.key].write = newVal
  }
}

async function savePermissions() {
  if (!selectedRole.value || selectedRole.value.is_super_admin) return
  saving.value = true
  try {
    const selectedPermIds: string[] = []
    for (const perm of permStore.permissions) {
      const access = rolePermissions.value[perm.key]
      if (access && access.grant_type === 'direct' && (access.read || access.write)) {
        selectedPermIds.push(perm.id)
      }
    }
    await api.put(`/v2/roles/${selectedRole.value.id}/permissions`, selectedPermIds)
    Message.success('权限保存成功')
    await selectRole(selectedRole.value)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '保存失败')
  } finally {
    saving.value = false
  }
}

function resourceTypeIcon(type: string) {
  return type === 'module' || type === 'page' ? IconFolder : IconFile
}

function resourceTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    module: '模块',
    page: '页面',
    api: '接口',
    data: '数据',
    action: '操作',
  }
  return labels[type] || type
}

function operationLabel(op: string): string {
  const labels: Record<string, string> = {
    create: '创建',
    read: '查看',
    update: '编辑',
    delete: '删除',
    execute: '执行',
    approve: '审核通过',
    reject: '审核驳回',
  }
  return labels[op] || op
}

function isWriteOperation(op: string): boolean {
  return ['create', 'update', 'delete', 'approve', 'reject', 'execute'].includes(op)
}
</script>

<template>
  <div>
    <PageHeader title="权限管理 (RBAC3)" subtitle="基于资源-操作模型的精细化权限管理，支持角色继承">
      <template #actions>
        <a-button
          v-if="selectedRole && !selectedRole.is_super_admin"
          type="primary"
          :loading="saving"
          @click="savePermissions"
        >
          保存权限
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" class="w-full">
      <div class="flex flex-col lg:flex-row gap-5">
        <div class="lg:w-[220px] shrink-0">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-3 mb-4">
            <p class="text-[12px] text-[#86868B] font-medium px-1 pb-2 pt-1">选择角色</p>
            <a-select
              :model-value="selectedRole?.id"
              placeholder="请选择角色"
              :style="{ width: '100%' }"
              @change="handleRoleChange"
            >
              <a-option
                v-for="role in sortedRoles"
                :key="role.id"
                :value="role.id"
                :label="role.display_name"
              >
                <div class="flex items-center gap-2">
                  <a-tag :color="roleColor(role)" size="small" class="!m-0">
                    {{ role.display_name }}
                  </a-tag>
                  <IconLock v-if="role.is_super_admin" :size="13" class="text-[#ff9500]" />
                </div>
              </a-option>
            </a-select>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-3">
            <a-tabs v-model:active-key="categoryFilter" type="rounded" size="small" class="rbac-resource-tabs">
              <a-tab-pane key="all" title="全部" />
              <a-tab-pane key="business" title="业务" />
              <a-tab-pane key="admin" title="管理" />
              <a-tab-pane key="database" title="数据库" />
            </a-tabs>
            <p class="text-[12px] text-[#86868B] font-medium px-3 pb-2 pt-1">资源树</p>
            <a-tree
              :data="resourceTreeData"
              :default-expand-all="true"
              :selected-keys="selectedResourceKeys"
              show-line
              block-node
              @select="handleTreeSelect"
            >
              <template #title="nodeData">
                <div class="flex items-center gap-2 min-w-0">
                  <component :is="resourceTypeIcon(nodeData.resource.type)" :size="14" class="shrink-0 text-[#86868B]" />
                  <span class="text-[13px] font-medium truncate">{{ nodeData.resource.name }}</span>
                  <a-tag
                    v-if="nodeData.resource.type !== 'module'"
                    size="small"
                    color="gray"
                    class="!m-0 ml-auto shrink-0"
                  >
                    {{ permStore.getPermissionsByResource(nodeData.key).length }}
                  </a-tag>
                </div>
              </template>
            </a-tree>
          </div>
        </div>

        <div class="flex-1 min-w-0">
          <div
            v-if="selectedRole?.is_super_admin"
            class="bg-[#ff9500]/[0.06] border border-[#ff9500]/[0.15] rounded-2xl p-5 mb-5 flex items-start gap-3"
          >
            <IconSafe :size="20" class="text-[#ff9500] mt-0.5 shrink-0" />
            <div>
              <p class="text-[14px] font-semibold text-[#1D1D1F] m-0">超级管理员拥有所有权限</p>
              <p class="text-[12px] text-[#86868B] m-0 mt-1">
                超级管理员默认拥有全部资源和操作权限，不可修改。
              </p>
            </div>
          </div>

          <div v-if="!selectedResource" class="h-full flex flex-col items-center justify-center text-center py-20">
            <div class="w-16 h-16 rounded-2xl bg-[#007aff]/[0.08] flex items-center justify-center mb-4">
              <IconFolder :size="28" class="text-[#007aff]" />
            </div>
            <p class="text-[16px] font-semibold text-[#1D1D1F] m-0">请选择资源模块</p>
            <p class="text-[13px] text-[#86868B] m-0 mt-1">从左侧资源树选择一个模块，查看并配置其页面权限与操作权限</p>
          </div>

          <div v-else class="space-y-5">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <component :is="resourceTypeIcon(selectedResource.type)" :size="18" class="text-[#007aff]" />
                <span class="text-[18px] font-semibold text-[#1D1D1F]">{{ selectedResource.name }}</span>
                <a-tag size="small" color="gray" class="!m-0">{{ resourceTypeLabel(selectedResource.type) }}</a-tag>
                <a-tag size="small" color="arcoblue" class="!m-0">
                  {{ currentResourcePermissions.length }} 项权限
                </a-tag>
              </div>
            </div>

            <div
              v-if="pagePermissions.length > 0"
              class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden"
            >
              <div class="flex items-center justify-between px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]">
                <div class="flex items-center gap-2">
                  <IconFile :size="16" class="text-[#007aff]" />
                  <span class="text-[14px] font-semibold text-[#1D1D1F]">页面权限</span>
                  <a-tag size="small" color="arcoblue" class="!m-0">{{ pagePermissions.length }} 项</a-tag>
                </div>
                <div v-if="selectedRole && !selectedRole.is_super_admin" class="flex items-center gap-4">
                  <a-checkbox
                    :model-value="isGroupAllReadSelected(pagePermissions)"
                    :indeterminate="isGroupPartialReadSelected(pagePermissions)"
                    @change="toggleGroupAllRead(pagePermissions)"
                  >
                    全选读
                  </a-checkbox>
                </div>
                <a-tag v-else color="green" size="small" class="!m-0">已拥有</a-tag>
              </div>
              <div class="p-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                <div
                  v-for="perm in pagePermissions"
                  :key="perm.id"
                  :class="[
                    'flex items-center justify-between px-3.5 py-3 rounded-xl border transition-all duration-200',
                    selectedRole?.is_super_admin
                      ? 'border-[#30d158]/20 bg-[#30d158]/[0.04] cursor-default'
                      : isInheritedPerm(perm.key)
                        ? 'border-[#34c759]/20 bg-[#34c759]/[0.04] cursor-default'
                        : (hasPermRead(perm.key) || hasPermWrite(perm.key))
                          ? 'border-[#007aff]/20 bg-[#007aff]/[0.04]'
                          : 'border-black/[0.06] bg-white',
                  ]"
                >
                  <div class="min-w-0 flex-1">
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 leading-tight">{{ perm.description || operationLabel(perm.operation) }}</p>
                    <p class="text-[11px] text-[#86868B] m-0 mt-0.5 font-mono">{{ perm.key }}</p>
                    <a-tag v-if="isInheritedPerm(perm.key)" size="small" color="green" class="!m-0 mt-1">继承自父角色</a-tag>
                  </div>
                  <div v-if="selectedRole?.is_super_admin" class="w-4 h-4 rounded-full bg-[#30d158] flex items-center justify-center shrink-0 ml-3">
                    <svg width="10" height="8" viewBox="0 0 10 8" fill="none"><path d="M1 4L3.5 6.5L9 1" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                  </div>
                  <div v-else class="flex flex-col items-end gap-2 shrink-0 ml-4">
                    <div
                      class="flex items-center gap-1.5"
                      :class="isInheritedPerm(perm.key) ? 'cursor-not-allowed' : 'cursor-pointer'"
                      @click.stop="togglePermRead(perm.key)"
                    >
                      <a-switch :model-value="hasPermRead(perm.key)" size="small" :disabled="isInheritedPerm(perm.key)" />
                      <span class="text-[12px] text-[#86868B] w-4 text-center">读</span>
                    </div>
                    <div
                      v-if="isWriteOperation(perm.operation)"
                      class="flex items-center gap-1.5"
                      :class="isInheritedPerm(perm.key) ? 'cursor-not-allowed' : 'cursor-pointer'"
                      @click.stop="togglePermWrite(perm.key)"
                    >
                      <a-switch :model-value="hasPermWrite(perm.key)" size="small" :disabled="isInheritedPerm(perm.key)" />
                      <span class="text-[12px] text-[#86868B] w-4 text-center">写</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div
              v-if="actionPermissions.length > 0"
              class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden"
            >
              <div class="flex items-center justify-between px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]">
                <div class="flex items-center gap-2">
                  <IconSafe :size="16" class="text-[#ff9500]" />
                  <span class="text-[14px] font-semibold text-[#1D1D1F]">操作权限</span>
                  <a-tag size="small" color="arcoblue" class="!m-0">{{ actionPermissions.length }} 项</a-tag>
                </div>
                <div v-if="selectedRole && !selectedRole.is_super_admin" class="flex items-center gap-4">
                  <a-checkbox
                    :model-value="isGroupAllReadSelected(actionPermissions)"
                    :indeterminate="isGroupPartialReadSelected(actionPermissions)"
                    @change="toggleGroupAllRead(actionPermissions)"
                  >
                    全选读
                  </a-checkbox>
                  <a-checkbox
                    :model-value="isGroupAllWriteSelected(actionPermissions)"
                    :indeterminate="isGroupPartialWriteSelected(actionPermissions)"
                    @change="toggleGroupAllWrite(actionPermissions)"
                  >
                    全选写
                  </a-checkbox>
                </div>
                <a-tag v-else color="green" size="small" class="!m-0">已拥有</a-tag>
              </div>
              <div class="p-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                <div
                  v-for="perm in actionPermissions"
                  :key="perm.id"
                  :class="[
                    'flex items-center justify-between px-3.5 py-3 rounded-xl border transition-all duration-200',
                    selectedRole?.is_super_admin
                      ? 'border-[#30d158]/20 bg-[#30d158]/[0.04] cursor-default'
                      : isInheritedPerm(perm.key)
                        ? 'border-[#34c759]/20 bg-[#34c759]/[0.04] cursor-default'
                        : (hasPermRead(perm.key) || hasPermWrite(perm.key))
                          ? 'border-[#007aff]/20 bg-[#007aff]/[0.04]'
                          : 'border-black/[0.06] bg-white',
                  ]"
                >
                  <div class="min-w-0 flex-1">
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 leading-tight">{{ perm.description || operationLabel(perm.operation) }}</p>
                    <p class="text-[11px] text-[#86868B] m-0 mt-0.5 font-mono">{{ perm.key }}</p>
                    <a-tag v-if="isInheritedPerm(perm.key)" size="small" color="green" class="!m-0 mt-1">继承自父角色</a-tag>
                  </div>
                  <div v-if="selectedRole?.is_super_admin" class="w-4 h-4 rounded-full bg-[#30d158] flex items-center justify-center shrink-0 ml-3">
                    <svg width="10" height="8" viewBox="0 0 10 8" fill="none"><path d="M1 4L3.5 6.5L9 1" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                  </div>
                  <div v-else class="flex flex-col items-end gap-2 shrink-0 ml-4">
                    <div
                      class="flex items-center gap-1.5"
                      :class="isInheritedPerm(perm.key) ? 'cursor-not-allowed' : 'cursor-pointer'"
                      @click.stop="togglePermRead(perm.key)"
                    >
                      <a-switch :model-value="hasPermRead(perm.key)" size="small" :disabled="isInheritedPerm(perm.key)" />
                      <span class="text-[12px] text-[#86868B] w-4 text-center">读</span>
                    </div>
                    <div
                      v-if="isWriteOperation(perm.operation)"
                      class="flex items-center gap-1.5"
                      :class="isInheritedPerm(perm.key) ? 'cursor-not-allowed' : 'cursor-pointer'"
                      @click.stop="togglePermWrite(perm.key)"
                    >
                      <a-switch :model-value="hasPermWrite(perm.key)" size="small" :disabled="isInheritedPerm(perm.key)" />
                      <span class="text-[12px] text-[#86868B] w-4 text-center">写</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </a-spin>
  </div>
</template>
