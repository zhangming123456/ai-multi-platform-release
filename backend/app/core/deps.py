from __future__ import annotations

from typing import Any, Callable, Optional

from fastapi import Depends, HTTPException, status
from fastapi.routing import APIRoute
from fastapi.security import OAuth2PasswordBearer
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import decode_access_token
from app.database import get_db
from app.models.user import User
from app.services.perm_expression import (
    evaluate_permission,
    extract_perm_keys,
    is_expression,
)
from app.services.rbac_service import get_user_effective_flat_permissions

oauth2_scheme = OAuth2PasswordBearer(tokenUrl="/api/auth/login")

_PERM_MARKER = "_requires_perm_info_"


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


def _infer_mode_from_key(permission_key: str) -> str:
    if is_expression(permission_key):
        return "read"
    parts = permission_key.split(":")
    if len(parts) >= 2 and parts[-1] in {"read", "write"}:
        return parts[-1]
    return "read"


def _describe_mode(mode: str) -> str:
    return "查看" if mode == "read" else "写入"


def require_permission(
    permission_key: str,
    context: Optional[dict[str, Any]] = None,
) -> Callable:
    mode = _infer_mode_from_key(permission_key)
    use_expression = is_expression(permission_key)

    if use_expression:
        async def _checker_expr(
            current_user: User = Depends(get_current_user),
            db: AsyncSession = Depends(get_db),
        ) -> User:
            flat_perms = await get_user_effective_flat_permissions(current_user.id, db)

            eval_ctx: dict[str, Any] = {
                "current_user": current_user,
            }
            if context:
                eval_ctx.update(context)

            try:
                has_access = evaluate_permission(permission_key, flat_perms, eval_ctx)
            except (SyntaxError, ValueError) as exc:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail=f"权限表达式错误: {exc}",
                )

            if has_access:
                return current_user

            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无访问权限",
            )

        return _checker_expr

    async def _checker(
        current_user: User = Depends(get_current_user),
        db: AsyncSession = Depends(get_db),
    ) -> User:
        flat_perms = await get_user_effective_flat_permissions(current_user.id, db)

        if permission_key in flat_perms:
            return current_user

        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=f"无{_describe_mode(mode)}权限",
        )

    return _checker


def RequiresPermissions(permission_expr: str, context: Optional[dict[str, Any]] = None):
    def decorator(endpoint):
        setattr(endpoint, _PERM_MARKER, {
            "expr": permission_expr,
            "ctx": context,
        })
        return endpoint
    return decorator


class PermAPIRoute(APIRoute):
    def __init__(self, path: str, endpoint: Callable, **kwargs: Any):
        perm_info = getattr(endpoint, _PERM_MARKER, None)
        if perm_info is not None:
            dep = require_permission(perm_info["expr"], perm_info["ctx"])
            if "dependencies" not in kwargs or kwargs["dependencies"] is None:
                kwargs["dependencies"] = []
            kwargs["dependencies"].append(Depends(dep))
        super().__init__(path, endpoint, **kwargs)
