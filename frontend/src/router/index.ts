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
          meta: { skipPermCheck: true },
        },
        {
          path: 'platforms',
          name: 'Platforms',
          component: () => import('@/pages/Platforms.vue'),
          meta: { permKey: 'platforms:read' },
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
          meta: { permKey: 'content:create' },
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
          meta: { permKey: 'db:read' },
        },
        {
          path: 'settings/user-creation-review',
          name: 'UserCreationReview',
          component: () => import('@/pages/UserCreationReview.vue'),
          meta: { permKey: 'review:read' },
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
      ],
    },
  ],
})

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
  const permStore = usePermissionStore()

  try {
    if (!userStore.userInfo) {
      await userStore.fetchUserInfo()
    }
    if (!userStore.userInfo) {
      throw new Error('无法获取用户信息')
    }
    if (
      permStore.lastPermissionsUserId !== userStore.userInfo.id ||
      Object.keys(permStore.permissions).length === 0
    ) {
      await permStore.loadPermissions(userStore.userInfo.id)
    }
  } catch {
    permStore.clearPermissions()
    userStore.logout()
    return { name: 'Login', query: { redirect: to.fullPath } }
  }

  if (!hasPerm(to)) {
    return { name: 'Forbidden', query: { from: to.fullPath } }
  }

  return true
})

export default router
