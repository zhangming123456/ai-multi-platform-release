# 账号管理与权限管理技术文档

## 1. 系统概述

本系统采用 **RBAC3（基于角色的访问控制）** 权限模型，支持角色继承、约束（互斥/先决/基数）和用户级权限覆盖。核心特点：

- **细粒度权限**：权限 key 采用 `{name}:{operation}:{read|write}` 标准化格式
- **角色继承**：子角色自动继承父角色权限，支持有向无环图（DAG）层级
- **多角色分配**：用户可同时拥有多个角色
- **约束引擎**：互斥角色、先决角色、角色成员数量限制
- **用户权限覆盖**：支持对用户单独授予或拒绝特定权限
- **权限表达式**：支持 `||`、`&`、`!`、`()` 组合的内置判断函数表达式

---

## 2. 数据模型设计

### 2.1 ER 关系图

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     users       │     │   rbac_roles    │     │ rbac_resources  │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK, UUID)   │◄────│ id (PK, UUID)   │     │ id (PK, UUID)   │
│ username (UQ)   │     │ name (UQ)       │     │ key (UQ)        │
│ email (UQ)      │     │ display_name    │     │ name            │
│ hashed_password │     │ description     │     │ description     │
│ nickname        │     │ role_type       │     │ parent_id (FK)  │
│ role            │     │ is_super_admin  │     │ is_active       │
│ avatar_url      │     │ is_builtin      │     └─────────────────┘
│ created_at      │     └─────────────────┘              │
│ updated_at      │              ▲                       │
└─────────────────┘              │                       │
         │                       │                       │
         │              ┌────────┴────────┐              │
         │              │rbac_role_hierarchy│            │
         │              ├─────────────────┤             │
         │              │ parent_role_id  │             │
         │              │ child_role_id   │             │
         │              └─────────────────┘             │
         │                       ▲                       │
         │                       │                       │
         └──────────────►┌──────┴──────┐◄──────────────┘
                         │ rbac_permissions              │
                         ├─────────────────┤             │
                         │ id (PK, UUID)   │             │
                         │ resource_id (FK)│             │
                         │ operation       │             │
                         │ key (UQ)        │             │
                         └─────────────────┘             │
                                  ▲                      │
                                  │                      │
                         ┌────────┴────────┐             │
                         │rbac_role_permissions│         │
                         ├─────────────────┤             │
                         │ role_id (FK)    │             │
                         │ permission_id   │             │
                         │ grant_type      │             │
                         └─────────────────┘             │
                                                         │
┌────────────────────────┐    ┌────────────────────────┐│
│rbac_user_role_assignments│   │rbac_user_permission_overrides│
├────────────────────────┤    ├────────────────────────┤│
│ user_id (FK)           │    │ user_id (FK)           ││
│ role_id (FK)           │    │ permission_key         ││
│ grant_type             │    │ granted                ││
│ valid_from             │    └────────────────────────┘│
│ valid_until            │                               │
└────────────────────────┘                               │
                                                         │
