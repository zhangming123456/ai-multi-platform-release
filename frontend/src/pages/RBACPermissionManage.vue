<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconSafe, IconLock } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PermissionModuleCard from '@/components/rbac/PermissionModuleCard.vue'
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

interface ModuleItemDef {
  id: string
  title: string
  subtitle: string
  readKey?: string
  writeKeys: string[]
}

interface ModuleDef {
  key: string
  label: string
  items: ModuleItemDef[]
}

const loading = ref(false)
const saving = ref(false)
const roles = ref<Role[]>([])
const resources = ref<Resource[]>([])
const permissions = ref<Permission[]>([])
const rolePermissions = ref<RolePermission[]>([])
const selectedRoleId = ref<string | null>(null)

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

const permissionMap = computed(() => {
  const map = new Map<string, Permission>()
  for (const p of permissions.value) {
    map.set(p.id, p)
  }
  return map
})

const resourceByKey = computed(() => {
  const map = new Map<string, Resource>()
  for (const r of resources.value) {
    map.set(r.key, r)
  }
  return map
})

const permissionByKey = computed(() => {
  const map = new Map<string, Permission>()
  for (const p of permissions.value) {
    map.set(p.key, p)
  }
  return map
})

const PAGE_TITLE_OVERRIDES: Record<string, string> = {
  users: '账号设置',
}

function pageItem(resource: Resource): ModuleItemDef {
  const perms = permissions.value.filter((p) => p.resource.id === resource.id && p.is_active)
  const readPerm = perms.find((p) => p.operation === 'read')
  const writePerm = perms.find((p) => p.operation === 'write')
  return {
    id: resource.id,
    title: PAGE_TITLE_OVERRIDES[resource.key] || resource.name,
    subtitle: '页面访问',
    readKey: readPerm?.key,
    writeKeys: writePerm ? [writePerm.key] : [],
  }
}

function actionItem(resourceKey: string, operation: string, title: string): ModuleItemDef | null {
  const perm = permissions.value.find(
    (p) => p.resource.key === resourceKey && p.operation === operation && p.is_active,
  )
  if (!perm) return null
  return {
    id: perm.id,
    title,
    subtitle: '操作权限',
    readKey: undefined,
    writeKeys: [perm.key],
  }
}

