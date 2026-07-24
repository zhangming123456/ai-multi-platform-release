from __future__ import annotations

from typing import Callable, Optional

from fastapi import Depends, HTTPException, status
from fastapi.security import OAuth2PasswordBearer
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import decode_access_token
from app.database import get_db
from app.models.role_permission import RolePermission
from app.models.user import User, UserRole
from app.models.user_permission import UserPermission

oauth2_scheme = OAuth2PasswordBearer(tokenUrl="/api/auth/login")


async def get_current_user(
    token: str = Depends(oauth2_scheme),
    db: AsyncSession = Depends(get_db),
) -> User:
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="无法验证凭据",
        headers={"WWW-Authenticate": "Bearer"},
    )
    payload = decode_access_token(token)
    if payload is None:
        raise credentials_exception
    user_id: Optional[int] = payload.get("sub")
    if user_id is None:
        raise credentials_exception
    result = await db.execute(select(User).where(User.id == int(user_id)))
    user = result.scalar_one_or_none()
    if user is None:
        raise credentials_exception
    return user


async def get_admin_user(
    current_user: User = Depends(get_current_user),
) -> User:
    if current_user.role != UserRole.admin:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="需要管理员权限",
        )
    return current_user


async def get_reviewer_user(
    current_user: User = Depends(get_current_user),
) -> User:
    if current_user.role not in (UserRole.admin, UserRole.reviewer):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="需要审核权限",
        )
    return current_user


async def _get_user_permission_keys(user: User, db: AsyncSession) -> list[str]:
    from app.routers.permissions import ALL_PERMISSION_KEYS, DEFAULT_ROLE_PERMISSIONS

    if user.role == UserRole.admin:
        return ALL_PERMISSION_KEYS

    custom_result = await db.execute(
        select(UserPermission.permission_key).where(UserPermission.user_id == user.id)
    )
    custom_keys = [row[0] for row in custom_result.all()]
    if custom_keys:
        return custom_keys

    result = await db.execute(
        select(RolePermission.permission_key).where(RolePermission.role == str(user.role))
    )
    keys = [row[0] for row in result.all()]
    if not keys:
        keys = DEFAULT_ROLE_PERMISSIONS.get(str(user.role), [])
    return keys


def require_permission(permission_key: str) -> Callable:
    async def _checker(
        current_user: User = Depends(get_current_user),
        db: AsyncSession = Depends(get_db),
    ) -> User:
        if current_user.role == UserRole.admin:
            return current_user
        keys = await _get_user_permission_keys(current_user, db)
        if permission_key not in keys:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无操作权限",
            )
        return current_user

    return _checker
