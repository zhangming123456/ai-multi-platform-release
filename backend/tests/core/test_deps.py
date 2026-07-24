from __future__ import annotations

import os
import tempfile
import uuid

import pytest
import pytest_asyncio
from fastapi import HTTPException
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

# Import models so their tables are registered on Base.metadata.
import app.models  # noqa: F401,E402
from app.core.deps import require_permission  # noqa: E402
from app.database import Base  # noqa: E402
from app.models.rbac_permission import RBACPermission  # noqa: E402
from app.models.rbac_resource import RBACResource  # noqa: E402
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_role_permission import RBACRolePermission  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.user import User, UserRole  # noqa: E402

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


async def _create_user(session: AsyncSession, role: str | None = None) -> User:
    user = User(
        id=str(uuid.uuid4()),
        username=f"user-{uuid.uuid4().hex[:8]}",
        email=f"{uuid.uuid4().hex[:8]}@example.com",
        hashed_password="secret",
        nickname="Test User",
        role=role or "operator",
    )
    session.add(user)
    await session.commit()
    return user


async def _create_role(session: AsyncSession) -> RBACRole:
    role = RBACRole(
        id=str(uuid.uuid4()),
        name=f"role-{uuid.uuid4().hex[:8]}",
        display_name="Test Role",
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
) -> RBACPermission:
    permission = RBACPermission(
        id=str(uuid.uuid4()),
        resource_id=resource_id,
        operation=operation,
        key=key,
    )
    session.add(permission)
    await session.commit()
    return permission


async def _assign_role(
    session: AsyncSession,
    user_id: str,
    role_id: str,
) -> RBACUserRoleAssignment:
    assignment = RBACUserRoleAssignment(
        id=str(uuid.uuid4()),
        user_id=user_id,
        role_id=role_id,
    )
    session.add(assignment)
    await session.commit()
    return assignment


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


async def test_require_permission_allows_when_granted(db):
    user = await _create_user(db)
    role = await _create_role(db)
    resource = await _create_resource(db)
    permission = await _create_permission(db, resource.id, "read", "users:read")

    await _assign_role(db, user.id, role.id)
    await _grant_permission(db, role.id, permission.id)

    checker = require_permission("users:read", "read")
    result = await checker(current_user=user, db=db)

    assert result is user


async def test_require_permission_denies_when_not_granted(db):
    user = await _create_user(db)

    checker = require_permission("users:read", "read")
    with pytest.raises(HTTPException) as exc_info:
        await checker(current_user=user, db=db)

    assert exc_info.value.status_code == 403
    assert exc_info.value.detail == "无查看权限"


async def test_require_permission_legacy_page_key_resolves_to_read(db):
    user = await _create_user(db)
    role = await _create_role(db)
    resource = await _create_resource(db)
    permission = await _create_permission(db, resource.id, "read", "dashboard:read")

    await _assign_role(db, user.id, role.id)
    await _grant_permission(db, role.id, permission.id)

    checker = require_permission("dashboard", "read")
    result = await checker(current_user=user, db=db)

    assert result is user


async def test_require_permission_unknown_mode_raises(db):
    with pytest.raises(ValueError, match="Unsupported permission mode: delete"):
        require_permission("users:read", "delete")


async def test_require_permission_write_granted(db):
    user = await _create_user(db)
    role = await _create_role(db)
    resource = await _create_resource(db)
    permission = await _create_permission(db, resource.id, "create", "users:create")

    await _assign_role(db, user.id, role.id)
    await _grant_permission(db, role.id, permission.id)

    checker = require_permission("users:create", "write")
    result = await checker(current_user=user, db=db)

    assert result is user


async def test_require_permission_write_denied(db):
    user = await _create_user(db)

    checker = require_permission("users:create", "write")
    with pytest.raises(HTTPException) as exc_info:
        await checker(current_user=user, db=db)

    assert exc_info.value.status_code == 403
    assert exc_info.value.detail == "无写入权限"


async def test_require_permission_admin_fallback_without_rbac_assignment(db):
    user = await _create_user(db, role=UserRole.admin)

    checker = require_permission("users:read", "read")
    result = await checker(current_user=user, db=db)

    assert result is user


async def test_require_permission_legacy_key_maps_accounts_to_users_read(db):
    user = await _create_user(db)
    role = await _create_role(db)
    resource = await _create_resource(db)
    permission = await _create_permission(db, resource.id, "read", "users:read")

    await _assign_role(db, user.id, role.id)
    await _grant_permission(db, role.id, permission.id)

    checker = require_permission("accounts", "read")
    result = await checker(current_user=user, db=db)

    assert result is user


async def test_require_permission_legacy_key_maps_user_create_to_users_create(db):
    user = await _create_user(db)
    role = await _create_role(db)
    resource = await _create_resource(db)
    permission = await _create_permission(db, resource.id, "create", "users:create")

    await _assign_role(db, user.id, role.id)
    await _grant_permission(db, role.id, permission.id)

    checker = require_permission("user:create", "write")
    result = await checker(current_user=user, db=db)

    assert result is user
