from __future__ import annotations

from datetime import datetime
from typing import Literal, Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, PermAPIRoute, RequiresPermissions
from app.database import get_db
from app.models.rbac_constraint import RBACConstraint, RBACConstraintRoleAssociation
from app.models.rbac_role import RBACRole
from app.models.user import User

router = APIRouter(prefix="/api/v2", tags=["RBAC 约束管理"], route_class=PermAPIRoute)


class RoleAssociationInfo(BaseModel):
    id: str
    role_id: str
    role_name: str
    role_display_name: str
    association_type: str

    model_config = {"from_attributes": True}


class ConstraintListItem(BaseModel):
    id: str
    name: str
    description: Optional[str] = None
    constraint_type: str
    config: dict
    is_active: bool
    created_at: datetime
    roles: list[RoleAssociationInfo] = Field(default_factory=list)

    model_config = {"from_attributes": True}


class ConstraintDetailResponse(ConstraintListItem):
    pass


class CreateConstraintRequest(BaseModel):
    name: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = None
    constraint_type: Literal["mutual_exclusive", "prerequisite", "cardinality"]
    config: dict = Field(default_factory=dict)
    is_active: bool = True
    role_associations: list[dict] = Field(default_factory=list)


class UpdateConstraintRequest(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = None
    config: Optional[dict] = None
    is_active: Optional[bool] = None
    role_associations: Optional[list[dict]] = None


async def _role_association_info(
    association: RBACConstraintRoleAssociation,
    role_name_map: dict[str, tuple[str, str]],
) -> RoleAssociationInfo:
    role_name, role_display_name = role_name_map.get(
        association.role_id, (association.role_id, association.role_id)
    )
    return RoleAssociationInfo(
        id=association.id,
        role_id=association.role_id,
        role_name=role_name,
        role_display_name=role_display_name,
        association_type=association.association_type,
    )


async def _build_constraint_item(
    constraint: RBACConstraint,
    db: AsyncSession,
) -> ConstraintListItem:
    associations_result = await db.execute(
        select(RBACConstraintRoleAssociation).where(
            RBACConstraintRoleAssociation.constraint_id == constraint.id
        )
    )
    associations = associations_result.scalars().all()

    role_ids = {association.role_id for association in associations}
    role_name_map: dict[str, tuple[str, str]] = {}
    if role_ids:
        roles_result = await db.execute(
            select(RBACRole.id, RBACRole.name, RBACRole.display_name).where(
                RBACRole.id.in_(role_ids)
            )
        )
        role_name_map = {
            row[0]: (row[1], row[2]) for row in roles_result.all() if row[1] is not None
        }

    roles = [
        await _role_association_info(association, role_name_map)
        for association in associations
    ]

    return ConstraintListItem(
        id=constraint.id,
        name=constraint.name,
        description=constraint.description,
        constraint_type=constraint.constraint_type,
        config=constraint.config or {},
        is_active=constraint.is_active,
        created_at=constraint.created_at,
        roles=roles,
    )


def _validate_association_types(
    constraint_type: str,
    associations: list[dict],
) -> None:
    valid_types: set[str] = set()
    if constraint_type == "mutual_exclusive":
        valid_types = {"subject"}
    elif constraint_type == "prerequisite":
        valid_types = {"subject", "prerequisite"}
    elif constraint_type == "cardinality":
        valid_types = {"subject"}

    for assoc in associations:
        assoc_type = assoc.get("association_type")
        if assoc_type not in valid_types:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"约束类型 {constraint_type} 不支持 association_type={assoc_type}，"
                f"支持的类型为: {', '.join(sorted(valid_types))}",
            )


