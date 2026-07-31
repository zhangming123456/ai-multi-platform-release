# 前端架构设计

## 目录结构

```
frontend/src/
├── App.vue                     # 根组件（router-view）
├── main.ts                     # 应用入口（Pinia + Router + Arco Design + 权限指令注册）
├── style.css                   # 全局样式（TailwindCSS v4 + Apple 色板 + Arco 主题覆盖 + 动画 + 响应式）
├── env.d.ts                    # 环境变量类型声明
├── router/
│   └── index.ts                # 路由定义 + beforeEach 权限守卫
├── stores/
│   ├── user.ts                 # 登录/用户信息/退出
│   ├── permission.ts           # 权限状态管理（有效权限加载、hasPermission 表达式求值）
│   ├── notification.ts         # 通知消息状态管理（未读计数、列表、已读标记）
│   ├── region.ts               # 时区/区域切换状态（函数式 store，非 defineStore）
│   └── tokenPlan.ts            # AI 模型配置状态管理
├── types/
│   └── index.ts                # 全局 TypeScript 类型定义
├── utils/
│   ├── api.ts                  # Axios 实例（baseURL + JWT 拦截器 + 401 处理）
│   ├── permExpression.ts       # 权限表达式 DSL 解析器（词法分析 + 递归下降解析，316 行）
│   ├── rbac.ts                 # RBAC 工具函数（管理类型权限过滤：ADMIN_ONLY_RESOURCE_KEYS、isAdminTypePermission、filterAdminOnlyPermissions）
│   └── time.ts                 # 时间/时区工具（dayjs 插件 + 11 个时区选项 + formatDateTime/formatRelativeTime）
├── directives/
│   └── permission.ts           # v-perm 指令（按钮级权限控制）
├── components/
│   ├── layout/
│   │   ├── AppLayout.vue       # 主布局（侧边栏 + 顶栏 + 内容区）
│   │   ├── AppHeader.vue       # 顶部导航（面包屑 + 通知铃铛 + 用户菜单）
│   │   ├── AppSidebar.vue      # 侧边导航（动态菜单 + 折叠/展开）
│   │   ├── PageHeader.vue      # 页面标题组件（标题 + 副标题 + actions 插槽）
│   │   └── RegionSwitcher.vue  # 时区切换器组件
│   ├── rbac/
│   │   ├── PermissionModuleCard.vue    # 权限模块卡片（批量选择 + 半选状态）
│   │   ├── ResourcePermissionCard.vue  # 资源权限卡片（含权限描述展示）
│   │   └── RoleInheritanceTree.vue     # 角色继承关系树可视化
│   ├── NotificationBell.vue    # 通知铃铛组件
│   └── shared/
│       ├── Modal.vue           # 通用弹窗组件
│       ├── PlatformIcon.vue    # 平台图标组件（公众号/小红书/抖音/视频号）
│       ├── SegmentedControl.vue # 分段控制器组件
│       ├── StatCard.vue        # 统计卡片组件（动画 + 图标 + 趋势）
│       └── StatusBadge.vue     # 状态标签组件
└── pages/
    ├── Login.vue               # 登录页（毛玻璃卡片）
    ├── Dashboard.vue           # 仪表盘
    ├── Platforms.vue           # 平台管理
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
    ├── RBACPermissionEnumManage.vue     # 权限字典管理
    ├── RBACPermissionEnumEdit.vue       # 权限字典编辑页（新增/编辑复用）
    └── RBACConstraintManage.vue         # RBAC 约束管理
```

## 路由定义

所有需鉴权子路由均挂载在 `AppLayout.vue` 下。路由 `meta` 扩展字段：`sidebarType`（top/content/review/platforms/rbac/system，用于侧边栏分组）、`sidebarOrder`（组内排序）、`icon`（图标 key，对应 AppSidebar 的 iconRegistry）、`permKey`（权限 key 或表达式）、`skipPermCheck`、`public`、`title`。

