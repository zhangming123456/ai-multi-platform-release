from __future__ import annotations

from datetime import datetime
from typing import Literal, Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource
from app.models.rbac_role import RBACRole
from app.models.rbac_role_hierarchy import RBACRoleHierarchy
from app.models.rbac_role_permission import RBACRolePermission
from app.models.user import User
from app.services.rbac_constraint_service import validate_role_hierarchy
from app.services.rbac_service import get_role_ancestors, get_user_effective_permissions

router = APIRouter(prefix="/api/v2", tags=["RBAC 角色管理"])


class RoleRef(BaseModel):
    id: str
    name: str
    display_name: str

    model_config = {"from_attributes": True}


class RoleListItem(BaseModel):
    id: str
    name: str
    display_name: str
    description: Optional[str] = None
    role_type: str
    is_super_admin: bool
    is_builtin: bool
    created_at: datetime
    updated_at: datetime
    parent_roles: list[RoleRef] = Field(default_factory=list)
    child_roles: list[RoleRef] = Field(default_factory=list)

    model_config = {"from_attributes": True}


class RoleDetailResponse(RoleListItem):
    pass


class CreateRoleRequest(BaseModel):
    name: str = Field(..., min_length=1, max_length=50)
    display_name: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = None
    role_type: Literal["admin", "other"] = "other"
    parent_role_ids: list[str] = Field(default_factory=list)


class UpdateRoleRequest(BaseModel):
    display_name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = None
    role_type: Optional[Literal["admin", "other"]] = None


class RolePermissionItem(BaseModel):
    id: str
    key: str
    operation: str
    resource_id: str
    resource_key: str
    resource_name: str
    grant_type: Literal["direct", "inherited"]

    model_config = {"from_attributes": True}


class UpdateRolePermissionsRequest(BaseModel):
    permission_ids: list[str]


async def _role_ref(role: RBACRole) -> RoleRef:
    return RoleRef(
        id=role.id,
        name=role.name,
        display_name=role.display_name,
    )


async def _build_role_item(
    role: RBACRole,
    db: AsyncSession,
    include_relations: bool = True,
) -> RoleListItem:
    parent_roles: list[RoleRef] = []
    child_roles: list[RoleRef] = []

    if include_relations:
        parents_result = await db.execute(
            select(RBACRole)
            .join(
                RBACRoleHierarchy,
                RBACRole.id == RBACRoleHierarchy.parent_role_id,
            )
            .where(RBACRoleHierarchy.child_role_id == role.id)
        )
        parent_roles = [await _role_ref(r) for r in parents_result.scalars().all()]

        children_result = await db.execute(
            select(RBACRole)
            .join(
                RBACRoleHierarchy,
                RBACRole.id == RBACRoleHierarchy.child_role_id,
            )
            .where(RBACRoleHierarchy.parent_role_id == role.id)
        )
        child_roles = [await _role_ref(r) for r in children_result.scalars().all()]

    return RoleListItem(
        id=role.id,
        name=role.name,
        display_name=role.display_name,
        description=role.description,
        role_type=role.role_type,
        is_super_admin=role.is_super_admin,
        is_builtin=role.is_builtin,
        created_at=role.created_at,
        updated_at=role.updated_at,
        parent_roles=parent_roles,
        child_roles=child_roles,
    )


@router.get(
    "/roles",
    response_model=list[RoleListItem],
    dependencies=[Depends(require_permission("roles:read", "read"))],
)
async def list_roles(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACRole).order_by(RBACRole.created_at.asc()))
    roles = result.scalars().all()
    return [await _build_role_item(role, db) for role in roles]


