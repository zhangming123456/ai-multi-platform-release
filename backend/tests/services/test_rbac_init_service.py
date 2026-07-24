from __future__ import annotations

import os
import tempfile
import uuid

import pytest
import pytest_asyncio
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

import app.models  # noqa: F401,E402
from app.database import Base  # noqa: E402
from app.models.rbac_permission import RBACPermission  # noqa: E402
from app.models.rbac_resource import RBACResource  # noqa: E402
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_role_permission import RBACRolePermission  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.role_permission import RolePermission  # noqa: E402
from app.models.user import User  # noqa: E402
from app.constants.permissions import ALL_PERMISSIONS, DEFAULT_ROLE_PERMISSIONS  # noqa: E402
from app.services.rbac_init_service import (  # noqa: E402
    _map_legacy_permission,
    _permission_key,
    init_rbac_system,
)

pytestmark = pytest.mark.asyncio


@pytest_asyncio.fixture
async def db():
    fd, db_path = tempfile.mkstemp(suffix=".db")
    os.close(fd)
    engine = create_async_engine(f"sqlite+aiosqlite:///{db_path}")
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    session_factory = async_sessionmaker(
        engine, class_=AsyncSession, expire_on_commit=False
    )
    async with session_factory() as session:
        yield session

    await engine.dispose()
    os.unlink(db_path)


async def _create_user(session: AsyncSession, role: str = "operator") -> User:
    user = User(
        id=str(uuid.uuid4()),
        username=f"user-{uuid.uuid4().hex[:8]}",
        email=f"{uuid.uuid4().hex[:8]}@example.com",
        hashed_password="secret",
        nickname="Test User",
        role=role,
    )
    session.add(user)
    await session.commit()
    return user


async def _create_legacy_role_permission(
    session: AsyncSession,
    role: str,
    permission_key: str,
    can_read: bool = True,
    can_write: bool = True,
) -> RolePermission:
    record = RolePermission(
        id=str(uuid.uuid4()),
        role=role,
        permission_key=permission_key,
        can_read=can_read,
        can_write=can_write,
    )
    session.add(record)
    await session.commit()
    return record


async def test_init_rbac_system_creates_builtin_roles_resources_permissions_and_syncs_users(
    db,
):
    admin_user = await _create_user(db, role="admin")
    operator_user = await _create_user(db, role="operator")
    await _create_user(db, role="unknown_role")

    await init_rbac_system(db)

    roles_result = await db.execute(select(RBACRole).order_by(RBACRole.name))
    roles = roles_result.scalars().all()
    role_names = {role.name for role in roles}
    assert role_names == {"admin", "manager", "operator", "reviewer"}

    admin_role = next(role for role in roles if role.name == "admin")
    assert admin_role.is_super_admin is True
    assert admin_role.is_builtin is True
    assert admin_role.display_name == "超级管理员"

    resources_result = await db.execute(select(RBACResource))
    resources = resources_result.scalars().all()
    resource_keys = {resource.key for resource in resources}
    expected_resource_keys = {
        _map_legacy_permission(p["key"])[0] for p in ALL_PERMISSIONS
    }
    assert resource_keys == expected_resource_keys

    users_resource = next(
        (resource for resource in resources if resource.key == "users"), None
    )
    assert users_resource is not None
    assert users_resource.name == "账号设置"

    templates_resource = next(
        (resource for resource in resources if resource.key == "templates"), None
    )
    assert templates_resource is not None
    assert templates_resource.name == "模板中心"

    permissions_result = await db.execute(select(RBACPermission))
    permissions = permissions_result.scalars().all()
    permissions_by_id = {permission.id: permission.key for permission in permissions}
    permission_keys = {permission.key for permission in permissions}
    expected_permission_keys = {
        _permission_key(*_map_legacy_permission(p["key"]))
        for p in ALL_PERMISSIONS
    }
    assert permission_keys == expected_permission_keys

    users_create_perm = next(
        (permission for permission in permissions if permission.key == "users:create"), None
    )
    assert users_create_perm is not None
    assert users_create_perm.resource_id == users_resource.id

    templates_create_perm = next(
        (permission for permission in permissions if permission.key == "templates:create"), None
    )
    assert templates_create_perm is not None
    assert templates_create_perm.resource_id == templates_resource.id

    role_permissions_result = await db.execute(select(RBACRolePermission))
    role_permissions = role_permissions_result.scalars().all()
    admin_role_permission_keys = {
        permissions_by_id[rp.permission_id]
        for rp in role_permissions
        if rp.role_id == admin_role.id
    }
    expected_admin_permission_keys = {
        _permission_key(*_map_legacy_permission(key))
        for key in DEFAULT_ROLE_PERMISSIONS["admin"]
    }
    assert admin_role_permission_keys == expected_admin_permission_keys

    assignments_result = await db.execute(select(RBACUserRoleAssignment))
    assignments = assignments_result.scalars().all()
    assignment_pairs = {(assignment.user_id, assignment.role_id) for assignment in assignments}
    assert (admin_user.id, admin_role.id) in assignment_pairs
    assert any(
        assignment.user_id == operator_user.id for assignment in assignments
    )


