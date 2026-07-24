from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models.publish_task import PublishTask
from app.models.user import User
from app.schemas.publish import PublishStatusResponse, PublishTaskCreate, PublishTaskResponse
from app.services.publish_service import create_publish_task, get_publish_stats, retry_task
from typing import Optional

router = APIRouter(prefix="/api/publish", tags=["发布管理"])


@router.get("/tasks", response_model=list[PublishTaskResponse])
async def list_tasks(
    status_filter: Optional[str] = Query(None, alias="status"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    query = (
        select(PublishTask)
        .join(PublishTask.account)
        .where(PublishTask.account.has(user_id=current_user.id))
    )
    if status_filter:
        query = query.where(PublishTask.status == status_filter)
    query = query.order_by(PublishTask.created_at.desc())
    query = query.offset((page - 1) * page_size).limit(page_size)
    result = await db.execute(query)
    return result.scalars().all()


@router.post("/tasks", response_model=PublishTaskResponse, status_code=status.HTTP_201_CREATED)
async def create_task(
    request: PublishTaskCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("publish:create", "write")),
):
    task = await create_publish_task(db, request)
    return task


@router.get("/tasks/{task_id}", response_model=PublishTaskResponse)
async def get_task(
    task_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(PublishTask).where(PublishTask.id == task_id))
    task = result.scalar_one_or_none()
    if not task:
        raise HTTPException(status_code=404, detail="任务不存在")
    return task


@router.post("/tasks/{task_id}/retry", response_model=PublishTaskResponse)
async def retry_publish_task(
    task_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("publish:retry", "write")),
):
    task = await retry_task(db, task_id)
    if not task:
        raise HTTPException(status_code=404, detail="任务不存在或状态不允许重试")
    return task


@router.get("/stats", response_model=PublishStatusResponse)
async def publish_stats(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    return await get_publish_stats(db)
