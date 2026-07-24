from datetime import datetime
from typing import Optional

from pydantic import BaseModel


class NotificationResponse(BaseModel):
    id: int
    type: str
    title: str
    content: Optional[str] = None
    related_id: Optional[int] = None
    is_read: bool
    created_at: datetime


class NotificationListResponse(BaseModel):
    total: int
    page: int
    page_size: int
    items: list[NotificationResponse]
