from __future__ import annotations

import enum
from datetime import datetime

from sqlalchemy import DateTime, ForeignKey, String, Text, func
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base
from typing import Optional


class Platform(str, enum.Enum):
    wechat_mp = "wechat_mp"
    xiaohongshu = "xiaohongshu"
    douyin = "douyin"
    wechat_video = "wechat_video"


class AccountStatus(str, enum.Enum):
    active = "active"
    inactive = "inactive"
    error = "error"


class Account(Base):
    __tablename__ = "accounts"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    user_id: Mapped[int] = mapped_column(ForeignKey("users.id"), nullable=False)
    platform: Mapped[Platform] = mapped_column(String(20), nullable=False)
    nickname: Mapped[str] = mapped_column(String(200), nullable=False)
    avatar_url: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    status: Mapped[AccountStatus] = mapped_column(
        String(20), default=AccountStatus.active, nullable=False
    )
    cookie_data: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    access_token: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    token_expires_at: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)
    last_check_at: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)
    error_message: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, server_default=func.now(), nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, server_default=func.now(), onupdate=func.now(), nullable=False
    )

    user = relationship("User", back_populates="accounts")
    publish_tasks = relationship(
        "PublishTask", back_populates="account", cascade="all, delete-orphan"
    )
