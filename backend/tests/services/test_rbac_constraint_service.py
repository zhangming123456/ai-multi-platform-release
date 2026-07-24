from __future__ import annotations

import os
import tempfile
import uuid
from datetime import datetime, timedelta

import pytest
import pytest_asyncio
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

import app.models  # noqa: F401,E402
from app.database import Base  # noqa: E402
from app.models.rbac_constraint import (  # noqa: E402
    RBACConstraint,
    RBACConstraintRoleAssociation,
)
from app.models.rbac_role import RBACRole  # noqa: E402
from app.models.rbac_role_hierarchy import RBACRoleHierarchy  # noqa: E402
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment  # noqa: E402
from app.models.user import User  # noqa: E402
from app.services.rbac_constraint_service import (  # noqa: E402
    validate_role_hierarchy,
    validate_user_role_assignments,
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


async def _create_role(session: AsyncSession) -> RBACRole:
    role = RBACRole(
        id=str(uuid.uuid4()),
        name=f"role-{uuid.uuid4().hex[:8]}",
        display_name="Test Role",
    )
    session.add(role)
    await session.commit()
    return role


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


async def _create_constraint(
    session: AsyncSession,
    constraint_type: str,
    *,
    config: dict | None = None,
    is_active: bool = True,
) -> RBACConstraint:
    constraint = RBACConstraint(
        id=str(uuid.uuid4()),
        name=f"constraint-{uuid.uuid4().hex[:8]}",
        constraint_type=constraint_type,
        config=config or {},
        is_active=is_active,
    )
    session.add(constraint)
    await session.commit()
    return constraint


async def _add_constraint_role(
    session: AsyncSession,
    constraint_id: str,
    role_id: str,
    association_type: str,
) -> RBACConstraintRoleAssociation:
    association = RBACConstraintRoleAssociation(
        id=str(uuid.uuid4()),
        constraint_id=constraint_id,
        role_id=role_id,
        association_type=association_type,
    )
    session.add(association)
    await session.commit()
    return association


async def test_validate_role_hierarchy_self_loop_is_invalid(db):
    role = await _create_role(db)
    assert await validate_role_hierarchy(role.id, role.id, db) is False


async def test_validate_role_hierarchy_allows_unrelated_roles(db):
    role_a = await _create_role(db)
    role_b = await _create_role(db)
    assert await validate_role_hierarchy(role_a.id, role_b.id, db) is True


async def test_validate_role_hierarchy_detects_multi_level_cycle(db):
    role_a = await _create_role(db)
    role_b = await _create_role(db)
    role_c = await _create_role(db)

    await _add_hierarchy(db, role_a.id, role_b.id)
    await _add_hierarchy(db, role_b.id, role_c.id)

    # Adding role_c -> role_a would close a cycle (A -> B -> C -> A),
    # so it must be rejected.
    assert await validate_role_hierarchy(role_c.id, role_a.id, db) is False


async def test_validate_role_hierarchy_allows_new_parent(db):
    role_a = await _create_role(db)
    role_b = await _create_role(db)
    role_c = await _create_role(db)

    await _add_hierarchy(db, role_a.id, role_b.id)
    await _add_hierarchy(db, role_b.id, role_c.id)

    # role_a is an ancestor of role_c, so adding role_a -> role_c is a
    # valid new parent relationship that does not create a cycle.
    assert await validate_role_hierarchy(role_a.id, role_c.id, db) is True


async def test_validate_role_hierarchy_redundant_edge(db):
    role_a = await _create_role(db)
    role_b = await _create_role(db)

    await _add_hierarchy(db, role_a.id, role_b.id)

    # Adding the same edge again is a redundant edge. The current
    # implementation returns True because it only rejects cycles and
    # self-loops; callers that want to avoid duplicate edges should
    # check for an existing RBACRoleHierarchy row separately.
    assert await validate_role_hierarchy(role_a.id, role_b.id, db) is True


async def test_validate_user_role_assignments_mutual_exclusive_violation(db):
    user = await _create_user(db)
    role_a = await _create_role(db)
    role_b = await _create_role(db)

    constraint = await _create_constraint(db, "mutual_exclusive")
    await _add_constraint_role(db, constraint.id, role_a.id, "subject")
    await _add_constraint_role(db, constraint.id, role_b.id, "subject")

    violations = await validate_user_role_assignments(
        user.id, {role_a.id, role_b.id}, db
    )

    assert len(violations) == 1
    assert "互斥" in violations[0]
    assert role_a.name in violations[0]
    assert role_b.name in violations[0]


async def test_validate_user_role_assignments_mutual_exclusive_passes_with_one_role(
    db,
):
    user = await _create_user(db)
    role_a = await _create_role(db)
    role_b = await _create_role(db)

    constraint = await _create_constraint(db, "mutual_exclusive")
    await _add_constraint_role(db, constraint.id, role_a.id, "subject")
    await _add_constraint_role(db, constraint.id, role_b.id, "subject")

    violations = await validate_user_role_assignments(user.id, {role_a.id}, db)
    assert violations == []


async def test_validate_user_role_assignments_prerequisite_violation(db):
    user = await _create_user(db)
    subject_role = await _create_role(db)
    prerequisite_role = await _create_role(db)

    constraint = await _create_constraint(db, "prerequisite")
    await _add_constraint_role(db, constraint.id, subject_role.id, "subject")
    await _add_constraint_role(
        db, constraint.id, prerequisite_role.id, "prerequisite"
    )

    violations = await validate_user_role_assignments(
        user.id, {subject_role.id}, db
    )

    assert len(violations) == 1
    assert "需要先拥有" in violations[0]
    assert subject_role.name in violations[0]
    assert prerequisite_role.name in violations[0]


async def test_validate_user_role_assignments_prerequisite_passes_when_met(db):
    user = await _create_user(db)
    subject_role = await _create_role(db)
    prerequisite_role = await _create_role(db)

    constraint = await _create_constraint(db, "prerequisite")
    await _add_constraint_role(db, constraint.id, subject_role.id, "subject")
    await _add_constraint_role(
        db, constraint.id, prerequisite_role.id, "prerequisite"
    )

    violations = await validate_user_role_assignments(
        user.id, {subject_role.id, prerequisite_role.id}, db
    )
    assert violations == []


async def test_validate_user_role_assignments_cardinality_violation(db):
    user = await _create_user(db)
    other_user = await _create_user(db)
    role = await _create_role(db)

    constraint = await _create_constraint(db, "cardinality", config={"max_users": 1})
    await _add_constraint_role(db, constraint.id, role.id, "subject")
    await _assign_role(db, other_user.id, role.id)

    violations = await validate_user_role_assignments(user.id, {role.id}, db)

    assert len(violations) == 1
    assert "最多只能分配给 1 个用户" in violations[0]
    assert role.name in violations[0]


async def test_validate_user_role_assignments_cardinality_passes_under_limit(db):
    user = await _create_user(db)
    other_user = await _create_user(db)
    role = await _create_role(db)

    constraint = await _create_constraint(db, "cardinality", config={"max_users": 2})
    await _add_constraint_role(db, constraint.id, role.id, "subject")
    await _assign_role(db, other_user.id, role.id)

    violations = await validate_user_role_assignments(user.id, {role.id}, db)
    assert violations == []


async def test_validate_user_role_assignments_ignores_inactive_constraints(db):
    user = await _create_user(db)
    role_a = await _create_role(db)
    role_b = await _create_role(db)

    constraint = await _create_constraint(
        db, "mutual_exclusive", is_active=False
    )
    await _add_constraint_role(db, constraint.id, role_a.id, "subject")
    await _add_constraint_role(db, constraint.id, role_b.id, "subject")

    violations = await validate_user_role_assignments(
        user.id, {role_a.id, role_b.id}, db
    )
    assert violations == []


async def test_validate_user_role_assignments_respects_validity_window(db):
    user = await _create_user(db)
    subject_role = await _create_role(db)
    prerequisite_role = await _create_role(db)

    constraint = await _create_constraint(db, "prerequisite")
    await _add_constraint_role(db, constraint.id, subject_role.id, "subject")
    await _add_constraint_role(
        db, constraint.id, prerequisite_role.id, "prerequisite"
    )

    now = datetime.utcnow()
    # The prerequisite assignment is not yet valid, so the check should fail.
    await _assign_role(
        db,
        user.id,
        prerequisite_role.id,
        valid_from=now + timedelta(days=1),
    )

    violations = await validate_user_role_assignments(
        user.id, {subject_role.id}, db
    )

    assert len(violations) == 1
    assert "需要先拥有" in violations[0]
