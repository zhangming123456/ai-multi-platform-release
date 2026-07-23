from __future__ import annotations

from datetime import datetime

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.publish_task import PublishTask, PublishTaskStatus
from app.schemas.publish import PublishStatusResponse, PublishTaskCreate
from typing import Optional


async def create_publish_task(db: AsyncSession, task_data: PublishTaskCreate) -> PublishTask:
    task = PublishTask(
        content_id=task_data.content_id,
        account_id=task_data.account_id,
        scheduled_at=task_data.scheduled_at,
        status=PublishTaskStatus.pending,
    )
    db.add(task)
    await db.flush()
    await db.refresh(task)
    return task


async def update_task_status(
    db: AsyncSession,
    task_id: int,
    status: PublishTaskStatus,
    error_message: Optional[str] = None,
) -> Optional[PublishTask]:
    result = await db.execute(select(PublishTask).where(PublishTask.id == task_id))
    task = result.scalar_one_or_none()
    if task is None:
        return None
    task.status = status
    if error_message:
        task.error_message = error_message
    if status == PublishTaskStatus.published:
        task.published_at = datetime.utcnow()
    await db.flush()
    await db.refresh(task)
    return task


async def retry_task(db: AsyncSession, task_id: int) -> Optional[PublishTask]:
    result = await db.execute(select(PublishTask).where(PublishTask.id == task_id))
    task = result.scalar_one_or_none()
    if task is None:
        return None
    if task.status != PublishTaskStatus.failed:
        return None
    task.status = PublishTaskStatus.pending
    task.retry_count += 1
    task.error_message = None
    await db.flush()
    await db.refresh(task)
    return task


async def get_publish_stats(db: AsyncSession) -> PublishStatusResponse:
    total_result = await db.execute(select(func.count(PublishTask.id)))
    total = total_result.scalar() or 0

    pending_result = await db.execute(
        select(func.count(PublishTask.id)).where(PublishTask.status == PublishTaskStatus.pending)
    )
    pending = pending_result.scalar() or 0

    publishing_result = await db.execute(
        select(func.count(PublishTask.id)).where(PublishTask.status == PublishTaskStatus.publishing)
    )
    publishing = publishing_result.scalar() or 0

    published_result = await db.execute(
        select(func.count(PublishTask.id)).where(PublishTask.status == PublishTaskStatus.published)
    )
    published = published_result.scalar() or 0

    failed_result = await db.execute(
        select(func.count(PublishTask.id)).where(PublishTask.status == PublishTaskStatus.failed)
    )
    failed = failed_result.scalar() or 0

    return PublishStatusResponse(
        total=total,
        pending=pending,
        publishing=publishing,
        published=published,
        failed=failed,
    )
