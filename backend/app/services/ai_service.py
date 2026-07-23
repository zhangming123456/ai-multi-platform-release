from __future__ import annotations

import json
from typing import Optional

from openai import AsyncOpenAI
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.models.model_config import ModelConfig
from app.schemas.content import AIVariant


class AIGenerationError(Exception):
    """AI 生成失败异常，附带可用模型配置列表供前端提示切换。"""

    def __init__(self, message: str, available_plans: list[dict] = None):
        super().__init__(message)
        self.message = message
        self.available_plans = available_plans or []


PLATFORM_STYLE_MAP = {
    "xiaohongshu": "小红书风格：活泼、种草、使用emoji、段落短小、口语化",
    "douyin": "抖音风格：吸引眼球、节奏快、有悬念感、适合短视频文案",
    "wechat_mp": "微信公众号风格：专业、深度、结构清晰、适合长文阅读",
    "wechat_video": "视频号风格：简洁、正能量、适合中年受众、有温度",
}


async def _get_available_plans(db: AsyncSession) -> list[dict]:
    """获取所有已启用的模型配置，供错误提示使用。"""
    result = await db.execute(
        select(ModelConfig).where(ModelConfig.enabled == True).order_by(ModelConfig.created_at)
    )
    plans = result.scalars().all()
    return [
        {
            "id": p.id,
            "name": p.name,
            "display_name": p.display_name,
            "provider": p.provider,
            "model": p.model,
        }
        for p in plans
    ]


async def generate_content_variants(
    topic: str,
    platform: str,
    db: AsyncSession,
    style: Optional[str] = None,
    keywords: Optional[list[str]] = None,
    count: int = 3,
    plan_id: Optional[str] = None,
) -> list[AIVariant]:
    api_key = settings.AI_API_KEY
    base_url = settings.AI_API_BASE_URL
    model = settings.AI_MODEL
    plan_name = "默认配置"

    if plan_id:
        result = await db.execute(select(ModelConfig).where(ModelConfig.id == plan_id))
        plan = result.scalar_one_or_none()
        if plan:
            api_key = plan.api_key or api_key
            base_url = plan.base_url or base_url
            model = plan.model or model
            plan_name = plan.name or model
        else:
            available = await _get_available_plans(db)
            raise AIGenerationError(
                f"未找到指定的模型配置（plan_id={plan_id}），请切换到可用的模型配置。",
                available_plans=available,
            )

    if not api_key:
        available = await _get_available_plans(db)
        raise AIGenerationError(
            "未配置 API Key，请在模型配置中添加有效的 API 密钥后重试。",
            available_plans=available,
        )

    client = AsyncOpenAI(api_key=api_key, base_url=base_url)

    platform_style = PLATFORM_STYLE_MAP.get(platform, "通用社交媒体风格")
    if style:
        platform_style += f"，额外要求：{style}"

    keywords_str = f"，关键词：{', '.join(keywords)}" if keywords else ""

    prompt = f"""请为以下主题生成 {count} 个不同风格的内容变体，目标平台：{platform}。
风格要求：{platform_style}{keywords_str}

主题：{topic}

请以 JSON 数组格式返回，每个变体包含 title（标题）、body（正文）、hashtags（标签列表）。
直接返回 JSON，不要包含其他说明文字。"""

    try:
        response = await client.chat.completions.create(
            model=model,
            messages=[
                {
                    "role": "system",
                    "content": "你是一个专业的社交媒体内容创作助手，擅长为不同平台生成适配的优质内容。",
                },
                {"role": "user", "content": prompt},
            ],
            temperature=0.8,
            max_tokens=4000,
        )
    except Exception as e:
        available = await _get_available_plans(db)
        raise AIGenerationError(
            f"模型「{plan_name}」调用失败：{str(e)}。请尝试切换到其他模型配置后重试。",
            available_plans=available,
        )

    content = response.choices[0].message.content.strip()
    if content.startswith("```"):
        content = content.split("```")[1]
        if content.startswith("json"):
            content = content[4:]
        content = content.strip()

    try:
        data = json.loads(content)
    except json.JSONDecodeError:
        available = await _get_available_plans(db)
        raise AIGenerationError(
            f"模型「{plan_name}」返回的内容格式异常，无法解析。请尝试切换到其他模型配置后重试。",
            available_plans=available,
        )

    variants = []
    for item in data:
        variants.append(
            AIVariant(
                title=item.get("title", ""),
                body=item.get("body", ""),
                hashtags=item.get("hashtags", []),
            )
        )
    return variants
