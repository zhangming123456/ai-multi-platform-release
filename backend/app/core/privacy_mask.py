from __future__ import annotations

from typing import Any


def mask_api_key(value: str) -> str:
    if len(value) <= 7:
        return "****"
    return value[:3] + "****" + value[-4:]


def mask_email(value: str) -> str:
    if "@" not in value:
        return "****"
    username, domain = value.split("@", 1)
    if len(username) <= 1:
        return "*@" + domain
    return username[0] + "***@" + domain


SENSITIVE_FIELD_HANDLERS: dict[str, callable] = {
    "api_key": mask_api_key,
    "access_token": lambda v: "****",
    "cookie_data": lambda v: "****",
    "hashed_password": lambda v: "****",
    "password": lambda v: "****",
    "email": mask_email,
    "phone": lambda v: v[:3] + "****" + v[-4:] if len(v) >= 7 else "****",
}

FULLY_EXCLUDED_PREFIXES = [
    "/api/db/",
    "/docs",
    "/openapi.json",
    "/redoc",
]

AUTH_PATH_PREFIXES = [
    "/api/auth/",
]


def _should_exclude_path(path: str) -> bool:
    return any(path.startswith(p) for p in FULLY_EXCLUDED_PREFIXES)


def _is_auth_path(path: str) -> bool:
    return any(path.startswith(p) for p in AUTH_PATH_PREFIXES)


def mask_value(key: str, value: Any, path: str = "") -> Any:
    if key == "access_token" and _is_auth_path(path):
        return value

    if key in SENSITIVE_FIELD_HANDLERS and isinstance(value, str) and value:
        return SENSITIVE_FIELD_HANDLERS[key](value)

    if isinstance(value, dict):
        return _mask_dict(value, path)
    if isinstance(value, list):
        return [_mask_element(item, path) for item in value]

    return value


def _mask_dict(data: dict, path: str = "") -> dict:
    return {k: mask_value(k, v, path) for k, v in data.items()}


def _mask_element(item: Any, path: str = "") -> Any:
    if isinstance(item, dict):
        return _mask_dict(item, path)
    if isinstance(item, list):
        return [_mask_element(sub, path) for sub in item]
    return item


def mask_response_data(data: Any, path: str = "") -> Any:
    if _should_exclude_path(path):
        return data

    if isinstance(data, dict):
        return _mask_dict(data, path)
    if isinstance(data, list):
        return [_mask_element(item, path) for item in data]

    return data
