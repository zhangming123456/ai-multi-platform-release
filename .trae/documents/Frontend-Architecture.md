# 前端架构设计

## 目录结构

```
frontend/src/
├── App.vue                     # 根组件（router-view）
├── main.ts                     # 应用入口（Pinia + Router + Arco Design + 权限指令注册）
├── style.css                   # 全局样式（TailwindCSS + 自定义动画）
├── env.d.ts                    # 环境变量类型声明
├── router/
│   └── index.ts                # 路由定义 + beforeEach 权限守卫
├── stores/
│   ├── user.ts                 # 登录/用户信息/退出
│   ├── permission.ts           # 权限状态管理（有效权限加载、hasPermission）
│   ├── notification.ts         # 通知状态（未读计数、列表、已读标记）
│   ├── region.ts               # 区域/平台切换状态
│   └── tokenPlan.ts            # AI 模型配置状态管理
├── types/
│   └── index.ts                # 全局 TypeScript 类型定义
├── utils/
│   ├── api.ts                  # Axios 实例（baseURL + JWT 拦截器 + 401 处理）
│   ├── permExpression.ts       # 权限表达式解析与求值工具
│   ├── rbac.ts                 # RBAC 相关辅助函数
│   └── time.ts                 # 时间格式化工具
├── directives/
│   └── permission.ts           # v-perm 指令（按钮级权限控制）
├── components/
│   ├── layout/
│   │   ├── AppLayout.vue       # 主布局（侧边栏 + 顶栏 + 内容区）
│   │   ├── AppHeader.vue       # 顶部导航（面包屑 + 通知铃铛 + 用户菜单）
│   │   ├── AppSidebar.vue      # 侧边导航（动态菜单 + 折叠/展开）
│   │   ├── PageHeader.vue      # 页面标题组件（标题 + 副标题 + actions 插槽）
│   │   └── RegionSwitcher.vue  # 平台/区域切换器
│   ├── rbac/
│   │   ├── PermissionModuleCard.vue   # 权限模块卡片（权限管理页用）
│   │   └── ResourcePermissionCard.vue # 资源权限卡片（含权限描述展示）
│   └── shared/
│       ├── Modal.vue           # 通用弹窗组件
│       ├── PlatformIcon.vue    # 平台图标组件（公众号/小红书/抖音/视频号）
│       ├── SegmentedControl.vue # 分段控制器组件
│       ├── StatCard.vue        # 统计卡片组件（动画 + 图标 + 趋势）
│       ├── StatusBadge.vue     # 状态标签组件
│       └── NotificationBell.vue # 通知铃铛组件
└── pages/
    ├── Login.vue               # 登录页（毛玻璃卡片）
    ├── Dashboard.vue           # 仪表盘
    ├── Platforms.vue           # 平台账号管理
    ├── ContentList.vue         # 内容列表
    ├── ContentCreate.vue       # AI 内容生成
    ├── Publish.vue             # 发布管理
    ├── Review.vue              # 内容审核
    ├── SqlReview.vue           # SQL 变更审核
    ├── Templates.vue           # 模板管理
    ├── TokenPlan.vue           # 模型配置
    ├── ApiDocs.vue             # API 文档（iframe Swagger）
    ├── DatabaseConsole.vue     # 数据库控制台
    ├── UserCreationReview.vue  # 用户注册审核
    ├── Profile.vue             # 个人中心
    ├── Forbidden.vue           # 403 无权限页
    ├── RBACUserManage.vue      # RBAC 用户管理
    ├── RBACUserCreate.vue      # RBAC 创建用户
    ├── RBACUserEdit.vue        # RBAC 编辑用户
    ├── RBACUserPassword.vue    # RBAC 修改密码
    ├── RBACUserPermissionCustomize.vue  # 用户自定义权限覆盖
    ├── RBACRoleManage.vue      # RBAC 角色管理
    ├── RBACPermissionManage.vue         # RBAC 权限管理
    ├── RBACPermissionEnumManage.vue     # 权限枚举编辑（超管专用）
    └── RBACConstraintManage.vue         # RBAC 约束管理
```

## 路由定义

