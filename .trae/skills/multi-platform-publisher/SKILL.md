---
name: "multi-platform-publisher"
description: "多平台内容矩阵管理与自动化发布系统。当需要开发、修改、审查该项目的任何代码（前端/后端/数据库/API）时，优先调用此 Skill 以获取项目上下文和开发规范。"
---

# 多平台矩阵管理与自动化发布系统

一站式内容创作与分发平台，支持小红书、抖音、微信视频号、微信公众号等主流社交媒体账号的统一管理、AI 驱动的内容变体生成以及多平台一键发布/定时发布。

---

## 1. 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 前端 | Vue 3 + TypeScript + Vite 6 + TailwindCSS | `npm create vite@latest . --template vue-ts` |
| UI 组件库 | Arco Design Vue + 自定义组件 | StatCard、StatusBadge、PlatformIcon、Modal、SegmentedControl |
| 状态管理 | Pinia | user store（登录/用户信息）、tokenPlan store（模型配置） |
| HTTP 客户端 | Axios | 请求拦截器注入 JWT、401 响应拦截器自动跳转登录页 |
| 后端 | Python 3.11+ / FastAPI | 异步 RESTful API |
| 数据库 | SQLite（开发）→ PostgreSQL（生产） | 通过 SQLAlchemy 2.0 异步模式 |
| ORM | SQLAlchemy 2.0（异步） | `mapped_column` + `Mapped` 类型注解 |
| 任务队列 | Celery + Redis | 发布任务调度 |
| 自动化 | Playwright | 浏览器自动化（小红书/抖音/视频号发布） |
| AI 接入 | OpenAI 兼容 API | 支持 DeepSeek / OpenAI / Moonshot / 智谱 AI / 自定义 |
| 容器化 | Docker + Docker Compose | 4 服务：frontend、backend、redis、celery-worker |

---

## 2. 数据库 ID 规范（重要）

**所有表的主键 ID 必须使用 UUID 字符串（`VARCHAR(36)`），唯一例外：超级管理员用户 ID 固定为字符串 `"1"`。**

- SQLAlchemy 模型定义：`id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))`
- Pydantic Schema 中所有 ID 字段类型为 `str`
- 前端 TypeScript 接口中所有 ID 字段类型为 `string`
- 路由参数中所有 ID 参数类型为 `str`
- 外键字段使用 `String(36)` 类型

---

## 3. 项目目录结构

