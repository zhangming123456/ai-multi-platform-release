from typing import Any

from app.services.platforms.base import PlatformAdapter, PublishResult, StatusResult


class WechatMPAdapter(PlatformAdapter):
    platform_name = "wechat_mp"

    async def publish(self, content: dict, account: Any) -> PublishResult:
        raise NotImplementedError("微信公众号发布功能待实现")

    async def check_status(self, account: Any) -> StatusResult:
        raise NotImplementedError("微信公众号状态检查功能待实现")
