from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource
from app.models.rbac_role import RBACRole
from app.models.rbac_role_hierarchy import RBACRoleHierarchy
from app.models.rbac_role_permission import RBACRolePermission
from app.models.rbac_user_permission_override import RBACUserPermissionOverride
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment


@dataclass
class UserEffectivePermissions:
    """Complete effective permission bundle for a user.

    Intersection model:
      有效权限 = 角色权限(含继承) ∩ 自定义权限(granted=True)

    Fields:
      access_map: role permissions merged with role-hierarchy inheritance
                  (raw, NOT filtered by overrides).
      granted_keys: permission keys explicitly granted=True by the user via
                    RBACUserPermissionOverride. When has_overrides is False
                    this set is empty and callers should return the full
                    role permission set.
      has_overrides: whether any RBACUserPermissionOverride records exist.
                     When False, effective = role permissions (default).
                     When True, effective = role permissions ∩ granted_keys.
      is_super_admin: whether the user (or any role in the closure) is a
                      super admin. Super admins are not constrained by
                      user-level permission overrides.
    """

    access_map: dict[str, "PermissionAccess"]
    granted_keys: set[str]
    has_overrides: bool
    is_super_admin: bool


READ_OPERATIONS = {"read", "execute", "custom_permissions"}
WRITE_OPERATIONS = {"create", "update", "delete", "approve", "reject", "execute", "custom_permissions"}

# Actions whose write permission also grants view access to the corresponding page.
ACTION_PARENT_PAGE: dict[str, str] = {
    "content": "content:read",
    "publish": "publish:read",
    "templates": "templates:read",
    "review": "review:read",
    "db_change": "sql_review:read",
    "db": "db:read",
    "db_history": "db:read",
}


@dataclass
class PermissionAccess:
    read: bool = False
    write: bool = False


def _permission_scope(perm_key: str) -> str:
    parts = perm_key.split(":")
    if len(parts) >= 3:
        return parts[2]
    return parts[-1] if len(parts) == 2 else "read"


def resolve_read_keys(perm_key: str) -> list[str]:
    parts = perm_key.split(":")
    if len(parts) == 2:
        return [perm_key]
    if len(parts) == 3:
        scope = parts[2]
        if scope == "read":
            return [perm_key]
        return [perm_key, f"{parts[0]}:{parts[1]}:read"]
    return [perm_key]


def resolve_write_keys(perm_key: str) -> list[str]:
    parts = perm_key.split(":")
    if len(parts) == 2:
        return [perm_key]
    if len(parts) == 3:
        if parts[2] == "write":
            return [perm_key]
    return [perm_key]


def _implied_read_keys(perm_key: str) -> list[str]:
    """Extra read keys implicitly granted when an action write key is granted."""
    parts = perm_key.split(":")
    if len(parts) != 3 or parts[2] != "write":
        return []
    implied: list[str] = []
    if not (parts[1] == "read"):
        implied.append(f"{parts[0]}:{parts[1]}:read")
    parent = ACTION_PARENT_PAGE.get(parts[0])
    if parent:
        implied.append(parent)
    return implied


async def flatten_effective_permissions(
    effective: dict[str, PermissionAccess],
    db: AsyncSession,
) -> dict[str, str]:
    """Resolve role permissions to the flat key set (with derivation).

    Derivation:
      - resolve_read_keys: write → also produce the :read variant
      - resolve_write_keys: keep :write as-is
      - _implied_read_keys: action :write → parent page :read
    """
    flat: set[str] = set()
    for perm_key, access in effective.items():
        if access.read or access.write:
            for read_key in resolve_read_keys(perm_key):
                flat.add(read_key)
        if access.write:
            for write_key in resolve_write_keys(perm_key):
                flat.add(write_key)
            for implied in _implied_read_keys(perm_key):
                flat.add(implied)

    if not flat:
        return {}

    names = await get_permission_display_names(list(flat), db)
    result: dict[str, str] = {}
    for key in flat:
        result[key] = names.get(key, key)
    return result


async def get_permission_display_names(keys: list[str], db: AsyncSession) -> dict[str, str]:
    if not keys:
        return {}
    result = await db.execute(
        select(RBACPermission.key, RBACResource.name)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.key.in_(keys))
    )
    names: dict[str, str] = {}
    for key, name in result.all():
        parts = key.split(":")
        is_action = len(parts) == 3
        if is_action and parts[2] == "read":
            names[key] = f"查看{name}" if not name.startswith("查看") else name
        else:
            names[key] = name

    for key in keys:
        if key in names:
            continue
        parts = key.split(":")
        if len(parts) != 3:
            continue
        resource, op, scope = parts
        if scope == "read":
            counterpart_key = f"{resource}:{op}:write"
        else:
            counterpart_key = f"{resource}:{op}:read"
        counterpart = names.get(counterpart_key)
        if counterpart:
            if scope == "read":
                names[key] = f"查看{counterpart}" if not counterpart.startswith("查看") else counterpart
            else:
                names[key] = counterpart

    return names


