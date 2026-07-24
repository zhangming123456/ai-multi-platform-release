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
from app.routers.roles import BUILTIN_ROLE_NAMES

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

DEFAULT_ROLE_PERMISSIONS: dict[str, list[str]] = {
    "admin": ALL_PERMISSION_KEYS,
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


class PermissionDef(BaseModel):
    key: str
    name: str
    group: str
    type: str


class RolePermissionResponse(BaseModel):
    role: str
    permissions: list[str]


class UpdateRolePermissionRequest(BaseModel):
    role: str
    permissions: list[str]


class AllPermissionsResponse(BaseModel):
    permissions: list[PermissionDef]
    roles: list[str]
    role_permissions: dict[str, list[str]]


class UserPermissionResponse(BaseModel):
    user_id: int
    role: str
    role_permissions: list[str]
    custom_permissions: Optional[list[str]] = None
    effective_permissions: list[str]


class UpdateUserPermissionRequest(BaseModel):
    permissions: list[str]


@router.get("/all", response_model=AllPermissionsResponse)
async def get_all_permissions(
    admin: User = Depends(get_admin_user),
    db: AsyncSession = Depends(get_db),
):
    role_perms: dict[str, list[str]] = {}
    all_roles = list(BUILTIN_ROLE_NAMES)

    custom_result = await db.execute(select(CustomRole.name).order_by(CustomRole.created_at.asc()))
    for row in custom_result.all():
        all_roles.append(row[0])

    for role in all_roles:
        result = await db.execute(
            select(RolePermission.permission_key).where(RolePermission.role == role)
        )
        keys = [row[0] for row in result.all()]
        if not keys:
            keys = DEFAULT_ROLE_PERMISSIONS.get(role, [])
        role_perms[role] = keys

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
        select(RolePermission.permission_key).where(RolePermission.role == role)
    )
    keys = [row[0] for row in result.all()]
    if not keys:
        keys = DEFAULT_ROLE_PERMISSIONS.get(role, [])
    return RolePermissionResponse(role=role, permissions=keys)


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

    admin_only_in_body = [k for k in body.permissions if k in ADMIN_ONLY_PERMISSIONS]
    if admin_only_in_body:
        raise HTTPException(
            status_code=400,
            detail=f"以下权限仅超级管理员拥有，不可分配给其他角色: {', '.join(admin_only_in_body)}",
        )

    await db.execute(delete(RolePermission).where(RolePermission.role == role))
    for key in body.permissions:
        db.add(RolePermission(role=role, permission_key=key))
    await db.commit()

    return RolePermissionResponse(role=role, permissions=body.permissions)


@router.get("/user/{user_id}", response_model=UserPermissionResponse)
async def get_user_permission_detail(
    user_id: int,
    admin: User = Depends(get_admin_user),
    db: AsyncSession = Depends(get_db),
):
    from app.models.user import User as UserModel

    result = await db.execute(select(UserModel).where(UserModel.id == user_id))
    target_user = result.scalar_one_or_none()
    if not target_user:
        raise HTTPException(status_code=404, detail="用户不存在")

    role_perms = await _get_role_permissions(str(target_user.role), db)

    custom_result = await db.execute(
        select(UserPermission.permission_key).where(UserPermission.user_id == user_id)
    )
    custom_keys = [row[0] for row in custom_result.all()]

    effective = custom_keys if custom_keys else role_perms

    return UserPermissionResponse(
        user_id=user_id,
        role=str(target_user.role),
        role_permissions=role_perms,
        custom_permissions=custom_keys if custom_keys else None,
        effective_permissions=effective,
    )


@router.put("/user/{user_id}", response_model=UserPermissionResponse)
async def update_user_permissions(
    user_id: int,
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

    role_perms = await _get_role_permissions(str(target_user.role), db)

    invalid_keys = [k for k in body.permissions if k not in role_perms]
    if invalid_keys:
        raise HTTPException(
            status_code=400,
            detail=f"以下权限不在该用户角色范围内: {', '.join(invalid_keys)}",
        )

    await db.execute(delete(UserPermission).where(UserPermission.user_id == user_id))
    for key in body.permissions:
        db.add(UserPermission(user_id=user_id, permission_key=key))
    await db.commit()

    return UserPermissionResponse(
        user_id=user_id,
        role=str(target_user.role),
        role_permissions=role_perms,
        custom_permissions=body.permissions,
        effective_permissions=body.permissions,
    )


@router.delete("/user/{user_id}/custom", response_model=UserPermissionResponse)
async def reset_user_permissions(
    user_id: int,
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

    role_perms = await _get_role_permissions(str(target_user.role), db)

    return UserPermissionResponse(
        user_id=user_id,
        role=str(target_user.role),
        role_permissions=role_perms,
        custom_permissions=None,
        effective_permissions=role_perms,
    )


async def _get_role_permissions(role: str, db: AsyncSession) -> list[str]:
    result = await db.execute(
        select(RolePermission.permission_key).where(RolePermission.role == role)
    )
    keys = [row[0] for row in result.all()]
    if not keys:
        keys = DEFAULT_ROLE_PERMISSIONS.get(role, [])
    return keys


async def get_user_permissions(user: User, db: AsyncSession) -> list[str]:
    if user.role == UserRole.admin:
        return ALL_PERMISSION_KEYS

    custom_result = await db.execute(
        select(UserPermission.permission_key).where(UserPermission.user_id == user.id)
    )
    custom_keys = [row[0] for row in custom_result.all()]
    if custom_keys:
        return custom_keys

    return await _get_role_permissions(str(user.role), db)
