from __future__ import annotations

import json
from datetime import datetime
from typing import Optional

from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    RBACPermission,
    RBACResource,
    RBACRole,
    RBACRoleHierarchy,
    RBACRolePermission,
    RBACUserRoleAssignment,
)


class PermissionAccess:
    def __init__(self, read: bool = False, write: bool = False):
        self.read = read
        self.write = write

    def to_dict(self) -> dict:
        return {"read": self.read, "write": self.write}


WRITE_OPERATIONS = {"create", "update", "delete", "approve", "reject", "execute"}
READ_OPERATIONS = {"read", "execute"}


async def get_role_ancestors(role_id: str, db: AsyncSession) -> list[RBACRole]:
    ancestors = []
    visited = set()
    stack = [role_id]

    while stack:
        current_id = stack.pop()
        if current_id in visited:
            continue
        visited.add(current_id)

        result = await db.execute(
            select(RBACRole)
            .join(RBACRoleHierarchy, RBACRole.id == RBACRoleHierarchy.parent_role_id)
            .where(RBACRoleHierarchy.child_role_id == current_id)
        )
        for parent in result.scalars().all():
            ancestors.append(parent)
            stack.append(parent.id)

    return ancestors


async def get_role_descendants(role_id: str, db: AsyncSession) -> list[RBACRole]:
    descendants = []
    visited = set()
    stack = [role_id]

    while stack:
        current_id = stack.pop()
        if current_id in visited:
            continue
        visited.add(current_id)

        result = await db.execute(
            select(RBACRole)
            .join(RBACRoleHierarchy, RBACRole.id == RBACRoleHierarchy.child_role_id)
            .where(RBACRoleHierarchy.parent_role_id == current_id)
        )
        for child in result.scalars().all():
            descendants.append(child)
            stack.append(child.id)

    return descendants


async def get_all_role_ids_with_ancestors(role_ids: set[str], db: AsyncSession) -> set[str]:
    all_role_ids = set(role_ids)
    for role_id in role_ids:
        ancestors = await get_role_ancestors(role_id, db)
        all_role_ids.update([a.id for a in ancestors])
    return all_role_ids


async def get_permissions_by_role_ids(
    role_ids: set[str],
    db: AsyncSession,
) -> list[tuple[RBACPermission, str]]:
    result = await db.execute(
        select(RBACPermission, RBACRolePermission.grant_type)
        .join(RBACRolePermission, RBACPermission.id == RBACRolePermission.permission_id)
        .where(
            RBACRolePermission.role_id.in_(list(role_ids)),
            RBACPermission.is_active == True,
        )
    )
    return [(row[0], row[1]) for row in result.all()]


async def get_assigned_roles(
    user_id: str,
    db: AsyncSession,
    at_time: Optional[datetime] = None,
) -> list[RBACRole]:
    if at_time is None:
        at_time = datetime.now()

    result = await db.execute(
        select(RBACRole)
        .join(RBACUserRoleAssignment, RBACRole.id == RBACUserRoleAssignment.role_id)
        .where(
            RBACUserRoleAssignment.user_id == user_id,
            RBACRole.is_active == True,
            (RBACUserRoleAssignment.valid_from.is_(None) | (RBACUserRoleAssignment.valid_from <= at_time)),
            (RBACUserRoleAssignment.valid_until.is_(None) | (RBACUserRoleAssignment.valid_until >= at_time)),
        )
    )
    return list(result.scalars().all())


async def is_super_admin(user_id: str, db: AsyncSession) -> bool:
    assigned = await get_assigned_roles(user_id, db)
    return any(r.is_super_admin for r in assigned)


