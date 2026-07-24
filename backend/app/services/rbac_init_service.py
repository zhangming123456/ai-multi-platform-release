from __future__ import annotations

from datetime import datetime

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource
from app.models.rbac_role import RBACRole
from app.models.rbac_role_permission import RBACRolePermission
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment
from app.models.role_permission import RolePermission
from app.models.user import User
from app.routers.permissions import ALL_PERMISSIONS, DEFAULT_ROLE_PERMISSIONS


BUILTIN_ROLES = [
    {
        "name": "admin",
        "display_name": "超级管理员",
        "is_super_admin": True,
        "is_builtin": True,
        "role_type": "admin",
    },
    {
        "name": "manager",
        "display_name": "管理员",
        "is_super_admin": False,
        "is_builtin": True,
        "role_type": "admin",
    },
    {
        "name": "operator",
        "display_name": "运营者",
        "is_super_admin": False,
        "is_builtin": True,
        "role_type": "other",
    },
    {
        "name": "reviewer",
        "display_name": "审核员",
        "is_super_admin": False,
        "is_builtin": True,
        "role_type": "other",
    },
]


def _map_legacy_permission(legacy_key: str) -> tuple[str, str]:
    """Map a legacy permission key to (resource_key, operation)."""
    if legacy_key == "db:history:read":
        return "db_history", "read"
    if ":" in legacy_key:
        resource_key, operation = legacy_key.rsplit(":", 1)
        return resource_key, operation
    if legacy_key == "accounts":
        return "users", "read"
    if legacy_key == "permission_manage":
        return "permissions", "read"
    return legacy_key, "read"


def _permission_key(resource_key: str, operation: str) -> str:
    return f"{resource_key}:{operation}"


async def _ensure_builtin_roles(db: AsyncSession) -> None:
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(
        [r["name"] for r in BUILTIN_ROLES]
    )))
    existing_by_name = {role.name: role for role in result.scalars().all()}

    for role_def in BUILTIN_ROLES:
        if role_def["name"] in existing_by_name:
            continue
        db.add(RBACRole(
            name=role_def["name"],
            display_name=role_def["display_name"],
            role_type=role_def["role_type"],
            is_super_admin=role_def["is_super_admin"],
            is_builtin=role_def["is_builtin"],
            created_at=datetime.utcnow(),
            updated_at=datetime.utcnow(),
        ))

    await db.commit()


async def _ensure_resources_and_permissions(db: AsyncSession) -> None:
    result = await db.execute(select(RBACResource).where(RBACResource.key.in_(
        [_map_legacy_permission(p["key"])[0] for p in ALL_PERMISSIONS]
    )))
    existing_resources = {resource.key: resource for resource in result.scalars().all()}

    permission_keys = [_permission_key(*_map_legacy_permission(p["key"])) for p in ALL_PERMISSIONS]
    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(permission_keys)))
    existing_permissions = {perm.key: perm for perm in result.scalars().all()}

    resource_cache: dict[str, RBACResource] = {}

    for perm_def in ALL_PERMISSIONS:
        legacy_key = perm_def["key"]
        resource_key, operation = _map_legacy_permission(legacy_key)
        perm_key = _permission_key(resource_key, operation)

        if resource_key in existing_resources:
            resource = existing_resources[resource_key]
        elif resource_key in resource_cache:
            resource = resource_cache[resource_key]
        else:
            if legacy_key == "accounts":
                resource_name = "账号设置"
            elif legacy_key == "permission_manage":
                resource_name = "权限设置"
            else:
                resource_name = perm_def["name"]

            resource_type = "page" if perm_def["type"] == "page" else "action"
            resource = RBACResource(
                key=resource_key,
                name=resource_name,
                type=resource_type,
                is_active=True,
                created_at=datetime.utcnow(),
            )
            db.add(resource)
            resource_cache[resource_key] = resource

        if perm_key in existing_permissions:
            continue

        # Flush to get resource.id for newly created resources.
        if resource.id is None:
            await db.flush()

        db.add(RBACPermission(
            resource_id=resource.id,
            operation=operation,
            key=perm_key,
            is_active=True,
            created_at=datetime.utcnow(),
        ))

    await db.commit()


