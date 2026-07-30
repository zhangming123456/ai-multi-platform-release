from __future__ import annotations

from datetime import datetime

from sqlalchemy import select, text
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource, _infer_resource_type
from app.models.rbac_role import RBACRole
from app.models.rbac_role_hierarchy import RBACRoleHierarchy
from app.models.rbac_role_permission import RBACRolePermission
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment
from app.models.user import User


BUILTIN_ROLES = [
    {
        "name": "admin",
        "display_name": "超级管理员",
        "is_super_admin": True,
        "is_builtin": True,
        "role_type": "admin",
    },
    {
        "name": "manager",
        "display_name": "管理员",
        "is_super_admin": False,
        "is_builtin": True,
        "role_type": "admin",
    },
    {
        "name": "operator",
        "display_name": "运营者",
        "is_super_admin": False,
        "is_builtin": True,
        "role_type": "other",
    },
    {
        "name": "reviewer",
        "display_name": "审核员",
        "is_super_admin": False,
        "is_builtin": True,
        "role_type": "other",
    },
]


RBAC_RESOURCES: list[dict] = [
    # ---------- 页面级权限 ----------
    {"key": "dashboard:read", "name": "仪表盘", "description": "系统首页仪表盘，展示核心数据概览"},
    {"key": "platforms:read", "name": "平台管理", "description": "管理已接入的第三方内容平台"},
    {"key": "content:read", "name": "内容列表", "description": "查看和管理所有内容列表"},
    {"key": "publish:read", "name": "发布管理", "description": "管理内容发布任务和发布计划"},
    {"key": "templates:read", "name": "模板管理", "description": "管理内容创作模板"},
    {"key": "review:read", "name": "内容审核", "description": "管理内容审核流程"},
    {"key": "sql_review:read", "name": "SQL审核", "description": "管理SQL变更审核流程"},
    {"key": "accounts:read", "name": "平台账号", "description": "管理各平台的登录账号信息"},
    {"key": "token_plan:read", "name": "Token方案", "description": "管理API Token用量方案"},
    {"key": "api_docs:read", "name": "API文档", "description": "查看系统API接口文档"},
    {"key": "db:read", "name": "数据库控制台", "description": "访问数据库控制台"},
    {"key": "users:read", "name": "用户管理", "description": "管理系统用户列表和基本信息"},
    {"key": "permissions:read", "name": "权限管理", "description": "管理角色权限分配"},
    {"key": "roles:read", "name": "角色管理", "description": "管理角色定义和角色继承关系"},
    {"key": "constraints:read", "name": "约束管理", "description": "管理职责分离约束规则"},
    # ---------- 系统管理操作 ----------
    {"key": "permissions:manage:read", "name": "查看权限字典", "description": "创建、编辑、删除权限资源定义"},
    {"key": "permissions:manage:write", "name": "维护权限字典", "description": "创建、编辑、删除权限资源定义"},
    {"key": "roles:manage:write", "name": "维护角色", "description": "创建、编辑、删除角色定义和层级关系"},
    {"key": "constraints:manage:write", "name": "维护约束", "description": "创建、编辑、删除职责分离约束规则"},
    # ---------- 内容操作 ----------
    {"key": "content:create:write", "name": "创建内容", "description": "创建新的内容条目"},
    {"key": "content:update:write", "name": "编辑内容", "description": "编辑已有内容条目的标题、正文等信息"},
    {"key": "content:delete:write", "name": "删除内容", "description": "删除已有内容条目"},
    {"key": "content:ai_generate:write", "name": "AI生成内容", "description": "使用AI辅助生成内容"},
    # ---------- 发布操作 ----------
    {"key": "publish:create:write", "name": "创建发布", "description": "创建内容发布任务"},
    {"key": "publish:retry:write", "name": "重试发布", "description": "重新执行失败的发布任务"},
    # ---------- 模板操作 ----------
    {"key": "templates:create:write", "name": "创建模板", "description": "创建新的内容模板"},
    {"key": "templates:update:write", "name": "编辑模板", "description": "编辑已有的内容模板"},
    {"key": "templates:delete:write", "name": "删除模板", "description": "删除已有的内容模板"},
    # ---------- 审核操作 ----------
    {"key": "review:submit:write", "name": "提交审核", "description": "将内容提交至审核流程"},
    {"key": "review:approve:write", "name": "通过审核", "description": "批准待审核的内容"},
    {"key": "review:reject:write", "name": "驳回审核", "description": "驳回审核不通过的内容"},
    # ---------- 数据库操作 ----------
    {"key": "db:execute:write", "name": "执行SQL", "description": "在数据库控制台中执行SQL语句"},
    # ---------- 用户管理操作 ----------
    {"key": "users:create:write", "name": "创建用户", "description": "创建新的系统用户"},
    {"key": "users:update:read", "name": "查看用户", "description": "查看用户详细信息"},
    {"key": "users:update:write", "name": "编辑用户", "description": "编辑用户的昵称、邮箱等基本信息"},
    {"key": "users:delete:write", "name": "删除用户", "description": "删除系统用户"},
    {"key": "users:change_password:write", "name": "修改用户密码", "description": "修改用户的登录密码"},
    {"key": "users:custom_permissions:write", "name": "自定义用户权限", "description": "为个别用户配置自定义权限覆盖"},
    # ---------- 平台账号操作 ----------
    {"key": "account:create:write", "name": "创建平台账号", "description": "创建新的平台登录账号"},
    {"key": "account:update:write", "name": "编辑平台账号", "description": "编辑平台账号信息"},
    {"key": "account:delete:write", "name": "删除平台账号", "description": "删除平台登录账号"},
    {"key": "account:check:write", "name": "校验平台账号", "description": "校验平台账号的有效性"},
    # ---------- SQL变更操作 ----------
    {"key": "db_change:submit:write", "name": "提交SQL变更", "description": "提交SQL变更申请"},
    {"key": "db_change:approve:write", "name": "通过SQL变更", "description": "批准SQL变更申请"},
    {"key": "db_change:reject:write", "name": "驳回SQL变更", "description": "驳回SQL变更申请"},
    # ---------- 模型配置操作 ----------
    {"key": "model_config:create:write", "name": "创建模型配置", "description": "创建新的AI模型配置"},
    {"key": "model_config:update:write", "name": "编辑模型配置", "description": "编辑AI模型配置参数"},
    {"key": "model_config:delete:write", "name": "删除模型配置", "description": "删除AI模型配置"},
    # ---------- SQL历史 ----------
    {"key": "db_history:view:read", "name": "查看SQL历史", "description": "查看SQL执行历史记录"},
]

