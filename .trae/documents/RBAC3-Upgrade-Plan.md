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

1. **权限表达式**：从扁平字符串 `permission_key` 变为 `resource:operation`（例：`content:update`）。
2. **角色不再只一个**：用户可同时拥有多个角色，会话中可激活子集。
3. **继承显式化**：子角色自动获得父角色权限，权限卡片中可区分“继承”与“自定义”。
4. **约束引擎**：新增互斥角色、先决角色、角色成员数量限制。
5. **权限中台化**：`require_permission` 统一走 RBAC3 服务，不再读取旧 `RolePermission`。

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
class RBACResource(Base):
    __tablename__ = "rbac_resources"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    key = Column(String(100), unique=True, nullable=False)          # dashboard / content / account
    name = Column(String(100), nullable=False)
    description = Column(Text, nullable=True)
    type = Column(String(20), nullable=False)                       # page / action
    parent_id = Column(String(36), ForeignKey("rbac_resources.id"), nullable=True)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
```

```python
# backend/app/models/rbac_permission.py
class RBACPermission(Base):
    __tablename__ = "rbac_permissions"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    resource_id = Column(String(36), ForeignKey("rbac_resources.id"), nullable=False)
    operation = Column(String(50), nullable=False)                  # read / write / create / update / delete / approve / reject / execute
    key = Column(String(150), unique=True, nullable=False)          # resource:operation
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
    1. 超级管理员 -> 所有权限全开。
    2. 查询用户已分配角色。
    3. 若 active_role_ids 提供，则取交集（会话级授权）。
    4. 对每个角色取祖先闭包。
    5. 聚合所有权限的 read/write。
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
| GET | `/api/v2/roles/{id}/permissions` | 有效权限 |
| GET | `/api/v2/roles/{id}/permissions/direct` | 直接权限 |
| GET | `/api/v2/roles/{id}/permissions/detail` | 继承 + 直接权限（带 grant_type） |
| PUT | `/api/v2/roles/{id}/permissions` | 更新直接权限 |

### 4.3 资源与权限（`rbac_permissions.py`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v2/resources` | 资源树 |
| GET | `/api/v2/permissions` | 权限列表 |
| POST | `/api/v2/resources` | 新增资源 |
| POST | `/api/v2/permissions` | 新增权限 |

### 4.4 约束（`rbac_constraints.py`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v2/constraints` | 约束列表 |
| POST | `/api/v2/constraints` | 创建约束 |
| PUT | `/api/v2/constraints/{id}` | 更新约束 |
| DELETE | `/api/v2/constraints/{id}` | 删除约束 |

### 4.5 认证接口（`auth.py`）

`GET /api/auth/me` 返回当前用户有效权限映射，供前端渲染菜单与按钮：

```json
{
  "id": "1",
  "username": "admin",
  "nickname": "超级管理员",
  "roles": [{ "id": "...", "name": "admin", "display_name": "超级管理员" }],
  "permissions": {
    "dashboard": { "read": true, "write": true },
    "content:update": { "read": true, "write": true }
  }
}
```

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

### 5.3 资源权限映射

将旧 `ALL_PERMISSIONS` 映射到 `rbac_resources` + `rbac_permissions`：

| 旧 key | 资源 key | 操作 |
|--------|----------|------|
| `dashboard` | `dashboard` | `read` |
| `accounts` | `users` | `read` |
| `user:create` | `users` | `create` |
| `user:update` | `users` | `update` |
| `content:ai_generate` | `content` | `ai_generate` |
| `account:create` | `account` | `create` |
| `db:execute` | `db` | `execute` |
| `db:history:read` | `db_history` | `read` |

