from __future__ import annotations

from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models import RBACPermission, RBACResource
from app.models.user import User

router = APIRouter(prefix="/api/v2", tags=["RBAC3 权限管理"])


class ResourceResponse(BaseModel):
    id: str
    key: str
    name: str
    type: str
    parent_id: Optional[str] = None
    description: Optional[str] = None
    is_active: bool
    sort_order: int

    model_config = {"from_attributes": True}


class CreateResourceRequest(BaseModel):
    key: str
    name: str
    type: str
    parent_id: Optional[str] = None
    description: Optional[str] = None
    sort_order: int = 0


class UpdateResourceRequest(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    is_active: Optional[bool] = None
    sort_order: Optional[int] = None


class PermissionResponse(BaseModel):
    id: str
    resource_id: str
    resource_key: str
    operation: str
    key: str
    description: Optional[str] = None
    is_active: bool

    model_config = {"from_attributes": True}


class CreatePermissionRequest(BaseModel):
    resource_id: str
    operation: str
    description: Optional[str] = None


class UpdatePermissionRequest(BaseModel):
    description: Optional[str] = None
    is_active: Optional[bool] = None


@router.get("/resources", response_model=list[ResourceResponse])
async def list_resources_v2(
    type: Optional[str] = None,
    parent_id: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    query = select(RBACResource).order_by(RBACResource.sort_order.asc(), RBACResource.key.asc())
    if type:
        query = query.where(RBACResource.type == type)
    if parent_id is not None:
        if parent_id == "null":
            query = query.where(RBACResource.parent_id.is_(None))
        else:
            query = query.where(RBACResource.parent_id == parent_id)
    result = await db.execute(query)
    return result.scalars().all()


@router.post("/resources", response_model=ResourceResponse, status_code=status.HTTP_201_CREATED)
async def create_resource_v2(
    body: CreateResourceRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("permissions:create", "write")),
):
    existing = await db.execute(select(RBACResource).where(RBACResource.key == body.key))
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="资源标识已存在")

    resource = RBACResource(
        key=body.key,
        name=body.name,
        type=body.type,
        parent_id=body.parent_id,
        description=body.description,
        sort_order=body.sort_order,
    )
    db.add(resource)
    await db.commit()
    await db.refresh(resource)
    return resource


@router.get("/resources/{resource_id}", response_model=ResourceResponse)
async def get_resource_v2(
    resource_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACResource).where(RBACResource.id == resource_id))
    resource = result.scalar_one_or_none()
    if not resource:
        raise HTTPException(status_code=404, detail="资源不存在")
    return resource


@router.put("/resources/{resource_id}", response_model=ResourceResponse)
async def update_resource_v2(
    resource_id: str,
    body: UpdateResourceRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("permissions:update", "write")),
):
    result = await db.execute(select(RBACResource).where(RBACResource.id == resource_id))
    resource = result.scalar_one_or_none()
    if not resource:
        raise HTTPException(status_code=404, detail="资源不存在")

    if body.name is not None:
        resource.name = body.name
    if body.description is not None:
        resource.description = body.description
    if body.is_active is not None:
        resource.is_active = body.is_active
    if body.sort_order is not None:
        resource.sort_order = body.sort_order

    await db.commit()
    await db.refresh(resource)
    return resource


@router.get("/permissions", response_model=list[PermissionResponse])
async def list_permissions_v2(
    resource_id: Optional[str] = None,
    operation: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    query = (
        select(RBACPermission, RBACResource.key.label("resource_key"))
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.is_active == True)
        .order_by(RBACResource.sort_order.asc(), RBACPermission.operation.asc())
    )
    if resource_id:
        query = query.where(RBACPermission.resource_id == resource_id)
    if operation:
        query = query.where(RBACPermission.operation == operation)

    result = await db.execute(query)
    permissions = []
    for perm, res_key in result.all():
        perm_dict = {c.name: getattr(perm, c.name) for c in RBACPermission.__table__.columns}
        perm_dict["resource_key"] = res_key
        permissions.append(perm_dict)
    return permissions


@router.post("/permissions", response_model=PermissionResponse, status_code=status.HTTP_201_CREATED)
async def create_permission_v2(
    body: CreatePermissionRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("permissions:create", "write")),
):
    resource = await db.execute(select(RBACResource).where(RBACResource.id == body.resource_id))
    if not resource.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="资源不存在")

    existing = await db.execute(
        select(RBACPermission).where(
            RBACPermission.resource_id == body.resource_id,
            RBACPermission.operation == body.operation,
        )
    )
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="该资源的此操作权限已存在")

    resource_obj = resource.scalar_one()
    key = f"{resource_obj.key}:{body.operation}"
    permission = RBACPermission(
        resource_id=body.resource_id,
        operation=body.operation,
        key=key,
        description=body.description,
    )
    db.add(permission)
    await db.commit()
    await db.refresh(permission)

    perm_dict = {c.name: getattr(permission, c.name) for c in RBACPermission.__table__.columns}
    perm_dict["resource_key"] = resource_obj.key
    return perm_dict


@router.get("/permissions/{permission_id}", response_model=PermissionResponse)
async def get_permission_v2(
    permission_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACPermission, RBACResource.key.label("resource_key"))
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.id == permission_id)
    )
    row = result.first()
    if not row:
        raise HTTPException(status_code=404, detail="权限不存在")
    perm, res_key = row
    perm_dict = {c.name: getattr(perm, c.name) for c in RBACPermission.__table__.columns}
    perm_dict["resource_key"] = res_key
    return perm_dict


@router.put("/permissions/{permission_id}", response_model=PermissionResponse)
async def update_permission_v2(
    permission_id: str,
    body: UpdatePermissionRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("permissions:update", "write")),
):
    result = await db.execute(
        select(RBACPermission, RBACResource.key.label("resource_key"))
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.id == permission_id)
    )
    row = result.first()
    if not row:
        raise HTTPException(status_code=404, detail="权限不存在")
    perm, res_key = row

    if body.description is not None:
        perm.description = body.description
    if body.is_active is not None:
        perm.is_active = body.is_active

    await db.commit()
    await db.refresh(perm)

    perm_dict = {c.name: getattr(perm, c.name) for c in RBACPermission.__table__.columns}
    perm_dict["resource_key"] = res_key
    return perm_dict