| 路由 | 名称 | 组件 | 鉴权 | 权限 Key | 说明 |
|------|------|------|------|----------|------|
| /login | Login | Login.vue | 公开 | — | 登录页 |
| / | Dashboard | Dashboard.vue | 需鉴权 | dashboard:read | 仪表盘 |
| /403 | Forbidden | Forbidden.vue | 需鉴权 | skipPermCheck | 无权限提示 |
| /profile | Profile | Profile.vue | 需鉴权 | skipPermCheck | 个人中心 |
| /platforms | Platforms | Platforms.vue | 需鉴权 | platforms:read | 平台管理 |
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
| /rbac/users/:id/edit | RBACUserEdit | RBACUserEdit.vue | 需鉴权 | users:update:read \|\| isSelf(id) | 编辑用户 |
| /rbac/users/:id/password | RBACUserPassword | RBACUserPassword.vue | 需鉴权 | users:change_password:write \|\| isSelf(id) | 修改密码 |
| /rbac/users/:id/permissions | RBACUserPermissionCustomize | RBACUserPermissionCustomize.vue | 需鉴权 | users:custom_permissions:read \|\| isSelf(id) | 自定义权限 |
| /rbac/roles | RBACRoleManage | RBACRoleManage.vue | 需鉴权 | roles:read | 角色管理 |
| /rbac/permissions | RBACPermissionManage | RBACPermissionManage.vue | 需鉴权 | permissions:read | 权限管理 |
| /rbac/permissions/enum | RBACPermissionEnumManage | RBACPermissionEnumManage.vue | 需鉴权 | permissions:manage:read | 权限字典管理 |
| /rbac/permissions/enum/create | RBACPermissionEnumCreate | RBACPermissionEnumEdit.vue | 需鉴权 | permissions:manage:write | 新增权限字典 |
| /rbac/permissions/enum/edit/:resourceId | RBACPermissionEnumEdit | RBACPermissionEnumEdit.vue | 需鉴权 | permissions:manage:write | 编辑权限字典 |
| /rbac/constraints | RBACConstraintManage | RBACConstraintManage.vue | 需鉴权 | constraints:read | 约束管理 |

### 路由守卫

```typescript
router.beforeEach(async (to) => {
  // 1. 已登录访问 /login → 重定向首页
  // 2. public 页面直接放行
  // 3. 无 token → /login（带 redirect query）
  // 4. 拉取用户信息 + 加载有效权限（/v2/me/permissions，按 userId 缓存）
  // 5. hasPerm(to) 校验失败 → /403（带 from query）
})

function hasPerm(to): boolean {
  if (to.meta.skipPermCheck) return true
  const permKey = to.meta.permKey
  if (!permKey) return true
  // 将 query 和 params 合并为上下文，供表达式中的 isSelf(id) 等函数使用
  return permStore.hasPermission(permKey, {
    ...(to.query ?? {}),
    ...(to.params ?? {}),
  })
}
```

权限守卫核心逻辑：
- `to.meta.public` → 直接放行（如 /login）
- `to.meta.skipPermCheck` → 仅校验登录，不校验权限（如 /403、/profile）
- `to.meta.permKey` → 调用 `permStore.hasPermission(permKey, ctx)` 校验，`ctx` 由 `to.query` 与 `to.params` 合并而成，使表达式中的 `isSelf(id)` 等函数可读取路由参数

## 权限体系

### 权限状态（stores/permission.ts）

```typescript
import { isExpression, evaluatePermission } from '@/utils/permExpression'
import type { PermContext } from '@/utils/permExpression'

export const usePermissionStore = defineStore('permission', () => {
  const permissions = ref<Record<string, string>>({})  // { key: name }
  const lastPermissionsUserId = ref<string | null>(null)  // 按 userId 缓存权限

  // keyOrExpr 可以是简单权限 key，也可以是表达式字符串
  // ctx 可选，用于向表达式中的 isSelf(id) 等函数注入上下文
  function hasPermission(keyOrExpr: string, ctx?: PermContext): boolean {
    if (!isExpression(keyOrExpr)) {
      return keyOrExpr in permissions.value
    }
    return evaluatePermission(keyOrExpr, permissions.value, _mergeContext(ctx))
  }

  async function loadPermissions(userId?: string) {
    const { data } = await api.get<Record<string, string>>('/v2/me/permissions')
    permissions.value = data || {}
    lastPermissionsUserId.value = userId ?? null
  }

  return { permissions, lastPermissionsUserId, hasPermission, loadPermissions, clearPermissions }
})
```

