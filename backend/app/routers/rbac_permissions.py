from __future__ import annotations

from datetime import datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_db
from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource

router = APIRouter(prefix="/api/v2", tags=["RBAC 权限管理"])


class ResourceRef(BaseModel):
    id: str
    key: str
    name: str
    type: str

    model_config = {"from_attributes": True}


class ResourceNode(BaseModel):
    id: str
    key: str
    name: str
    description: Optional[str] = None
    type: str
    parent_id: Optional[str] = None
    is_active: bool
    created_at: datetime
    children: list["ResourceNode"] = Field(default_factory=list)

    model_config = {"from_attributes": True}


class PermissionItem(BaseModel):
    id: str
    key: str
    operation: str
    is_active: bool
    created_at: datetime
    resource: ResourceRef

    model_config = {"from_attributes": True}


class CreateResourceRequest(BaseModel):
    key: str
    name: str
    description: Optional[str] = None
    type: str
    parent_id: Optional[str] = None
    is_active: bool = True


class CreatePermissionRequest(BaseModel):
    resource_id: str
    operation: str
    key: Optional[str] = None
    is_active: bool = True


@router.get("/resources", response_model=list[ResourceNode])
async def list_resources(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(RBACResource))
    resources = result.scalars().all()

    node_map: dict[str, ResourceNode] = {}
    for resource in resources:
        node_map[resource.id] = ResourceNode(
            id=resource.id,
            key=resource.key,
            name=resource.name,
            description=resource.description,
            type=resource.type,
            parent_id=resource.parent_id,
            is_active=resource.is_active,
            created_at=resource.created_at,
            children=[],
        )

    roots: list[ResourceNode] = []
    for resource in resources:
        node = node_map[resource.id]
        if resource.parent_id is None:
            roots.append(node)
        elif resource.parent_id in node_map:
            node_map[resource.parent_id].children.append(node)

    return roots


@router.get("/permissions", response_model=list[PermissionItem])
async def list_permissions(db: AsyncSession = Depends(get_db)):
    result = await db.execute(
        select(RBACPermission, RBACResource)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
    )
    permissions: list[PermissionItem] = []
    for permission, resource in result.all():
        permissions.append(PermissionItem(
            id=permission.id,
            key=permission.key,
            operation=permission.operation,
            is_active=permission.is_active,
            created_at=permission.created_at,
            resource=ResourceRef(
                id=resource.id,
                key=resource.key,
                name=resource.name,
                type=resource.type,
            ),
        ))
    return permissions


@router.post("/resources", response_model=ResourceNode, status_code=status.HTTP_201_CREATED)
async def create_resource(body: CreateResourceRequest, db: AsyncSession = Depends(get_db)):
    existing = await db.execute(select(RBACResource).where(RBACResource.key == body.key))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="资源 key 已存在",
        )

    if body.parent_id is not None:
        parent = await db.execute(select(RBACResource).where(RBACResource.id == body.parent_id))
        if parent.scalar_one_or_none() is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="父资源不存在",
            )

    resource = RBACResource(
        key=body.key,
        name=body.name,
        description=body.description,
        type=body.type,
        parent_id=body.parent_id,
        is_active=body.is_active,
    )
    db.add(resource)
    await db.flush()
    await db.refresh(resource)

    return ResourceNode(
        id=resource.id,
        key=resource.key,
        name=resource.name,
        description=resource.description,
        type=resource.type,
        parent_id=resource.parent_id,
        is_active=resource.is_active,
        created_at=resource.created_at,
        children=[],
    )


@router.post("/permissions", response_model=PermissionItem, status_code=status.HTTP_201_CREATED)
async def create_permission(body: CreatePermissionRequest, db: AsyncSession = Depends(get_db)):
    resource_result = await db.execute(
        select(RBACResource).where(RBACResource.id == body.resource_id)
    )
    resource = resource_result.scalar_one_or_none()
    if resource is None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="资源不存在",
        )

    key = body.key if body.key is not None else f"{resource.key}:{body.operation}"

    existing = await db.execute(select(RBACPermission).where(RBACPermission.key == key))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="权限 key 已存在",
        )

    permission = RBACPermission(
        resource_id=body.resource_id,
        operation=body.operation,
        key=key,
        is_active=body.is_active,
    )
    db.add(permission)
    await db.flush()
    await db.refresh(permission)

    return PermissionItem(
        id=permission.id,
        key=permission.key,
        operation=permission.operation,
        is_active=permission.is_active,
        created_at=permission.created_at,
        resource=ResourceRef(
            id=resource.id,
            key=resource.key,
            name=resource.name,
            type=resource.type,
        ),
    )
