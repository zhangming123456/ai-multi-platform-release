from __future__ import annotations

import uuid
from typing import Optional

from sqlalchemy import Boolean, String, Text, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class PermissionOperation(str):
    create = "create"
    read = "read"
    update = "update"
    delete = "delete"
    execute = "execute"
    approve = "approve"
    reject = "reject"


class RBACPermission(Base):
    __tablename__ = "rbac_permissions"
    __table_args__ = (
        UniqueConstraint("resource_id", "operation", name="uq_rbac_permission_resource_operation"),
        UniqueConstraint("key", name="uq_rbac_permission_key"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    resource_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    operation: Mapped[str] = mapped_column(String(20), nullable=False)
    key: Mapped[str] = mapped_column(String(300), nullable=False)
    description: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True, nullable=False)