上下文构建：
- `_buildDefaultContext()`：从 `useUserStore()` 读取当前用户 `{ id, role }`，构建默认 `PermContext`
- `_mergeContext(overrides?)`：将传入的 `overrides` 与默认上下文浅合并，未传则返回默认上下文
- 导出类型 `PermContext`（再导出自 `utils/permExpression`）

### 权限指令（directives/permission.ts）

支持两种绑定形式：
- 字符串形式：`v-perm="'users:read'"`
- 对象形式：`v-perm="{ key: 'users:read', ctx: { user_id: 'xxx' } }"`

```typescript
interface PermBinding {
  key: string
  ctx?: PermContext
}

// 通过 WeakMap 缓存被移除元素的位置（Comment 占位符），支持响应式恢复
const _permCache = new WeakMap<HTMLElement, {
  placeholder: Comment
  originalParent: Node
  originalNext: Node | null
}>()

const vPerm: ObjectDirective<HTMLElement, string | PermBinding> = {
  mounted: checkAndApply,   // 初次挂载时检查权限
  updated: checkAndApply,   // 依赖更新时重新检查，支持权限变化后恢复
  beforeUnmount(el) { _permCache.delete(el) },
}
```

`checkAndApply` 逻辑：
- 无权限且未缓存 → `removeEl`：在原位置插入 `Comment('v-perm')` 占位符并移除元素，缓存位置信息
- 有权限且已缓存 → `restoreEl`：根据占位符将元素恢复到原位置并移除占位符
- 通过 `mounted` + `updated` 双钩子实现权限变化时的响应式显隐

使用方式：
```vue
<button v-perm="'users:create:write'">创建用户</button>
<button v-perm="{ key: 'users:delete:write', ctx: { user_id: id } }">删除</button>
```

### 权限表达式工具（utils/permExpression.ts）

权限表达式 DSL 解析器（316 行），实现词法分析（Tokenizer）+ 递归下降解析（Parser）+ 求值（Evaluator）：

- `isExpression(source)`：判断字符串是否为表达式（含 `| & ( ) !` 字符）
- `evaluatePermission(expr, permissions, ctx)`：对表达式求值
- `PermContext` 类型：`{ currentUser, user_id, account, content, target_user, current_role_ids, target_role_ids }`

支持的语法：
- 基础权限：`dashboard:read`
- 与/或/非：`a:read & b:read`、`a:read || b:read`、`!isAdmin()`
- 括号分组：`(a:read || b:read) & !isSelf(id)`
- 内置函数：`isSelf()`、`isAdmin()`、`isSuperAdmin()`、`isBuiltInAdmin()`、`isOwnAccount()`、`isOwnContent()`、`hasSameRole()`

典型用例：`users:update:read || isSelf(id)` —— 有更新权限或是用户本人即可访问。

### RBAC 工具（utils/rbac.ts）

管理类型权限过滤工具，用于对非超管用户隐藏仅管理员可见的权限项：

- `ADMIN_ONLY_RESOURCE_KEYS`：`Set`，包含 `db`、`db_history`、`api_docs`、`permissions`、`roles`、`constraints` 等仅管理员可访问的资源 key
- `isAdminTypePermission(permKey)`：根据权限 key 首段判断是否为管理类型
- `filterAdminOnlyPermissions<T>(items)`：过滤掉 `readKey` 或首个 `writeKey` 属于管理类型的项

## 数据流

```
页面组件 (Pages)
    ├── 调用 api.get/post/delete (utils/api.ts)
    │       ├── 请求拦截器：注入 JWT token
    │       └── 响应拦截器：401 → 跳转登录页
    ├── 调用 Pinia Store (stores/*.ts)
    │       ├── user store：登录/用户信息/退出
    │       ├── permission store：权限加载与校验（含表达式求值）
    │       ├── notification store：通知消息管理
    │       ├── region store：时区/区域切换（函数式）
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

`AppSidebar.vue` 不再硬编码菜单项，而是通过 `router.getRoutes()` 动态读取路由 `meta` 生成菜单：

```typescript
// 1. 读取所有带 sidebarType 的路由
const sidebarRoutes = router.getRoutes().filter(r => r.meta.sidebarType && r.meta.title)

