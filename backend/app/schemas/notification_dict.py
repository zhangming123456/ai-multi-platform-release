from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


class NotificationDictCreate(BaseModel):
    category: str = Field(..., description="field 或 enum")
    group_key: str = Field(..., max_length=100, description="分组 key")
    dict_key: str = Field(..., max_length=200, description="字典 key")
    dict_value: str = Field(..., max_length=200, description="字典值")
    is_active: bool = Field(default=True)
    is_private: bool = Field(default=False, description="是否隐私化处理，隐藏原始值")
    sort_order: int = Field(default=0)


class NotificationDictUpdate(BaseModel):
    category: Optional[str] = Field(default=None, max_length=20)
    group_key: Optional[str] = Field(default=None, max_length=100)
    dict_key: Optional[str] = Field(default=None, max_length=200)
    dict_value: Optional[str] = Field(default=None, max_length=200)
    is_active: Optional[bool] = None
    is_private: Optional[bool] = None
    sort_order: Optional[int] = None


class NotificationDictResponse(BaseModel):
    id: str
    category: str
    group_key: str
    dict_key: str
    dict_value: str
    is_active: bool
    is_private: bool
    sort_order: int
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class NotificationDictListResponse(BaseModel):
    items: list[NotificationDictResponse]
    total: int


class NotificationDictGroupResponse(BaseModel):
    group_key: str
    items: list[NotificationDictResponse]


class ResolvedField(BaseModel):
    key: str
    value: str
