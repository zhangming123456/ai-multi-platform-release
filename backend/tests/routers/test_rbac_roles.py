from __future__ import annotations

import os
import tempfile
import uuid

import pytest
import pytest_asyncio
from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

import app.models  # noqa: F401,E402
from app.core.security import create_access_token  # noqa: E402
from app.database import Base, get_db  # noqa: E402
from app.models.rbac_permission import RBACPermission  # noqa: E402
from app.models.rbac_resource import RBACResource  # noqa: E402
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_role_hierarchy import RBACRoleHierarchy  # noqa: E402
from app.models.rbac_role_permission import RBACRolePermission  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.user import User, UserRole  # noqa: E402
from app.routers import rbac_roles  # noqa: E402

pytestmark = pytest.mark.asyncio


@pytest_asyncio.fixture
async def test_app():
    fd, db_path = tempfile.mkstemp(suffix=".db")
    os.close(fd)
    engine = create_async_engine(f"sqlite+aiosqlite:///{db_path}")
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    session_factory = async_sessionmaker(
        engine, class_=AsyncSession, expire_on_commit=False
    )

    async def override_get_db():
        async with session_factory() as session:
            try:
                yield session
                await session.commit()
            except Exception:
                await session.rollback()
                raise
            finally:
                await session.close()

    app = FastAPI()
    app.include_router(rbac_roles.router)
    app.dependency_overrides[get_db] = override_get_db

    yield app

    await engine.dispose()
    os.unlink(db_path)


@pytest_asyncio.fixture
async def client(test_app):
    async with AsyncClient(transport=ASGITransport(app=test_app), base_url="http://test") as ac:
        yield ac


@pytest_asyncio.fixture
async def auth_headers(test_app):
    db_gen = test_app.dependency_overrides[get_db]()
    session = await db_gen.__anext__()
    try:
        user = User(
            id=str(uuid.uuid4()),
            username=f"admin-{uuid.uuid4().hex[:8]}",
            email=f"{uuid.uuid4().hex[:8]}@example.com",
            hashed_password="secret",
            nickname="Admin User",
            role=UserRole.admin,
        )
        session.add(user)
        await session.flush()

        admin_role = RBACRole(
            name="admin",
            display_name="超级管理员",
            role_type="admin",
            is_super_admin=True,
            is_builtin=True,
        )
        session.add(admin_role)
        await session.flush()
        session.add(
            RBACUserRoleAssignment(
                user_id=user.id, role_id=admin_role.id, grant_type="direct"
            )
        )
        await session.commit()
        token = create_access_token({"sub": user.id})
        return {"Authorization": f"Bearer {token}"}
    finally:
        try:
            await db_gen.aclose()
        except Exception:
            pass


@pytest_asyncio.fixture
async def db_session(test_app):
    db_gen = test_app.dependency_overrides[get_db]()
    session = await db_gen.__anext__()
    try:
        yield session
    finally:
        try:
            await db_gen.aclose()
        except Exception:
            pass


