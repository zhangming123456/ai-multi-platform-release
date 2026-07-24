from __future__ import annotations

import enum
import uuid
from datetime import datetime

from sqlalchemy import Boolean, DateTime, ForeignKey, String, Text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base
from typing import Optional


class ContentStatus(str, enum.Enum):
    draft = "draft"
    ready = "ready"
    pending_review = "pending_review"
    rejected = "rejected"
    published = "published"


class Content(Base):
    __tablename__ = "contents"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id: Mapped[str] = mapped_column(String(36), ForeignKey("users.id"), nullable=False)
    title: Mapped[str] = mapped_column(String(500), nullable=False)
    body: Mapped[str] = mapped_column(Text, nullable=False)
    platform: Mapped[str] = mapped_column(String(20), nullable=False)
    status: Mapped[ContentStatus] = mapped_column(
        String(20), default=ContentStatus.draft, nullable=False
    )
    media_urls: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    ai_generated: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)
    original_content_id: Mapped[Optional[str]] = mapped_column(
        String(36), ForeignKey("contents.id"), nullable=True
    )
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, onupdate=datetime.now, nullable=False
    )

    user = relationship("User", back_populates="contents")
    publish_tasks = relationship(
        "PublishTask", back_populates="content", cascade="all, delete-orphan"
    )
    variants = relationship(
        "Content", backref="original", remote_side=[id], foreign_keys=[original_content_id]
    )
