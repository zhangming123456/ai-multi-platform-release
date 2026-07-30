# RBAC3 权限模型迭代升级技术文档

> **目标：** 将现有基于 `User.role` + `RolePermission` + `UserPermission` 的 RBAC0 模型升级为支持角色继承、约束（SoD/基数/先决条件）和会话级授权的 RBAC3 模型，同时保持前端菜单、路由、页面按钮的权限校验完全联动。
>
> **参考：** NIST RBAC 标准（RBAC0 基础、RBAC1 角色层次、RBAC2 约束、RBAC3 = RBAC1 + RBAC2）。
>
> **技术栈：** FastAPI + SQLAlchemy 2.0（Async）+ SQLite；Vue 3 + TypeScript + Pinia + Arco Design + Vue Router。

---

## 1. 现状与目标对比

### 1.1 当前模型（RBAC0）

| 层级 | 表/文件 | 说明 |
|------|---------|------|
| 用户 | `users` | `role` 字段决定默认角色 |
| 自定义角色 | `custom_roles` | 扩展内置角色 |
| 角色权限 | `role_permissions` | `role` + `permission_key` + `can_read`/`can_write` |
| 用户权限覆盖 | `user_permissions` | 用户级自定义权限 |
| 接口 | `users.py` / `roles.py` / `permissions.py` | 旧版 CRUD + 权限解析 |
| 前端 | `Accounts.vue` / `RoleManage.vue` / `PermissionManage.vue` | 基于旧权限键渲染 |

### 1.2 目标模型（RBAC3）

| 层级 | 新增表/文件 | 能力 |
|------|-------------|------|
| 用户-角色 | `rbac_user_role_assignments` | 多对多、有效时间范围、支持一用户多角色 |
| 角色 | `rbac_roles` | 内置/自定义、超级管理员、角色类型 |
| 角色继承 | `rbac_role_hierarchy` | 有向无环图（DAG），子角色继承父角色权限 |
| 资源 | `rbac_resources` | 页面/操作资源树，支持层级 |
| 权限 | `rbac_permissions` | 资源 + 操作粒度（read/write/create/update/delete/approve/reject/execute） |
| 角色权限 | `rbac_role_permissions` | `direct` / `inherited` 来源标记 |
| 约束 | `rbac_constraints` + `rbac_constraint_role_associations` | 静态/动态职责分离、先决角色、基数约束 |
| 接口 | `rbac_users.py` / `rbac_roles.py` / `rbac_permissions.py` / `rbac_constraints.py` | v2 API |
| 前端 | `RBACUserManage.vue` / `RBACRoleManage.vue` / `RBACPermissionManage.vue` / `RBACConstraintManage.vue` | RBAC3 管理页 |

### 1.3 核心变化

1. **权限表达式**：从扁平字符串 `permission_key` 变为标准化的 `{name}:{operation}:{read|write}` 或 `{name}:read` 格式。权限类型根据 key 格式自动推断，无需在数据库中存储显式 type 字段。
2. **角色不再只一个**：用户可同时拥有多个角色，会话中可激活子集。
3. **继承显式化**：子角色自动获得父角色权限，权限卡片中可区分"继承"与"自定义"。
4. **约束引擎**：新增互斥角色、先决角色、角色成员数量限制。
5. **权限中台化**：`require_permission` 统一走 RBAC3 服务，不再读取旧 `RolePermission`。
6. **权限描述**：每个权限枚举均需配置 `description`，前端权限管理界面展示权限描述信息。

---

## 2. 数据库模型设计

### 2.1 表结构

```python
# backend/app/models/rbac_role.py
class RBACRole(Base):
    __tablename__ = "rbac_roles"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    name = Column(String(50), unique=True, nullable=False)          # 英文标识
    display_name = Column(String(100), nullable=False)              # 中文展示
    description = Column(Text, nullable=True)
    role_type = Column(String(20), default="other")                 # admin / other
    is_super_admin = Column(Boolean, default=False)
    is_builtin = Column(Boolean, default=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
```

```python
# backend/app/models/rbac_resource.py

def _infer_resource_type(key: str) -> str:
    """根据权限 key 格式自动推断资源类型：
       - {name}:read                → page（页面级权限）
       - {name}:{operation}:{read|write} → action（操作级权限）
    """
    parts = key.split(":")
    if len(parts) == 2 and parts[1] in ("read", "write"):
        return "page"
    return "action"


class RBACResource(Base):
    __tablename__ = "rbac_resources"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    key = Column(String(100), unique=True, nullable=False)          # 权限 key，如 dashboard:read / content:create:write
    name = Column(String(100), nullable=False)                      # 中文显示名称
    description = Column(Text, nullable=True)                       # 权限描述说明
    parent_id = Column(String(36), ForeignKey("rbac_resources.id"), nullable=True)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)

    @property
    def type(self) -> str:
        """从 key 格式推断类型，不再存储为数据库字段"""
        return _infer_resource_type(self.key)
```

```python
# backend/app/models/rbac_permission.py
class RBACPermission(Base):
    __tablename__ = "rbac_permissions"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    resource_id = Column(String(36), ForeignKey("rbac_resources.id"), nullable=False)
    operation = Column(String(50), nullable=False)                  # read / write / create / update / delete / approve / reject / execute
    key = Column(String(150), unique=True, nullable=False)          # {name}:{operation}:{read|write} 或 {name}:read
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
```

