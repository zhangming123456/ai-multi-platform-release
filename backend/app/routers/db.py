from __future__ import annotations

import re
from typing import Any, Optional

from fastapi import APIRouter, Depends, HTTPException, Query, status
from pydantic import BaseModel
from sqlalchemy import func, select, text

from app.core.deps import get_admin_user
from app.database import async_session_factory
from app.models.sql_history import SqlHistory
from app.models.user import User

router = APIRouter(prefix="/api/db", tags=["数据库管理"])


class SqlRequest(BaseModel):
    sql: str
    page: int = 1
    page_size: int = 20


class SqlResponse(BaseModel):
    success: bool
    message: str
    columns: list[str] = []
    rows: list[list] = []
    row_count: int = 0
    total_count: int = 0
    page_size: int = 0
    is_query: bool = False
    need_review: bool = False
    change_type: Optional[str] = None


class SqlHistoryItem(BaseModel):
    id: str
    username: str
    sql_text: str
    is_success: bool
    message: str
    created_at: str

    model_config = {"from_attributes": True}


class SqlHistoryResponse(BaseModel):
    items: list[SqlHistoryItem]
    total: int
    page: int
    page_size: int


_is_select_re = re.compile(r"^\s*(SELECT|PRAGMA|EXPLAIN)", re.IGNORECASE)
_is_update_re = re.compile(r"^\s*UPDATE\b", re.IGNORECASE)
_is_delete_re = re.compile(r"^\s*DELETE\b", re.IGNORECASE)
_is_paginated_re = re.compile(r"^\s*SELECT\b", re.IGNORECASE)
_limit_re = re.compile(r"\s+LIMIT\s+(\d+)\s*", re.IGNORECASE)
_offset_re = re.compile(r"\s+OFFSET\s+(\d+)\s*", re.IGNORECASE)
# match LIMIT ... OFFSET ... or LIMIT ...  at the end of a query (after ; is stripped)
_limit_offset_re = re.compile(
    r"\s+LIMIT\s+(?P<limit>\d+)(\s+OFFSET\s+(?P<offset>\d+))?\s*;?$",
    re.IGNORECASE,
)


def _strip_sql_limit_offset(sql: str) -> tuple[str, int | None, int | None]:
    """返回 (去除 LIMIT/OFFSET 后的 SQL, limit值或None, offset值或None)"""
    m = _limit_offset_re.search(sql)
    if m:
        limit = int(m.group("limit"))
        offset = int(m.group("offset")) if m.group("offset") else 0
        stripped = sql[: m.start()] + sql[m.end() :].strip()
        if stripped.endswith(";"):
            stripped = stripped[:-1].strip()
        return stripped, limit, offset
    return sql, None, None


_time_column_names = {"created_at", "updated_at", "time", "date", "datetime", "timestamp",
                       "create_time", "update_time", "created_time", "modified_at",
                       "last_check_at", "token_expires_at", "publish_time", "generated_at"}


def _find_time_column(columns: list[str]) -> int | None:
    for i, col in enumerate(columns):
        if col.lower() in _time_column_names:
            return i
    return None


def _sort_key(value: Any) -> str:
    if value is None:
        return ""
    return str(value)


@router.get("/history", response_model=SqlHistoryResponse)
async def list_history(
    page: int = Query(default=1, ge=1),
    page_size: int = Query(default=20, ge=1, le=200),
    admin: User = Depends(get_admin_user),
):
    async with async_session_factory() as session:
        count_result = await session.execute(
            select(func.count()).select_from(SqlHistory)
        )
        total = count_result.scalar() or 0

        offset = (page - 1) * page_size
        result = await session.execute(
            select(SqlHistory, User.username)
            .join(User, SqlHistory.user_id == User.id)
            .order_by(SqlHistory.created_at.desc())
            .offset(offset)
            .limit(page_size)
        )
        rows = result.all()
        items = [
            SqlHistoryItem(
                id=row.SqlHistory.id,
                username=row.username,
                sql_text=row.SqlHistory.sql_text,
                is_success=row.SqlHistory.is_success,
                message=row.SqlHistory.message,
                created_at=row.SqlHistory.created_at.isoformat(),
            )
            for row in rows
        ]

        return SqlHistoryResponse(
            items=items,
            total=total,
            page=page,
            page_size=page_size,
        )


