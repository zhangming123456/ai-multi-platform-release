from __future__ import annotations

import json
from datetime import datetime

from pydantic import BaseModel, field_validator
from typing import Optional, List


class ContentCreate(BaseModel):
    title: str
    body: str
    platform: str
    status: str = "draft"
    media_urls: Optional[str] = None


class ContentUpdate(BaseModel):
    title: Optional[str] = None
    body: Optional[str] = None
    platform: Optional[str] = None
    status: Optional[str] = None
    media_urls: Optional[str] = None


class ContentResponse(BaseModel):
    id: int
    user_id: int
    title: str
    body: str
    platform: str
    status: str
    media_urls: Optional[str] = None
    ai_generated: bool
    original_content_id: Optional[int] = None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class AIGenerateRequest(BaseModel):
    topic: str
    platform: str
    style: Optional[str] = None
    keywords: Optional[list[str]] = None
    count: int = 3
    plan_id: Optional[str] = None


class UploadedFile(BaseModel):
    data: str
    mime_type: str


class AIGenerateStreamRequest(BaseModel):
    topic: str
    platforms: list[str]
    style: Optional[str] = None
    keywords: Optional[list[str]] = None
    plan_id: Optional[str] = None
    model_id: Optional[str] = None
    files: Optional[List[UploadedFile]] = None


class AIVariant(BaseModel):
    title: str
    body: str
    hashtags: list[str]


class AIGenerateResponse(BaseModel):
    variants: list[AIVariant]


class AIGenerationRecordResponse(BaseModel):
    id: int
    user_id: int
    topic: str
    platform: str
    plan_id: Optional[str] = None
    model: Optional[str] = None
    title: str
    body: str
    hashtags: list[str] = []
    created_at: datetime

    model_config = {"from_attributes": True}

    @field_validator("hashtags", mode="before")
    @classmethod
    def parse_hashtags(cls, v):
        if isinstance(v, str):
            try:
                return json.loads(v)
            except json.JSONDecodeError:
                return []
        return v or []
