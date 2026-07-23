from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel
from typing import Optional


class AccountCreate(BaseModel):
    platform: str
    nickname: str
    avatar_url: Optional[str] = None
    cookie_data: Optional[str] = None
    access_token: Optional[str] = None
    token_expires_at: Optional[datetime] = None


class AccountUpdate(BaseModel):
    nickname: Optional[str] = None
    avatar_url: Optional[str] = None
    status: Optional[str] = None
    cookie_data: Optional[str] = None
    access_token: Optional[str] = None
    token_expires_at: Optional[datetime] = None


class AccountResponse(BaseModel):
    id: int
    user_id: int
    platform: str
    nickname: str
    avatar_url: Optional[str] = None
    status: str
    cookie_data: Optional[str] = None
    access_token: Optional[str] = None
    token_expires_at: Optional[datetime] = None
    last_check_at: Optional[datetime] = None
    error_message: Optional[str] = None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class AccountStatusResponse(BaseModel):
    id: int
    status: str
    last_check_at: Optional[datetime] = None
    error_message: Optional[str] = None