┌────────────────────────┐    ┌────────────────────────┐
│   rbac_constraints     │    │rbac_constraint_role_associations│
├────────────────────────┤    ├────────────────────────┤
│ id (PK, UUID)          │    │ constraint_id (FK)     │
│ name                   │    │ role_id (FK)           │
│ constraint_type        │    │ association_type       │
│ config (JSON)          │    └────────────────────────┘
└────────────────────────┘
```

### 2.2 核心模型说明

#### users（用户表）

| 字段            | 类型        | 说明                               |
| --------------- | ----------- | ---------------------------------- |
| id              | String(36)  | 主键，UUID。超级管理员固定为 `"1"` |
| username        | String(100) | 登录用户名，唯一索引               |
| email           | String(255) | 邮箱，唯一索引                     |
| hashed_password | String(255) | bcrypt 哈希后的密码                |
| nickname        | String(100) | 显示昵称                           |
| role            | String(50)  | 角色名称（保留作标签/迁移用）      |
| avatar_url      | String(500) | 头像链接                           |

#### rbac_roles（角色表）

| 字段           | 类型        | 说明                        |
| -------------- | ----------- | --------------------------- |
| id             | String(36)  | 主键，UUID                  |
| name           | String(50)  | 角色标识名，唯一            |
| display_name   | String(100) | 显示名称                    |
| description    | Text        | 角色描述                    |
| role_type      | String(20)  | 角色类型：`admin` / `other` |
| is_super_admin | Boolean     | 是否为超级管理员            |
| is_builtin     | Boolean     | 是否为内置角色（不可删除）  |

**role_type 的作用**：

- `admin`：可配置所有权限，包括数据库相关权限
- `other`：不可配置数据库相关权限和 admin-only 权限

#### rbac_resources（资源表）

| 字段        | 类型        | 说明                  |
| ----------- | ----------- | --------------------- |
| id          | String(36)  | 主键，UUID            |
| key         | String(100) | 权限 key，唯一        |
| name        | String(100) | 中文显示名称          |
| description | Text        | 权限描述说明          |
| parent_id   | String(36)  | 父资源 ID，支持资源树 |
| is_active   | Boolean     | 是否启用              |

**资源类型推断**（无需存储 type 字段）：

- `{name}:read` / `{name}:write`（2 段式）→ `page`（页面权限）
- `{name}:{operation}:{read|write}`（3 段式）→ `action`（操作权限）

#### rbac_permissions（权限表）

| 字段        | 类型        | 说明                                                                           |
| ----------- | ----------- | ------------------------------------------------------------------------------ |
| id          | String(36)  | 主键，UUID                                                                     |
| resource_id | String(36)  | 关联资源 ID                                                                    |
| operation   | String(50)  | 操作类型：read / write / create / update / delete / approve / reject / execute |
| key         | String(150) | 权限 key，唯一                                                                 |
| is_active   | Boolean     | 是否启用                                                                       |

#### rbac_role_hierarchy（角色继承表）

| 字段           | 类型       | 说明       |
| -------------- | ---------- | ---------- |
| id             | String(36) | 主键，UUID |
| parent_role_id | String(36) | 父角色 ID  |
| child_role_id  | String(36) | 子角色 ID  |

**约束**：禁止成环（parent 不能是 child 的后代，也不能等于 child）。

#### rbac_role_permissions（角色权限表）

| 字段          | 类型       | 说明                             |
| ------------- | ---------- | -------------------------------- |
| id            | String(36) | 主键，UUID                       |
| role_id       | String(36) | 角色 ID                          |
| permission_id | String(36) | 权限 ID                          |
| grant_type    | String(20) | 授权来源：`direct` / `inherited` |

#### rbac_user_role_assignments（用户角色分配表）

| 字段        | 类型       | 说明                             |
| ----------- | ---------- | -------------------------------- |
| id          | String(36) | 主键，UUID                       |
| user_id     | String(36) | 用户 ID                          |
| role_id     | String(36) | 角色 ID                          |
| grant_type  | String(20) | 分配类型：`direct` / `inherited` |
| valid_from  | DateTime   | 生效时间                         |
| valid_until | DateTime   | 失效时间                         |

#### rbac_user_permission_overrides（用户权限覆盖表）

| 字段           | 类型        | 说明                              |
| -------------- | ----------- | --------------------------------- |
| id             | String(36)  | 主键，UUID                        |
| user_id        | String(36)  | 用户 ID                           |
| permission_key | String(150) | 权限 key                          |
| granted        | Boolean     | 是否授予（true=授予，false=拒绝） |

**唯一约束**：`(user_id, permission_key)`

#### rbac_constraints（约束表）

| 字段            | 类型        | 说明                                                |
| --------------- | ----------- | --------------------------------------------------- |
| id              | String(36)  | 主键，UUID                                          |
| name            | String(100) | 约束名称                                            |
| constraint_type | String(50)  | 类型：mutual_exclusive / prerequisite / cardinality |
| config          | JSON        | 类型相关配置                                        |
| is_active       | Boolean     | 是否启用                                            |

**config 示例**：

- `mutual_exclusive`：`{"scope": "static"}`
- `prerequisite`：`{"require_all": true}`
- `cardinality`：`{"max_users": 3}`

---

## 3. 权限体系设计

### 3.1 权限 key 格式

系统权限 key 采用标准化格式：

| 类型     | 格式                               | 示例                                        | 说明                        |
| -------- | ---------------------------------- | ------------------------------------------- | --------------------------- |
| 页面权限 | `{name}:read`                      | `dashboard:read`、`content:read`            | 2 段式，控制页面/菜单可见性 |
| 操作权限 | `{name}:{operation}:{read\|write}` | `content:create:write`、`users:update:read` | 3 段式，控制具体操作        |

### 3.2 核心页面权限

| 权限 key           | 名称         | 说明               |
| ------------------ | ------------ | ------------------ |
| `dashboard:read`   | 仪表盘       | 系统首页仪表盘     |
| `platforms:read`   | 平台管理     | 第三方内容平台管理 |
| `content:read`     | 内容列表     | 内容管理页面       |
| `publish:read`     | 发布管理     | 发布任务管理       |
| `templates:read`   | 模板管理     | 内容模板管理       |
| `review:read`      | 内容审核     | 内容审核页面       |
| `sql_review:read`  | SQL审核      | SQL变更审核页面    |
| `accounts:read`    | 平台账号     | 各平台登录账号管理 |
| `token_plan:read`  | Token方案    | API Token用量方案  |
| `api_docs:read`    | API文档      | API接口文档        |
| `db:read`          | 数据库控制台 | 数据库控制台       |
| `users:read`       | 用户管理     | 系统用户列表管理   |
| `permissions:read` | 权限管理     | 角色权限分配       |
| `roles:read`       | 角色管理     | 角色定义和继承关系 |
| `constraints:read` | 约束管理     | 职责分离约束规则   |

### 3.3 核心操作权限

| 权限 key                         | 名称           | 所属分组   |
| -------------------------------- | -------------- | ---------- |
| `permissions:manage:write`       | 维护权限字典   | 权限中台   |
| `roles:manage:write`             | 维护角色       | 权限中台   |
| `constraints:manage:write`       | 维护约束       | 权限中台   |
| `content:create:write`           | 创建内容       | 内容管理   |
| `content:update:write`           | 编辑内容       | 内容管理   |
| `content:delete:write`           | 删除内容       | 内容管理   |
| `content:ai_generate:write`      | AI生成内容     | 内容管理   |
| `publish:create:write`           | 创建发布       | 发布管理   |
| `publish:retry:write`            | 重试发布       | 发布管理   |
| `templates:create:write`         | 创建模板       | 模板管理   |
| `templates:update:write`         | 编辑模板       | 模板管理   |
| `templates:delete:write`         | 删除模板       | 模板管理   |
| `review:submit:write`            | 提交审核       | 审核管理   |
| `review:approve:write`           | 通过审核       | 审核管理   |
| `review:reject:write`            | 驳回审核       | 审核管理   |
| `db:execute:write`               | 执行SQL        | 数据库管理 |
| `users:create:write`             | 创建用户       | 用户管理   |
| `users:update:read`              | 查看用户       | 用户管理   |
| `users:update:write`             | 编辑用户       | 用户管理   |
| `users:delete:write`             | 删除用户       | 用户管理   |
| `users:change_password:write`    | 修改用户密码   | 用户管理   |
| `users:custom_permissions:write` | 自定义用户权限 | 用户管理   |
| `account:view:read`              | 查看平台账号   | 平台账号   |
| `account:create:write`           | 创建平台账号   | 平台账号   |
| `account:update:write`           | 编辑平台账号   | 平台账号   |
| `account:delete:write`           | 删除平台账号   | 平台账号   |
| `account:check:write`            | 校验平台账号   | 平台账号   |
| `db_change:submit:write`         | 提交SQL变更    | SQL审核    |
| `db_change:approve:write`        | 通过SQL变更    | SQL审核    |
| `db_change:reject:write`         | 驳回SQL变更    | SQL审核    |
| `model_config:create:write`      | 创建模型配置   | 模型配置   |
| `model_config:update:write`      | 编辑模型配置   | 模型配置   |
| `model_config:delete:write`      | 删除模型配置   | 模型配置   |
| `db_history:view:read`           | 查看SQL历史    | 数据库管理 |

### 3.4 权限计算机制

`backend/app/services/rbac_service.py` 核心算法：

```python
async def get_user_effective_permissions(
    user_id: str,
    db: AsyncSession,
) -> dict[str, PermissionAccess]:
    """
    1. 超级管理员 -> 所有权限全开。
    2. 查询用户已分配角色。
    3. 对每个角色取祖先闭包，聚合角色权限（role_access_map）。
    4. 查询用户自定义权限覆盖（user_permission_overrides）。
    5. 无自定义覆盖 -> 返回全部角色权限。
    6. 有自定义覆盖 -> 取交集：有效 = 角色权限 ∩ {key | override.granted=true}。
    """