const modules = computed<ModuleDef[]>(() => {
  const pageKeys = [
    'dashboard',
    'platforms',
    'content',
    'publish',
    'templates',
    'review',
    'sql_review',
    'accounts',
    'token_plan',
    'api_docs',
    'db',
    'permissions',
    'roles',
    'constraints',
    'users',
  ]
  const pageItems: ModuleItemDef[] = []
  for (const key of pageKeys) {
    const resource = resourceByKey.value.get(key)
    if (resource) pageItems.push(pageItem(resource))
  }

  const systemItems: ModuleItemDef[] = []
  const systemActions: Array<[string, string, string]> = [
    ['users', 'read', '查看用户'],
    ['users', 'write', '管理用户'],
    ['users', 'create', '创建用户'],
    ['users', 'update', '编辑用户'],
    ['users', 'delete', '删除用户'],
    ['users', 'change_password', '修改用户密码'],
    ['model_config', 'create', '创建模型配置'],
    ['model_config', 'update', '编辑模型配置'],
    ['model_config', 'delete', '删除模型配置'],
    ['db', 'execute', '执行SQL命令'],
    ['db_history', 'read', '查看SQL历史'],
  ]
  for (const [rk, op, title] of systemActions) {
    const item = actionItem(rk, op, title)
    if (item) systemItems.push(item)
  }

  const contentItems: ModuleItemDef[] = []
  const contentActions: Array<[string, string, string]> = [
    ['content', 'create', '创建内容'],
    ['content', 'update', '编辑内容'],
    ['content', 'delete', '删除内容'],
    ['content', 'ai_generate', 'AI生成'],
    ['templates', 'create', '创建模板'],
    ['templates', 'update', '编辑模板'],
    ['templates', 'delete', '删除模板'],
    ['publish', 'create', '创建发布任务'],
    ['publish', 'retry', '重试发布任务'],
  ]
  for (const [rk, op, title] of contentActions) {
    const item = actionItem(rk, op, title)
    if (item) contentItems.push(item)
  }

  const reviewItems: ModuleItemDef[] = []
  const reviewActions: Array<[string, string, string]> = [
    ['review', 'submit', '提交内容审核'],
    ['review', 'approve', '审核通过'],
    ['review', 'reject', '审核驳回'],
    ['db_change', 'submit', '提交SQL变更'],
    ['db_change', 'approve', 'SQL变更审核通过'],
    ['db_change', 'reject', 'SQL变更驳回'],
  ]
  for (const [rk, op, title] of reviewActions) {
    const item = actionItem(rk, op, title)
    if (item) reviewItems.push(item)
  }

  const basicItems: ModuleItemDef[] = []
  const basicActions: Array<[string, string, string]> = [
    ['account', 'read', '查看平台账号'],
    ['account', 'create', '添加平台账号'],
    ['account', 'update', '编辑平台账号'],
    ['account', 'delete', '删除平台账号'],
    ['account', 'check', '检测账号状态'],
  ]
  for (const [rk, op, title] of basicActions) {
    const item = actionItem(rk, op, title)
    if (item) basicItems.push(item)
  }

  const coveredKeys = new Set<string>()
  for (const item of [...pageItems, ...systemItems, ...contentItems, ...reviewItems, ...basicItems]) {
    if (item.readKey) coveredKeys.add(item.readKey)
    for (const key of item.writeKeys) coveredKeys.add(key)
  }

  const otherItems: ModuleItemDef[] = []
  const otherByResource = new Map<string, { resource: Permission['resource']; keys: string[] }>()
  for (const perm of permissions.value) {
    if (!perm.is_active || coveredKeys.has(perm.key)) continue
    let group = otherByResource.get(perm.resource.key)
    if (!group) {
      group = { resource: perm.resource, keys: [] }
      otherByResource.set(perm.resource.key, group)
    }
    group.keys.push(perm.key)
  }
  for (const group of otherByResource.values()) {
    const readKey = group.keys.find((k) => k.endsWith(':read'))
    const writeKeys = group.keys.filter((k) => k !== readKey)
    otherItems.push({
      id: `other-${group.resource.key}`,
      title: group.resource.name,
      subtitle: '其他权限',
      readKey,
      writeKeys,
    })
  }

  const result: ModuleDef[] = [
    { key: 'page', label: '页面权限', items: pageItems },
    { key: 'system', label: '系统设置', items: systemItems },
    { key: 'content', label: '内容管理', items: contentItems },
    { key: 'review', label: '审核管理', items: reviewItems },
    { key: 'basic', label: '基础', items: basicItems },
  ]
  if (otherItems.length > 0) {
    result.push({ key: 'other', label: '其他权限', items: otherItems })
  }
  return result
})

const allPermissionKeys = computed(() => {
  return new Set(permissions.value.filter((p) => p.is_active).map((p) => p.key))
})

const rolePermissionKeys = computed(() => {
  return new Set(rolePermissions.value.map((rp) => rp.key))
})

const inheritedKeys = computed(() => {
  return new Set(rolePermissions.value.filter((rp) => rp.grant_type === 'inherited').map((rp) => rp.key))
})