// 2. 按 sidebarType 分组，组内按 sidebarOrder 排序
// 3. 通过 SIDEBAR_GROUPS 配置决定是否包装为分组（wrapGroup）
// 4. 每个路由项权限过滤调用 permStore.hasPermission(permKey, ctx)
```

`SIDEBAR_GROUPS` 配置（按 order 排序）：

| key | name | order | wrapGroup | icon |
|-----|------|-------|-----------|------|
| top | （无名称，平铺） | 0 | false | — |
| content | 内容管理 | 1 | true | file |
| review | 审核管理 | 2 | true | check |
| platforms | 平台管理 | 3 | false | apps |
| rbac | 权限管理 | 4 | true | safe |
| system | 系统管理 | 5 | true | tool |

内置 `iconRegistry` 图标注册表，将路由 `meta.icon` 字符串（如 `home`/`file`/`send`/`apps`/`settings`/`code`/`safe`/`storage`/`tool`/`check`/`user`/`edit`）映射到 Arco Design 图标组件。

菜单项权限过滤：`permStore.hasPermission(permKey, route.meta.ctx)` —— 与路由守卫一致，支持表达式。分组可见性：任一子菜单可见则显示该分组。

## 技术栈

| 类别 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 构建工具 | Vite | 8.1.0 | 开发服务器 + 构建 |
| 框架 | Vue | ^3.5.39 | Composition API + `<script setup>` |
| 语言 | TypeScript | ~6.0.2 | 类型检查（vue-tsc ^3.3.5） |
| 路由 | Vue Router | ^4.6.4 | History 模式 |
| 状态 | Pinia | ^4.0.2 | setup store 风格 |
| UI 库 | Arco Design Vue | ^2.58.0 | 字节跳动组件库 |
| CSS | Tailwind CSS | ^4.3.3 | v4 Vite 插件模式（`@tailwindcss/vite`），无 `tailwind.config.js` 配置文件，通过 `@import 'tailwindcss'` 引入 |
| HTTP | axios | ^1.18.1 | 请求拦截 + 401 处理 |
| 时间 | dayjs | ^1.11.21 | utc/timezone/relativeTime 插件 + zh-cn 本地化 |
| 工具 | lodash-es | ^4.18.1 | 函数式工具库 |
| 图标 | lucide-vue-next | ^1.0.0 | 图标组件库 |
| API 文档 | swagger-ui-dist | ^5.32.11 | iframe 嵌入 Swagger UI |

## 全局样式（src/style.css）

`src/style.css` 是全局样式入口，包含以下几部分：

- **Tailwind CSS v4 引入**：`@import 'tailwindcss'`（Vite 插件模式，无独立配置文件）
- **Apple 设计系统色板 CSS 变量**（`:root`）：
  - 主色：`--apple-blue` `#007aff`、`--apple-green` `#34c759`、`--apple-indigo` `#5856d6`、`--apple-orange` `#ff9500`、`--apple-red` `#ff3b30`、`--apple-teal` `#5ac8fa`、`--apple-purple` `#af52de`、`--apple-pink` `#ff2d55`、`--apple-yellow` `#ffcc00`
  - 文本/背景：`--apple-bg` `#f5f5f7`、`--apple-card` `#ffffff`、`--apple-text-primary` `#1d1d1f`、`--apple-text-secondary` `#86868b`、`--apple-text-tertiary` `#aeaeb2`
  - 导航配色：`--apple-nav-dashboard`/`accounts`/`content`/`publish`/`templates`
- **Arco Design 主题覆盖**：通过 `--arco-theme-primary`/`success`/`warning`/`danger`/`info` 变量将 Arco 主色对齐 Apple 色板；并覆盖 `.arco-layout-sider`、`.arco-menu-*`、`.arco-card`、`.arco-table`、`.arco-modal*` 等组件样式
- **动画关键帧**：
  - `fade-up`：上浮淡入
  - `fade-in`：纯淡入
  - `scale-in`：缩放淡入
  - `bar-grow`：进度条/柱状图增长
- **四档响应式断点**：
  - `max-width: 768px`：平板布局调整（栅格列宽）
  - `min-width: 769px`：桌面端栅格
  - `max-width: 480px`：手机端组件尺寸缩减（按钮/输入/卡片/模态/表单/标签等）
  - `max-width: 248px`：超窄屏（侧边栏折叠态）精细化字号/间距调整（覆盖几乎所有 Arco 组件）
- **基础样式**：`-apple-system` 字体栈、`box-sizing: border-box`、自定义滚动条、`::selection` 选区配色