async def flatten_effective_permissions(
    effective: dict[str, PermissionAccess],
    db: AsyncSession,
) -> dict[str, str]:
    """
    将 PermissionAccess 展平为 { key: name } 格式。
    - write 权限隐含授予对应 read 权限和父页面 read 权限。
    """
```

### 3.5 约束校验

`backend/app/services/rbac_constraint_service.py`：

```python
async def validate_user_role_assignments(
    user_id: str,
    proposed_role_ids: set[str],
    db: AsyncSession,
) -> list[str]:
    """返回所有违反约束的说明；空列表表示通过。"""

async def validate_role_hierarchy(
    parent_role_id: str,
    child_role_id: str,
    db: AsyncSession,
) -> bool:
    """禁止成环。"""
```

---

## 4. 核心 API 接口

### 4.1 RBAC3 用户管理（/api/v2）

| 方法   | 路径                                    | 说明                              |
| ------ | --------------------------------------- | --------------------------------- |
| GET    | /api/v2/users                           | 用户列表（含角色分配）            |
| POST   | /api/v2/users                           | 创建用户                          |
| GET    | /api/v2/users/{id}                      | 用户详情                          |
| PUT    | /api/v2/users/{id}                      | 更新用户                          |
| DELETE | /api/v2/users/{id}                      | 删除用户（禁止删除 admin 与本人） |
| PUT    | /api/v2/users/{id}/password             | 修改密码                          |
| GET    | /api/v2/users/{id}/password-status      | 查询是否为默认密码                |
| PUT    | /api/v2/users/{id}/roles                | 分配/替换角色                     |
| GET    | /api/v2/users/{id}/permissions          | 用户有效权限                      |
| GET    | /api/v2/users/{id}/permission-overrides | 用户自定义权限覆盖列表            |
| PUT    | /api/v2/users/{id}/permission-overrides | 全量替换用户权限覆盖              |
| GET    | /api/v2/me/permissions                  | 当前用户有效权限 `{key: name}`    |

### 4.2 RBAC3 角色管理（/api/v2）

| 方法   | 路径                                                    | 说明                                             |
| ------ | ------------------------------------------------------- | ------------------------------------------------ |
| GET    | /api/v2/roles                                           | 角色列表                                         |
| POST   | /api/v2/roles                                           | 创建角色                                         |
| GET    | /api/v2/roles/{id}                                      | 角色详情                                         |
| PUT    | /api/v2/roles/{id}                                      | 更新角色                                         |
| DELETE | /api/v2/roles/{id}                                      | 删除角色                                         |
| POST   | /api/v2/roles/{id}/parents                              | 添加父角色                                       |
| DELETE | /api/v2/roles/{id}/parents/{parent_id}                  | 移除父角色                                       |
| GET    | /api/v2/roles/{id}/permissions                          | 有效权限                                         |
| GET    | /api/v2/roles/{id}/permissions/direct                   | 直接权限列表                                     |
| GET    | /api/v2/roles/{id}/permissions/detail                   | 继承 + 直接权限                                  |
| PUT    | /api/v2/roles/{id}/permissions                          | 更新直接权限                                     |
| GET    | /api/v2/roles/{id}/permissions/preview/{parent_role_id} | 预览添加父角色后继承的权限                       |
| GET    | /api/v2/roles/{id}/inheritance                          | 角色继承关系（祖先链 + 后代树 + 各层级直接权限） |

### 4.3 RBAC3 资源与权限管理（/api/v2）

| 方法   | 路径                     | 说明                                 |
| ------ | ------------------------ | ------------------------------------ |
| GET    | /api/v2/resources        | 资源树                               |
| POST   | /api/v2/resources        | 新增资源                             |
| PUT    | /api/v2/resources/{id}   | 更新资源                             |
| DELETE | /api/v2/resources/{id}   | 删除资源                             |
| GET    | /api/v2/permission-enums | 权限枚举（key + name + description） |
| GET    | /api/v2/permission-pages | 页面权限枚举                         |
| POST   | /api/v2/permissions      | 新增权限                             |
| PUT    | /api/v2/permissions/{id} | 更新权限                             |
| DELETE | /api/v2/permissions/{id} | 删除权限                             |

### 4.4 认证与用户管理

#### POST /api/v2/users — 创建用户

**行为差异**：

- **管理员**：直接创建用户，返回 201
- **非管理员**：创建 `UserCreationRequest` 审核记录，返回 202

**安全校验**：

- 非管理员只能创建 `operator` 角色
- 用户名/邮箱唯一性校验
- 禁止创建 `admin` 角色

#### PUT /api/v2/users/{id}/password — 修改密码

**密码策略**：

| 场景                     | 行为                     |
| ------------------------ | ------------------------ |
| 超级管理员重置他人密码   | 无需旧密码，直接重置     |
| 管理员重置他人密码       | 无需旧密码，直接重置     |
| 用户修改自己的默认密码   | 无需旧密码，直接设新密码 |
| 用户修改自己的非默认密码 | 必须提供旧密码验证       |

**新密码格式验证**：

- 首字符必须为字母（a-zA-Z）
- 仅允许字符：`a-zA-Z0-9._@$`
- 长度至少 6 位

---

## 5. 权限校验机制

### 5.1 依赖注入层级

```
get_current_user          — 解析 JWT Token，获取当前用户
  └─ require_permission   — 细粒度权限校验（支持权限表达式）