对于页面权限，默认同时生成 `read` 操作；对于操作权限，按原 key 解析为资源 + 操作。

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
  const permissions = ref<Record<string, { read: boolean; write: boolean }>>({})

  function hasPermission(key: string, mode: 'read' | 'write' = 'read'): boolean {
    const access = permissions.value[key]
    if (!access) return false
    return mode === 'read' ? access.read : access.write
  }

  async function loadPermissions() {
    const { data } = await api.get('/auth/me')
    permissions.value = data.permissions || {}
  }

  return { permissions, hasPermission, loadPermissions }
})
```

### 6.2 路由守卫（`frontend/src/router/index.ts`）

```typescript
router.beforeEach(async (to) => {
  const userStore = useUserStore()
  const permStore = usePermissionStore()

  if (to.meta.public) return true
  if (!userStore.token) return '/login'

  if (!userStore.userInfo) await userStore.fetchUserInfo()
  if (!permStore.permissions || Object.keys(permStore.permissions).length === 0) {
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

### 6.4 侧边栏（`AppSidebar.vue`）

菜单项统一使用 `permissionStore.hasPermission(permKey, 'read')` 过滤；不再使用 `userStore.userInfo.role` 硬编码。

分组结构：

- 内容管理
- 审核管理
- 平台管理
- 权限管理
  - 账号设置（`users:read`）
  - 角色设置（`roles:read`）
  - 权限设置（`permissions:read`）
  - 职责分离（`constraints:read`）
- 系统管理
  - Token 配置
  - API 文档
  - 数据库管理

### 6.5 管理页面

| 页面 | 路径 | 核心功能 |
|------|------|----------|
| `RBACUserManage.vue` | `/rbac/users` | 用户列表、创建、编辑、角色分配 |
| `RBACRoleManage.vue` | `/rbac/roles` | 角色 CRUD、父子关系、约束提示 |
| `RBACPermissionManage.vue` | `/rbac/permissions` | 按资源树配置角色权限，区分继承/直接 |
| `RBACConstraintManage.vue` | `/rbac/constraints` | 互斥/先决/基数约束管理 |

设计原则（调用 `frontend-design` 技能细化）：

- Apple HIG 风格：大圆角、柔和阴影、玻璃态面板。
- 响应式：左侧树/列表 + 右侧详情/卡片。
- 权限卡片使用颜色区分：蓝色=自定义，绿色=继承，灰色=未授权。
- 全选读/写仅作用于可配置（非继承）权限。

---

## 7. 文件结构

```
backend/app/
├── core/
│   └── deps.py                    # require_permission 切到 RBAC3
├── models/
│   ├── rbac_role.py
│   ├── rbac_resource.py
│   ├── rbac_permission.py
│   ├── rbac_role_hierarchy.py
│   ├── rbac_role_permission.py
│   ├── rbac_user_role_assignment.py
│   ├── rbac_constraint.py
│   ├── rbac_constraint_role_association.py
│   ├── user.py                    # 保留，role 字段仅作标签/迁移用
│   ├── custom_role.py             # 删除
│   ├── role_permission.py         # 删除
│   └── user_permission.py         # 删除
├── routers/
│   ├── rbac_users.py
│   ├── rbac_roles.py
│   ├── rbac_permissions.py
│   ├── rbac_constraints.py
│   ├── users.py                   # 删除
│   ├── roles.py                   # 删除
│   └── permissions.py             # 删除
├── services/
│   ├── rbac_service.py
│   ├── rbac_constraint_service.py
│   └── rbac_init_service.py
frontend/src/
├── stores/
│   └── permission.ts
├── directives/
│   └── permission.ts
├── router/
│   └── index.ts
├── components/layout/
│   └── AppSidebar.vue
└── pages/
    ├── RBACUserManage.vue
    ├── RBACRoleManage.vue
    ├── RBACPermissionManage.vue
    ├── RBACConstraintManage.vue
    ├── Accounts.vue               # 删除
    ├── RoleManage.vue             # 删除
    └── PermissionManage.vue       # 删除
```

---

## 8. 实施任务清单

按模块、可独立验证的方式拆分。

### Phase 1：数据库与核心服务

- [ ] **Task 1.1**：创建 RBAC3 模型文件并注册到 `models/__init__.py`。
- [ ] **Task 1.2**：实现 `rbac_service.py`（继承、有效权限、has_permission）。
- [ ] **Task 1.3**：实现 `rbac_constraint_service.py`（DAG、SoD、先决、基数）。
- [ ] **Task 1.4**：实现 `rbac_init_service.py`（资源/权限/角色种子 + 旧角色同步）。
- [ ] **Task 1.5**：更新 `deps.py` 的 `require_permission`。

### Phase 2：后端接口

- [ ] **Task 2.1**：创建 `rbac_permissions.py`（资源/权限树）。
- [ ] **Task 2.2**：创建 `rbac_roles.py`（CRUD + 继承 + 权限配置 + detail 接口）。
- [ ] **Task 2.3**：创建 `rbac_users.py`（CRUD + 角色分配 + 用户创建审核）。
- [ ] **Task 2.4**：创建 `rbac_constraints.py`。
- [ ] **Task 2.5**：更新 `auth.py` 的 `me` 接口返回 RBAC3 权限。
- [ ] **Task 2.6**：在 `main.py` 注册新路由、移除旧路由。
- [ ] **Task 2.7**：将其他业务路由（accounts、contents 等）的旧权限校验替换为 `require_permission("resource:operation")`。

### Phase 3：前端基础设施

- [ ] **Task 3.1**：创建 `stores/permission.ts`。
- [ ] **Task 3.2**：创建 `directives/permission.ts`。
- [ ] **Task 3.3**：更新 `router/index.ts` 守卫。
- [ ] **Task 3.4**：更新 `AppSidebar.vue` 动态菜单。

### Phase 4：前端管理页

- [ ] **Task 4.1**：`RBACPermissionManage.vue`（资源树 + 角色权限配置，继承/直接区分）。
- [ ] **Task 4.2**：`RBACRoleManage.vue`（角色 CRUD + 父子关系 + 约束提示）。
- [ ] **Task 4.3**：`RBACUserManage.vue`（用户 CRUD + 角色分配）。
- [ ] **Task 4.4**：`RBACConstraintManage.vue`。

### Phase 5：清理与迁移

- [ ] **Task 5.1**：删除旧模型文件（确认无引用后）。
- [ ] **Task 5.2**：删除旧前端页面。
- [ ] **Task 5.3**：删除旧路由文件。
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
2. **权限键映射**：部分旧 key（如 `db:history:read`）拆分为 `db_history:read` 资源，需要在初始化服务中显式映射。
3. **前端硬编码 role 检查**：`AppSidebar.vue` 和页面中所有 `userInfo.role === 'admin'` 都要替换为权限检查。
4. **会话激活**：第一版默认激活用户所有角色；如需会话级角色切换，后续扩展 `active_role_ids` 参数。
5. **约束性能**：互斥约束在用户角色分配时校验，角色继承扩展后用户数大时需注意查询性能（可缓存）。

---

## 11. 附录：关键权限键对照表

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
| `user:create` | `users:create` | 创建用户 |
| `user:update` | `users:update` | 编辑用户 |
| `user:delete` | `users:delete` | 删除用户 |
| `user:change_password` | `users:change_password` | 修改密码 |
| `content:create` | `content:create` | 创建内容 |
| `content:update` | `content:update` | 编辑内容 |
| `content:delete` | `content:delete` | 删除内容 |
| `content:ai_generate` | `content:ai_generate` | AI 生成 |
| `review:submit` | `review:submit` | 提交审核 |
| `review:approve` | `review:approve` | 审核通过 |
| `review:reject` | `review:reject` | 审核驳回 |
| `db_change:submit` | `db_change:submit` | 提交 SQL 变更 |
| `db_change:approve` | `db_change:approve` | SQL 变更通过 |
| `db_change:reject` | `db_change:reject` | SQL 变更驳回 |
| `template:create` | `templates:create` | 创建模板 |
| `template:update` | `templates:update` | 编辑模板 |
| `template:delete` | `templates:delete` | 删除模板 |
| `account:create` | `account:create` | 添加平台账号 |
| `account:update` | `account:update` | 编辑平台账号 |
| `account:delete` | `account:delete` | 删除平台账号 |
| `account:check` | `account:check` | 检测账号状态 |
| `publish:create` | `publish:create` | 创建发布任务 |
| `publish:retry` | `publish:retry` | 重试发布任务 |
| `model_config:create` | `model_config:create` | 创建模型配置 |
| `model_config:update` | `model_config:update` | 编辑模型配置 |
| `model_config:delete` | `model_config:delete` | 删除模型配置 |
| `db:execute` | `db:execute` | 执行 SQL |
| `db:history:read` | `db_history:read` | 查看 SQL 历史 |

---

*文档版本：v1.0*
*编写时间：2026-07-25*