```python
# backend/app/models/rbac_role_hierarchy.py
class RBACRoleHierarchy(Base):
    __tablename__ = "rbac_role_hierarchy"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    parent_role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    child_role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
```

```python
# backend/app/models/rbac_role_permission.py
class RBACRolePermission(Base):
    __tablename__ = "rbac_role_permissions"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    permission_id = Column(String(36), ForeignKey("rbac_permissions.id"), nullable=False)
    grant_type = Column(String(20), default="direct")               # direct / inherited
    created_at = Column(DateTime, default=datetime.utcnow)
```

```python
# backend/app/models/rbac_user_role_assignment.py
class RBACUserRoleAssignment(Base):
    __tablename__ = "rbac_user_role_assignments"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id = Column(String(36), ForeignKey("users.id"), nullable=False)
    role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    grant_type = Column(String(20), default="direct")               # direct / inherited
    valid_from = Column(DateTime, nullable=True)
    valid_until = Column(DateTime, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
```

```python
# backend/app/models/rbac_constraint.py
class RBACConstraint(Base):
    __tablename__ = "rbac_constraints"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    name = Column(String(100), nullable=False)
    description = Column(Text, nullable=True)
    constraint_type = Column(String(50), nullable=False)            # mutual_exclusive / prerequisite / cardinality
    config = Column(JSON, default=dict)                             # 类型相关配置
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)

class RBACConstraintRoleAssociation(Base):
    __tablename__ = "rbac_constraint_role_associations"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    constraint_id = Column(String(36), ForeignKey("rbac_constraints.id"), nullable=False)
    role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    association_type = Column(String(50), nullable=False)           # subject / target / prerequisite
```

### 2.2 待清理的旧表

- `custom_roles`
- `role_permissions`
- `user_permissions`

清理时机：数据迁移完成后，在 `main.py` 的 `Base.metadata.drop_all/create_all` 周期中不再包含上述模型，或直接写 Alembic 迁移删除。

---

## 3. 权限计算服务

### 3.1 核心算法

`backend/app/services/rbac_service.py`：

```python
READ_OPERATIONS = {"read", "execute"}
WRITE_OPERATIONS = {"create", "update", "delete", "approve", "reject", "execute"}

async def get_role_ancestors(role_id: str, db: AsyncSession) -> list[RBACRole]:
    """返回该角色的所有父角色（沿 hierarchy 向上）。"""
    ...

async def get_role_descendants(role_id: str, db: AsyncSession) -> list[RBACRole]:
    """返回该角色的所有子角色（沿 hierarchy 向下），用于 DAG 校验。"""
    ...

async def get_user_effective_permissions(
    user_id: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> dict[str, PermissionAccess]:
    """
    1. 超级管理员 -> 所有权限全开（不受自定义权限约束）。
    2. 查询用户已分配角色。
    3. 若 active_role_ids 提供，则取交集（会话级授权）。
    4. 对每个角色取祖先闭包，聚合角色权限（role_access_map）。
    5. 查询用户自定义权限覆盖（user_permission_overrides）。
    6. 无自定义覆盖 → 返回全部角色权限。
    7. 有自定义覆盖 → 取交集：有效 = 角色权限 ∩ {key | override.granted=true}。
    """
    ...

async def flatten_effective_permissions(
    effective: dict[str, PermissionAccess],
    db: AsyncSession,
) -> dict[str, str]:
    """
    将 PermissionAccess 展平为 { key: name } 格式。
    - read/write 权限分别展开为独立的 key
    - write 权限隐含授予对应 read 权限和父页面 read 权限
    """
    ...

async def has_permission(
    user_id: str,
    permission_key: str,
    mode: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> bool:
    ...
```

### 3.2 约束校验

`backend/app/services/rbac_constraint_service.py`：

```python
async def validate_user_role_assignments(
    user_id: str,
    proposed_role_ids: set[str],
    db: AsyncSession,
) -> list[str]:
    """返回所有违反约束的说明；空列表表示通过。"""
    ...

async def validate_role_hierarchy(
    parent_role_id: str,
    child_role_id: str,
    db: AsyncSession,
) -> bool:
    """禁止成环：parent 不能是 child 的后代，也不能等于 child。"""
    ...
```

约束类型：

| 类型 | config 示例 | 说明 |
|------|-------------|------|
| `mutual_exclusive` | `{"scope": "static"}` | 同一用户不能同时拥有两个互斥角色 |
| `prerequisite` | `{"require_all": true}` | 拥有目标角色前必须先拥有先决角色 |
| `cardinality` | `{"max_users": 3}` | 某角色最多分配用户数 |

---

## 4. 后端接口设计

### 4.1 用户与角色分配（`rbac_users.py`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v2/users` | 用户列表（含角色分配） |
| POST | `/api/v2/users` | 创建用户，无角色时按 `User.role` 自动同步 |
| PUT | `/api/v2/users/{id}` | 更新用户，支持修改角色分配 |
| PUT | `/api/v2/users/{id}/password` | 修改密码 |
| PUT | `/api/v2/users/{id}/roles` | 分配/替换 RBAC3 角色 |
| GET | `/api/v2/users/{id}/permissions` | 用户有效权限 |

