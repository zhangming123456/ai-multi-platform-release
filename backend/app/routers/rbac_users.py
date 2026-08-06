from __future__ import annotations

from datetime import datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import PermAPIRoute, RequiresPermissions, get_current_user
from app.core.security import hash_password, validate_password_format, verify_password
from app.database import get_db
from app.models.notification import Notification, NotificationType
from app.models.rbac_permission import RBACPermission
from app.models.rbac_resource import RBACResource
from app.models.rbac_role import RBACRole
from app.models.rbac_role_permission import RBACRolePermission
from app.models.rbac_user_permission_override import RBACUserPermissionOverride
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment
from app.models.user import User, UserRole
from app.models.user_creation_request import UserCreationRequest
from app.schemas.auth import UserInfo
from app.services.notification_broadcaster import broadcaster
from app.services.rbac_constraint_service import validate_user_role_assignments
from app.services.rbac_service import (
    _closure_has_super_admin,
    _role_closure_role_ids,
    get_user_effective_flat_permissions,
    has_permission_direct,
)

router = APIRouter(prefix="/api/v2", tags=["RBAC 用户管理"], route_class=PermAPIRoute)


class RoleAssignmentInfo(BaseModel):
    id: str
    name: str
    display_name: str
    role_type: str = "other"

    model_config = {"from_attributes": True}


class UserListItem(UserInfo):
    roles: list[RoleAssignmentInfo] = []


class UserDetailResponse(UserListItem):
    pass


class _PermResourceRef(BaseModel):
    id: str
    key: str
    name: str
    description: Optional[str] = None

    model_config = {"from_attributes": True}


class AvailablePermission(BaseModel):
    id: str
    key: str
    operation: str
    is_active: bool
    created_at: datetime
    resource: _PermResourceRef

    model_config = {"from_attributes": True}


class UserPermissionsResponse(BaseModel):
    effective_permissions: dict[str, str]
    available_permissions: list[AvailablePermission]


class CreateUserRequest(BaseModel):
    username: str = Field(..., min_length=1, max_length=100)
    password: str = Field(..., min_length=6, max_length=255)
    nickname: str = Field(..., min_length=1, max_length=100)
    email: Optional[str] = Field(None, max_length=255)
    avatar_url: Optional[str] = None
    role: Optional[str] = "operator"
    role_ids: list[str] = Field(default_factory=list)


class UpdateUserRequest(BaseModel):
    nickname: Optional[str] = Field(None, min_length=1, max_length=100)
    email: Optional[str] = Field(None, max_length=255)
    avatar_url: Optional[str] = None
    role_ids: Optional[list[str]] = None


class UpdateUserRolesRequest(BaseModel):
    role_ids: list[str]


class UpdateUserPasswordRequest(BaseModel):
    new_password: str = Field(..., min_length=6, max_length=255)
    old_password: Optional[str] = None


async def _user_roles(user_id: str, db: AsyncSession) -> list[RoleAssignmentInfo]:
    result = await db.execute(
        select(RBACRole)
        .join(
            RBACUserRoleAssignment,
            RBACRole.id == RBACUserRoleAssignment.role_id,
        )
        .where(RBACUserRoleAssignment.user_id == user_id)
    )
    return [
        RoleAssignmentInfo(
            id=role.id, name=role.name, display_name=role.display_name, role_type=role.role_type
        )
        for role in result.scalars().all()
    ]


async def _build_user_item(user: User, db: AsyncSession) -> UserDetailResponse:
    base = UserInfo.model_validate(user)
    roles = await _user_roles(user.id, db)
    return UserDetailResponse(**base.model_dump(), roles=roles)


def _is_admin(user: User) -> bool:
    return user.role == UserRole.admin.value or str(user.role) == UserRole.admin.value


def _is_manager_or_admin(user: User) -> bool:
    return str(user.role) in {UserRole.admin.value, UserRole.manager.value}


async def _get_super_admin_role_id(db: AsyncSession) -> str | None:
    result = await db.execute(select(RBACRole.id).where(RBACRole.is_super_admin.is_(True)))
    row = result.first()
    return row[0] if row else None