async def _ensure_builtin_role_permissions(db: AsyncSession) -> None:
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(
        list(DEFAULT_ROLE_PERMISSIONS.keys())
    )))
    roles_by_name = {role.name: role for role in result.scalars().all()}

    permission_keys = [
        _permission_key(*_map_legacy_permission(key))
        for keys in DEFAULT_ROLE_PERMISSIONS.values()
        for key in keys
    ]
    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(permission_keys)))
    permissions_by_key = {perm.key: perm for perm in result.scalars().all()}

    role_permission_pairs = [
        (roles_by_name[role_name].id, permissions_by_key[_permission_key(*_map_legacy_permission(key))].id)
        for role_name, keys in DEFAULT_ROLE_PERMISSIONS.items()
        if role_name in roles_by_name
        for key in keys
        if _permission_key(*_map_legacy_permission(key)) in permissions_by_key
    ]

    if not role_permission_pairs:
        return

    role_ids = {pair[0] for pair in role_permission_pairs}
    permission_ids = {pair[1] for pair in role_permission_pairs}
    result = await db.execute(
        select(RBACRolePermission.role_id, RBACRolePermission.permission_id).where(
            RBACRolePermission.role_id.in_(role_ids),
            RBACRolePermission.permission_id.in_(permission_ids),
        )
    )
    existing_pairs = set(result.all())

    for role_id, permission_id in role_permission_pairs:
        if (role_id, permission_id) in existing_pairs:
            continue
        db.add(RBACRolePermission(
            role_id=role_id,
            permission_id=permission_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()


async def sync_user_role_assignments(db: AsyncSession) -> None:
    result = await db.execute(select(User.id, User.role))
    users = result.all()

    if not users:
        return

    role_names = {user.role for user in users}
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(role_names)))
    roles_by_name = {role.name: role for role in result.scalars().all()}

    user_role_pairs = [
        (user.id, roles_by_name[user.role].id)
        for user in users
        if user.role in roles_by_name
    ]

    if not user_role_pairs:
        return

    user_ids = {pair[0] for pair in user_role_pairs}
    role_ids = {pair[1] for pair in user_role_pairs}
    result = await db.execute(
        select(RBACUserRoleAssignment.user_id, RBACUserRoleAssignment.role_id).where(
            RBACUserRoleAssignment.user_id.in_(user_ids),
            RBACUserRoleAssignment.role_id.in_(role_ids),
        )
    )
    existing_pairs = set(result.all())

    for user_id, role_id in user_role_pairs:
        if (user_id, role_id) in existing_pairs:
            continue
        db.add(RBACUserRoleAssignment(
            user_id=user_id,
            role_id=role_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()


async def migrate_legacy_role_permissions(db: AsyncSession) -> None:
    result = await db.execute(select(RolePermission))
    legacy_records = result.scalars().all()

    if not legacy_records:
        return

    role_names = {record.role for record in legacy_records}
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(role_names)))
    roles_by_name = {role.name: role for role in result.scalars().all()}

    permission_keys = [
        _permission_key(*_map_legacy_permission(record.permission_key))
        for record in legacy_records
    ]
    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(permission_keys)))
    permissions_by_key = {perm.key: perm for perm in result.scalars().all()}

    role_permission_pairs = [
        (roles_by_name[record.role].id, permissions_by_key[_permission_key(*_map_legacy_permission(record.permission_key))].id)
        for record in legacy_records
        if record.role in roles_by_name
        and _permission_key(*_map_legacy_permission(record.permission_key)) in permissions_by_key
    ]

    if not role_permission_pairs:
        return

    role_ids = {pair[0] for pair in role_permission_pairs}
    permission_ids = {pair[1] for pair in role_permission_pairs}
    result = await db.execute(
        select(RBACRolePermission.role_id, RBACRolePermission.permission_id).where(
            RBACRolePermission.role_id.in_(role_ids),
            RBACRolePermission.permission_id.in_(permission_ids),
        )
    )
    existing_pairs = set(result.all())

    for role_id, permission_id in role_permission_pairs:
        if (role_id, permission_id) in existing_pairs:
            continue
        db.add(RBACRolePermission(
            role_id=role_id,
            permission_id=permission_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()


async def init_rbac_system(db: AsyncSession) -> None:
    await _ensure_builtin_roles(db)
    await _ensure_resources_and_permissions(db)
    await _ensure_builtin_role_permissions(db)
    await sync_user_role_assignments(db)
    await migrate_legacy_role_permissions(db)