### 4.2 角色与继承（`rbac_roles.py`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v2/roles` | 角色列表 |
| POST | `/api/v2/roles` | 创建角色 |
| PUT | `/api/v2/roles/{id}` | 更新角色 |
| DELETE | `/api/v2/roles/{id}` | 删除角色 |
| POST | `/api/v2/roles/{id}/parents` | 添加父角色 |
| DELETE | `/api/v2/roles/{id}/parents/{parent_id}` | 移除父角色 |
| GET | `/api/v2/roles/{id}/permissions` | 有效权限（返回 `{ key: name }` 格式，含继承） |
| GET | `/api/v2/roles/{id}/permissions/direct` | 直接权限 |
| GET | `/api/v2/roles/{id}/permissions/detail` | 继承 + 直接权限（带 grant_type） |
| PUT | `/api/v2/roles/{id}/permissions` | 更新直接权限 |

### 4.3 资源与权限（`rbac_permissions.py`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v2/resources` | 资源树（含 `description` 字段，类型由 key 推断） |
| GET | `/api/v2/permissions` | 权限列表 |
| GET | `/api/v2/permission-enums` | 权限枚举列表（key + name + description） |
| GET | `/api/v2/permission-pages` | 页面权限枚举（根据 key 最后 segment 为 `read` 筛选） |
| POST | `/api/v2/resources` | 新增资源（key + name + description，key 格式校验） |
| PUT | `/api/v2/resources/{id}` | 更新资源（支持修改 name、key、description） |
| DELETE | `/api/v2/resources/{id}` | 删除资源及其关联权限 |
| POST | `/api/v2/permissions` | 新增权限（key 格式校验） |
| PUT | `/api/v2/permissions/{id}` | 更新权限 |
| DELETE | `/api/v2/permissions/{id}` | 删除权限 |

**权限 key 格式规范：**

| 类型 | 格式 | 示例 | 说明 |
|------|------|------|------|
| 页面权限 | `{name}:read` | `dashboard:read`、`content:read` | 2 段式，控制页面/菜单可见性 |
| 操作权限 | `{name}:{operation}:{read|write}` | `content:create:write`、`users:update:read` | 3 段式，控制具体操作 |

**权限 key 校验规则（`_is_valid_permission_key`）：**
- 2 段式：两段均非空，第一段不含 `read`/`write`
- 3 段式：三段均非空，第三段为 `read` 或 `write`，第一二段不含 `read`/`write`

**页面权限筛选（`_is_page_permission_key`）：**
- 2 段式且第二段为 `read` 或 `write` 即为页面权限
- 前端路由守卫通过 `route.meta.permKey` 配置对应的页面权限 key

**资源类型推断（`_infer_resource_type`）：**
- `{name}:read` 或 `{name}:write` → `"page"`
- 其它格式 → `"action"`

### 4.4 约束（`rbac_constraints.py`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v2/constraints` | 约束列表 |
| POST | `/api/v2/constraints` | 创建约束 |
| PUT | `/api/v2/constraints/{id}` | 更新约束 |
| DELETE | `/api/v2/constraints/{id}` | 删除约束 |

### 4.5 认证接口（`auth.py`）

`GET /api/auth/me` 返回当前用户信息、角色和项目全部权限枚举：

```json
{
  "id": "1",
  "username": "admin",
  "nickname": "超级管理员",
  "role": "admin",
  "roles": [
    { "id": "role-uuid-1", "name": "admin", "display_name": "超级管理员" }
  ],
  "permissions": {
    "dashboard:read": "仪表盘",
    "content:read": "内容列表",
    "users:create:write": "创建用户"
  }
}
```

- `roles`：用户当前分配的角色列表，前端根据 `role.id` 调用 `/api/v2/roles/{id}/permissions` 获取角色权限
- `permissions`：项目所有权限枚举 `{ key: name }`，用于前端展示权限名称

用户有效权限由 `/api/v2/me/permissions` 获取（返回 `{ key: name }`），后端已内置**交集计算**：
- 无自定义权限覆盖 → 返回角色权限的全部
- 有自定义权限覆盖 → 返回角色权限 ∩ 自定义权限（仅两者共有的权限生效）

前端 `permissionStore.loadPermissions()` 直接调用 `/api/v2/me/permissions` 获取最终有效权限。

### 4.6 依赖注入

`backend/app/core/deps.py`：

```python
def require_permission(permission_key: str, mode: str = "read"):
    async def checker(
        current_user: User = Depends(get_current_user),
        db: AsyncSession = Depends(get_db),
    ):
        if await has_permission(current_user.id, permission_key, mode, db):
            return current_user
        raise HTTPException(status_code=403, detail="无权限")
    return checker
```

所有旧路由统一替换为 `require_permission("resource:operation", "read"/"write")`。

---

## 5. 初始化与数据迁移

### 5.1 初始化服务

`backend/app/services/rbac_init_service.py`：