@router.post(
    "/roles",
    response_model=RoleDetailResponse,
    status_code=status.HTTP_201_CREATED,
    dependencies=[Depends(require_permission("roles:manage:write", "write"))],
)
async def create_role(
    body: CreateRoleRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    existing = await db.execute(select(RBACRole).where(RBACRole.name == body.name))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="角色标识已存在",
        )

    for parent_id in body.parent_role_ids:
        parent = await db.execute(select(RBACRole).where(RBACRole.id == parent_id))
        if parent.scalar_one_or_none() is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"父角色 {parent_id} 不存在",
            )

    role = RBACRole(
        name=body.name,
        display_name=body.display_name,
        description=body.description,
        role_type=body.role_type,
        is_super_admin=False,
        is_builtin=False,
        created_at=datetime.utcnow(),
        updated_at=datetime.utcnow(),
    )
    db.add(role)
    await db.flush()
    await db.refresh(role)

    for parent_id in body.parent_role_ids:
        db.add(RBACRoleHierarchy(
            parent_role_id=parent_id,
            child_role_id=role.id,
            created_at=datetime.utcnow(),
        ))

    await db.commit()
    await db.refresh(role)
    return await _build_role_item(role, db)


@router.get(
    "/roles/{role_id}",
    response_model=RoleDetailResponse,
    dependencies=[Depends(require_permission("roles:read", "read"))],
)
async def get_role(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if role is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )
    return await _build_role_item(role, db)


@router.put(
    "/roles/{role_id}",
    response_model=RoleDetailResponse,
    dependencies=[Depends(require_permission("roles:manage:write", "write"))],
)
async def update_role(
    role_id: str,
    body: UpdateRoleRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if role is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    if role.is_builtin and body.role_type is not None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="内置角色不可修改类型",
        )

    update_data = body.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        setattr(role, field, value)
    role.updated_at = datetime.utcnow()

    await db.flush()
    await db.commit()
    await db.refresh(role)
    return await _build_role_item(role, db)


@router.delete(
    "/roles/{role_id}",
    status_code=status.HTTP_204_NO_CONTENT,
    dependencies=[Depends(require_permission("roles:manage:write", "write"))],
)
async def delete_role(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if role is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    if role.is_builtin:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="内置角色不可删除",
        )

    await db.execute(delete(RBACRoleHierarchy).where(
        (RBACRoleHierarchy.parent_role_id == role_id) |
        (RBACRoleHierarchy.child_role_id == role_id)
    ))
    await db.execute(delete(RBACRolePermission).where(RBACRolePermission.role_id == role_id))
    await db.execute(delete(RBACRole).where(RBACRole.id == role_id))
    await db.commit()
    return None


@router.post(
    "/roles/{role_id}/parents",
    response_model=RoleDetailResponse,
    dependencies=[Depends(require_permission("roles:manage:write", "write"))],
)
async def add_parent_role(
    role_id: str,
    body: RoleRef,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if role is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    parent_id = body.id
    if role_id == parent_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="角色不能设置自己为父角色",
        )

    parent = await db.execute(select(RBACRole).where(RBACRole.id == parent_id))
    if parent.scalar_one_or_none() is None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="父角色不存在",
        )

    if not await validate_role_hierarchy(parent_id, role_id, db):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="添加父角色会形成循环继承",
        )

    existing = await db.execute(
        select(RBACRoleHierarchy).where(
            RBACRoleHierarchy.parent_role_id == parent_id,
            RBACRoleHierarchy.child_role_id == role_id,
        )
    )
    if existing.scalar_one_or_none() is None:
        db.add(RBACRoleHierarchy(
            parent_role_id=parent_id,
            child_role_id=role_id,
            created_at=datetime.utcnow(),
        ))
        await db.commit()

    await db.refresh(role)
    return await _build_role_item(role, db)


