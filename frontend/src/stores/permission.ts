import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/utils/api'

export interface PermissionAccess {
  read: boolean
  write: boolean
}

export interface RBACRole {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  scope: string
  is_system: boolean
  is_super_admin: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface RBACResource {
  id: string
  key: string
  name: string
  type: string
  parent_id: string | null
  description: string | null
  is_active: boolean
  sort_order: number
}

export interface RBACPermission {
  id: string
  resource_id: string
  resource_key: string
  operation: string
  key: string
  description: string | null
  is_active: boolean
}

export interface UserRoleAssignment {
  id: string
  user_id: string
  role_id: string
  role_name: string
  role_display_name: string
  granted_by: string | null
  valid_from: string | null
  valid_until: string | null
  created_at: string
}

export interface SSDConstraint {
  id: string
  name: string
  max_roles: number
  description: string | null
  is_active: boolean
}

export interface DSDConstraint {
  id: string
  name: string
  max_roles: number
  description: string | null
  is_active: boolean
}

export const usePermissionStore = defineStore('permission', () => {
  const roles = ref<RBACRole[]>([])
  const resources = ref<RBACResource[]>([])
  const permissions = ref<RBACPermission[]>([])
  const userRoles = ref<UserRoleAssignment[]>([])
  const activeRoleIds = ref<string[]>([])
  const effectivePermissions = ref<Record<string, PermissionAccess>>({})
  const ssdConstraints = ref<SSDConstraint[]>([])
  const dsdConstraints = ref<DSDConstraint[]>([])
  const loading = ref(false)

  const isSuperAdmin = computed(() => {
    return roles.value.some(r => r.is_super_admin && userRoles.value.some(ur => ur.role_id === r.id))
  })

  const systemRoles = computed(() => roles.value.filter(r => r.is_system))
  const customRoles = computed(() => roles.value.filter(r => !r.is_system))

  function hasPermission(key: string, mode: 'read' | 'write' = 'read'): boolean {
    if (isSuperAdmin.value) return true
    const access = effectivePermissions.value[key]
    if (!access) return false
    return mode === 'read' ? access.read : access.write
  }

  function hasAnyPermission(keys: string[], mode?: 'read' | 'write'): boolean {
    return keys.some(k => hasPermission(k, mode))
  }

  function hasAllPermissions(keys: string[], mode?: 'read' | 'write'): boolean {
    return keys.every(k => hasPermission(k, mode))
  }

  function hasRole(roleName: string): boolean {
    return userRoles.value.some(ur => ur.role_name === roleName)
  }

  function hasActiveRole(roleName: string): boolean {
    return activeRoleIds.value.some(rid => {
      const role = roles.value.find(r => r.id === rid)
      return role?.name === roleName
    })
  }

  async function fetchRoles(): Promise<RBACRole[]> {
    const res = await api.get<RBACRole[]>('/v2/roles')
    roles.value = res.data
    return res.data
  }

  async function fetchResources(): Promise<RBACResource[]> {
    const res = await api.get<RBACResource[]>('/v2/resources')
    resources.value = res.data
    return res.data
  }

  async function fetchPermissions(): Promise<RBACPermission[]> {
    const res = await api.get<RBACPermission[]>('/v2/permissions')
    permissions.value = res.data
    return res.data
  }

  async function fetchUserRoles(userId: string): Promise<UserRoleAssignment[]> {
    const res = await api.get<UserRoleAssignment[]>(`/v2/users/${userId}/roles`)
    userRoles.value = res.data
    return res.data
  }

  async function fetchUserPermissions(userId: string) {
    const res = await api.get<{ user_id: string; permissions: Record<string, PermissionAccess> }>(
      `/v2/users/${userId}/permissions`
    )
    effectivePermissions.value = res.data.permissions
  }

  async function assignRoleToUser(userId: string, roleId: string) {
    await api.post(`/v2/users/${userId}/roles`, { role_id: roleId })
  }

  async function revokeRoleFromUser(userId: string, roleId: string) {
    await api.delete(`/v2/users/${userId}/roles/${roleId}`)
  }

  async function fetchSSDConstraints(): Promise<SSDConstraint[]> {
    const res = await api.get<SSDConstraint[]>('/v2/constraints/ssd')
    ssdConstraints.value = res.data
    return res.data
  }

  async function fetchDSDConstraints(): Promise<DSDConstraint[]> {
    const res = await api.get<DSDConstraint[]>('/v2/constraints/dsd')
    dsdConstraints.value = res.data
    return res.data
  }

  async function loadAllPermissionsData() {
    loading.value = true
    try {
      await Promise.all([
        fetchRoles(),
        fetchResources(),
        fetchPermissions(),
      ])
    } finally {
      loading.value = false
    }
  }

  function getResourceTree(): (RBACResource & { children?: RBACResource[] })[] {
    const map = new Map<string, RBACResource & { children?: RBACResource[] }>()
    const roots: (RBACResource & { children?: RBACResource[] })[] = []

    for (const r of resources.value) {
      map.set(r.id, { ...r, children: [] })
    }

    for (const r of map.values()) {
      if (r.parent_id && map.has(r.parent_id)) {
        map.get(r.parent_id)!.children!.push(r)
      } else {
        roots.push(r)
      }
    }

    return roots.sort((a, b) => a.sort_order - b.sort_order)
  }

  function getPermissionsByResource(resourceId: string): RBACPermission[] {
    return permissions.value.filter(p => p.resource_id === resourceId)
  }

  return {
    roles,
    resources,
    permissions,
    userRoles,
    activeRoleIds,
    effectivePermissions,
    ssdConstraints,
    dsdConstraints,
    loading,
    isSuperAdmin,
    systemRoles,
    customRoles,
    hasPermission,
    hasAnyPermission,
    hasAllPermissions,
    hasRole,
    hasActiveRole,
    fetchRoles,
    fetchResources,
    fetchPermissions,
    fetchUserRoles,
    fetchUserPermissions,
    assignRoleToUser,
    revokeRoleFromUser,
    fetchSSDConstraints,
    fetchDSDConstraints,
    loadAllPermissionsData,
    getResourceTree,
    getPermissionsByResource,
  }
})
