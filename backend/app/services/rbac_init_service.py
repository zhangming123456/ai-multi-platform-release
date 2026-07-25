from __future__ import annotations

from datetime import datetime

from sqlalchemy import select, text
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource
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


# RBAC3 resources and their operations. Page resources always include a read
# operation; action resources expose the operations they support.
RBAC_RESOURCES: list[dict] = [
    {"key": "dashboard", "name": "仪表盘", "type": "page", "operations": ["read"]},
    {"key": "platforms", "name": "平台管理", "type": "page", "operations": ["read"]},
    {"key": "content", "name": "内容工坊", "type": "page", "operations": ["read", "create", "update", "delete", "ai_generate"]},
    {"key": "publish", "name": "发布管理", "type": "page", "operations": ["read", "create", "retry"]},
    {"key": "templates", "name": "模板中心", "type": "page", "operations": ["read", "create", "update", "delete"]},
    {"key": "review", "name": "内容审核", "type": "page", "operations": ["read", "submit", "approve", "reject"]},
    {"key": "sql_review", "name": "SQL 审核", "type": "page", "operations": ["read"]},
    {"key": "accounts", "name": "平台账号", "type": "page", "operations": ["read"]},
    {"key": "token_plan", "name": "Token 配置", "type": "page", "operations": ["read"]},
    {"key": "api_docs", "name": "API 文档", "type": "page", "operations": ["read"]},
    {"key": "db", "name": "数据库管理", "type": "page", "operations": ["read", "execute"]},
    {"key": "permissions", "name": "权限设置", "type": "page", "operations": ["read", "write"]},
    {"key": "roles", "name": "角色设置", "type": "page", "operations": ["read", "write"]},
    {"key": "constraints", "name": "职责分离", "type": "page", "operations": ["read", "write"]},
    {"key": "users", "name": "用户", "type": "action", "operations": ["read", "write", "create", "update", "delete", "change_password"]},
    {"key": "account", "name": "平台账号", "type": "action", "operations": ["read", "create", "update", "delete", "check"]},
    {"key": "db_change", "name": "SQL 变更", "type": "action", "operations": ["submit", "approve", "reject"]},
    {"key": "model_config", "name": "模型配置", "type": "action", "operations": ["create", "update", "delete"]},
    {"key": "db_history", "name": "SQL 历史", "type": "action", "operations": ["read"]},
]


ALL_PERMISSION_KEYS: list[str] = [
    f"{resource['key']}:{operation}"
    for resource in RBAC_RESOURCES
    for operation in resource["operations"]
]


DEFAULT_ROLE_PERMISSIONS: dict[str, list[str]] = {
    "admin": ALL_PERMISSION_KEYS,
    "manager": [
        "dashboard:read", "platforms:read", "content:read", "publish:read",
        "templates:read", "review:read", "sql_review:read", "accounts:read", "account:read",
        "token_plan:read", "api_docs:read", "permissions:read", "permissions:write",
        "roles:read", "roles:write", "constraints:read", "constraints:write",
        "users:read", "users:write", "users:create", "users:update", "users:change_password",
        "content:create", "content:update", "content:delete", "content:ai_generate",
        "review:submit", "review:approve", "review:reject",
        "db_change:submit", "db_change:approve", "db_change:reject",
        "templates:create", "templates:update", "templates:delete",
        "account:create", "account:update", "account:delete", "account:check",
        "publish:create", "publish:retry",
        "model_config:create", "model_config:update", "model_config:delete",
        "db:execute", "db_history:read",
    ],
    "operator": [
        "dashboard:read", "platforms:read", "content:read", "publish:read",
        "templates:read", "accounts:read", "account:read", "token_plan:read", "api_docs:read",
        "users:create", "users:update", "users:change_password",
        "content:create", "content:update", "content:delete", "content:ai_generate",
        "review:submit",
        "db_change:submit",
        "templates:create", "templates:update", "templates:delete",
        "account:create", "account:update", "account:delete", "account:check",
        "publish:create", "publish:retry",
    ],
    "reviewer": [
        "dashboard:read", "content:read", "review:read", "sql_review:read",
        "platforms:read", "account:read",
        "review:approve", "review:reject",
        "db_change:approve", "db_change:reject",
        "templates:create", "templates:update", "templates:delete",
    ],
}


def _map_legacy_permission(legacy_key: str) -> tuple[str, str]:
    """Map a legacy permission key to (resource_key, operation)."""
    if legacy_key == "db:history:read":
        return "db_history", "read"
    if legacy_key.startswith("user:"):
        operation = legacy_key.split(":", 1)[1]
        return "users", operation
    if legacy_key.startswith("template:"):
        operation = legacy_key.split(":", 1)[1]
        return "templates", operation
    if legacy_key.startswith("account:"):
        operation = legacy_key.split(":", 1)[1]
        return "account", operation
    if legacy_key == "accounts":
        return "users", "read"
    if legacy_key == "permission_manage":
        return "permissions", "read"
    if legacy_key == "database":
        return "db", "read"
    if ":" in legacy_key:
        resource_key, operation = legacy_key.rsplit(":", 1)
        return resource_key, operation
    return legacy_key, "read"


def _permission_key(resource_key: str, operation: str) -> str:
    return f"{resource_key}:{operation}"


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
    resource_keys = [resource["key"] for resource in RBAC_RESOURCES]
    result = await db.execute(select(RBACResource).where(RBACResource.key.in_(resource_keys)))
    existing_resources = {resource.key: resource for resource in result.scalars().all()}

    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(ALL_PERMISSION_KEYS)))
    existing_permissions = {perm.key: perm for perm in result.scalars().all()}

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
                type=resource_def["type"],
                is_active=True,
                created_at=datetime.utcnow(),
            )
            db.add(resource)
            resource_cache[resource_key] = resource

        for operation in resource_def["operations"]:
            perm_key = _permission_key(resource_key, operation)
            if perm_key in existing_permissions:
                continue

            if resource.id is None:
                await db.flush()

            db.add(RBACPermission(
                resource_id=resource.id,
                operation=operation,
                key=perm_key,
                is_active=True,
                created_at=datetime.utcnow(),
            ))

    await db.commit()


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
    """Migrate legacy role_permissions table data into RBAC3 role permissions.

    This function no longer imports the old constants/models at module level.
    It reads from the legacy table via raw SQL so that the migration still works
    when the old model files have been removed.
    """
    try:
        result = await db.execute(
            text("SELECT role, permission_key, can_read, can_write FROM role_permissions")
        )
    except Exception:
        # Legacy table does not exist; nothing to migrate.
        return

    legacy_records = result.mappings().all()
    if not legacy_records:
        return

    role_names = {record["role"] for record in legacy_records}
    result = await db.execute(select(RBACRole).where(RBACRole.name.in_(role_names)))
    roles_by_name = {role.name: role for role in result.scalars().all()}

    permission_keys = [
        _permission_key(*_map_legacy_permission(record["permission_key"]))
        for record in legacy_records
    ]
    result = await db.execute(select(RBACPermission).where(RBACPermission.key.in_(permission_keys)))
    permissions_by_key = {perm.key: perm for perm in result.scalars().all()}

    role_permission_pairs = [
        (roles_by_name[record["role"]].id, permissions_by_key[_permission_key(*_map_legacy_permission(record["permission_key"]))].id)
        for record in legacy_records
        if (record["can_read"] or record["can_write"])
        and record["role"] in roles_by_name
        and _permission_key(*_map_legacy_permission(record["permission_key"])) in permissions_by_key
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
