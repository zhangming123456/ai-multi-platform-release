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
    denied_keys: set[str] | None = None,
) -> dict[str, str]:
    deny = denied_keys or set()
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

    flat -= deny

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
    7. 有自定义覆盖 → 有效 = 角色权限 - {key | override.granted=false}。
       仅移除明确拒绝的权限，未覆盖的权限保持角色默认值。
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
        return {}

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
        return {
            permission.key: PermissionAccess(read=True, write=True)
            for permission in all_permissions.scalars().all()
        }

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
        return {
            permission.key: PermissionAccess(read=True, write=True)
            for permission in all_permissions.scalars().all()
        }

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
        return role_access_map

    denied_keys: set[str] = {
        override.permission_key
        for override in overrides_list
        if not override.granted
    }

    access_map: dict[str, PermissionAccess] = {}
    for key, access in role_access_map.items():
        if key not in denied_keys:
            access_map[key] = access

    return access_map


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
    if mode not in {"read", "write"}:
        raise ValueError(f"Unsupported permission mode: {mode}")
    effective = await get_user_effective_permissions(
        user_id,
        db,
        active_role_ids=active_role_ids,
    )
    access = effective.get(permission_key)
    if access is None:
        return False
    if mode == "read":
        return access.read or access.write
    return access.write


async def has_permission_direct(
    user_id: str,
    permission_key: str,
    mode: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> bool:
    if mode not in {"read", "write"}:
        raise ValueError(f"Unsupported permission mode: {mode}")

    all_role_ids = await _role_closure_role_ids(user_id, db, active_role_ids)
    if not all_role_ids:
        return False

    if await _closure_has_super_admin(all_role_ids, db):
        return True

    candidate_keys = resolve_read_keys(permission_key) if mode == "read" else resolve_write_keys(permission_key)

    result = await db.execute(
        select(RBACPermission.key)
        .join(RBACRolePermission, RBACRolePermission.permission_id == RBACPermission.id)
        .where(
            RBACRolePermission.role_id.in_(all_role_ids),
            RBACPermission.key.in_(candidate_keys),
            RBACPermission.is_active.is_(True),
        )
    )
    granted_keys = {row[0] for row in result.all()}

    write_candidates = set(resolve_write_keys(permission_key))
    read_candidates = set(resolve_read_keys(permission_key))

    deny_result = await db.execute(
        select(RBACUserPermissionOverride).where(
            RBACUserPermissionOverride.user_id == user_id,
            RBACUserPermissionOverride.permission_key.in_(list(read_candidates | write_candidates)),
            RBACUserPermissionOverride.granted.is_(False),
        )
    )
    explicitly_denied = deny_result.scalars().first() is not None
    if explicitly_denied:
        return False

    if mode == "read":
        return bool(granted_keys & (read_candidates | write_candidates))
    return bool(granted_keys & write_candidates)
