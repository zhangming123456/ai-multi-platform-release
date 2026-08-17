# 后端 API 接口文档

## 路由模块总览

| 路由前缀                   | 模块文件                         | 说明                         |
| -------------------------- | -------------------------------- | ---------------------------- |
| /api/auth                  | routers/auth.py                  | 认证（登录、注册、用户信息） |
| /api/accounts              | routers/accounts.py              | 账号管理 CRUD + 状态检查     |
| /api/contents              | routers/contents.py              | 内容管理 CRUD + AI 生成      |
| /api/dashboard             | routers/dashboard.py             | 仪表盘聚合统计               |
| /api/model-configs         | routers/model_configs.py         | AI 模型配置管理              |
| /api/models                | routers/models.py                | 可用模型列表查询             |
| /api/publish               | routers/publish.py               | 发布任务管理                 |
| /api/templates             | routers/templates.py             | 模板管理                     |
| /api/notifications         | routers/notifications.py         | 通知消息管理                 |
| /api/reviews               | routers/reviews.py               | 内容审核管理                 |
| /api/db-changes            | routers/db_changes.py            | 数据库变更请求管理           |
| /api/user-creation-reviews | routers/user_creation_reviews.py | 用户创建审核                 |
| /api/db                    | routers/db.py                    | SQL 执行与历史               |
| **/api/v2**                | **RBAC3 中台**                   | **权限中台 v2 API**          |
| /api/v2                    | routers/rbac_users.py            | RBAC3 用户管理               |
| /api/v2                    | routers/rbac_roles.py            | RBAC3 角色管理               |
| /api/v2                    | routers/rbac_permissions.py      | RBAC3 资源与权限管理         |
| /api/v2                    | routers/rbac_constraints.py      | RBAC3 约束管理               |

---

## 认证相关

| 方法 | 路径               | 说明         | 鉴权 |
| ---- | ------------------ | ------------ | ---- |
| POST | /api/auth/login    | 用户登录     | 否   |
| POST | /api/auth/register | 用户注册     | 否   |
| GET  | /api/auth/me       | 获取当前用户 | 是   |

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

| 方法 | 路径                 | 说明               | 鉴权 |
| ---- | -------------------- | ------------------ | ---- |
| GET  | /api/dashboard/stats | 获取仪表盘聚合统计 | 是   |

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

| 方法   | 路径                     | 说明         | 鉴权 |
| ------ | ------------------------ | ------------ | ---- |
| GET    | /api/accounts/           | 获取账号列表 | 是   |
| POST   | /api/accounts/           | 添加新账号   | 是   |
| DELETE | /api/accounts/{id}       | 删除账号     | 是   |
| POST   | /api/accounts/{id}/check | 检查账号状态 | 是   |

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

| 方法   | 路径                      | 说明         | 鉴权 |
| ------ | ------------------------- | ------------ | ---- |
| GET    | /api/contents/            | 获取内容列表 | 是   |
| POST   | /api/contents/ai-generate | AI 生成内容  | 是   |
| DELETE | /api/contents/{id}        | 删除内容     | 是   |

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

| 方法 | 路径                          | 说明             | 鉴权 |
| ---- | ----------------------------- | ---------------- | ---- |
| GET  | /api/publish/tasks            | 获取发布任务列表 | 是   |
| POST | /api/publish/tasks            | 创建发布任务     | 是   |
| POST | /api/publish/tasks/{id}/retry | 重试失败任务     | 是   |

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

| 方法 | 路径            | 说明         | 鉴权 |
| ---- | --------------- | ------------ | ---- |
| GET  | /api/templates/ | 获取模板列表 | 是   |

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

| 方法   | 路径                    | 说明         | 鉴权 |
| ------ | ----------------------- | ------------ | ---- |
| GET    | /api/model-configs      | 获取配置列表 | 是   |
| POST   | /api/model-configs      | 创建模型配置 | 是   |
| PUT    | /api/model-configs/{id} | 更新模型配置 | 是   |
| DELETE | /api/model-configs/{id} | 删除模型配置 | 是   |

---

## 可用模型

| 方法 | 路径        | 说明             | 鉴权 |
| ---- | ----------- | ---------------- | ---- |
| GET  | /api/models | 获取可用模型列表 | 是   |

---

## 角色管理

| 方法   | 路径                     | 说明           | 鉴权         |
| ------ | ------------------------ | -------------- | ------------ |
| GET    | /api/roles/              | 获取角色列表   | 是（管理员） |
| POST   | /api/roles/custom        | 创建自定义角色 | 是（管理员） |
| PUT    | /api/roles/custom/{name} | 更新自定义角色 | 是（管理员） |
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

