<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconSafe, IconLock } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import ResourcePermissionCard from '@/components/rbac/ResourcePermissionCard.vue'
import api from '@/utils/api'

interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface Resource {
  id: string
  key: string
  name: string
  description: string | null
  type: 'page' | 'action'
  parent_id: string | null
  is_active: boolean
  children: Resource[]
}

interface Permission {
  id: string
  key: string
  operation: string
  is_active: boolean
  resource: {
    id: string
    key: string
    name: string
    type: string
  }
}

interface RolePermission {
  id: string
  key: string
  operation: string
  resource_id: string
  resource_key: string
  resource_name: string
  grant_type: 'direct' | 'inherited'
}

type PermissionCategory = 'page' | 'action' | 'both'

const CATEGORY_LABELS: Record<PermissionCategory, string> = {
  page: '页面权限',
  action: '操作权限',
  both: '既有操作权限也有页面权限',
}

const loading = ref(false)
const saving = ref(false)
const roles = ref<Role[]>([])
const resources = ref<Resource[]>([])
const permissions = ref<Permission[]>([])
const rolePermissions = ref<RolePermission[]>([])
const selectedRoleId = ref<string | null>(null)
const activeCategory = ref<PermissionCategory>('page')

const BUILTIN_COLORS: Record<string, string> = {
  admin: 'red',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
}

const CUSTOM_COLORS = ['arcoblue', 'purple', 'cyan', 'orange', 'pink', 'gold', 'lime', 'magenta']

const roleColorMap = computed(() => {
  const map: Record<string, string> = {}
  let customIdx = 0
  for (const r of roles.value) {
    if (BUILTIN_COLORS[r.name]) {
      map[r.name] = BUILTIN_COLORS[r.name]
    } else {
      map[r.name] = CUSTOM_COLORS[customIdx % CUSTOM_COLORS.length]
      customIdx++
    }
  }
  return map
})

function roleColor(role: Role): string {
  return roleColorMap.value[role.name] || 'arcoblue'
}

const sortedRoles = computed(() => {
  return [...roles.value].sort((a, b) => {
    if (a.is_super_admin) return -1
    if (b.is_super_admin) return 1
    const builtinOrder = ['manager', 'operator', 'reviewer']
    const aIdx = builtinOrder.indexOf(a.name)
    const bIdx = builtinOrder.indexOf(b.name)
    if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx
    if (aIdx !== -1) return -1
    if (bIdx !== -1) return 1
    return 0
  })
})

const selectedRole = computed(() => {
  return roles.value.find((r) => r.id === selectedRoleId.value) || null
})

function resourceCategory(resource: Resource): PermissionCategory {
  if (resource.type === 'action') return 'action'
  const ops = permissions.value
    .filter((p) => p.resource.id === resource.id && p.is_active)
    .map((p) => p.operation)
  const hasRead = ops.includes('read')
  const hasOther = ops.some((o) => o !== 'read')
  if (hasRead && hasOther) return 'both'
  return 'page'
}

const categoryCounts = computed(() => {
  const counts: Record<PermissionCategory, number> = { page: 0, action: 0, both: 0 }
  for (const resource of resources.value) {
    counts[resourceCategory(resource)]++
  }
  return counts
})

const filteredResources = computed(() => {
  return resources.value.filter((r) => resourceCategory(r) === activeCategory.value)
})

const permissionMap = computed(() => {
  const map = new Map<string, Permission>()
  for (const p of permissions.value) {
    map.set(p.id, p)
  }
  return map
})

async function fetchRoles() {
  const res = await api.get<Role[]>('/v2/roles')
  roles.value = Array.isArray(res.data) ? res.data : []
  if (sortedRoles.value.length > 0 && !selectedRoleId.value) {
    const firstEditable = sortedRoles.value.find((r) => !r.is_super_admin)
    selectedRoleId.value = firstEditable?.id || sortedRoles.value[0].id
  }
}

async function fetchResources() {
  const res = await api.get<Resource[]>('/v2/resources')
  resources.value = Array.isArray(res.data) ? res.data : []
}

async function fetchPermissions() {
  const res = await api.get<Permission[]>('/v2/permissions')
  permissions.value = Array.isArray(res.data) ? res.data : []
}

async function fetchRolePermissions(roleId: string) {
  const res = await api.get<RolePermission[]>(`/v2/roles/${roleId}/permissions/detail`)
  rolePermissions.value = Array.isArray(res.data) ? res.data : []
}

async function loadAll() {
  loading.value = true
  try {
    await Promise.all([fetchRoles(), fetchResources(), fetchPermissions()])
    if (selectedRoleId.value) {
      await fetchRolePermissions(selectedRoleId.value)
    }
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载权限数据失败')
  } finally {
    loading.value = false
  }
}

async function onRoleChange(roleId: unknown) {
  const id = String(roleId)
  selectedRoleId.value = id
  rolePermissions.value = []
  if (!id) return
  loading.value = true
  try {
    await fetchRolePermissions(id)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载角色权限失败')
  } finally {
    loading.value = false
  }
}

