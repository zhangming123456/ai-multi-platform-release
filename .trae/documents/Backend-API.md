# 后端 API 接口文档

## 路由模块总览

| 路由前缀 | 模块文件 | 说明 |
|----------|----------|------|
| /api/auth | routers/auth.py | 认证（登录、注册、用户信息） |
| /api/accounts | routers/accounts.py | 账号管理 CRUD + 状态检查 |
| /api/contents | routers/contents.py | 内容管理 CRUD + AI 生成 |
| /api/dashboard | routers/dashboard.py | 仪表盘聚合统计 |
| /api/model-configs | routers/model_configs.py | AI 模型配置管理 |
| /api/models | routers/models.py | 可用模型列表查询 |
| /api/publish | routers/publish.py | 发布任务管理 |
| /api/templates | routers/templates.py | 模板管理 |
| /api/roles | routers/roles.py | 角色管理（内置+自定义） |
| /api/permissions | routers/permissions.py | 角色权限与用户权限管理 |
| /api/users | routers/users.py | 用户管理（管理员功能） |
| /api/notifications | routers/notifications.py | 通知消息管理 |
| /api/reviews | routers/reviews.py | 内容审核管理 |
| /api/db-changes | routers/db_changes.py | 数据库变更请求管理 |

---

## 认证相关

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | /api/auth/login | 用户登录 | 否 |
| POST | /api/auth/register | 用户注册 | 否 |
| GET | /api/auth/me | 获取当前用户 | 是 |

```typescript
interface LoginRequest {
  email: string
  password: string
}

interface LoginResponse {
  access_token: string
  token_type: string
  user: UserInfo
}

interface UserInfo {
  id: string
  email: string
  username: string
  nickname: string
  role: 'admin' | 'manager' | 'operator' | 'reviewer'
  avatar_url?: string | null
  created_at: string
}
```

---

## 仪表盘

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/dashboard/stats | 获取仪表盘聚合统计 | 是 |

```typescript
interface DashboardStats {
  total_accounts: number
  today_published: number
  pending_tasks: number
  ai_generated_count: number
  platform_stats: PlatformStat[]
  recent_publishes: RecentPublish[]
}

interface PlatformStat {
  platform: Platform
  name: string
  accounts: number
  active: number
  articles: number
}

interface RecentPublish {
  id: string
  title: string
  platform: Platform
  account: string
  status: PublishStatus
  time: string
}
```

---

## 账号管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/accounts/ | 获取账号列表 | 是 |
| POST | /api/accounts/ | 添加新账号 | 是 |
| DELETE | /api/accounts/{id} | 删除账号 | 是 |
| POST | /api/accounts/{id}/check | 检查账号状态 | 是 |

```typescript
type Platform = 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
type AccountStatus = 'active' | 'inactive' | 'error'

interface Account {
  id: string
  user_id: string
  platform: Platform
  nickname: string
  avatar_url: string | null
  status: AccountStatus
  cookie_data: string | null
  access_token: string | null
  token_expires_at: string | null
  last_check_at: string | null
  error_message: string | null
  created_at: string
  updated_at: string
}

interface CreateAccountRequest {
  platform: string
  nickname: string
  cookie_data?: string
  access_token?: string
}
```

---

## 内容管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/contents/ | 获取内容列表 | 是 |
| POST | /api/contents/ai-generate | AI 生成内容 | 是 |
| DELETE | /api/contents/{id} | 删除内容 | 是 |

```typescript
interface Content {
  id: string
  user_id: string
  title: string
  body: string
  platform: string
  status: 'draft' | 'ready' | 'published'
  media_urls: string[]
  ai_generated: boolean
  original_content_id?: string
  created_at: string
  updated_at: string
}

interface AIGenerateRequest {
  topic: string
  platform: string
  count: number
  plan_id: string
  keywords?: string[]
}

interface AIGenerateResponse {
  variants: Array<{
    title: string
    body: string
    hashtags: string[]
    suggested_image_ratio: string
  }>
}
```

---