```python
async def init_rbac_system(db: AsyncSession):
    """应用启动时调用：创建系统角色、资源、权限，并同步旧用户角色。"""
    await _ensure_builtin_roles(db)
    await _ensure_resources_and_permissions(db)
    await _ensure_builtin_role_permissions(db)
    await sync_user_role_assignments(db)

async def sync_user_role_assignments(db: AsyncSession):
    """将 users.role 同步为 rbac_user_role_assignments。"""
    result = await db.execute(select(User.id, User.role))
    for user_id, role_name in result.all():
        role = await db.execute(select(RBACRole).where(RBACRole.name == role_name))
        role_obj = role.scalar_one_or_none()
        if role_obj:
            existing = await db.execute(
                select(RBACUserRoleAssignment).where(
                    RBACUserRoleAssignment.user_id == user_id,
                    RBACUserRoleAssignment.role_id == role_obj.id,
                )
            )
            if not existing.scalar_one_or_none():
                db.add(RBACUserRoleAssignment(user_id=user_id, role_id=role_obj.id))
```

### 5.2 内置角色映射

| 旧角色 | RBAC3 角色 | 父角色 |
|--------|-----------|--------|
| `admin` | 超级管理员 | 无 |
| `manager` | 管理员 | 无 |
| `operator` | 运营者 | 无 |
| `reviewer` | 审核员 | 无 |

初始不预设父子关系，管理员后续可在「角色设置」中配置。

### 5.3 资源权限字典

`RBAC_RESOURCES` 在 `backend/app/services/rbac_init_service.py` 中定义，每个资源包含 `key`、`name` 和 `description`，权限类型由 key 格式自动推断：

| key | name | description | 类型 |
|-----|------|-------------|------|
| `dashboard:read` | 仪表盘 | 系统首页仪表盘，展示核心数据概览 | 页面 |
| `platforms:read` | 平台管理 | 管理已接入的第三方内容平台 | 页面 |
| `content:read` | 内容列表 | 查看和管理所有内容列表 | 页面 |
| `publish:read` | 发布管理 | 管理内容发布任务和发布计划 | 页面 |
| `templates:read` | 模板管理 | 管理内容创作模板 | 页面 |
| `review:read` | 内容审核 | 管理内容审核流程 | 页面 |
| `sql_review:read` | SQL审核 | 管理SQL变更审核流程 | 页面 |
| `accounts:read` | 平台账号 | 管理各平台的登录账号信息 | 页面 |
| `token_plan:read` | Token方案 | 管理API Token用量方案 | 页面 |
| `api_docs:read` | API文档 | 查看系统API接口文档 | 页面 |
| `db:read` | 数据库控制台 | 访问数据库控制台 | 页面 |
| `users:read` | 用户管理 | 管理系统用户列表和基本信息 | 页面 |
| `permissions:read` | 权限管理 | 管理角色权限分配 | 页面 |
| `roles:read` | 角色管理 | 管理角色定义和角色继承关系 | 页面 |
| `constraints:read` | 约束管理 | 管理职责分离约束规则 | 页面 |
| `permissions:manage:write` | 维护权限字典 | 创建、编辑、删除权限资源定义 | 操作 |
| `roles:manage:write` | 维护角色 | 创建、编辑、删除角色定义和层级关系 | 操作 |
| `constraints:manage:write` | 维护约束 | 创建、编辑、删除职责分离约束规则 | 操作 |
| `content:create:write` | 创建内容 | 创建新的内容条目 | 操作 |
| `content:update:write` | 编辑内容 | 编辑已有内容条目的标题、正文等信息 | 操作 |
| `content:delete:write` | 删除内容 | 删除已有内容条目 | 操作 |
| `content:ai_generate:write` | AI生成内容 | 使用AI辅助生成内容 | 操作 |
| `publish:create:write` | 创建发布 | 创建内容发布任务 | 操作 |
| `publish:retry:write` | 重试发布 | 重新执行失败的发布任务 | 操作 |
| `templates:create:write` | 创建模板 | 创建新的内容模板 | 操作 |
| `templates:update:write` | 编辑模板 | 编辑已有的内容模板 | 操作 |
| `templates:delete:write` | 删除模板 | 删除已有的内容模板 | 操作 |
| `review:submit:write` | 提交审核 | 将内容提交至审核流程 | 操作 |
| `review:approve:write` | 通过审核 | 批准待审核的内容 | 操作 |
| `review:reject:write` | 驳回审核 | 驳回审核不通过的内容 | 操作 |
| `db:execute:write` | 执行SQL | 在数据库控制台中执行SQL语句 | 操作 |
| `users:create:write` | 创建用户 | 创建新的系统用户 | 操作 |
| `users:update:read` | 查看用户 | 查看用户详细信息 | 操作 |
| `users:update:write` | 编辑用户 | 编辑用户的昵称、邮箱等基本信息 | 操作 |
| `users:delete:write` | 删除用户 | 删除系统用户 | 操作 |
| `users:change_password:write` | 修改用户密码 | 修改用户的登录密码 | 操作 |
| `users:custom_permissions:write` | 自定义用户权限 | 为个别用户配置自定义权限覆盖 | 操作 |
| `account:view:read` | 查看平台账号 | 查看平台账号的详细信息 | 操作 |
| `account:create:write` | 创建平台账号 | 创建新的平台登录账号 | 操作 |
| `account:update:write` | 编辑平台账号 | 编辑平台账号信息 | 操作 |
| `account:delete:write` | 删除平台账号 | 删除平台登录账号 | 操作 |
| `account:check:write` | 校验平台账号 | 校验平台账号的有效性 | 操作 |
| `db_change:submit:write` | 提交SQL变更 | 提交SQL变更申请 | 操作 |
| `db_change:approve:write` | 通过SQL变更 | 批准SQL变更申请 | 操作 |
| `db_change:reject:write` | 驳回SQL变更 | 驳回SQL变更申请 | 操作 |
| `model_config:create:write` | 创建模型配置 | 创建新的AI模型配置 | 操作 |
| `model_config:update:write` | 编辑模型配置 | 编辑AI模型配置参数 | 操作 |
| `model_config:delete:write` | 删除模型配置 | 删除AI模型配置 | 操作 |
| `db_history:view:read` | 查看SQL历史 | 查看SQL执行历史记录 | 操作 |

