<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconSafe, IconLock, IconEdit } from '@arco-design/web-vue/es/icon'
import { orderBy } from 'lodash-es'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'
import { isAdminTypePermission } from '@/utils/rbac'

const router = useRouter()

interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface ResourceRef {
  id: string
  key: string
  name: string
  description?: string | null
}

interface Permission {
  id: string
  key: string
  operation: string
  is_active: boolean
  resource: ResourceRef
}

function _isPageKey(key: string): boolean {
  const parts = key.split(':')
  return parts.length === 2 && (parts[1] === 'read' || parts[1] === 'write')
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
  permId: string
  resourceId: string
  title: string
  subtitle: string
  resourceName: string
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

const isNonAdminRole = computed(() => selectedRole.value?.role_type === 'other')

function filterModuleItems(items: ModuleItemDef[]): ModuleItemDef[] {
  if (!isNonAdminRole.value) return items
  return items.filter((item) => {
    const checkKey = item.readKey || item.writeKeys[0]
    if (!checkKey) return false
    return !isAdminTypePermission(checkKey)
  })
}

const permissionMap = computed(() => {
  const map = new Map<string, Permission>()
  for (const p of permissions.value) {
    map.set(p.id, p)
  }
  return map
})

const activePermissions = computed(() => {
  return orderBy(
    permissions.value.filter((p) => p.is_active),
    ['key'],
    ['asc'],
  )
})

const pageItemDefs = computed<ModuleItemDef[]>(() => {
  return activePermissions.value
    .filter((p) => _isPageKey(p.key))
    .map((p) => ({
      permId: p.id,
      resourceId: p.resource.id,
      title: p.resource.name,
      subtitle: '页面访问',
      resourceName: p.resource.name,
      readKey: p.key,
      writeKeys: [] as string[],
    }))
})

function buildActionGroupMap(
  filterKeys?: string[],
): Map<string, { readPerm?: Permission; writePerm?: Permission }> {
  const groups = new Map<string, { readPerm?: Permission; writePerm?: Permission }>()
  for (const perm of activePermissions.value) {
    if (_isPageKey(perm.key)) continue
    const rk = perm.resource.key.split(':')[0]
    if (filterKeys && !filterKeys.includes(rk)) continue
    const parts = perm.key.split(':')
    const baseKey = `${parts[0]}:${parts[1]}`
    if (!groups.has(baseKey)) {
      groups.set(baseKey, {})
    }
    const group = groups.get(baseKey)!
    if (parts.length >= 3 && parts[2] === 'write') {
      group.writePerm = perm
    } else {
      group.readPerm = perm
    }
  }
  return groups
}

function groupMapToItems(
  groups: Map<string, { readPerm?: Permission; writePerm?: Permission }>,
): ModuleItemDef[] {
  const items: ModuleItemDef[] = []
  for (const [, group] of groups) {
    const writePerm = group.writePerm
    const readPerm = group.readPerm
    const displayPerm = writePerm || readPerm!
    items.push({
      permId: displayPerm.id,
      resourceId: displayPerm.resource.id,
      title: displayPerm.resource.name,
      subtitle: '操作权限',
      resourceName: displayPerm.resource.name,
      readKey: readPerm?.key,
      writeKeys: writePerm ? [writePerm.key] : [],
    })
  }
  return items
}

function actionItemsForKeys(resourceKeys: string[]): ModuleItemDef[] {
  return groupMapToItems(buildActionGroupMap(resourceKeys))
}

const modules = computed<ModuleDef[]>(() => {
  const systemItems = actionItemsForKeys([
    'users',
    'permissions',
    'roles',
    'constraints',
    'model_config',
  ])

  const contentItems = actionItemsForKeys(['content', 'templates', 'publish'])

  const reviewItems = actionItemsForKeys(['review', 'db_change'])

  const basicItems = actionItemsForKeys(['account', 'db', 'db_history'])

  const coveredKeys = new Set<string>()
  for (const items of [pageItemDefs.value, systemItems, contentItems, reviewItems, basicItems]) {
    for (const item of items) {
      if (item.readKey) coveredKeys.add(item.readKey)
      for (const k of item.writeKeys) coveredKeys.add(k)
    }
  }

  const otherGroups = buildActionGroupMap()
  const otherItems = groupMapToItems(otherGroups).filter((item) => {
    const k = item.readKey || item.writeKeys[0]
    return !coveredKeys.has(k)
  })

  const filteredPage = filterModuleItems(pageItemDefs.value)
  const filteredSystem = filterModuleItems(systemItems)
  const filteredContent = filterModuleItems(contentItems)
  const filteredReview = filterModuleItems(reviewItems)
  const filteredBasic = filterModuleItems(basicItems)
  const filteredOther = filterModuleItems(otherItems)

  const result: ModuleDef[] = [{ key: 'page', label: '页面权限', items: filteredPage }]
  if (filteredSystem.length > 0)
    result.push({ key: 'system', label: '系统设置', items: filteredSystem })
  if (filteredContent.length > 0)
    result.push({ key: 'content', label: '内容管理', items: filteredContent })
  if (filteredReview.length > 0)
    result.push({ key: 'review', label: '审核管理', items: filteredReview })
  if (filteredBasic.length > 0)
    result.push({ key: 'basic', label: '基础操作', items: filteredBasic })
  if (filteredOther.length > 0)
    result.push({ key: 'other', label: '其他权限', items: filteredOther })
  return result
})

const allPermissionKeys = computed(() => {
  return new Set(activePermissions.value.map((p) => p.key))
})

const rolePermissionKeys = computed(() => {
  return new Set(rolePermissions.value.map((rp) => rp.key))
})

const inheritedKeys = computed(() => {
  return new Set(
    rolePermissions.value.filter((rp) => rp.grant_type === 'inherited').map((rp) => rp.key),
  )
})

const effectiveKeys = computed(() => {
  if (selectedRole.value?.is_super_admin) return allPermissionKeys.value
  return rolePermissionKeys.value
})

function getModuleReadKeys(module: ModuleDef): string[] {
  const keys: string[] = []
  for (const item of module.items) {
    if (item.readKey) keys.push(item.readKey)
  }
  return keys
}

function getModuleWriteKeys(module: ModuleDef): string[] {
  const keys: string[] = []
  for (const item of module.items) {
    for (const k of item.writeKeys) keys.push(k)
  }
  return keys
}

function moduleReadChecked(module: ModuleDef): boolean {
  const keys = getModuleReadKeys(module)
  if (keys.length === 0) return false
  return keys.every((k) => effectiveKeys.value.has(k))
}

function moduleReadIndeterminate(module: ModuleDef): boolean {
  const keys = getModuleReadKeys(module)
  const checkedCount = keys.filter((k) => effectiveKeys.value.has(k)).length
  return checkedCount > 0 && checkedCount < keys.length
}

function moduleWriteChecked(module: ModuleDef): boolean {
  const keys = getModuleWriteKeys(module)
  if (keys.length === 0) return false
  return keys.every((k) => effectiveKeys.value.has(k))
}

function moduleWriteIndeterminate(module: ModuleDef): boolean {
  const keys = getModuleWriteKeys(module)
  const checkedCount = keys.filter((k) => effectiveKeys.value.has(k)).length
  return checkedCount > 0 && checkedCount < keys.length
}

function toggleModuleAllRead(module: ModuleDef, checked: boolean) {
  if (selectedRole.value?.is_super_admin) return
  for (const key of getModuleReadKeys(module)) {
    if (inheritedKeys.value.has(key)) continue
    if (checked) {
      addRolePermission(key)
    } else {
      removeRolePermission(key)
    }
  }
}

function toggleModuleAllWrite(module: ModuleDef, checked: boolean) {
  if (selectedRole.value?.is_super_admin) return
  for (const key of getModuleWriteKeys(module)) {
    if (inheritedKeys.value.has(key)) continue
    if (checked) {
      addRolePermission(key)
    } else {
      removeRolePermission(key)
    }
  }
}

function toggleReadKey(key: string) {
  if (selectedRole.value?.is_super_admin) return
  if (inheritedKeys.value.has(key)) return
  if (effectiveKeys.value.has(key)) {
    removeRolePermission(key)
  } else {
    addRolePermission(key)
  }
}

function toggleWriteKey(key: string) {
  if (selectedRole.value?.is_super_admin) return
  if (inheritedKeys.value.has(key)) return
  if (effectiveKeys.value.has(key)) {
    removeRolePermission(key)
  } else {
    addRolePermission(key)
  }
}

async function fetchRoles() {
  const res = await api.get<Role[]>('/v2/roles')
  roles.value = Array.isArray(res.data) ? res.data : []
  if (sortedRoles.value.length > 0 && !selectedRoleId.value) {
    const firstEditable = sortedRoles.value.find((r) => !r.is_super_admin)
    selectedRoleId.value = firstEditable?.id || sortedRoles.value[0].id
  }
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
    await Promise.all([fetchRoles(), fetchPermissions()])
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

function addRolePermission(key: string) {
  if (rolePermissionKeys.value.has(key)) return
  const perm = permissionMap.value.get(activePermissions.value.find((p) => p.key === key)?.id || '')
  if (!perm) return
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

function selectedDirectPermissionIds(): string[] {
  const ids: string[] = []
  for (const rp of rolePermissions.value) {
    if (rp.grant_type === 'direct') {
      const perm = activePermissions.value.find((p) => p.key === rp.key)
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
    <PageHeader
      title="权限管理"
      subtitle="为不同角色分配页面访问和操作权限，超级管理员拥有所有权限"
    >
      <template #actions>
        <a-button
          type="outline"
          v-perm="'permissions:manage:read'"
          @click="router.push({ name: 'RBACPermissionEnumManage' })"
        >
          <template #icon><IconEdit :size="14" /></template>
          权限字典管理
        </a-button>
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
            <div
              v-for="module in modules"
              :key="module.key"
              class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5"
            >
              <div class="flex items-center justify-between mb-4">
                <div class="flex items-center gap-2">
                  <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">{{ module.label }}</h3>
                  <span class="text-[12px] text-[#86868B] font-medium">{{
                    module.items.length
                  }}</span>
                </div>
                <div class="flex items-center gap-3">
                  <a-checkbox
                    v-if="module.items.some((i) => i.readKey)"
                    :model-value="moduleReadChecked(module)"
                    :indeterminate="moduleReadIndeterminate(module)"
                    :disabled="selectedRole.is_super_admin"
                    @change="toggleModuleAllRead(module, $event)"
                  >
                    读
                  </a-checkbox>
                  <a-checkbox
                    v-if="module.items.some((i) => i.writeKeys.length > 0)"
                    :model-value="moduleWriteChecked(module)"
                    :indeterminate="moduleWriteIndeterminate(module)"
                    :disabled="selectedRole.is_super_admin"
                    @change="toggleModuleAllWrite(module, $event)"
                  >
                    写
                  </a-checkbox>
                </div>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div
                  v-for="item in module.items"
                  :key="item.permId"
                  class="flex items-center justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
                >
                  <div class="min-w-0 mr-3 flex items-center gap-2">
                    <div class="min-w-0">
                      <div class="flex items-center gap-2">
                        <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">
                          {{ item.title }}
                        </p>
                      </div>
                      <p class="text-[11px] text-[#86868B] m-0 truncate">{{ item.subtitle }}</p>
                    </div>
                  </div>
                  <div class="flex flex-col items-end gap-1 shrink-0">
                    <a-checkbox
                      v-if="item.readKey"
                      :model-value="effectiveKeys.has(item.readKey)"
                      :disabled="selectedRole.is_super_admin || inheritedKeys.has(item.readKey)"
                      @change="toggleReadKey(item.readKey!)"
                    >
                      读
                    </a-checkbox>
                    <a-checkbox
                      v-if="item.writeKeys.length > 0"
                      :model-value="item.writeKeys.some((k) => effectiveKeys.has(k))"
                      :disabled="
                        selectedRole.is_super_admin ||
                        item.writeKeys.some((k) => inheritedKeys.has(k))
                      "
                      @change="toggleWriteKey(item.writeKeys[0])"
                    >
                      写
                    </a-checkbox>
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
