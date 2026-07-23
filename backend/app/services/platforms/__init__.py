from __future__ import annotations

from typing import Dict, Optional

from app.services.platforms.base import PlatformAdapter, PublishResult, StatusResult
from app.services.platforms.douyin import DouyinAdapter
from app.services.platforms.wechat import WechatMPAdapter
from app.services.platforms.wechat_video import WechatVideoAdapter
from app.services.platforms.xiaohongshu import XiaohongshuAdapter

PLATFORM_ADAPTERS: dict[str, PlatformAdapter] = {
    "wechat_mp": WechatMPAdapter(),
    "xiaohongshu": XiaohongshuAdapter(),
    "douyin": DouyinAdapter(),
    "wechat_video": WechatVideoAdapter(),
}


def get_platform_adapter(platform: str) -> Optional[PlatformAdapter]:
    return PLATFORM_ADAPTERS.get(platform)