const effectiveKeys = computed(() => {
  if (selectedRole.value?.is_super_admin) return allPermissionKeys.value
  return rolePermissionKeys.value
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

async function onRoleChange(roleId: string) {
  selectedRoleId.value = roleId
  rolePermissions.value = []
  if (!roleId) return
  loading.value = true
  try {
    await fetchRolePermissions(roleId)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载角色权限失败')
  } finally {
    loading.value = false
  }
}

function ensurePermission(key: string) {
  const perm = permissionByKey.value.get(key)
  if (!perm) return null
  return perm
}

function addRolePermission(key: string) {
  const perm = ensurePermission(key)
  if (!perm) return
  if (rolePermissionKeys.value.has(key)) return
  rolePermissions.value.push({
    id: `direct-${key}`,
    key,
    operation: perm.operation,
    resource_id: perm.resource.id,
    resource_key: perm.resource.key,
    resource_name: perm.resource.name,
    grant_type: 'direct',
  })
}

function removeRolePermission(key: string) {
  rolePermissions.value = rolePermissions.value.filter((rp) => rp.key !== key)
}

function togglePermission(key: string) {
  if (selectedRole.value?.is_super_admin) return
  const existing = rolePermissions.value.find((rp) => rp.key === key)
  if (existing?.grant_type === 'inherited') return
  if (existing) {
    removeRolePermission(key)
  } else {
    addRolePermission(key)
  }
}

function toggleWrite(keys: string[]) {
  if (selectedRole.value?.is_super_admin) return
  const hasAny = keys.some((key) => rolePermissionKeys.value.has(key))
  for (const key of keys) {
    const existing = rolePermissions.value.find((rp) => rp.key === key)
    if (existing?.grant_type === 'inherited') continue
    if (hasAny) {
      removeRolePermission(key)
    } else {
      addRolePermission(key)
    }
  }
}

function selectAllRead(module: ModuleDef) {
  if (selectedRole.value?.is_super_admin) return
  for (const item of module.items) {
    if (!item.readKey) continue
    if (inheritedKeys.value.has(item.readKey)) continue
    addRolePermission(item.readKey)
  }
}

function selectAllWrite(module: ModuleDef) {
  if (selectedRole.value?.is_super_admin) return
  for (const item of module.items) {
    for (const key of item.writeKeys) {
      if (inheritedKeys.value.has(key)) continue
      addRolePermission(key)
    }
  }
}

function deselectAllRead(keys: string[]) {
  if (selectedRole.value?.is_super_admin) return
  for (const key of keys) {
    removeRolePermission(key)
  }
}

function deselectAllWrite(keys: string[]) {
  if (selectedRole.value?.is_super_admin) return
  for (const key of keys) {
    removeRolePermission(key)
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
    <PageHeader title="权限管理" subtitle="为不同角色分配页面访问和操作权限，超级管理员拥有所有权限">
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
        <div class="lg:w-[200px] shrink-0">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-4">
            <p class="text-[12px] text-[#86868B] font-medium px-1 pb-3">选择角色</p>
            <div class="flex flex-col gap-2">
              <button
                v-for="role in sortedRoles"
                :key="role.id"
                type="button"
                class="flex items-center justify-between w-full px-3 py-2.5 rounded-xl border text-left transition-all"
                :class="[
                  selectedRoleId === role.id
                    ? 'bg-[#007AFF]/10 border-[#007AFF]/30'
                    : 'bg-transparent border-black/[0.04] hover:bg-black/[0.02]',
                ]"
                @click="onRoleChange(role.id)"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <a-tag :color="roleColor(role)" size="small" class="!m-0 shrink-0">
                    {{ role.display_name }}
                  </a-tag>
                </div>
                <IconLock
                  v-if="role.is_super_admin"
                  :size="12"
                  class="text-[#ff9500] shrink-0 ml-2"
                />
              </button>
            </div>

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
            <PermissionModuleCard
              v-for="module in modules"
              :key="module.key"
              :title="module.label"
              :count="module.items.length"
              :items="module.items"
              :effective-keys="effectiveKeys"
              :inherited-keys="inheritedKeys"
              :readonly="selectedRole.is_super_admin"
              @toggle-read="togglePermission"
              @toggle-write="toggleWrite"
              @select-all-read="selectAllRead(module)"
              @select-all-write="selectAllWrite(module)"
              @deselect-all-read="deselectAllRead"
              @deselect-all-write="deselectAllWrite"
            />
          </div>
        </div>
      </div>
    </a-spin>
  </div>
</template>