async def get_user_effective_permissions(
    user_id: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> dict[str, PermissionAccess]:
    if await is_super_admin(user_id, db):
        return await _get_all_permissions_with_full_access(db)

    assigned_roles = await get_assigned_roles(user_id, db)
    assigned_role_ids = {r.id for r in assigned_roles}

    if active_role_ids:
        role_ids = {rid for rid in active_role_ids if rid in assigned_role_ids}
    else:
        role_ids = assigned_role_ids

    all_role_ids = await get_all_role_ids_with_ancestors(role_ids, db)
    perms = await get_permissions_by_role_ids(all_role_ids, db)

    result: dict[str, PermissionAccess] = {}
    for perm, _grant_type in perms:
        key = perm.key
        if key not in result:
            result[key] = PermissionAccess()
        if perm.operation in READ_OPERATIONS:
            result[key].read = True
        if perm.operation in WRITE_OPERATIONS:
            result[key].write = True

    return result


async def _get_all_permissions_with_full_access(db: AsyncSession) -> dict[str, PermissionAccess]:
    result = await db.execute(
        select(RBACPermission).where(RBACPermission.is_active == True)
    )
    perms = result.scalars().all()
    return {p.key: PermissionAccess(read=True, write=True) for p in perms}


async def has_permission(
    user_id: str,
    permission_key: str,
    mode: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> bool:
    if await is_super_admin(user_id, db):
        return True
    perms = await get_user_effective_permissions(user_id, db, active_role_ids)
    access = perms.get(permission_key)
    if not access:
        return False
    if mode == "read":
        return access.read
    if mode == "write":
        return access.write
    return False


async def validate_hierarchy_dag(
    parent_role_id: str,
    child_role_id: str,
    db: AsyncSession,
) -> bool:
    descendants = await get_role_descendants(child_role_id, db)
    descendant_ids = {d.id for d in descendants}
    return parent_role_id not in descendant_ids and parent_role_id != child_role_id


async def get_role_effective_permissions(
    role_id: str,
    db: AsyncSession,
) -> dict[str, PermissionAccess]:
    all_role_ids = await get_all_role_ids_with_ancestors({role_id}, db)
    perms = await get_permissions_by_role_ids(all_role_ids, db)

    result: dict[str, PermissionAccess] = {}
    for perm, _grant_type in perms:
        key = perm.key
        if key not in result:
            result[key] = PermissionAccess()
        if perm.operation in READ_OPERATIONS:
            result[key].read = True
        if perm.operation in WRITE_OPERATIONS:
            result[key].write = True

    return result


async def get_role_direct_permission_ids(
    role_id: str,
    db: AsyncSession,
) -> set[str]:
    result = await db.execute(
        select(RBACRolePermission.permission_id).where(
            RBACRolePermission.role_id == role_id,
            RBACRolePermission.grant_type == "direct",
        )
    )
    return {row[0] for row in result.all()}


async def get_role_direct_permissions(
    role_id: str,
    db: AsyncSession,
) -> dict[str, PermissionAccess]:
    result = await db.execute(
        select(RBACPermission, RBACRolePermission).join(
            RBACRolePermission,
            RBACPermission.id == RBACRolePermission.permission_id,
        ).where(
            RBACRolePermission.role_id == role_id,
            RBACRolePermission.grant_type == "direct",
        )
    )
    perms: dict[str, PermissionAccess] = {}
    for perm, _ in result.all():
        key = perm.key
        if key not in perms:
            perms[key] = PermissionAccess()
        if perm.operation in READ_OPERATIONS:
            perms[key].read = True
        if perm.operation in WRITE_OPERATIONS:
            perms[key].write = True
    return perms


async def get_role_permission_detail(
    role_id: str,
    db: AsyncSession,
) -> dict[str, dict]:
    effective = await get_role_effective_permissions(role_id, db)
    direct = await get_role_direct_permissions(role_id, db)

    result: dict[str, dict] = {}
    for key, access in effective.items():
        is_direct = key in direct
        result[key] = {
            "read": access.read,
            "write": access.write,
            "grant_type": "direct" if is_direct else "inherited",
        }
    return result


async def update_role_direct_permissions(
    role_id: str,
    permission_ids: list[str],
    db: AsyncSession,
):
    await db.execute(
        delete(RBACRolePermission).where(
            RBACRolePermission.role_id == role_id,
            RBACRolePermission.grant_type == "direct",
        )
    )
    for pid in permission_ids:
        db.add(RBACRolePermission(role_id=role_id, permission_id=pid, grant_type="direct"))
    await db.flush()


def parse_active_role_ids(active_role_ids_str: Optional[str]) -> list[str]:
    if not active_role_ids_str:
        return []
    try:
        return json.loads(active_role_ids_str)
    except (json.JSONDecodeError, TypeError):
        return []