@router.delete(
    "/roles/{role_id}/parents/{parent_id}",
    response_model=RoleDetailResponse,
    dependencies=[Depends(require_permission("roles:manage:write", "write"))],
)
async def remove_parent_role(
    role_id: str,
    parent_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role = result.scalar_one_or_none()
    if role is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    await db.execute(
        delete(RBACRoleHierarchy).where(
            RBACRoleHierarchy.parent_role_id == parent_id,
            RBACRoleHierarchy.child_role_id == role_id,
        )
    )
    await db.commit()
    await db.refresh(role)
    return await _build_role_item(role, db)


async def _role_permission_item(
    role_permission: RBACRolePermission,
    permission: RBACPermission,
    resource: RBACResource,
    grant_type: str,
) -> RolePermissionItem:
    return RolePermissionItem(
        id=role_permission.id,
        key=permission.key,
        operation=permission.operation,
        resource_id=resource.id,
        resource_key=resource.key,
        resource_name=resource.name,
        grant_type=grant_type,  # type: ignore[arg-type]
    )


@router.get(
    "/roles/{role_id}/permissions",
    response_model=dict[str, str],
)
async def get_role_permissions(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    role = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if role.scalar_one_or_none() is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    ancestors = await get_role_ancestors(role_id, db)
    ancestor_ids = {ancestor.id for ancestor in ancestors}
    all_role_ids = {role_id} | ancestor_ids

    result = await db.execute(
        select(RBACPermission.key, RBACResource.name)
        .join(RBACRolePermission, RBACRolePermission.permission_id == RBACPermission.id)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(
            RBACRolePermission.role_id.in_(all_role_ids),
            RBACPermission.is_active.is_(True),
        )
    )

    effective: dict[str, str] = {}
    for key, name in result.all():
        effective[key] = name

    return effective


@router.get(
    "/roles/{role_id}/permissions/direct",
    response_model=list[RolePermissionItem],
    dependencies=[Depends(require_permission("roles:read", "read"))],
)
async def get_role_direct_permissions(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    role = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if role.scalar_one_or_none() is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    result = await db.execute(
        select(RBACRolePermission, RBACPermission, RBACResource)
        .join(RBACPermission, RBACRolePermission.permission_id == RBACPermission.id)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(
            RBACRolePermission.role_id == role_id,
            RBACPermission.is_active.is_(True),
        )
    )

    return [
        await _role_permission_item(role_permission, permission, resource, "direct")
        for role_permission, permission, resource in result.all()
    ]


@router.get(
    "/roles/{role_id}/permissions/detail",
    response_model=list[RolePermissionItem],
    dependencies=[Depends(require_permission("roles:read", "read"))],
)
async def get_role_permissions_detail(
    role_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    role = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if role.scalar_one_or_none() is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    ancestors = await get_role_ancestors(role_id, db)
    ancestor_ids = {ancestor.id for ancestor in ancestors}
    all_role_ids = {role_id} | ancestor_ids

    result = await db.execute(
        select(RBACRolePermission, RBACPermission, RBACResource)
        .join(RBACPermission, RBACRolePermission.permission_id == RBACPermission.id)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(
            RBACRolePermission.role_id.in_(all_role_ids),
            RBACPermission.is_active.is_(True),
        )
    )

    items: list[RolePermissionItem] = []
    for rp, perm, resource in result.all():
        is_inherited = rp.role_id in ancestor_ids
        items.append(await _role_permission_item(
            rp, perm, resource,
            "inherited" if is_inherited else "direct",
        ))

    return items


@router.put(
    "/roles/{role_id}/permissions",
    response_model=list[RolePermissionItem],
    dependencies=[Depends(require_permission("roles:manage:write", "write"))],
)
async def update_role_permissions(
    role_id: str,
    body: UpdateRolePermissionsRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    role = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    role_obj = role.scalar_one_or_none()
    if role_obj is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="角色不存在",
        )

    if role_obj.is_super_admin:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员角色拥有所有权限，不可修改",
        )

    requested_ids = set(body.permission_ids)
    permissions_result = await db.execute(
        select(RBACPermission).where(RBACPermission.id.in_(requested_ids))
    )
    found_permissions = permissions_result.scalars().all()
    found_ids = {permission.id for permission in found_permissions}
    missing_ids = requested_ids - found_ids
    if missing_ids:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"无效的权限 ID: {', '.join(sorted(missing_ids))}",
        )

    await db.execute(
        delete(RBACRolePermission).where(RBACRolePermission.role_id == role_id)
    )

    for permission_id in requested_ids:
        db.add(RBACRolePermission(
            role_id=role_id,
            permission_id=permission_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()
    return await get_role_direct_permissions(role_id, db, current_user)
