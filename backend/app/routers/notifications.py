from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user
from app.database import get_db
from app.models.notification import Notification
from app.models.user import User
from app.schemas.notification import NotificationResponse, NotificationListResponse

router = APIRouter(prefix="/api/notifications", tags=["通知管理"])


@router.get("/", response_model=NotificationListResponse)
async def list_notifications(
    page: int = 1,
    page_size: int = 20,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """获取当前用户的通知列表"""
    offset = (page - 1) * page_size
    
    # 查询总数
    total_result = await db.execute(
        select(Notification).where(Notification.user_id == current_user.id)
    )
    total = len(total_result.scalars().all())
    
    # 查询分页数据
    result = await db.execute(
        select(Notification)
        .where(Notification.user_id == current_user.id)
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
                type=n.type.value,
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
            Notification.is_read == False,
        )
    )
    count = len(result.scalars().all())
    return {"count": count}


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
            Notification.is_read == False,
        )
        .values(is_read=True)
    )
    await db.commit()
    
    return {"message": "已全部标记为已读"}
