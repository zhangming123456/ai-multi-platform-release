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


# Execute is intentionally treated as both read and write: performing an action
# implies both viewing it and writing/mutating state.
READ_OPERATIONS = {"read", "execute"}
WRITE_OPERATIONS = {"create", "update", "delete", "approve", "reject", "execute"}


@dataclass
class PermissionAccess:
    read: bool = False
    write: bool = False


async def get_role_ancestors(role_id: str, db: AsyncSession) -> list[RBACRole]:
    """Return all ancestor roles by walking up the hierarchy (child -> parent).

    Uses a batched BFS so that all parents of the current frontier are fetched
    in a single query per hierarchy level. Defensively handles cycles by
    tracking visited role IDs.
    """
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
    """Return all descendant roles by walking down the hierarchy (parent -> child).

    Uses a batched BFS so that all children of the current frontier are fetched
    in a single query per hierarchy level. Defensively handles cycles by
    tracking visited role IDs.
    """
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

    # If any directly assigned role is a super admin, grant everything without
    # building the ancestor closure.
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
    raise ValueError(f"Unsupported permission mode: {mode}")


async def has_permission_direct(
    user_id: str,
    permission_key: str,
    mode: str,
    db: AsyncSession,
    active_role_ids: Optional[list[str]] = None,
) -> bool:
    """Return True if the user has the requested permission mode.

    This is a targeted check that avoids building the user's full effective
    permission map. It still walks the ancestor closure for assigned roles to
    honor inherited role permissions.
    """
    if mode not in {"read", "write"}:
        raise ValueError(f"Unsupported permission mode: {mode}")

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
        return False

    # If any directly assigned role is a super admin, grant the permission only
    # when the requested key exists and is active.
    super_admin_result = await db.execute(
        select(RBACRole).where(
            RBACRole.id.in_(assigned_role_ids),
            RBACRole.is_super_admin.is_(True),
        )
    )
    if super_admin_result.scalars().first() is not None:
        permission_result = await db.execute(
            select(RBACPermission).where(
                RBACPermission.key == permission_key,
                RBACPermission.is_active.is_(True),
            )
        )
        return permission_result.scalars().first() is not None

    # Build the ancestor closure for each assigned role once.
    all_role_ids: set[str] = set(assigned_role_ids)
    for role_id in assigned_role_ids:
        ancestors = await get_role_ancestors(role_id, db)
        all_role_ids.update(ancestor.id for ancestor in ancestors)

    # Check whether any role in the closure has the requested permission with a
    # sufficient operation.
    result = await db.execute(
        select(RBACRolePermission, RBACPermission)
        .join(RBACPermission, RBACRolePermission.permission_id == RBACPermission.id)
        .where(
            RBACRolePermission.role_id.in_(all_role_ids),
            RBACPermission.key == permission_key,
            RBACPermission.is_active.is_(True),
        )
    )

    for role_permission, permission in result.all():
        operation = permission.operation.lower()
        if mode == "read" and operation in READ_OPERATIONS:
            return True
        if mode == "write" and operation in WRITE_OPERATIONS:
            return True

    return False
