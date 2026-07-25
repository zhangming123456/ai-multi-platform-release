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
from app.core.security import create_access_token, hash_password  # noqa: E402
from app.database import Base, get_db  # noqa: E402
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.user import User, UserRole  # noqa: E402
from app.models.user_creation_request import UserCreationRequest, UserCreationStatus  # noqa: E402
from app.routers import rbac_users  # noqa: E402

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
    app.include_router(rbac_users.router)
    app.dependency_overrides[get_db] = override_get_db

    yield app

    await engine.dispose()
    os.unlink(db_path)


@pytest_asyncio.fixture
async def client(test_app):
    async with AsyncClient(transport=ASGITransport(app=test_app), base_url="http://test") as ac:
        yield ac


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


async def _auth_headers(test_app, role: UserRole = UserRole.admin):
    db_gen = test_app.dependency_overrides[get_db]()
    session = await db_gen.__anext__()
    try:
        user = User(
            id=str(uuid.uuid4()),
            username=f"user-{uuid.uuid4().hex[:8]}",
            email=f"{uuid.uuid4().hex[:8]}@example.com",
            hashed_password="secret",
            nickname="Test User",
            role=role,
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
        return {"Authorization": f"Bearer {token}"}, user.id
    finally:
        try:
            await db_gen.aclose()
        except Exception:
            pass


@pytest_asyncio.fixture
async def admin_headers(test_app):
    headers, _ = await _auth_headers(test_app, UserRole.admin)
    return headers


@pytest_asyncio.fixture
async def operator_headers(test_app):
    db_gen = test_app.dependency_overrides[get_db]()
    session = await db_gen.__anext__()
    try:
        user = User(
            id=str(uuid.uuid4()),
            username=f"operator-{uuid.uuid4().hex[:8]}",
            email=f"{uuid.uuid4().hex[:8]}@example.com",
            hashed_password="secret",
            nickname="Operator User",
            role=UserRole.operator,
        )
        session.add(user)
        await session.flush()

        role = RBACRole(name="creator", display_name="创建者", role_type="other")
        session.add(role)
        await session.flush()

        from app.models.rbac_resource import RBACResource
        from app.models.rbac_permission import RBACPermission
        from app.models.rbac_role_permission import RBACRolePermission

        resource = RBACResource(key="users", name="用户", type="action")
        session.add(resource)
        await session.flush()
        perm = RBACPermission(resource_id=resource.id, operation="create", key="users:create")
        session.add(perm)
        await session.flush()
        session.add(RBACRolePermission(role_id=role.id, permission_id=perm.id, grant_type="direct"))
        session.add(RBACUserRoleAssignment(user_id=user.id, role_id=role.id, grant_type="direct"))
        await session.commit()

        token = create_access_token({"sub": user.id})
        return {"Authorization": f"Bearer {token}"}
    finally:
        try:
            await db_gen.aclose()
        except Exception:
            pass


async def test_list_users(client, admin_headers, db_session):
    user = User(
        username="listuser",
        email="list@example.com",
        hashed_password="secret",
        nickname="列表用户",
        role=UserRole.operator,
    )
    db_session.add(user)
    await db_session.commit()

    response = await client.get("/api/v2/users", headers=admin_headers)
    assert response.status_code == 200
    data = response.json()
    assert any(u["username"] == "listuser" for u in data)


async def test_create_user_by_admin(client, admin_headers, db_session):
    role = RBACRole(name="operator", display_name="运营者", role_type="other", is_builtin=True)
    db_session.add(role)
    await db_session.commit()

    response = await client.post("/api/v2/users", json={
        "username": "newop",
        "password": "newop123",
        "nickname": "新运营",
        "role": "operator",
    }, headers=admin_headers)
    assert response.status_code == 201
    data = response.json()
    assert data["username"] == "newop"
    assert len(data["roles"]) == 1
    assert data["roles"][0]["name"] == "operator"


async def test_create_user_with_role_ids(client, admin_headers, db_session):
    role = RBACRole(name="custom", display_name="自定义", role_type="other")
    db_session.add(role)
    await db_session.commit()

    response = await client.post("/api/v2/users", json={
        "username": "customuser",
        "password": "custom123",
        "nickname": "自定义用户",
        "role_ids": [role.id],
    }, headers=admin_headers)
    assert response.status_code == 201
    data = response.json()
    assert any(r["id"] == role.id for r in data["roles"])


async def test_create_user_by_operator_submits_review(client, operator_headers, db_session):
    response = await client.post("/api/v2/users", json={
        "username": "reviewuser",
        "password": "review123",
        "nickname": "审核用户",
        "role": "operator",
    }, headers=operator_headers)
    assert response.status_code == 202


async def test_create_user_rejects_admin(client, admin_headers):
    response = await client.post("/api/v2/users", json={
        "username": "newadmin",
        "password": "admin123",
        "nickname": "新管理员",
        "role": "admin",
    }, headers=admin_headers)
    assert response.status_code == 400


async def test_get_user(client, admin_headers, db_session):
    user = User(
        username="getuser",
        email="get@example.com",
        hashed_password="secret",
        nickname="查询用户",
        role=UserRole.operator,
    )
    db_session.add(user)
    await db_session.commit()

    response = await client.get(f"/api/v2/users/{user.id}", headers=admin_headers)
    assert response.status_code == 200
    data = response.json()
    assert data["username"] == "getuser"


async def test_update_user_roles(client, admin_headers, db_session):
    user = User(
        username="roleuser",
        email="role@example.com",
        hashed_password="secret",
        nickname="角色用户",
        role=UserRole.operator,
    )
    role = RBACRole(name="editor", display_name="编辑", role_type="other")
    db_session.add_all([user, role])
    await db_session.commit()

    response = await client.put(
        f"/api/v2/users/{user.id}/roles",
        json={"role_ids": [role.id]},
        headers=admin_headers,
    )
    assert response.status_code == 200
    data = response.json()
    assert any(r["id"] == role.id for r in data["roles"])


async def test_update_user_roles_rejects_constraint(client, admin_headers, db_session):
    role_a = RBACRole(name="a", display_name="A", role_type="other")
    role_b = RBACRole(name="b", display_name="B", role_type="other")
    user = User(
        username="constraintuser",
        email="constraint@example.com",
        hashed_password="secret",
        nickname="约束用户",
        role=UserRole.operator,
    )
    db_session.add_all([role_a, role_b, user])
    await db_session.commit()

    from app.models.rbac_constraint import RBACConstraint, RBACConstraintRoleAssociation
    constraint = RBACConstraint(
        name="mutual",
        constraint_type="mutual_exclusive",
        config={"scope": "static"},
    )
    db_session.add(constraint)
    await db_session.flush()
    db_session.add_all([
        RBACConstraintRoleAssociation(constraint_id=constraint.id, role_id=role_a.id, association_type="subject"),
        RBACConstraintRoleAssociation(constraint_id=constraint.id, role_id=role_b.id, association_type="subject"),
    ])
    await db_session.commit()

    response = await client.put(
        f"/api/v2/users/{user.id}/roles",
        json={"role_ids": [role_a.id, role_b.id]},
        headers=admin_headers,
    )
    assert response.status_code == 400


async def test_change_password(client, admin_headers, db_session):
    user = User(
        username="pwduser",
        email="pwd@example.com",
        hashed_password=hash_password("oldpass1"),
        nickname="密码用户",
        role=UserRole.operator,
    )
    db_session.add(user)
    await db_session.commit()

    response = await client.put(
        f"/api/v2/users/{user.id}/password",
        json={"new_password": "newpass1"},
        headers=admin_headers,
    )
    assert response.status_code == 200


async def test_delete_user(client, admin_headers, db_session):
    user = User(
        username="deluser",
        email="del@example.com",
        hashed_password="secret",
        nickname="删除用户",
        role=UserRole.operator,
    )
    db_session.add(user)
    await db_session.commit()

    response = await client.delete(f"/api/v2/users/{user.id}", headers=admin_headers)
    assert response.status_code == 204


async def test_unauthorized_request_returns_401(client):
    response = await client.get("/api/v2/users")
    assert response.status_code == 401
