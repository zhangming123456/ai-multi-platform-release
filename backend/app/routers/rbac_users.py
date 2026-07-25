from __future__ import annotations

from datetime import datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.core.security import hash_password, validate_password_format, verify_password
from app.database import get_db
from app.models.rbac_role import RBACRole
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment
from app.models.user import User, UserRole
from app.models.user_creation_request import UserCreationRequest, UserCreationStatus
from app.schemas.auth import UserInfo
from app.services.rbac_constraint_service import validate_user_role_assignments
from app.services.rbac_service import get_user_effective_permissions

router = APIRouter(prefix="/api/v2", tags=["RBAC 用户管理"])


class RoleAssignmentInfo(BaseModel):
    id: str
    name: str
    display_name: str

    model_config = {"from_attributes": True}


class UserListItem(UserInfo):
    roles: list[RoleAssignmentInfo] = []


class UserDetailResponse(UserListItem):
    pass


class CreateUserRequest(BaseModel):
    username: str = Field(..., min_length=1, max_length=100)
    password: str = Field(..., min_length=6, max_length=255)
    nickname: str = Field(..., min_length=1, max_length=100)
    email: Optional[str] = Field(None, max_length=255)
    avatar_url: Optional[str] = None
    role: Optional[str] = "operator"
    role_ids: list[str] = Field(default_factory=list)


class UpdateUserRequest(BaseModel):
    nickname: Optional[str] = Field(None, min_length=1, max_length=100)
    email: Optional[str] = Field(None, max_length=255)
    avatar_url: Optional[str] = None
    role_ids: Optional[list[str]] = None


class UpdateUserRolesRequest(BaseModel):
    role_ids: list[str]


class UpdateUserPasswordRequest(BaseModel):
    new_password: str = Field(..., min_length=6, max_length=255)
    old_password: Optional[str] = None


class UserPermissionItem(BaseModel):
    key: str
    read: bool
    write: bool


async def _user_roles(user_id: str, db: AsyncSession) -> list[RoleAssignmentInfo]:
    result = await db.execute(
        select(RBACRole)
        .join(
            RBACUserRoleAssignment,
            RBACRole.id == RBACUserRoleAssignment.role_id,
        )
        .where(RBACUserRoleAssignment.user_id == user_id)
    )
    return [
        RoleAssignmentInfo(id=role.id, name=role.name, display_name=role.display_name)
        for role in result.scalars().all()
    ]


async def _build_user_item(user: User, db: AsyncSession) -> UserDetailResponse:
    base = UserInfo.model_validate(user)
    roles = await _user_roles(user.id, db)
    return UserDetailResponse(**base.model_dump(), roles=roles)


def _is_admin(user: User) -> bool:
    return user.role == UserRole.admin.value or str(user.role) == UserRole.admin.value


def _is_manager_or_admin(user: User) -> bool:
    return str(user.role) in {UserRole.admin.value, UserRole.manager.value}


@router.get(
    "/users",
    response_model=list[UserDetailResponse],
    dependencies=[Depends(require_permission("users:read", "read"))],
)
async def list_users(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).order_by(User.created_at.desc()))
    users = result.scalars().all()
    return [await _build_user_item(user, db) for user in users]


@router.post(
    "/users",
    response_model=UserDetailResponse,
    status_code=status.HTTP_201_CREATED,
    dependencies=[Depends(require_permission("users:create", "write"))],
)
async def create_user(
    body: CreateUserRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    if body.role == UserRole.admin.value:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号唯一，不可创建",
        )

    if not _is_manager_or_admin(current_user):
        if body.role not in (None, UserRole.operator.value):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="仅可创建运营者角色账号",
            )
        if body.role_ids:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="非管理员不可直接分配 RBAC3 角色",
            )

    existing = await db.execute(select(User).where(User.username == body.username))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="该用户名已存在",
        )
    if body.email:
        email_existing = await db.execute(select(User).where(User.email == body.email))
        if email_existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="该邮箱已被使用",
            )

    valid, msg = validate_password_format(body.password)
    if not valid:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=msg,
        )

    if not _is_manager_or_admin(current_user):
        creation_req = UserCreationRequest(
            requester_id=current_user.id,
            username=body.username,
            email=body.email,
            hashed_password=hash_password(body.password),
            nickname=body.nickname,
            role=body.role or UserRole.operator.value,
            avatar_url=body.avatar_url,
        )
        db.add(creation_req)
        await db.flush()
        await db.commit()
        raise HTTPException(
            status_code=status.HTTP_202_ACCEPTED,
            detail="账号创建申请已提交审核，请等待管理员审批",
        )

    user = User(
        username=body.username,
        email=body.email,
        hashed_password=hash_password(body.password),
        nickname=body.nickname,
        role=body.role or UserRole.operator.value,
        avatar_url=body.avatar_url,
    )
    db.add(user)
    await db.flush()
    await db.refresh(user)

    role_ids = body.role_ids
    if not role_ids and body.role:
        role_result = await db.execute(
            select(RBACRole).where(RBACRole.name == body.role)
        )
        role_obj = role_result.scalar_one_or_none()
        if role_obj:
            role_ids = [role_obj.id]

    if role_ids:
        violations = await validate_user_role_assignments(
            user.id, set(role_ids), db
        )
        if violations:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="; ".join(violations),
            )
        for role_id in role_ids:
            db.add(RBACUserRoleAssignment(
                user_id=user.id,
                role_id=role_id,
                grant_type="direct",
                created_at=datetime.utcnow(),
            ))

    await db.commit()
    await db.refresh(user)
    return await _build_user_item(user, db)