async def _apply_role_associations(
    constraint_id: str,
    associations: list[dict],
    db: AsyncSession,
) -> None:
    await db.execute(
        delete(RBACConstraintRoleAssociation).where(
            RBACConstraintRoleAssociation.constraint_id == constraint_id
        )
    )

    seen_pairs: set[tuple[str, str]] = set()
    for assoc in associations:
        role_id = assoc.get("role_id")
        assoc_type = assoc.get("association_type")
        if not role_id or not assoc_type:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="角色关联必须提供 role_id 和 association_type",
            )

        pair = (role_id, assoc_type)
        if pair in seen_pairs:
            continue
        seen_pairs.add(pair)

        role_result = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
        if role_result.scalar_one_or_none() is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"角色 {role_id} 不存在",
            )

        db.add(
            RBACConstraintRoleAssociation(
                constraint_id=constraint_id,
                role_id=role_id,
                association_type=assoc_type,
            )
        )


@router.get(
    "/constraints",
    response_model=list[ConstraintListItem],
)
@RequiresPermissions("constraints:read")
async def list_constraints(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACConstraint).order_by(RBACConstraint.created_at.desc())
    )
    constraints = result.scalars().all()
    return [await _build_constraint_item(constraint, db) for constraint in constraints]


@router.post(
    "/constraints",
    response_model=ConstraintDetailResponse,
    status_code=status.HTTP_201_CREATED,
)
@RequiresPermissions("constraints:manage:write")
async def create_constraint(
    body: CreateConstraintRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    existing = await db.execute(
        select(RBACConstraint).where(RBACConstraint.name == body.name)
    )
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="约束名称已存在",
        )

    _validate_association_types(body.constraint_type, body.role_associations)

    constraint = RBACConstraint(
        name=body.name,
        description=body.description,
        constraint_type=body.constraint_type,
        config=body.config,
        is_active=body.is_active,
        created_at=datetime.utcnow(),
    )
    db.add(constraint)
    await db.flush()
    await db.refresh(constraint)

    await _apply_role_associations(constraint.id, body.role_associations, db)

    await db.commit()
    await db.refresh(constraint)
    return await _build_constraint_item(constraint, db)


@router.get(
    "/constraints/{constraint_id}",
    response_model=ConstraintDetailResponse,
)
@RequiresPermissions("constraints:read")
async def get_constraint(
    constraint_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACConstraint).where(RBACConstraint.id == constraint_id)
    )
    constraint = result.scalar_one_or_none()
    if constraint is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="约束不存在",
        )
    return await _build_constraint_item(constraint, db)


@router.put(
    "/constraints/{constraint_id}",
    response_model=ConstraintDetailResponse,
)
@RequiresPermissions("constraints:manage:write")
async def update_constraint(
    constraint_id: str,
    body: UpdateConstraintRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACConstraint).where(RBACConstraint.id == constraint_id)
    )
    constraint = result.scalar_one_or_none()
    if constraint is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="约束不存在",
        )

    update_data = body.model_dump(exclude_unset=True)

    if "name" in update_data:
        name_existing = await db.execute(
            select(RBACConstraint).where(
                RBACConstraint.name == update_data["name"],
                RBACConstraint.id != constraint_id,
            )
        )
        if name_existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail="约束名称已存在",
            )

    if "role_associations" in update_data:
        _validate_association_types(
            constraint.constraint_type,
            update_data["role_associations"],
        )
        await _apply_role_associations(
            constraint_id,
            update_data.pop("role_associations"),
            db,
        )

    for field, value in update_data.items():
        setattr(constraint, field, value)

    await db.flush()
    await db.commit()
    await db.refresh(constraint)
    return await _build_constraint_item(constraint, db)


@router.delete(
    "/constraints/{constraint_id}",
    status_code=status.HTTP_204_NO_CONTENT,
)
@RequiresPermissions("constraints:manage:write")
async def delete_constraint(
    constraint_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(RBACConstraint).where(RBACConstraint.id == constraint_id)
    )
    constraint = result.scalar_one_or_none()
    if constraint is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="约束不存在",
        )

    await db.execute(
        delete(RBACConstraintRoleAssociation).where(
            RBACConstraintRoleAssociation.constraint_id == constraint_id
        )
    )
    await db.execute(
        delete(RBACConstraint).where(RBACConstraint.id == constraint_id)
    )
    await db.commit()
    return None