每个权限资源对应一条 `rbac_resources` 记录和一条 `rbac_permissions` 记录。operation 从 key 中提取：2 段式取第二段，3 段式取中间段。

### 5.4 旧权限数据迁移

启动时：
1. 读取 `RolePermission` 表。
2. 找到对应 `RBACRole`（按 `role` 名称匹配）和 `RBACPermission`（按 `permission_key` 匹配）。
3. 若 `can_read` 或 `can_write` 为 true，创建 `RBACRolePermission` 直接授权记录。
4. 迁移完成后标记旧表待清理。

---

## 6. 前端设计

### 6.1 权限状态（`frontend/src/stores/permission.ts`）

```typescript
export const usePermissionStore = defineStore('permission', () => {
  const permissions = ref<Record<string, string>>({})

  function hasPermission(key: string, _mode: 'read' | 'write' = 'read'): boolean {
    return key in permissions.value
  }

  async function loadPermissions(userId?: string) {
    const { data } = await api.get<Record<string, string>>('/v2/me/permissions')
    permissions.value = data || {}
    lastPermissionsUserId.value = userId ?? null
  }

  return { permissions, hasPermission, loadPermissions, clearPermissions }
})
```

权限数据流：
1. `/api/auth/me` 返回 `roles`（用户角色列表）和 `permissions`（项目全部权限枚举 `{key: name}`）
2. 前端调用 `/api/v2/me/permissions` 获取用户最终有效权限 `{key: name}`
3. 后端在 `get_user_effective_permissions` 中完成交集计算：角色权限 ∩ 自定义权限覆盖
4. `hasPermission(key)` 检查 `key` 是否在有效权限集合中

### 6.2 路由守卫（`frontend/src/router/index.ts`）

```typescript
router.beforeEach(async (to) => {
  const userStore = useUserStore()
  const permStore = usePermissionStore()

  if (to.meta.public) return true
  if (!userStore.token) return '/login'

  if (!userStore.userInfo) await userStore.fetchUserInfo()
  if (Object.keys(permStore.permissions).length === 0) {
    await permStore.loadPermissions()
  }

  if (to.meta.skipPermCheck) return true
  const key = to.meta.permKey as string
  if (key && !permStore.hasPermission(key, 'read')) return '/403'

  return true
})
```

### 6.3 指令（`frontend/src/directives/permission.ts`）

```typescript
const vPerm = {
  mounted(el: HTMLElement, binding: DirectiveBinding<{ key: string; mode?: 'read' | 'write' }>) {
    const permStore = usePermissionStore()
    const { key, mode = 'read' } = binding.value
    if (!permStore.hasPermission(key, mode)) {
      el.style.display = 'none'
    }
  },
}
export default vPerm
```

注册：

```typescript
app.directive('perm', vPerm)
```

### 6.4 权限类型判断（前端通用工具函数）

```typescript
/**
 * 根据权限 key 判断是否为页面权限：
 * - {name}:read 或 {name}:write → 页面权限（2段式）
 * - {name}:{operation}:{read|write} → 操作权限（3段式）
 */
function _isPageKey(key: string): boolean {
  const parts = key.split(':')
  return parts.length === 2 && (parts[1] === 'read' || parts[1] === 'write')
}
```

所有前端组件（`RBACPermissionManage.vue`、`RBACUserPermissionCustomize.vue`、`RBACPermissionEnumManage.vue`、`ResourcePermissionCard.vue`）均使用此函数判断权限类型，不再依赖后端返回的 `type` 字段。

### 6.5 侧边栏（`AppSidebar.vue`）

菜单项统一使用 `permissionStore.hasPermission(permKey, 'read')` 过滤；不再使用 `userStore.userInfo.role` 硬编码。

### 6.6 管理页面

| 页面 | 路径 | 核心功能 |
|------|------|----------|
| `RBACUserManage.vue` | `/rbac/users` | 用户列表、创建、编辑、角色分配 |
| `RBACRoleManage.vue` | `/rbac/roles` | 角色 CRUD、父子关系、约束提示 |
| `RBACPermissionManage.vue` | `/rbac/permissions` | 按资源树配置角色权限，区分继承/直接，显示权限描述 |
| `RBACPermissionEnumManage.vue` | `/rbac/permission-enum` | 超级管理员专用：权限枚举 CRUD（key、名称、描述编辑） |
| `RBACUserPermissionCustomize.vue` | `/rbac/user-permissions` | 用户级自定义权限覆盖 |
| `RBACConstraintManage.vue` | `/rbac/constraints` | 互斥/先决/基数约束管理 |

