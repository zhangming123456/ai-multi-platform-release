from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import Column, DateTime, ForeignKey, String

from app.database import Base


class RBACRoleHierarchy(Base):
    __tablename__ = "rbac_role_hierarchy"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    parent_role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    child_role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
