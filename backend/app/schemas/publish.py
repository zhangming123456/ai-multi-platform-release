from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel
from typing import Optional


class PublishTaskCreate(BaseModel):
    content_id: int
    account_id: int
    scheduled_at: Optional[datetime] = None


class PublishTaskResponse(BaseModel):
    id: int
    content_id: int
    account_id: int
    status: str
    scheduled_at: Optional[datetime] = None
    published_at: Optional[datetime] = None
    error_message: Optional[str] = None
    retry_count: int
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class PublishStatusResponse(BaseModel):
    total: int
    pending: int
    publishing: int
    published: int
    failed: int