权限描述展示与编辑：
- 权限枚举列表中，每个权限显示 `description` 字段
- 超管可在 `RBACPermissionEnumManage.vue` 中编辑权限的 key、名称和描述
- 创建新权限时需填写 key、名称、描述三个字段
- 权限卡片组件 `ResourcePermissionCard.vue` 中使用 `_isPageKey()` 判断权限图标

---

## 7. 文件结构

```
backend/app/
├── core/
│   └── deps.py                    # require_permission 切到 RBAC3
├── models/
│   ├── rbac_role.py
│   ├── rbac_resource.py           # 移除 type 字段，添加 _infer_resource_type()、description
│   ├── rbac_permission.py
│   ├── rbac_role_hierarchy.py
│   ├── rbac_role_permission.py
│   ├── rbac_user_role_assignment.py
│   ├── rbac_user_permission_override.py
│   ├── rbac_constraint.py
│   ├── rbac_constraint_role_association.py
│   └── user.py                    # 保留，role 字段仅作标签/迁移用
├── routers/
│   ├── rbac_users.py
│   ├── rbac_roles.py
│   ├── rbac_permissions.py        # 新增 description、权限 key 校验、页面权限筛选
│   └── rbac_constraints.py
├── services/
│   ├── rbac_service.py
│   ├── rbac_constraint_service.py
│   └── rbac_init_service.py      # RBAC_RESOURCES 含 description，type 由 key 推断
frontend/src/
├── stores/
│   └── permission.ts             # 权限状态管理，权限为 {open, name} 结构
├── directives/
│   └── permission.ts
├── router/
│   └── index.ts
├── components/layout/
│   └── AppSidebar.vue
├── components/rbac/
│   └── ResourcePermissionCard.vue # 使用 _isPageKey() 判断权限图标
└── pages/
    ├── RBACUserManage.vue
    ├── RBACUserCreate.vue
    ├── RBACUserEdit.vue
    ├── RBACUserPassword.vue
    ├── RBACRoleManage.vue
    ├── RBACPermissionManage.vue
    ├── RBACPermissionEnumManage.vue  # 权限枚举管理（含 description 编辑）
    ├── RBACUserPermissionCustomize.vue
    └── RBACConstraintManage.vue
```

---

## 8. 实施任务清单

按模块、可独立验证的方式拆分。

### Phase 1：数据库与核心服务

- [x] **Task 1.1**：创建 RBAC3 模型文件并注册到 `models/__init__.py`。
- [x] **Task 1.2**：实现 `rbac_service.py`（继承、有效权限、has_permission）。
- [x] **Task 1.3**：实现 `rbac_constraint_service.py`（DAG、SoD、先决、基数）。
- [x] **Task 1.4**：实现 `rbac_init_service.py`（资源/权限/角色种子 + 旧角色同步）。
- [x] **Task 1.5**：更新 `deps.py` 的 `require_permission`。

### Phase 2：后端接口

- [x] **Task 2.1**：创建 `rbac_permissions.py`（资源/权限树）。
- [x] **Task 2.2**：创建 `rbac_roles.py`（CRUD + 继承 + 权限配置 + detail 接口）。
- [x] **Task 2.3**：创建 `rbac_users.py`（CRUD + 角色分配 + 用户创建审核）。
- [x] **Task 2.4**：创建 `rbac_constraints.py`。
- [x] **Task 2.5**：更新 `auth.py` 的 `me` 接口返回 RBAC3 权限。
- [x] **Task 2.6**：在 `main.py` 注册新路由、移除旧路由。
- [x] **Task 2.7**：将其他业务路由（accounts、contents 等）的旧权限校验替换为 `require_permission("resource:operation")`。

### Phase 3：前端基础设施

- [x] **Task 3.1**：创建 `stores/permission.ts`。
- [x] **Task 3.2**：创建 `directives/permission.ts`。
- [x] **Task 3.3**：更新 `router/index.ts` 守卫。
- [x] **Task 3.4**：更新 `AppSidebar.vue` 动态菜单。

### Phase 4：前端管理页

- [x] **Task 4.1**：`RBACPermissionManage.vue`（资源树 + 角色权限配置，继承/直接区分）。
- [x] **Task 4.2**：`RBACRoleManage.vue`（角色 CRUD + 父子关系 + 约束提示）。
- [x] **Task 4.3**：`RBACUserManage.vue`（用户 CRUD + 角色分配）。
- [x] **Task 4.4**：`RBACConstraintManage.vue`。

### Phase 5：清理与迁移

- [x] **Task 5.1**：删除旧模型文件（确认无引用后）。`custom_roles`、`role_permissions`、`user_permissions` 已移除。
- [x] **Task 5.2**：删除旧前端页面。`Accounts.vue`、`RoleManage.vue`、`PermissionManage.vue` 已移除。
- [x] **Task 5.3**：删除旧路由文件。`users.py`、`roles.py`、`permissions.py` 已移除。
- [ ] **Task 5.4**：重置数据库验证完整流程（管理员/运营者/审核员登录、菜单、权限配置）。

---

## 9. 验收标准