```
ai-multi-platform-release/
├── backend/
│   ├── app/
│   │   ├── main.py                # FastAPI 应用入口 + 种子数据
│   │   ├── celery_app.py          # Celery 任务队列配置
│   │   ├── core/
│   │   │   ├── auth.py            # JWT 认证工具
│   │   │   ├── deps.py            # 依赖项（get_current_user、get_admin_user、PermissionChecker）
│   │   │   └── database.py        # 数据库引擎 + SessionLocal
│   │   ├── models/                # SQLAlchemy 数据模型
│   │   │   ├── __init__.py
│   │   │   ├── user.py            # 用户模型
│   │   │   ├── account.py         # 平台账号模型
│   │   │   ├── content.py         # 内容模型
│   │   │   ├── publish_task.py    # 发布任务模型
│   │   │   ├── template.py        # 模板模型
│   │   │   ├── model_config.py    # AI 模型配置模型
│   │   │   ├── custom_role.py     # 自定义角色模型
│   │   │   ├── role_permission.py # 角色权限模型
│   │   │   ├── user_permission.py # 用户自定义权限模型
│   │   │   ├── notification.py    # 通知模型
│   │   │   ├── ai_generation.py   # AI 生成记录模型
│   │   │   ├── sql_history.py     # SQL 执行历史模型
│   │   │   └── sql_change_request.py # SQL 变更请求模型
│   │   ├── routers/               # API 路由
│   │   │   ├── auth.py            # /api/auth — 登录、注册、用户信息
│   │   │   ├── accounts.py        # /api/accounts — 账号 CRUD + 状态检查
│   │   │   ├── contents.py        # /api/contents — 内容管理 + AI 生成
│   │   │   ├── dashboard.py       # /api/dashboard — 仪表盘统计
│   │   │   ├── model_configs.py   # /api/model-configs — AI 模型配置
│   │   │   ├── models.py          # /api/models — 可用模型列表
│   │   │   ├── publish.py         # /api/publish — 发布任务
│   │   │   ├── templates.py       # /api/templates — 模板管理
│   │   │   ├── roles.py           # /api/roles — 角色管理
│   │   │   ├── permissions.py     # /api/permissions — 权限管理
│   │   │   ├── users.py           # /api/users — 用户管理
│   │   │   ├── notifications.py   # /api/notifications — 通知管理
│   │   │   ├── reviews.py         # /api/reviews — 内容审核
│   │   │   └── db_changes.py      # /api/db-changes — 数据库变更请求
│   │   └── schemas/               # Pydantic Schema（请求/响应模型）
│   │       ├── auth.py
│   │       ├── account.py
│   │       ├── content.py
│   │       ├── publish.py
│   │       ├── template.py
│   │       ├── review.py
│   │       └── notification.py
│   ├── requirements.txt
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── App.vue                # 根组件
│   │   ├── main.ts                # 应用入口（Pinia + Router + Arco Design）
│   │   ├── router/index.ts        # 路由定义 + beforeEach 鉴权守卫
│   │   ├── stores/
│   │   │   ├── user.ts            # 登录/用户信息状态管理
│   │   │   ├── tokenPlan.ts       # AI 模型配置状态管理
│   │   │   └── notification.ts    # 通知状态管理
│   │   ├── types/index.ts         # 全局 TypeScript 类型定义（所有 ID 字段为 string）
│   │   ├── utils/api.ts           # Axios 实例（baseURL + 拦截器）
│   │   ├── components/
│   │   │   ├── layout/            # AppLayout、AppHeader、AppSidebar、PageHeader
│   │   │   └── shared/            # Modal、PlatformIcon、SegmentedControl、StatCard、StatusBadge
│   │   └── pages/                 # 页面组件
│   │       ├── Login.vue          # 登录页
│   │       ├── Dashboard.vue      # 仪表盘
│   │       ├── Accounts.vue       # 账号管理
│   │       ├── ContentCreate.vue  # AI 内容生成
│   │       ├── ContentList.vue    # 内容列表
│   │       ├── Publish.vue        # 发布管理
│   │       ├── Templates.vue      # 模板中心
│   │       ├── TokenPlan.vue      # 模型配置
│   │       ├── RoleManage.vue     # 角色管理（管理员）
│   │       ├── PermissionManage.vue # 权限管理（管理员）
│   │       └── ApiDocs.vue        # API 文档（iframe Swagger）
│   ├── package.json
│   └── Dockerfile
├── .trae/
│   ├── documents/
│   │   ├── PRD.md                 # 产品需求文档
│   │   └── Technical-Architecture.md # 技术架构文档
│   └── skills/
│       └── multi-platform-publisher/SKILL.md # 本文件
└── docker-compose.yml
```

---

## 4. 用户角色体系

| 角色 | 角色名 | ID | role_type | 说明 |
|------|--------|-----|-----------|------|
| 超级管理员 | admin | `"1"`（固定） | admin | 系统最高权限，不可编辑，拥有所有权限 |
| 管理员 | manager | UUID | admin | 管理系统配置、用户权限，可编辑 |
| 运营者 | operator | UUID | other | 日常内容运营，不可配置数据库权限 |
| 审核员 | reviewer | UUID | other | 内容审核，不可配置数据库权限 |
| 自定义角色 | 自定义 | UUID | admin/other | 由管理员创建，可选择管理类型或普通类型 |

角色排序规则：超级管理员 → 管理员 → 运营者 → 审核员 → 其他自定义角色

---

## 5. 权限系统

### 5.1 权限键格式

`{资源}:{操作}`，如 `content:create`、`database`、`db:execute`、`db:history:read`

### 5.2 权限层级

1. **角色权限**（role_permissions 表）：角色默认拥有的权限集合
2. **用户自定义权限**（user_permissions 表）：针对特定用户的额外权限配置

权限继承机制：用户无自定义权限时继承角色默认权限；有自定义权限时以自定义权限为准。

### 5.3 角色类型权限约束

- `role_type: "admin"` — 可配置数据库相关权限（`database`、`db:execute`、`db:history:read`）
- `role_type: "other"` — 不支持数据库相关权限配置

### 5.4 超级管理员特殊规则

- 用户 ID 固定为 `"1"`
- 角色不可编辑，权限不可修改
- 始终拥有所有系统权限
- 前端角色列表中标识为"仅查看"

---

## 6. 数据模型概要

### 6.1 核心表

| 表名 | 说明 | 关键外键 |
|------|------|----------|
| users | 用户 | - |
| accounts | 平台账号 | user_id → users.id |
| contents | 内容 | user_id → users.id, original_content_id → contents.id |
| publish_tasks | 发布任务 | content_id → contents.id, account_id → accounts.id |
| templates | 模板 | - |
| model_configs | AI 模型配置 | user_id → users.id |
| custom_roles | 自定义角色 | - |
| role_permissions | 角色权限 | role → custom_roles.name |
| user_permissions | 用户权限 | user_id → users.id, granted_by → users.id |
| notifications | 通知 | user_id → users.id |
| ai_generations | AI 生成记录 | user_id → users.id |
| sql_histories | SQL 执行历史 | user_id → users.id |
| sql_change_requests | SQL 变更请求 | user_id → users.id, reviewed_by → users.id |

