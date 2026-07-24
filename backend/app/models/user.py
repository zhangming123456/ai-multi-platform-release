from __future__ import annotations

import enum
from datetime import datetime

from sqlalchemy import DateTime, String
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base
from typing import Optional


class UserRole(str, enum.Enum):
    admin = "admin"
    operator = "operator"
    reviewer = "reviewer"


class User(Base):
    __tablename__ = "users"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    username: Mapped[str] = mapped_column(String(100), unique=True, index=True, nullable=False)
    email: Mapped[Optional[str]] = mapped_column(String(255), unique=True, index=True, nullable=True)
    hashed_password: Mapped[str] = mapped_column(String(255), nullable=False)
    nickname: Mapped[str] = mapped_column(String(100), nullable=False)
    role: Mapped[str] = mapped_column(String(50), default="operator", nullable=False)
    avatar_url: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, onupdate=datetime.now, nullable=False
    )

    accounts = relationship("Account", back_populates="user", cascade="all, delete-orphan")
    contents = relationship(
        "Content",
        back_populates="user",
        cascade="all, delete-orphan",
        foreign_keys="Content.user_id",
    )
    sql_histories = relationship(
        "SqlHistory", back_populates="user", cascade="all, delete-orphan"
    )