```

### 5.2 路由权限注解模式（推荐用法）

所有业务路由统一采用 `route_class=PermAPIRoute` + `@RequiresPermissions("permission_key")` 装饰器模式，而非在每个端点显式写 `Depends(require_permission(...))`。

```python
from app.core.deps import PermAPIRoute, RequiresPermissions

router = APIRouter(prefix="/api/v2", tags=["RBAC 用户管理"], route_class=PermAPIRoute)

@router.get("/users", response_model=list[UserDetailResponse])
@RequiresPermissions("users:read")
async def list_users(db: AsyncSession = Depends(get_db),
                     current_user: User = Depends(get_current_user)):
    ...
```

**机制说明**：

- `@RequiresPermissions(expr, context=None)` 装饰器将权限表达式与上下文写入端点函数的 `_requires_perm_info_` 标记位
- `PermAPIRoute` 在注册路由时读取该标记，自动构造 `require_permission(expr, ctx)` 依赖并追加到 `dependencies`
- 支持传入上下文（如 `{"content": some_content_obj}`）供表达式内置函数使用
- 少数特殊端点（如 `GET /users/{id}/permissions`、`GET /users/{id}/permission-overrides`）因需要细粒度的同角色访问控制，未使用装饰器，而是在函数体内调用 `has_permission_direct` 进行手工校验

### 5.3 require_permission 依赖

```python
# 普通权限校验
require_permission("content:create:write")

