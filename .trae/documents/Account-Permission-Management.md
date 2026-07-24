# 账号管理与权限管理技术文档

## 1. 系统概述

本系统采用 **RBAC（基于角色的访问控制）** 模型，支持角色级和用户级双重权限控制。核心特点：

- **细粒度权限**：每个权限项支持独立的"读"和"写"控制
- **双重权限层**：角色默认权限 + 用户自定义权限（覆盖式）
- **内置角色 + 自定义角色**：支持管理员创建自定义角色并分配权限
- **审核工作流**：非管理类型角色创建账号需提交审核

---

## 2. 数据模型设计

### 2.1 ER 关系图

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     users       │     │  custom_roles   │     │ role_permissions│
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK, UUID)   │     │ id (PK, UUID)   │     │ id (PK, UUID)   │
│ username (UQ)   │     │ name (UQ)       │◄────│ role            │
│ email (UQ)      │     │ display_name    │     │ permission_key  │
│ hashed_password │     │ description     │     │ can_read        │
│ nickname        │     │ role_type       │     │ can_write       │
│ role            │     │ created_at      │     │ created_at      │
│ avatar_url      │     └─────────────────┘     └─────────────────┘
│ created_at      │                              UQ(role, permission_key)
│ updated_at      │
└─────────────────┘     ┌─────────────────┐     ┌──────────────────────┐
         │               │ user_permissions │     │ user_creation_requests│
         │               ├─────────────────┤     ├──────────────────────┤
         │               │ id (PK, UUID)   │     │ id (PK, UUID)        │
         └──────────────►│ user_id (FK)    │     │ requester_id (FK)    │
                         │ permission_key  │     │ username             │
                         │ can_read        │     │ email                │
                         │ can_write       │     │ hashed_password      │
                         │ created_at      │     │ nickname             │
                         └─────────────────┘     │ role                 │
                          UQ(user_id, perm_key)  │ avatar_url           │
                                                 │ status (pending/     │
                                                 │   approved/rejected) │
                                                 │ reviewer_id (FK)     │
                                                 │ reject_reason        │
                                                 │ created_at           │
                                                 │ updated_at           │
                                                 └──────────────────────┘
