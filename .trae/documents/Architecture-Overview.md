# 架构设计与技术选型

## 1. 系统架构

```mermaid
flowchart TB
    subgraph Frontend["前端层 (Vue 3 + Vite)"]
        UI["Web UI 界面 (Arco Design Vue + TailwindCSS)"]
        Router["Vue Router (含鉴权守卫)"]
        Store["Pinia 状态管理"]
        APIClient["Axios API 客户端"]
    end

    subgraph Backend["后端层 (FastAPI + Python 3.11+)"]
        API["RESTful API 路由"]
        Auth["JWT 认证中间件"]
        Services["业务服务层"]
        AIEngine["AI 内容生成引擎"]
        Publisher["多平台发布调度器"]
        TaskQueue["任务队列 (Celery)"]
    end

    subgraph Data["数据层"]
        DB["SQLite 数据库 (开发) → PostgreSQL (生产)"]
        Redis["Redis 缓存/队列"]
        FileStorage["文件存储 (本地/OSS)"]
    end

    subgraph External["外部服务"]
        WeChatAPI["微信公众号 API"]
        XHSAuto["小红书 Playwright 自动化"]
        DouyinAuto["抖音 Playwright 自动化"]
        VCAuto["视频号 Playwright 自动化"]
        LLM["大模型 API (DeepSeek/OpenAI/Moonshot/智谱)"]
    end

    UI --> Router
    Router --> Store
    Store --> APIClient
    APIClient -->|HTTP REST| API
    API --> Auth
    Auth --> Services
    Services --> AIEngine
    Services --> Publisher
    AIEngine -->|API 调用| LLM
    Publisher --> TaskQueue
    TaskQueue --> WeChatAPI
    TaskQueue --> XHSAuto
    TaskQueue --> DouyinAuto
    TaskQueue --> VCAuto
    Services --> DB
    Services --> Redis
    Services --> FileStorage
```

## 2. 后端服务架构

```mermaid
flowchart LR
    subgraph API_Layer["API 路由层"]
        AuthRouter["/api/auth"]
        DashboardRouter["/api/dashboard"]
        AccountRouter["/api/accounts"]
        ContentRouter["/api/contents"]
        PublishRouter["/api/publish"]
        TemplateRouter["/api/templates"]
        ModelConfigRouter["/api/model-configs"]
        ModelsRouter["/api/models"]
    end

    subgraph Service_Layer["业务服务层"]
        AuthService["认证服务"]
        AccountService["账号管理服务"]
        ContentService["内容管理服务"]
        AIService["AI 生成服务"]
        PublishService["发布调度服务"]
        TemplateService["模板管理服务"]
    end

    subgraph Adapter_Layer["平台适配层"]
        WeChatAdapter["微信公众号适配器"]
        XHSAdapter["小红书适配器"]
        DouyinAdapter["抖音适配器"]
        VideoAdapter["视频号适配器"]
    end

    subgraph Infra["基础设施层"]
        DB_Layer["数据库 (SQLite)"]
        Redis_Layer["Redis"]
        Celery_Layer["Celery Worker"]
        Playwright_Layer["Playwright 浏览器池"]
    end

    AuthRouter --> AuthService
    DashboardRouter --> AccountService
    DashboardRouter --> ContentService
    DashboardRouter --> PublishService
    AccountRouter --> AccountService
    ContentRouter --> ContentService
    ContentRouter --> AIService
    PublishRouter --> PublishService
    TemplateRouter --> TemplateService
    ModelConfigRouter --> AuthService

    PublishService --> WeChatAdapter
    PublishService --> XHSAdapter
    PublishService --> DouyinAdapter
    PublishService --> VideoAdapter

    AuthService --> DB_Layer
    AccountService --> DB_Layer
    ContentService --> DB_Layer
    PublishService --> Redis_Layer
    PublishService --> Celery_Layer
    XHSAdapter --> Playwright_Layer
    DouyinAdapter --> Playwright_Layer
    VideoAdapter --> Playwright_Layer
```

## 3. 技术栈总览

| 层级 | 技术选型 |
|------|----------|
| **前端框架** | Vue 3 + TypeScript + Vite 6 |
| **UI 组件库** | Arco Design Vue + 自定义组件 |
| **状态管理** | Pinia |
| **HTTP 客户端** | Axios（JWT 拦截器） |
| **后端框架** | Python 3.11+ / FastAPI |
| **ORM** | SQLAlchemy 2.0（异步模式）+ Alembic |
| **数据库** | SQLite（开发）→ PostgreSQL（生产） |
| **数据库 ID 规范** | 所有表主键使用 UUID VARCHAR(36)，超级管理员用户 ID 固定为 `"1"` |
| **任务队列** | Celery + Redis |
| **缓存** | Redis |
| **浏览器自动化** | Playwright |
| **AI 接入** | OpenAI 兼容 API（DeepSeek / OpenAI / Moonshot / 智谱 AI / 自定义） |
| **容器化** | Docker + Docker Compose |

## 4. 部署架构

```yaml
# docker-compose.yml 服务拓扑
services:
  frontend:
    build: docker/frontend/Dockerfile
    ports: ["5173:5173"]
    volumes: ["./frontend:/app"]
    depends_on: [backend]

  backend:
    build: docker/backend/Dockerfile
    ports: ["8000:8000"]
    volumes: ["./backend:/app"]
    environment: [DATABASE_URL, REDIS_URL, ...]
    depends_on: [redis]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  celery-worker:
    build: docker/backend/Dockerfile
    command: celery -A app.celery_app worker
    environment: [DATABASE_URL, REDIS_URL, ...]
    depends_on: [redis, backend]
```
