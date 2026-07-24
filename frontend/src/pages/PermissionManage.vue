<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconSafe, IconLock } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'

interface PermissionDef {
  key: string
  name: string
  group: string
  type: string
}

interface PermAccess {
  read: boolean
  write: boolean
}

interface AllPermissionsResponse {
  permissions: PermissionDef[]
  roles: string[]
  role_permissions: Record<string, Record<string, PermAccess>>
}

const loading = ref(false)
const saving = ref(false)
const permissions = ref<PermissionDef[]>([])
const roles = ref<string[]>([])
const rolePermissions = ref<Record<string, Record<string, PermAccess>>>({})
const activeRole = ref('operator')

interface RoleDef {
  name: string
  display_name: string
  description: string | null
  is_builtin: boolean
  is_super_admin: boolean
  role_type: string
}

const BUILTIN_COLORS: Record<string, string> = {
  admin: 'red',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
}

const CUSTOM_COLORS = ['arcoblue', 'purple', 'cyan', 'orange', 'pink', 'gold', 'lime', 'magenta']

const roleDefs = ref<RoleDef[]>([])

const roleLabels = computed(() => {
  const map: Record<string, string> = {}
  for (const r of roleDefs.value) {
    map[r.name] = r.display_name
  }
  return map
})

const roleColors = computed(() => {
  const map: Record<string, string> = {}
  let customIdx = 0
  for (const r of roleDefs.value) {
    if (BUILTIN_COLORS[r.name]) {
      map[r.name] = BUILTIN_COLORS[r.name]
    } else {
      map[r.name] = CUSTOM_COLORS[customIdx % CUSTOM_COLORS.length]
      customIdx++
    }
  }
  return map
})

function rLabel(name: string): string {
  return roleLabels.value[name] || name
}

function rColor(name: string): string {
  return roleColors.value[name] || 'arcoblue'
}

function isSuperAdmin(name: string): boolean {
  const def = roleDefs.value.find((r) => r.name === name)
  return def?.is_super_admin ?? false
}

function isAdminType(name: string): boolean {
  const def = roleDefs.value.find((r) => r.name === name)
  return def?.role_type === 'admin'
}

function ensurePerm(key: string) {
  if (!rolePermissions.value[activeRole.value]) {
    rolePermissions.value[activeRole.value] = {}
  }
  if (!rolePermissions.value[activeRole.value][key]) {
    rolePermissions.value[activeRole.value][key] = { read: false, write: false }
  }
}

const DB_PERM_KEYS = ['database', 'db:execute', 'db:history:read']

const sortedRoles = computed(() => {
  return [...roles.value].sort((a, b) => {
    if (isSuperAdmin(a)) return -1
    if (isSuperAdmin(b)) return 1
    const builtinOrder = ['manager', 'operator', 'reviewer']
    const aIdx = builtinOrder.indexOf(a)
    const bIdx = builtinOrder.indexOf(b)
    if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx
    if (aIdx !== -1) return -1
    if (bIdx !== -1) return 1
    return 0
  })
})

const groupedPermissions = computed(() => {
  const groups: Record<string, PermissionDef[]> = {}
  const activeIsAdminType = isAdminType(activeRole.value)
  for (const p of permissions.value) {
    if (!activeIsAdminType && DB_PERM_KEYS.includes(p.key)) continue
    if (!groups[p.group]) groups[p.group] = []
    groups[p.group].push(p)
  }
  return groups
})

const editableRoles = computed(() => roles.value.filter((r) => !isSuperAdmin(r)))

const currentRolePerms = computed(() => rolePermissions.value[activeRole.value] || {})

function hasPermRead(key: string): boolean {
  return currentRolePerms.value[key]?.read ?? false
}

function hasPermWrite(key: string): boolean {
  return currentRolePerms.value[key]?.write ?? false
}

function togglePermRead(key: string) {
  if (isSuperAdmin(activeRole.value)) return
  ensurePerm(key)
  rolePermissions.value[activeRole.value][key].read = !rolePermissions.value[activeRole.value][key].read
}