async def _assert_not_assigning_super_admin(
    role_ids: set[str],
    user_id: str,
    db: AsyncSession,
) -> None:
    super_admin_role_id = await _get_super_admin_role_id(db)
    if not super_admin_role_id or super_admin_role_id not in role_ids:
        return
    if user_id == "1":
        return
    raise HTTPException(
        status_code=status.HTTP_400_BAD_REQUEST,
        detail="超级管理员角色仅限系统唯一管理员，不可分配给其他用户",
    )


async def _notify_user_role_change(
    user_id: str,
    user_nickname: str,
    role_ids: list[str],
    prev_role_ids: set[str],
    db: AsyncSession,
    current_user: User,
) -> None:
    if not role_ids:
        return
    if user_id == current_user.id:
        return

    new_role_set = set(role_ids)
    if new_role_set == prev_role_ids:
        return

    result = await db.execute(select(RBACRole.display_name).where(RBACRole.id.in_(role_ids)))
    role_names = [row[0] for row in result.all()]
    role_list = "、".join(role_names) if role_names else "无"
    prev_result = await db.execute(
        select(RBACRole.display_name).where(RBACRole.id.in_(prev_role_ids))
    )
    prev_role_names = [row[0] for row in prev_result.all()]
    prev_role_list = "、".join(prev_role_names) if prev_role_names else "无"

    if not prev_role_names and role_names:
        content = f"你的角色已被 {current_user.nickname} 添加「{role_list}」"
    elif prev_role_names and not role_names:
        content = f"你的角色已被 {current_user.nickname} 移除「{prev_role_list}」"
    else:
        content = (
            f"你的角色已被 {current_user.nickname} 从「{prev_role_list}」更新为「{role_list}」"
        )

    notification = Notification(
        user_id=user_id,
        type=NotificationType.role_permissions_updated,
        title="角色分配已变更",
        content=content,
    )
    db.add(notification)
    await db.flush()

    await broadcaster.broadcast(
        user_id,
        {
            "id": notification.id,
            "type": notification.type.value
            if hasattr(notification.type, "value")
            else notification.type,
            "title": notification.title,
            "content": notification.content,
            "related_id": notification.related_id,
            "is_read": notification.is_read,
            "created_at": notification.created_at.isoformat() if notification.created_at else None,
        },
    )


async def _notify_user_info_change(
    user_id: str,
    changed_fields: list[dict],
    db: AsyncSession,
    current_user: User,
) -> None:
    if not changed_fields or user_id == current_user.id:
        return

    from app.services.notification_dict_service import is_field_private, resolve_field

    def _mask(val: str) -> str:
        if not val:
            return ""
        length = len(val)
        if length == 1:
            return val
        if length == 2:
            return val[0] + "*"
        if length == 3:
            return val[0] + "*" + val[-1]
        if length >= 9:
            return val[:3] + "***" + val[-2:]
        return val[:2] + "***" + val[-1:]

    if len(changed_fields) == 1:
        item = changed_fields[0]
        field = item["field"]
        field_label = await resolve_field("user_fields", field)
        is_private = await is_field_private("user_fields", field)
        old_val = (item.get("old") or "").strip()
        new_val = (item.get("new") or "").strip()

        if is_private:
            old_display = _mask(old_val)
            new_display = _mask(new_val)
        else:
            old_display = old_val
            new_display = new_val

        if old_val and new_val:
            detail = f"从「{old_display}」更新为「{new_display}」"
        elif new_val:
            detail = f"添加「{new_display}」"
        elif old_val:
            detail = f"移除「{old_display}」"
        else:
            detail = ""
        content = (
            f"你的{field_label}已被 {current_user.nickname} 修改：{detail}"
            if detail
            else f"你的{field_label}已被 {current_user.nickname} 修改"
        )
    else:
        parts = []
        for item in changed_fields:
            field = item["field"]
            field_label = await resolve_field("user_fields", field)
            is_private = await is_field_private("user_fields", field)
            old_val = (item.get("old") or "").strip()
            new_val = (item.get("new") or "").strip()

            if is_private:
                old_display = _mask(old_val)
                new_display = _mask(new_val)
            else:
                old_display = old_val
                new_display = new_val

            if old_val and new_val:
                parts.append(f"{field_label} 从「{old_display}」更新为「{new_display}」")
            elif new_val:
                parts.append(f"{field_label} 添加「{new_display}」")
            elif old_val:
                parts.append(f"{field_label} 移除「{old_display}」")
        content = f"你的个人信息已被 {current_user.nickname} 修改：{'；'.join(parts)}"

    notification = Notification(
        user_id=user_id,
        type=NotificationType.role_updated,
        title="个人信息已变更",
        content=content,
    )
    db.add(notification)
    await db.flush()

    await broadcaster.broadcast(
        user_id,
        {
            "id": notification.id,
            "type": notification.type.value
            if hasattr(notification.type, "value")
            else notification.type,
            "title": notification.title,
            "content": notification.content,
            "related_id": notification.related_id,
            "is_read": notification.is_read,
            "created_at": notification.created_at.isoformat() if notification.created_at else None,
        },
    )


