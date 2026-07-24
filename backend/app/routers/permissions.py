from __future__ import annotations

from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_admin_user, get_current_user
from app.database import get_db
from app.models.custom_role import CustomRole
from app.models.role_permission import RolePermission
from app.models.user import User, UserRole
from app.models.user_permission import UserPermission
from app.routers.roles import BUILTIN_ROLES, BUILTIN_ROLE_NAMES

router = APIRouter(prefix="/api/permissions", tags=["权限管理"])

ALL_PERMISSIONS = [
    {"key": "dashboard", "name": "仪表盘", "group": "页面权限", "type": "page"},
    {"key": "platforms", "name": "平台管理", "group": "页面权限", "type": "page"},
    {"key": "content", "name": "内容工坊", "group": "页面权限", "type": "page"},
    {"key": "publish", "name": "发布管理", "group": "页面权限", "type": "page"},
    {"key": "templates", "name": "模板中心", "group": "页面权限", "type": "page"},
    {"key": "review", "name": "内容审核", "group": "页面权限", "type": "page"},
    {"key": "sql_review", "name": "SQL 审核", "group": "页面权限", "type": "page"},
    {"key": "accounts", "name": "账号管理", "group": "页面权限", "type": "page"},
    {"key": "token_plan", "name": "Token 配置", "group": "页面权限", "type": "page"},
    {"key": "api_docs", "name": "API 文档", "group": "页面权限", "type": "page"},
    {"key": "database", "name": "数据库管理", "group": "页面权限", "type": "page"},
    {"key": "permission_manage", "name": "权限管理", "group": "页面权限", "type": "page"},
    {"key": "user_perm_manage", "name": "用户权限配置", "group": "页面权限", "type": "action"},
    {"key": "user:create", "name": "创建用户", "group": "系统设置", "type": "action"},
    {"key": "user:update", "name": "编辑用户", "group": "系统设置", "type": "action"},
    {"key": "user:delete", "name": "删除用户", "group": "系统设置", "type": "action"},
    {"key": "user:change_password", "name": "修改用户密码", "group": "系统设置", "type": "action"},
    {"key": "content:create", "name": "创建内容", "group": "内容管理", "type": "action"},
    {"key": "content:update", "name": "编辑内容", "group": "内容管理", "type": "action"},
    {"key": "content:delete", "name": "删除内容", "group": "内容管理", "type": "action"},
    {"key": "content:ai_generate", "name": "AI 生成", "group": "内容管理", "type": "action"},
    {"key": "review:submit", "name": "提交内容审核", "group": "审核管理", "type": "action"},
    {"key": "review:approve", "name": "审核通过", "group": "审核管理", "type": "action"},
    {"key": "review:reject", "name": "审核驳回", "group": "审核管理", "type": "action"},
    {"key": "db_change:submit", "name": "提交SQL变更", "group": "审核管理", "type": "action"},
    {"key": "db_change:approve", "name": "SQL变更审核通过", "group": "审核管理", "type": "action"},
    {"key": "db_change:reject", "name": "SQL变更驳回", "group": "审核管理", "type": "action"},
    {"key": "template:create", "name": "创建模板", "group": "内容管理", "type": "action"},
    {"key": "template:update", "name": "编辑模板", "group": "内容管理", "type": "action"},
    {"key": "template:delete", "name": "删除模板", "group": "内容管理", "type": "action"},
    {"key": "account:create", "name": "添加平台账号", "group": "基础", "type": "action"},
    {"key": "account:update", "name": "编辑平台账号", "group": "基础", "type": "action"},
    {"key": "account:delete", "name": "删除平台账号", "group": "基础", "type": "action"},
    {"key": "account:check", "name": "检测账号状态", "group": "基础", "type": "action"},
    {"key": "publish:create", "name": "创建发布任务", "group": "内容管理", "type": "action"},
    {"key": "publish:retry", "name": "重试发布任务", "group": "内容管理", "type": "action"},
    {"key": "model_config:create", "name": "创建模型配置", "group": "系统设置", "type": "action"},
    {"key": "model_config:update", "name": "编辑模型配置", "group": "系统设置", "type": "action"},
    {"key": "model_config:delete", "name": "删除模型配置", "group": "系统设置", "type": "action"},
    {"key": "db:execute", "name": "执行SQL命令", "group": "系统设置", "type": "action"},
    {"key": "db:history:read", "name": "查看SQL历史", "group": "系统设置", "type": "action"},
]

