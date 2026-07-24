from __future__ import annotations

import json

from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
from starlette.responses import Response

from app.core.privacy_mask import mask_response_data, _should_exclude_path


def _clean_headers(headers: dict) -> dict:
    cleaned = dict(headers)
    cleaned.pop("content-length", None)
    cleaned.pop("transfer-encoding", None)
    return cleaned


class PrivacyMaskMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next: RequestResponseEndpoint) -> Response:
        response = await call_next(request)

        path = request.url.path
        if _should_exclude_path(path):
            return response

        content_type = response.headers.get("content-type", "")
        if "application/json" not in content_type:
            return response

        body = b""
        async for chunk in response.body_iterator:
            body += chunk

        try:
            data = json.loads(body)
            masked = mask_response_data(data, path)
            new_body = json.dumps(masked, ensure_ascii=False, separators=(",", ":")).encode("utf-8")

            return Response(
                content=new_body,
                status_code=response.status_code,
                headers=_clean_headers(dict(response.headers)),
                media_type="application/json",
            )
        except (json.JSONDecodeError, UnicodeDecodeError):
            return Response(
                content=body,
                status_code=response.status_code,
                headers=_clean_headers(dict(response.headers)),
                media_type=content_type,
            )
