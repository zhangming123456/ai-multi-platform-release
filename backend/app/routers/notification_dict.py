from __future__ import annotations

from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query, status
from pydantic import BaseModel
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import PermAPIRoute, RequiresPermissions, get_current_user
from app.database import get_db
from app.models.notification_dict import DictCategory, NotificationDict
from app.models.user import User
from app.schemas.notification_dict import (
    NotificationDictCreate,
    NotificationDictGroupResponse,
    NotificationDictListResponse,
    NotificationDictResponse,
    NotificationDictUpdate,
)

router = APIRouter(prefix="/api/notification-dict", tags=["通知消息字典"], route_class=PermAPIRoute)


class MessageResponse(BaseModel):
    message: str


@router.get("/groups", response_model=list[NotificationDictGroupResponse])
async def list_groups(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(NotificationDict).order_by(
            NotificationDict.sort_order.asc(), NotificationDict.created_at.asc()
        )
    )
    items = result.scalars().all()

    groups: dict[str, list[NotificationDictResponse]] = {}
    for item in items:
        gk = item.group_key
        if gk not in groups:
            groups[gk] = []
        groups[gk].append(NotificationDictResponse.model_validate(item))

    return [
        NotificationDictGroupResponse(group_key=gk, items=grp_items)
        for gk, grp_items in groups.items()
    ]


@router.get("/", response_model=NotificationDictListResponse)
async def list_dict(
    category: Optional[str] = Query(default=None),
    group_key: Optional[str] = Query(default=None),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    stmt = select(NotificationDict).order_by(
        NotificationDict.sort_order.asc(), NotificationDict.created_at.asc()
    )
    count_stmt = select(NotificationDict)

    if category:
        stmt = stmt.where(NotificationDict.category == category)
        count_stmt = count_stmt.where(NotificationDict.category == category)
    if group_key:
        stmt = stmt.where(NotificationDict.group_key == group_key)
        count_stmt = count_stmt.where(NotificationDict.group_key == group_key)

    result = await db.execute(stmt)
    items = result.scalars().all()
    count_result = await db.execute(count_stmt)
    total = len(count_result.scalars().all())

    return NotificationDictListResponse(
        items=[NotificationDictResponse.model_validate(item) for item in items],
        total=total,
    )


@router.post("/", response_model=NotificationDictResponse)
@RequiresPermissions("notification:enum:write")
async def create_dict(
    body: NotificationDictCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    if body.category not in ("field", "enum"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST, detail="category 必须为 field 或 enum"
        )

    existing = await db.execute(
        select(NotificationDict).where(
            NotificationDict.group_key == body.group_key,
            NotificationDict.dict_key == body.dict_key,
        )
    )
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT, detail="该分组下已存在相同的 dict_key"
        )

    entry = NotificationDict(
        category=DictCategory(body.category),
        group_key=body.group_key,
        dict_key=body.dict_key,
        dict_value=body.dict_value,
        is_active=body.is_active,
        is_private=body.is_private,
        sort_order=body.sort_order,
    )
    db.add(entry)
    await db.flush()
    await db.commit()
    await db.refresh(entry)
    return NotificationDictResponse.model_validate(entry)


@router.put("/{entry_id}", response_model=NotificationDictResponse)
@RequiresPermissions("notification:enum:write")
async def update_dict(
    entry_id: str,
    body: NotificationDictUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(NotificationDict).where(NotificationDict.id == entry_id))
    entry = result.scalar_one_or_none()
    if not entry:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="字典条目不存在")

    update_data = body.model_dump(exclude_unset=True)
    if "category" in update_data and update_data["category"] not in ("field", "enum"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST, detail="category 必须为 field 或 enum"
        )

    for field, value in update_data.items():
        if field == "category" and value is not None:
            setattr(entry, field, DictCategory(value))
        else:
            setattr(entry, field, value)

    await db.commit()
    await db.refresh(entry)
    return NotificationDictResponse.model_validate(entry)


@router.delete("/{entry_id}", response_model=MessageResponse)
@RequiresPermissions("notification:enum:write")
async def delete_dict(
    entry_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(NotificationDict).where(NotificationDict.id == entry_id))
    entry = result.scalar_one_or_none()
    if not entry:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="字典条目不存在")

    await db.execute(delete(NotificationDict).where(NotificationDict.id == entry_id))
    await db.commit()
    return MessageResponse(message="删除成功")