function togglePermWrite(key: string) {
  if (isSuperAdmin(activeRole.value)) return
  ensurePerm(key)
  rolePermissions.value[activeRole.value][key].write = !rolePermissions.value[activeRole.value][key].write
}

function toggleGroupAllRead(group: string) {
  if (isSuperAdmin(activeRole.value)) return
  const groupPerms = groupedPermissions.value[group] || []
  const keys = groupPerms.map((p) => p.key)
  const allSelected = keys.every((k) => hasPermRead(k))
  const newVal = !allSelected
  for (const k of keys) {
    ensurePerm(k)
    rolePermissions.value[activeRole.value][k].read = newVal
  }
}

function toggleGroupAllWrite(group: string) {
  if (isSuperAdmin(activeRole.value)) return
  const groupPerms = groupedPermissions.value[group] || []
  const keys = groupPerms.map((p) => p.key)
  const allSelected = keys.every((k) => hasPermWrite(k))
  const newVal = !allSelected
  for (const k of keys) {
    ensurePerm(k)
    rolePermissions.value[activeRole.value][k].write = newVal
  }
}

function isGroupAllReadSelected(group: string): boolean {
  const groupPerms = groupedPermissions.value[group] || []
  return groupPerms.every((p) => hasPermRead(p.key))
}

function isGroupAllWriteSelected(group: string): boolean {
  const groupPerms = groupedPermissions.value[group] || []
  return groupPerms.every((p) => hasPermWrite(p.key))
}

function isGroupPartialReadSelected(group: string): boolean {
  const groupPerms = groupedPermissions.value[group] || []
  const selectedCount = groupPerms.filter((p) => hasPermRead(p.key)).length
  return selectedCount > 0 && selectedCount < groupPerms.length
}

function isGroupPartialWriteSelected(group: string): boolean {
  const groupPerms = groupedPermissions.value[group] || []
  const selectedCount = groupPerms.filter((p) => hasPermWrite(p.key)).length
  return selectedCount > 0 && selectedCount < groupPerms.length
}

onMounted(async () => {
  loading.value = true
  try {
    const [permRes, rolesRes] = await Promise.all([
      api.get<AllPermissionsResponse>('/permissions/all'),
      api.get<RoleDef[]>('/roles'),
    ])
    permissions.value = permRes.data.permissions
    roles.value = permRes.data.roles
    rolePermissions.value = JSON.parse(JSON.stringify(permRes.data.role_permissions))
    roleDefs.value = rolesRes.data
    if (editableRoles.value.length > 0) {
      activeRole.value = editableRoles.value[0]
    }
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载权限数据失败')
  } finally {
    loading.value = false
  }
})

