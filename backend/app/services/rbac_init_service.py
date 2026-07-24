from __future__ import annotations

import json

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    RBACPermission,
    RBACResource,
    RBACRole,
    RBACRoleHierarchy,
    RBACRolePermission,
    RBACUserRoleAssignment,
)
from app.models.user import User


BUILTIN_ROLES = [
    {
        "name": "super_admin",
        "display_name": "超级管理员",
        "description": "系统最高权限，拥有所有页面和操作权限，不可修改",
        "role_type": "admin",
        "is_system": True,
        "is_super_admin": True,
    },
    {
        "name": "admin",
        "display_name": "管理员",
        "description": "系统管理，可配置所有权限",
        "role_type": "admin",
        "is_system": True,
        "is_super_admin": False,
    },
    {
        "name": "manager",
        "display_name": "业务管理员",
        "description": "业务管理，通常不接触数据库敏感权限",
        "role_type": "admin",
        "is_system": True,
        "is_super_admin": False,
    },
    {
        "name": "operator",
        "display_name": "运营者",
        "description": "日常运营操作",
        "role_type": "other",
        "is_system": True,
        "is_super_admin": False,
    },
    {
        "name": "reviewer",
        "display_name": "审核员",
        "description": "内容/数据审核",
        "role_type": "other",
        "is_system": True,
        "is_super_admin": False,
    },
    {
        "name": "auditor",
        "display_name": "审计员",
        "description": "只读查看日志与数据，用于合规",
        "role_type": "other",
        "is_system": True,
        "is_super_admin": False,
    },
]

BUILTIN_RESOURCES = [
    {"key": "dashboard", "name": "仪表盘", "type": "page", "parent_key": None, "sort_order": 1},
    {"key": "platforms", "name": "平台管理", "type": "page", "parent_key": None, "sort_order": 2},
    {"key": "content", "name": "内容工坊", "type": "page", "parent_key": None, "sort_order": 3},
    {"key": "publish", "name": "发布管理", "type": "page", "parent_key": None, "sort_order": 4},
    {"key": "templates", "name": "模板中心", "type": "page", "parent_key": None, "sort_order": 5},
    {"key": "review", "name": "内容审核", "type": "page", "parent_key": None, "sort_order": 6},
    {"key": "sql_review", "name": "SQL审核", "type": "page", "parent_key": None, "sort_order": 7},
    {"key": "accounts", "name": "账号管理", "type": "page", "parent_key": None, "sort_order": 8},
    {"key": "token_plan", "name": "Token配置", "type": "page", "parent_key": None, "sort_order": 9},
    {"key": "api_docs", "name": "API文档", "type": "page", "parent_key": None, "sort_order": 10},
    {"key": "database", "name": "数据库管理", "type": "page", "parent_key": None, "sort_order": 11},
    {"key": "permission_manage", "name": "权限管理", "type": "page", "parent_key": None, "sort_order": 12},
    {"key": "user_perm_manage", "name": "用户权限配置", "type": "action", "parent_key": None, "sort_order": 13},
    {"key": "users", "name": "用户管理", "type": "module", "parent_key": None, "sort_order": 14},
    {"key": "roles", "name": "角色管理", "type": "module", "parent_key": None, "sort_order": 15},
    {"key": "permissions", "name": "权限管理", "type": "module", "parent_key": None, "sort_order": 16},
    {"key": "constraints", "name": "约束管理", "type": "module", "parent_key": None, "sort_order": 17},
    {"key": "sessions", "name": "会话管理", "type": "module", "parent_key": None, "sort_order": 18},
    {"key": "audit_logs", "name": "审计日志", "type": "page", "parent_key": None, "sort_order": 19},
    {"key": "user_creation_review", "name": "用户创建审核", "type": "page", "parent_key": None, "sort_order": 20},
    {"key": "template", "name": "模板操作", "type": "action", "parent_key": None, "sort_order": 20},
    {"key": "account", "name": "平台账号操作", "type": "action", "parent_key": None, "sort_order": 21},
    {"key": "db_change", "name": "SQL变更", "type": "action", "parent_key": None, "sort_order": 22},
    {"key": "model_config", "name": "模型配置", "type": "action", "parent_key": None, "sort_order": 23},
    {"key": "db", "name": "数据库操作", "type": "action", "parent_key": None, "sort_order": 24},
]

PERMISSION_OPERATIONS = ["create", "read", "update", "delete"]

PAGE_RESOURCES = [r for r in BUILTIN_RESOURCES if r["type"] in ("page", "module")]

OPERATION_LABELS = {
    "create": "创建",
    "read": "查看",
    "update": "编辑",
    "delete": "删除",
    "execute": "执行",
    "approve": "审核通过",
    "reject": "审核驳回",
}

