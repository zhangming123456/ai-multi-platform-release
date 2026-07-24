from __future__ import annotations

from datetime import datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query, status
from pydantic import BaseModel
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models import (
    RBACRole,
    RBACRoleHierarchy,
    RBACRolePermission,
    RBACUserRoleAssignment,
)
from app.models.user import User
from app.services.rbac_constraint_service import validate_ssd
from app.services.rbac_service import (
    PermissionAccess,
    get_role_ancestors,
    get_role_descendants,
    get_role_direct_permission_ids,
    get_role_direct_permissions,
    get_role_effective_permissions,
    get_role_permission_detail,
    update_role_direct_permissions,
    validate_hierarchy_dag,
)

router = APIRouter(prefix="/api/v2/roles", tags=["RBAC3 角色管理"])


class RoleResponse(BaseModel):
    id: str
    name: str
    display_name: str
    description: Optional[str] = None
    role_type: str
    scope: str
    is_system: bool
    is_super_admin: bool
    is_active: bool
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class CreateRoleRequest(BaseModel):
    name: str
    display_name: str
    description: Optional[str] = None
    role_type: str = "other"
    scope: str = "system"


class UpdateRoleRequest(BaseModel):
    display_name: Optional[str] = None
    description: Optional[str] = None
    role_type: Optional[str] = None
    is_active: Optional[bool] = None


class RolePermissionResponse(BaseModel):
    permission_key: str
    grant_type: str
    read: bool
    write: bool


class RoleEffectivePermissionsResponse(BaseModel):
    role_id: str
    permissions: dict[str, dict]


class HierarchyRequest(BaseModel):
    parent_role_id: str


@router.get("", response_model=list[RoleResponse])
async def list_roles_v2(
    role_type: Optional[str] = None,
    is_active: Optional[bool] = None,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("roles:read")),
):
    query = select(RBACRole).order_by(RBACRole.created_at.asc())
    if role_type:
        query = query.where(RBACRole.role_type == role_type)
    if is_active is not None:
        query = query.where(RBACRole.is_active == is_active)
    result = await db.execute(query)
    return result.scalars().all()


@router.post("", response_model=RoleResponse, status_code=status.HTTP_201_CREATED)
async def create_role_v2(
    body: CreateRoleRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("roles:create", "write")),
):
    existing = await db.execute(select(RBACRole).where(RBACRole.name == body.name))
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="角色名称已存在")

    role = RBACRole(
        name=body.name,
        display_name=body.display_name,
        description=body.description,
        role_type=body.role_type,
        scope=body.scope,
        is_system=False,
        is_super_admin=False,
        is_active=True,
    )
    db.add(role)
    await db.commit()
    await db.refresh(role)
    return role