async def _get_user_role_names(user: User, db: AsyncSession) -> set[str]:
    """Get all role names assigned to a user."""
    result = await db.execute(
        select(RBACRole.name)
        .join(RBACUserRoleAssignment, RBACRole.id == RBACUserRoleAssignment.role_id)
        .where(RBACUserRoleAssignment.user_id == user.id)
    )
    return {row[0] for row in result.all()}


async def _can_access_user_target(
    current_user: User,
    target_user: User,
    db: AsyncSession,
    allow_self: bool = True,
) -> bool:
    """Check if current_user can access/manage target_user.

    Admin/manager roles can access all users.
    Non-admin roles (operator, reviewer) can only access users with the same role.
    """
    if _is_manager_or_admin(current_user):
        return True
    if allow_self and current_user.id == target_user.id:
        return True
    current_roles = await _get_user_role_names(current_user, db)
    target_roles = await _get_user_role_names(target_user, db)
    return bool(current_roles & target_roles)


@router.get(
    "/users",
    response_model=list[UserDetailResponse],
)
@RequiresPermissions("users:read")
async def list_users(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).order_by(User.created_at.desc()))
    users = result.scalars().all()

    if not _is_manager_or_admin(current_user):
        current_roles = await _get_user_role_names(current_user, db)
        filtered: list[User] = []
        for user in users:
            if user.id == current_user.id:
                filtered.append(user)
                continue
            target_roles = await _get_user_role_names(user, db)
            if current_roles & target_roles:
                filtered.append(user)
        users = filtered

    return [await _build_user_item(user, db) for user in users]


@router.post(
    "/users",
    response_model=UserDetailResponse,
    status_code=status.HTTP_201_CREATED,
)
@RequiresPermissions("users:create:write")
async def create_user(
    body: CreateUserRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    if body.role == UserRole.admin.value:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号唯一，不可创建",
        )

    if not _is_manager_or_admin(current_user):
        if body.role not in (None, UserRole.operator.value):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="仅可创建运营者角色账号",
            )
        if body.role_ids:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="非管理员不可直接分配 RBAC3 角色",
            )

    existing = await db.execute(select(User).where(User.username == body.username))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="该用户名已存在",
        )
    if body.email:
        email_existing = await db.execute(select(User).where(User.email == body.email))
        if email_existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="该邮箱已被使用",
            )

    valid, msg = validate_password_format(body.password)
    if not valid:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=msg,
        )

    if not _is_manager_or_admin(current_user):
        creation_req = UserCreationRequest(
            requester_id=current_user.id,
            username=body.username,
            email=body.email,
            hashed_password=hash_password(body.password),
            nickname=body.nickname,
            role=body.role or UserRole.operator.value,
            avatar_url=body.avatar_url,
        )
        db.add(creation_req)
        await db.flush()
        await db.commit()
        raise HTTPException(
            status_code=status.HTTP_202_ACCEPTED,
            detail="账号创建申请已提交审核，请等待管理员审批",
        )

    user = User(
        username=body.username,
        email=body.email,
        hashed_password=hash_password(body.password),
        nickname=body.nickname,
        role=body.role or UserRole.operator.value,
        avatar_url=body.avatar_url,
    )
    db.add(user)
    await db.flush()
    await db.refresh(user)

    role_ids = body.role_ids
    if not role_ids and body.role:
        role_result = await db.execute(select(RBACRole).where(RBACRole.name == body.role))
        role_obj = role_result.scalar_one_or_none()
        if role_obj:
            role_ids = [role_obj.id]

    if role_ids:
        violations = await validate_user_role_assignments(user.id, set(role_ids), db)
        if violations:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="; ".join(violations),
            )

        await _assert_not_assigning_super_admin(set(role_ids), user.id, db)

        for role_id in role_ids:
            db.add(
                RBACUserRoleAssignment(
                    user_id=user.id,
                    role_id=role_id,
                    grant_type="direct",
                    created_at=datetime.utcnow(),
                )
            )

    await db.commit()
    await db.refresh(user)
    return await _build_user_item(user, db)


