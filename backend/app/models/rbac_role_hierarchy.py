from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import DateTime, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class InheritanceType(str):
    implicit = "implicit"
    explicit = "explicit"


class RBACRoleHierarchy(Base):
    __tablename__ = "rbac_role_hierarchy"
    __table_args__ = (
        UniqueConstraint("parent_role_id", "child_role_id", name="uq_rbac_role_hierarchy_parent_child"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    parent_role_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    child_role_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    inheritance_type: Mapped[str] = mapped_column(String(20), default="explicit", nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, nullable=False
    )
