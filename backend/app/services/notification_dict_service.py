from sqlalchemy import select

from app.database import async_session_factory
from app.models.notification_dict import NotificationDict


async def resolve_field(group_key: str, key: str) -> str:
    async with async_session_factory() as session:
        result = await session.execute(
            select(NotificationDict.dict_value).where(
                NotificationDict.group_key == group_key,
                NotificationDict.dict_key == key,
                NotificationDict.is_active,
            )
        )
        row = result.first()
        if row and row[0]:
            return row[0]
    return key


async def is_field_private(group_key: str, key: str) -> bool:
    async with async_session_factory() as session:
        result = await session.execute(
            select(NotificationDict.is_private).where(
                NotificationDict.group_key == group_key,
                NotificationDict.dict_key == key,
                NotificationDict.is_active,
            )
        )
        row = result.first()
        if row and row[0]:
            return True
    return False


async def load_dict_map(group_key: str) -> dict[str, str]:
    async with async_session_factory() as session:
        result = await session.execute(
            select(NotificationDict.dict_key, NotificationDict.dict_value).where(
                NotificationDict.group_key == group_key,
                NotificationDict.is_active,
            )
        )
        return {row[0]: row[1] for row in result.all()}
