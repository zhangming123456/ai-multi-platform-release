from typing import Any

from app.services.platforms.base import PlatformAdapter, PublishResult, StatusResult


class WechatVideoAdapter(PlatformAdapter):
    platform_name = "wechat_video"

    async def publish(self, content: dict, account: Any) -> PublishResult:
        raise NotImplementedError("微信视频号发布功能待实现")

    async def check_status(self, account: Any) -> StatusResult:
        raise NotImplementedError("微信视频号状态检查功能待实现")
