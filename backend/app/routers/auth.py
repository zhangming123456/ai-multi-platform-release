from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user
from app.core.security import hash_password, verify_password
from app.database import get_db
from app.models.user import User
from app.schemas.auth import (
    LoginRequest,
    LoginResponse,
    PasswordChange,
    ProfileUpdate,
    UserCreate,
    UserInfo,
    UserInfoWithPermissions,
)
from app.services.auth_service import (
    authenticate_user,
    create_user,
    generate_token,
    get_user_by_email,
    get_user_by_username,
)

router = APIRouter(prefix="/api/auth", tags=["认证"])


@router.post("/login", response_model=LoginResponse)
async def login(request: LoginRequest, db: AsyncSession = Depends(get_db)):
    user = await authenticate_user(db, request.username, request.password)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="用户名或密码错误",
        )
    token = generate_token(user)
    return LoginResponse(
        access_token=token,
        user=UserInfo.model_validate(user),
    )


@router.post("/register", response_model=UserInfo, status_code=status.HTTP_201_CREATED)
async def register(request: UserCreate, db: AsyncSession = Depends(get_db)):
    if request.role == "admin":
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="超级管理员账号唯一，不可通过注册创建",
        )
    existing = await get_user_by_username(db, request.username)
    if existing:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="该用户名已存在",
        )
    if request.email:
        existing_email = await get_user_by_email(db, request.email)
        if existing_email:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="该邮箱已被使用",
            )
    user = await create_user(db, request)
    return UserInfo.model_validate(user)


@router.get("/me", response_model=UserInfoWithPermissions)
async def get_me(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    from app.routers.permissions import get_user_permissions

    permissions = await get_user_permissions(current_user, db)
    user_info = UserInfo.model_validate(current_user)
    permissions_info = {k: v.model_dump() for k, v in permissions.items()}
    return UserInfoWithPermissions(**user_info.model_dump(), permissions=permissions_info)


@router.put("/profile", response_model=UserInfo)
async def update_profile(
    request: ProfileUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    update_data = request.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        setattr(current_user, field, value)
    await db.flush()
    await db.refresh(current_user)
    return UserInfo.model_validate(current_user)


@router.put("/password")
async def change_password(
    request: PasswordChange,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    if not verify_password(request.old_password, current_user.hashed_password):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="旧密码错误",
        )
    current_user.hashed_password = hash_password(request.new_password)
    await db.flush()
    return {"message": "密码修改成功"}
