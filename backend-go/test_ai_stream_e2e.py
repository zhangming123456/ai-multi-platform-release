#!/usr/bin/env python3
"""E2E 验证：带完整 response_schema 的 AI 巡店流式分析。"""

import json
import requests

BASE = "http://localhost:8000"


def login():
    r = requests.post(
        f"{BASE}/api/auth/login", json={"username": "admin", "password": "admin123"}
    )
    r.raise_for_status()
    return r.json()["access_token"]


def main():
    token = login()
    headers = {"Authorization": f"Bearer {token}"}
    tpl = requests.get(f"{BASE}/api/inspection-templates/all", headers=headers).json()
    template_id = tpl[0]["id"] if tpl else ""
    print("template_id:", template_id)
    tpl_detail = requests.get(
        f"{BASE}/api/inspection-templates/{template_id}", headers=headers
    ).json()
    skills = [
        {
            "item_id": t["id"],
            "name": t["title"],
            "category": t["category"] or "",
            "standard": t["standard"] or "",
            "standard_image": t["standard_image"] or "",
            "max_score": t["max_score"],
            "score_options": t["score_options"] or [],
            "require_remark": t["require_remark"],
            "require_photo": t["require_photo"],
        }
        for t in (tpl_detail.get("items") or [])
    ]
    print("skills count:", len(skills))
    schema = {
        "type": "object",
        "properties": {
            "scores": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "item_id": {"type": "string"},
                        "score": {"type": "number"},
                        "comment": {"type": "string"},
                    },
                    "required": ["item_id", "score", "comment"],
                },
            },
            "issues": {"type": "string"},
            "suggestion": {"type": "string"},
            "summary": {"type": "string"},
            "high_risk_problems": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "item_id": {"type": "string"},
                        "item_name": {"type": "string"},
                        "level": {"type": "string"},
                        "desc": {"type": "string"},
                    },
                    "required": ["item_id", "item_name", "level", "desc"],
                },
            },
            "main_problems": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "item_id": {"type": "string"},
                        "item_name": {"type": "string"},
                        "level": {"type": "string"},
                        "desc": {"type": "string"},
                    },
                    "required": ["item_id", "item_name", "level", "desc"],
                },
            },
            "priority_suggest": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "title": {"type": "string"},
                        "desc": {"type": "string"},
                    },
                    "required": ["title", "desc"],
                },
            },
            "business_suggest": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "title": {"type": "string"},
                        "desc": {"type": "string"},
                    },
                    "required": ["title", "desc"],
                },
            },
        },
        "required": [
            "scores",
            "issues",
            "suggestion",
            "summary",
            "high_risk_problems",
            "main_problems",
            "priority_suggest",
            "business_suggest",
        ],
    }
    payload = {
        "store_id": "",
        "template_id": template_id,
        "plan_id": "plan-1786500925519-1",
        "model_id": "deepseek-v4-flash",
        "photos": [],
        "keywords": "门店地面有油污，灭火器过期",
        "skills": skills,
        "response_schema": schema,
    }
    r = requests.post(
        f"{BASE}/api/inspections/ai-analyze-stream",
        json=payload,
        headers=headers,
        stream=True,
    )
    r.raise_for_status()
    result_event = None
    count = 0
    current_event = {}
    for line in r.iter_lines():
        if not line:
            # blank line terminates one SSE event
            if current_event:
                count += 1
                evt = current_event.get("event")
                data = current_event.get("data", "")
                print("EVT:", evt, data[:200] if isinstance(data, str) else "")
                if evt == "result":
                    try:
                        result_event = json.loads(data)
                    except json.JSONDecodeError:
                        pass
                current_event = {}
            continue
        text = line.decode("utf-8")
        if text.startswith("event:"):
            current_event["event"] = text[6:].strip()
        elif text.startswith("data:"):
            current_event["data"] = text[5:].strip()
        elif text.strip() == "[DONE]":
            break
    print(f"events: {count}")
    if not result_event:
        print("FAIL: no result event")
        return
    print("result keys:", list(result_event.keys()))
    for k in [
        "summary",
        "high_risk_problems",
        "main_problems",
        "priority_suggest",
        "business_suggest",
    ]:
        v = result_event.get(k)
        print(f"{k}: {type(v).__name__} len={len(v) if isinstance(v, list) else 'n/a'}")
    hrp = result_event.get("high_risk_problems", [])
    mp = result_event.get("main_problems", [])
    print("sample high_risk item_id:", hrp[0].get("item_id") if hrp else None)
    print("sample main item_id:", mp[0].get("item_id") if mp else None)


if __name__ == "__main__":
    main()