# 权限表达式校验（支持内置判断函数）
require_permission("users:update:read || isSelf(id)")
require_permission("isAdmin() || isOwnContent(content)")
```

**表达式判断**：通过 `is_expression(permission_key)` 判断是否为表达式（字符串中包含 `|`、`&`、`(`、`)`、`!` 任一字符即为表达式）。

**校验流程**：

1. 获取用户有效权限（`get_user_effective_permissions`）
2. 查询 `RBACUserPermissionOverride` 表中 `granted=False` 的记录，得到 `denied_keys` 集合
3. 调用 `flatten_effective_permissions(effective, db, denied_keys)` 展平为 `{ key: name }` 格式（denied_keys 中的 key 会被剔除）
4. 若 `is_expression(permission_key)` 为 True → 调用 `evaluate_permission(permission_key, flat_perms, eval_ctx)` 进行表达式求值（表达式语法错误返回 500）
5. 若为普通 key → 检查 key 是否在展平后的权限集合中
6. 未通过 → 403 "无权限" / "无访问权限"

### 5.4 前端路由守卫

```typescript
router.beforeEach(async (to) => {
  const token = localStorage.getItem('token')
  if (to.name === 'Login' && token) return { name: 'Dashboard' }
  if (to.meta.public) return true
  if (!token) return { name: 'Login', query: { redirect: to.fullPath } }

  const userStore = useUserStore()
  const permStore = usePermissionStore()

  if (!userStore.userInfo) await userStore.fetchUserInfo()
  if (
    permStore.lastPermissionsUserId !== userStore.userInfo.id ||
    Object.keys(permStore.permissions).length === 0
  ) {
    await permStore.loadPermissions(userStore.userInfo.id)
  }

  if (!hasPerm(to)) return { name: 'Forbidden', query: { from: to.fullPath } }
  return true
})