async def get_role_ancestors(role_id: str, db: AsyncSession) -> list[RBACRole]:
    ancestors: list[RBACRole] = []
    seen: set[str] = {role_id}
    frontier: set[str] = {role_id}

    while frontier:
        result = await db.execute(
            select(RBACRole)
            .join(
                RBACRoleHierarchy,
                RBACRole.id == RBACRoleHierarchy.parent_role_id,
            )
            .where(RBACRoleHierarchy.child_role_id.in_(frontier))
        )
        frontier = set()
        for role in result.scalars().all():
            if role.id in seen:
                continue
            seen.add(role.id)
            ancestors.append(role)
            frontier.add(role.id)

    return ancestors


async def get_role_descendants(role_id: str, db: AsyncSession) -> list[RBACRole]:
    descendants: list[RBACRole] = []
    seen: set[str] = {role_id}
    frontier: set[str] = {role_id}

    while frontier:
        result = await db.execute(
            select(RBACRole)
            .join(
                RBACRoleHierarchy,
                RBACRole.id == RBACRoleHierarchy.child_role_id,
            )
            .where(RBACRoleHierarchy.parent_role_id.in_(frontier))
        )
        frontier = set()
        for role in result.scalars().all():
            if role.id in seen:
                continue
            seen.add(role.id)
            descendants.append(role)
            frontier.add(role.id)

    return descendants


