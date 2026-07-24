from __future__ import annotations

from datetime import datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query, status
from pydantic import BaseModel
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.core.security import hash_password, validate_password_format
from app.database import get_db
from app.models import RBACRole, RBACUserRoleAssignment
from app.models.user import User, UserRole
from app.models.user_creation_request import UserCreationRequest
from app.services.rbac_constraint_service import validate_ssd
from app.services.rbac_service import get_user_effective_permissions, has_permission, is_super_admin

router = APIRouter(prefix="/api/v2/users", tags=["RBAC3 用户管理"])


class UserResponse(BaseModel):
    id: str
    username: str
    email: Optional[str] = None
    nickname: str
    role: str
    is_active: bool = True
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class CreateUserRequest(BaseModel):
    username: str
    password: str
    nickname: str
    email: Optional[str] = None
    role: str = "operator"
    role_ids: Optional[list[str]] = None


class UpdateUserRequest(BaseModel):
    nickname: Optional[str] = None
    email: Optional[str] = None
    is_active: Optional[bool] = None


class ChangePasswordRequest(BaseModel):
    new_password: str


class UserRoleAssignmentResponse(BaseModel):
    id: str
    user_id: str
    role_id: str
    role_name: str
    role_display_name: str
    granted_by: Optional[str] = None
    valid_from: Optional[datetime] = None
    valid_until: Optional[datetime] = None
    created_at: datetime


class AssignRoleRequest(BaseModel):
    role_id: str
    valid_from: Optional[datetime] = None
    valid_until: Optional[datetime] = None


class UserPermissionsResponse(BaseModel):
    user_id: str
    permissions: dict[str, dict]


@router.get("", response_model=list[UserResponse])
async def list_users_v2(
    skip: int = 0,
    limit: int = 100,
    role: Optional[str] = None,
    keyword: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("users:read")),
):
    query = select(User)
    if role:
        query = query.where(User.role == role)
    if keyword:
        query = query.where(
            (User.username.ilike(f"%{keyword}%")) |
            (User.nickname.ilike(f"%{keyword}%"))
        )
    query = query.order_by(User.created_at.desc()).offset(skip).limit(limit)
    result = await db.execute(query)
    return result.scalars().all()


@router.post("", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user_v2(
    body: CreateUserRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("users:create", "write")),
):
    can_approve_creation = await has_permission(
        current_user.id, "user_creation_review:approve", "write", db
    )

    if not can_approve_creation and body.role != UserRole.operator.value:
        raise HTTPException(status_code=403, detail="非审批人只能创建 operator 角色")

    if body.role == UserRole.admin.value:
        raise HTTPException(status_code=403, detail="禁止创建 admin 角色")

    existing = await db.execute(select(User).where(User.username == body.username))
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="用户名已存在")

    if body.email:
        existing_email = await db.execute(select(User).where(User.email == body.email))
        if existing_email.scalar_one_or_none():
            raise HTTPException(status_code=400, detail="邮箱已被使用")

    password = body.password or f"{body.username}123"

    if not can_approve_creation:
        existing_request = await db.execute(
            select(UserCreationRequest).where(
                UserCreationRequest.username == body.username,
                UserCreationRequest.status == "pending",
            )
        )
        if existing_request.scalar_one_or_none():
            raise HTTPException(status_code=400, detail="该用户名已存在待审核申请")

        request = UserCreationRequest(
            requester_id=current_user.id,
            username=body.username,
            email=body.email,
            hashed_password=hash_password(password),
            nickname=body.nickname,
            role=body.role,
            avatar_url=None,
            status="pending",
        )
        db.add(request)
        await db.commit()
        raise HTTPException(
            status_code=status.HTTP_202_ACCEPTED,
            detail="账号创建申请已提交审核，请等待管理员审批",
        )

    user = User(
        username=body.username,
        hashed_password=hash_password(password),
        nickname=body.nickname,
        email=body.email,
        role=body.role,
    )
    db.add(user)
    await db.flush()
    await db.refresh(user)

    if not body.role_ids:
        rbac_role_result = await db.execute(select(RBACRole).where(RBACRole.name == body.role))
        rbac_role = rbac_role_result.scalar_one_or_none()
        if rbac_role:
            body.role_ids = [rbac_role.id]

    if body.role_ids:
        for role_id in body.role_ids:
            role = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
            if not role.scalar_one_or_none():
                raise HTTPException(status_code=400, detail=f"角色 {role_id} 不存在")
            await validate_ssd(user.id, db, new_role_id=role_id)
            assignment = RBACUserRoleAssignment(
                user_id=user.id,
                role_id=role_id,
                granted_by=current_user.id,
            )
            db.add(assignment)

    await db.commit()
    await db.refresh(user)
    return user