@router.get(
    "/users/{user_id}",
    response_model=UserDetailResponse,
)
@RequiresPermissions("users:read")
async def get_user(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )
    if not await _can_access_user_target(current_user, user, db):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无法访问该用户：非管理角色只能操作同角色用户",
        )
    return await _build_user_item(user, db)


@router.put(
    "/users/{user_id}",
    response_model=UserDetailResponse,
)
@RequiresPermissions("users:update:write")
async def update_user(
    user_id: str,
    body: UpdateUserRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if not await _can_access_user_target(current_user, user, db):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无法访问该用户：非管理角色只能操作同角色用户",
        )

    if not _is_manager_or_admin(current_user) and body.role_ids is not None:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅管理员可修改角色分配",
        )

    if _is_admin(user) and body.role_ids is not None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员角色不可修改",
        )

    update_data = body.model_dump(exclude_unset=True)

    changed_info_fields: list[dict] = []
    for field, value in list(update_data.items()):
        if field == "role_ids":
            continue
        old_value = getattr(user, field, None)
        if old_value != value:
            setattr(user, field, value)
            changed_info_fields.append(
                {
                    "field": field,
                    "old": str(old_value) if old_value else "",
                    "new": str(value) if value else "",
                }
            )
        else:
            update_data.pop(field, None)

    if "email" in update_data and update_data["email"] and update_data["email"] != user.email:
        existing = await db.execute(
            select(User).where(User.email == update_data["email"], User.id != user_id)
        )
        if existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="该邮箱已被其他用户使用",
            )

    if body.role_ids is not None:
        violations = await validate_user_role_assignments(user.id, set(body.role_ids), db)
        if violations:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="; ".join(violations),
            )

        await _assert_not_assigning_super_admin(set(body.role_ids), user_id, db)

        prev_role_result = await db.execute(
            select(RBACUserRoleAssignment.role_id).where(RBACUserRoleAssignment.user_id == user_id)
        )
        prev_role_ids = {row[0] for row in prev_role_result.all()}

        await db.execute(
            delete(RBACUserRoleAssignment).where(RBACUserRoleAssignment.user_id == user_id)
        )
        for role_id in body.role_ids:
            db.add(
                RBACUserRoleAssignment(
                    user_id=user_id,
                    role_id=role_id,
                    grant_type="direct",
                    created_at=datetime.utcnow(),
                )
            )

        await _notify_user_role_change(
            user_id=user.id,
            user_nickname=user.nickname,
            role_ids=body.role_ids,
            prev_role_ids=prev_role_ids,
            db=db,
            current_user=current_user,
        )

    if changed_info_fields:
        await _notify_user_info_change(
            user_id=user.id,
            changed_fields=changed_info_fields,
            db=db,
            current_user=current_user,
        )

    user.updated_at = datetime.utcnow()
    await db.flush()
    await db.commit()
    await db.refresh(user)
    return await _build_user_item(user, db)