async def test_list_roles_returns_admin_fixture(client, auth_headers):
    response = await client.get("/api/v2/roles", headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert len(data) == 1
    assert data[0]["name"] == "admin"


async def test_create_role(client, auth_headers):
    response = await client.post("/api/v2/roles", json={
        "name": "editor",
        "display_name": "编辑",
        "description": "内容编辑角色",
        "role_type": "other",
    }, headers=auth_headers)
    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "editor"
    assert data["display_name"] == "编辑"
    assert data["role_type"] == "other"
    assert data["is_builtin"] is False
    assert data["parent_roles"] == []
    assert data["child_roles"] == []


async def test_create_role_rejects_duplicate_name(client, auth_headers):
    payload = {
        "name": "duplicate-role",
        "display_name": "重复角色",
        "role_type": "other",
    }
    first = await client.post("/api/v2/roles", json=payload, headers=auth_headers)
    assert first.status_code == 201

    second = await client.post("/api/v2/roles", json=payload, headers=auth_headers)
    assert second.status_code == 409


async def test_get_role(client, auth_headers):
    create_resp = await client.post("/api/v2/roles", json={
        "name": "viewer",
        "display_name": "查看者",
        "role_type": "other",
    }, headers=auth_headers)
    role_id = create_resp.json()["id"]

    response = await client.get(f"/api/v2/roles/{role_id}", headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert data["id"] == role_id
    assert data["name"] == "viewer"


async def test_get_role_not_found(client, auth_headers):
    response = await client.get(f"/api/v2/roles/{uuid.uuid4()}", headers=auth_headers)
    assert response.status_code == 404


async def test_update_role(client, auth_headers):
    create_resp = await client.post("/api/v2/roles", json={
        "name": "updatable",
        "display_name": "可更新",
        "role_type": "other",
    }, headers=auth_headers)
    role_id = create_resp.json()["id"]

    response = await client.put(f"/api/v2/roles/{role_id}", json={
        "display_name": "已更新",
        "description": "更新后的描述",
    }, headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert data["display_name"] == "已更新"
    assert data["description"] == "更新后的描述"


async def test_delete_role(client, auth_headers, db_session):
    role = RBACRole(
        name="deletable",
        display_name="可删除",
        role_type="other",
        is_builtin=False,
        is_super_admin=False,
    )
    db_session.add(role)
    await db_session.commit()

    response = await client.delete(f"/api/v2/roles/{role.id}", headers=auth_headers)
    assert response.status_code == 204

    get_resp = await client.get(f"/api/v2/roles/{role.id}", headers=auth_headers)
    assert get_resp.status_code == 404


async def test_delete_builtin_role_forbidden(client, auth_headers, db_session):
    role = RBACRole(
        name="builtin-test",
        display_name="内置测试",
        role_type="admin",
        is_builtin=True,
        is_super_admin=False,
    )
    db_session.add(role)
    await db_session.commit()

    response = await client.delete(f"/api/v2/roles/{role.id}", headers=auth_headers)
    assert response.status_code == 400


async def test_add_parent_role(client, auth_headers, db_session):
    parent = RBACRole(name="parent", display_name="父角色", role_type="other")
    child = RBACRole(name="child", display_name="子角色", role_type="other")
    db_session.add_all([parent, child])
    await db_session.commit()

    response = await client.post(
        f"/api/v2/roles/{child.id}/parents",
        json={"id": parent.id, "name": parent.name, "display_name": parent.display_name},
        headers=auth_headers,
    )
    assert response.status_code == 200
    data = response.json()
    assert any(p["id"] == parent.id for p in data["parent_roles"])


async def test_add_parent_role_rejects_cycle(client, auth_headers, db_session):
    role_a = RBACRole(name="a", display_name="A", role_type="other")
    role_b = RBACRole(name="b", display_name="B", role_type="other")
    db_session.add_all([role_a, role_b])
    await db_session.commit()

    db_session.add(RBACRoleHierarchy(parent_role_id=role_a.id, child_role_id=role_b.id))
    await db_session.commit()

    response = await client.post(
        f"/api/v2/roles/{role_a.id}/parents",
        json={"id": role_b.id, "name": role_b.name, "display_name": role_b.display_name},
        headers=auth_headers,
    )
    assert response.status_code == 400


async def test_remove_parent_role(client, auth_headers, db_session):
    parent = RBACRole(name="parent-r", display_name="父角色", role_type="other")
    child = RBACRole(name="child-r", display_name="子角色", role_type="other")
    db_session.add_all([parent, child])
    await db_session.commit()

    db_session.add(RBACRoleHierarchy(parent_role_id=parent.id, child_role_id=child.id))
    await db_session.commit()

    response = await client.delete(
        f"/api/v2/roles/{child.id}/parents/{parent.id}",
        headers=auth_headers,
    )
    assert response.status_code == 200
    data = response.json()
    assert data["parent_roles"] == []


async def test_role_permissions_inheritance(client, auth_headers, db_session):
    parent = RBACRole(name="parent-p", display_name="父角色", role_type="other")
    child = RBACRole(name="child-p", display_name="子角色", role_type="other")
    db_session.add_all([parent, child])
    await db_session.commit()

    resource = RBACResource(key="posts", name="文章", type="page")
    db_session.add(resource)
    await db_session.flush()

    perm = RBACPermission(resource_id=resource.id, operation="read", key="posts:read")
    db_session.add(perm)
    await db_session.flush()

    db_session.add(RBACRolePermission(role_id=parent.id, permission_id=perm.id, grant_type="direct"))
    db_session.add(RBACRoleHierarchy(parent_role_id=parent.id, child_role_id=child.id))
    await db_session.commit()

    response = await client.get(f"/api/v2/roles/{child.id}/permissions", headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert len(data) == 1
    assert data[0]["key"] == "posts:read"
    assert data[0]["grant_type"] == "inherited"


async def test_role_permissions_direct(client, auth_headers, db_session):
    role = RBACRole(name="direct-role", display_name="直接权限角色", role_type="other")
    db_session.add(role)
    await db_session.flush()

    resource = RBACResource(key="articles", name="文章", type="page")
    db_session.add(resource)
    await db_session.flush()

    perm = RBACPermission(resource_id=resource.id, operation="write", key="articles:write")
    db_session.add(perm)
    await db_session.flush()

    db_session.add(RBACRolePermission(role_id=role.id, permission_id=perm.id, grant_type="direct"))
    await db_session.commit()

    response = await client.get(f"/api/v2/roles/{role.id}/permissions/direct", headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert len(data) == 1
    assert data[0]["key"] == "articles:write"
    assert data[0]["grant_type"] == "direct"


async def test_update_role_permissions(client, auth_headers, db_session):
    role = RBACRole(name="updatable-perm", display_name="可更新权限", role_type="other")
    db_session.add(role)
    await db_session.flush()

    resource = RBACResource(key="docs", name="文档", type="page")
    db_session.add(resource)
    await db_session.flush()

    perm = RBACPermission(resource_id=resource.id, operation="read", key="docs:read")
    db_session.add(perm)
    await db_session.commit()

    response = await client.put(
        f"/api/v2/roles/{role.id}/permissions",
        json={"permission_ids": [perm.id]},
        headers=auth_headers,
    )
    assert response.status_code == 200
    data = response.json()
    assert len(data) == 1
    assert data[0]["key"] == "docs:read"


async def test_update_role_permissions_rejects_super_admin(client, auth_headers, db_session):
    role = RBACRole(
        name="super",
        display_name="超级管理员",
        role_type="admin",
        is_super_admin=True,
        is_builtin=True,
    )
    db_session.add(role)
    await db_session.commit()

    response = await client.put(
        f"/api/v2/roles/{role.id}/permissions",
        json={"permission_ids": []},
        headers=auth_headers,
    )
    assert response.status_code == 400


async def test_update_role_permissions_rejects_invalid_ids(client, auth_headers, db_session):
    role = RBACRole(name="invalid-perm", display_name="无效权限", role_type="other")
    db_session.add(role)
    await db_session.commit()

    response = await client.put(
        f"/api/v2/roles/{role.id}/permissions",
        json={"permission_ids": [str(uuid.uuid4())]},
        headers=auth_headers,
    )
    assert response.status_code == 400


async def test_unauthorized_request_returns_401(client):
    response = await client.get("/api/v2/roles")
    assert response.status_code == 401