async def compute_user_effective_permissions(
    user_id: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> UserEffectivePermissions:
    """Compute the full effective permission bundle for a user.

    Processing pipeline (matches the design diagram layers):

      1. User-role assignments (RBACUserRoleAssignment) filtered by validity
         window. When active_role_ids is provided it intersects with the
         assigned set (session-level activation).
      2. Role inheritance closure: for every directly assigned role the
         ancestor chain is resolved via RBACRoleHierarchy so permissions
         propagate up the hierarchy.
      3. Super-admin shortcut: any role (directly assigned or inherited)
         marked with is_super_admin=True yields the full permission set
         and the user is considered immune from custom overrides.
      4. Role-level permission aggregation: every (role, permission) pair
         from RBACRolePermission is merged into an access map with the
         read/write scope derived from the permission key.
      5. User custom overrides (RBACUserPermissionOverride):
         - When any override exists (has_overrides=True), granted_keys
           collects the keys with granted=True. The effective permission
           set is computed later as the intersection of the role
           permission closure (access_map → flatten) and granted_keys.
         - access_map itself is NOT filtered here; intersection is applied
           in get_user_effective_flat_permissions() after derivation.
         - Super admins bypass this step entirely.

    Returns a UserEffectivePermissions dataclass.
    """
    now = datetime.utcnow()

    result = await db.execute(
        select(RBACUserRoleAssignment).where(
            RBACUserRoleAssignment.user_id == user_id,
            (
                RBACUserRoleAssignment.valid_from.is_(None)
                | (RBACUserRoleAssignment.valid_from <= now)
            ),
            (
                RBACUserRoleAssignment.valid_until.is_(None)
                | (RBACUserRoleAssignment.valid_until >= now)
            ),
        )
    )
    assignments = result.scalars().all()

    assigned_role_ids = {assignment.role_id for assignment in assignments}
    if active_role_ids is not None:
        assigned_role_ids &= set(active_role_ids)

    if not assigned_role_ids:
        return UserEffectivePermissions(
            access_map={},
            granted_keys=set(),
            has_overrides=False,
            is_super_admin=False,
        )

    super_admin_result = await db.execute(
        select(RBACRole).where(
            RBACRole.id.in_(assigned_role_ids),
            RBACRole.is_super_admin.is_(True),
        )
    )
    if super_admin_result.scalars().first() is not None:
        all_permissions = await db.execute(
            select(RBACPermission).where(RBACPermission.is_active.is_(True))
        )
        access_map = {
            permission.key: PermissionAccess(read=True, write=True)
            for permission in all_permissions.scalars().all()
        }
        return UserEffectivePermissions(
            access_map=access_map,
            granted_keys=set(),
            has_overrides=False,
            is_super_admin=True,
        )

    all_role_ids: set[str] = set()
    for role_id in assigned_role_ids:
        all_role_ids.add(role_id)
        ancestors = await get_role_ancestors(role_id, db)
        all_role_ids.update(ancestor.id for ancestor in ancestors)

    super_admin_result = await db.execute(
        select(RBACRole).where(
            RBACRole.id.in_(all_role_ids),
            RBACRole.is_super_admin.is_(True),
        )
    )
    if super_admin_result.scalars().first() is not None:
        all_permissions = await db.execute(
            select(RBACPermission).where(RBACPermission.is_active.is_(True))
        )
        access_map = {
            permission.key: PermissionAccess(read=True, write=True)
            for permission in all_permissions.scalars().all()
        }
        return UserEffectivePermissions(
            access_map=access_map,
            granted_keys=set(),
            has_overrides=False,
            is_super_admin=True,
        )

    result = await db.execute(
        select(RBACRolePermission, RBACPermission)
        .join(RBACPermission, RBACRolePermission.permission_id == RBACPermission.id)
        .where(
            RBACRolePermission.role_id.in_(all_role_ids),
            RBACPermission.is_active.is_(True),
        )
    )

    role_access_map: dict[str, PermissionAccess] = {}
    for role_permission, permission in result.all():
        scope = _permission_scope(permission.key)
        access = role_access_map.get(permission.key)
        if access is None:
            access = PermissionAccess()
            role_access_map[permission.key] = access
        if scope == "read":
            access.read = True
        elif scope == "write":
            access.write = True
        else:
            if permission.operation.lower() in READ_OPERATIONS:
                access.read = True
            if permission.operation.lower() in WRITE_OPERATIONS:
                access.write = True

    overrides = await db.execute(
        select(RBACUserPermissionOverride).where(
            RBACUserPermissionOverride.user_id == user_id,
        )
    )
    overrides_list = overrides.scalars().all()

    if not overrides_list:
        return UserEffectivePermissions(
            access_map=role_access_map,
            granted_keys=set(),
            has_overrides=False,
            is_super_admin=False,
        )

    # 交集模型：有效权限 = 角色权限 ∩ 自定义权限(granted=True)
    # 这里不过滤 access_map，只收集 granted_keys，交集在 flatten 之后应用
    granted_keys: set[str] = {
        override.permission_key
        for override in overrides_list
        if override.granted
    }

    return UserEffectivePermissions(
        access_map=role_access_map,
        granted_keys=granted_keys,
        has_overrides=True,
        is_super_admin=False,
    )


async def get_user_effective_permissions(
    user_id: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> dict[str, PermissionAccess]:
    """Backward-compatible wrapper around compute_user_effective_permissions.

    Returns only the access_map portion (role permissions + inheritance,
    NOT filtered by overrides). Callers that need the intersection with
    granted_keys or the super-admin flag should call
    compute_user_effective_permissions() or
    get_user_effective_flat_permissions() directly.
    """
    bundle = await compute_user_effective_permissions(user_id, db, active_role_ids)
    return bundle.access_map


async def get_user_effective_flat_permissions(
    user_id: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> dict[str, str]:
    """返回当前用户最终有效权限（key → display_name）。

    交集模型：
      - 无自定义权限覆盖 → 有效权限 = 角色权限（含继承、派生）
      - 有自定义权限覆盖 → 有效权限 = 角色权限(含继承、派生) ∩ granted_keys
      - 超级管理员 → 全部权限，不受覆盖约束
    """
    bundle = await compute_user_effective_permissions(user_id, db, active_role_ids)
    flat = await flatten_effective_permissions(bundle.access_map, db)

    if bundle.is_super_admin or not bundle.has_overrides:
        return flat

    # 交集：只保留用户明确 granted=True 的权限
    return {k: v for k, v in flat.items() if k in bundle.granted_keys}


async def _role_closure_role_ids(
    user_id: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> set[str]:
    now = datetime.utcnow()
    result = await db.execute(
        select(RBACUserRoleAssignment).where(
            RBACUserRoleAssignment.user_id == user_id,
            (
                RBACUserRoleAssignment.valid_from.is_(None)
                | (RBACUserRoleAssignment.valid_from <= now)
            ),
            (
                RBACUserRoleAssignment.valid_until.is_(None)
                | (RBACUserRoleAssignment.valid_until >= now)
            ),
        )
    )
    assignments = result.scalars().all()

    assigned_role_ids = {assignment.role_id for assignment in assignments}
    if active_role_ids is not None:
        assigned_role_ids &= set(active_role_ids)

    if not assigned_role_ids:
        return set()

    all_role_ids: set[str] = set(assigned_role_ids)
    for role_id in assigned_role_ids:
        ancestors = await get_role_ancestors(role_id, db)
        all_role_ids.update(ancestor.id for ancestor in ancestors)
    return all_role_ids


async def _closure_has_super_admin(role_ids: set[str], db: AsyncSession) -> bool:
    if not role_ids:
        return False
    result = await db.execute(
        select(RBACRole).where(
            RBACRole.id.in_(role_ids),
            RBACRole.is_super_admin.is_(True),
        )
    )
    return result.scalars().first() is not None


async def has_permission(
    user_id: str,
    permission_key: str,
    mode: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> bool:
    """判断用户是否拥有指定权限（交集模型）。

    有效权限 = 角色权限(含继承、派生) ∩ 自定义权限(granted=True)
    超级管理员不受覆盖约束。
    """
    if mode not in {"read", "write"}:
        raise ValueError(f"Unsupported permission mode: {mode}")

    bundle = await compute_user_effective_permissions(user_id, db, active_role_ids)

    # 超级管理员：直接放行
    if bundle.is_super_admin:
        return True

    # 角色权限闭包（含派生）
    flat = await flatten_effective_permissions(bundle.access_map, db)

    # 无自定义覆盖 → 角色权限即为有效权限
    if not bundle.has_overrides:
        in_role = permission_key in flat
        if not in_role and mode == "read":
            # write 派生 read：检查 write 变体是否在角色权限中
            parts = permission_key.split(":")
            if len(parts) == 3 and parts[2] == "read":
                in_role = f"{parts[0]}:{parts[1]}:write" in flat
        return in_role

    # 交集：必须在角色权限闭包中 且 必须在 granted_keys 中
    candidate_keys = set(resolve_read_keys(permission_key)) if mode == "read" else set(resolve_write_keys(permission_key))
    if mode == "read":
        # write 派生 read
        parts = permission_key.split(":")
        if len(parts) == 3 and parts[2] == "read":
            candidate_keys |= {f"{parts[0]}:{parts[1]}:write"}

    return any(
        key in flat and key in bundle.granted_keys
        for key in candidate_keys
    )


async def has_permission_direct(
    user_id: str,
    permission_key: str,
    mode: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> bool:
    """判断用户是否拥有指定权限（交集模型，直接查询数据库）。

    与 has_permission 等价，但避免加载全部有效权限，用于高频调用场景。
    有效权限 = 角色权限(含继承、派生) ∩ 自定义权限(granted=True)
    超级管理员不受覆盖约束。
    """
    if mode not in {"read", "write"}:
        raise ValueError(f"Unsupported permission mode: {mode}")

    all_role_ids = await _role_closure_role_ids(user_id, db, active_role_ids)
    if not all_role_ids:
        return False

    # 超级管理员：直接放行
    if await _closure_has_super_admin(all_role_ids, db):
        return True

    candidate_keys = set(resolve_read_keys(permission_key)) if mode == "read" else set(resolve_write_keys(permission_key))
    if mode == "read":
        parts = permission_key.split(":")
        if len(parts) == 3 and parts[2] == "read":
            candidate_keys |= {f"{parts[0]}:{parts[1]}:write"}

    # 1. 检查角色权限闭包是否包含候选 key
    role_result = await db.execute(
        select(RBACPermission.key)
        .join(RBACRolePermission, RBACRolePermission.permission_id == RBACPermission.id)
        .where(
            RBACRolePermission.role_id.in_(all_role_ids),
            RBACPermission.key.in_(list(candidate_keys)),
            RBACPermission.is_active.is_(True),
        )
    )
    role_granted = {row[0] for row in role_result.all()}
    if not role_granted:
        return False

    # 2. 检查用户是否有自定义权限覆盖
    override_count_result = await db.execute(
        select(RBACUserPermissionOverride.permission_key).where(
            RBACUserPermissionOverride.user_id == user_id,
        )
    )
    override_rows = override_count_result.all()
    if not override_rows:
        # 无自定义覆盖 → 角色权限即为有效权限
        return True

    # 3. 交集：必须在 granted_keys 中
    granted_true_result = await db.execute(
        select(RBACUserPermissionOverride.permission_key).where(
            RBACUserPermissionOverride.user_id == user_id,
            RBACUserPermissionOverride.permission_key.in_(list(candidate_keys)),
            RBACUserPermissionOverride.granted.is_(True),
        )
    )
    granted_true = {row[0] for row in granted_true_result.all()}
    return bool(role_granted & granted_true)