@router.put(
    "/users/{user_id}/roles",
    response_model=UserDetailResponse,
)
@RequiresPermissions("users:update:write")
async def update_user_roles(
    user_id: str,
    body: UpdateUserRolesRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if _is_admin(user):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员角色不可修改",
        )

    if not _is_manager_or_admin(current_user):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅管理员可修改用户角色分配",
        )

    violations = await validate_user_role_assignments(user.id, set(body.role_ids), db)
    if violations:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="; ".join(violations),
        )

    await _assert_not_assigning_super_admin(set(body.role_ids), user_id, db)

    prev_role_result = await db.execute(
        select(RBACUserRoleAssignment.role_id).where(RBACUserRoleAssignment.user_id == user_id)
    )
    prev_role_ids = {row[0] for row in prev_role_result.all()}

    await db.execute(
        delete(RBACUserRoleAssignment).where(RBACUserRoleAssignment.user_id == user_id)
    )
    for role_id in body.role_ids:
        db.add(
            RBACUserRoleAssignment(
                user_id=user_id,
                role_id=role_id,
                grant_type="direct",
                created_at=datetime.utcnow(),
            )
        )

    await _notify_user_role_change(
        user_id=user.id,
        user_nickname=user.nickname,
        role_ids=body.role_ids,
        prev_role_ids=prev_role_ids,
        db=db,
        current_user=current_user,
    )

    await db.commit()
    await db.refresh(user)
    return await _build_user_item(user, db)


@router.put(
    "/users/{user_id}/password",
)
@RequiresPermissions("users:change_password:write")
async def change_user_password(
    user_id: str,
    body: UpdateUserPasswordRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if not _is_manager_or_admin(current_user) and current_user.id != user_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="仅管理员或用户本人可修改密码",
        )

    valid, msg = validate_password_format(body.new_password)
    if not valid:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=msg,
        )

    is_admin_reset = _is_manager_or_admin(current_user)

    if not is_admin_reset:
        default_pwd = f"{user.username}123"
        current_is_default = verify_password(default_pwd, user.hashed_password)

        if not current_is_default:
            if not body.old_password:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="非默认密码，请输入旧密码进行验证",
                )
            if not verify_password(body.old_password, user.hashed_password):
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="旧密码验证失败",
                )

    user.hashed_password = hash_password(body.new_password)
    await db.flush()
    await db.commit()
    return {"message": "密码修改成功"}


@router.get(
    "/users/{user_id}/permissions",
    response_model=UserPermissionsResponse,
)
async def get_user_permissions(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    is_self = user_id == current_user.id
    if not is_self:
        if not await _can_access_user_target(current_user, user, db, allow_self=False):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无法访问该用户：非管理角色只能操作同角色用户",
            )
        if not await has_permission_direct(
            current_user.id, "users:custom_permissions:read", "read", db
        ):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无权查看其他用户的自定义权限",
            )

    # 交集模型：有效权限 = 角色权限(含继承、派生) ∩ 自定义权限(granted=True)
    # 超级管理员不受覆盖约束，无覆盖时直接使用角色权限
    effective_permissions = await get_user_effective_flat_permissions(user_id, db)

    all_role_ids = await _role_closure_role_ids(user_id, db)
    if await _closure_has_super_admin(all_role_ids, db):
        perm_keys_result = await db.execute(
            select(RBACPermission.key).where(RBACPermission.is_active.is_(True))
        )
        role_permission_keys = {row[0] for row in perm_keys_result.all()}
    else:
        role_perm_result = await db.execute(
            select(RBACPermission.key)
            .join(RBACRolePermission, RBACRolePermission.permission_id == RBACPermission.id)
            .where(
                RBACRolePermission.role_id.in_(all_role_ids),
                RBACPermission.is_active.is_(True),
            )
        )
        role_permission_keys = {row[0] for row in role_perm_result.all()}

    perm_result = await db.execute(
        select(RBACPermission, RBACResource)
        .join(RBACResource, RBACPermission.resource_id == RBACResource.id)
        .where(RBACPermission.key.in_(role_permission_keys))
        .order_by(RBACPermission.created_at.asc())
    )
    available_permissions: list[AvailablePermission] = []
    for permission, resource in perm_result.all():
        available_permissions.append(
            AvailablePermission(
                id=permission.id,
                key=permission.key,
                operation=permission.operation,
                is_active=permission.is_active,
                created_at=permission.created_at,
                resource=_PermResourceRef(
                    id=resource.id,
                    key=resource.key,
                    name=resource.name,
                    description=resource.description,
                ),
            )
        )

    return UserPermissionsResponse(
        effective_permissions=effective_permissions,
        available_permissions=available_permissions,
    )


class PermissionOverrideEntry(BaseModel):
    permission_key: str
    granted: bool


class UpdateUserPermissionOverridesRequest(BaseModel):
    overrides: list[PermissionOverrideEntry]