@router.get("/{user_id}", response_model=UserResponse)
async def get_user_v2(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("users:read")),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")
    return user


@router.put("/{user_id}", response_model=UserResponse)
async def update_user_v2(
    user_id: str,
    body: UpdateUserRequest,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("users:update", "write")),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")

    if body.nickname is not None:
        user.nickname = body.nickname
    if body.email is not None:
        user.email = body.email
    if body.is_active is not None:
        pass

    await db.commit()
    await db.refresh(user)
    return user


@router.delete("/{user_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_user_v2(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("users:delete", "write")),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")
    if await is_super_admin(user.id, db):
        raise HTTPException(status_code=400, detail="超级管理员不可删除")

    await db.execute(delete(RBACUserRoleAssignment).where(RBACUserRoleAssignment.user_id == user_id))
    await db.delete(user)
    await db.commit()


@router.put("/{user_id}/password", status_code=status.HTTP_204_NO_CONTENT)
async def change_user_password_v2(
    user_id: str,
    body: ChangePasswordRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("users:change_password", "write")),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")

    valid, msg = validate_password_format(body.new_password)
    if not valid:
        raise HTTPException(status_code=400, detail=msg)

    user.hashed_password = hash_password(body.new_password)
    await db.commit()


@router.get("/{user_id}/roles", response_model=list[UserRoleAssignmentResponse])
async def get_user_roles_v2(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("users:read")),
):
    user = await db.execute(select(User).where(User.id == user_id))
    if not user.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="用户不存在")

    result = await db.execute(
        select(RBACUserRoleAssignment, RBACRole.name, RBACRole.display_name)
        .join(RBACRole, RBACUserRoleAssignment.role_id == RBACRole.id)
        .where(RBACUserRoleAssignment.user_id == user_id)
        .order_by(RBACUserRoleAssignment.created_at.desc())
    )
    assignments = []
    for assignment, role_name, role_display_name in result.all():
        data = {c.name: getattr(assignment, c.name) for c in RBACUserRoleAssignment.__table__.columns}
        data["role_name"] = role_name
        data["role_display_name"] = role_display_name
        assignments.append(data)
    return assignments


@router.post("/{user_id}/roles", status_code=status.HTTP_201_CREATED)
async def assign_role_to_user_v2(
    user_id: str,
    body: AssignRoleRequest,
    db: AsyncSession = Depends(get_db),
    admin: User = Depends(require_permission("users:assign_role", "write")),
):
    user = await db.execute(select(User).where(User.id == user_id))
    if not user.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="用户不存在")

    role = await db.execute(select(RBACRole).where(RBACRole.id == body.role_id))
    if not role.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")

    existing = await db.execute(
        select(RBACUserRoleAssignment).where(
            RBACUserRoleAssignment.user_id == user_id,
            RBACUserRoleAssignment.role_id == body.role_id,
        )
    )
    if existing.scalar_one_or_none():
        raise HTTPException(status_code=400, detail="该用户已拥有此角色")

    await validate_ssd(user_id, db, new_role_id=body.role_id)

    assignment = RBACUserRoleAssignment(
        user_id=user_id,
        role_id=body.role_id,
        granted_by=admin.id,
        valid_from=body.valid_from,
        valid_until=body.valid_until,
    )
    db.add(assignment)
    await db.commit()
    return {"message": "角色分配成功"}


@router.delete("/{user_id}/roles/{role_id}", status_code=status.HTTP_204_NO_CONTENT)
async def revoke_role_from_user_v2(
    user_id: str,
    role_id: str,
    db: AsyncSession = Depends(get_db),
    _admin: User = Depends(require_permission("users:revoke_role", "write")),
):
    user = await db.execute(select(User).where(User.id == user_id))
    if not user.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="用户不存在")

    role = await db.execute(select(RBACRole).where(RBACRole.id == role_id))
    if not role.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="角色不存在")

    await db.execute(
        delete(RBACUserRoleAssignment).where(
            RBACUserRoleAssignment.user_id == user_id,
            RBACUserRoleAssignment.role_id == role_id,
        )
    )
    await db.commit()


@router.get("/{user_id}/permissions", response_model=UserPermissionsResponse)
async def get_user_permissions_v2(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("users:read")),
):
    user = await db.execute(select(User).where(User.id == user_id))
    if not user.scalar_one_or_none():
        raise HTTPException(status_code=404, detail="用户不存在")

    perms = await get_user_effective_permissions(user_id, db)
    return UserPermissionsResponse(
        user_id=user_id,
        permissions={k: v.to_dict() for k, v in perms.items()},
    )
