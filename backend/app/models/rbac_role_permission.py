from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import DateTime, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class GrantType(str):
    direct = "direct"
    inherited = "inherited"


class RBACRolePermission(Base):
    __tablename__ = "rbac_role_permissions"
    __table_args__ = (
        UniqueConstraint("role_id", "permission_id", "grant_type", name="uq_rbac_role_perm_role_perm_grant"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    role_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    permission_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    grant_type: Mapped[str] = mapped_column(String(20), default="direct", nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, nullable=False
    )
