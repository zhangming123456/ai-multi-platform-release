from __future__ import annotations

from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_admin_user
from app.database import get_db
from app.models.custom_role import CustomRole
from app.models.role_permission import RolePermission
from app.models.user import User, UserRole

router = APIRouter(prefix="/api/roles", tags=["角色管理"])


class RoleDef(BaseModel):
    name: str
    display_name: str
    description: Optional[str] = None
    is_builtin: bool = False
    is_super_admin: bool = False
    role_type: str = "other"

    model_config = {"from_attributes": True}


class CreateRoleRequest(BaseModel):
    name: str
    display_name: str
    description: Optional[str] = None
    role_type: str = "other"


class UpdateRoleRequest(BaseModel):
    display_name: Optional[str] = None
    description: Optional[str] = None
    role_type: Optional[str] = None


BUILTIN_ROLES: dict[str, dict] = {
    "admin": {"name": "admin", "display_name": "超级管理员", "description": "系统最高权限，拥有所有页面和操作权限，不可修改", "role_type": "admin"},
    "manager": {"name": "manager", "display_name": "管理员", "description": "管理系统配置、用户权限和日常运营", "role_type": "admin"},
    "operator": {"name": "operator", "display_name": "运营者", "description": "日常内容运营与发布管理", "role_type": "other"},
    "reviewer": {"name": "reviewer", "display_name": "审核员", "description": "内容与SQL变更审核", "role_type": "other"},
}

BUILTIN_ROLE_NAMES = list(BUILTIN_ROLES.keys())


def _is_builtin(name: str) -> bool:
    return name in BUILTIN_ROLE_NAMES


@router.get("", response_model=list[RoleDef])
async def list_roles(
    db: AsyncSession = Depends(get_db),
):
    roles: list[RoleDef] = []

    for role_name, info in BUILTIN_ROLES.items():
        roles.append(RoleDef(
            name=info["name"],
            display_name=info["display_name"],
            description=info["description"] or "",
            is_builtin=True,
            is_super_admin=(role_name == "admin"),
            role_type=info["role_type"],
        ))

    result = await db.execute(select(CustomRole).order_by(CustomRole.created_at.asc()))
    for cr in result.scalars().all():
        roles.append(RoleDef(
            name=cr.name,
            display_name=cr.display_name,
            description=cr.description or "",
            is_builtin=False,
            role_type=cr.role_type,
        ))

    return roles


@router.post("", response_model=RoleDef, status_code=status.HTTP_201_CREATED)
async def create_role(
    body: CreateRoleRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(get_admin_user),
):
    if _is_builtin(body.name):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="角色名称与内置角色冲突",
        )

    existing = await db.execute(select(CustomRole).where(CustomRole.name == body.name))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="该角色名称已存在",
        )

    cr = CustomRole(
        name=body.name,
        display_name=body.display_name,
        description=body.description,
        role_type=body.role_type,
    )
    db.add(cr)
    await db.flush()
    await db.refresh(cr)

    return RoleDef(
        name=cr.name,
        display_name=cr.display_name,
        description=cr.description,
        is_builtin=False,
        role_type=cr.role_type,
    )


@router.put("/{role_name}", response_model=RoleDef)
async def update_role(
    role_name: str,
    body: UpdateRoleRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(get_admin_user),
):
    if _is_builtin(role_name):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="内置角色不可修改",
        )

    result = await db.execute(select(CustomRole).where(CustomRole.name == role_name))
    cr = result.scalar_one_or_none()
    if not cr:
        raise HTTPException(status_code=404, detail="角色不存在")

    if body.display_name is not None:
        cr.display_name = body.display_name
    if body.description is not None:
        cr.description = body.description
    if body.role_type is not None:
        cr.role_type = body.role_type

    await db.flush()
    await db.refresh(cr)

    return RoleDef(
        name=cr.name,
        display_name=cr.display_name,
        description=cr.description,
        is_builtin=False,
        role_type=cr.role_type,
    )


@router.delete("/{role_name}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_role(
    role_name: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(get_admin_user),
):
    if _is_builtin(role_name):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="内置角色不可删除",
        )

    result = await db.execute(select(CustomRole).where(CustomRole.name == role_name))
    cr = result.scalar_one_or_none()
    if not cr:
        raise HTTPException(status_code=404, detail="角色不存在")

    user_count = await db.execute(
        select(User).where(User.role == role_name)
    )
    if user_count.scalars().first():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="该角色下仍有用户，无法删除",
        )

    await db.execute(delete(RolePermission).where(RolePermission.role == role_name))
    await db.delete(cr)
    await db.flush()