## 发布管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/publish/tasks | 获取发布任务列表 | 是 |
| POST | /api/publish/tasks | 创建发布任务 | 是 |
| POST | /api/publish/tasks/{id}/retry | 重试失败任务 | 是 |

```typescript
type PublishTaskStatus = 'pending' | 'publishing' | 'published' | 'failed'

interface PublishTask {
  id: string
  content_id: string
  account_id: string
  status: PublishTaskStatus
  scheduled_at?: string | null
  published_at?: string | null
  error_message?: string | null
  retry_count: number
  created_at: string
  updated_at: string
}

interface CreatePublishTaskRequest {
  content_id: string
  account_id: string
  scheduled_at?: string
}
```

---

## 模板管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/templates/ | 获取模板列表 | 是 |

```typescript
interface Template {
  id: string
  name: string
  platform: string
  thumbnail_url?: string | null
  config?: string | null
  created_at: string
  updated_at: string
}
```

---

## 模型配置

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/model-configs | 获取配置列表 | 是 |
| POST | /api/model-configs | 创建模型配置 | 是 |
| PUT | /api/model-configs/{id} | 更新模型配置 | 是 |
| DELETE | /api/model-configs/{id} | 删除模型配置 | 是 |

---

## 可用模型

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/models | 获取可用模型列表 | 是 |

---

## 角色管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/roles/ | 获取角色列表 | 是（管理员） |
| POST | /api/roles/custom | 创建自定义角色 | 是（管理员） |
| PUT | /api/roles/custom/{name} | 更新自定义角色 | 是（管理员） |
| DELETE | /api/roles/custom/{name} | 删除自定义角色 | 是（管理员） |

```typescript
interface RoleDef {
  name: string
  display_name: string
  description?: string
  is_builtin: boolean
  is_super_admin: boolean
  role_type: 'admin' | 'other'
}

interface CustomRole extends RoleDef {
  id: string
  created_at: string
}
```

内置角色定义（按排序）：
1. **超级管理员 (admin)** — 系统最高权限，ID=1，不可编辑，`role_type: "admin"`
2. **管理员 (manager)** — 管理系统配置，`role_type: "admin"`
3. **运营者 (operator)** — 日常内容运营，`role_type: "other"`
4. **审核员 (reviewer)** — 内容审核，`role_type: "other"`

---

## 权限管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/permissions/role/{role} | 获取角色权限配置 | 是（管理员） |
| PUT | /api/permissions/role/{role} | 更新角色权限 | 是（管理员） |
| GET | /api/permissions/user/{user_id} | 获取用户自定义权限 | 是（管理员/本人） |
| PUT | /api/permissions/user/{user_id} | 更新用户自定义权限 | 是（管理员） |
| GET | /api/permissions/user/{user_id}/effective | 获取用户有效权限 | 是（管理员） |

权限键格式：`{资源}:{操作}`，如 `content:create`、`database`、`db:execute`

角色类型与数据库权限约束：
- `role_type: "admin"` — 可配置数据库相关权限（`database`、`db:execute`、`db:history:read`）
- `role_type: "other"` — 不支持数据库相关权限配置

---

## 用户管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/users/ | 用户列表 | 是（管理员） |
| GET | /api/users/{user_id} | 用户详情 | 是（管理员） |
| PUT | /api/users/{user_id} | 更新用户信息 | 是（管理员） |

---

## 通知管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/notifications/ | 获取通知列表 | 是 |
| PUT | /api/notifications/{id}/read | 标记已读 | 是 |

```typescript
interface NotificationResponse {
  id: string
  type: string
  title: string
  content?: string
  related_id?: string
  is_read: boolean
  created_at: string
}
```

---

## 内容审核

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/reviews/ | 获取待审核内容 | 是（审核员） |
| PUT | /api/reviews/{content_id} | 审核内容 | 是（审核员） |

---

## 数据库变更请求

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/db-changes/ | 获取变更请求列表 | 是（管理员） |
| POST | /api/db-changes/ | 创建变更请求 | 是（管理员） |
| PUT | /api/db-changes/{change_id} | 审批/拒绝变更 | 是（管理员） |
