from __future__ import annotations

import uuid
from typing import Optional

from sqlalchemy import Boolean, Integer, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class RBACDSDConstraint(Base):
    __tablename__ = "rbac_dsd_constraints"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    name: Mapped[str] = mapped_column(String(200), unique=True, nullable=False)
    max_roles: Mapped[int] = mapped_column(Integer, default=1, nullable=False)
    description: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True, nullable=False)


class RBACDSDConstraintRole(Base):
    __tablename__ = "rbac_dsd_constraint_roles"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    constraint_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    role_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