EXTRA_PERMISSIONS = [
    {"resource_key": "content", "operation": "ai_generate", "name": "AI生成"},
    {"resource_key": "review", "operation": "submit", "name": "提交审核"},
    {"resource_key": "review", "operation": "approve", "name": "审核通过"},
    {"resource_key": "review", "operation": "reject", "name": "审核驳回"},
    {"resource_key": "db_change", "operation": "submit", "name": "提交SQL变更"},
    {"resource_key": "db_change", "operation": "approve", "name": "SQL变更审核通过"},
    {"resource_key": "db_change", "operation": "reject", "name": "SQL变更审核驳回"},
    {"resource_key": "template", "operation": "create", "name": "创建模板"},
    {"resource_key": "template", "operation": "update", "name": "编辑模板"},
    {"resource_key": "template", "operation": "delete", "name": "删除模板"},
    {"resource_key": "account", "operation": "create", "name": "添加平台账号"},
    {"resource_key": "account", "operation": "update", "name": "编辑平台账号"},
    {"resource_key": "account", "operation": "delete", "name": "删除平台账号"},
    {"resource_key": "account", "operation": "check", "name": "检测账号状态"},
    {"resource_key": "publish", "operation": "create", "name": "创建发布任务"},
    {"resource_key": "publish", "operation": "retry", "name": "重试发布任务"},
    {"resource_key": "model_config", "operation": "create", "name": "创建模型配置"},
    {"resource_key": "model_config", "operation": "update", "name": "编辑模型配置"},
    {"resource_key": "model_config", "operation": "delete", "name": "删除模型配置"},
    {"resource_key": "db", "operation": "execute", "name": "执行SQL命令"},
    {"resource_key": "db", "operation": "history:read", "name": "查看SQL历史"},
    {"resource_key": "users", "operation": "create", "name": "创建用户"},
    {"resource_key": "users", "operation": "update", "name": "编辑用户"},
    {"resource_key": "users", "operation": "delete", "name": "删除用户"},
    {"resource_key": "users", "operation": "change_password", "name": "修改用户密码"},
    {"resource_key": "users", "operation": "assign_role", "name": "分配角色"},
    {"resource_key": "users", "operation": "revoke_role", "name": "撤销角色"},
    {"resource_key": "user_creation_review", "operation": "read", "name": "查看用户创建审核"},
    {"resource_key": "user_creation_review", "operation": "approve", "name": "通过用户创建审核"},
    {"resource_key": "user_creation_review", "operation": "reject", "name": "驳回用户创建审核"},
]


async def init_builtin_roles(db: AsyncSession):
    for role_data in BUILTIN_ROLES:
        result = await db.execute(
            select(RBACRole).where(RBACRole.name == role_data["name"])
        )
        role = result.scalar_one_or_none()
        if not role:
            role = RBACRole(**role_data)
            db.add(role)
    await db.flush()


async def init_builtin_resources_and_permissions(db: AsyncSession):
    resource_map: dict[str, RBACResource] = {}

    for res_data in BUILTIN_RESOURCES:
        result = await db.execute(
            select(RBACResource).where(RBACResource.key == res_data["key"])
        )
        resource = result.scalar_one_or_none()
        if not resource:
            resource = RBACResource(
                key=res_data["key"],
                name=res_data["name"],
                type=res_data["type"],
                sort_order=res_data["sort_order"],
            )
            db.add(resource)
        resource_map[res_data["key"]] = resource

    await db.flush()

    for res_key, resource in resource_map.items():
        for op in PERMISSION_OPERATIONS:
            perm_key = f"{res_key}:{op}"
            result = await db.execute(
                select(RBACPermission).where(RBACPermission.key == perm_key)
            )
            if not result.scalar_one_or_none():
                perm = RBACPermission(
                    resource_id=resource.id,
                    operation=op,
                    key=perm_key,
                    description=f"{resource.name} - {OPERATION_LABELS.get(op, op)}",
                )
                db.add(perm)

    for extra in EXTRA_PERMISSIONS:
        res_key = extra["resource_key"]
        op = extra["operation"]
        perm_key = f"{res_key}:{op}"
        resource = resource_map.get(res_key)
        if not resource:
            continue
        result = await db.execute(
            select(RBACPermission).where(RBACPermission.key == perm_key)
        )
        if not result.scalar_one_or_none():
            perm = RBACPermission(
                resource_id=resource.id,
                operation=op,
                key=perm_key,
                description=extra.get("name", perm_key),
            )
            db.add(perm)

    await db.flush()


async def init_role_hierarchy(db: AsyncSession):
    role_name_to_id: dict[str, str] = {}
    result = await db.execute(select(RBACRole))
    for role in result.scalars().all():
        role_name_to_id[role.name] = role.id

    hierarchy = [
        ("super_admin", "admin"),
        ("admin", "manager"),
        ("admin", "auditor"),
        ("manager", "operator"),
        ("manager", "reviewer"),
    ]

    for parent_name, child_name in hierarchy:
        parent_id = role_name_to_id.get(parent_name)
        child_id = role_name_to_id.get(child_name)
        if not parent_id or not child_id:
            continue
        existing = await db.execute(
            select(RBACRoleHierarchy).where(
            RBACRoleHierarchy.parent_role_id == parent_id,
            RBACRoleHierarchy.child_role_id == child_id,
        )
        )
        if not existing.scalar_one_or_none():
            db.add(RBACRoleHierarchy(
                parent_role_id=parent_id,
                child_role_id=child_id,
                inheritance_type="explicit",
            ))

    await db.flush()