function selectedDirectPermissionIds(): string[] {
  const ids: string[] = []
  for (const rp of rolePermissions.value) {
    if (rp.grant_type === 'direct') {
      const perm = permissionMap.value.get(
        permissions.value.find((p) => p.key === rp.key)?.id || '',
      )
      if (perm) ids.push(perm.id)
    }
  }
  return ids
}

function togglePermission(key: string, _mode: 'read' | 'write') {
  if (selectedRole.value?.is_super_admin) return
  const existing = rolePermissions.value.find((rp) => rp.key === key)
  if (existing?.grant_type === 'inherited') return

  const perm = permissionMap.value.get(
    permissions.value.find((p) => p.key === key)?.id || '',
  )
  if (!perm) return

  const currentlyHas = Boolean(existing)
  let updated = rolePermissions.value.filter((rp) => rp.key !== key)

  if (!currentlyHas) {
    updated.push({
      id: `direct-${key}`,
      key,
      operation: perm.operation,
      resource_id: perm.resource.id,
      resource_key: perm.resource.key,
      resource_name: perm.resource.name,
      grant_type: 'direct',
    })
  }

  rolePermissions.value = updated
}

function toggleResourceRead(resource: Resource) {
  const readPerm = permissions.value.find(
    (p) => p.resource.id === resource.id && p.operation === 'read',
  )
  if (readPerm) togglePermission(readPerm.key, 'read')
}

function toggleResourceWrite(resource: Resource) {
  const writePerm = permissions.value.find(
    (p) => p.resource.id === resource.id && p.operation === 'write',
  )
  if (writePerm) togglePermission(writePerm.key, 'write')
}

async function savePermissions() {
  if (!selectedRoleId.value || selectedRole.value?.is_super_admin) return
  saving.value = true
  try {
    const ids = selectedDirectPermissionIds()
    await api.put(`/v2/roles/${selectedRoleId.value}/permissions`, {
      permission_ids: ids,
    })
    Message.success(`${selectedRole.value?.display_name || '角色'} 权限保存成功`)
    await fetchRolePermissions(selectedRoleId.value)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(loadAll)
</script>

<template>
  <div class="page-main">
    <PageHeader title="权限设置" subtitle="按资源树为角色配置访问与操作权限，子角色继承的权限不可取消">
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

    <a-spin :loading="loading" tip="加载中..." class="w-full">
      <div class="flex flex-col lg:flex-row gap-5">
        <div class="lg:w-[260px] shrink-0">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-4">
            <p class="text-[12px] text-[#86868B] font-medium px-1 pb-3">选择角色</p>
            <a-select
              :model-value="selectedRoleId || undefined"
              placeholder="选择角色"
              @change="onRoleChange"
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
                  <IconLock v-if="role.is_super_admin" :size="12" class="text-[#ff9500]" />
                </div>
              </a-option>
            </a-select>

            <div
              v-if="selectedRole?.is_super_admin"
              class="mt-4 p-3 rounded-xl bg-[#ff9500]/[0.06] border border-[#ff9500]/[0.15] flex items-start gap-2.5"
            >
              <IconSafe :size="16" class="text-[#ff9500] mt-0.5 shrink-0" />
              <p class="text-[12px] text-[#1D1D1F] m-0 leading-relaxed">
                超级管理员拥有所有权限，不可修改。
              </p>
            </div>

            <div
              v-else-if="selectedRole"
              class="mt-4 p-3 rounded-xl bg-black/[0.02] border border-black/[0.04]"
            >
              <p class="text-[12px] text-[#86868B] m-0 leading-relaxed">
                {{ selectedRole.description || '暂无描述' }}
              </p>
            </div>
          </div>
        </div>

        <div class="flex-1 min-w-0">
          <div v-if="!selectedRole" class="empty-state">
            <a-empty description="请选择角色" />
          </div>

          <div v-else class="space-y-4">
            <a-tabs v-model:active-key="activeCategory" type="rounded" size="medium">
              <a-tab-pane
                v-for="key in (['page', 'action', 'both'] as PermissionCategory[])"
                :key="key"
                :title="`${CATEGORY_LABELS[key]} (${categoryCounts[key]})`"
              >
                <div class="space-y-4 pt-2">
                  <ResourcePermissionCard
                    v-for="resource in filteredResources"
                    :key="resource.id"
                    :resource="resource"
                    :depth="0"
                    :selected-role="selectedRole"
                    :role-permission-map="new Map(rolePermissions.map((rp) => [rp.key, rp]))"
                    :permission-map="permissionMap"
                    :permissions="permissions"
                    @toggle-permission="togglePermission"
                    @toggle-resource-read="toggleResourceRead"
                    @toggle-resource-write="toggleResourceWrite"
                  />
                  <a-empty
                    v-if="filteredResources.length === 0"
                    description="该分类下暂无资源"
                  />
                </div>
              </a-tab-pane>
            </a-tabs>
          </div>
        </div>
      </div>
    </a-spin>
  </div>
</template>
