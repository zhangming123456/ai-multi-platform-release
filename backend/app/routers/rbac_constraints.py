from __future__ import annotations

from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models import (
    RBACDSDConstraint,
    RBACDSDConstraintRole,
    RBACSSDConstraint,
    RBACSSDConstraintRole,
    RBACRole,
)
from app.models.user import User

router = APIRouter(prefix="/api/v2/constraints", tags=["RBAC3 约束管理"])


class SSDConstraintResponse(BaseModel):
    id: str
    name: str
    max_roles: int
    description: Optional[str] = None
    is_active: bool

    model_config = {"from_attributes": True}


class DSDConstraintResponse(BaseModel):
    id: str
    name: str
    max_roles: int
    description: Optional[str] = None
    is_active: bool

    model_config = {"from_attributes": True}


class CreateConstraintRequest(BaseModel):
    name: str
    max_roles: int = 1
    description: Optional[str] = None


class UpdateConstraintRequest(BaseModel):
    name: Optional[str] = None
    max_roles: Optional[int] = None
    description: Optional[str] = None
    is_active: Optional[bool] = None


class ConstraintRoleRequest(BaseModel):
    role_id: str


class ConstraintRoleItem(BaseModel):
    role_id: str
    role_name: str
    role_display_name: str


@router.get("/ssd", response_model=list[SSDConstraintResponse])
async def list_ssd_constraints(
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACSSDConstraint).order_by(RBACSSDConstraint.name.asc())
    )
    return result.scalars().all()


@router.post("/ssd", response_model=SSDConstraintResponse, status_code=status.HTTP_201_CREATED)
async def create_ssd_constraint(
    body: CreateConstraintRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:create", "write")),
):
    existing = await db.execute(
        select(RBACSSDConstraint).where(RBACSSDConstraint.name == body.name)
    )
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="SSD 约束名称已存在")

    constraint = RBACSSDConstraint(
        name=body.name,
        max_roles=body.max_roles,
        description=body.description,
        is_active=True,
    )
    db.add(constraint)
    await db.commit()
    await db.refresh(constraint)
    return constraint


@router.put("/ssd/{constraint_id}", response_model=SSDConstraintResponse)
async def update_ssd_constraint(
    constraint_id: str,
    body: UpdateConstraintRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:update", "write")),
):
    result = await db.execute(
        select(RBACSSDConstraint).where(RBACSSDConstraint.id == constraint_id)
    )
    constraint = result.scalar_one_or_none()
    if not constraint:
        raise HTTPException(status_code=404, detail="SSD 约束不存在")

    if body.name is not None:
        constraint.name = body.name
    if body.max_roles is not None:
        constraint.max_roles = body.max_roles
    if body.description is not None:
        constraint.description = body.description
    if body.is_active is not None:
        constraint.is_active = body.is_active

    await db.commit()
    await db.refresh(constraint)
    return constraint


@router.delete("/ssd/{constraint_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_ssd_constraint(
    constraint_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:delete", "write")),
):
    result = await db.execute(
        select(RBACSSDConstraint).where(RBACSSDConstraint.id == constraint_id)
    )
    if not result.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="SSD 约束不存在")

    await db.execute(
        delete(RBACSSDConstraintRole).where(RBACSSDConstraintRole.constraint_id == constraint_id)
    )
    await db.execute(
        delete(RBACSSDConstraint).where(RBACSSDConstraint.id == constraint_id)
    )
    await db.commit()


@router.get("/ssd/{constraint_id}/roles", response_model=list[ConstraintRoleItem])
async def get_ssd_constraint_roles(
    constraint_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    constraint = await db.execute(
        select(RBACSSDConstraint).where(RBACSSDConstraint.id == constraint_id)
    )
    if not constraint.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="SSD 约束不存在")

    result = await db.execute(
        select(RBACSSDConstraintRole, RBACRole.name, RBACRole.display_name)
        .join(RBACRole, RBACSSDConstraintRole.role_id == RBACRole.id)
        .where(RBACSSDConstraintRole.constraint_id == constraint_id)
    )
    items = []
    for cr, role_name, role_display_name in result.all():
        items.append({
            "role_id": cr.role_id,
            "role_name": role_name,
            "role_display_name": role_display_name,
        })
    return items


@router.post("/ssd/{constraint_id}/roles", status_code=status.HTTP_201_CREATED)
async def add_ssd_constraint_role(
    constraint_id: str,
    body: ConstraintRoleRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:update", "write")),
):
    constraint = await db.execute(
        select(RBACSSDConstraint).where(RBACSSDConstraint.id == constraint_id)
    )
    if not constraint.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="SSD 约束不存在")

    role = await db.execute(select(RBACRole).where(RBACRole.id == body.role_id))
    if not role.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")

    existing = await db.execute(
        select(RBACSSDConstraintRole).where(
            RBACSSDConstraintRole.constraint_id == constraint_id,
            RBACSSDConstraintRole.role_id == body.role_id,
        )
    )
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="该角色已在此约束中")

    db.add(RBACSSDConstraintRole(constraint_id=constraint_id, role_id=body.role_id))
    await db.commit()
    return {"message": "角色已添加到 SSD 约束"}