```

### 2.2 模型详细说明

#### 2.2.1 User（用户表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | String(36) | 主键，UUID。超级管理员固定为 `"1"` |
| username | String(100) | 登录用户名，唯一索引 |
| email | String(255) | 邮箱，可选，唯一索引 |
| hashed_password | String(255) | bcrypt 哈希后的密码 |
| nickname | String(100) | 显示昵称 |
| role | String(50) | 角色名称，默认 `"operator"` |
| avatar_url | String(500) | 头像链接，可选 |

**角色枚举（UserRole）**：
- `admin` — 超级管理员（系统唯一，ID="1"）
- `manager` — 管理员
- `operator` — 运营者
- `reviewer` — 审核员

#### 2.2.2 CustomRole（自定义角色表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | String(36) | 主键，UUID |
| name | String(50) | 角色标识名，唯一 |
| display_name | String(100) | 显示名称 |
| description | Text | 角色描述 |
| role_type | String(20) | 角色类型：`"admin"`（管理类型）或 `"other"`（其他类型） |

**role_type 的作用**：
- `"admin"`：可配置数据库相关权限（database、db:execute、db:history:read）及 admin-only 权限
- `"other"`：不可配置数据库相关权限和 admin-only 权限

#### 2.2.3 RolePermission（角色权限表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | String(36) | 主键，UUID |
| role | String(50) | 角色名（关联 User.role 或 CustomRole.name） |
| permission_key | String(50) | 权限键 |
| can_read | Boolean | 读权限，默认 true |
| can_write | Boolean | 写权限，默认 true |

**唯一约束**：`(role, permission_key)`

#### 2.2.4 UserPermission（用户自定义权限表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | String(36) | 主键，UUID |
| user_id | String(36) | 关联用户 ID |
| permission_key | String(50) | 权限键 |
| can_read | Boolean | 读权限 |
| can_write | Boolean | 写权限 |

**唯一约束**：`(user_id, permission_key)`

#### 2.2.5 UserCreationRequest（用户创建审核表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | String(36) | 主键，UUID |
| requester_id | String(36) | 申请人 ID，外键 → users.id |
| username | String(100) | 待创建的用户名 |
| email | String(255) | 待创建的邮箱 |
| hashed_password | String(255) | 待创建用户的密码哈希 |
| nickname | String(100) | 待创建用户的昵称 |
| role | String(50) | 待创建用户的角色 |
| avatar_url | String(500) | 头像链接 |
| status | String(20) | 审核状态：`pending`/`approved`/`rejected` |
| reviewer_id | String(36) | 审批人 ID，外键 → users.id |
| reject_reason | String(500) | 驳回原因 |

---

## 3. 权限体系设计

### 3.1 权限定义

系统权限分为两类：**页面权限**（`type=page`）和 **操作权限**（`type=action`）。

#### 页面权限

| 权限键 | 名称 | 说明 |
|--------|------|------|
| `dashboard` | 仪表盘 | 首页数据看板 |
| `platforms` | 平台管理 | 平台账号管理页面 |
| `content` | 内容工坊 | 内容管理页面 |
| `publish` | 发布管理 | 多平台发布页面 |
| `templates` | 模板中心 | 内容模板管理 |
| `review` | 内容审核 | 内容审核页面 |
| `sql_review` | SQL 审核 | SQL 变更审核页面 |
| `accounts` | 账号管理 | 用户账号管理页面 |
| `token_plan` | Token 配置 | Token 套餐配置 |
| `api_docs` | API 文档 | API 文档页面 |
| `database` | 数据库管理 | 数据库控制台（仅管理类型角色） |
| `permission_manage` | 权限管理 | 权限配置页面（仅管理类型角色） |
| `user_perm_manage` | 用户权限配置 | 用户自定义权限操作 |

#### 操作权限

| 权限键 | 名称 | 所属分组 |
|--------|------|----------|
| `user:create` | 创建用户 | 系统设置 |
| `user:update` | 编辑用户 | 系统设置 |
| `user:delete` | 删除用户 | 系统设置 |
| `user:change_password` | 修改用户密码 | 系统设置 |
| `content:create` | 创建内容 | 内容管理 |
| `content:update` | 编辑内容 | 内容管理 |
| `content:delete` | 删除内容 | 内容管理 |
| `content:ai_generate` | AI 生成 | 内容管理 |
| `review:submit` | 提交内容审核 | 审核管理 |
| `review:approve` | 审核通过 | 审核管理 |
| `review:reject` | 审核驳回 | 审核管理 |
| `db_change:submit` | 提交SQL变更 | 审核管理 |
| `db_change:approve` | SQL变更审核通过 | 审核管理 |
| `db_change:reject` | SQL变更驳回 | 审核管理 |
| `template:create` | 创建模板 | 内容管理 |
| `template:update` | 编辑模板 | 内容管理 |
| `template:delete` | 删除模板 | 内容管理 |
| `account:create` | 添加平台账号 | 基础 |
| `account:update` | 编辑平台账号 | 基础 |
| `account:delete` | 删除平台账号 | 基础 |
| `account:check` | 检测账号状态 | 基础 |
| `publish:create` | 创建发布任务 | 内容管理 |
| `publish:retry` | 重试发布任务 | 内容管理 |
| `model_config:create` | 创建模型配置 | 系统设置 |
| `model_config:update` | 编辑模型配置 | 系统设置 |
| `model_config:delete` | 删除模型配置 | 系统设置 |
| `db:execute` | 执行SQL命令 | 系统设置 |
| `db:history:read` | 查看SQL历史 | 系统设置 |

#### 特殊权限分组

**ADMIN_ONLY_PERMISSIONS**（仅管理类型角色可分配）：
```
user_perm_manage, permission_manage, database, db:execute,
db:history:read, model_config:create, model_config:update,
model_config:delete, user:delete
```

**DATABASE_PERMISSIONS**（数据库相关，仅管理类型角色）：
```
database, db:execute, db:history:read
```

### 3.2 读写分离权限模型

每个权限项包含两个独立开关：

```typescript
interface PermissionAccess {
  read: boolean   // 读权限：是否可查看/访问
  write: boolean  // 写权限：是否可编辑/操作
}
```

**应用场景**：
- 开启"读" + 关闭"写" → 用户可查看权限配置弹窗但无法编辑
- 开启"读" + 开启"写" → 用户可正常查看和操作
- 关闭"读" → 用户无法看到对应菜单或页面

### 3.3 权限继承与覆盖机制

权限解析优先级（从高到低）：

```
用户自定义权限（UserPermission） > 角色数据库权限（RolePermission） > 角色默认权限（DEFAULT_ROLE_PERMISSIONS）
```

**具体逻辑**：
1. 若 `user.role == "admin"` → 直接返回所有权限（读写全开）
2. 查询 `UserPermission` 表，若有记录 → 使用用户自定义权限
3. 查询 `RolePermission` 表，若有记录 → 使用角色数据库权限
4. 否则使用 `DEFAULT_ROLE_PERMISSIONS` 中定义的内置默认权限

**数据库权限过滤**：非管理类型角色的权限在返回前会被过滤掉 DATABASE_PERMISSIONS。

### 3.4 内置角色默认权限

| 角色 | 包含权限 |
|------|----------|
| **admin** | 全部 42 项权限 |
| **manager** | 除 database、db:execute、db:history:read、model_config:*、user:delete 外的所有权限 |
| **operator** | 日常运营相关权限（dashboard, content, publish, templates, platforms, accounts, token_plan, api_docs + 对应的 create/update/delete 操作权限） |
| **reviewer** | 审核相关权限（dashboard, content, review, sql_review, platforms + review:approve/reject + db_change:approve/reject + template:*） |

---

## 4. 核心 API 接口

### 4.1 认证与用户管理

#### POST /api/users/ — 创建用户

**权限要求**：`user:create` 写权限

**行为差异**：
- **管理员（admin/manager）**：直接创建用户，返回 201
- **非管理员（operator/reviewer 等）**：创建 `UserCreationRequest` 审核记录，返回 202

**请求体**：
```json
{
  "username": "newuser",
  "password": "newuser123",
  "nickname": "新用户",
  "email": "new@test.com",
  "role": "operator"
}
```

**202 响应**：
```json
{"detail": "账号创建申请已提交审核，请等待管理员审批"}
```

**安全校验**：
- 非管理员只能创建 `operator` 角色
- 用户名唯一性校验
- 邮箱唯一性校验
- 禁止创建 `admin` 角色

#### PUT /api/users/{user_id}/password — 修改密码

**权限要求**：`user:change_password` 写权限

**密码策略**：
| 场景 | 行为 |
|------|------|
| 超级管理员重置他人密码 | 无需旧密码，直接重置 |
| 管理员（有权限）重置他人密码 | 无需旧密码，直接重置 |
| 用户修改自己的默认密码 | 无需旧密码，直接设新密码 |
| 用户修改自己的非默认密码 | 必须提供旧密码验证 |

**新密码格式验证**：
- 首字符必须为字母（大小写均可）
- 仅允许字符：`a-zA-Z0-9._@$`
- 长度至少 6 位

**请求体**：
```json
{
  "new_password": "NewPwd@2024",
  "old_password": "OldPwd123"
}
```

#### GET /api/users/{user_id}/password-status — 查询密码状态

**权限要求**：登录用户（本人或管理员）

**响应**：
```json
{"is_default_password": true}
```

### 4.2 角色管理

#### GET /api/roles — 获取角色列表

**响应**（无需特殊权限）：
```json
[
  {
    "name": "admin",
    "display_name": "超级管理员",
    "description": "系统最高权限...",
    "is_builtin": true,
    "is_super_admin": true,
    "role_type": "admin"
  },
  {
    "name": "myrole",
    "display_name": "我的角色",
    "description": "自定义角色",
    "is_builtin": false,
    "is_super_admin": false,
    "role_type": "other"
  }
]
```

#### POST /api/roles — 创建自定义角色

**权限要求**：管理员（admin/manager）

**请求体**：
```json
{
  "name": "content_editor",
  "display_name": "内容编辑",
  "description": "负责内容编辑的角色",
  "role_type": "other"
}
```

#### PUT /api/roles/{role_name} — 更新自定义角色

**权限要求**：管理员

**说明**：内置角色不可修改

#### DELETE /api/roles/{role_name} — 删除自定义角色

**权限要求**：管理员

**保护机制**：
- 内置角色不可删除
- 角色下仍有用户时不可删除
- 自动清理关联的 RolePermission 记录

### 4.3 权限管理

#### GET /api/permissions/all — 获取所有权限和角色权限

**权限要求**：登录用户

**响应**：
```json
{
  "permissions": [{"key": "dashboard", "name": "仪表盘", "group": "页面权限", "type": "page"}, ...],
  "roles": ["admin", "manager", "operator", "reviewer", "myrole"],
  "role_permissions": {
    "admin": {"dashboard": {"read": true, "write": true}, ...},
    "operator": {"dashboard": {"read": true, "write": true}, ...}
  }
}
```

#### PUT /api/permissions/role/{role} — 更新角色权限

**权限要求**：管理员

**保护机制**：
- 超级管理员角色不可修改
- 非管理类型角色不可配置数据库相关权限
- 非管理类型角色不可分配 admin-only 权限

**请求体**：
```json
{
  "role": "operator",
  "permissions": {
    "dashboard": {"read": true, "write": false},
    "content": {"read": true, "write": true}
  }
}
```

#### GET /api/permissions/user/{user_id} — 获取用户权限详情

**权限要求**：管理员或用户本人

**响应**：
```json
{
  "user_id": "uuid",
  "role": "operator",
  "role_permissions": {"dashboard": {"read": true, "write": true}, ...},
  "custom_permissions": null,
  "effective_permissions": {"dashboard": {"read": true, "write": true}, ...},
  "is_editable": false
}
```

**字段说明**：
- `custom_permissions`：用户自定义权限（null 表示未自定义，使用角色默认）
- `effective_permissions`：最终生效权限（自定义 > 角色默认）
- `is_editable`：当前用户是否可编辑目标用户的权限

#### PUT /api/permissions/user/{user_id} — 更新用户自定义权限

**权限要求**：管理员

**保护机制**：
- 超级管理员不可修改
- 只能在角色已有权限范围内自定义
- 非管理类型角色不可配置数据库权限

#### DELETE /api/permissions/user/{user_id}/custom — 重置用户权限

**权限要求**：管理员

**行为**：删除 UserPermission 记录，恢复为角色默认权限

### 4.4 用户创建审核

#### GET /api/user-creation-reviews/ — 获取待审核列表

**权限要求**：`review` 读权限 + 管理员角色

**响应**：
```json
[
  {
    "id": "uuid",
    "requester_id": "uuid",
    "requester_name": "运营小二",
    "username": "newuser",
    "email": "new@test.com",
    "nickname": "新用户",
    "role": "operator",
    "status": "pending",
    "reviewer_id": null,
    "reviewer_name": null,
    "reject_reason": null,
    "created_at": "2026-07-24T21:00:41"
  }
]
```

#### POST /api/user-creation-reviews/{id}/approve — 审批通过

**权限要求**：`review:approve` 写权限 + 管理员角色

**行为**：
1. 校验审核请求状态为 pending
2. 校验用户名未被占用
3. 创建 User 记录
4. 更新审核请求状态为 approved，记录审批人

#### POST /api/user-creation-reviews/{id}/reject — 审批驳回

**权限要求**：`review:reject` 写权限 + 管理员角色

**请求体**：
```json
{"reason": "用户名不符合规范"}
```

---

## 5. 权限校验机制

### 5.1 依赖注入层级

```
get_current_user          — 解析 JWT Token，获取当前用户
  └─ get_admin_user       — 要求 admin/manager 角色
  └─ get_reviewer_user    — 要求 admin/manager/reviewer 角色
  └─ require_permission(key, mode)  — 细粒度权限校验
