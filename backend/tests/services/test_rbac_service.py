from __future__ import annotations

import os
import tempfile
import uuid
from datetime import datetime, timedelta

import pytest
import pytest_asyncio
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

# Import models so their tables are registered on Base.metadata.
import app.models  # noqa: F401,E402
from app.database import Base  # noqa: E402
from app.models.rbac_permission import RBACPermission  # noqa: E402
from app.models.rbac_resource import RBACResource  # noqa: E402
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_role_hierarchy import RBACRoleHierarchy  # noqa: E402
from app.models.rbac_role_permission import RBACRolePermission  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.user import User  # noqa: E402
from app.services.rbac_service import (  # noqa: E402
    PermissionAccess,
    get_role_ancestors,
    get_role_descendants,
    get_user_effective_permissions,
    has_permission,
    has_permission_direct,
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


async def _create_user(session: AsyncSession) -> User:
    user = User(
        id=str(uuid.uuid4()),
        username=f"user-{uuid.uuid4().hex[:8]}",
        email=f"{uuid.uuid4().hex[:8]}@example.com",
        hashed_password="secret",
        nickname="Test User",
    )
    session.add(user)
    await session.commit()
    return user


async def _create_role(
    session: AsyncSession, *, is_super_admin: bool = False
) -> RBACRole:
    role = RBACRole(
        id=str(uuid.uuid4()),
        name=f"role-{uuid.uuid4().hex[:8]}",
        display_name="Test Role",
        is_super_admin=is_super_admin,
    )
    session.add(role)
    await session.commit()
    return role


async def _create_resource(session: AsyncSession) -> RBACResource:
    resource = RBACResource(
        id=str(uuid.uuid4()),
        key=f"resource-{uuid.uuid4().hex[:8]}",
        name="Test Resource",
        type="entity",
    )
    session.add(resource)
    await session.commit()
    return resource


async def _create_permission(
    session: AsyncSession,
    resource_id: str,
    operation: str,
    key: str,
    *,
    is_active: bool = True,
) -> RBACPermission:
    permission = RBACPermission(
        id=str(uuid.uuid4()),
        resource_id=resource_id,
        operation=operation,
        key=key,
        is_active=is_active,
    )
    session.add(permission)
    await session.commit()
    return permission


async def _assign_role(
    session: AsyncSession,
    user_id: str,
    role_id: str,
    *,
    valid_from: datetime | None = None,
    valid_until: datetime | None = None,
) -> RBACUserRoleAssignment:
    assignment = RBACUserRoleAssignment(
        id=str(uuid.uuid4()),
        user_id=user_id,
        role_id=role_id,
        valid_from=valid_from,
        valid_until=valid_until,
    )
    session.add(assignment)
    await session.commit()
    return assignment


async def _add_hierarchy(
    session: AsyncSession, parent_role_id: str, child_role_id: str
) -> RBACRoleHierarchy:
    link = RBACRoleHierarchy(
        id=str(uuid.uuid4()),
        parent_role_id=parent_role_id,
        child_role_id=child_role_id,
    )
    session.add(link)
    await session.commit()
    return link


async def _grant_permission(
    session: AsyncSession, role_id: str, permission_id: str
) -> RBACRolePermission:
    grant = RBACRolePermission(
        id=str(uuid.uuid4()),
        role_id=role_id,
        permission_id=permission_id,
    )
    session.add(grant)
    await session.commit()
    return grant


async def test_get_role_ancestors_returns_ancestors_and_ignores_cycles(db):
    role_a = await _create_role(db)
    role_b = await _create_role(db)
    role_c = await _create_role(db)

    await _add_hierarchy(db, role_a.id, role_b.id)
    await _add_hierarchy(db, role_b.id, role_c.id)
    # Cycle: C is also a child of A, closing the loop. It must be ignored.
    await _add_hierarchy(db, role_c.id, role_a.id)

    ancestors = await get_role_ancestors(role_c.id, db)
    ancestor_ids = {role.id for role in ancestors}

    assert ancestor_ids == {role_a.id, role_b.id}


async def test_get_role_descendants_returns_descendants_and_ignores_cycles(db):
    role_a = await _create_role(db)
    role_b = await _create_role(db)
    role_c = await _create_role(db)

    await _add_hierarchy(db, role_a.id, role_b.id)
    await _add_hierarchy(db, role_b.id, role_c.id)
    # Cycle: A is also a child of C, closing the loop. It must be ignored.
    await _add_hierarchy(db, role_c.id, role_a.id)

    descendants = await get_role_descendants(role_a.id, db)
    descendant_ids = {role.id for role in descendants}

    assert descendant_ids == {role_b.id, role_c.id}


async def test_get_user_effective_permissions_super_admin_returns_all_active_permissions(
    db,
):
    user = await _create_user(db)
    admin_role = await _create_role(db, is_super_admin=True)
    await _assign_role(db, user.id, admin_role.id)

    resource = await _create_resource(db)
    active_permission = await _create_permission(
        db, resource.id, "read", "posts:read"
    )
    inactive_permission = await _create_permission(
        db, resource.id, "write", "posts:write", is_active=False
    )

    effective = await get_user_effective_permissions(user.id, db)

    assert effective == {
        active_permission.key: PermissionAccess(read=True, write=True),
    }
    assert inactive_permission.key not in effective


async def test_get_user_effective_permissions_aggregates_read_write_from_multiple_roles(
    db,
):
    user = await _create_user(db)
    role_reader = await _create_role(db)
    role_writer = await _create_role(db)
    await _assign_role(db, user.id, role_reader.id)
    await _assign_role(db, user.id, role_writer.id)

    resource = await _create_resource(db)
    read_permission = await _create_permission(
        db, resource.id, "read", "posts:read"
    )
    write_permission = await _create_permission(
        db, resource.id, "update", "posts:update"
    )

    await _grant_permission(db, role_reader.id, read_permission.id)
    await _grant_permission(db, role_writer.id, write_permission.id)

    effective = await get_user_effective_permissions(user.id, db)

    assert effective["posts:read"] == PermissionAccess(read=True, write=False)
    assert effective["posts:update"] == PermissionAccess(read=False, write=True)


async def test_get_user_effective_permissions_filters_by_active_role_ids(db):
    user = await _create_user(db)
    role_allowed = await _create_role(db)
    role_filtered = await _create_role(db)
    await _assign_role(db, user.id, role_allowed.id)
    await _assign_role(db, user.id, role_filtered.id)

    resource = await _create_resource(db)
    allowed_permission = await _create_permission(
        db, resource.id, "read", "allowed:read"
    )
    filtered_permission = await _create_permission(
        db, resource.id, "read", "filtered:read"
    )
    await _grant_permission(db, role_allowed.id, allowed_permission.id)
    await _grant_permission(db, role_filtered.id, filtered_permission.id)

    effective = await get_user_effective_permissions(
        user.id, db, active_role_ids=[role_allowed.id]
    )

    assert "allowed:read" in effective
    assert "filtered:read" not in effective


async def test_get_user_effective_permissions_excludes_expired_or_not_yet_valid_assignments(
    db,
):
    user = await _create_user(db)
    role_valid = await _create_role(db)
    role_not_yet = await _create_role(db)
    role_expired = await _create_role(db)

    resource = await _create_resource(db)
    valid_permission = await _create_permission(
        db, resource.id, "read", "valid:read"
    )
    not_yet_permission = await _create_permission(
        db, resource.id, "read", "future:read"
    )
    expired_permission = await _create_permission(
        db, resource.id, "read", "expired:read"
    )

    await _grant_permission(db, role_valid.id, valid_permission.id)
    await _grant_permission(db, role_not_yet.id, not_yet_permission.id)
    await _grant_permission(db, role_expired.id, expired_permission.id)

    now = datetime.utcnow()
    await _assign_role(db, user.id, role_valid.id)
    await _assign_role(
        db, user.id, role_not_yet.id, valid_from=now + timedelta(days=1)
    )
    await _assign_role(
        db, user.id, role_expired.id, valid_until=now - timedelta(days=1)
    )

    effective = await get_user_effective_permissions(user.id, db)

    assert "valid:read" in effective
    assert "future:read" not in effective
    assert "expired:read" not in effective


async def test_get_user_effective_permissions_inherited_super_admin_returns_all_active_permissions(
    db,
):
    user = await _create_user(db)
    admin_role = await _create_role(db, is_super_admin=True)
    child_role = await _create_role(db)
    await _add_hierarchy(db, admin_role.id, child_role.id)
    await _assign_role(db, user.id, child_role.id)

    resource = await _create_resource(db)
    active_permission = await _create_permission(
        db, resource.id, "read", "posts:read"
    )
    inactive_permission = await _create_permission(
        db, resource.id, "write", "posts:write", is_active=False
    )

    effective = await get_user_effective_permissions(user.id, db)

    assert effective == {
        active_permission.key: PermissionAccess(read=True, write=True),
    }
    assert inactive_permission.key not in effective


async def test_has_permission_read_write_and_unknown_mode_raises(db):
    user = await _create_user(db)
    role = await _create_role(db)
    await _assign_role(db, user.id, role.id)

    resource = await _create_resource(db)
    permission = await _create_permission(db, resource.id, "read", "reports:read")
    write_permission = await _create_permission(
        db, resource.id, "update", "reports:write"
    )
    await _grant_permission(db, role.id, permission.id)
    await _grant_permission(db, role.id, write_permission.id)

    assert await has_permission(user.id, "reports:read", "read", db) is True
    assert await has_permission(user.id, "reports:write", "write", db) is True
    assert await has_permission(user.id, "missing:key", "read", db) is False

    with pytest.raises(ValueError, match="Unsupported permission mode: delete"):
        await has_permission(user.id, "reports:read", "delete", db)


async def test_has_permission_direct_checks_assigned_role_and_ancestors(db):
    user = await _create_user(db)
    role = await _create_role(db)
    parent_role = await _create_role(db)
    await _add_hierarchy(db, parent_role.id, role.id)
    await _assign_role(db, user.id, role.id)

    resource = await _create_resource(db)
    permission = await _create_permission(db, resource.id, "read", "reports:read")
    parent_permission = await _create_permission(
        db, resource.id, "update", "reports:write"
    )
    await _grant_permission(db, role.id, permission.id)
    await _grant_permission(db, parent_role.id, parent_permission.id)

    assert await has_permission_direct(user.id, "reports:read", "read", db) is True
    assert await has_permission_direct(user.id, "reports:write", "write", db) is True
    assert await has_permission_direct(user.id, "missing:key", "read", db) is False

    with pytest.raises(ValueError, match="Unsupported permission mode: delete"):
        await has_permission_direct(user.id, "reports:read", "delete", db)