1. 管理员登录后能看到「权限管理」分组下的账号设置、角色设置、权限设置、职责分离。
2. 运营者登录后看不到「权限管理」分组。
3. 在「角色设置」中创建一个子角色并添加父角色后，子角色自动继承父角色权限，权限卡片显示「继承自父角色」且不可取消。
4. 给子角色额外勾选自定义权限，保存后只保留直接权限。
5. 创建一个互斥约束后，给同一用户分配两个互斥角色时接口返回 400。
6. 所有业务接口（内容、发布、账号等）均通过 `require_permission` 校验。
7. 前端 `npx vue-tsc -b --noEmit` 无类型错误。
8. 后端启动时自动完成旧数据迁移，旧 `RolePermission` 数据不丢失。

---

## 10. 风险与注意点

1. **旧用户角色同步**：`users.role` 字段保留，仅作为初始角色分配来源；后续以 `rbac_user_role_assignments` 为准。
2. **权限键映射**：部分旧 key（如 `db:history:read`）拆分为 `db_history:view:read` 资源，需要在初始化服务中显式映射。
3. **前端硬编码 role 检查**：`AppSidebar.vue` 和页面中所有 `userInfo.role === 'admin'` 都要替换为权限检查。
4. **会话激活**：第一版默认激活用户所有角色；如需会话级角色切换，后续扩展 `active_role_ids` 参数。
5. **约束性能**：互斥约束在用户角色分配时校验，角色继承扩展后用户数大时需注意查询性能（可缓存）。
6. **权限类型推断**：权限类型不再存储在数据库中，通过 `_infer_resource_type(key)` 函数根据 key 格式动态计算。确保所有新增权限遵循 `{name}:{operation}:{read|write}` 命名规范。
7. **权限描述必填**：`RBAC_RESOURCES` 中每条记录需填写 `description`，前端权限管理界面依赖此字段展示。系统启动时 `_ensure_resources_and_permissions()` 自动将描述写入数据库。

---

## 11. 附录：权限枚举完整对照表

### 11.1 页面权限（2段式 `{name}:read`）

| 权限 key | 名称 | 描述 |
|----------|------|------|
| `dashboard:read` | 仪表盘 | 系统首页仪表盘，展示核心数据概览 |
| `platforms:read` | 平台管理 | 管理已接入的第三方内容平台 |
| `content:read` | 内容列表 | 查看和管理所有内容列表 |
| `publish:read` | 发布管理 | 管理内容发布任务和发布计划 |
| `templates:read` | 模板管理 | 管理内容创作模板 |
| `review:read` | 内容审核 | 管理内容审核流程 |
| `sql_review:read` | SQL审核 | 管理SQL变更审核流程 |
| `accounts:read` | 平台账号 | 管理各平台的登录账号信息 |
| `token_plan:read` | Token方案 | 管理API Token用量方案 |
| `api_docs:read` | API文档 | 查看系统API接口文档 |
| `db:read` | 数据库控制台 | 访问数据库控制台 |
| `users:read` | 用户管理 | 管理系统用户列表和基本信息 |
| `permissions:read` | 权限管理 | 管理角色权限分配 |
| `roles:read` | 角色管理 | 管理角色定义和角色继承关系 |
| `constraints:read` | 约束管理 | 管理职责分离约束规则 |

### 11.2 操作权限（3段式 `{name}:{operation}:{read|write}`）

| 权限 key | 名称 | 描述 |
|----------|------|------|
| `permissions:manage:write` | 维护权限字典 | 创建、编辑、删除权限资源定义 |
| `roles:manage:write` | 维护角色 | 创建、编辑、删除角色定义和层级关系 |
| `constraints:manage:write` | 维护约束 | 创建、编辑、删除职责分离约束规则 |
| `content:create:write` | 创建内容 | 创建新的内容条目 |
| `content:update:write` | 编辑内容 | 编辑已有内容条目的标题、正文等信息 |
| `content:delete:write` | 删除内容 | 删除已有内容条目 |
| `content:ai_generate:write` | AI生成内容 | 使用AI辅助生成内容 |
| `publish:create:write` | 创建发布 | 创建内容发布任务 |
| `publish:retry:write` | 重试发布 | 重新执行失败的发布任务 |
| `templates:create:write` | 创建模板 | 创建新的内容模板 |
| `templates:update:write` | 编辑模板 | 编辑已有的内容模板 |
| `templates:delete:write` | 删除模板 | 删除已有的内容模板 |
| `review:submit:write` | 提交审核 | 将内容提交至审核流程 |
| `review:approve:write` | 通过审核 | 批准待审核的内容 |
| `review:reject:write` | 驳回审核 | 驳回审核不通过的内容 |
| `db:execute:write` | 执行SQL | 在数据库控制台中执行SQL语句 |
| `users:create:write` | 创建用户 | 创建新的系统用户 |
| `users:update:read` | 查看用户 | 查看用户详细信息 |
| `users:update:write` | 编辑用户 | 编辑用户的昵称、邮箱等基本信息 |
| `users:delete:write` | 删除用户 | 删除系统用户 |
| `users:change_password:write` | 修改用户密码 | 修改用户的登录密码 |
| `users:custom_permissions:write` | 自定义用户权限 | 为个别用户配置自定义权限覆盖 |
| `account:view:read` | 查看平台账号 | 查看平台账号的详细信息 |
| `account:create:write` | 创建平台账号 | 创建新的平台登录账号 |
| `account:update:write` | 编辑平台账号 | 编辑平台账号信息 |
| `account:delete:write` | 删除平台账号 | 删除平台登录账号 |
| `account:check:write` | 校验平台账号 | 校验平台账号的有效性 |
| `db_change:submit:write` | 提交SQL变更 | 提交SQL变更申请 |
| `db_change:approve:write` | 通过SQL变更 | 批准SQL变更申请 |
| `db_change:reject:write` | 驳回SQL变更 | 驳回SQL变更申请 |
| `model_config:create:write` | 创建模型配置 | 创建新的AI模型配置 |
| `model_config:update:write` | 编辑模型配置 | 编辑AI模型配置参数 |
| `model_config:delete:write` | 删除模型配置 | 删除AI模型配置 |
| `db_history:view:read` | 查看SQL历史 | 查看SQL执行历史记录 |