@router.get(
    "/users/{user_id}",
    response_model=UserDetailResponse,
    dependencies=[Depends(require_permission("users:read", "read"))],
)
async def get_user(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )
    return await _build_user_item(user, db)


@router.put(
    "/users/{user_id}",
    response_model=UserDetailResponse,
    dependencies=[Depends(require_permission("users:update", "write"))],
)
async def update_user(
    user_id: str,
    body: UpdateUserRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if not _is_manager_or_admin(current_user) and user.id != current_user.id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅可编辑自己的信息",
        )

    if _is_admin(user):
        if body.role_ids is not None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="超级管理员角色不可修改",
            )

    update_data = body.model_dump(exclude_unset=True)

    if "email" in update_data and update_data["email"] and update_data["email"] != user.email:
        existing = await db.execute(
            select(User).where(User.email == update_data["email"], User.id != user_id)
        )
        if existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="该邮箱已被其他用户使用",
            )

    for field, value in update_data.items():
        if field == "role_ids":
            continue
        setattr(user, field, value)

    if body.role_ids is not None:
        violations = await validate_user_role_assignments(
            user.id, set(body.role_ids), db
        )
        if violations:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="; ".join(violations),
            )

        await db.execute(
            delete(RBACUserRoleAssignment).where(
                RBACUserRoleAssignment.user_id == user_id
            )
        )
        for role_id in body.role_ids:
            db.add(RBACUserRoleAssignment(
                user_id=user_id,
                role_id=role_id,
                grant_type="direct",
                created_at=datetime.utcnow(),
            ))

    user.updated_at = datetime.utcnow()
    await db.flush()
    await db.commit()
    await db.refresh(user)
    return await _build_user_item(user, db)


@router.put(
    "/users/{user_id}/roles",
    response_model=UserDetailResponse,
    dependencies=[Depends(require_permission("users:write", "write"))],
)
async def update_user_roles(
    user_id: str,
    body: UpdateUserRolesRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if _is_admin(user):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员角色不可修改",
        )

    violations = await validate_user_role_assignments(
        user.id, set(body.role_ids), db
    )
    if violations:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="; ".join(violations),
        )

    await db.execute(
        delete(RBACUserRoleAssignment).where(
            RBACUserRoleAssignment.user_id == user_id
        )
    )
    for role_id in body.role_ids:
        db.add(RBACUserRoleAssignment(
            user_id=user_id,
            role_id=role_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()
    await db.refresh(user)
    return await _build_user_item(user, db)


@router.put(
    "/users/{user_id}/password",
    dependencies=[Depends(require_permission("users:change_password", "write"))],
)
async def change_user_password(
    user_id: str,
    body: UpdateUserPasswordRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if not _is_manager_or_admin(current_user) and user.id != current_user.id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅可修改自己的密码",
        )

    valid, msg = validate_password_format(body.new_password)
    if not valid:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=msg,
        )

    is_admin_reset = _is_manager_or_admin(current_user)

    if not is_admin_reset:
        default_pwd = f"{user.username}123"
        current_is_default = verify_password(default_pwd, user.hashed_password)

        if not current_is_default:
            if not body.old_password:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="非默认密码，请输入旧密码进行验证",
                )
            if not verify_password(body.old_password, user.hashed_password):
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="旧密码验证失败",
                )

    user.hashed_password = hash_password(body.new_password)
    await db.flush()
    await db.commit()
    return {"message": "密码修改成功"}


@router.get(
    "/users/{user_id}/permissions",
    response_model=dict[str, UserPermissionItem],
    dependencies=[Depends(require_permission("users:read", "read"))],
)
async def get_user_permissions(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    effective = await get_user_effective_permissions(user_id, db)
    return {
        key: UserPermissionItem(key=key, read=access.read, write=access.write)
        for key, access in effective.items()
    }


@router.delete(
    "/users/{user_id}",
    status_code=status.HTTP_204_NO_CONTENT,
    dependencies=[Depends(require_permission("users:delete", "write"))],
)
async def delete_user(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )
    if _is_admin(user):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号不可移除",
        )
    if user.id == current_user.id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="不能删除自己的账号",
        )

    await db.execute(
        delete(RBACUserRoleAssignment).where(
            RBACUserRoleAssignment.user_id == user_id
        )
    )
    await db.delete(user)
    await db.commit()
    return None


@router.get(
    "/users/{user_id}/password-status",
    dependencies=[Depends(require_permission("users:read", "read"))],
)
async def get_password_status(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if not _is_manager_or_admin(current_user) and user.id != current_user.id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无权查看",
        )

    default_pwd = f"{user.username}123"
    is_default = verify_password(default_pwd, user.hashed_password)
    return {"is_default_password": is_default}
