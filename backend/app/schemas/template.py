from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel
from typing import Optional


class TemplateCreate(BaseModel):
    name: str
    platform: str
    thumbnail_url: Optional[str] = None
    config: Optional[str] = None


class TemplateUpdate(BaseModel):
    name: Optional[str] = None
    platform: Optional[str] = None
    thumbnail_url: Optional[str] = None
    config: Optional[str] = None


class TemplateResponse(BaseModel):
    id: str
    name: str
    platform: str
    thumbnail_url: Optional[str] = None
    config: Optional[str] = None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}
