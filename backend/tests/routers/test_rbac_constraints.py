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
from app.models.rbac_constraint import RBACConstraint, RBACConstraintRoleAssociation  # noqa: E402
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.user import User, UserRole  # noqa: E402
from app.routers import rbac_constraints  # noqa: E402

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
    app.include_router(rbac_constraints.router)
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


@pytest_asyncio.fixture
async def two_roles(db_session):
    role_a = RBACRole(
        name=f"role-a-{uuid.uuid4().hex[:8]}",
        display_name="Role A",
        role_type="other",
    )
    role_b = RBACRole(
        name=f"role-b-{uuid.uuid4().hex[:8]}",
        display_name="Role B",
        role_type="other",
    )
    db_session.add_all([role_a, role_b])
    await db_session.commit()
    return role_a, role_b


async def test_list_constraints_empty(client, auth_headers):
    response = await client.get("/api/v2/constraints", headers=auth_headers)
    assert response.status_code == 200
    assert response.json() == []


async def test_create_mutual_exclusive_constraint(client, auth_headers, two_roles):
    role_a, role_b = two_roles
    response = await client.post("/api/v2/constraints", json={
        "name": "mutual-test",
        "description": "互斥约束测试",
        "constraint_type": "mutual_exclusive",
        "config": {"scope": "static"},
        "role_associations": [
            {"role_id": role_a.id, "association_type": "subject"},
            {"role_id": role_b.id, "association_type": "subject"},
        ],
    }, headers=auth_headers)
    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "mutual-test"
    assert data["constraint_type"] == "mutual_exclusive"
    assert len(data["roles"]) == 2


async def test_create_prerequisite_constraint(client, auth_headers, two_roles):
    role_a, role_b = two_roles
    response = await client.post("/api/v2/constraints", json={
        "name": "prereq-test",
        "constraint_type": "prerequisite",
        "config": {"require_all": True},
        "role_associations": [
            {"role_id": role_a.id, "association_type": "subject"},
            {"role_id": role_b.id, "association_type": "prerequisite"},
        ],
    }, headers=auth_headers)
    assert response.status_code == 201
    data = response.json()
    assert data["constraint_type"] == "prerequisite"
    association_types = {r["association_type"] for r in data["roles"]}
    assert association_types == {"subject", "prerequisite"}


async def test_create_cardinality_constraint(client, auth_headers, two_roles):
    role_a, _ = two_roles
    response = await client.post("/api/v2/constraints", json={
        "name": "cardinality-test",
        "constraint_type": "cardinality",
        "config": {"max_users": 3},
        "role_associations": [
            {"role_id": role_a.id, "association_type": "subject"},
        ],
    }, headers=auth_headers)
    assert response.status_code == 201
    data = response.json()
    assert data["config"]["max_users"] == 3


async def test_create_constraint_rejects_invalid_association_type(
    client, auth_headers, two_roles
):
    role_a, _ = two_roles
    response = await client.post("/api/v2/constraints", json={
        "name": "invalid-test",
        "constraint_type": "mutual_exclusive",
        "role_associations": [
            {"role_id": role_a.id, "association_type": "prerequisite"},
        ],
    }, headers=auth_headers)
    assert response.status_code == 400


async def test_create_constraint_rejects_missing_role(client, auth_headers):
    response = await client.post("/api/v2/constraints", json={
        "name": "missing-role-test",
        "constraint_type": "mutual_exclusive",
        "role_associations": [
            {"role_id": str(uuid.uuid4()), "association_type": "subject"},
        ],
    }, headers=auth_headers)
    assert response.status_code == 400


async def test_create_constraint_rejects_duplicate_name(client, auth_headers, two_roles):
    role_a, _ = two_roles
    response = await client.post("/api/v2/constraints", json={
        "name": "dup-name",
        "constraint_type": "mutual_exclusive",
        "role_associations": [{"role_id": role_a.id, "association_type": "subject"}],
    }, headers=auth_headers)
    assert response.status_code == 201

    response2 = await client.post("/api/v2/constraints", json={
        "name": "dup-name",
        "constraint_type": "mutual_exclusive",
        "role_associations": [{"role_id": role_a.id, "association_type": "subject"}],
    }, headers=auth_headers)
    assert response2.status_code == 409


async def test_get_constraint(client, auth_headers, two_roles):
    role_a, _ = two_roles
    create_resp = await client.post("/api/v2/constraints", json={
        "name": "get-test",
        "constraint_type": "mutual_exclusive",
        "role_associations": [{"role_id": role_a.id, "association_type": "subject"}],
    }, headers=auth_headers)
    constraint_id = create_resp.json()["id"]

    response = await client.get(f"/api/v2/constraints/{constraint_id}", headers=auth_headers)
    assert response.status_code == 200
    assert response.json()["id"] == constraint_id


async def test_get_constraint_not_found(client, auth_headers):
    response = await client.get(f"/api/v2/constraints/{uuid.uuid4()}", headers=auth_headers)
    assert response.status_code == 404


async def test_update_constraint(client, auth_headers, two_roles):
    role_a, role_b = two_roles
    create_resp = await client.post("/api/v2/constraints", json={
        "name": "update-test",
        "constraint_type": "mutual_exclusive",
        "role_associations": [{"role_id": role_a.id, "association_type": "subject"}],
    }, headers=auth_headers)
    constraint_id = create_resp.json()["id"]

    response = await client.put(f"/api/v2/constraints/{constraint_id}", json={
        "description": "updated",
        "config": {"scope": "dynamic"},
        "role_associations": [
            {"role_id": role_a.id, "association_type": "subject"},
            {"role_id": role_b.id, "association_type": "subject"},
        ],
    }, headers=auth_headers)
    assert response.status_code == 200
    data = response.json()
    assert data["description"] == "updated"
    assert data["config"] == {"scope": "dynamic"}
    assert len(data["roles"]) == 2


async def test_update_constraint_rejects_duplicate_name(
    client, auth_headers, two_roles
):
    role_a, _ = two_roles
    await client.post("/api/v2/constraints", json={
        "name": "existing-name",
        "constraint_type": "mutual_exclusive",
        "role_associations": [{"role_id": role_a.id, "association_type": "subject"}],
    }, headers=auth_headers)

    create_resp = await client.post("/api/v2/constraints", json={
        "name": "another-name",
        "constraint_type": "mutual_exclusive",
        "role_associations": [{"role_id": role_a.id, "association_type": "subject"}],
    }, headers=auth_headers)
    constraint_id = create_resp.json()["id"]

    response = await client.put(f"/api/v2/constraints/{constraint_id}", json={
        "name": "existing-name",
    }, headers=auth_headers)
    assert response.status_code == 409


async def test_delete_constraint(client, auth_headers, two_roles):
    role_a, _ = two_roles
    create_resp = await client.post("/api/v2/constraints", json={
        "name": "delete-test",
        "constraint_type": "mutual_exclusive",
        "role_associations": [{"role_id": role_a.id, "association_type": "subject"}],
    }, headers=auth_headers)
    constraint_id = create_resp.json()["id"]

    response = await client.delete(f"/api/v2/constraints/{constraint_id}", headers=auth_headers)
    assert response.status_code == 204

    associations = await client.get(f"/api/v2/constraints/{constraint_id}", headers=auth_headers)
    assert associations.status_code == 404


async def test_unauthorized_request_returns_401(client):
    response = await client.get("/api/v2/constraints")
    assert response.status_code == 401
