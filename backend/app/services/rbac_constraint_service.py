from __future__ import annotations

from typing import Optional

from fastapi import HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    RBACDSDConstraint,
    RBACDSDConstraintRole,
    RBACSSDConstraint,
    RBACSSDConstraintRole,
)
from app.services.rbac_service import get_assigned_roles


class SSDViolationError(HTTPException):
    def __init__(self, message: str):
        super().__init__(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=message,
        )


class DSDViolationError(HTTPException):
    def __init__(self, message: str):
        super().__init__(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=message,
        )


async def get_ssd_constraint_role_ids(
    constraint_id: str,
    db: AsyncSession,
) -> set[str]:
    result = await db.execute(
        select(RBACSSDConstraintRole.role_id).where(
            RBACSSDConstraintRole.constraint_id == constraint_id
        )
    )
    return {row[0] for row in result.all()}


async def get_dsd_constraint_role_ids(
    constraint_id: str,
    db: AsyncSession,
) -> set[str]:
    result = await db.execute(
        select(RBACDSDConstraintRole.role_id).where(
            RBACDSDConstraintRole.constraint_id == constraint_id
        )
    )
    return {row[0] for row in result.all()}


async def validate_ssd(
    user_id: str,
    db: AsyncSession,
    new_role_id: Optional[str] = None,
):
    assigned_roles = await get_assigned_roles(user_id, db)
    user_role_ids = {r.id for r in assigned_roles}
    if new_role_id:
        user_role_ids.add(new_role_id)

    result = await db.execute(
        select(RBACSSDConstraint).where(RBACSSDConstraint.is_active == True)
    )
    constraints = result.scalars().all()

    for constraint in constraints:
        constraint_role_ids = await get_ssd_constraint_role_ids(constraint.id, db)
        count = len(user_role_ids & constraint_role_ids)
        if count > constraint.max_roles:
            raise SSDViolationError(
                f"违反静态职责分离约束「{constraint.name}」: "
                f"用户在该角色集合中最多只能拥有 {constraint.max_roles} 个角色"
            )


async def validate_dsd(
    active_role_ids: list[str],
    db: AsyncSession,
):
    role_id_set = set(active_role_ids)

    result = await db.execute(
        select(RBACDSDConstraint).where(RBACDSDConstraint.is_active == True)
    )
    constraints = result.scalars().all()

    for constraint in constraints:
        constraint_role_ids = await get_dsd_constraint_role_ids(constraint.id, db)
        count = len(role_id_set & constraint_role_ids)
        if count > constraint.max_roles:
            raise DSDViolationError(
                f"违反动态职责分离约束「{constraint.name}」: "
                f"当前会话在该角色集合中最多只能激活 {constraint.max_roles} 个角色"
            )
