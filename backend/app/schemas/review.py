from datetime import datetime
from typing import Optional

from pydantic import BaseModel


class ReviewResponse(BaseModel):
    id: int
    title: str
    body: str
    platform: str
    status: str
    created_at: datetime
    user_id: int
    username: Optional[str] = None
