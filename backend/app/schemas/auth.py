from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel
from typing import Optional


class LoginRequest(BaseModel):
    username: str
    password: str


class Token(BaseModel):
    access_token: str
    token_type: str = "bearer"


class PermissionAccessInfo(BaseModel):
    read: bool = True
    write: bool = True


class UserInfo(BaseModel):
    id: str
    username: str
    email: Optional[str] = None
    nickname: str
    role: str
    avatar_url: Optional[str] = None
    created_at: datetime

    model_config = {"from_attributes": True}


class UserInfoWithPermissions(UserInfo):
    permissions: dict[str, PermissionAccessInfo] = {}


class LoginResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
    user: UserInfo


class UserCreate(BaseModel):
    username: str
    password: str
    nickname: str
    email: Optional[str] = None
    role: str = "operator"
    avatar_url: Optional[str] = None


class UserUpdate(BaseModel):
    username: Optional[str] = None
    email: Optional[str] = None
    nickname: Optional[str] = None
    role: Optional[str] = None
    avatar_url: Optional[str] = None


class UserPasswordChange(BaseModel):
    new_password: str
    old_password: Optional[str] = None


class ProfileUpdate(BaseModel):
    nickname: Optional[str] = None
    avatar_url: Optional[str] = None


class PasswordChange(BaseModel):
    old_password: str
    new_password: str