| 方法 | 路径                                      | 说明               | 鉴权              |
| ---- | ----------------------------------------- | ------------------ | ----------------- |
| GET  | /api/permissions/role/{role}              | 获取角色权限配置   | 是（管理员）      |
| PUT  | /api/permissions/role/{role}              | 更新角色权限       | 是（管理员）      |
| GET  | /api/permissions/user/{user_id}           | 获取用户自定义权限 | 是（管理员/本人） |
| PUT  | /api/permissions/user/{user_id}           | 更新用户自定义权限 | 是（管理员）      |
| GET  | /api/permissions/user/{user_id}/effective | 获取用户有效权限   | 是（管理员）      |

权限键格式：`{资源}:{操作}`，如 `content:create`、`database`、`db:execute`

角色类型与数据库权限约束：

- `role_type: "admin"` — 可配置数据库相关权限（`database`、`db:execute`、`db:history:read`）
- `role_type: "other"` — 不支持数据库相关权限配置

---

## 用户管理

| 方法 | 路径                 | 说明         | 鉴权         |
| ---- | -------------------- | ------------ | ------------ |
| GET  | /api/users/          | 用户列表     | 是（管理员） |
| GET  | /api/users/{user_id} | 用户详情     | 是（管理员） |
| PUT  | /api/users/{user_id} | 更新用户信息 | 是（管理员） |

---

## 通知管理

| 方法 | 路径                         | 说明         | 鉴权 |
| ---- | ---------------------------- | ------------ | ---- |
| GET  | /api/notifications/          | 获取通知列表 | 是   |
| PUT  | /api/notifications/{id}/read | 标记已读     | 是   |

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

| 方法 | 路径                      | 说明           | 鉴权         |
| ---- | ------------------------- | -------------- | ------------ |
| GET  | /api/reviews/             | 获取待审核内容 | 是（审核员） |
| PUT  | /api/reviews/{content_id} | 审核内容       | 是（审核员） |

---

## 数据库变更请求

| 方法 | 路径                        | 说明             | 鉴权         |
| ---- | --------------------------- | ---------------- | ------------ |
| GET  | /api/db-changes/            | 获取变更请求列表 | 是（管理员） |
| POST | /api/db-changes/            | 创建变更请求     | 是（管理员） |
| PUT  | /api/db-changes/{change_id} | 审批/拒绝变更    | 是（管理员） |

---

## RBAC3 权限中台（/api/v2）

### 用户管理

| 方法   | 路径                                    | 说明                            | 鉴权                                 |
| ------ | --------------------------------------- | ------------------------------- | ------------------------------------ |
| GET    | /api/v2/users                           | 用户列表（含角色分配）          | 是（users:read）                     |
| POST   | /api/v2/users                           | 创建用户                        | 是（users:create:write）             |
| GET    | /api/v2/users/{id}                      | 用户详情                        | 是（users:read）                     |
| PUT    | /api/v2/users/{id}                      | 更新用户                        | 是（users:update:write）             |
| DELETE | /api/v2/users/{id}                      | 删除用户                        | 是（users:delete:write）             |
| PUT    | /api/v2/users/{id}/password             | 修改密码                        | 是（users:change_password:write）    |
| GET    | /api/v2/users/{id}/password-status      | 检测是否默认密码                | 是（users:read）                     |
| PUT    | /api/v2/users/{id}/roles                | 分配/替换角色                   | 是（users:update:write）             |
| GET    | /api/v2/users/{id}/permissions          | 用户有效权限                    | 是                                   |
| GET    | /api/v2/users/{id}/permission-overrides | 用户权限覆盖列表                | 是                                   |
| PUT    | /api/v2/users/{id}/permission-overrides | 更新权限覆盖（全量替换）        | 是（users:custom_permissions:write） |
| GET    | /api/v2/me/permissions                  | 当前用户有效权限（{key: name}） | 是                                   |

> **创建用户审核流程**：`POST /api/v2/users` 由非管理员（manager/admin 之外的角色）发起时，不会直接创建账号，而是自动写入 `UserCreationRequest` 提交审核，接口返回 `202 Accepted`，提示"账号创建申请已提交审核，请等待管理员审批"。管理员/经理角色可直接创建账号。

```typescript
interface UserListItem {
  id: string
  email: string
  username: string
  nickname: string
  role: string
  avatar_url?: string | null
  created_at: string
  roles: RoleAssignmentInfo[]
}

interface RoleAssignmentInfo {
  id: string
  name: string
  display_name: string
  role_type: string
}

interface CreateUserRequest {
  username: string
  password: string
  nickname: string
  email?: string
  avatar_url?: string
  role?: string
  role_ids: string[]
}

interface UpdateUserRequest {
  nickname?: string
  email?: string
  avatar_url?: string
  role_ids?: string[]
}

interface UpdateUserPasswordRequest {
  new_password: string
  old_password?: string
}

interface UpdateUserRolesRequest {
  role_ids: string[]
}

interface PermissionOverrideEntry {
  permission_key: string
  granted: boolean
}

interface UpdateUserPermissionOverridesRequest {
  overrides: PermissionOverrideEntry[]
}

interface PasswordStatusResponse {
  is_default_password: boolean
}
```

