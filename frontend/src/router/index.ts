import { createRouter, createWebHistory } from 'vue-router'
import type { RouteLocationNormalized } from 'vue-router'
import { useUserStore } from '@/stores/user'

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
          meta: { permKey: 'dashboard' },
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
          meta: { permKey: 'platforms' },
        },
        {
          path: 'accounts',
          name: 'Accounts',
          component: () => import('@/pages/Accounts.vue'),
          meta: { permKey: 'accounts' },
        },
        {
          path: 'content',
          name: 'ContentList',
          component: () => import('@/pages/ContentList.vue'),
          meta: { permKey: 'content' },
        },
        {
          path: 'content/create',
          name: 'ContentCreate',
          component: () => import('@/pages/ContentCreate.vue'),
          meta: { permKey: 'content' },
        },
        {
          path: 'publish',
          name: 'Publish',
          component: () => import('@/pages/Publish.vue'),
          meta: { permKey: 'publish' },
        },
        {
          path: 'review',
          name: 'Review',
          component: () => import('@/pages/Review.vue'),
          meta: { permKey: 'review' },
        },
        {
          path: 'sql-review',
          name: 'SqlReview',
          component: () => import('@/pages/SqlReview.vue'),
          meta: { permKey: 'sql_review' },
        },
        {
          path: 'templates',
          name: 'Templates',
          component: () => import('@/pages/Templates.vue'),
          meta: { permKey: 'templates' },
        },
        {
          path: 'settings/token-plan',
          name: 'TokenPlan',
          component: () => import('@/pages/TokenPlan.vue'),
          meta: { permKey: 'token_plan' },
        },
        {
          path: 'developer/docs',
          name: 'ApiDocs',
          component: () => import('@/pages/ApiDocs.vue'),
          meta: { permKey: 'api_docs' },
        },
        {
          path: 'developer/database',
          name: 'DatabaseConsole',
          component: () => import('@/pages/DatabaseConsole.vue'),
          meta: { permKey: 'database', adminOnly: true },
        },
        {
          path: 'settings/permissions',
          name: 'PermissionManage',
          component: () => import('@/pages/PermissionManage.vue'),
          meta: { permKey: 'permission_manage', adminOnly: true },
        },
        {
          path: 'settings/roles',
          name: 'RoleManage',
          component: () => import('@/pages/RoleManage.vue'),
          meta: { permKey: 'permission_manage', adminOnly: true },
        },
      ],
    },
  ],
})

let userInfoReady = false

function hasPerm(to: RouteLocationNormalized): boolean {
  if (to.meta.skipPermCheck) return true
  const permKey = to.meta.permKey as string | undefined
  if (!permKey) return true
  const userStore = useUserStore()
  if (!userStore.userInfo?.permissions) return true
  return userStore.userInfo.permissions.includes(permKey)
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
  if (!userInfoReady || !userStore.userInfo) {
    try {
      await userStore.fetchUserInfo()
    } catch {
      userStore.logout()
      return { name: 'Login', query: { redirect: to.fullPath } }
    }
    userInfoReady = true
  }

  if (!hasPerm(to)) {
    return { name: 'Forbidden', query: { from: to.fullPath } }
  }

  return true
})

export default router
