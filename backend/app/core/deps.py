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
    user_id: str = payload.get("sub")
    if user_id is None:
        raise credentials_exception
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise credentials_exception
    return user


async def get_admin_user(
    current_user: User = Depends(get_current_user),
) -> User:
    if current_user.role not in (UserRole.admin, UserRole.manager):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="需要管理员权限",
        )
    return current_user


async def get_reviewer_user(
    current_user: User = Depends(get_current_user),
) -> User:
    if current_user.role not in (UserRole.admin, UserRole.manager, UserRole.reviewer):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="需要审核权限",
        )
    return current_user


async def _get_user_permission_map(user: User, db: AsyncSession) -> dict[str, dict[str, bool]]:
    from app.routers.permissions import ALL_PERMISSION_KEYS, DEFAULT_ROLE_PERMISSIONS

    if user.role == UserRole.admin:
        return {k: {"read": True, "write": True} for k in ALL_PERMISSION_KEYS}

    custom_result = await db.execute(
        select(UserPermission).where(UserPermission.user_id == user.id)
    )
    custom_rows = custom_result.scalars().all()
    if custom_rows:
        return {r.permission_key: {"read": r.can_read, "write": r.can_write} for r in custom_rows}

    result = await db.execute(
        select(RolePermission).where(RolePermission.role == str(user.role))
    )
    rows = result.scalars().all()
    if rows:
        return {r.permission_key: {"read": r.can_read, "write": r.can_write} for r in rows}

    keys = DEFAULT_ROLE_PERMISSIONS.get(str(user.role), [])
    return {k: {"read": True, "write": True} for k in keys}


def require_permission(permission_key: str, mode: str = "read") -> Callable:
    async def _checker(
        current_user: User = Depends(get_current_user),
        db: AsyncSession = Depends(get_db),
    ) -> User:
        if current_user.role == UserRole.admin:
            return current_user
        perm_map = await _get_user_permission_map(current_user, db)
        access = perm_map.get(permission_key)
        if access is None:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无操作权限",
            )
        if mode == "write" and not access.get("write", False):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无写入权限",
            )
        if mode == "read" and not access.get("read", False):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无查看权限",
            )
        return current_user

    return _checker