### 角色管理

| 方法   | 路径                                                    | 说明                                               | 鉴权                     |
| ------ | ------------------------------------------------------- | -------------------------------------------------- | ------------------------ |
| GET    | /api/v2/roles                                           | 角色列表                                           | 是（roles:read）         |
| POST   | /api/v2/roles                                           | 创建角色                                           | 是（roles:manage:write） |
| GET    | /api/v2/roles/{id}                                      | 角色详情                                           | 是（roles:read）         |
| PUT    | /api/v2/roles/{id}                                      | 更新角色                                           | 是（roles:manage:write） |
| DELETE | /api/v2/roles/{id}                                      | 删除角色                                           | 是（roles:manage:write） |
| POST   | /api/v2/roles/{id}/parents                              | 添加父角色                                         | 是（roles:manage:write） |
| DELETE | /api/v2/roles/{id}/parents/{parent_id}                  | 移除父角色                                         | 是（roles:manage:write） |
| GET    | /api/v2/roles/{id}/permissions                          | 角色有效权限                                       | 是                       |
| GET    | /api/v2/roles/{id}/permissions/direct                   | 角色直接权限                                       | 是（roles:read）         |
| GET    | /api/v2/roles/{id}/permissions/detail                   | 继承 + 直接权限（含 grant_type）                   | 是（roles:read）         |
| PUT    | /api/v2/roles/{id}/permissions                          | 更新直接权限                                       | 是（roles:manage:write） |
| GET    | /api/v2/roles/{id}/permissions/preview/{parent_role_id} | 预览继承指定父角色后的权限                         | 是（roles:read）         |
| GET    | /api/v2/roles/{id}/inheritance                          | 角色继承关系图（ancestor_chain + descendant_tree） | 是（roles:read）         |

```typescript
interface RoleDef {
  id: string
  name: string
  display_name: string
  description?: string
  role_type: 'admin' | 'other'
  is_super_admin: boolean
  is_builtin: boolean
  created_at: string
  updated_at: string
}

interface RolePermissionItem {
  id: string
  key: string
  name: string
  grant_type: 'direct' | 'inherited'
}

interface InheritanceRoleRef {
  id: string
  name: string
  display_name: string
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface InheritanceNode {
  role: InheritanceRoleRef
  direct_permissions: Record<string, string>
  level: number
}

interface InheritanceResponse {
  role: InheritanceRoleRef
  direct_permissions: Record<string, string>
  ancestor_chain: InheritanceNode[]
  descendant_tree: InheritanceNode[]
}
```

内置角色定义：

1. **超级管理员 (admin)** — 系统最高权限，ID=1，不可编辑，`role_type: "admin"`
2. **管理员 (manager)** — 管理系统配置，`role_type: "admin"`
3. **运营者 (operator)** — 日常内容运营，`role_type: "other"`
4. **审核员 (reviewer)** — 内容审核，`role_type: "other"`

### 资源与权限管理

| 方法   | 路径                     | 说明                                 | 鉴权 |
| ------ | ------------------------ | ------------------------------------ | ---- |
| GET    | /api/v2/resources        | 资源树                               | 是   |
| POST   | /api/v2/resources        | 新增资源                             | 是   |
| PUT    | /api/v2/resources/{id}   | 更新资源                             | 是   |
| DELETE | /api/v2/resources/{id}   | 删除资源                             | 是   |
| GET    | /api/v2/permissions      | 权限列表                             | 是   |
| GET    | /api/v2/permission-enums | 权限枚举（key + name + description） | 是   |
| GET    | /api/v2/permission-pages | 页面权限枚举                         | 是   |
| POST   | /api/v2/permissions      | 新增权限                             | 是   |
| PUT    | /api/v2/permissions/{id} | 更新权限                             | 是   |
| DELETE | /api/v2/permissions/{id} | 删除权限                             | 是   |

```typescript
interface ResourceNode {
  id: string
  key: string
  name: string
  description?: string
  parent_id?: string
  is_active: boolean
  created_at: string
  children: ResourceNode[]
}

interface PermissionEnumItem {
  id: string
  key: string
  name: string
  description?: string
}
```

**权限 key 格式规范**：

| 类型     | 格式                               | 示例                                        | 说明                        |
| -------- | ---------------------------------- | ------------------------------------------- | --------------------------- |
| 页面权限 | `{name}:read`                      | `dashboard:read`、`content:read`            | 2 段式，控制页面/菜单可见性 |
| 操作权限 | `{name}:{operation}:{read\|write}` | `content:create:write`、`users:update:read` | 3 段式，控制具体操作        |

