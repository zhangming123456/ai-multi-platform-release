from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import Column, DateTime, ForeignKey, String

from app.database import Base


class RBACRolePermission(Base):
    __tablename__ = "rbac_role_permissions"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    permission_id = Column(String(36), ForeignKey("rbac_permissions.id"), nullable=False)
    grant_type = Column(String(20), default="direct")
    created_at = Column(DateTime, default=datetime.utcnow)
