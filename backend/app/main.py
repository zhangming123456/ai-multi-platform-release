from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy import select

from app.config import settings
from app.core.privacy_middleware import PrivacyMaskMiddleware
from app.core.security import hash_password
from app.database import Base, async_session_factory, engine
from app.models import User, UserRole
from app.routers import (
    accounts,
    auth,
    contents,
    dashboard,
    db,
    db_changes,
    model_configs,
    models,
    notification_dict,
    notifications,
    publish,
    rbac_constraints,
    rbac_permissions,
    rbac_roles,
    rbac_users,
    reviews,
    templates,
    user_creation_reviews,
)
from app.services.rbac_init_service import init_rbac_system


@asynccontextmanager
async def lifespan(app: FastAPI):
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    async with async_session_factory() as session:
        result = await session.execute(select(User.id))
        existing_ids = {row[0] for row in result.all()}

        if "1" not in existing_ids:
            session.add(
                User(
                    id="1",
                    username="admin",
                    email="admin@admin.com",
                    hashed_password=hash_password("admin123"),
                    nickname="超级管理员",
                    role=UserRole.admin,
                )
            )

        result = await session.execute(select(User.username))
        existing_usernames = {row[0] for row in result.all()}

        seed_users = [
            ("manager", "manager@example.com", "manager123", "管理员", UserRole.manager),
            ("operator", "operator@example.com", "operator123", "运营小二", UserRole.operator),
            ("reviewer", "reviewer@example.com", "reviewer123", "审核专员", UserRole.reviewer),
        ]
        for username, email, password, nickname, role in seed_users:
            if username not in existing_usernames:
                session.add(
                    User(
                        username=username,
                        email=email,
                        hashed_password=hash_password(password),
                        nickname=nickname,
                        role=role,
                    )
                )
        await session.commit()

    async with async_session_factory() as session:
        await init_rbac_system(session)
        await _seed_notification_dict(session)
        await session.commit()

    yield


app = FastAPI(
    title="多平台矩阵管理系统 API",
    description="支持小红书、抖音、微信视频号、微信公众号等平台的账号管理、AI内容生成和多平台发布",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins_list,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.add_middleware(PrivacyMaskMiddleware)

app.include_router(auth.router)
app.include_router(accounts.router)
app.include_router(contents.router)
app.include_router(dashboard.router)
app.include_router(model_configs.router)
app.include_router(models.router)
app.include_router(notifications.router)
app.include_router(publish.router)
app.include_router(reviews.router)
app.include_router(templates.router)
app.include_router(db.router)
app.include_router(db_changes.router)
app.include_router(user_creation_reviews.router)
app.include_router(rbac_users.router)
app.include_router(rbac_roles.router)
app.include_router(rbac_permissions.router)
app.include_router(rbac_constraints.router)
app.include_router(notification_dict.router)


async def _seed_notification_dict(session):
    from app.models.notification_dict import DictCategory, NotificationDict

    result = await session.execute(select(NotificationDict.id))
    if result.first():
        return

    seed_entries = [
        ("field", "user_fields", "nickname", "昵称", 1),
        ("field", "user_fields", "email", "邮箱", 2),
        ("field", "user_fields", "username", "用户名", 3),
        ("field", "user_fields", "phone", "手机号", 4),
        ("field", "user_fields", "status", "状态", 5),
        ("enum", "notification_type", "review_submit", "审核提交", 1),
        ("enum", "notification_type", "review_approved", "审核通过", 2),
        ("enum", "notification_type", "review_rejected", "审核驳回", 3),
        ("enum", "notification_type", "role_updated", "角色更新", 4),
        ("enum", "notification_type", "role_permissions_updated", "权限变更", 5),
        ("enum", "user_status", "active", "启用", 1),
        ("enum", "user_status", "inactive", "禁用", 2),
    ]
    for category, group_key, dict_key, dict_value, sort_order in seed_entries:
        session.add(
            NotificationDict(
                category=DictCategory(category),
                group_key=group_key,
                dict_key=dict_key,
                dict_value=dict_value,
                sort_order=sort_order,
            )
        )


@app.get("/")
async def root():
    return {
        "message": "欢迎使用多平台矩阵管理系统 API",
        "docs": "/docs",
        "version": "1.0.0",
    }