ALL_PERMISSION_KEYS = [p["key"] for p in ALL_PERMISSIONS]

ADMIN_ONLY_PERMISSIONS = [
    "user_perm_manage",
    "permission_manage",
    "database",
    "db:execute",
    "db:history:read",
    "model_config:create",
    "model_config:update",
    "model_config:delete",
    "user:delete",
]

DATABASE_PERMISSIONS = [
    "database",
    "db:execute",
    "db:history:read",
]


async def _get_role_type(role: str, db: AsyncSession) -> str:
    if role in BUILTIN_ROLES:
        return BUILTIN_ROLES[role].get("role_type", "other")
    result = await db.execute(select(CustomRole.role_type).where(CustomRole.name == role))
    row = result.scalar_one_or_none()
    return row or "other"

DEFAULT_ROLE_PERMISSIONS: dict[str, list[str]] = {
    "admin": ALL_PERMISSION_KEYS,
    "manager": [
        "dashboard", "platforms", "content", "publish", "templates", "review", "sql_review",
        "accounts", "token_plan", "api_docs", "permission_manage",
        "user:create", "user:update", "user:change_password", "user_perm_manage",
        "content:create", "content:update", "content:delete", "content:ai_generate",
        "review:submit", "review:approve", "review:reject",
        "db_change:submit", "db_change:approve", "db_change:reject",
        "template:create", "template:update", "template:delete",
        "account:create", "account:update", "account:delete", "account:check",
        "publish:create", "publish:retry",
    ],
    "operator": [
        "dashboard", "content", "publish", "templates", "platforms",
        "accounts", "token_plan", "api_docs",
        "user:create", "user:update", "user:change_password",
        "content:create", "content:update", "content:delete", "content:ai_generate",
        "review:submit",
        "db_change:submit",
        "template:create", "template:update", "template:delete",
        "account:create", "account:update", "account:delete", "account:check",
        "publish:create", "publish:retry",
    ],
    "reviewer": [
        "dashboard", "content", "review", "sql_review", "platforms",
        "review:approve", "review:reject",
        "db_change:approve", "db_change:reject",
        "template:create", "template:update", "template:delete",
    ],
}


class PermissionAccess(BaseModel):
    read: bool = True
    write: bool = True

class PermissionDef(BaseModel):
    key: str
    name: str
    group: str
    type: str


class RolePermissionResponse(BaseModel):
    role: str
    permissions: dict[str, PermissionAccess]


class UpdateRolePermissionRequest(BaseModel):
    role: str
    permissions: dict[str, PermissionAccess]


class AllPermissionsResponse(BaseModel):
    permissions: list[PermissionDef]
    roles: list[str]
    role_permissions: dict[str, dict[str, PermissionAccess]]


class UserPermissionResponse(BaseModel):
    user_id: str
    role: str
    role_permissions: dict[str, PermissionAccess]
    custom_permissions: Optional[dict[str, PermissionAccess]] = None
    effective_permissions: dict[str, PermissionAccess]
    is_editable: bool = False


class UpdateUserPermissionRequest(BaseModel):
    permissions: dict[str, PermissionAccess]


def _list_to_access_map(keys: list[str]) -> dict[str, PermissionAccess]:
    return {k: PermissionAccess(read=True, write=True) for k in keys}


def _access_map_to_list(perm_map: dict[str, PermissionAccess]) -> list[str]:
    return [k for k in sorted(perm_map.keys()) if perm_map[k].read or perm_map[k].write]