// permKey 支持表达式，将 query+params 合并为上下文
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
```

### 5.5 前端权限工具

```typescript
// stores/permission.ts
function hasPermission(keyOrExpr: string, ctx?: PermContext): boolean {
  if (!isExpression(keyOrExpr)) {
    return keyOrExpr in permissions.value
  }
  return evaluatePermission(keyOrExpr, permissions.value, _mergeContext(ctx))
}

// directives/permission.ts —— DOM 移除/恢复机制（WeakMap 缓存 + Comment 占位符）
const _permCache = new WeakMap<
  HTMLElement,
  {
    placeholder: Comment
    originalParent: Node
    originalNext: Node | null
  }
>()

function removeEl(el: HTMLElement) {
  const parent = el.parentNode
  if (!parent) return
  const placeholder = document.createComment('v-perm')
  const next = el.nextSibling
  parent.insertBefore(placeholder, el)
  parent.removeChild(el)
  _permCache.set(el, { placeholder, originalParent: parent, originalNext: next })
}

function restoreEl(el: HTMLElement) {
  const cache = _permCache.get(el)
  if (!cache) return
  const { placeholder, originalParent, originalNext } = cache
  if (!placeholder.parentNode) {
    originalParent.insertBefore(el, originalNext)
  } else {
    placeholder.parentNode.insertBefore(el, placeholder)
    placeholder.parentNode.removeChild(placeholder)
  }
  _permCache.delete(el)
}