@router.get("/{role_id}", response_model=RoleResponse)
async def get_role_v2(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("roles:read")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if not role:
        raise HTTPException(status_code=404, detail="角色不存在")
    return role


@router.put("/{role_id}", response_model=RoleResponse)
async def update_role_v2(
    role_id: str,
    body: UpdateRoleRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("roles:update", "write")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if not role:
        raise HTTPException(status_code=404, detail="角色不存在")
    if role.is_system:
        raise HTTPException(status_code=400, detail="系统内置角色不可修改")

    if body.display_name is not None:
        role.display_name = body.display_name
    if body.description is not None:
        role.description = body.description
    if body.role_type is not None:
        role.role_type = body.role_type
    if body.is_active is not None:
        role.is_active = body.is_active

    await db.commit()
    await db.refresh(role)
    return role


@router.delete("/{role_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_role_v2(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("roles:delete", "write")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if not role:
        raise HTTPException(status_code=404, detail="角色不存在")
    if role.is_system:
        raise HTTPException(status_code=400, detail="系统内置角色不可删除")
    if role.is_super_admin:
        raise HTTPException(status_code=400, detail="超级管理员角色不可删除")

    user_count = await db.execute(
        select(RBACUserRoleAssignment).where(RBACUserRoleAssignment.role_id == role_id)
    )
    if user_count.scalars().first():
        raise HTTPException(status_code=400, detail="该角色下仍有用户，无法删除")

    await db.execute(delete(RBACRolePermission).where(RBACRolePermission.role_id == role_id))
    await db.execute(delete(RBACRoleHierarchy).where(
        (RBACRoleHierarchy.parent_role_id == role_id) |
        (RBACRoleHierarchy.child_role_id == role_id)
    ))
    await db.delete(role)
    await db.commit()


@router.get("/{role_id}/permissions", response_model=RoleEffectivePermissionsResponse)
async def get_role_permissions_v2(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("roles:read")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if not result.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")

    perms = await get_role_effective_permissions(role_id, db)
    return RoleEffectivePermissionsResponse(
        role_id=role_id,
        permissions={k: v.to_dict() for k, v in perms.items()},
    )


@router.get("/{role_id}/permissions/direct", response_model=RoleEffectivePermissionsResponse)
async def get_role_direct_permissions_v2(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("roles:read")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if not result.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")

    perms = await get_role_direct_permissions(role_id, db)
    return RoleEffectivePermissionsResponse(
        role_id=role_id,
        permissions={k: v.to_dict() for k, v in perms.items()},
    )


@router.get("/{role_id}/permissions/detail", response_model=RoleEffectivePermissionsResponse)
async def get_role_permission_detail_v2(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("roles:read")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if not result.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")

    perms = await get_role_permission_detail(role_id, db)
    return RoleEffectivePermissionsResponse(
        role_id=role_id,
        permissions=perms,
    )


@router.put("/{role_id}/permissions")
async def update_role_permissions_v2(
    role_id: str,
    permission_ids: list[str],
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("roles:update", "write")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if not role:
        raise HTTPException(status_code=404, detail="角色不存在")
    if role.is_system and role.is_super_admin:
        raise HTTPException(status_code=400, detail="超级管理员权限不可修改")

    await update_role_direct_permissions(role_id, permission_ids, db)
    await db.commit()
    return {"message": "权限更新成功"}


@router.get("/{role_id}/ancestors", response_model=list[RoleResponse])
async def get_role_ancestors_v2(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("roles:read")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if not result.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")
    return await get_role_ancestors(role_id, db)


@router.get("/{role_id}/descendants", response_model=list[RoleResponse])
async def get_role_descendants_v2(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("roles:read")),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if not result.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")
    return await get_role_descendants(role_id, db)


@router.post("/{role_id}/parents", status_code=status.HTTP_201_CREATED)
async def add_parent_role_v2(
    role_id: str,
    body: HierarchyRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("roles:update", "write")),
):
    for rid in [role_id, body.parent_role_id]:
        result = await db.execute(select(RBACRole).where(RBACRole.id == rid))
        if not result.scalar_one_or_none():
            raise HTTPException(status_code=404, detail=f"角色 {rid} 不存在")

    if not await validate_hierarchy_dag(body.parent_role_id, role_id, db):
        raise HTTPException(status_code=400, detail="会造成循环继承，禁止添加")

    existing = await db.execute(
        select(RBACRoleHierarchy).where(
            RBACRoleHierarchy.parent_role_id == body.parent_role_id,
            RBACRoleHierarchy.child_role_id == role_id,
        )
    )
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="该继承关系已存在")

    hierarchy = RBACRoleHierarchy(
        parent_role_id=body.parent_role_id,
        child_role_id=role_id,
        inheritance_type="explicit",
    )
    db.add(hierarchy)
    await db.commit()
    return {"message": "父角色添加成功"}


@router.delete("/{role_id}/parents/{parent_id}", status_code=status.HTTP_204_NO_CONTENT)
async def remove_parent_role_v2(
    role_id: str,
    parent_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("roles:update", "write")),
):
    child_role = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if not child_role.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="子角色不存在")
    parent_role = await db.execute(select(RBACRole).where(RBACRole.id == parent_id))
    if not parent_role.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="父角色不存在")

    await db.execute(
        delete(RBACRoleHierarchy).where(
            RBACRoleHierarchy.parent_role_id == parent_id,
            RBACRoleHierarchy.child_role_id == role_id,
        )
    )
    await db.commit()
