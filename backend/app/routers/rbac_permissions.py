from __future__ import annotations

from datetime import datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource, _infer_resource_type
from app.models.rbac_user_permission_override import RBACUserPermissionOverride
from app.models.user import User
from app.services.rbac_service import get_user_effective_permissions, flatten_effective_permissions

router = APIRouter(prefix="/api/v2", tags=["RBAC 权限管理"])


def _is_valid_permission_key(key: str) -> bool:
    parts = key.split(":")
    if len(parts) == 2:
        return (
            all(bool(part) for part in parts)
            and "read" not in parts[0]
            and "write" not in parts[0]
        )
    if len(parts) == 3:
        return (
            all(bool(part) for part in parts)
            and parts[2] in ("read", "write")
            and "read" not in parts[0]
            and "write" not in parts[0]
            and "read" not in parts[1]
            and "write" not in parts[1]
        )
    return False


def _is_page_permission_key(key: str) -> bool:
    parts = key.split(":")
    return len(parts) == 2 and parts[1] in ("read", "write")


class ResourceRef(BaseModel):
    id: str
    key: str
    name: str
    description: Optional[str] = None

    model_config = {"from_attributes": True}


class ResourceNode(BaseModel):
    id: str
    key: str
    name: str
    description: Optional[str] = None
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


class PermissionEnumItem(BaseModel):
    key: str
    name: str


class PermissionEnumDetail(BaseModel):
    key: str
    name: str
    description: Optional[str] = None


class CreateResourceRequest(BaseModel):
    key: str
    name: str
    description: Optional[str] = None
    parent_id: Optional[str] = None
    is_active: bool = True


class UpdateResourceRequest(BaseModel):
    key: Optional[str] = None
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = None
    is_active: Optional[bool] = None


class CreatePermissionRequest(BaseModel):
    resource_id: str
    operation: str = Field(..., min_length=1, max_length=50)
    key: Optional[str] = None
    is_active: bool = True


class UpdatePermissionRequest(BaseModel):
    key: Optional[str] = None
    operation: Optional[str] = None
    is_active: Optional[bool] = None


class UserPermissionValue(BaseModel):
    key: str
    name: str


# ---------------------------------------------------------------------------
# 当前用户权限
# ---------------------------------------------------------------------------

@router.get(
    "/me/permissions",
    response_model=dict[str, str],
)
async def get_my_permissions(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    effective = await get_user_effective_permissions(current_user.id, db)
    deny_result = await db.execute(
        select(RBACUserPermissionOverride.permission_key).where(
            RBACUserPermissionOverride.user_id == current_user.id,
            RBACUserPermissionOverride.granted.is_(False),
        )
    )
    denied_keys = {row[0] for row in deny_result.all()}
    return await flatten_effective_permissions(effective, db, denied_keys)


# ---------------------------------------------------------------------------
# 权限枚举（key:name，筛选已开启的权限）
# ---------------------------------------------------------------------------

@router.get(
    "/permission-enums",
    response_model=list[PermissionEnumDetail],
    dependencies=[Depends(require_permission("permissions:read", "read"))],
)
async def list_permission_enum(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACPermission, RBACResource)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.is_active.is_(True))
        .order_by(RBACPermission.key)
    )
    items: list[PermissionEnumDetail] = []
    for perm, resource in result.all():
        items.append(PermissionEnumDetail(
            key=perm.key,
            name=resource.name,
            description=resource.description,
        ))
    return items


# ---------------------------------------------------------------------------
# 页面相关权限（type=page，根据 key 格式筛选：{name}:read）
# ---------------------------------------------------------------------------

@router.get(
    "/permission-pages",
    response_model=list[PermissionEnumDetail],
    dependencies=[Depends(require_permission("permissions:read", "read"))],
)
async def list_page_permissions(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACPermission, RBACResource)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.is_active.is_(True))
        .order_by(RBACPermission.key)
    )
    items: list[PermissionEnumDetail] = []
    for perm, resource in result.all():
        if not _is_page_permission_key(perm.key):
            continue
        items.append(PermissionEnumDetail(
            key=perm.key,
            name=resource.name,
            description=resource.description,
        ))
    return items


# ---------------------------------------------------------------------------
# 资源 CRUD
# ---------------------------------------------------------------------------

def _resource_type(key: str) -> str:
    return _infer_resource_type(key)


