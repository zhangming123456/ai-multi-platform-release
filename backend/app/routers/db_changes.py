from __future__ import annotations

from typing import Any, Optional

from fastapi import APIRouter, Body, Depends, HTTPException, Query, status
from pydantic import BaseModel
from sqlalchemy import func, select, text
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import async_session_factory, get_db
from app.models.notification import Notification, NotificationType
from app.models.sql_change_request import (
    SqlChangeRequest,
    SqlChangeStatus,
    SqlChangeType,
)
from app.models.sql_history import SqlHistory
from app.models.user import User, UserRole

router = APIRouter(prefix="/api/db-changes", tags=["SQL 变更审核"])

REQUIRED_APPROVALS = 2


class SubmitChangeRequest(BaseModel):
    sql: str
    change_type: str
    description: Optional[str] = None


class SqlChangeItem(BaseModel):
    id: str
    requester_id: str
    requester_name: Optional[str] = None
    change_type: str
    sql_text: str
    description: Optional[str]
    status: str
    approvals: int
    required_approvals: int
    approved_by: list[str]
    reject_reason: Optional[str]
    execute_message: Optional[str]
    created_at: str


class SqlChangeListResponse(BaseModel):
    items: list[SqlChangeItem]
    total: int


def _to_item(req: SqlChangeRequest, requester_name: Optional[str] = None) -> SqlChangeItem:
    approved_by = [x for x in req.approved_by.split(",") if x.strip()]
    return SqlChangeItem(
        id=req.id,
        requester_id=req.requester_id,
        requester_name=requester_name,
        change_type=req.change_type,
        sql_text=req.sql_text,
        description=req.description,
        status=req.status,
        approvals=req.approvals,
        required_approvals=req.required_approvals,
        approved_by=approved_by,
        reject_reason=req.reject_reason,
        execute_message=req.execute_message,
        created_at=req.created_at.isoformat(),
    )


async def _execute_change(db: AsyncSession, req: SqlChangeRequest) -> tuple[bool, str]:
    """实际执行已通过审核的 SQL 变更，返回 (是否成功, 消息)"""
    try:
        await db.execute(text(req.sql_text))
        await db.commit()
        return True, "命令执行成功"
    except Exception as e:
        await db.rollback()
        return False, f"执行失败: {str(e)}"


@router.post("/submit", response_model=SqlChangeItem)
async def submit_change(
    body: SubmitChangeRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("db_change:submit")),
):
    """提交 DELETE/UPDATE 变更审核请求"""
    sql = body.sql.strip()
    if not sql:
        raise HTTPException(status_code=400, detail="SQL 命令不能为空")

    if body.change_type not in (
        SqlChangeType.update.value,
        SqlChangeType.delete.value,
        SqlChangeType.row_delete.value,
    ):
        raise HTTPException(status_code=400, detail="不支持的变更类型")

    req = SqlChangeRequest(
        requester_id=current_user.id,
        change_type=body.change_type,
        sql_text=sql,
        description=body.description,
        status=SqlChangeStatus.pending.value,
        required_approvals=REQUIRED_APPROVALS,
    )
    db.add(req)
    await db.commit()
    await db.refresh(req)

    reviewers_result = await db.execute(
        select(User).where(User.role.in_([UserRole.admin, UserRole.reviewer]))
    )
    reviewers = reviewers_result.scalars().all()
    type_label = "删除" if "delete" in body.change_type else "修改"
    for reviewer in reviewers:
        db.add(
            Notification(
                user_id=reviewer.id,
                type=NotificationType.review_submit,
                title="SQL 变更待审核",
                content=f"{current_user.nickname} 提交了一条 SQL {type_label}操作待审核：{sql[:100]}",
                related_id=req.id,
            )
        )
    await db.commit()

    return _to_item(req, current_user.nickname)


