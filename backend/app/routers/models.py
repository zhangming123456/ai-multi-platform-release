from typing import List

import httpx
from fastapi import APIRouter, HTTPException, status
from pydantic import BaseModel

router = APIRouter(prefix="/api/models", tags=["模型"])


class FetchModelsRequest(BaseModel):
    base_url: str
    api_key: str


class FetchModelsResponse(BaseModel):
    models: List[str]


@router.post("/fetch", response_model=FetchModelsResponse)
async def fetch_models(request: FetchModelsRequest):
    url = request.base_url.rstrip("/") + "/models"
    headers = {
        "Authorization": f"Bearer {request.api_key}",
        "Content-Type": "application/json",
    }
    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.get(url, headers=headers)
            resp.raise_for_status()
            data = resp.json()
    except httpx.TimeoutException:
        raise HTTPException(
            status_code=status.HTTP_504_GATEWAY_TIMEOUT,
            detail="请求超时，请检查地址是否可达",
        )
    except httpx.HTTPStatusError as e:
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail=f"服务商返回错误：HTTP {e.response.status_code}",
        )
    except httpx.RequestError as e:
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail=f"请求失败：{str(e)}",
        )

    models = sorted([m["id"] for m in data.get("data", []) if "id" in m])
    return FetchModelsResponse(models=models)
