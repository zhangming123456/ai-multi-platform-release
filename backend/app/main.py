from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy import select

from app.config import settings
from app.core.security import hash_password
from app.database import Base, async_session_factory, engine
from app.models import User, UserRole
from app.routers import accounts, auth, contents, dashboard, model_configs, models, publish, templates


@asynccontextmanager
async def lifespan(app: FastAPI):
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    async with async_session_factory() as session:
        result = await session.execute(select(User).where(User.email == "admin@admin.com"))
        admin = result.scalar_one_or_none()
        if admin is None:
            admin = User(
                email="admin@admin.com",
                hashed_password=hash_password("admin123"),
                nickname="管理员",
                role=UserRole.admin,
            )
            session.add(admin)
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

app.include_router(auth.router)
app.include_router(accounts.router)
app.include_router(contents.router)
app.include_router(dashboard.router)
app.include_router(model_configs.router)
app.include_router(models.router)
app.include_router(publish.router)
app.include_router(templates.router)


@app.get("/")
async def root():
    return {
        "message": "欢迎使用多平台矩阵管理系统 API",
        "docs": "/docs",
        "version": "1.0.0",
    }
