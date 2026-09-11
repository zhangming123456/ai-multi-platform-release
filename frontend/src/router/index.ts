import { createRouter, createWebHistory, type RouteMeta } from 'vue-router'
import type { RouteLocationNormalized } from 'vue-router'
import { watch } from 'vue'
import { useUserStore } from '@/stores/user'
import { usePermissionStore } from '@/stores/permission'

declare module 'vue-router' {
  interface RouteMeta {
    sidebarType?:
      | 'top'
      | 'content'
      | 'review'
      | 'inspection'
      | 'platforms'
      | 'rbac'
      | 'system'
      | 'material'
      | 'dev'
    sidebarOrder?: number
    icon?: string
  }
}

export type SidebarType = NonNullable<RouteMeta['sidebarType']>

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/pages/Login.vue'),
      meta: { title: '登录', public: true },
    },
    {
      path: '/',
      component: () => import('@/components/layout/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/pages/Dashboard.vue'),
          meta: {
            title: '仪表盘',
            permKey: 'dashboard:read',
            sidebarType: 'top',
            sidebarOrder: 0,
            icon: 'home',
          },
        },
        {
          path: '403',
          name: 'Forbidden',
          component: () => import('@/pages/Forbidden.vue'),
          meta: { title: '无权限', skipPermCheck: true },
        },
        {
          path: 'profile',
          name: 'Profile',
          component: () => import('@/pages/Profile.vue'),
          meta: { title: '个人中心', skipPermCheck: true },
        },
        {
          path: 'platforms',
          name: 'Platforms',
          component: () => import('@/pages/Platforms.vue'),
          meta: {
            title: '平台管理',
            permKey: 'platforms:read',
            sidebarType: 'platforms',
            sidebarOrder: 0,
            icon: 'apps',
          },
        },
        {
          path: 'content',
          name: 'ContentList',
          component: () => import('@/pages/ContentList.vue'),
          meta: {
            title: '内容列表',
            permKey: 'content:read',
            sidebarType: 'content',
            sidebarOrder: 0,
            icon: 'file',
          },
        },
        {
          path: 'content/create',
          name: 'ContentCreate',
          component: () => import('@/pages/ContentCreate.vue'),
          meta: {
            title: '创作内容',
            permKey: 'content:read',
            sidebarType: 'content',
            sidebarOrder: 1,
            icon: 'file',
          },
        },
        {
          path: 'content/generations',
          name: 'AIGenerationHistory',
          component: () => import('@/pages/AIGenerationHistory.vue'),
          meta: {
            title: '生成历史',
            permKey: 'content:read',
            sidebarType: 'content',
            sidebarOrder: 2,
            icon: 'clock',
          },
        },
        {
          path: 'campaigns',
          name: 'CampaignManage',
          component: () => import('@/pages/CampaignManage.vue'),
          meta: {
            title: '活动管理',
            permKey: 'campaign:read',
            sidebarType: 'content',
            sidebarOrder: 3,
            icon: 'gift',
          },
        },
        {
          path: 'campaigns/create',
          name: 'CampaignCreate',
          component: () => import('@/pages/CampaignEdit.vue'),
          meta: { title: '新增活动', permKey: 'campaign:create:write' },
        },
        {
          path: 'campaigns/:id',
          name: 'CampaignDetail',
          component: () => import('@/pages/CampaignDetail.vue'),
          meta: { title: '活动详情', permKey: 'campaign:read' },
        },
        {
          path: 'campaigns/:id/edit',
          name: 'CampaignEdit',
          component: () => import('@/pages/CampaignEdit.vue'),
          meta: { title: '编辑活动', permKey: 'campaign:update:write' },
        },
        {
          path: 'publish',
          name: 'Publish',
          component: () => import('@/pages/Publish.vue'),
          meta: {
            title: '发布管理',
            permKey: 'publish:read',
            sidebarType: 'content',
            sidebarOrder: 4,
            icon: 'send',
          },
        },
        {
          path: 'review',
          name: 'Review',
          component: () => import('@/pages/Review.vue'),
          meta: {
            title: '内容审核',
            permKey: 'review:read',
            sidebarType: 'review',
            sidebarOrder: 0,
            icon: 'check',
          },
        },
        {
          path: 'sql-review',
          name: 'SqlReview',
          component: () => import('@/pages/SqlReview.vue'),
          meta: {
            title: 'SQL审核',
            permKey: 'sql_review:read',
            sidebarType: 'review',
            sidebarOrder: 1,
            icon: 'storage',
          },
        },
        {
          path: 'templates',
          name: 'Templates',
          component: () => import('@/pages/Templates.vue'),
          meta: {
            title: '模板管理',
            permKey: 'templates:read',
            sidebarType: 'content',
            sidebarOrder: 5,
            icon: 'apps',
          },
        },
        {
          path: 'materials/images',
          name: 'MaterialImageList',
          component: () => import('@/pages/MaterialList.vue'),
          meta: {
            title: '图片素材',
            permKey: 'material:read',
            sidebarType: 'material',
            sidebarOrder: 0,
            icon: 'image',
          },
        },
        {
          path: 'settings/token-plan',
          name: 'TokenPlan',
          component: () => import('@/pages/TokenPlan.vue'),
          meta: {
            title: 'AI 模型服务商管理',
            permKey: 'token_plan:read',
            sidebarType: 'system',
            sidebarOrder: 0,
            icon: 'settings',
          },
        },
        {
          path: 'settings/creative',
          name: 'CreativeSettings',
          component: () => import('@/pages/CreativeSettings.vue'),
          meta: {
            title: '创作设置',
            permKey: 'templates:read',
            sidebarType: 'system',
            sidebarOrder: 1,
            icon: 'settings',
          },
        },
        {
          path: 'stores',
          name: 'StoreManage',
          component: () => import('@/pages/StoreManage.vue'),
          meta: {
            title: '门店管理',
            permKey: 'stores:read',
            sidebarType: 'inspection',
            sidebarOrder: 0,
            icon: 'apps',
          },
        },
        {
          path: 'inspection',
          name: 'InspectionList',
          component: () => import('@/pages/InspectionList.vue'),
          meta: {
            title: '巡店检查',
            permKey: 'inspection:read',
            sidebarType: 'inspection',
            sidebarOrder: 1,
            icon: 'check',
          },
        },
        {
          path: 'inspection/templates',
          name: 'InspectionTemplateList',
          component: () => import('@/pages/InspectionTemplateList.vue'),
          meta: {
            title: '检查表模板',
            permKey: 'inspection:template:read',
            sidebarType: 'inspection',
            sidebarOrder: 2,
            icon: 'edit',
          },
        },
        {
          path: 'inspection/templates/create',
          name: 'InspectionTemplateCreate',
          component: () => import('@/pages/InspectionTemplateEdit.vue'),
          meta: { title: '新建检查表模板', permKey: 'inspection:template:create:write' },
        },
        {
          path: 'inspection/templates/:id/edit',
          name: 'InspectionTemplateEdit',
          component: () => import('@/pages/InspectionTemplateEdit.vue'),
          meta: { title: '编辑检查表模板', permKey: 'inspection:template:update:write' },
        },
        {
          path: 'inspection/materials',
          name: 'InspectionMaterialList',
          component: () => import('@/pages/InspectionMaterialList.vue'),
          meta: {
            title: '素材管理',
            permKey: 'inspection:material:read',
            sidebarType: 'inspection',
            sidebarOrder: 3,
            icon: 'file',
          },
        },
        {
          path: 'inspection/materials/create',
          name: 'InspectionMaterialCreate',
          component: () => import('@/pages/InspectionMaterialEdit.vue'),
          meta: { title: '新建素材', permKey: 'inspection:material:create:write' },
        },
        {
          path: 'inspection/materials/:id/edit',
          name: 'InspectionMaterialEdit',
          component: () => import('@/pages/InspectionMaterialEdit.vue'),
          meta: { title: '编辑素材', permKey: 'inspection:material:update:write' },
        },
        {
          path: 'inspection-tasks',
          name: 'InspectionTaskList',
          component: () => import('@/pages/InspectionTaskList.vue'),
          meta: {
            title: '整改任务',
            permKey: 'inspection:task:read',
            sidebarType: 'inspection',
            sidebarOrder: 4,
            icon: 'check',
          },
        },
        {
          path: 'inspection-tasks/:id',
          name: 'InspectionTaskDetail',
          component: () => import('@/pages/InspectionTaskDetail.vue'),
          meta: { title: '整改任务详情', permKey: 'inspection:task:read' },
        },
        {
          path: 'inspection/create',
          name: 'InspectionCreate',
          component: () => import('@/pages/InspectionEdit.vue'),
          meta: { title: '发起巡店', permKey: 'inspection:create:write' },
        },
        {
          path: 'inspection/:id',
          name: 'InspectionDetail',
          component: () => import('@/pages/InspectionDetail.vue'),
          meta: { title: '巡店详情', permKey: 'inspection:read' },
        },
        {
          path: 'inspection/:id/edit',
          name: 'InspectionEdit',
          component: () => import('@/pages/InspectionEdit.vue'),
          meta: { title: '编辑巡店', permKey: 'inspection:update:write' },
        },
        {
          path: 'developer/docs',
          name: 'ApiDocs',
          component: () => import('@/pages/ApiDocs.vue'),
          meta: {
            title: 'API文档',
            permKey: 'api_docs:read',
            sidebarType: 'system',
            sidebarOrder: 1,
            icon: 'code',
          },
        },
        {
          path: 'developer/database',
          name: 'DatabaseConsole',
          component: () => import('@/pages/DatabaseConsole.vue'),
          meta: {
            title: '数据库控制台',
            permKey: 'db:read',
            sidebarType: 'system',
            sidebarOrder: 2,
            icon: 'storage',
          },
        },
        {
          path: 'settings/user-creation-review',
          name: 'UserCreationReview',
          component: () => import('@/pages/UserCreationReview.vue'),
          meta: {
            title: '用户注册审核',
            permKey: 'review:read',
            sidebarType: 'review',
            sidebarOrder: 2,
            icon: 'user',
          },
        },
        {
          path: 'rbac/users',
          name: 'RBACUserManage',
          component: () => import('@/pages/RBACUserManage.vue'),
          meta: {
            title: '用户管理',
            permKey: 'users:read',
            sidebarType: 'rbac',
            sidebarOrder: 0,
            icon: 'safe',
          },
        },
        {
          path: 'rbac/users/create',
          name: 'RBACUserCreate',
          component: () => import('@/pages/RBACUserCreate.vue'),
          meta: { title: '创建用户', permKey: 'users:create:write' },
        },
        {
          path: 'rbac/users/:id/edit',
          name: 'RBACUserEdit',
          component: () => import('@/pages/RBACUserEdit.vue'),
          meta: { title: '编辑用户', permKey: 'users:update:read || isSelf(id)' },
        },
        {
          path: 'rbac/users/:id/password',
          name: 'RBACUserPassword',
          component: () => import('@/pages/RBACUserPassword.vue'),
          meta: { title: '修改密码', permKey: 'users:change_password:write || isSelf(id)' },
        },
        {
          path: 'rbac/users/:id/permissions',
          name: 'RBACUserPermissionCustomize',
          component: () => import('@/pages/RBACUserPermissionCustomize.vue'),
          meta: { title: '自定义权限', permKey: 'users:custom_permissions:read || isSelf(id)' },
        },
        {
          path: 'rbac/roles',
          name: 'RBACRoleManage',
          component: () => import('@/pages/RBACRoleManage.vue'),
          meta: {
            title: '角色管理',
            permKey: 'roles:read',
            sidebarType: 'rbac',
            sidebarOrder: 1,
            icon: 'user',
          },
        },
        {
          path: 'rbac/permissions',
          name: 'RBACPermissionManage',
          component: () => import('@/pages/RBACPermissionManage.vue'),
          meta: {
            title: '权限管理',
            permKey: 'permissions:read',
            sidebarType: 'rbac',
            sidebarOrder: 2,
            icon: 'safe',
          },
        },
        {
          path: 'rbac/permissions/enum',
          name: 'RBACPermissionEnumManage',
          component: () => import('@/pages/RBACPermissionEnumManage.vue'),
          meta: {
            title: '权限字典管理',
            permKey: 'permissions:manage:read',
            sidebarType: 'rbac',
            sidebarOrder: 3,
            icon: 'edit',
          },
        },
        {
          path: 'rbac/permissions/enum/create',
          name: 'RBACPermissionEnumCreate',
          component: () => import('@/pages/RBACPermissionEnumEdit.vue'),
          meta: { title: '新增权限字典', permKey: 'permissions:manage:write' },
        },
        {
          path: 'rbac/permissions/enum/edit/:resourceId',
          name: 'RBACPermissionEnumEdit',
          component: () => import('@/pages/RBACPermissionEnumEdit.vue'),
          meta: { title: '编辑权限字典', permKey: 'permissions:manage:write' },
        },
        {
          path: 'rbac/constraints',
          name: 'RBACConstraintManage',
          component: () => import('@/pages/RBACConstraintManage.vue'),
          meta: {
            title: '约束管理',
            permKey: 'constraints:read',
            sidebarType: 'rbac',
            sidebarOrder: 4,
            icon: 'safe',
          },
        },
        {
          path: 'notifications',
          name: 'Notifications',
          component: () => import('@/pages/Notifications.vue'),
          meta: { title: '通知中心', skipPermCheck: true },
        },
        {
          path: 'settings/notification-dict',
          name: 'NotificationDictManage',
          component: () => import('@/pages/NotificationDictManage.vue'),
          meta: {
            title: '通知消息字典',
            permKey: 'notification:enum:read',
            sidebarType: 'system',
            sidebarOrder: 3,
            icon: 'tool',
          },
        },
        {
          path: 'settings/holiday-sources',
          name: 'HolidaySources',
          component: () => import('@/pages/HolidaySources.vue'),
          meta: {
            title: '日历订阅源',
            permKey: 'holiday_source:read',
            sidebarType: 'system',
            sidebarOrder: 4,
            icon: 'tool',
          },
        },
        {
          path: 'dev/condition-test',
          name: 'ConditionTest',
          component: () => import('@/pages/ConditionTest.vue'),
          meta: {
            title: '条件组件测试',
            skipPermCheck: true,
            sidebarType: 'dev',
            sidebarOrder: 0,
            icon: 'code',
          },
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
  return permStore.hasPermission(permKey, {
    ...(to.query ?? {}),
    ...(to.params ?? {}),
  })
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

router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  document.title = title ? `${title} - 多平台矩阵管理` : '多平台矩阵管理系统'
})

let _permWatchStarted = false

export function startRoutePermWatcher() {
  if (_permWatchStarted) return
  const permStore = usePermissionStore()
  watch(
    () => permStore.permissions,
    () => {
      const token = localStorage.getItem('token')
      if (!token) return
      if (Object.keys(permStore.permissions).length === 0) return
      const current = router.currentRoute.value
      if (current.meta.public || current.meta.skipPermCheck) return
      if (current.name === 'Login' || current.name === 'Forbidden') return
      if (!hasPerm(current)) {
        router.replace({ name: 'Forbidden', query: { from: current.fullPath } })
      }
    },
  )
  _permWatchStarted = true
}

export default router
