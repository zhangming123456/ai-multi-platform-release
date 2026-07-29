from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import Boolean, Column, DateTime, ForeignKey, String, UniqueConstraint

from app.database import Base


class RBACUserPermissionOverride(Base):
    """Per-user permission override table.

    Allows administrators to grant or deny specific permissions for individual
    users, overriding the role-based permissions computed from role assignments.
    """

    __tablename__ = "rbac_user_permission_overrides"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id = Column(String(36), ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    permission_key = Column(String(150), nullable=False)
    granted = Column(Boolean, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    __table_args__ = (
        UniqueConstraint("user_id", "permission_key", name="uq_user_permission_override"),
    )
