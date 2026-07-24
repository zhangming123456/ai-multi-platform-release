# 前端架构设计

## 目录结构

```
frontend/src/
├── App.vue                     # 根组件（router-view）
├── main.ts                     # 应用入口（Pinia + Router + Arco Design）
├── style.css                   # 全局样式（TailwindCSS + 自定义动画）
├── router/
│   └── index.ts                # 路由定义 + beforeEach 守卫
├── stores/
│   ├── user.ts                 # 登录/用户信息状态管理
│   └── tokenPlan.ts            # AI 模型配置状态管理
├── types/
│   └── index.ts                # 全局 TypeScript 类型定义
├── utils/
│   └── api.ts                  # Axios 实例（baseURL + 拦截器）
├── components/
│   ├── layout/
│   │   ├── AppLayout.vue       # 主布局（侧边栏 + 顶栏 + 内容区）
│   │   ├── AppHeader.vue       # 顶部导航（面包屑 + 搜索 + 通知）
│   │   ├── AppSidebar.vue      # 侧边导航（菜单 + 折叠/展开）
│   │   └── PageHeader.vue      # 页面标题组件（标题 + 副标题 + actions 插槽）
│   └── shared/
│       ├── Modal.vue           # 通用弹窗组件
│       ├── PlatformIcon.vue    # 平台图标组件（公众号/小红书/抖音/视频号）
│       ├── SegmentedControl.vue # 分段控制器组件
│       ├── StatCard.vue        # 统计卡片组件（动画 + 图标 + 趋势）
│       └── StatusBadge.vue     # 状态标签组件
└── pages/
    ├── Login.vue               # 登录页（毛玻璃卡片）
    ├── Dashboard.vue           # 仪表盘
    ├── Accounts.vue            # 账号管理
    ├── ContentCreate.vue       # AI 内容生成
    ├── ContentList.vue         # 内容列表
    ├── Publish.vue             # 发布管理
    ├── Templates.vue           # 模板中心
    ├── TokenPlan.vue           # 模型配置
    ├── RoleManage.vue          # 角色管理（管理员）
    ├── PermissionManage.vue    # 权限管理（管理员）
    └── ApiDocs.vue             # API 文档（iframe Swagger）
```

## 路由定义

| 路由 | 名称 | 组件 | 鉴权 | 说明 |
|------|------|------|------|------|
| /login | Login | Login.vue | 公开 | 管理员登录页 |
| / | Dashboard | Dashboard.vue | 需鉴权 | 仪表盘（数据概览） |
| /accounts | Accounts | Accounts.vue | 需鉴权 | 账号矩阵管理 |
| /content | ContentList | ContentList.vue | 需鉴权 | 内容列表 |
| /content/create | ContentCreate | ContentCreate.vue | 需鉴权 | AI 内容生成 |
| /publish | Publish | Publish.vue | 需鉴权 | 发布管理中心 |
| /templates | Templates | Templates.vue | 需鉴权 | 模板中心 |
| /settings/token-plan | TokenPlan | TokenPlan.vue | 需鉴权 | AI 模型配置 |
| /settings/roles | RoleManage | RoleManage.vue | 需鉴权 | 角色管理（管理员） |
| /settings/permissions | PermissionManage | PermissionManage.vue | 需鉴权 | 权限管理（管理员） |
| /developer/docs | ApiDocs | ApiDocs.vue | 需鉴权 | Swagger API 文档 |

### 路由守卫

- `router.beforeEach`：未登录用户访问需鉴权页面 → 重定向到 `/login`
- `router.beforeEach`：已登录用户访问 `/login` → 重定向到 `/`
- 鉴权依据：localStorage 中的 JWT token

## 数据流

```
页面组件 (Pages)
    ├── 调用 api.get/post/delete (utils/api.ts)
    │       ├── 请求拦截器：注入 JWT token
    │       └── 响应拦截器：401 → 跳转登录页
    ├── 调用 Pinia Store (stores/*.ts)
    │       ├── user store：登录/用户信息/退出
    │       └── tokenPlan store：模型配置 CRUD
    └── 本地状态管理 (ref/reactive/computed)
            ├── 列表数据 (ref<T[]>)
            ├── 加载状态 (ref<boolean>)
            ├── 筛选/分页 (computed)
            └── 表单数据 (reactive)
```

## 组件通讯模式

- **父 → 子**：Props（如 PageHeader 的 title/subtitle、StatCard 的 title/value/trend/icon/color）
- **子 → 父**：Emits（如 AppSidebar 的 toggle、Modal 的 update:visible）
- **跨层级**：Pinia Store（如 user token、tokenPlan 的 activePlan）
- **路由参数**：Vue Router（如 /content/create 依赖 tokenPlan.activePlan）
