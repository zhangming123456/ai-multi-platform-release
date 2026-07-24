from __future__ import annotations

import uuid
from datetime import datetime
from typing import Optional

from sqlalchemy import DateTime, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class RBACUserRoleAssignment(Base):
    __tablename__ = "rbac_user_role_assignments"
    __table_args__ = (
        UniqueConstraint("user_id", "role_id", name="uq_rbac_user_role_user_role"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    role_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    granted_by: Mapped[Optional[str]] = mapped_column(String(36), nullable=True)
    valid_from: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)
    valid_until: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, nullable=False
    )
