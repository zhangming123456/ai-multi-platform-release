from __future__ import annotations

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import create_access_token, hash_password, verify_password
from app.models.user import User
from app.schemas.auth import UserCreate
from typing import Optional


async def authenticate_user(db: AsyncSession, login_id: str, password: str) -> Optional[User]:
    result = await db.execute(
        select(User).where((User.username == login_id) | (User.email == login_id))
    )
    user = result.scalar_one_or_none()
    if user is None:
        return None
    if not verify_password(password, user.hashed_password):
        return None
    return user


async def create_user(db: AsyncSession, user_data: UserCreate) -> User:
    user = User(
        username=user_data.username,
        email=user_data.email,
        hashed_password=hash_password(user_data.password),
        nickname=user_data.nickname,
        role=user_data.role,
        avatar_url=user_data.avatar_url,
    )
    db.add(user)
    await db.flush()
    await db.refresh(user)
    return user


async def get_user_by_username(db: AsyncSession, username: str) -> Optional[User]:
    result = await db.execute(select(User).where(User.username == username))
    return result.scalar_one_or_none()


async def get_user_by_email(db: AsyncSession, email: str) -> Optional[User]:
    result = await db.execute(select(User).where(User.email == email))
    return result.scalar_one_or_none()


def generate_token(user: User) -> str:
    return create_access_token(data={"sub": str(user.id), "username": user.username, "role": user.role})
