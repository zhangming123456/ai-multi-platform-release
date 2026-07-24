from __future__ import annotations

import uuid

from sqlalchemy import Boolean, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class RBACRolePrerequisite(Base):
    __tablename__ = "rbac_role_prerequisites"
    __table_args__ = (
        UniqueConstraint("role_id", "prerequisite_role_id", name="uq_rbac_role_prereq_role_prereq"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    role_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    prerequisite_role_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True, nullable=False)
