from __future__ import annotations

from datetime import datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.core.security import hash_password
from app.database import get_db
from app.models.user import User, UserRole
from app.models.user_creation_request import UserCreationRequest, UserCreationStatus

router = APIRouter(prefix="/api/user-creation-reviews", tags=["用户创建审核"])


class UserCreationRequestResponse(BaseModel):
    id: str
    requester_id: str
    requester_name: str
    username: str
    email: Optional[str] = None
    nickname: str
    role: str
    status: str
    reviewer_id: Optional[str] = None
    reviewer_name: Optional[str] = None
    reject_reason: Optional[str] = None
    created_at: datetime

    model_config = {"from_attributes": True}


class RejectRequest(BaseModel):
    reason: str


@router.get("/", response_model=list[UserCreationRequestResponse])
async def list_user_creation_requests(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("review")),
):
    if current_user.role not in (UserRole.admin, UserRole.manager):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅管理员可查看用户创建审核",
        )

    result = await db.execute(
        select(
            UserCreationRequest.id,
            UserCreationRequest.requester_id,
            User.nickname.label("requester_name"),
            UserCreationRequest.username,
            UserCreationRequest.email,
            UserCreationRequest.nickname,
            UserCreationRequest.role,
            UserCreationRequest.status,
            UserCreationRequest.reviewer_id,
            UserCreationRequest.reject_reason,
            UserCreationRequest.created_at,
        )
        .join(User, User.id == UserCreationRequest.requester_id)
        .where(UserCreationRequest.status == UserCreationStatus.pending)
        .order_by(UserCreationRequest.created_at.desc())
    )
    rows = result.all()

    result2 = await db.execute(select(User.nickname, User.id))
    user_names = {row[1]: row[0] for row in result2.all()}

    return [
        UserCreationRequestResponse(
            id=row[0],
            requester_id=row[1],
            requester_name=row[2],
            username=row[3],
            email=row[4],
            nickname=row[5],
            role=row[6],
            status=row[7],
            reviewer_id=row[8],
            reviewer_name=user_names.get(row[8]) if row[8] else None,
            reject_reason=row[9],
            created_at=row[10],
        )
        for row in rows
    ]


@router.post("/{request_id}/approve", response_model=UserCreationRequestResponse)
async def approve_user_creation(
    request_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("review:approve", "write")),
):
    if current_user.role not in (UserRole.admin, UserRole.manager):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅管理员可审批用户创建",
        )

    result = await db.execute(
        select(UserCreationRequest).where(UserCreationRequest.id == request_id)
    )
    req = result.scalar_one_or_none()
    if not req:
        raise HTTPException(status_code=404, detail="审核请求不存在")

    if req.status != UserCreationStatus.pending:
        raise HTTPException(status_code=400, detail="该请求已被处理")

    existing = await db.execute(select(User).where(User.username == req.username))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"用户名 {req.username} 已存在",
        )

    user = User(
        username=req.username,
        email=req.email,
        hashed_password=req.hashed_password,
        nickname=req.nickname,
        role=req.role,
        avatar_url=req.avatar_url,
    )
    db.add(user)

    req.status = UserCreationStatus.approved
    req.reviewer_id = current_user.id
    await db.flush()
    await db.commit()
    await db.refresh(req)

    req_user_result = await db.execute(select(User.nickname).where(User.id == req.requester_id))
    req_user_name = req_user_result.scalar_one_or_none() or ""

    return UserCreationRequestResponse(
        id=req.id,
        requester_id=req.requester_id,
        requester_name=req_user_name,
        username=req.username,
        email=req.email,
        nickname=req.nickname,
        role=req.role,
        status=req.status,
        reviewer_id=req.reviewer_id,
        reviewer_name=current_user.nickname,
        reject_reason=req.reject_reason,
        created_at=req.created_at,
    )


@router.post("/{request_id}/reject", response_model=UserCreationRequestResponse)
async def reject_user_creation(
    request_id: str,
    body: RejectRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("review:reject", "write")),
):
    if current_user.role not in (UserRole.admin, UserRole.manager):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅管理员可驳回用户创建",
        )

    result = await db.execute(
        select(UserCreationRequest).where(UserCreationRequest.id == request_id)
    )
    req = result.scalar_one_or_none()
    if not req:
        raise HTTPException(status_code=404, detail="审核请求不存在")

    if req.status != UserCreationStatus.pending:
        raise HTTPException(status_code=400, detail="该请求已被处理")

    req.status = UserCreationStatus.rejected
    req.reviewer_id = current_user.id
    req.reject_reason = body.reason
    await db.flush()
    await db.commit()
    await db.refresh(req)

    req_user_result = await db.execute(select(User.nickname).where(User.id == req.requester_id))
    req_user_name = req_user_result.scalar_one_or_none() or ""

    return UserCreationRequestResponse(
        id=req.id,
        requester_id=req.requester_id,
        requester_name=req_user_name,
        username=req.username,
        email=req.email,
        nickname=req.nickname,
        role=req.role,
        status=req.status,
        reviewer_id=req.reviewer_id,
        reviewer_name=current_user.nickname,
        reject_reason=req.reject_reason,
        created_at=req.created_at,
    )