const vPerm: ObjectDirective<HTMLElement, string | PermBinding> = {
  mounted: checkAndApply,
  updated: checkAndApply,
  beforeUnmount(el) {
    _permCache.delete(el)
  },
}
```

**v-perm 指令说明**：

- 相比 `display: none`，采用真实 DOM 移除 + `Comment('v-perm')` 占位符机制，权限不足时元素从 DOM 树中彻底移除
- `WeakMap` 缓存原节点关系（父节点、兄弟节点、占位符），权限恢复时可在原位置重新插入
- `mounted` 与 `updated` 钩子均调用 `checkAndApply`，权限状态变化时自动移除/恢复
- binding.value 支持字符串（权限 key/表达式）或对象 `{ key, ctx }` 两种形式

---

## 6. 密码管理

### 6.1 密码策略

| 规则     | 说明                          |
| -------- | ----------------------------- |
| 首字符   | 必须为字母（a-zA-Z）          |
| 合法字符 | 字母 + 数字 + `.` `_` `@` `$` |
| 最小长度 | 6 位                          |
| 存储方式 | bcrypt 哈希                   |

### 6.2 默认密码机制

创建账号时自动生成默认密码：`${username}123`

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

### 7.1 RBAC3 用户管理（RBACUserManage.vue）

**功能**：

- 用户列表展示（用户名/昵称/邮箱/角色/创建时间）
- 角色筛选
- 添加账号（管理员直接创建，非管理员提交审核）
- 编辑用户信息
- 修改密码（区分默认密码/旧密码流程）
- 用户自定义权限配置
- 删除用户

**权限控制**：

| 操作按钮 | 显示条件                                       |
| -------- | ---------------------------------------------- |
| 添加账号 | `users:create:write` 权限                      |
| 权限配置 | 管理员 或 本人（非 admin 用户）                |
| 修改密码 | 管理员 或 本人 + `users:change_password:write` |
| 编辑     | 管理员 或 本人 + `users:update:write`          |
| 删除     | `users:delete:write` + 非 admin 用户           |

### 7.2 角色管理（RBACRoleManage.vue）

**功能**：

- 角色列表（按超级管理员→管理员→运营者→审核员→自定义角色排序）
- 角色 CRUD（内置角色不可编辑/删除）
- 父子关系配置（DAG 校验，禁止成环）
- 约束提示（互斥/先决/基数）

### 7.3 权限管理（RBACPermissionManage.vue）

**功能**：

- 按资源树配置角色权限
- 区分继承权限与直接权限（继承权限不可取消）
- 显示权限描述信息
- 超级管理员显示为只读"已拥有"状态

### 7.4 权限字典管理（RBACPermissionEnumManage.vue）

**功能**：

- 权限枚举 CRUD（key、名称、描述编辑）
- 权限 key 格式校验
- 页面权限与操作权限筛选

### 7.5 用户自定义权限（RBACUserPermissionCustomize.vue）

**功能**：

- 为用户单独授予或拒绝特定权限
- 覆盖角色默认权限
- 重置为角色默认权限

### 7.6 约束管理（RBACConstraintManage.vue）

**功能**：

- 互斥/先决/基数约束 CRUD
- 约束角色关联配置
- 实时校验提示

### 7.7 用户创建审核（UserCreationReview.vue）

**功能**：

- 待审核申请列表
- 审批通过（自动创建用户）
- 审批驳回（需填写驳回原因）
- 仅管理员可见

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
├── core/
│   └── deps.py                    # require_permission（支持权限表达式）
├── models/
│   ├── rbac_role.py               # RBACRole 模型
│   ├── rbac_resource.py           # RBACResource 模型（含 _infer_resource_type）
│   ├── rbac_permission.py         # RBACPermission 模型
│   ├── rbac_role_hierarchy.py     # RBACRoleHierarchy 模型
│   ├── rbac_role_permission.py    # RBACRolePermission 模型
│   ├── rbac_user_role_assignment.py  # RBACUserRoleAssignment 模型
│   ├── rbac_user_permission_override.py  # RBACUserPermissionOverride 模型
│   ├── rbac_constraint.py         # RBACConstraint 模型
│   ├── user.py                    # User 模型（保留 role 字段作迁移用）
│   └── user_creation_request.py   # UserCreationRequest 模型
├── routers/
│   ├── rbac_users.py              # RBAC3 用户管理
│   ├── rbac_roles.py              # RBAC3 角色管理
│   ├── rbac_permissions.py        # RBAC3 资源与权限管理
│   ├── rbac_constraints.py        # RBAC3 约束管理
│   └── user_creation_reviews.py   # 用户创建审核
├── services/
│   ├── rbac_service.py            # 权限计算服务
│   ├── rbac_constraint_service.py # 约束校验服务
│   ├── rbac_init_service.py       # RBAC3 初始化服务
│   └── perm_expression.py         # 权限表达式解析与求值
└── schemas/
    └── auth.py                    # 用户/权限相关 Pydantic Schema
```

### 前端

```
frontend/src/
├── pages/
│   ├── RBACUserManage.vue         # RBAC3 用户管理
│   ├── RBACUserCreate.vue         # 创建用户
│   ├── RBACUserEdit.vue           # 编辑用户
│   ├── RBACUserPassword.vue       # 修改密码
│   ├── RBACUserPermissionCustomize.vue  # 用户自定义权限
│   ├── RBACRoleManage.vue         # 角色管理
│   ├── RBACPermissionManage.vue   # 权限管理
│   ├── RBACPermissionEnumManage.vue  # 权限字典管理
│   ├── RBACConstraintManage.vue   # 约束管理
│   └── UserCreationReview.vue     # 用户创建审核
├── stores/
│   └── permission.ts              # 权限状态管理
├── directives/
│   └── permission.ts              # v-perm 指令
├── router/
│   └── index.ts                   # 路由 + 路由守卫
├── components/layout/
│   └── AppSidebar.vue             # 侧边栏菜单（权限驱动）
└── components/rbac/
    └── ResourcePermissionCard.vue # 权限卡片（继承/直接区分）
```