async def test_migrate_legacy_role_permissions(db):
    await init_rbac_system(db)

    await _create_legacy_role_permission(db, "manager", "content:create")
    await _create_legacy_role_permission(db, "manager", "accounts")
    await _create_legacy_role_permission(db, "unknown_role", "content:create")

    await init_rbac_system(db)

    manager_role = (
        await db.execute(select(RBACRole).where(RBACRole.name == "manager"))
    ).scalar_one()
    content_create_perm = (
        await db.execute(
            select(RBACPermission).where(RBACPermission.key == "content:create")
        )
    ).scalar_one()
    users_read_perm = (
        await db.execute(
            select(RBACPermission).where(RBACPermission.key == "users:read")
        )
    ).scalar_one()

    role_permissions_result = await db.execute(
        select(RBACRolePermission).where(
            RBACRolePermission.role_id == manager_role.id
        )
    )
    migrated_pairs = {
        (rp.role_id, rp.permission_id) for rp in role_permissions_result.scalars().all()
    }
    assert (manager_role.id, content_create_perm.id) in migrated_pairs
    assert (manager_role.id, users_read_perm.id) in migrated_pairs


async def test_migrate_legacy_role_permissions_skips_denied_records(db):
    await init_rbac_system(db)

    # Use a permission not present in the manager default set so that any
    # existing RBACRolePermission could only come from legacy migration.
    await _create_legacy_role_permission(
        db, "manager", "user:delete", can_read=False, can_write=False
    )

    await init_rbac_system(db)

    manager_role = (
        await db.execute(select(RBACRole).where(RBACRole.name == "manager"))
    ).scalar_one()
    users_delete_perm = (
        await db.execute(
            select(RBACPermission).where(RBACPermission.key == "users:delete")
        )
    ).scalar_one()

    role_permissions_result = await db.execute(
        select(RBACRolePermission).where(
            RBACRolePermission.role_id == manager_role.id,
            RBACRolePermission.permission_id == users_delete_perm.id,
        )
    )
    assert role_permissions_result.scalar_one_or_none() is None


async def test_init_rbac_system_is_idempotent(db):
    await _create_user(db, role="admin")
    await _create_legacy_role_permission(db, "admin", "content:create")

    await init_rbac_system(db)

    roles_count_first = (
        await db.execute(select(RBACRole))
    ).scalars().all().__len__()
    resources_count_first = (
        await db.execute(select(RBACResource))
    ).scalars().all().__len__()
    permissions_count_first = (
        await db.execute(select(RBACPermission))
    ).scalars().all().__len__()
    role_permissions_count_first = (
        await db.execute(select(RBACRolePermission))
    ).scalars().all().__len__()
    assignments_count_first = (
        await db.execute(select(RBACUserRoleAssignment))
    ).scalars().all().__len__()

    await init_rbac_system(db)

    assert (await db.execute(select(RBACRole))).scalars().all().__len__() == roles_count_first
    assert (
        await db.execute(select(RBACResource))
    ).scalars().all().__len__() == resources_count_first
    assert (
        await db.execute(select(RBACPermission))
    ).scalars().all().__len__() == permissions_count_first
    assert (
        await db.execute(select(RBACRolePermission))
    ).scalars().all().__len__() == role_permissions_count_first
    assert (
        await db.execute(select(RBACUserRoleAssignment))
    ).scalars().all().__len__() == assignments_count_first