```

### 5.2 require_permission 装饰器

```python
require_permission("user:create", "write")
```

**校验流程**：
1. 若 `user.role == "admin"` → 直接放行
2. 获取用户权限映射（用户自定义 > 角色数据库 > 角色默认）
3. 若权限键不存在 → 403"无操作权限"
4. 若 `mode="write"` 且 `can_write=False` → 403"无写入权限"
5. 若 `mode="read"` 且 `can_read=False` → 403"无查看权限"

### 5.3 前端路由守卫

```typescript
router.beforeEach(async (to) => {
  // 1. 检查是否已登录（有 token）
  // 2. 获取用户信息（含 permissions）
  // 3. 检查 meta.permKey 对应的 read 权限
  // 4. admin 角色直接放行
})
```

### 5.4 前端权限工具函数

```typescript
// store 中的权限检查
function hasPermission(key: string, mode: 'read' | 'write' = 'read'): boolean

// 组件中的权限检查
function hasStorePerm(key: string, mode: 'read' | 'write' = 'write'): boolean
```

---

## 6. 密码管理

### 6.1 密码策略

| 规则 | 说明 |
|------|------|
| 首字符 | 必须为字母（a-zA-Z） |
| 合法字符 | 字母 + 数字 + `.` `_` `@` `$` |
| 最小长度 | 6 位 |
| 存储方式 | bcrypt 哈希 |

### 6.2 默认密码机制

创建账号时自动生成默认密码：`${username}123`

**前端自动填充**：输入用户名后自动填充密码和昵称

### 6.3 密码修改流程

```
用户打开修改密码弹窗
  │
  ├─ 管理员（admin/manager）→ 直接显示新密码输入框
  │
  └─ 非管理员 → 调用 GET /password-status
        │
        ├─ is_default_password=true → 显示"可直接设置新密码"
        │
        └─ is_default_password=false → 显示旧密码输入框（必填）
