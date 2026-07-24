from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from typing import Optional

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models.template import Template
from app.models.user import User
from app.schemas.template import TemplateCreate, TemplateResponse, TemplateUpdate

router = APIRouter(prefix="/api/templates", tags=["模板管理"])


@router.get("/", response_model=list[TemplateResponse])
async def list_templates(
    platform: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    query = select(Template)
    if platform:
        query = query.where(Template.platform == platform)
    query = query.order_by(Template.created_at.desc())
    result = await db.execute(query)
    return result.scalars().all()


@router.post("/", response_model=TemplateResponse, status_code=status.HTTP_201_CREATED)
async def create_template(
    request: TemplateCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("template:create")),
):
    template = Template(
        name=request.name,
        platform=request.platform,
        thumbnail_url=request.thumbnail_url,
        config=request.config,
    )
    db.add(template)
    await db.flush()
    await db.refresh(template)
    return template


@router.get("/{template_id}", response_model=TemplateResponse)
async def get_template(
    template_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(select(Template).where(Template.id == template_id))
    template = result.scalar_one_or_none()
    if not template:
        raise HTTPException(status_code=404, detail="模板不存在")
    return template


@router.put("/{template_id}", response_model=TemplateResponse)
async def update_template(
    template_id: int,
    request: TemplateUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("template:update")),
):
    result = await db.execute(select(Template).where(Template.id == template_id))
    template = result.scalar_one_or_none()
    if not template:
        raise HTTPException(status_code=404, detail="模板不存在")

    update_data = request.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        setattr(template, field, value)

    await db.flush()
    await db.refresh(template)
    return template


@router.delete("/{template_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_template(
    template_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("template:delete")),
):
    result = await db.execute(select(Template).where(Template.id == template_id))
    template = result.scalar_one_or_none()
    if not template:
        raise HTTPException(status_code=404, detail="模板不存在")
    await db.delete(template)
    await db.flush()