@router.post("/execute", response_model=SqlResponse)
async def execute_sql(
    request: SqlRequest,
    admin: User = Depends(get_admin_user),
):
    sql = request.sql.strip()
    if not sql:
        raise HTTPException(status_code=400, detail="SQL 命令不能为空")

    dangerous_keywords = ["DROP TABLE", "DROP DATABASE", "TRUNCATE", "ALTER TABLE", "ATTACH", "DETACH", "VACUUM", "REINDEX"]
    upper_sql = sql.upper()
    for keyword in dangerous_keywords:
        if keyword in upper_sql:
            return SqlResponse(
                success=False,
                message=f"禁止执行危险命令: {keyword}",
            )

    if _is_delete_re.match(sql):
        return SqlResponse(
            success=False,
            need_review=True,
            change_type="delete",
            message="DELETE 删除操作需要提交审核，由至少 2 名审核员通过后自动执行",
        )
    if _is_update_re.match(sql):
        return SqlResponse(
            success=False,
            need_review=True,
            change_type="update",
            message="UPDATE 修改操作需要提交审核，由至少 2 名审核员通过后自动执行",
        )

    executed_sql = sql

    async with async_session_factory() as session:
        try:
            is_query = bool(_is_select_re.match(sql))
            can_paginate = bool(_is_paginated_re.match(sql))

            total_count = 0
            if can_paginate:
                base_sql, sql_limit, sql_offset = _strip_sql_limit_offset(sql.rstrip(';').strip())

                count_sql = f"SELECT COUNT(*) AS cnt FROM ({base_sql}) AS _sub"
                count_result = await session.execute(text(count_sql))
                total_count = count_result.scalar() or 0

                page = max(1, request.page)
                page_size = max(1, min(request.page_size, 200))
                offset = (page - 1) * page_size
                paged_sql = f"SELECT * FROM ({base_sql}) AS _sub LIMIT {page_size} OFFSET {offset}"
                executed_sql = paged_sql
                result = await session.execute(text(paged_sql))
            else:
                page_size = max(1, min(request.page_size, 200))
                result = await session.execute(text(sql))

            await session.commit()

            if is_query and result.returns_rows:
                columns = list(result.keys())
                rows = [list(row) for row in result.fetchall()]

                time_col_idx = _find_time_column(columns)
                if time_col_idx is not None:
                    rows.sort(key=lambda r: _sort_key(r[time_col_idx]), reverse=True)

                msg = f"查询成功，共 {total_count} 行，当前第 {total_count > 0 and max(1, request.page) or 1} 页"
                response = SqlResponse(
                    success=True,
                    message=msg,
                    columns=columns,
                    rows=rows,
                    row_count=len(rows),
                    total_count=total_count,
                    page_size=page_size,
                    is_query=True,
                )
            else:
                msg = "命令执行成功"
                response = SqlResponse(
                    success=True,
                    message=msg,
                    is_query=False,
                )

            history = SqlHistory(
                user_id=admin.id,
                sql_text=executed_sql,
                is_success=True,
                message=msg,
            )
            session.add(history)
            await session.commit()

            return response
        except Exception as e:
            await session.rollback()
            msg = f"执行失败: {str(e)}"

            history = SqlHistory(
                user_id=admin.id,
                sql_text=executed_sql,
                is_success=False,
                message=msg,
            )
            session.add(history)
            await session.commit()

            return SqlResponse(
                success=False,
                message=msg,
            )