```

---

## 7. 前端页面结构

### 7.1 账号管理（Accounts.vue）

**功能**：
- 用户列表展示（用户名/昵称/邮箱/角色/创建时间）
- 角色筛选
- 添加账号（管理员直接创建，非管理员提交审核）
- 编辑用户信息
- 修改密码（区分默认密码/旧密码流程）
- 用户自定义权限配置（读/写独立控制）
- 删除用户

**权限控制**：
| 操作按钮 | 显示条件 |
|----------|----------|
| 添加账号 | `user:create` 写权限 |
| 权限配置 | 管理员 或 本人（非 admin 用户） |
| 修改密码 | 管理员 或 本人 + `user:change_password` 写权限 |
| 编辑 | 管理员 或 本人 + `user:update` 写权限 |
| 删除 | `user:delete` 写权限 + 非 admin 用户 |

### 7.2 权限管理（PermissionManage.vue）

**功能**：
- 左侧角色列表（按超级管理员→管理员→运营者→审核员→自定义角色排序）
- 右侧权限配置面板（按分组展示权限项）
- 每个权限项独立控制读/写开关
- 支持按分组全选读/全选写
- 超级管理员显示为只读"已拥有"状态
- 非管理类型角色自动隐藏数据库相关权限

### 7.3 用户创建审核（UserCreationReview.vue）

**功能**：
- 待审核申请列表（申请人/申请账号/昵称/邮箱/分配角色/申请时间）
- 审批通过（自动创建用户）
- 审批驳回（需填写驳回原因）
- 仅管理员（admin/manager）可见

---

## 8. 安全设计

### 8.1 超级管理员保护

- 系统唯一，ID 固定为 `"1"`
- 注册接口禁止创建 admin 角色
- 更新接口禁止将其他用户改为 admin
- 删除接口禁止删除 admin 用户
- 权限编辑接口禁止修改 admin 角色的权限

### 8.2 角色类型隔离

- **管理类型（admin）**：可配置所有权限，包括数据库和 admin-only 权限
- **其他类型（other）**：不可配置数据库权限和 admin-only 权限

### 8.3 密码安全

- bcrypt 哈希存储，不可逆
- 新密码格式强制校验
- 旧密码验证防止未授权修改
- 默认密码机制确保初始可登录

### 8.4 隐私保护

- `PrivacyMaskMiddleware` 中间件对响应中的邮箱等敏感信息自动脱敏
- 用户列表中邮箱显示为 `o***@example.com` 格式

---

## 9. 文件结构

### 后端

```
backend/app/
├── models/
│   ├── user.py                    # User 模型 + UserRole 枚举
│   ├── custom_role.py             # CustomRole 模型
│   ├── role_permission.py         # RolePermission 模型
│   ├── user_permission.py         # UserPermission 模型
│   └── user_creation_request.py   # UserCreationRequest 模型
├── routers/
│   ├── users.py                   # 用户 CRUD + 密码管理
│   ├── roles.py                   # 角色 CRUD + 内置角色定义
│   ├── permissions.py             # 权限管理 + 权限定义 + 权限解析
│   └── user_creation_reviews.py   # 用户创建审核
├── core/
│   ├── deps.py                    # 权限依赖注入（require_permission）
│   └── security.py                # 密码哈希/验证 + JWT + 密码格式校验
└── schemas/
    └── auth.py                    # 用户/权限相关 Pydantic Schema
```

### 前端

```
frontend/src/
├── pages/
│   ├── Accounts.vue               # 账号管理页面
│   ├── PermissionManage.vue       # 权限管理页面
│   ├── RoleManage.vue             # 角色管理页面
│   └── UserCreationReview.vue     # 用户创建审核页面
├── stores/
│   └── user.ts                    # 用户状态 + 权限工具函数
├── router/
│   └── index.ts                   # 路由 + 路由守卫
└── components/layout/
    └── AppSidebar.vue             # 侧边栏菜单（权限驱动）
```
