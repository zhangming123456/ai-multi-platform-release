from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.core.security import hash_password, validate_password_format, verify_password
from app.database import get_db
from app.models.custom_role import CustomRole
from app.models.user import User, UserRole
from app.models.user_creation_request import UserCreationRequest
from app.routers.roles import BUILTIN_ROLE_NAMES
from app.schemas.auth import (
    UserCreate,
    UserInfo,
    UserPasswordChange,
    UserUpdate,
)

router = APIRouter(prefix="/api/users", tags=["用户管理"])


async def _validate_role(role: str, db: AsyncSession):
    if role in BUILTIN_ROLE_NAMES:
        return
    result = await db.execute(select(CustomRole).where(CustomRole.name == role))
    if not result.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"角色 '{role}' 不存在",
        )


@router.get("/", response_model=list[UserInfo])
async def list_users(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("accounts")),
):
    result = await db.execute(select(User).order_by(User.created_at.desc()))
    return result.scalars().all()


@router.post("/", response_model=UserInfo, status_code=status.HTTP_201_CREATED)
async def create_user(
    request: UserCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("user:create", "write")),
):
    if current_user.role not in (UserRole.admin, UserRole.manager):
        if request.role not in (None, "operator"):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="仅可创建运营者角色账号",
            )

    if request.role == "admin":
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号唯一，不可创建",
        )

    await _validate_role(request.role, db)

    existing = await db.execute(select(User).where(User.username == request.username))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="该用户名已存在",
        )
    if request.email:
        email_existing = await db.execute(select(User).where(User.email == request.email))
        if email_existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="该邮箱已被使用",
            )

    if current_user.role not in (UserRole.admin, UserRole.manager):
        creation_req = UserCreationRequest(
            requester_id=current_user.id,
            username=request.username,
            email=request.email,
            hashed_password=hash_password(request.password),
            nickname=request.nickname,
            role=request.role,
            avatar_url=request.avatar_url,
        )
        db.add(creation_req)
        await db.flush()
        await db.commit()
        raise HTTPException(
            status_code=status.HTTP_202_ACCEPTED,
            detail="账号创建申请已提交审核，请等待管理员审批",
        )

    user = User(
        username=request.username,
        email=request.email,
        hashed_password=hash_password(request.password),
        nickname=request.nickname,
        role=request.role,
        avatar_url=request.avatar_url,
    )
    db.add(user)
    await db.flush()
    await db.refresh(user)
    return UserInfo.model_validate(user)


@router.put("/{user_id}", response_model=UserInfo)
async def update_user(
    user_id: str,
    request: UserUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("user:update")),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")

    if current_user.role not in (UserRole.admin, UserRole.manager):
        if user.id != current_user.id:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="仅可编辑自己的信息",
            )
        if request.role is not None and request.role != user.role:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无权限切换角色",
            )

    if user.role == "admin" and "role" in (request.model_dump(exclude_unset=True)):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员角色不可修改",
        )

    if "role" in (request.model_dump(exclude_unset=True)) and request.role == "admin":
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号唯一，不可将其他用户设为超级管理员",
        )

    update_data = request.model_dump(exclude_unset=True)

    if "role" in update_data:
        await _validate_role(update_data["role"], db)

    if "username" in update_data and update_data["username"] != user.username:
        existing = await db.execute(
            select(User).where(User.username == update_data["username"], User.id != user_id)
        )
        if existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="该用户名已被使用",
            )

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
        setattr(user, field, value)

    await db.flush()
    await db.refresh(user)
    return UserInfo.model_validate(user)


@router.put("/{user_id}/password")
async def change_user_password(
    user_id: str,
    request: UserPasswordChange,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("user:change_password", "write")),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")

    if current_user.role not in (UserRole.admin, UserRole.manager) and user.id != current_user.id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅可修改自己的密码",
        )

    valid, msg = validate_password_format(request.new_password)
    if not valid:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=msg,
        )

    is_admin_reset = current_user.role in (UserRole.admin, UserRole.manager)

    if not is_admin_reset:
        default_pwd = f"{user.username}123"
        current_is_default = verify_password(default_pwd, user.hashed_password)

        if not current_is_default:
            if not request.old_password:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="非默认密码，请输入旧密码进行验证",
                )
            if not verify_password(request.old_password, user.hashed_password):
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="旧密码验证失败",
                )

    user.hashed_password = hash_password(request.new_password)
    await db.flush()
    return {"message": "密码修改成功"}


@router.get("/{user_id}/password-status")
async def get_password_status(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")

    if current_user.role not in (UserRole.admin, UserRole.manager) and user.id != current_user.id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无权查看",
        )

    default_pwd = f"{user.username}123"
    is_default = verify_password(default_pwd, user.hashed_password)
    return {"is_default_password": is_default}


@router.delete("/{user_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_user(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("user:delete")),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="用户不存在")
    if user.role == "admin":
        raise HTTPException(status_code=400, detail="超级管理员账号不可移除")
    if user.id == current_user.id:
        raise HTTPException(status_code=400, detail="不能删除自己的账号")
    await db.delete(user)
    await db.flush()
