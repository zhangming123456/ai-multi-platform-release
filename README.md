# AI 多平台内容发布系统

一站式 AI 驱动的多平台内容创作与自动化分发平台，支持小红书、抖音、微信视频号、微信公众号等主流社交媒体账号的统一管理、AI 内容生成、智能排版以及多平台一键发布/定时发布。

## ✨ 特性

- **🤖 AI 内容生成**：输入主题/关键词，AI 自动生成适配多平台风格的文案变体，支持图片/视频多模态分析
- **📱 多平台矩阵管理**：统一管理微信公众号、小红书、抖音、视频号等平台账号
- **🚀 一键发布**：选择内容和账号，一键发布到多个平台，支持定时发布
- **📊 数据仪表盘**：实时统计账号数、发布量、AI 生成次数等核心指标
- **🎨 模板中心**：内置多平台内容模板，快速复用优质内容结构
- **⚙️ 模型配置**：灵活切换 AI 模型提供方（OpenAI / DeepSeek / Moonshot / 智谱 AI / 自定义）
- **📱 响应式设计**：完美适配桌面端、平板、移动端，支持 248px 超小屏
- **🔒 JWT 认证**：安全的用户认证与权限管理

## 🏗️ 技术栈

### 后端

- **框架**：FastAPI（异步 Python Web 框架）
- **数据库**：SQLite + aiosqlite（异步驱动）
- **ORM**：SQLAlchemy 2.0（异步模式）
- **任务队列**：Celery + Redis
- **AI SDK**：OpenAI Python SDK（兼容多模型提供方）
- **认证**：JWT（PyJWT）
- **代码规范**：Ruff（lint + format）

### 前端

- **框架**：Vue 3 + TypeScript
- **构建工具**：Vite
- **UI 组件库**：Arco Design Vue
- **状态管理**：Pinia
- **路由**：Vue Router 4
- **样式**：Tailwind CSS v4 + SCSS
- **图标**：Lucide Vue Next
- **API 文档**：Swagger UI
- **代码规范**：ESLint + Prettier

### 部署

- **容器化**：Docker + Docker Compose
- **多阶段构建**：前端 Nginx 部署、后端 Uvicorn 部署

## 📁 项目结构

```
ai-multi-platform-release/
├── backend/                 # 后端服务
│   ├── app/
│   │   ├── core/           # 核心模块（依赖注入、安全）
│   │   ├── models/         # 数据模型
│   │   ├── routers/        # API 路由
│   │   ├── schemas/        # Pydantic 数据模型
│   │   ├── services/       # 业务逻辑
│   │   │   └── platforms/  # 各平台发布适配器
│   │   ├── config.py       # 配置管理
│   │   ├── database.py     # 数据库连接
│   │   └── main.py         # 应用入口
│   ├── pyproject.toml
│   └── requirements.txt
├── frontend/               # 前端应用
│   ├── src/
│   │   ├── components/     # 组件
│   │   │   ├── layout/     # 布局组件
│   │   │   └── shared/     # 共享组件
│   │   ├── pages/          # 页面
│   │   ├── stores/         # Pinia 状态
│   │   ├── router/         # 路由配置
│   │   ├── types/          # TypeScript 类型
│   │   ├── utils/          # 工具函数
│   │   └── style.css       # 全局样式
│   └── package.json
├── docker/                 # Docker 构建文件
│   ├── backend/Dockerfile
│   └── frontend/Dockerfile
├── docker-compose.yml      # Docker Compose 配置
└── .env.example            # 环境变量示例
```

## 🚀 快速开始

### 环境要求

- Python 3.9+
- Node.js 18+
- Redis（可选，Celery 任务队列需要）

### Docker 部署（推荐）

```bash
# 克隆项目
git clone https://github.com/zhangming123456/ai-multi-platform-release.git
cd ai-multi-platform-release

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps
```

服务启动后：
- 前端：http://localhost:5173
- 后端 API：http://localhost:8000
- API 文档：http://localhost:8000/docs
- 前端内嵌 API 文档：http://localhost:5173/developer/docs

### 本地开发

#### 后端

```bash
cd backend

# 创建虚拟环境
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# 安装依赖
pip install -r requirements.txt

# 配置环境变量
cp .env.example .env
# 编辑 .env，填入 AI_API_KEY 等配置

# 启动开发服务器
uvicorn app.main:app --reload --port 8000
```

#### 前端

```bash
cd frontend

# 安装依赖
npm install

# 配置环境变量（可选）
cp .env.development .env.local

# 启动开发服务器
npm run dev
```

## 🔐 默认账号