ALL_PERMISSION_KEYS: list[str] = [r["key"] for r in RBAC_RESOURCES]


DEFAULT_ROLE_PERMISSIONS: dict[str, list[str]] = {
    "admin": ALL_PERMISSION_KEYS,
    "manager": [
        "dashboard:read", "platforms:read", "content:read", "publish:read",
        "templates:read", "review:read", "sql_review:read", "accounts:read",
        "token_plan:read", "api_docs:read",
        "permissions:read", "roles:read", "constraints:read",
        "permissions:manage:read", "permissions:manage:write", "roles:manage:write", "constraints:manage:write",
        "users:read", "users:create:write", "users:update:read", "users:update:write",
        "users:delete:write", "users:change_password:write", "users:custom_permissions:write",
        "content:create:write", "content:update:write", "content:delete:write", "content:ai_generate:write",
        "review:submit:write", "review:approve:write", "review:reject:write",
        "db_change:submit:write", "db_change:approve:write", "db_change:reject:write",
        "templates:create:write", "templates:update:write", "templates:delete:write",
        "account:create:write", "account:update:write", "account:delete:write", "account:check:write",
        "publish:create:write", "publish:retry:write",
        "model_config:create:write", "model_config:update:write", "model_config:delete:write",
        "db:execute:write", "db_history:view:read",
    ],
    "operator": [
        "dashboard:read", "platforms:read", "content:read", "publish:read",
        "templates:read", "accounts:read", "token_plan:read", "api_docs:read",
        "users:read", "users:create:write", "users:update:write", "users:change_password:write",
        "users:custom_permissions:write", "roles:read",
        "content:create:write", "content:update:write", "content:delete:write", "content:ai_generate:write",
        "review:submit:write", "db_change:submit:write",
        "templates:create:write", "templates:update:write", "templates:delete:write",
        "account:create:write", "account:update:write", "account:delete:write", "account:check:write",
        "publish:create:write", "publish:retry:write",
    ],
    "reviewer": [
        "dashboard:read", "content:read", "review:read", "sql_review:read",
        "platforms:read", "accounts:read",
        "review:approve:write", "review:reject:write",
        "db_change:approve:write", "db_change:reject:write",
        "templates:create:write", "templates:update:write", "templates:delete:write",
    ],
}


