from __future__ import annotations

from typing import Callable

from fastapi import Depends, HTTPException, status
from fastapi.security import OAuth2PasswordBearer
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import decode_access_token
from app.database import get_db
from app.models.user import User
from app.services.rbac_service import has_permission_direct

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


def require_permission(permission_key: str, mode: str = "read") -> Callable:
    if mode not in {"read", "write"}:
        raise ValueError(f"Unsupported permission mode: {mode}")

    async def _checker(
        current_user: User = Depends(get_current_user),
        db: AsyncSession = Depends(get_db),
    ) -> User:
        has_access = await has_permission_direct(
            current_user.id, permission_key, mode, db
        )
        if has_access:
            return current_user

        if mode == "read":
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无查看权限",
            )
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无写入权限",
        )

    return _checker
