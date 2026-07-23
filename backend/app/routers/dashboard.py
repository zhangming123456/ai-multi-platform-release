from __future__ import annotations

from datetime import datetime

from fastapi import APIRouter, Depends
from pydantic import BaseModel
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user
from app.database import get_db
from app.models.account import Account, AccountStatus
from app.models.content import Content
from app.models.publish_task import PublishTask, PublishTaskStatus
from app.models.user import User

router = APIRouter(prefix="/api/dashboard", tags=["仪表盘"])

PLATFORM_NAMES = {
    "wechat_mp": "微信公众号",
    "xiaohongshu": "小红书",
    "douyin": "抖音",
    "wechat_video": "视频号",
}


class PlatformStat(BaseModel):
    platform: str
    name: str
    accounts: int
    active: int
    articles: int


class RecentPublish(BaseModel):
    id: int
    title: str
    platform: str
    account: str
    status: str
    time: str


class DashboardStats(BaseModel):
    total_accounts: int
    today_published: int
    pending_tasks: int
    ai_generated_count: int
    platform_stats: list[PlatformStat]
    recent_publishes: list[RecentPublish]


@router.get("/stats", response_model=DashboardStats)
async def get_dashboard_stats(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    # 总账号数
    total_accounts_result = await db.execute(
        select(func.count(Account.id)).where(Account.user_id == current_user.id)
    )
    total_accounts = total_accounts_result.scalar() or 0

    # 今日发布数
    today = datetime.utcnow().date()
    today_published_result = await db.execute(
        select(func.count(PublishTask.id))
        .join(PublishTask.account)
        .where(
            Account.user_id == current_user.id,
            PublishTask.status == PublishTaskStatus.published,
            func.date(PublishTask.published_at) == today,
        )
    )
    today_published = today_published_result.scalar() or 0

    # 待处理任务
    pending_result = await db.execute(
        select(func.count(PublishTask.id))
        .join(PublishTask.account)
        .where(
            Account.user_id == current_user.id,
            PublishTask.status.in_([PublishTaskStatus.pending, PublishTaskStatus.publishing]),
        )
    )
    pending_tasks = pending_result.scalar() or 0

    # AI 生成内容数
    ai_count_result = await db.execute(
        select(func.count(Content.id))
        .where(Content.user_id == current_user.id, Content.ai_generated == True)
    )
    ai_generated_count = ai_count_result.scalar() or 0

    # 各平台统计
    platform_stats = []
    for platform_key, platform_name in PLATFORM_NAMES.items():
        acct_count_result = await db.execute(
            select(func.count(Account.id)).where(
                Account.user_id == current_user.id, Account.platform == platform_key
            )
        )
        acct_count = acct_count_result.scalar() or 0

        active_count_result = await db.execute(
            select(func.count(Account.id)).where(
                Account.user_id == current_user.id,
                Account.platform == platform_key,
                Account.status == AccountStatus.active,
            )
        )
        active_count = active_count_result.scalar() or 0

        articles_result = await db.execute(
            select(func.count(Content.id)).where(
                Content.user_id == current_user.id, Content.platform == platform_key
            )
        )
        articles = articles_result.scalar() or 0

        platform_stats.append(
            PlatformStat(
                platform=platform_key,
                name=platform_name,
                accounts=acct_count,
                active=active_count,
                articles=articles,
            )
        )

    # 最近发布（取最近5条）
    recent_result = await db.execute(
        select(PublishTask, Content, Account)
        .join(Content, PublishTask.content_id == Content.id)
        .join(Account, PublishTask.account_id == Account.id)
        .where(Account.user_id == current_user.id)
        .order_by(PublishTask.created_at.desc())
        .limit(5)
    )
    recent_publishes = []
    for row in recent_result:
        task, content, account = row
        recent_publishes.append(
            RecentPublish(
                id=task.id,
                title=content.title,
                platform=account.platform,
                account=account.nickname,
                status=task.status.value if hasattr(task.status, "value") else str(task.status),
                time=task.created_at.strftime("%H:%M") if task.created_at else "",
            )
        )

    return DashboardStats(
        total_accounts=total_accounts,
        today_published=today_published,
        pending_tasks=pending_tasks,
        ai_generated_count=ai_generated_count,
        platform_stats=platform_stats,
        recent_publishes=recent_publishes,
    )