| 邮箱 | 密码 | 角色 |
|------|------|------|
| admin@admin.com | admin123 | 管理员 |

## 📱 功能模块

### 1. 仪表盘（Dashboard）

- 核心指标统计卡片（总账号数、今日发布、待处理任务、AI 生成次数）
- 平台账号健康度面板
- 最近发布记录列表

### 2. 账号管理（Accounts）

- 多平台账号卡片式管理
- 支持微信公众号、小红书、抖音、视频号
- Cookie / Token 授权管理
- 账号状态实时检测与刷新

### 3. AI 内容创建（Content Create）

- 主题/关键词输入 + 平台多选
- AI 流式生成多平台文案变体（SSE 实时输出）
- 图片/视频多模态上传与分析
- 图片自动压缩转 base64
- API 调用日志实时查看
- 三栏布局：生成预览（上）+ API 日志（左下）+ 创作输入（右下）

### 4. 内容管理（Content List）

- 已生成内容列表查看
- 按平台筛选
- 内容删除管理

### 5. 发布管理（Publish）

- 发布任务队列（待发布/发布中/已发布/失败）
- 按平台筛选
- 创建发布任务（选择内容 + 账号 + 时间）
- 失败任务重试

### 6. 模板中心（Templates）

- 多平台内容模板
- 按平台分类查看
- 模板快速复用

### 7. 模型配置（Token Plan）

- AI 模型提供方管理
- 支持 OpenAI / DeepSeek / Moonshot / 智谱 AI / 自定义
- 每个模型独立配置上下文窗口
- 模型类型多选（文本生成 / 推理模型 / 视觉理解 / 图片生成 / 视频生成）
- API Key、Base URL 配置
- 模型启用/禁用开关

### 8. API 文档（Api Docs）

- 内嵌 Swagger UI
- 在线查看和调试所有 API 接口

## ⚙️ 配置说明

### 后端环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DATABASE_URL` | 数据库连接地址 | `sqlite+aiosqlite:///./app.db` |
| `REDIS_URL` | Redis 连接地址 | `redis://localhost:6379/0` |
| `SECRET_KEY` | JWT 签名密钥 | `your-secret-key-change-in-production` |
| `ACCESS_TOKEN_EXPIRE_MINUTES` | Token 过期时间（分钟） | `1440` |
| `AI_API_KEY` | AI 模型 API Key | - |
| `AI_API_BASE_URL` | AI 模型 Base URL | `https://api.deepseek.com/v1` |
| `AI_MODEL` | 默认 AI 模型 | `deepseek-chat` |
| `CORS_ORIGINS` | 允许的 CORS 源 | `["http://localhost:3000","http://localhost:5173"]` |

### 前端环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `VITE_API_BASE_URL` | API 基础地址（直接请求） | `http://localhost:8000` |
| `VITE_API_PROXY_TARGET` | API 代理目标（开发服务器代理） | `http://localhost:8000` |

## 🧩 API 概览

### 认证

- `POST /api/auth/login` - 用户登录
- `POST /api/auth/register` - 用户注册

### 仪表盘

- `GET /api/dashboard/stats` - 获取仪表盘统计数据

### 账号管理

- `GET /api/accounts` - 获取账号列表
- `POST /api/accounts` - 添加账号
- `DELETE /api/accounts/{id}` - 删除账号
- `POST /api/accounts/{id}/refresh` - 刷新账号状态

### 内容管理

- `GET /api/contents` - 获取内容列表
- `POST /api/contents/ai-generate` - AI 生成内容
- `POST /api/contents/ai-generate-stream` - AI 流式生成内容（SSE）
- `DELETE /api/contents/{id}` - 删除内容

### 发布管理

- `GET /api/publish/tasks` - 获取发布任务列表
- `POST /api/publish/tasks` - 创建发布任务
- `POST /api/publish/tasks/{id}/retry` - 重试发布任务

### 模型配置

- `GET /api/model-configs` - 获取模型配置列表
- `POST /api/model-configs` - 创建模型配置
- `PUT /api/model-configs/{id}` - 更新模型配置
- `DELETE /api/model-configs/{id}` - 删除模型配置

更多 API 详情请访问 Swagger 文档：`http://localhost:8000/docs`

## 📷 截图

### 登录页

Apple 风格毛玻璃登录卡片，渐变光斑背景。

### 仪表盘

统计卡片 + 账号健康度 + 最近发布记录。

### AI 内容创建

三栏布局：生成预览 + API 日志 + 创作输入，支持 SSE 流式输出。

### 账号管理

卡片式多平台账号管理。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

MIT License
