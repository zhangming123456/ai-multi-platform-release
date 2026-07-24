from __future__ import annotations

import re
from datetime import datetime, timedelta

import bcrypt
from jose import JWTError, jwt

from app.config import settings
from typing import Optional


def hash_password(password: str) -> str:
    return bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")


def verify_password(plain_password: str, hashed_password: str) -> bool:
    return bcrypt.checkpw(plain_password.encode("utf-8"), hashed_password.encode("utf-8"))


PASSWORD_PATTERN = re.compile(r"^[a-zA-Z][a-zA-Z0-9._@$]*$")


def validate_password_format(password: str) -> tuple[bool, str]:
    if not password:
        return False, "密码不能为空"
    if len(password) < 6:
        return False, "密码长度至少6位"
    if not password[0].isalpha():
        return False, "密码首字符必须为字母"
    if not PASSWORD_PATTERN.match(password):
        return False, "密码只能包含字母、数字和 . _ @ $ 字符"
    return True, ""


def is_default_password(username: str, password: str) -> bool:
    return password == f"{username}123"


def create_access_token(data: dict, expires_delta: Optional[timedelta] = None) -> str:
    to_encode = data.copy()
    expire = datetime.utcnow() + (
        expires_delta or timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    )
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, settings.SECRET_KEY, algorithm="HS256")


def decode_access_token(token: str) -> Optional[dict]:
    try:
        payload = jwt.decode(token, settings.SECRET_KEY, algorithms=["HS256"])
        return payload
    except JWTError:
        return None