| 路由 | 名称 | 组件 | 鉴权 | 权限 Key | 说明 |
|------|------|------|------|----------|------|
| /login | Login | Login.vue | 公开 | — | 登录页 |
| / | Dashboard | Dashboard.vue | 需鉴权 | dashboard:read | 仪表盘 |
| /403 | Forbidden | Forbidden.vue | 需鉴权 | skipPermCheck | 无权限提示 |
| /profile | Profile | Profile.vue | 需鉴权 | skipPermCheck | 个人中心 |
| /platforms | Platforms | Platforms.vue | 需鉴权 | platforms:read | 平台账号管理 |
| /content | ContentList | ContentList.vue | 需鉴权 | content:read | 内容列表 |
| /content/create | ContentCreate | ContentCreate.vue | 需鉴权 | content:read | 创作内容 |
| /publish | Publish | Publish.vue | 需鉴权 | publish:read | 发布管理 |
| /review | Review | Review.vue | 需鉴权 | review:read | 内容审核 |
| /sql-review | SqlReview | SqlReview.vue | 需鉴权 | sql_review:read | SQL 审核 |
| /templates | Templates | Templates.vue | 需鉴权 | templates:read | 模板管理 |
| /settings/token-plan | TokenPlan | TokenPlan.vue | 需鉴权 | token_plan:read | Token 方案 |
| /settings/user-creation-review | UserCreationReview | UserCreationReview.vue | 需鉴权 | review:read | 用户注册审核 |
| /developer/docs | ApiDocs | ApiDocs.vue | 需鉴权 | api_docs:read | API 文档 |
| /developer/database | DatabaseConsole | DatabaseConsole.vue | 需鉴权 | db:read | 数据库控制台 |
| /rbac/users | RBACUserManage | RBACUserManage.vue | 需鉴权 | users:read | 用户管理 |
| /rbac/users/create | RBACUserCreate | RBACUserCreate.vue | 需鉴权 | users:create:write | 创建用户 |
| /rbac/users/:id/edit | RBACUserEdit | RBACUserEdit.vue | 需鉴权 | users:update:write | 编辑用户 |
| /rbac/users/:id/password | RBACUserPassword | RBACUserPassword.vue | 需鉴权 | users:change_password:write | 修改密码 |
| /rbac/users/:id/permissions | RBACUserPermissionCustomize | RBACUserPermissionCustomize.vue | 需鉴权 | skipPermCheck | 自定义权限 |
| /rbac/roles | RBACRoleManage | RBACRoleManage.vue | 需鉴权 | roles:read | 角色管理 |
| /rbac/permissions | RBACPermissionManage | RBACPermissionManage.vue | 需鉴权 | permissions:read | 权限管理 |
| /rbac/permissions/enum | RBACPermissionEnumManage | RBACPermissionEnumManage.vue | 需鉴权 | permissions:read | 权限字典管理 |
| /rbac/constraints | RBACConstraintManage | RBACConstraintManage.vue | 需鉴权 | constraints:read | 约束管理 |

### 路由守卫

```typescript
router.beforeEach(async (to) => {
  // 1. 公开页面直接放行
  // 2. 无 token → /login
  // 3. 获取用户信息 + 加载有效权限（/api/v2/me/permissions）
  // 4. 检查 to.meta.permKey，无权限 → /403
})
```

权限守卫核心逻辑：
- `to.meta.public` → 直接放行
- `to.meta.skipPermCheck` → 仅校验登录，不校验权限
- `to.meta.permKey` → 调用 `permStore.hasPermission(permKey)` 校验

## 权限体系

### 权限状态（stores/permission.ts）

```typescript
export const usePermissionStore = defineStore('permission', () => {
  const permissions = ref<Record<string, string>>({})  // { key: name }

  function hasPermission(key: string): boolean {
    return key in permissions.value
  }

  async function loadPermissions(userId?: string) {
    const { data } = await api.get<Record<string, string>>('/api/v2/me/permissions')
    permissions.value = data || {}
  }

  return { permissions, hasPermission, loadPermissions, clearPermissions }
})
```

### 权限指令（directives/permission.ts）

```typescript
const vPerm = {
  mounted(el: HTMLElement, binding: DirectiveBinding<string>) {
    const permStore = usePermissionStore()
    const key = binding.value
    if (!permStore.hasPermission(key)) {
      el.style.display = 'none'
    }
  },
}
```

使用方式：`<button v-perm="'users:create:write'">创建用户</button>`

### 权限表达式工具（utils/permExpression.ts）

支持复杂权限表达式的解析与求值：
- 基础权限：`dashboard:read`
- 组合表达式：`users:delete:write & !isBuiltInAdmin(user_id) & !isSelf(user_id)`
- 内置函数：`isAdmin()`、`isSelf()`、`isBuiltInAdmin()`、`hasSameRole()` 等

## 数据流

```
页面组件 (Pages)
    ├── 调用 api.get/post/delete (utils/api.ts)
    │       ├── 请求拦截器：注入 JWT token
    │       └── 响应拦截器：401 → 跳转登录页
    ├── 调用 Pinia Store (stores/*.ts)
    │       ├── user store：登录/用户信息/退出
    │       ├── permission store：权限加载与校验
    │       ├── notification store：通知管理
    │       └── tokenPlan store：模型配置 CRUD
    └── 本地状态管理 (ref/reactive/computed)
            ├── 列表数据 (ref<T[]>)
            ├── 加载状态 (ref<boolean>)
            ├── 筛选/分页 (computed)
            └── 表单数据 (reactive)
```

## 组件通讯模式

- **父 → 子**：Props（如 PageHeader 的 title/subtitle、StatCard 的 title/value）
- **子 → 父**：Emits（如 AppSidebar 的 toggle、Modal 的 update:visible）
- **跨层级**：Pinia Store（如 user token、permissions、notifications）
- **路由参数**：Vue Router（如 /rbac/users/:id/edit 依赖 userId）
- **DOM 控制**：自定义指令（v-perm 控制按钮显示/隐藏）

## 侧边栏菜单动态渲染

`AppSidebar.vue` 基于 `permissionStore.hasPermission(key)` 动态过滤菜单项：

```typescript
const menuItems = [
  { key: 'dashboard', label: '仪表盘', permKey: 'dashboard:read', path: '/' },
  { key: 'platforms', label: '平台管理', permKey: 'platforms:read', path: '/platforms' },
  // ...
]

const visibleMenus = computed(() =>
  menuItems.filter(item => permStore.hasPermission(item.permKey))
)
```

菜单分组可见性：任一子菜单可见则显示该分组。