@router.get(
    "/users/{user_id}/permission-overrides",
    response_model=list[PermissionOverrideEntry],
)
async def get_user_permission_overrides(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """获取用户的自定义权限覆盖列表。"""
    is_self = user_id == current_user.id
    if not is_self:
        result = await db.execute(select(User).where(User.id == user_id))
        target_user = result.scalar_one_or_none()
        if target_user is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="用户不存在",
            )
        if not await _can_access_user_target(current_user, target_user, db, allow_self=False):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无法访问该用户：非管理角色只能操作同角色用户",
            )
        if not await has_permission_direct(
            current_user.id, "users:custom_permissions:read", "read", db
        ):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="无权查看其他用户的自定义权限覆盖",
            )
    result = await db.execute(
        select(RBACUserPermissionOverride)
        .where(
            RBACUserPermissionOverride.user_id == user_id,
        )
        .order_by(RBACUserPermissionOverride.permission_key)
    )
    overrides = result.scalars().all()
    return [
        PermissionOverrideEntry(
            permission_key=o.permission_key,
            granted=o.granted,
        )
        for o in overrides
    ]


@router.put(
    "/users/{user_id}/permission-overrides",
    response_model=list[PermissionOverrideEntry],
)
@RequiresPermissions("users:custom_permissions:write")
async def update_user_permission_overrides(
    user_id: str,
    body: UpdateUserPermissionOverridesRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """更新用户的自定义权限覆盖（全量替换）。"""
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if _is_admin(user):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号不可修改权限覆盖",
        )

    if not await _can_access_user_target(current_user, user, db, allow_self=False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无法访问该用户：非管理角色只能操作同角色用户",
        )

    # Delete all existing overrides for this user.
    await db.execute(
        delete(RBACUserPermissionOverride).where(
            RBACUserPermissionOverride.user_id == user_id,
        )
    )

    # Insert new overrides.
    for entry in body.overrides:
        db.add(
            RBACUserPermissionOverride(
                user_id=user_id,
                permission_key=entry.permission_key,
                granted=entry.granted,
            )
        )

    await db.commit()

    # Return the saved overrides.
    result = await db.execute(
        select(RBACUserPermissionOverride)
        .where(
            RBACUserPermissionOverride.user_id == user_id,
        )
        .order_by(RBACUserPermissionOverride.permission_key)
    )
    overrides = result.scalars().all()
    return [
        PermissionOverrideEntry(
            permission_key=o.permission_key,
            granted=o.granted,
        )
        for o in overrides
    ]


@router.delete(
    "/users/{user_id}/permission-overrides",
    status_code=status.HTTP_204_NO_CONTENT,
)
@RequiresPermissions("users:custom_permissions:write")
async def reset_user_permission_overrides(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    """重置用户自定义权限覆盖（移除该用户全部 override 记录，回到角色默认权限）。"""
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if _is_admin(user):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号不可修改权限覆盖",
        )

    if not await _can_access_user_target(current_user, user, db, allow_self=False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无法访问该用户：非管理角色只能操作同角色用户",
        )

    await db.execute(
        delete(RBACUserPermissionOverride).where(
            RBACUserPermissionOverride.user_id == user_id,
        )
    )
    await db.commit()
    return None


@router.delete(
    "/users/{user_id}",
    status_code=status.HTTP_204_NO_CONTENT,
)
@RequiresPermissions("users:delete:write")
async def delete_user(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )
    if _is_admin(user):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号不可移除",
        )
    if user.id == current_user.id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="不能删除自己的账号",
        )
    if not await _can_access_user_target(current_user, user, db, allow_self=False):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无法访问该用户：非管理角色只能操作同角色用户",
        )

    await db.execute(
        delete(RBACUserRoleAssignment).where(RBACUserRoleAssignment.user_id == user_id)
    )
    await db.delete(user)
    await db.commit()
    return None


@router.get(
    "/users/{user_id}/password-status",
)
@RequiresPermissions("users:read")
async def get_password_status(
    user_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="用户不存在",
        )

    if not _is_manager_or_admin(current_user) and user.id != current_user.id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无权查看",
        )

    default_pwd = f"{user.username}123"
    is_default = verify_password(default_pwd, user.hashed_password)
    return {"is_default_password": is_default}
