import { createRouter, createWebHistory } from 'vue-router'
import type { RouteLocationNormalized } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { usePermissionStore } from '@/stores/permission'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/pages/Login.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/components/layout/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/pages/Dashboard.vue'),
          meta: { permKey: 'dashboard:read' },
        },
        {
          path: '403',
          name: 'Forbidden',
          component: () => import('@/pages/Forbidden.vue'),
          meta: { skipPermCheck: true },
        },
        {
          path: 'profile',
          name: 'Profile',
          component: () => import('@/pages/Profile.vue'),
        },
        {
          path: 'platforms',
          name: 'Platforms',
          component: () => import('@/pages/Platforms.vue'),
          meta: { permKey: 'platforms:read' },
        },
        {
          path: 'accounts',
          name: 'Accounts',
          redirect: { name: 'RBACUserManage' },
        },
        {
          path: 'content',
          name: 'ContentList',
          component: () => import('@/pages/ContentList.vue'),
          meta: { permKey: 'content:read' },
        },
        {
          path: 'content/create',
          name: 'ContentCreate',
          component: () => import('@/pages/ContentCreate.vue'),
          meta: { permKey: 'content:read' },
        },
        {
          path: 'publish',
          name: 'Publish',
          component: () => import('@/pages/Publish.vue'),
          meta: { permKey: 'publish:read' },
        },
        {
          path: 'review',
          name: 'Review',
          component: () => import('@/pages/Review.vue'),
          meta: { permKey: 'review:read' },
        },
        {
          path: 'sql-review',
          name: 'SqlReview',
          component: () => import('@/pages/SqlReview.vue'),
          meta: { permKey: 'sql_review:read' },
        },
        {
          path: 'templates',
          name: 'Templates',
          component: () => import('@/pages/Templates.vue'),
          meta: { permKey: 'templates:read' },
        },
        {
          path: 'settings/token-plan',
          name: 'TokenPlan',
          component: () => import('@/pages/TokenPlan.vue'),
          meta: { permKey: 'token_plan:read' },
        },
        {
          path: 'developer/docs',
          name: 'ApiDocs',
          component: () => import('@/pages/ApiDocs.vue'),
          meta: { permKey: 'api_docs:read' },
        },
        {
          path: 'developer/database',
          name: 'DatabaseConsole',
          component: () => import('@/pages/DatabaseConsole.vue'),
          meta: { permKey: 'database:read' },
        },
        {
          path: 'settings/permissions',
          name: 'PermissionManage',
          redirect: { name: 'RBACPermissionManage' },
        },
        {
          path: 'settings/roles',
          name: 'RoleManage',
          redirect: { name: 'RBACRoleManage' },
        },
        {
          path: 'rbac/users',
          name: 'RBACUserManage',
          component: () => import('@/pages/RBACUserManage.vue'),
          meta: { permKey: 'users:read' },
        },
        {
          path: 'rbac/roles',
          name: 'RBACRoleManage',
          component: () => import('@/pages/RBACRoleManage.vue'),
          meta: { permKey: 'roles:read' },
        },
        {
          path: 'rbac/permissions',
          name: 'RBACPermissionManage',
          component: () => import('@/pages/RBACPermissionManage.vue'),
          meta: { permKey: 'permissions:read' },
        },
        {
          path: 'rbac/constraints',
          name: 'RBACConstraintManage',
          component: () => import('@/pages/RBACConstraintManage.vue'),
          meta: { permKey: 'constraints:read' },
        },
        {
          path: 'settings/user-creation-review',
          name: 'UserCreationReview',
          component: () => import('@/pages/UserCreationReview.vue'),
          meta: { permKey: 'user_creation_review:read' },
        },
      ],
    },
  ],
})

let lastPermissionsUserId: string | null = null

function hasPerm(to: RouteLocationNormalized): boolean {
  if (to.meta.skipPermCheck) return true
  const permKey = to.meta.permKey as string | undefined
  if (!permKey) return true
  const permStore = usePermissionStore()
  return permStore.hasPermission(permKey, 'read')
}

router.beforeEach(async (to) => {
  const token = localStorage.getItem('token')

  if (to.name === 'Login' && token) {
    return { name: 'Dashboard' }
  }

  if (to.meta.public) {
    return true
  }

  if (!token) {
    return { name: 'Login', query: { redirect: to.fullPath } }
  }

  const userStore = useUserStore()
  if (!userStore.userInfo) {
    try {
      await userStore.fetchUserInfo()
    } catch {
      userStore.logout()
      lastPermissionsUserId = null
      return { name: 'Login', query: { redirect: to.fullPath } }
    }
  }

  const permStore = usePermissionStore()
  const userId = userStore.userInfo?.id
  if (userId && lastPermissionsUserId !== userId) {
    try {
      await permStore.loadAllPermissionsData()
      await Promise.all([
        permStore.fetchUserRoles(userId),
        permStore.fetchUserPermissions(userId),
      ])
      lastPermissionsUserId = userId
    } catch {
      // 加载失败继续，权限检查会返回 false
    }
  }

  if (!hasPerm(to)) {
    return { name: 'Forbidden', query: { from: to.fullPath } }
  }

  return true
})

export default router
