from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy import select

from app.config import settings
from app.core.privacy_middleware import PrivacyMaskMiddleware
from app.core.security import hash_password
from app.database import Base, async_session_factory, engine
from app.models import User, UserRole
from app.routers import accounts, auth, contents, dashboard, db, db_changes, model_configs, models, notifications, permissions, publish, reviews, roles, templates, user_creation_reviews, users


@asynccontextmanager
async def lifespan(app: FastAPI):
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
        await conn.run_sync(Base.metadata.create_all)

    async with async_session_factory() as session:
        session.add(User(
            id="1",
            username="admin",
            email="admin@admin.com",
            hashed_password=hash_password("admin123"),
            nickname="超级管理员",
            role=UserRole.admin,
        ))
        session.add(User(
            username="manager",
            email="manager@example.com",
            hashed_password=hash_password("manager123"),
            nickname="管理员",
            role=UserRole.manager,
        ))
        session.add(User(
            username="operator",
            email="operator@example.com",
            hashed_password=hash_password("operator123"),
            nickname="运营小二",
            role=UserRole.operator,
        ))
        session.add(User(
            username="reviewer",
            email="reviewer@example.com",
            hashed_password=hash_password("reviewer123"),
            nickname="审核专员",
            role=UserRole.reviewer,
        ))
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
app.include_router(users.router)
app.include_router(db.router)
app.include_router(db_changes.router)
app.include_router(permissions.router)
app.include_router(roles.router)
app.include_router(user_creation_reviews.router)


@app.get("/")
async def root():
    return {
        "message": "欢迎使用多平台矩阵管理系统 API",
        "docs": "/docs",
        "version": "1.0.0",
    }
