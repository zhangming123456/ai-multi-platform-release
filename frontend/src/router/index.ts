import { createRouter, createWebHistory } from 'vue-router'

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
        },
        {
          path: 'accounts',
          name: 'Accounts',
          component: () => import('@/pages/Accounts.vue'),
        },
        {
          path: 'content',
          name: 'ContentList',
          component: () => import('@/pages/ContentList.vue'),
        },
        {
          path: 'content/create',
          name: 'ContentCreate',
          component: () => import('@/pages/ContentCreate.vue'),
        },
        {
          path: 'publish',
          name: 'Publish',
          component: () => import('@/pages/Publish.vue'),
        },
        {
          path: 'templates',
          name: 'Templates',
          component: () => import('@/pages/Templates.vue'),
        },
        {
          path: 'settings/token-plan',
          name: 'TokenPlan',
          component: () => import('@/pages/TokenPlan.vue'),
        },
        {
          path: 'developer/docs',
          name: 'ApiDocs',
          component: () => import('@/pages/ApiDocs.vue'),
        },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) {
    return { name: 'Login' }
  }
  if (to.name === 'Login' && token) {
    return { name: 'Dashboard' }
  }
})

export default router
