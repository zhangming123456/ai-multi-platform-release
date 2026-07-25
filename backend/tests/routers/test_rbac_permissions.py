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
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.user import User, UserRole  # noqa: E402
from app.routers import rbac_permissions  # noqa: E402

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
    app.include_router(rbac_permissions.router)
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


async def test_list_resources_returns_tree(client, auth_headers):
    # Use the API to create the hierarchy, which also exercises POST.
    parent_resp = await client.post("/api/v2/resources", json={
        "key": "tree-parent",
        "name": "Parent",
        "type": "page",
    }, headers=auth_headers)
    assert parent_resp.status_code == 201
    parent_id = parent_resp.json()["id"]

    child_resp = await client.post("/api/v2/resources", json={
        "key": "tree-child",
        "name": "Child",
        "type": "action",
        "parent_id": parent_id,
    }, headers=auth_headers)
    assert child_resp.status_code == 201

    response = await client.get("/api/v2/resources", headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert isinstance(data, list)

    root = next((node for node in data if node["key"] == "tree-parent"), None)
    assert root is not None
    assert root["parent_id"] is None
    assert any(child["key"] == "tree-child" for child in root["children"])

    child_in_tree = next(
        (child for child in root["children"] if child["key"] == "tree-child"), None
    )
    assert child_in_tree is not None
    assert child_in_tree["parent_id"] == parent_id


async def test_list_permissions_returns_permissions_with_resources(client, auth_headers):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "perm-resource",
        "name": "Permission Resource",
        "type": "page",
    }, headers=auth_headers)
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    perm_resp = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "read",
        "key": "perm-resource:read",
    }, headers=auth_headers)
    assert perm_resp.status_code == 201

    response = await client.get("/api/v2/permissions", headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert isinstance(data, list)

    item = next((p for p in data if p["key"] == "perm-resource:read"), None)
    assert item is not None
    assert item["operation"] == "read"
    assert item["resource"]["id"] == resource["id"]
    assert item["resource"]["key"] == "perm-resource"
    assert item["resource"]["name"] == "Permission Resource"
    assert item["resource"]["type"] == "page"


async def test_create_resource_creates_and_rejects_duplicate(client, auth_headers):
    response = await client.post("/api/v2/resources", json={
        "key": "unique-resource",
        "name": "Unique Resource",
        "type": "page",
    }, headers=auth_headers)
    assert response.status_code == 201
    data = response.json()
    assert data["key"] == "unique-resource"
    assert data["name"] == "Unique Resource"
    assert data["type"] == "page"
    assert data["is_active"] is True
    assert data["children"] == []

    duplicate = await client.post("/api/v2/resources", json={
        "key": "unique-resource",
        "name": "Duplicate",
        "type": "page",
    }, headers=auth_headers)
    assert duplicate.status_code == 409


async def test_create_resource_rejects_invalid_parent(client, auth_headers):
    response = await client.post("/api/v2/resources", json={
        "key": "orphan-resource",
        "name": "Orphan",
        "type": "page",
        "parent_id": "non-existent-parent-id",
    }, headers=auth_headers)
    assert response.status_code == 400


async def test_create_permission_creates_and_generates_key(client, auth_headers):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "posts",
        "name": "Posts",
        "type": "page",
    }, headers=auth_headers)
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    response = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "create",
    }, headers=auth_headers)
    assert response.status_code == 201
    data = response.json()
    assert data["key"] == "posts:create"
    assert data["operation"] == "create"
    assert data["resource"]["id"] == resource["id"]
    assert data["resource"]["key"] == "posts"


async def test_create_permission_rejects_duplicate_key(client, auth_headers):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "articles",
        "name": "Articles",
        "type": "page",
    }, headers=auth_headers)
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    first = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "delete",
        "key": "articles:delete",
    }, headers=auth_headers)
    assert first.status_code == 201

    duplicate = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "delete",
        "key": "articles:delete",
    }, headers=auth_headers)
    assert duplicate.status_code == 409


async def test_create_permission_rejects_invalid_resource(client, auth_headers):
    response = await client.post("/api/v2/permissions", json={
        "resource_id": "non-existent-resource-id",
        "operation": "read",
    }, headers=auth_headers)
    assert response.status_code == 400


async def test_unauthorized_request_returns_401(client):
    response = await client.get("/api/v2/resources")
    assert response.status_code == 401

    response = await client.post("/api/v2/resources", json={
        "key": "no-auth",
        "name": "No Auth",
        "type": "page",
    })
    assert response.status_code == 401


async def test_create_resource_rejects_invalid_type(client, auth_headers):
    response = await client.post("/api/v2/resources", json={
        "key": "bad-type",
        "name": "Bad Type",
        "type": "invalid",
    }, headers=auth_headers)
    assert response.status_code == 422


async def test_create_permission_rejects_invalid_operation(client, auth_headers):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "perm-op",
        "name": "Perm Op",
        "type": "page",
    }, headers=auth_headers)
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    response = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "invalid",
    }, headers=auth_headers)
    assert response.status_code == 422


async def test_create_permission_rejects_invalid_key_format(client, auth_headers):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "perm-key",
        "name": "Perm Key",
        "type": "page",
    }, headers=auth_headers)
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    response = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "read",
        "key": "invalid-key",
    }, headers=auth_headers)
    assert response.status_code == 400
