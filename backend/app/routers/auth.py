from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user
from app.database import get_db
from app.models.user import User
from app.schemas.auth import LoginRequest, LoginResponse, UserCreate, UserInfo
from app.services.auth_service import (
    authenticate_user,
    create_user,
    generate_token,
    get_user_by_email,
)

router = APIRouter(prefix="/api/auth", tags=["认证"])


@router.post("/login", response_model=LoginResponse)
async def login(request: LoginRequest, db: AsyncSession = Depends(get_db)):
    user = await authenticate_user(db, request.email, request.password)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="邮箱或密码错误",
        )
    token = generate_token(user)
    return LoginResponse(
        access_token=token,
        user=UserInfo.model_validate(user),
    )


@router.post("/register", response_model=UserInfo, status_code=status.HTTP_201_CREATED)
async def register(request: UserCreate, db: AsyncSession = Depends(get_db)):
    existing = await get_user_by_email(db, request.email)
    if existing:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="该邮箱已注册",
        )
    user = await create_user(db, request)
    return UserInfo.model_validate(user)


@router.get("/me", response_model=UserInfo)
async def get_me(current_user: User = Depends(get_current_user)):
    return UserInfo.model_validate(current_user)
