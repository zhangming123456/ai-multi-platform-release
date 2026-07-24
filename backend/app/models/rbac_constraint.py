from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import Boolean, Column, DateTime, ForeignKey, JSON, String, Text

from app.database import Base


class RBACConstraint(Base):
    __tablename__ = "rbac_constraints"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    name = Column(String(100), nullable=False)
    description = Column(Text, nullable=True)
    constraint_type = Column(String(50), nullable=False)
    config = Column(JSON, default=dict)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)


class RBACConstraintRoleAssociation(Base):
    __tablename__ = "rbac_constraint_role_associations"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    constraint_id = Column(String(36), ForeignKey("rbac_constraints.id"), nullable=False)
    role_id = Column(String(36), ForeignKey("rbac_roles.id"), nullable=False)
    association_type = Column(String(50), nullable=False)