async def init_default_permissions_for_roles(db: AsyncSession):
    role_name_to_id: dict[str, str] = {}
    result = await db.execute(select(RBACRole))
    for role in result.scalars().all():
        role_name_to_id[role.name] = role.id

    perm_key_to_id: dict[str, str] = {}
    result = await db.execute(select(RBACPermission))
    for perm in result.scalars().all():
        perm_key_to_id[perm.key] = perm.id

    default_perms = {
        "admin": list(perm_key_to_id.keys()),
        "manager": [
            "dashboard:read",
            "platforms:read",
            "content:read", "content:create", "content:update", "content:delete", "content:ai_generate",
            "publish:read", "publish:create", "publish:retry",
            "templates:read", "template:create", "template:update", "template:delete",
            "review:read", "review:submit", "review:approve", "review:reject",
            "sql_review:read", "db_change:submit", "db_change:approve", "db_change:reject",
            "accounts:read", "account:create", "account:update", "account:delete", "account:check",
            "token_plan:read",
            "api_docs:read",
            "permission_manage:read",
            "users:read", "users:create", "users:update", "users:delete", "users:change_password",
            "users:assign_role", "users:revoke_role",
            "user_creation_review:read", "user_creation_review:approve", "user_creation_review:reject",
            "roles:read",
            "permissions:read",
        ],
        "operator": [
            "dashboard:read",
            "platforms:read",
            "content:read", "content:create", "content:update", "content:delete", "content:ai_generate",
            "publish:read", "publish:create", "publish:retry",
            "templates:read", "template:create", "template:update", "template:delete",
            "review:read", "review:submit",
            "sql_review:read", "db_change:submit",
            "accounts:read", "account:create", "account:update", "account:delete", "account:check",
            "token_plan:read",
            "api_docs:read",
            "users:read", "users:create", "users:update", "users:change_password",
        ],
        "reviewer": [
            "dashboard:read",
            "platforms:read",
            "content:read",
            "review:read", "review:approve", "review:reject",
            "sql_review:read", "db_change:approve", "db_change:reject",
            "templates:read", "template:create", "template:update", "template:delete",
            "api_docs:read",
        ],
        "auditor": [
            "dashboard:read",
            "audit_logs:read",
            "api_docs:read",
        ],
    }

    for role_name, perm_keys in default_perms.items():
        role_id = role_name_to_id.get(role_name)
        if not role_id:
            continue

        existing_result = await db.execute(
            select(RBACRolePermission.permission_id).where(
                RBACRolePermission.role_id == role_id,
                RBACRolePermission.grant_type == "direct",
            )
        )
        existing_perm_ids = {row[0] for row in existing_result.all()}

        for pk in perm_keys:
            pid = perm_key_to_id.get(pk)
            if pid and pid not in existing_perm_ids:
                db.add(RBACRolePermission(
                    role_id=role_id,
                    permission_id=pid,
                    grant_type="direct",
                ))

    await db.flush()


async def assign_super_admin_to_first_user(db: AsyncSession):
    super_admin_role = None
    result = await db.execute(
        select(RBACRole).where(RBACRole.name == "super_admin")
    )
    super_admin_role = result.scalar_one_or_none()
    if not super_admin_role:
        return

    admin_user = None
    result = await db.execute(select(User).where(User.username == "admin"))
    admin_user = result.scalar_one_or_none()
    if not admin_user:
        result = await db.execute(select(User).limit(1))
        admin_user = result.scalar_one_or_none()

    if admin_user:
        existing = await db.execute(
            select(RBACUserRoleAssignment).where(
                RBACUserRoleAssignment.user_id == admin_user.id,
                RBACUserRoleAssignment.role_id == super_admin_role.id,
            )
        )
        if not existing.scalar_one_or_none():
            db.add(RBACUserRoleAssignment(
                user_id=admin_user.id,
                role_id=super_admin_role.id,
            ))
            await db.flush()


async def sync_user_role_assignments(db: AsyncSession):
    role_name_to_id: dict[str, str] = {}
    result = await db.execute(select(RBACRole))
    for role in result.scalars().all():
        role_name_to_id[role.name] = role.id

    result = await db.execute(select(User))
    for user in result.scalars().all():
        rbac_role_id = role_name_to_id.get(user.role)
        if not rbac_role_id:
            continue
        existing = await db.execute(
            select(RBACUserRoleAssignment).where(
                RBACUserRoleAssignment.user_id == user.id,
                RBACUserRoleAssignment.role_id == rbac_role_id,
            )
        )
        if not existing.scalar_one_or_none():
            db.add(RBACUserRoleAssignment(
                user_id=user.id,
                role_id=rbac_role_id,
            ))
    await db.flush()


async def init_rbac_system(db: AsyncSession):
    await init_builtin_roles(db)
    await init_builtin_resources_and_permissions(db)
    await init_role_hierarchy(db)
    await init_default_permissions_for_roles(db)
    await assign_super_admin_to_first_user(db)
    await sync_user_role_assignments(db)
    await db.commit()
