from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import Boolean, Column, DateTime, ForeignKey, String, Text

from app.database import Base


def _infer_resource_type(key: str) -> str:
    parts = key.split(":")
    if len(parts) == 2 and parts[1] in ("read", "write"):
        return "page"
    return "action"


class RBACResource(Base):
    __tablename__ = "rbac_resources"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    key = Column(String(100), unique=True, nullable=False)
    name = Column(String(100), nullable=False)
    description = Column(Text, nullable=True)
    parent_id = Column(String(36), ForeignKey("rbac_resources.id"), nullable=True)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)

    @property
    def type(self) -> str:
        return _infer_resource_type(self.key)
