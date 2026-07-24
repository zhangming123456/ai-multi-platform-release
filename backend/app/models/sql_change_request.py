from __future__ import annotations

import enum
from datetime import datetime
from typing import Optional

from sqlalchemy import DateTime, ForeignKey, Integer, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class SqlChangeType(str, enum.Enum):
    update = "update"
    delete = "delete"
    row_delete = "row_delete"


class SqlChangeStatus(str, enum.Enum):
    pending = "pending"
    approved = "approved"
    rejected = "rejected"
    executed = "executed"
    execute_failed = "execute_failed"


class SqlChangeRequest(Base):
    __tablename__ = "sql_change_requests"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    requester_id: Mapped[int] = mapped_column(ForeignKey("users.id"), nullable=False)
    change_type: Mapped[str] = mapped_column(String(20), nullable=False)
    sql_text: Mapped[str] = mapped_column(Text, nullable=False)
    description: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    status: Mapped[str] = mapped_column(
        String(20), default=SqlChangeStatus.pending.value, nullable=False
    )
    approvals: Mapped[int] = mapped_column(Integer, default=0, nullable=False)
    required_approvals: Mapped[int] = mapped_column(Integer, default=2, nullable=False)
    approved_by: Mapped[str] = mapped_column(String(500), default="", nullable=False)
    reject_reason: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    execute_message: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, onupdate=datetime.now, nullable=False
    )