async function savePermissions() {
  if (isSuperAdmin(activeRole.value)) return
  saving.value = true
  try {
    const permsToSave: Record<string, PermAccess> = {}
    const rolePermMap = rolePermissions.value[activeRole.value] || {}
    for (const [key, access] of Object.entries(rolePermMap)) {
      if (access.read || access.write) {
        permsToSave[key] = { read: access.read, write: access.write }
      }
    }
    await api.put(`/permissions/role/${activeRole.value}`, {
      role: activeRole.value,
      permissions: permsToSave,
    })
    Message.success(`${rLabel(activeRole.value)} 权限保存成功`)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader title="权限管理" subtitle="为不同角色分配页面访问和操作权限，超级管理员拥有所有权限">
      <template #actions>
        <a-button
          v-if="!isSuperAdmin(activeRole)"
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
        <div class="lg:w-[200px] shrink-0">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-3">
            <p class="text-[12px] text-[#86868B] font-medium px-3 pb-2 pt-1">选择角色</p>
            <div
              v-for="role in sortedRoles"
              :key="role"
              :class="[
                'flex items-center gap-2.5 px-3 py-2.5 rounded-xl cursor-pointer transition-all duration-200 mb-1',
                activeRole === role
                  ? 'bg-[#007aff]/[0.08] text-[#007aff]'
                  : 'hover:bg-black/[0.03] text-[#1D1D1F]',
              ]"
              @click="activeRole = role"
            >
              <a-tag :color="rColor(role)" size="small" class="!m-0">
                {{ rLabel(role) }}
              </a-tag>
              <IconLock v-if="isSuperAdmin(role)" :size="13" class="text-[#ff9500]" />
              {{ isSuperAdmin(role) ? '' : '' }}
            </div>
          </div>
        </div>

        <div class="flex-1 min-w-0">
          <div
            v-if="isSuperAdmin(activeRole)"
            class="bg-[#ff9500]/[0.06] border border-[#ff9500]/[0.15] rounded-2xl p-5 mb-5 flex items-start gap-3"
          >
            <IconSafe :size="20" class="text-[#ff9500] mt-0.5 shrink-0" />
            <div>
              <p class="text-[14px] font-semibold text-[#1D1D1F] m-0">超级管理员拥有所有权限</p>
              <p class="text-[12px] text-[#86868B] m-0 mt-1">
                超级管理员默认拥有全部页面和操作权限，不可修改。
              </p>
            </div>
          </div>

          <div class="space-y-4">
            <div
              v-for="(perms, group) in groupedPermissions"
              :key="group"
              class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden"
            >
              <div
                class="flex items-center justify-between px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]"
              >
                <div class="flex items-center gap-2">
                  <span class="text-[14px] font-semibold text-[#1D1D1F]">{{ group }}</span>
                  <a-tag size="small" color="gray" class="!m-0">{{ perms.length }}</a-tag>
                </div>
                <div v-if="!isSuperAdmin(activeRole)" class="flex items-center gap-4">
                  <a-checkbox
                    :model-value="isGroupAllReadSelected(group)"
                    :indeterminate="isGroupPartialReadSelected(group)"
                    @change="toggleGroupAllRead(group)"
                  >
                    全选读
                  </a-checkbox>
                  <a-checkbox
                    :model-value="isGroupAllWriteSelected(group)"
                    :indeterminate="isGroupPartialWriteSelected(group)"
                    @change="toggleGroupAllWrite(group)"
                  >
                    全选写
                  </a-checkbox>
                </div>
                <a-tag v-else color="green" size="small" class="!m-0">已拥有</a-tag>
              </div>

              <div class="p-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-2 gap-2">
                <div
                  v-for="perm in perms"
                  :key="perm.key"
                  :class="[
                    'flex items-center justify-between px-3.5 py-3 rounded-xl border transition-all duration-200',
                    isSuperAdmin(activeRole)
                      ? 'border-[#30d158]/20 bg-[#30d158]/[0.04] cursor-default'
                      : (hasPermRead(perm.key) || hasPermWrite(perm.key))
                        ? 'border-[#007aff]/20 bg-[#007aff]/[0.04]'
                        : 'border-black/[0.06] bg-white',
                  ]"
                >
                  <div class="min-w-0 flex-1">
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 leading-tight">{{ perm.name }}</p>
                    <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                      {{ perm.type === 'page' ? '页面访问' : '操作权限' }}
                    </p>
                  </div>
                  <div v-if="isSuperAdmin(activeRole)" class="w-4 h-4 rounded-full bg-[#30d158] flex items-center justify-center shrink-0 ml-3">
                    <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                      <path d="M1 4L3.5 6.5L9 1" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                  </div>
                  <div v-else class="flex items-center gap-3 shrink-0 ml-4">
                    <div class="flex items-center gap-1 cursor-pointer" @click.stop="togglePermRead(perm.key)">
                      <a-switch :model-value="hasPermRead(perm.key)" size="small" />
                      <span class="text-[12px] text-[#86868B] w-4 text-center">读</span>
                    </div>
                    <div class="flex items-center gap-1 cursor-pointer" @click.stop="togglePermWrite(perm.key)">
                      <a-switch :model-value="hasPermWrite(perm.key)" size="small" />
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
