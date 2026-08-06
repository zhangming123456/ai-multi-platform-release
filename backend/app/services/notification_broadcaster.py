import asyncio
import contextlib
from collections import defaultdict


class NotificationBroadcaster:
    def __init__(self) -> None:
        self._queues: dict[str, list[asyncio.Queue[dict]]] = defaultdict(list)

    def subscribe(self, user_id: str) -> asyncio.Queue[dict]:
        q: asyncio.Queue[dict] = asyncio.Queue(maxsize=128)
        self._queues[user_id].append(q)
        return q

    def unsubscribe(self, user_id: str, queue: asyncio.Queue[dict]) -> None:
        queues = self._queues.get(user_id)
        if queues:
            with contextlib.suppress(ValueError):
                queues.remove(queue)
            if not queues:
                del self._queues[user_id]

    async def broadcast(self, user_id: str, notification: dict) -> None:
        queues = self._queues.get(user_id)
        if not queues:
            return
        dead: list[asyncio.Queue[dict]] = []
        for q in queues:
            try:
                q.put_nowait(notification)
            except asyncio.QueueFull:
                with contextlib.suppress(asyncio.QueueEmpty):
                    q.get_nowait()
                try:
                    q.put_nowait(notification)
                except asyncio.QueueFull:
                    dead.append(q)
        for q in dead:
            queues.remove(q)
        if not queues:
            del self._queues[user_id]


broadcaster = NotificationBroadcaster()
