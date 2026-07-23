from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Any, Optional


@dataclass
class PublishResult:
    success: bool
    platform_post_id: Optional[str] = None
    error_message: Optional[str] = None
    raw_response: Optional[dict[str, Any]] = None


@dataclass
class StatusResult:
    is_active: bool
    error_message: Optional[str] = None
    extra_data: Optional[dict[str, Any]] = None


class PlatformAdapter(ABC):
    platform_name: str = ""

    @abstractmethod
    async def publish(self, content: dict, account: Any) -> PublishResult:
        raise NotImplementedError

    @abstractmethod
    async def check_status(self, account: Any) -> StatusResult:
        raise NotImplementedError
