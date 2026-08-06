import asyncio
import json
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.responses import StreamingResponse
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user
from app.core.security import decode_access_token
from app.database import async_session_factory, get_db
from app.models.notification import Notification
from app.models.user import User
from app.schemas.notification import NotificationListResponse, NotificationResponse
from app.services.notification_broadcaster import broadcaster

router = APIRouter(prefix="/api/notifications", tags=["通知管理"])


async def _notification_to_dict(n: Notification) -> dict:
    return {
        "id": n.id,
        "type": n.type.value if hasattr(n.type, "value") else n.type,
        "title": n.title,
        "content": n.content,
        "related_id": n.related_id,
        "is_read": n.is_read,
        "created_at": n.created_at.isoformat() if n.created_at else None,
    }


@router.get("/stream")
async def notification_stream(
    token: str,
):
    payload = decode_access_token(token)
    if payload is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="令牌无效")
    user_id: str = payload.get("sub")
    if user_id is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="令牌无效")

    sub_queue: asyncio.Queue[dict] = broadcaster.subscribe(user_id)
    known_ids: set[str] = set()

    async def event_generator():
        try:
            yield 'event: connected\ndata: {"status":"ok"}\n\n'

            while True:
                task_poll = asyncio.ensure_future(_poll_db(user_id, known_ids))
                task_broadcast = asyncio.ensure_future(sub_queue.get())

                done, _ = await asyncio.wait(
                    [task_poll, task_broadcast],
                    return_when=asyncio.FIRST_COMPLETED,
                )

                for task in [task_poll, task_broadcast]:
                    if not task.done():
                        task.cancel()

                if task_poll.done():
                    try:
                        result = task_poll.result()
                        if result:
                            yield result
                    except Exception:
                        pass

                if task_broadcast.done():
                    try:
                        item = task_broadcast.result()
                        if item and item.get("id") and item["id"] not in known_ids:
                            known_ids.add(item["id"])
                            data = json.dumps([item], ensure_ascii=False)
                            yield f"event: notification\ndata: {data}\n\n"
                    except Exception:
                        pass
        finally:
            broadcaster.unsubscribe(user_id, sub_queue)

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )


async def _poll_db(user_id: str, known_ids: set[str]) -> Optional[str]:
    async with async_session_factory() as session:
        result = await session.execute(
            select(Notification)
            .where(
                Notification.user_id == user_id,
            )
            .order_by(Notification.created_at.desc())
            .limit(20)
        )
        notifications = result.scalars().all()
        current_ids = {n.id for n in notifications}
        new_ids = current_ids - known_ids
        known_ids.update(current_ids)
        if new_ids:
            new_notifications = [n for n in notifications if n.id in new_ids]
            data = json.dumps(
                [_notification_to_dict(n) for n in new_notifications],
                ensure_ascii=False,
            )
            return f"event: notification\ndata: {data}\n\n"
    return None


@router.get("/", response_model=NotificationListResponse)
async def list_notifications(
    page: int = 1,
    page_size: int = 20,
    type: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """获取当前用户的通知列表"""
    offset = (page - 1) * page_size

    base_where = Notification.user_id == current_user.id
    if type:
        base_where = base_where & (Notification.type == type)

    total_result = await db.execute(select(Notification).where(base_where))
    total = len(total_result.scalars().all())

    result = await db.execute(
        select(Notification)
        .where(base_where)
        .order_by(Notification.created_at.desc())
        .offset(offset)
        .limit(page_size)
    )
    notifications = result.scalars().all()

    return NotificationListResponse(
        total=total,
        page=page,
        page_size=page_size,
        items=[
            NotificationResponse(
                id=n.id,
                type=n.type.value if hasattr(n.type, "value") else n.type,
                title=n.title,
                content=n.content,
                related_id=n.related_id,
                is_read=n.is_read,
                created_at=n.created_at,
            )
            for n in notifications
        ],
    )


@router.get("/unread-count", response_model=dict)
async def get_unread_count(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """获取未读通知数量"""
    result = await db.execute(
        select(Notification).where(
            Notification.user_id == current_user.id,
            ~Notification.is_read,
        )
    )
    count = len(result.scalars().all())
    return {"count": count}


@router.get("/type-counts", response_model=dict)
async def get_type_counts(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """获取各类型通知总数"""
    result = await db.execute(select(Notification).where(Notification.user_id == current_user.id))
    notifications = result.scalars().all()

    counts: dict[str, int] = {}
    for n in notifications:
        t = n.type.value if hasattr(n.type, "value") else n.type
        counts[t] = counts.get(t, 0) + 1

    return {"total": len(notifications), "counts": counts}


@router.post("/{notification_id}/read")
async def mark_as_read(
    notification_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """标记通知为已读"""
    result = await db.execute(
        select(Notification).where(
            Notification.id == notification_id,
            Notification.user_id == current_user.id,
        )
    )
    notification = result.scalar_one_or_none()

    if not notification:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="通知不存在",
        )

    notification.is_read = True
    await db.commit()

    return {"message": "已标记为已读"}


@router.post("/read-all")
async def mark_all_as_read(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """标记所有通知为已读"""
    await db.execute(
        update(Notification)
        .where(
            Notification.user_id == current_user.id,
            ~Notification.is_read,
        )
        .values(is_read=True)
    )
    await db.commit()

    return {"message": "已全部标记为已读"}