@router.get(
    "/resources",
    response_model=list[ResourceNode],
    dependencies=[Depends(require_permission("permissions:read", "read"))],
)
async def list_resources(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACResource).order_by(RBACResource.created_at.asc()))
    resources = result.scalars().all()

    node_map: dict[str, ResourceNode] = {}
    for resource in resources:
        node_map[resource.id] = ResourceNode(
            id=resource.id,
            key=resource.key,
            name=resource.name,
            description=resource.description,
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


@router.post(
    "/resources",
    response_model=ResourceNode,
    status_code=status.HTTP_201_CREATED,
    dependencies=[Depends(require_permission("permissions:manage:write", "write"))],
)
async def create_resource(
    body: CreateResourceRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    if not _is_valid_permission_key(body.key):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="权限 key 格式无效：page 为 {name}:read，action 为 {name}:{operation}:{read|write}",
        )

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
        parent_id=resource.parent_id,
        is_active=resource.is_active,
        created_at=resource.created_at,
        children=[],
    )


@router.put(
    "/resources/{resource_id}",
    response_model=ResourceRef,
    dependencies=[Depends(require_permission("permissions:manage:write", "write"))],
)
async def update_resource(
    resource_id: str,
    body: UpdateResourceRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACResource).where(RBACResource.id == resource_id))
    resource = result.scalar_one_or_none()
    if resource is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="资源不存在",
        )

    update_data = body.model_dump(exclude_unset=True)

    if "key" in update_data and update_data["key"] != resource.key:
        if not _is_valid_permission_key(update_data["key"]):
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="权限 key 格式无效",
            )
        existing = await db.execute(
            select(RBACResource).where(RBACResource.key == update_data["key"], RBACResource.id != resource_id)
        )
        if existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail="资源 key 已存在",
            )

    for field, value in update_data.items():
        setattr(resource, field, value)

    await db.flush()
    await db.commit()
    await db.refresh(resource)
    return ResourceRef(
        id=resource.id,
        key=resource.key,
        name=resource.name,
        description=resource.description,
    )


@router.delete(
    "/resources/{resource_id}",
    status_code=status.HTTP_204_NO_CONTENT,
    dependencies=[Depends(require_permission("permissions:manage:write", "write"))],
)
async def delete_resource(
    resource_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACResource).where(RBACResource.id == resource_id))
    resource = result.scalar_one_or_none()
    if resource is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="资源不存在",
        )

    await db.execute(delete(RBACPermission).where(RBACPermission.resource_id == resource_id))
    await db.execute(delete(RBACResource).where(RBACResource.id == resource_id))
    await db.commit()
    return None


# ---------------------------------------------------------------------------
# 权限 CRUD
# ---------------------------------------------------------------------------

@router.get(
    "/permissions",
    response_model=list[PermissionItem],
    dependencies=[Depends(require_permission("permissions:read", "read"))],
)
async def list_permissions(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACPermission, RBACResource)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .order_by(RBACPermission.created_at.asc())
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
                description=resource.description,
            ),
        ))
    return permissions


@router.post(
    "/permissions",
    response_model=PermissionItem,
    status_code=status.HTTP_201_CREATED,
    dependencies=[Depends(require_permission("permissions:manage:write", "write"))],
)
async def create_permission(
    body: CreatePermissionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
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

    if not _is_valid_permission_key(key):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="权限 key 格式无效：page 为 {name}:read，action 为 {name}:{operation}:{read|write}",
        )

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
            description=resource.description,
        ),
    )


@router.put(
    "/permissions/{permission_id}",
    response_model=PermissionItem,
    dependencies=[Depends(require_permission("permissions:manage:write", "write"))],
)
async def update_permission(
    permission_id: str,
    body: UpdatePermissionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACPermission, RBACResource)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.id == permission_id)
    )
    row = result.first()
    if row is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="权限不存在",
        )
    permission, resource = row

    update_data = body.model_dump(exclude_unset=True)

    if "key" in update_data and update_data["key"] != permission.key:
        if not _is_valid_permission_key(update_data["key"]):
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="权限 key 格式无效",
            )
        existing = await db.execute(
            select(RBACPermission).where(RBACPermission.key == update_data["key"], RBACPermission.id != permission_id)
        )
        if existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail="权限 key 已存在",
            )

    for field, value in update_data.items():
        setattr(permission, field, value)

    await db.flush()
    await db.commit()
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
            description=resource.description,
        ),
    )


@router.delete(
    "/permissions/{permission_id}",
    status_code=status.HTTP_204_NO_CONTENT,
    dependencies=[Depends(require_permission("permissions:manage:write", "write"))],
)
async def delete_permission(
    permission_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(RBACPermission).where(RBACPermission.id == permission_id))
    permission = result.scalar_one_or_none()
    if permission is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="权限不存在",
        )

    await db.delete(permission)
    await db.commit()
    return None
