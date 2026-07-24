from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.rbac_permission import RBACPermission
from app.models.rbac_role import RBACRole
from app.models.rbac_role_hierarchy import RBACRoleHierarchy
from app.models.rbac_role_permission import RBACRolePermission
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment


READ_OPERATIONS = {"read", "execute"}
WRITE_OPERATIONS = {"create", "update", "delete", "approve", "reject", "execute"}


@dataclass
class PermissionAccess:
    read: bool = False
    write: bool = False


async def get_role_ancestors(role_id: str, db: AsyncSession) -> list[RBACRole]:
    """Return all ancestor roles by walking up the hierarchy (parent -> child).

    Defensively handles cycles by tracking visited role IDs.
    """
    ancestors: list[RBACRole] = []
    seen_role_ids: set[str] = {role_id}
    current_level_ids: set[str] = {role_id}

    while current_level_ids:
        result = await db.execute(
            select(RBACRole)
            .join(
                RBACRoleHierarchy,
                RBACRole.id == RBACRoleHierarchy.parent_role_id,
            )
            .where(RBACRoleHierarchy.child_role_id.in_(current_level_ids))
        )
        next_level_ids: set[str] = set()
        for role in result.scalars().all():
            if role.id in seen_role_ids:
                continue
            seen_role_ids.add(role.id)
            ancestors.append(role)
            next_level_ids.add(role.id)
        current_level_ids = next_level_ids

    return ancestors


async def get_role_descendants(role_id: str, db: AsyncSession) -> list[RBACRole]:
    """Return all descendant roles by walking down the hierarchy (parent -> child).

    Defensively handles cycles by tracking visited role IDs.
    """
    descendants: list[RBACRole] = []
    seen_role_ids: set[str] = {role_id}
    current_level_ids: set[str] = {role_id}

    while current_level_ids:
        result = await db.execute(
            select(RBACRole)
            .join(
                RBACRoleHierarchy,
                RBACRole.id == RBACRoleHierarchy.child_role_id,
            )
            .where(RBACRoleHierarchy.parent_role_id.in_(current_level_ids))
        )
        next_level_ids: set[str] = set()
        for role in result.scalars().all():
            if role.id in seen_role_ids:
                continue
            seen_role_ids.add(role.id)
            descendants.append(role)
            next_level_ids.add(role.id)
        current_level_ids = next_level_ids

    return descendants


async def get_user_effective_permissions(
    user_id: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> dict[str, PermissionAccess]:
    """Compute the user's effective permission map.

    Super-admin roles grant read/write access to every active permission.
    Otherwise, permissions are aggregated from the user's valid role
    assignments and their ancestor role closures.
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

    # Build the ancestor closure for each assigned role once.
    all_role_ids: set[str] = set()
    for role_id in assigned_role_ids:
        all_role_ids.add(role_id)
        ancestors = await get_role_ancestors(role_id, db)
        all_role_ids.update(ancestor.id for ancestor in ancestors)

    # If any role in the closure is a super admin, grant everything.
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

    # Aggregate permissions from the role closure.
    result = await db.execute(
        select(RBACRolePermission, RBACPermission)
        .join(RBACPermission, RBACRolePermission.permission_id == RBACPermission.id)
        .where(
            RBACRolePermission.role_id.in_(all_role_ids),
            RBACPermission.is_active.is_(True),
        )
    )

    access_map: dict[str, PermissionAccess] = {}
    for role_permission, permission in result.all():
        operation = permission.operation.lower()
        access = access_map.get(permission.key)
        if access is None:
            access = PermissionAccess()
            access_map[permission.key] = access
        if operation in READ_OPERATIONS:
            access.read = True
        if operation in WRITE_OPERATIONS:
            access.write = True

    return access_map


async def has_permission(
    user_id: str,
    permission_key: str,
    mode: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> bool:
    """Return True if the user has the requested permission mode."""
    effective = await get_user_effective_permissions(
        user_id,
        db,
        active_role_ids=active_role_ids,
    )
    access = effective.get(permission_key)
    if access is None:
        return False
    if mode == "read":
        return access.read
    if mode == "write":
        return access.write
    return False
