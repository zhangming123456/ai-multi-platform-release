from fastapi import APIRouter, Body, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models.content import Content, ContentStatus
from app.models.notification import Notification, NotificationType
from app.models.rbac_permission import RBACPermission
from app.models.rbac_role_hierarchy import RBACRoleHierarchy
from app.models.rbac_role_permission import RBACRolePermission
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment
from app.models.user import User
from app.schemas.review import ReviewResponse

router = APIRouter(prefix="/api/reviews", tags=["审核管理"])


async def _user_ids_with_permission(
    permission_key: str,
    db: AsyncSession,
) -> set[str]:
    """Return user IDs whose roles grant the requested permission (including inheritance)."""
    perm_result = await db.execute(
        select(RBACPermission.id).where(
            RBACPermission.key == permission_key,
            RBACPermission.is_active.is_(True),
        )
    )
    perm = perm_result.scalar_one_or_none()
    if perm is None:
        return set()

    rp_result = await db.execute(
        select(RBACRolePermission.role_id).where(
            RBACRolePermission.permission_id == perm.id
        )
    )
    role_ids = {row for row in rp_result.scalars().all()}
    if not role_ids:
        return set()

    # Include descendant roles so inherited permissions are honored.
    all_role_ids = set(role_ids)
    frontier = set(role_ids)
    seen = set(role_ids)
    while frontier:
        result = await db.execute(
            select(RBACRoleHierarchy.child_role_id).where(
                RBACRoleHierarchy.parent_role_id.in_(frontier)
            )
        )
        frontier = set()
        for row in result.scalars().all():
            if row not in seen:
                seen.add(row)
                frontier.add(row)
                all_role_ids.add(row)

    user_result = await db.execute(
        select(User.id)
        .join(
            RBACUserRoleAssignment,
            User.id == RBACUserRoleAssignment.user_id,
        )
        .where(RBACUserRoleAssignment.role_id.in_(all_role_ids))
        .distinct()
    )
    return {row[0] for row in user_result.all()}


@router.get("/", response_model=list[ReviewResponse])
async def list_pending_reviews(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("review:read")),
):
    """获取待审核列表"""
    result = await db.execute(
        select(Content)
        .where(Content.status == ContentStatus.pending_review)
        .order_by(Content.created_at.desc())
    )
    contents = result.scalars().all()

    return [
        ReviewResponse(
            id=c.id,
            title=c.title,
            body=c.body,
            platform=c.platform,
            status=c.status.value,
            created_at=c.created_at,
            user_id=c.user_id,
            username=c.user.username if c.user else None,
        )
        for c in contents
    ]


@router.post("/{content_id}/submit", response_model=ReviewResponse)
async def submit_for_review(
    content_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("review:submit")),
):
    """提交内容审核"""
    result = await db.execute(select(Content).where(Content.id == content_id))
    content = result.scalar_one_or_none()

    if not content:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="内容不存在",
        )

    if content.user_id != current_user.id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="只能提交自己的内容",
        )

    if content.status not in [ContentStatus.draft, ContentStatus.rejected]:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="当前状态不允许提交审核",
        )

    content.status = ContentStatus.pending_review
    await db.commit()

    # 创建通知：发给所有拥有 review:approve 权限的用户
    reviewer_ids = await _user_ids_with_permission("review:approve", db)
    reviewers_result = await db.execute(select(User).where(User.id.in_(reviewer_ids)))
    reviewers = reviewers_result.scalars().all()

    for reviewer in reviewers:
        notification = Notification(
            user_id=reviewer.id,
            type=NotificationType.review_submit,
            title="新内容待审核",
            content=f"{current_user.nickname} 提交了内容「{content.title}」待审核",
            related_id=content.id,
        )
        db.add(notification)

    await db.commit()

    return ReviewResponse(
        id=content.id,
        title=content.title,
        body=content.body,
        platform=content.platform,
        status=content.status.value,
        created_at=content.created_at,
        user_id=content.user_id,
        username=current_user.username,
    )


@router.post("/{content_id}/approve", response_model=ReviewResponse)
async def approve_content(
    content_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("review:approve", "write")),
):
    """审核通过"""

    result = await db.execute(select(Content).where(Content.id == content_id))
    content = result.scalar_one_or_none()

    if not content:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="内容不存在",
        )

    if content.status != ContentStatus.pending_review:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="内容不在待审核状态",
        )

    content.status = ContentStatus.ready
    await db.commit()

    # 创建通知：发给内容创建者
    notification = Notification(
        user_id=content.user_id,
        type=NotificationType.review_approved,
        title="内容审核通过",
        content=f"您的内容「{content.title}」已通过审核",
        related_id=content.id,
    )
    db.add(notification)
    await db.commit()

    return ReviewResponse(
        id=content.id,
        title=content.title,
        body=content.body,
        platform=content.platform,
        status=content.status.value,
        created_at=content.created_at,
        user_id=content.user_id,
        username=content.user.username if content.user else None,
    )


@router.post("/{content_id}/reject", response_model=ReviewResponse)
async def reject_content(
    content_id: str,
    reason: str = Body(..., embed=True),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("review:reject")),
):
    """审核驳回"""

    result = await db.execute(select(Content).where(Content.id == content_id))
    content = result.scalar_one_or_none()

    if not content:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="内容不存在",
        )

    if content.status != ContentStatus.pending_review:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="内容不在待审核状态",
        )

    content.status = ContentStatus.rejected
    await db.commit()

    # 创建通知：发给内容创建者
    notification = Notification(
        user_id=content.user_id,
        type=NotificationType.review_rejected,
        title="内容审核驳回",
        content=f"您的内容「{content.title}」已被驳回，原因：{reason}",
        related_id=content.id,
    )
    db.add(notification)
    await db.commit()

    return ReviewResponse(
        id=content.id,
        title=content.title,
        body=content.body,
        platform=content.platform,
        status=content.status.value,
        created_at=content.created_at,
        user_id=content.user_id,
        username=content.user.username if content.user else None,
    )
