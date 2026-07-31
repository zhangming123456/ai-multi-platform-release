import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/utils/api'
import { isExpression, evaluatePermission } from '@/utils/permExpression'
import type { PermContext } from '@/utils/permExpression'
import { useUserStore } from '@/stores/user'

export type { PermContext } from '@/utils/permExpression'

function _buildDefaultContext(): PermContext {
  const userStore = useUserStore()
  return {
    currentUser: {
      id: userStore.userInfo?.id ?? '',
      role: userStore.userInfo?.role ?? '',
    },
  }
}

function _mergeContext(overrides?: Partial<PermContext>): PermContext {
  if (!overrides) return _buildDefaultContext()
  return { ..._buildDefaultContext(), ...overrides }
}

export const usePermissionStore = defineStore('permission', () => {
  const permissions = ref<Record<string, string>>({})
  const lastPermissionsUserId = ref<string | null>(null)

  function hasPermission(keyOrExpr: string, ctx?: PermContext): boolean {
    if (!isExpression(keyOrExpr)) {
      return keyOrExpr in permissions.value
    }
    return evaluatePermission(keyOrExpr, permissions.value, _mergeContext(ctx))
  }

  function clearPermissions() {
    permissions.value = {}
    lastPermissionsUserId.value = null
  }

  async function loadPermissions(userId?: string) {
    const { data } = await api.get<Record<string, string>>('/v2/me/permissions')
    permissions.value = data || {}
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