@router.get("/all", response_model=AllPermissionsResponse)
async def get_all_permissions(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    role_perms: dict[str, dict[str, PermissionAccess]] = {}
    all_roles = list(BUILTIN_ROLE_NAMES)

    custom_result = await db.execute(select(CustomRole.name).order_by(CustomRole.created_at.asc()))
    for row in custom_result.all():
        all_roles.append(row[0])

    for role in all_roles:
        result = await db.execute(
            select(RolePermission).where(RolePermission.role == role)
        )
        rows = result.scalars().all()
        if rows:
            role_perms[role] = {
                r.permission_key: PermissionAccess(read=r.can_read, write=r.can_write) for r in rows
            }
        else:
            role_perms[role] = _list_to_access_map(DEFAULT_ROLE_PERMISSIONS.get(role, []))

    return AllPermissionsResponse(
        permissions=[PermissionDef(**p) for p in ALL_PERMISSIONS],
        roles=all_roles,
        role_permissions=role_perms,
    )


@router.get("/role/{role}", response_model=RolePermissionResponse)
async def get_role_permissions(
    role: str,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    result = await db.execute(
        select(RolePermission).where(RolePermission.role == role)
    )
    rows = result.scalars().all()
    if rows:
        perm_map = {r.permission_key: PermissionAccess(read=r.can_read, write=r.can_write) for r in rows}
    else:
        perm_map = _list_to_access_map(DEFAULT_ROLE_PERMISSIONS.get(role, []))
    return RolePermissionResponse(role=role, permissions=perm_map)


@router.put("/role/{role}", response_model=RolePermissionResponse)
async def update_role_permissions(
    role: str,
    body: UpdateRolePermissionRequest,
    admin: User = Depends(get_admin_user),
    db: AsyncSession = Depends(get_db),
):
    all_valid_roles = list(BUILTIN_ROLE_NAMES)
    custom_result = await db.execute(select(CustomRole.name))
    for row in custom_result.all():
        all_valid_roles.append(row[0])

    if role not in all_valid_roles:
        raise HTTPException(status_code=400, detail="无效的角色")

    if role == UserRole.admin.value:
        raise HTTPException(status_code=400, detail="超级管理员拥有所有权限，不可修改")

    invalid_keys = [k for k in body.permissions if k not in ALL_PERMISSION_KEYS]
    if invalid_keys:
        raise HTTPException(status_code=400, detail=f"无效的权限: {', '.join(invalid_keys)}")

    role_type = await _get_role_type(role, db)
    if role_type != "admin":
        db_keys_in_body = [k for k in body.permissions if k in DATABASE_PERMISSIONS]
        if db_keys_in_body:
            raise HTTPException(
                status_code=400,
                detail=f"该角色类型不支持配置数据库相关权限: {', '.join(db_keys_in_body)}",
            )

    if role_type != "admin":
        admin_only_in_body = [k for k in body.permissions if k in ADMIN_ONLY_PERMISSIONS]
        if admin_only_in_body:
            raise HTTPException(
                status_code=400,
                detail=f"以下权限不可分配给普通类型角色: {', '.join(admin_only_in_body)}",
            )

    await db.execute(delete(RolePermission).where(RolePermission.role == role))
    for key, access in body.permissions.items():
        db.add(RolePermission(role=role, permission_key=key, can_read=access.read, can_write=access.write))
    await db.commit()

    return RolePermissionResponse(role=role, permissions=body.permissions)


@router.get("/user/{user_id}", response_model=UserPermissionResponse)
async def get_user_permission_detail(
    user_id: str,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    from app.models.user import User as UserModel

    if current_user.role not in ("admin", "manager") and current_user.id != user_id:
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="无权限查看")

    result = await db.execute(select(UserModel).where(UserModel.id == user_id))
    target_user = result.scalar_one_or_none()
    if not target_user:
        raise HTTPException(status_code=404, detail="用户不存在")

    role_perm_map = await _get_role_permissions(str(target_user.role), db)

    custom_result = await db.execute(
        select(UserPermission).where(UserPermission.user_id == user_id)
    )
    custom_rows = custom_result.scalars().all()

    if custom_rows:
        custom_map = {r.permission_key: PermissionAccess(read=r.can_read, write=r.can_write) for r in custom_rows}
        effective = custom_map
    else:
        custom_map = None
        effective = role_perm_map

    is_editable = current_user.role in ("admin", "manager") and str(target_user.role) != UserRole.admin.value

    return UserPermissionResponse(
        user_id=user_id,
        role=str(target_user.role),
        role_permissions=role_perm_map,
        custom_permissions=custom_map,
        effective_permissions=effective,
        is_editable=is_editable,
    )


@router.put("/user/{user_id}", response_model=UserPermissionResponse)
async def update_user_permissions(
    user_id: str,
    body: UpdateUserPermissionRequest,
    admin: User = Depends(get_admin_user),
    db: AsyncSession = Depends(get_db),
):
    from app.models.user import User as UserModel

    result = await db.execute(select(UserModel).where(UserModel.id == user_id))
    target_user = result.scalar_one_or_none()
    if not target_user:
        raise HTTPException(status_code=404, detail="用户不存在")

    if str(target_user.role) == UserRole.admin.value:
        raise HTTPException(status_code=400, detail="超级管理员拥有所有权限，不可修改")

    role_perm_map = await _get_role_permissions(str(target_user.role), db)

    invalid_keys = [k for k in body.permissions if k not in role_perm_map]
    if invalid_keys:
        raise HTTPException(
            status_code=400,
            detail=f"以下权限不在该用户角色范围内: {', '.join(invalid_keys)}",
        )

    user_role_type = await _get_role_type(str(target_user.role), db)
    if user_role_type != "admin":
        db_keys_in_body = [k for k in body.permissions if k in DATABASE_PERMISSIONS]
        if db_keys_in_body:
            raise HTTPException(
                status_code=400,
                detail=f"该用户角色类型不支持配置数据库相关权限: {', '.join(db_keys_in_body)}",
            )

    await db.execute(delete(UserPermission).where(UserPermission.user_id == user_id))
    for key, access in body.permissions.items():
        db.add(UserPermission(user_id=user_id, permission_key=key, can_read=access.read, can_write=access.write))
    await db.commit()

    return UserPermissionResponse(
        user_id=user_id,
        role=str(target_user.role),
        role_permissions=role_perm_map,
        custom_permissions=body.permissions,
        effective_permissions=body.permissions,
    )


@router.delete("/user/{user_id}/custom", response_model=UserPermissionResponse)
async def reset_user_permissions(
    user_id: str,
    admin: User = Depends(get_admin_user),
    db: AsyncSession = Depends(get_db),
):
    from app.models.user import User as UserModel

    result = await db.execute(select(UserModel).where(UserModel.id == user_id))
    target_user = result.scalar_one_or_none()
    if not target_user:
        raise HTTPException(status_code=404, detail="用户不存在")

    if str(target_user.role) == UserRole.admin.value:
        raise HTTPException(status_code=400, detail="超级管理员无需重置")

    await db.execute(delete(UserPermission).where(UserPermission.user_id == user_id))
    await db.commit()

    role_perm_map = await _get_role_permissions(str(target_user.role), db)

    return UserPermissionResponse(
        user_id=user_id,
        role=str(target_user.role),
        role_permissions=role_perm_map,
        custom_permissions=None,
        effective_permissions=role_perm_map,
    )


async def _get_role_permissions(role: str, db: AsyncSession) -> dict[str, PermissionAccess]:
    result = await db.execute(
        select(RolePermission).where(RolePermission.role == role)
    )
    rows = result.scalars().all()
    if rows:
        return {r.permission_key: PermissionAccess(read=r.can_read, write=r.can_write) for r in rows}
    default_keys = DEFAULT_ROLE_PERMISSIONS.get(role, [])
    return _list_to_access_map(default_keys)


async def get_user_permissions(user: User, db: AsyncSession) -> dict[str, PermissionAccess]:
    if user.role == UserRole.admin:
        return _list_to_access_map(ALL_PERMISSION_KEYS)

    custom_result = await db.execute(
        select(UserPermission).where(UserPermission.user_id == user.id)
    )
    custom_rows = custom_result.scalars().all()
    if custom_rows:
        perm_map = {r.permission_key: PermissionAccess(read=r.can_read, write=r.can_write) for r in custom_rows}
    else:
        perm_map = await _get_role_permissions(str(user.role), db)

    role_type = await _get_role_type(str(user.role), db)
    if role_type != "admin":
        perm_map = {k: v for k, v in perm_map.items() if k not in DATABASE_PERMISSIONS}
    return perm_map