### 约束管理

| 方法   | 路径                     | 说明     | 鉴权 |
| ------ | ------------------------ | -------- | ---- |
| GET    | /api/v2/constraints      | 约束列表 | 是   |
| POST   | /api/v2/constraints      | 创建约束 | 是   |
| PUT    | /api/v2/constraints/{id} | 更新约束 | 是   |
| DELETE | /api/v2/constraints/{id} | 删除约束 | 是   |

```typescript
interface ConstraintItem {
  id: string
  name: string
  description?: string
  constraint_type: 'mutual_exclusive' | 'prerequisite' | 'cardinality'
  config: Record<string, any>
  is_active: boolean
  created_at: string
}
```

约束类型说明：

- `mutual_exclusive`：互斥角色，同一用户不能同时拥有
- `prerequisite`：先决角色，拥有目标角色前必须先拥有先决角色
- `cardinality`：基数约束，限制某角色的最大分配用户数

### 认证接口扩展

`GET /api/auth/me` 返回当前用户信息、角色列表和项目全部权限枚举：

```json
{
  "id": "1",
  "username": "admin",
  "nickname": "超级管理员",
  "role": "admin",
  "roles": [{ "id": "role-uuid-1", "name": "admin", "display_name": "超级管理员" }],
  "permissions": {
    "dashboard:read": "仪表盘",
    "content:read": "内容列表",
    "users:create:write": "创建用户"
  }
}
```

- `roles`：用户当前分配的角色列表
- `permissions`：项目所有权限枚举 `{ key: name }`，用于前端展示权限名称

用户有效权限由 `GET /api/v2/me/permissions` 获取（返回 `{ key: name }`），后端已内置交集计算：

- 无自定义权限覆盖 → 返回角色权限的全部
- 有自定义权限覆盖 → 返回角色权限 ∩ 自定义权限（仅两者共有的权限生效）

### 权限校验机制

#### 路由权限注解模式

所有业务路由均使用 `route_class=PermAPIRoute` 配合 `@RequiresPermissions("permission_key")` 装饰器声明所需权限。`PermAPIRoute` 在路由注册时读取装饰器写入的标记，自动将权限校验作为依赖注入到对应端点。

采用该模式的路由模块：`accounts`、`contents`、`publish`、`templates`、`reviews`、`db`、`db_changes`、`dashboard`、`model_configs`、`models`、`user_creation_reviews`，以及 RBAC3 中台的 `rbac_users`、`rbac_roles`、`rbac_permissions`、`rbac_constraints`。

**例外**：`notifications.py` 未使用 `PermAPIRoute`，通过数据隔离（查询时按 `user_id` 过滤）控制访问，每个用户只能看到自己的通知。

#### 权限表达式支持

`require_permission` 支持两种模式：

1. **简单 key 模式**：直接传权限键，如 `@RequiresPermissions("users:read")`。校验时判断该 key 是否在用户有效权限集合中。
2. **表达式模式**：通过 `is_expression()` 判断字符串是否包含 `||`、`&`、`!`、`()` 运算符来识别。表达式支持下列运算符及内置函数调用，最终求值为布尔结果：
   - `||`：逻辑或
   - `&&`：逻辑与（`&` 单字符亦会触发表达式模式）
   - `!`：逻辑非
   - `()`：分组
   - 内置函数：如 `isSelf(user_id)` 判断目标用户是否为当前用户

表达式示例：

- `permissions:read||permissions:manage:read` — 满足任一权限即可
- `users:update:write && isSelf(user_id)` — 同时具备写入权限且操作本人记录

#### 权限覆盖（Override）机制

`require_permission` 在校验权限时会查询 `RBACUserPermissionOverride` 中当前用户被拒绝的权限（`granted=False`），并将这些 `permission_key` 收集为 `denied_keys` 传入 `flatten_effective_permissions`。

`flatten_effective_permissions(effective, db, denied_keys)` 的处理流程：

1. 先按 `resolve_read_keys` / `resolve_write_keys` / `_implied_read_keys` 对角色权限进行派生（如 `:write` 自动派生对应 `:read` 及父页面 `:read`）。
2. 派生完成后，通过集合差集 `flat -= deny` 应用拒绝项。

由于拒绝在派生之后应用，因此可以单独拒绝某个 `:read` 权限而保留对应的 `:write` 权限，实现细粒度的权限收回。

权限覆盖的读写接口：

- `GET /api/v2/users/{id}/permission-overrides` — 查询覆盖列表（本人或具备 `users:custom_permissions:read` 的同角色用户可查）
- `PUT /api/v2/users/{id}/permission-overrides` — 全量替换覆盖列表（需 `users:custom_permissions:write`，超级管理员账号不可设置）
