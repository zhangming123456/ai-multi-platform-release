import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/utils/api'

export interface PermissionAccess {
  read: boolean
  write: boolean
}

export const usePermissionStore = defineStore('permission', () => {
  const permissions = ref<Record<string, PermissionAccess>>({})
  const lastPermissionsUserId = ref<string | null>(null)

  function hasPermission(key: string, mode: 'read' | 'write' = 'read'): boolean {
    const access = permissions.value[key]
    if (!access) return false
    return mode === 'read' ? access.read : access.write
  }

  function clearPermissions() {
    permissions.value = {}
    lastPermissionsUserId.value = null
  }

  async function loadPermissions(userId?: string) {
    const { data } = await api.get<{ permissions?: Record<string, PermissionAccess> }>('/auth/me')
    permissions.value = data.permissions || {}
    lastPermissionsUserId.value = userId ?? null
  }

  return {
    permissions,
    lastPermissionsUserId,
    hasPermission,
    loadPermissions,
    clearPermissions,
  }
})
