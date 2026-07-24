from __future__ import annotations

import os
import tempfile

import pytest
import pytest_asyncio
from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

import app.models  # noqa: F401,E402
from app.database import Base, get_db  # noqa: E402
from app.routers import rbac_permissions  # noqa: E402

pytestmark = pytest.mark.asyncio


@pytest_asyncio.fixture
async def client():
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

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as ac:
        yield ac

    await engine.dispose()
    os.unlink(db_path)


async def test_list_resources_returns_tree(client):
    # Use the API to create the hierarchy, which also exercises POST.
    parent_resp = await client.post("/api/v2/resources", json={
        "key": "tree-parent",
        "name": "Parent",
        "type": "page",
    })
    assert parent_resp.status_code == 201
    parent_id = parent_resp.json()["id"]

    child_resp = await client.post("/api/v2/resources", json={
        "key": "tree-child",
        "name": "Child",
        "type": "action",
        "parent_id": parent_id,
    })
    assert child_resp.status_code == 201

    response = await client.get("/api/v2/resources")
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


async def test_list_permissions_returns_permissions_with_resources(client):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "perm-resource",
        "name": "Permission Resource",
        "type": "page",
    })
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    perm_resp = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "read",
        "key": "perm-resource:read",
    })
    assert perm_resp.status_code == 201

    response = await client.get("/api/v2/permissions")
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


async def test_create_resource_creates_and_rejects_duplicate(client):
    response = await client.post("/api/v2/resources", json={
        "key": "unique-resource",
        "name": "Unique Resource",
        "type": "page",
    })
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
    })
    assert duplicate.status_code == 409


async def test_create_resource_rejects_invalid_parent(client):
    response = await client.post("/api/v2/resources", json={
        "key": "orphan-resource",
        "name": "Orphan",
        "type": "page",
        "parent_id": "non-existent-parent-id",
    })
    assert response.status_code == 400


async def test_create_permission_creates_and_generates_key(client):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "posts",
        "name": "Posts",
        "type": "page",
    })
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    response = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "create",
    })
    assert response.status_code == 201
    data = response.json()
    assert data["key"] == "posts:create"
    assert data["operation"] == "create"
    assert data["resource"]["id"] == resource["id"]
    assert data["resource"]["key"] == "posts"


async def test_create_permission_rejects_duplicate_key(client):
    resource_resp = await client.post("/api/v2/resources", json={
        "key": "articles",
        "name": "Articles",
        "type": "page",
    })
    assert resource_resp.status_code == 201
    resource = resource_resp.json()

    first = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "delete",
        "key": "articles:delete",
    })
    assert first.status_code == 201

    duplicate = await client.post("/api/v2/permissions", json={
        "resource_id": resource["id"],
        "operation": "delete",
        "key": "articles:delete",
    })
    assert duplicate.status_code == 409


async def test_create_permission_rejects_invalid_resource(client):
    response = await client.post("/api/v2/permissions", json={
        "resource_id": "non-existent-resource-id",
        "operation": "read",
    })
    assert response.status_code == 400