### 6.2 所有主键必须使用 UUID

见"数据库 ID 规范"章节。

---

## 7. API 路由总览

| 路由前缀 | 模块 | 说明 |
|----------|------|------|
| /api/auth | routers/auth.py | 登录、注册、用户信息 |
| /api/accounts | routers/accounts.py | 账号 CRUD + 状态检查 |
| /api/contents | routers/contents.py | 内容管理 + AI 生成 |
| /api/dashboard | routers/dashboard.py | 仪表盘统计 |
| /api/model-configs | routers/model_configs.py | AI 模型配置 |
| /api/models | routers/models.py | 可用模型列表 |
| /api/publish | routers/publish.py | 发布任务管理 |
| /api/templates | routers/templates.py | 模板管理 |
| /api/roles | routers/roles.py | 角色管理 |
| /api/permissions | routers/permissions.py | 权限管理 |
| /api/users | routers/users.py | 用户管理 |
| /api/notifications | routers/notifications.py | 通知管理 |
| /api/reviews | routers/reviews.py | 内容审核 |
| /api/db-changes | routers/db_changes.py | 数据库变更请求 |

---

## 8. 前端路由

| 路由 | 页面 | 组件 | 权限 |
|------|------|------|------|
| /login | 登录页 | Login.vue | 公开 |
| / | 仪表盘 | Dashboard.vue | 鉴权 |
| /accounts | 账号管理 | Accounts.vue | 鉴权 |
| /content | 内容列表 | ContentList.vue | 鉴权 |
| /content/create | AI 内容生成 | ContentCreate.vue | 鉴权 |
| /publish | 发布管理 | Publish.vue | 鉴权 |
| /templates | 模板中心 | Templates.vue | 鉴权 |
| /settings/token-plan | 模型配置 | TokenPlan.vue | 鉴权 |
| /settings/roles | 角色管理 | RoleManage.vue | 鉴权（管理员） |
| /settings/permissions | 权限管理 | PermissionManage.vue | 鉴权（管理员） |
| /developer/docs | API 文档 | ApiDocs.vue | 鉴权 |

路由守卫：
- 未登录访问需鉴权页面 → 重定向 /login
- 已登录访问 /login → 重定向 /
- 鉴权依据：localStorage 中的 JWT token
- 侧边栏菜单根据角色和权限动态显示

---

## 9. 开发规范

### 9.1 代码风格

- 后端：遵循 FastAPI 最佳实践，使用 Pydantic v2，异步 SQLAlchemy
- 前端：Vue 3 Composition API，TypeScript 严格模式，Arco Design Vue 组件库
- 所有文件不含注释（若不被明确要求）——保持代码简洁

### 9.2 新增模型时

1. 在 `backend/app/models/` 创建 SQLAlchemy 模型，ID 使用 UUID 字符串
2. 在 `backend/app/schemas/` 创建 Pydantic Schema，ID 字段类型为 `str`
3. 在 `backend/app/routers/` 创建 API 路由
4. 在 `frontend/src/types/index.ts` 添加 TypeScript 接口，ID 字段类型为 `string`
5. 在 `frontend/src/pages/` 创建页面组件

### 9.3 权限控制

- 后端 API 使用 `get_current_user` 或 `get_admin_user` 依赖注入
- 细粒度权限使用 `require_permission(perm_key)` 装饰器
- 前端路由使用 `beforeEach` 守卫 + Pinia user store 的角色信息

### 9.4 数据库迁移

- 使用 Alembic 进行数据库迁移管理
- 开发阶段使用 `drop_all` + `create_all`（每次重启自动重建）
- 生产环境使用 Alembic 升级脚本

---

## 10. 设计风格

- 现代 Apple 风格 SaaS 后台管理界面，毛玻璃效果
- 主色调：iOS 系统蓝 (#007AFF)，搭配深紫、翠绿、橙色、天蓝
- 背景色：浅灰 (#F5F5F7)，白色毛玻璃面板
- 字体：系统字体栈 (-apple-system, BlinkMacSystemFont, "PingFang SC", "Helvetica Neue")
- 侧边栏：白色毛玻璃 + 细微边框，支持折叠/展开
- 响应式设计：桌面优先 → 平板 → 移动端
