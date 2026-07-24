from __future__ import annotations

from datetime import datetime

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.deps import get_current_user, require_permission
from app.database import get_db
from app.models.account import Account, AccountStatus
from app.models.user import User
from app.schemas.account import AccountCreate, AccountResponse, AccountStatusResponse, AccountUpdate
from app.services.platforms import get_platform_adapter
from typing import Optional

router = APIRouter(prefix="/api/accounts", tags=["账号管理"])


@router.get("/", response_model=list[AccountResponse])
async def list_accounts(
    platform: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    query = select(Account).where(Account.user_id == current_user.id)
    if platform:
        query = query.where(Account.platform == platform)
    query = query.order_by(Account.created_at.desc())
    result = await db.execute(query)
    return result.scalars().all()


@router.post("/", response_model=AccountResponse, status_code=status.HTTP_201_CREATED)
async def create_account(
    request: AccountCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("account:create")),
):
    account = Account(
        user_id=current_user.id,
        platform=request.platform,
        nickname=request.nickname,
        avatar_url=request.avatar_url,
        cookie_data=request.cookie_data,
        access_token=request.access_token,
        token_expires_at=request.token_expires_at,
    )
    db.add(account)
    await db.flush()
    await db.refresh(account)
    return account


@router.get("/{account_id}", response_model=AccountResponse)
async def get_account(
    account_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
):
    result = await db.execute(
        select(Account).where(Account.id == account_id, Account.user_id == current_user.id)
    )
    account = result.scalar_one_or_none()
    if not account:
        raise HTTPException(status_code=404, detail="账号不存在")
    return account


@router.put("/{account_id}", response_model=AccountResponse)
async def update_account(
    account_id: int,
    request: AccountUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("account:update")),
):
    result = await db.execute(
        select(Account).where(Account.id == account_id, Account.user_id == current_user.id)
    )
    account = result.scalar_one_or_none()
    if not account:
        raise HTTPException(status_code=404, detail="账号不存在")

    update_data = request.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        setattr(account, field, value)

    await db.flush()
    await db.refresh(account)
    return account


@router.delete("/{account_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_account(
    account_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("account:delete")),
):
    result = await db.execute(
        select(Account).where(Account.id == account_id, Account.user_id == current_user.id)
    )
    account = result.scalar_one_or_none()
    if not account:
        raise HTTPException(status_code=404, detail="账号不存在")
    await db.delete(account)
    await db.flush()


@router.post("/{account_id}/check", response_model=AccountStatusResponse)
async def check_account_status(
    account_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("account:check")),
):
    result = await db.execute(
        select(Account).where(Account.id == account_id, Account.user_id == current_user.id)
    )
    account = result.scalar_one_or_none()
    if not account:
        raise HTTPException(status_code=404, detail="账号不存在")

    adapter = get_platform_adapter(account.platform)
    if adapter:
        try:
            status_result = await adapter.check_status(account)
            account.status = (
                AccountStatus.active if status_result.is_active else AccountStatus.error
            )
            account.error_message = status_result.error_message
        except NotImplementedError:
            account.status = AccountStatus.active
            account.error_message = None
    else:
        account.status = AccountStatus.active

    account.last_check_at = datetime.utcnow()
    await db.flush()
    await db.refresh(account)

    return AccountStatusResponse(
        id=account.id,
        status=account.status.value,
        last_check_at=account.last_check_at,
        error_message=account.error_message,
    )