@router.delete("/ssd/{constraint_id}/roles/{role_id}", status_code=status.HTTP_204_NO_CONTENT)
async def remove_ssd_constraint_role(
    constraint_id: str,
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:update", "write")),
):
    await db.execute(
        delete(RBACSSDConstraintRole).where(
            RBACSSDConstraintRole.constraint_id == constraint_id,
            RBACSSDConstraintRole.role_id == role_id,
        )
    )
    await db.commit()


@router.get("/dsd", response_model=list[DSDConstraintResponse])
async def list_dsd_constraints(
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACDSDConstraint).order_by(RBACDSDConstraint.name.asc())
    )
    return result.scalars().all()


@router.post("/dsd", response_model=DSDConstraintResponse, status_code=status.HTTP_201_CREATED)
async def create_dsd_constraint(
    body: CreateConstraintRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:create", "write")),
):
    existing = await db.execute(
        select(RBACDSDConstraint).where(RBACDSDConstraint.name == body.name)
    )
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="DSD 约束名称已存在")

    constraint = RBACDSDConstraint(
        name=body.name,
        max_roles=body.max_roles,
        description=body.description,
        is_active=True,
    )
    db.add(constraint)
    await db.commit()
    await db.refresh(constraint)
    return constraint


@router.put("/dsd/{constraint_id}", response_model=DSDConstraintResponse)
async def update_dsd_constraint(
    constraint_id: str,
    body: UpdateConstraintRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:update", "write")),
):
    result = await db.execute(
        select(RBACDSDConstraint).where(RBACDSDConstraint.id == constraint_id)
    )
    constraint = result.scalar_one_or_none()
    if not constraint:
        raise HTTPException(status_code=404, detail="DSD 约束不存在")

    if body.name is not None:
        constraint.name = body.name
    if body.max_roles is not None:
        constraint.max_roles = body.max_roles
    if body.description is not None:
        constraint.description = body.description
    if body.is_active is not None:
        constraint.is_active = body.is_active

    await db.commit()
    await db.refresh(constraint)
    return constraint


@router.delete("/dsd/{constraint_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_dsd_constraint(
    constraint_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:delete", "write")),
):
    result = await db.execute(
        select(RBACDSDConstraint).where(RBACDSDConstraint.id == constraint_id)
    )
    if not result.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="DSD 约束不存在")

    await db.execute(
        delete(RBACDSDConstraintRole).where(RBACDSDConstraintRole.constraint_id == constraint_id)
    )
    await db.execute(
        delete(RBACDSDConstraint).where(RBACDSDConstraint.id == constraint_id)
    )
    await db.commit()


@router.get("/dsd/{constraint_id}/roles", response_model=list[ConstraintRoleItem])
async def get_dsd_constraint_roles(
    constraint_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    constraint = await db.execute(
        select(RBACDSDConstraint).where(RBACDSDConstraint.id == constraint_id)
    )
    if not constraint.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="DSD 约束不存在")

    result = await db.execute(
        select(RBACDSDConstraintRole, RBACRole.name, RBACRole.display_name)
        .join(RBACRole, RBACDSDConstraintRole.role_id == RBACRole.id)
        .where(RBACDSDConstraintRole.constraint_id == constraint_id)
    )
    items = []
    for cr, role_name, role_display_name in result.all():
        items.append({
            "role_id": cr.role_id,
            "role_name": role_name,
            "role_display_name": role_display_name,
        })
    return items


@router.post("/dsd/{constraint_id}/roles", status_code=status.HTTP_201_CREATED)
async def add_dsd_constraint_role(
    constraint_id: str,
    body: ConstraintRoleRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:update", "write")),
):
    constraint = await db.execute(
        select(RBACDSDConstraint).where(RBACDSDConstraint.id == constraint_id)
    )
    if not constraint.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="DSD 约束不存在")

    role = await db.execute(select(RBACRole).where(RBACRole.id == body.role_id))
    if not role.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")

    existing = await db.execute(
        select(RBACDSDConstraintRole).where(
            RBACDSDConstraintRole.constraint_id == constraint_id,
            RBACDSDConstraintRole.role_id == body.role_id,
        )
    )
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="该角色已在此约束中")

    db.add(RBACDSDConstraintRole(constraint_id=constraint_id, role_id=body.role_id))
    await db.commit()
    return {"message": "角色已添加到 DSD 约束"}


@router.delete("/dsd/{constraint_id}/roles/{role_id}", status_code=status.HTTP_204_NO_CONTENT)
async def remove_dsd_constraint_role(
    constraint_id: str,
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("constraints:update", "write")),
):
    await db.execute(
        delete(RBACDSDConstraintRole).where(
            RBACDSDConstraintRole.constraint_id == constraint_id,
            RBACDSDConstraintRole.role_id == role_id,
        )
    )
    await db.commit()
