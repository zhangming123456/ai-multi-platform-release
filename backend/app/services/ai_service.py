from __future__ import annotations

import json
import re
from typing import AsyncGenerator, Optional

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


def _parse_model_field(raw: str) -> list[dict]:
    """解析 model 字段，兼容 JSON 数组格式和旧版逗号分隔格式。

    JSON 格式: [{"id": "gpt-4o", "types": ["text","vision"], "contextInput": 128000, "contextOutput": 4096}, ...]
    旧版格式: "gpt-4o,deepseek-chat" → [{"id": "gpt-4o", "types": ["text"], ...}]
    """
    if not raw or not raw.strip():
        return []
    raw = raw.strip()
    # 尝试 JSON 解析
    if raw.startswith("["):
        try:
            data = json.loads(raw)
            if isinstance(data, list):
                result = []
                for item in data:
                    if not isinstance(item, dict) or not item.get("id"):
                        continue
                    types = item.get("types") or item.get("type") or "text"
                    if isinstance(types, str):
                        types = [types]
                    result.append({
                        "id": item["id"],
                        "types": types,
                        "contextInput": item.get("contextInput"),
                        "contextOutput": item.get("contextOutput"),
                    })
                return result
        except json.JSONDecodeError:
            pass
    # 回退到逗号分隔
    return [
        {"id": m.strip(), "types": ["text"], "contextInput": None, "contextOutput": None}
        for m in raw.split(",")
        if m.strip()
    ]


async def _resolve_model_config(
    plan_id: Optional[str],
    db: AsyncSession,
    model_id: Optional[str] = None,
) -> tuple[str, str, str, str]:
    """解析模型配置，返回 (api_key, base_url, model, plan_name)。失败时抛出 AIGenerationError。

    model 字段存储 JSON 数组 [{"id": "...", "type": "..."}]。
    若指定 model_id 且在列表中，则使用该模型；否则使用第一个。
    """
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
            model_entries = _parse_model_field(plan.model)
            model_ids = [e["id"] for e in model_entries]
            if model_id and model_id in model_ids:
                model = model_id
            elif model_ids:
                model = model_ids[0]
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

    return api_key, base_url, model, plan_name


async def generate_content_variants(
    topic: str,
    platform: str,
    db: AsyncSession,
    style: Optional[str] = None,
    keywords: Optional[list[str]] = None,
    count: int = 3,
    plan_id: Optional[str] = None,
    model_id: Optional[str] = None,
) -> list[AIVariant]:
    api_key, base_url, model, plan_name = await _resolve_model_config(plan_id, db, model_id)

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


def _parse_streamed_content(content: str) -> AIVariant:
    """解析流式生成的文本内容，提取标题、正文和标签。"""
    title = ""
    body = ""
    hashtags_str = ""

    title_match = re.search(r"【标题】\s*\n?(.*?)(?=【正文】|$)", content, re.DOTALL)
    body_match = re.search(r"【正文】\s*\n?(.*?)(?=【标签】|$)", content, re.DOTALL)
    tags_match = re.search(r"【标签】\s*\n?(.*?)$", content, re.DOTALL)

    if title_match:
        title = title_match.group(1).strip()
    if body_match:
        body = body_match.group(1).strip()
    if tags_match:
        hashtags_str = tags_match.group(1).strip()

    if not title and not body:
        # Fallback: try JSON parsing
        content_clean = content.strip()
        if content_clean.startswith("```"):
            content_clean = content_clean.split("```")[1]
            if content_clean.startswith("json"):
                content_clean = content_clean[4:]
            content_clean = content_clean.strip()
        try:
            data = json.loads(content_clean)
            item = data[0] if isinstance(data, list) and data else data if isinstance(data, dict) else {}
            return AIVariant(
                title=item.get("title", ""),
                body=item.get("body", ""),
                hashtags=item.get("hashtags", []),
            )
        except (json.JSONDecodeError, IndexError, TypeError):
            return AIVariant(title="(未解析到标题)", body=content.strip(), hashtags=[])

    hashtags = [t.lstrip("#").strip() for t in re.split(r"[\s,，]+", hashtags_str) if t.strip()]
    return AIVariant(title=title, body=body, hashtags=hashtags)


async def generate_content_stream(
    topic: str,
    platform: str,
    db: AsyncSession,
    style: Optional[str] = None,
    keywords: Optional[list[str]] = None,
    plan_id: Optional[str] = None,
    model_id: Optional[str] = None,
    files: Optional[list] = None,
) -> AsyncGenerator[dict, None]:
    """流式生成内容，yield 事件字典。事件类型: log / chunk / done / error。"""
    try:
        api_key, base_url, model, plan_name = await _resolve_model_config(plan_id, db, model_id)
    except AIGenerationError as e:
        yield {"event": "error", "message": e.message, "available_plans": e.available_plans}
        return

    yield {"event": "log", "level": "info", "message": f"使用模型配置 {plan_name}（{model}）"}

    client = AsyncOpenAI(api_key=api_key, base_url=base_url)

    platform_style = PLATFORM_STYLE_MAP.get(platform, "通用社交媒体风格")
    if style:
        platform_style += f"，额外要求：{style}"

    keywords_str = f"，关键词：{', '.join(keywords)}" if keywords else ""

    file_hint = ""
    if files:
        file_hint = "\n\n请结合上传的图片/视频内容进行分析创作，将素材中的关键信息融入文案。"

    prompt = f"""请为以下主题生成一篇适合{platform}发布的内容。
风格要求：{platform_style}{keywords_str}

主题：{topic}{file_hint}

请严格按以下格式输出（不要添加其他内容）：

【标题】
标题内容

【正文】
正文内容

【标签】
#标签1 #标签2 #标签3
"""

    if files:
        img_count = sum(1 for f in files if f.mime_type.startswith("image/"))
        vid_count = sum(1 for f in files if f.mime_type.startswith("video/"))
        parts = []
        if img_count:
            parts.append(f"{img_count} 张图片")
        if vid_count:
            parts.append(f"{vid_count} 个视频")
        yield {"event": "log", "level": "info", "message": f"附带 {'、'.join(parts)}（多模态分析）"}

    user_content: list[dict] = [{"type": "text", "text": prompt}]
    for f in files or []:
        if f.mime_type.startswith("video/"):
            user_content.append(
                {
                    "type": "video_url",
                    "video_url": {"url": f"data:{f.mime_type};base64,{f.data}"},
                }
            )
        else:
            user_content.append(
                {
                    "type": "image_url",
                    "image_url": {"url": f"data:{f.mime_type};base64,{f.data}"},
                }
            )

    try:
        create_kwargs: dict = {
            "model": model,
            "messages": [
                {
                    "role": "system",
                    "content": "你是一个专业的社交媒体内容创作助手，擅长为不同平台生成适配的优质内容。",
                },
                {"role": "user", "content": user_content if files else prompt},
            ],
            "temperature": 0.8,
            "max_tokens": 4000,
            "stream": True,
        }
        stream = await client.chat.completions.create(**create_kwargs)
    except Exception as e:
        available = await _get_available_plans(db)
        yield {
            "event": "error",
            "message": f"模型「{plan_name}」调用失败：{str(e)}。请尝试切换到其他模型配置后重试。",
            "available_plans": available,
        }
        return

    full_content = ""
    async for chunk in stream:
        if chunk.choices and chunk.choices[0].delta.content:
            delta = chunk.choices[0].delta.content
            full_content += delta
            yield {"event": "chunk", "text": delta}

    variant = _parse_streamed_content(full_content)
    yield {"event": "done", "variant": variant, "model": model, "plan_name": plan_name}
