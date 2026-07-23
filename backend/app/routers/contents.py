from __future__ import annotations

import json

from fastapi import APIRouter, Depends, HTTPException, Query, status
from fastapi.responses import StreamingResponse
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user
from app.database import get_db
from app.models.ai_generation import AIGenerationRecord
from app.models.content import Content
from app.models.model_config import ModelConfig
from app.models.user import User
from app.schemas.content import (
    AIGenerateRequest,
    AIGenerateResponse,
    AIGenerateStreamRequest,
    AIGenerationRecordResponse,
    ContentCreate,
    ContentResponse,
    ContentUpdate,
)
from app.services.ai_service import AIGenerationError, generate_content_stream, generate_content_variants
from typing import Optional

router = APIRouter(prefix="/api/contents", tags=["内容管理"])

PLATFORM_LABELS = {
    "wechat_mp": "公众号",
    "xiaohongshu": "小红书",
    "douyin": "抖音",
    "wechat_video": "视频号",
}


def _sse(event: str, data: dict) -> str:
    return f"event: {event}\ndata: {json.dumps(data, ensure_ascii=False)}\n\n"


@router.get("/", response_model=list[ContentResponse])
async def list_contents(
    platform: Optional[str] = None,
    status_filter: Optional[str] = Query(None, alias="status"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    query = select(Content).where(Content.user_id == current_user.id)
    if platform:
        query = query.where(Content.platform == platform)
    if status_filter:
        query = query.where(Content.status == status_filter)
    query = query.order_by(Content.created_at.desc())
    query = query.offset((page - 1) * page_size).limit(page_size)
    result = await db.execute(query)
    return result.scalars().all()


@router.post("/", response_model=ContentResponse, status_code=status.HTTP_201_CREATED)
async def create_content(
    request: ContentCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    content = Content(
        user_id=current_user.id,
        title=request.title,
        body=request.body,
        platform=request.platform,
        status=request.status,
        media_urls=request.media_urls,
    )
    db.add(content)
    await db.flush()
    await db.refresh(content)
    return content


@router.get("/ai-generations", response_model=list[AIGenerationRecordResponse])
async def list_ai_generations(
    platform: Optional[str] = None,
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    query = select(AIGenerationRecord).where(AIGenerationRecord.user_id == current_user.id)
    if platform:
        query = query.where(AIGenerationRecord.platform == platform)
    query = query.order_by(AIGenerationRecord.created_at.desc())
    query = query.offset((page - 1) * page_size).limit(page_size)
    result = await db.execute(query)
    return result.scalars().all()


@router.get("/{content_id}", response_model=ContentResponse)
async def get_content(
    content_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(Content).where(Content.id == content_id, Content.user_id == current_user.id)
    )
    content = result.scalar_one_or_none()
    if not content:
        raise HTTPException(status_code=404, detail="内容不存在")
    return content


@router.put("/{content_id}", response_model=ContentResponse)
async def update_content(
    content_id: int,
    request: ContentUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(Content).where(Content.id == content_id, Content.user_id == current_user.id)
    )
    content = result.scalar_one_or_none()
    if not content:
        raise HTTPException(status_code=404, detail="内容不存在")

    update_data = request.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        setattr(content, field, value)

    await db.flush()
    await db.refresh(content)
    return content


@router.delete("/{content_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_content(
    content_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(Content).where(Content.id == content_id, Content.user_id == current_user.id)
    )
    content = result.scalar_one_or_none()
    if not content:
        raise HTTPException(status_code=404, detail="内容不存在")
    await db.delete(content)
    await db.flush()


@router.post("/ai-generate", response_model=AIGenerateResponse)
async def ai_generate(
    request: AIGenerateRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    try:
        variants = await generate_content_variants(
            topic=request.topic,
            platform=request.platform,
            db=db,
            style=request.style,
            keywords=request.keywords,
            count=request.count,
            plan_id=request.plan_id,
        )
    except AIGenerationError as e:
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail={
                "message": e.message,
                "available_plans": e.available_plans,
            },
        )

    model_name = None
    if request.plan_id:
        plan_result = await db.execute(
            select(ModelConfig).where(ModelConfig.id == request.plan_id)
        )
        plan = plan_result.scalar_one_or_none()
        if plan:
            model_name = plan.model

    for variant in variants:
        record = AIGenerationRecord(
            user_id=current_user.id,
            topic=request.topic,
            platform=request.platform,
            plan_id=request.plan_id,
            model=model_name,
            title=variant.title,
            body=variant.body,
            hashtags=json.dumps(variant.hashtags, ensure_ascii=False),
        )
        db.add(record)
    await db.flush()

    return AIGenerateResponse(variants=variants)


@router.post("/ai-generate-stream")
async def ai_generate_stream(
    request: AIGenerateStreamRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """SSE 流式生成端点：逐平台流式输出 LLM 内容，同时推送实时日志。"""

    async def event_generator():
        for platform in request.platforms:
            label = PLATFORM_LABELS.get(platform, platform)
            yield _sse("log", {"level": "req", "message": f"开始为「{label}」生成内容…"})

            async for event in generate_content_stream(
                topic=request.topic,
                platform=platform,
                db=db,
                style=request.style,
                keywords=request.keywords,
                plan_id=request.plan_id,
                model_id=request.model_id,
                files=request.files,
            ):
                evt = event["event"]
                if evt == "chunk":
                    yield _sse("chunk", {"platform": platform, "text": event["text"]})
                elif evt == "done":
                    variant = event["variant"]
                    model_name = event.get("model")
                    record = AIGenerationRecord(
                        user_id=current_user.id,
                        topic=request.topic,
                        platform=platform,
                        plan_id=request.plan_id,
                        model=model_name,
                        title=variant.title,
                        body=variant.body,
                        hashtags=json.dumps(variant.hashtags, ensure_ascii=False),
                    )
                    db.add(record)
                    await db.flush()
                    yield _sse(
                        "done",
                        {
                            "platform": platform,
                            "variant": {
                                "title": variant.title,
                                "body": variant.body,
                                "hashtags": variant.hashtags,
                            },
                        },
                    )
                elif evt == "error":
                    yield _sse(
                        "error",
                        {
                            "platform": platform,
                            "message": event["message"],
                            "available_plans": event.get("available_plans", []),
                        },
                    )
                elif evt == "log":
                    yield _sse(
                        "log",
                        {
                            "platform": platform,
                            "level": event.get("level", "info"),
                            "message": event.get("message", ""),
                        },
                    )

        yield _sse("complete", {"message": "全部生成完成"})

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )
