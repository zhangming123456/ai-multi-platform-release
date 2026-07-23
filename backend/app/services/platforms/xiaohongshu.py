from typing import Any

from app.services.platforms.base import PlatformAdapter, PublishResult, StatusResult


class XiaohongshuAdapter(PlatformAdapter):
    platform_name = "xiaohongshu"

    async def publish(self, content: dict, account: Any) -> PublishResult:
        raise NotImplementedError("小红书发布功能待实现")

    async def check_status(self, account: Any) -> StatusResult:
        raise NotImplementedError("小红书状态检查功能待实现")
