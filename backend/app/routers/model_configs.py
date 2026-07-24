from datetime import datetime
from typing import List, Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models.model_config import ModelConfig
from app.models.user import User

router = APIRouter(prefix="/api/model-configs", tags=["模型配置"])


class ModelConfigCreate(BaseModel):
    id: Optional[str] = Field(default=None, max_length=64)
    name: str = Field(..., max_length=100)
    display_name: str = Field(..., max_length=100)
    provider: str = Field(..., max_length=20)
    mode: str = Field(..., max_length=20)
    api_format: str = Field(default="openai_chat", max_length=50)
    api_key: Optional[str] = Field(default=None, max_length=500)
    base_url: Optional[str] = Field(default=None, max_length=500)
    full_url: bool = False
    model: str = Field(..., max_length=2000)
    multimodal: bool = False
    model_series: str = Field(default="default", max_length=50)
    context_input: int = Field(default=128000)
    context_output: int = Field(default=4096)
    tool_call_rounds: int = Field(default=200)
    enabled: bool = False
    monthly_quota: int = Field(default=1000000)
    used_tokens: int = Field(default=0)


class ModelConfigUpdate(BaseModel):
    name: Optional[str] = Field(default=None, max_length=100)
    display_name: Optional[str] = Field(default=None, max_length=100)
    provider: Optional[str] = Field(default=None, max_length=20)
    mode: Optional[str] = Field(default=None, max_length=20)
    api_format: Optional[str] = Field(default=None, max_length=50)
    api_key: Optional[str] = Field(default=None, max_length=500)
    base_url: Optional[str] = Field(default=None, max_length=500)
    full_url: Optional[bool] = None
    model: Optional[str] = Field(default=None, max_length=2000)
    multimodal: Optional[bool] = None
    model_series: Optional[str] = Field(default=None, max_length=50)
    context_input: Optional[int] = None
    context_output: Optional[int] = None
    tool_call_rounds: Optional[int] = None
    enabled: Optional[bool] = None
    monthly_quota: Optional[int] = None
    used_tokens: Optional[int] = None


class ModelConfigResponse(BaseModel):
    id: str
    name: str
    display_name: str
    provider: str
    mode: str
    api_format: str
    api_key: Optional[str]
    base_url: Optional[str]
    full_url: bool
    model: str
    multimodal: bool
    model_series: str
    context_input: int
    context_output: int
    tool_call_rounds: int
    enabled: bool
    monthly_quota: int
    used_tokens: int
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class ModelConfigListResponse(BaseModel):
    data: List[ModelConfigResponse]


@router.get("", response_model=ModelConfigListResponse)
async def list_model_configs(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(ModelConfig).order_by(ModelConfig.created_at))
    configs = result.scalars().all()
    return ModelConfigListResponse(data=[ModelConfigResponse.model_validate(c) for c in configs])


@router.post("", response_model=ModelConfigResponse, status_code=status.HTTP_201_CREATED)
async def create_model_config(
    request: ModelConfigCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("model_config:create")),
):
    config_id = request.id or f"plan-{int(__import__('time').time() * 1000)}"
    existing = await db.execute(select(ModelConfig).where(ModelConfig.id == config_id))
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST, detail="配置 ID 已存在"
        )
    config = ModelConfig(id=config_id, **request.model_dump(exclude={"id"}))
    db.add(config)
    await db.commit()
    await db.refresh(config)
    return ModelConfigResponse.model_validate(config)


@router.put("/{config_id}", response_model=ModelConfigResponse)
async def update_model_config(
    config_id: str,
    request: ModelConfigUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("model_config:update")),
):
    result = await db.execute(select(ModelConfig).where(ModelConfig.id == config_id))
    config = result.scalar_one_or_none()
    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND, detail="配置不存在"
        )
    for key, value in request.model_dump(exclude_unset=True).items():
        setattr(config, key, value)
    await db.commit()
    await db.refresh(config)
    return ModelConfigResponse.model_validate(config)


@router.delete("/{config_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_model_config(
    config_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("model_config:delete")),
):
    result = await db.execute(select(ModelConfig).where(ModelConfig.id == config_id))
    config = result.scalar_one_or_none()
    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND, detail="配置不存在"
        )
    await db.delete(config)
    await db.commit()
    return None