def _map_legacy_permission(legacy_key: str) -> str:
    """Map a legacy 2-segment permission key to the new 3-segment format."""
    legacy_map: dict[str, str] = {
        "db:history:read": "db_history:view:read",
        "db:execute": "db:execute:write",
        "db_history:read": "db_history:view:read",
    }

    if legacy_key in legacy_map:
        return legacy_map[legacy_key]

    if legacy_key.count(":") == 2:
        return legacy_key

    write_ops = {"create", "update", "delete", "approve", "reject", "submit",
                 "retry", "check", "execute", "ai_generate",
                 "change_password", "custom_permissions"}
    read_ops = {"read"}

    if ":" in legacy_key:
        prefix, operation = legacy_key.rsplit(":", 1)
        if legacy_key.startswith("user:"):
            operation = legacy_key.split(":", 1)[1]
            return f"users:{operation}:write"
        if legacy_key.startswith("template:"):
            operation = legacy_key.split(":", 1)[1]
            return f"templates:{operation}:write"
        if operation in write_ops:
            return f"{prefix}:{operation}:write"
        if operation in read_ops:
            return f"{prefix}:read:read"
        return f"{prefix}:{operation}:write"

    single_map: dict[str, str] = {
        "accounts": "users:read:read",
        "permission_manage": "permissions:read",
        "database": "db:read",
    }
    if legacy_key in single_map:
        return single_map[legacy_key]
    return f"{legacy_key}:read"


async def _ensure_builtin_roles(db: AsyncSession) -> None:
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(
        [r["name"] for r in BUILTIN_ROLES]
    )))
    existing_by_name = {role.name: role for role in result.scalars().all()}

    for role_def in BUILTIN_ROLES:
        if role_def["name"] in existing_by_name:
            continue
        db.add(RBACRole(
            name=role_def["name"],
            display_name=role_def["display_name"],
            role_type=role_def["role_type"],
            is_super_admin=role_def["is_super_admin"],
            is_builtin=role_def["is_builtin"],
            created_at=datetime.utcnow(),
            updated_at=datetime.utcnow(),
        ))

    await db.commit()


async def _ensure_resources_and_permissions(db: AsyncSession) -> None:
    """Create RBAC resources and permissions from RBAC_RESOURCES.

    Each entry in RBAC_RESOURCES creates one RBACResource and one RBACPermission.
    Resource type is inferred from the key format:
      - {name}:read              → page
      - {name}:{operation}:{read|write} → action
    """
    resource_keys = [r["key"] for r in RBAC_RESOURCES]
    result = await db.execute(select(RBACResource).where(RBACResource.key.in_(resource_keys)))
    existing_resources = {r.key: r for r in result.scalars().all()}

    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(ALL_PERMISSION_KEYS)))
    existing_permissions = {p.key: p for p in result.scalars().all()}

    resource_cache: dict[str, RBACResource] = {}

    for resource_def in RBAC_RESOURCES:
        resource_key = resource_def["key"]
        if resource_key in existing_resources:
            resource = existing_resources[resource_key]
        elif resource_key in resource_cache:
            resource = resource_cache[resource_key]
        else:
            resource = RBACResource(
                key=resource_key,
                name=resource_def["name"],
                description=resource_def.get("description"),
                is_active=True,
                created_at=datetime.utcnow(),
            )
            db.add(resource)
            resource_cache[resource_key] = resource

        if resource_key in existing_permissions:
            continue

        if resource.id is None:
            await db.flush()

        operation = _extract_operation(resource_key)
        db.add(RBACPermission(
            resource_id=resource.id,
            operation=operation,
            key=resource_key,
            is_active=True,
            created_at=datetime.utcnow(),
        ))

    await db.commit()


def _extract_operation(key: str) -> str:
    """Extract the operation from a permission key.

    {name}:read                    → operation = "read"
    {name}:{operation}:{read|write} → operation = middle segment
    """
    segments = key.split(":")
    if len(segments) == 2:
        return segments[-1]
    if len(segments) >= 3:
        return segments[-2]
    return segments[-1]


