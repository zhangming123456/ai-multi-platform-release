from datetime import datetime

from sqlalchemy import DateTime, Integer, String, Boolean
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base
from typing import Optional


class ModelConfig(Base):
    __tablename__ = "model_configs"

    id: Mapped[str] = mapped_column(String(64), primary_key=True)
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    display_name: Mapped[str] = mapped_column(String(100), nullable=False)
    provider: Mapped[str] = mapped_column(String(20), nullable=False)
    mode: Mapped[str] = mapped_column(String(20), nullable=False)
    api_format: Mapped[str] = mapped_column(String(50), nullable=False, default="openai_chat")
    api_key: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    base_url: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    full_url: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)
    model: Mapped[str] = mapped_column(String(2000), nullable=False)
    multimodal: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)
    model_series: Mapped[str] = mapped_column(String(50), nullable=False, default="default")
    context_input: Mapped[int] = mapped_column(Integer, default=128000, nullable=False)
    context_output: Mapped[int] = mapped_column(Integer, default=4096, nullable=False)
    tool_call_rounds: Mapped[int] = mapped_column(Integer, default=200, nullable=False)
    enabled: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)
    monthly_quota: Mapped[int] = mapped_column(Integer, default=1000000, nullable=False)
    used_tokens: Mapped[int] = mapped_column(Integer, default=0, nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, nullable=False
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.now, onupdate=datetime.now, nullable=False
    )