@router.get("/", response_model=SqlChangeListResponse)
async def list_changes(
    status_filter: Optional[str] = Query(default=None, alias="status"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """获取 SQL 变更审核列表"""
    stmt = select(SqlChangeRequest).order_by(SqlChangeRequest.created_at.desc())
    if status_filter:
        stmt = stmt.where(SqlChangeRequest.status == status_filter)
    else:
        stmt = stmt.where(SqlChangeRequest.status == SqlChangeStatus.pending.value)

    result = await db.execute(stmt)
    reqs = result.scalars().all()

    items = []
    for req in reqs:
        user_result = await db.execute(
            select(User).where(User.id == req.requester_id)
        )
        requester = user_result.scalar_one_or_none()
        items.append(_to_item(req, requester.nickname if requester else None))

    return SqlChangeListResponse(items=items, total=len(items))


@router.post("/{change_id}/approve", response_model=SqlChangeItem)
async def approve_change(
    change_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("db_change:approve", "write")),
):
    """审核通过 SQL 变更（达到 2 人通过后自动执行）"""

    result = await db.execute(
        select(SqlChangeRequest).where(SqlChangeRequest.id == change_id)
    )
    req = result.scalar_one_or_none()
    if not req:
        raise HTTPException(status_code=404, detail="审核请求不存在")

    if req.status not in (SqlChangeStatus.pending.value, SqlChangeStatus.approved.value):
        raise HTTPException(status_code=400, detail="该请求已被处理")

    if req.requester_id == current_user.id:
        raise HTTPException(status_code=400, detail="不能审核自己提交的请求")

    approved_ids = [x for x in req.approved_by.split(",") if x.strip()]
    if current_user.id in approved_ids:
        raise HTTPException(status_code=400, detail="您已审核过该请求")

    approved_ids.append(current_user.id)
    req.approved_by = ",".join(str(i) for i in approved_ids)
    req.approvals = len(approved_ids)

    if req.approvals >= req.required_approvals:
        success, msg = await _execute_change(db, req)
        if success:
            req.status = SqlChangeStatus.executed.value
            req.execute_message = msg
            notify_content = f"您提交的 SQL 变更已通过审核并自动执行成功：{req.sql_text[:80]}"
        else:
            req.status = SqlChangeStatus.execute_failed.value
            req.execute_message = msg
            notify_content = f"您提交的 SQL 变更审核通过但执行失败：{msg}"

        db.add(
            SqlHistory(
                user_id=req.requester_id,
                sql_text=req.sql_text,
                is_success=success,
                message=msg,
            )
        )
        db.add(
            Notification(
                user_id=req.requester_id,
                type=NotificationType.review_approved,
                title="SQL 变更已执行" if success else "SQL 变更执行失败",
                content=notify_content,
                related_id=req.id,
            )
        )
    else:
        req.status = SqlChangeStatus.approved.value
        db.add(
            Notification(
                user_id=req.requester_id,
                type=NotificationType.review_approved,
                title="SQL 变更审核进度",
                content=f"您提交的 SQL 变更已获得 {req.approvals}/{req.required_approvals} 人审核通过",
                related_id=req.id,
            )
        )

    await db.commit()
    await db.refresh(req)

    user_result = await db.execute(select(User).where(User.id == req.requester_id))
    requester = user_result.scalar_one_or_none()
    return _to_item(req, requester.nickname if requester else None)


@router.post("/{change_id}/reject", response_model=SqlChangeItem)
async def reject_change(
    change_id: str,
    reason: str = Body(default="", embed=True),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("db_change:reject", "write")),
):
    """驳回 SQL 变更请求"""

    result = await db.execute(
        select(SqlChangeRequest).where(SqlChangeRequest.id == change_id)
    )
    req = result.scalar_one_or_none()
    if not req:
        raise HTTPException(status_code=404, detail="审核请求不存在")

    if req.status != SqlChangeStatus.pending.value:
        raise HTTPException(status_code=400, detail="该请求已被处理")

    req.status = SqlChangeStatus.rejected.value
    req.reject_reason = reason or "未填写原因"

    db.add(
        Notification(
            user_id=req.requester_id,
            type=NotificationType.review_rejected,
            title="SQL 变更被驳回",
            content=f"您提交的 SQL 变更被驳回，原因：{req.reject_reason}",
            related_id=req.id,
        )
    )
    await db.commit()
    await db.refresh(req)

    user_result = await db.execute(select(User).where(User.id == req.requester_id))
    requester = user_result.scalar_one_or_none()
    return _to_item(req, requester.nickname if requester else None)
