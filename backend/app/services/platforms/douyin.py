from typing import Any

from app.services.platforms.base import PlatformAdapter, PublishResult, StatusResult


class DouyinAdapter(PlatformAdapter):
    platform_name = "douyin"

    async def publish(self, content: dict, account: Any) -> PublishResult:
        raise NotImplementedError("抖音发布功能待实现")

    async def check_status(self, account: Any) -> StatusResult:
        raise NotImplementedError("抖音状态检查功能待实现")