async def _ensure_builtin_role_permissions(db: AsyncSession) -> None:
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(
        list(DEFAULT_ROLE_PERMISSIONS.keys())
    )))
    roles_by_name = {role.name: role for role in result.scalars().all()}

    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(ALL_PERMISSION_KEYS)))
    permissions_by_key = {perm.key: perm for perm in result.scalars().all()}

    role_permission_pairs = [
        (roles_by_name[role_name].id, permissions_by_key[key].id)
        for role_name, keys in DEFAULT_ROLE_PERMISSIONS.items()
        if role_name in roles_by_name
        for key in keys
        if key in permissions_by_key
    ]

    if not role_permission_pairs:
        return

    role_ids = {pair[0] for pair in role_permission_pairs}
    permission_ids = {pair[1] for pair in role_permission_pairs}
    result = await db.execute(
        select(RBACRolePermission.role_id, RBACRolePermission.permission_id).where(
            RBACRolePermission.role_id.in_(role_ids),
            RBACRolePermission.permission_id.in_(permission_ids),
        )
    )
    existing_pairs = set(result.all())

    for role_id, permission_id in role_permission_pairs:
        if (role_id, permission_id) in existing_pairs:
            continue
        db.add(RBACRolePermission(
            role_id=role_id,
            permission_id=permission_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()


async def sync_user_role_assignments(db: AsyncSession) -> None:
    result = await db.execute(select(User.id, User.role))
    users = result.all()

    if not users:
        return

    role_names = {user.role for user in users}
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(role_names)))
    roles_by_name = {role.name: role for role in result.scalars().all()}

    user_role_pairs = [
        (user.id, roles_by_name[user.role].id)
        for user in users
        if user.role in roles_by_name
    ]

    if not user_role_pairs:
        return

    user_ids = {pair[0] for pair in user_role_pairs}
    role_ids = {pair[1] for pair in user_role_pairs}
    result = await db.execute(
        select(RBACUserRoleAssignment.user_id, RBACUserRoleAssignment.role_id).where(
            RBACUserRoleAssignment.user_id.in_(user_ids),
            RBACUserRoleAssignment.role_id.in_(role_ids),
        )
    )
    existing_pairs = set(result.all())

    for user_id, role_id in user_role_pairs:
        if (user_id, role_id) in existing_pairs:
            continue
        db.add(RBACUserRoleAssignment(
            user_id=user_id,
            role_id=role_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()


async def migrate_legacy_role_permissions(db: AsyncSession) -> None:
    """Migrate legacy role_permissions table data into RBAC3 role permissions."""
    try:
        result = await db.execute(
            text("SELECT role, permission_key, can_read, can_write FROM role_permissions")
        )
    except Exception:
        return

    legacy_records = result.mappings().all()
    if not legacy_records:
        return

    role_names = {record["role"] for record in legacy_records}
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(role_names)))
    roles_by_name = {role.name: role for role in result.scalars().all()}

    permission_keys = [
        _map_legacy_permission(record["permission_key"])
        for record in legacy_records
    ]
    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(permission_keys)))
    permissions_by_key = {perm.key: perm for perm in result.scalars().all()}

    role_permission_pairs = [
        (roles_by_name[record["role"]].id, permissions_by_key[_map_legacy_permission(record["permission_key"])].id)
        for record in legacy_records
        if (record["can_read"] or record["can_write"])
        and record["role"] in roles_by_name
        and _map_legacy_permission(record["permission_key"]) in permissions_by_key
    ]

    if not role_permission_pairs:
        return

    role_ids = {pair[0] for pair in role_permission_pairs}
    permission_ids = {pair[1] for pair in role_permission_pairs}
    result = await db.execute(
        select(RBACRolePermission.role_id, RBACRolePermission.permission_id).where(
            RBACRolePermission.role_id.in_(role_ids),
            RBACRolePermission.permission_id.in_(permission_ids),
        )
    )
    existing_pairs = set(result.all())

    for role_id, permission_id in role_permission_pairs:
        if (role_id, permission_id) in existing_pairs:
            continue
        db.add(RBACRolePermission(
            role_id=role_id,
            permission_id=permission_id,
            grant_type="direct",
            created_at=datetime.utcnow(),
        ))

    await db.commit()


async def init_rbac_system(db: AsyncSession) -> None:
    await _ensure_builtin_roles(db)
    await _ensure_resources_and_permissions(db)
    await _ensure_builtin_role_permissions(db)
    await sync_user_role_assignments(db)
    await migrate_legacy_role_permissions(db)