### 11.3 旧权限键迁移对照

| 旧权限键 | RBAC3 权限键 | 说明 |
|----------|--------------|------|
| `dashboard` | `dashboard:read` | 仪表盘 |
| `platforms` | `platforms:read` | 平台管理 |
| `content` | `content:read` | 内容工坊 |
| `publish` | `publish:read` | 发布管理 |
| `templates` | `templates:read` | 模板中心 |
| `review` | `review:read` | 内容审核 |
| `sql_review` | `sql_review:read` | SQL 审核 |
| `accounts` | `users:read` | 账号设置页面 |
| `token_plan` | `token_plan:read` | Token 配置 |
| `api_docs` | `api_docs:read` | API 文档 |
| `database` | `db:read` | 数据库管理 |
| `permission_manage` | `permissions:read` | 权限设置页面 |
| `user:create` | `users:create:write` | 创建用户 |
| `user:update` | `users:update:write` | 编辑用户 |
| `user:delete` | `users:delete:write` | 删除用户 |
| `user:change_password` | `users:change_password:write` | 修改密码 |
| `content:create` | `content:create:write` | 创建内容 |
| `content:update` | `content:update:write` | 编辑内容 |
| `content:delete` | `content:delete:write` | 删除内容 |
| `content:ai_generate` | `content:ai_generate:write` | AI 生成 |
| `review:submit` | `review:submit:write` | 提交审核 |
| `review:approve` | `review:approve:write` | 审核通过 |
| `review:reject` | `review:reject:write` | 审核驳回 |
| `db_change:submit` | `db_change:submit:write` | 提交 SQL 变更 |
| `db_change:approve` | `db_change:approve:write` | SQL 变更通过 |
| `db_change:reject` | `db_change:reject:write` | SQL 变更驳回 |
| `template:create` | `templates:create:write` | 创建模板 |
| `template:update` | `templates:update:write` | 编辑模板 |
| `template:delete` | `templates:delete:write` | 删除模板 |
| `account:create` | `account:create:write` | 添加平台账号 |
| `account:update` | `account:update:write` | 编辑平台账号 |
| `account:delete` | `account:delete:write` | 删除平台账号 |
| `account:check` | `account:check:write` | 检测账号状态 |
| `publish:create` | `publish:create:write` | 创建发布任务 |
| `publish:retry` | `publish:retry:write` | 重试发布任务 |
| `model_config:create` | `model_config:create:write` | 创建模型配置 |
| `model_config:update` | `model_config:update:write` | 编辑模型配置 |
| `model_config:delete` | `model_config:delete:write` | 删除模型配置 |
| `db:execute` | `db:execute:write` | 执行 SQL |
| `db:history:read` | `db_history:view:read` | 查看 SQL 历史 |

**迁移规则**（`rbac_init_service.py` 中的 `_map_legacy_permission()`）：

1. `db:history:read` → `db_history:view:read`（特殊映射）
2. `db:execute` → `db:execute:write`（特殊映射）
3. `user:*` 系列 → `users:*:write`（统一加 `users:` 前缀和 `:write` 后缀）
4. `template:*` 系列 → `templates:*:write`（统一加 `templates:` 前缀和 `:write` 后缀）
5. 其他 2 段式：操作是 write 类型的（`create`/`update`/`delete`/`approve`/`reject`/`submit`/`retry`/`check`/`execute`/`ai_generate`/`change_password`/`custom_permissions`）→ 加 `:write` 后缀；操作是 `read` → 加 `:read` 后缀
6. 单段式：`permission_manage` → `permissions:read`；`database` → `db:read`；其他 → 加 `:read` 后缀

---

*文档版本：v1.3*
*最后更新：2026-07-30*
*变更说明：v1.3 - 根据项目实际代码更新实施任务清单，标记 Phase 1~4 及 Phase 5 清理任务为已完成；同步更新文件结构，移除已删除的旧文件；更新版本信息。*
*v1.2 - 重构权限接口数据格式：/api/auth/me 的 permissions 改为 { key: name } 返回全部权限枚举，用户有效权限改由 /api/v2/roles/{id}/permissions 获取（也返回 { key: name }）；前端权限 store 改为汇总多角色权限；更新 API 接口文档。*
*v1.1 - 移除 RBACResource.type 字段，改为基于 key 格式推断类型；添加 description 字段到权限枚举；更新前后端权限筛选逻辑；更新 API 接口文档。*
