# AI 多平台内容发布系统

一站式 AI 驱动的多平台内容创作与自动化分发平台，支持小红书、抖音、微信视频号、微信公众号等主流社交媒体账号的统一管理、AI 内容生成、智能排版以及多平台一键发布/定时发布。

## 特性

- AI 内容生成：输入主题/关键词，AI 自动生成适配多平台风格的文案变体
- 多平台矩阵管理：统一管理微信公众号、小红书、抖音、视频号等平台账号
- 一键发布：选择内容和账号，一键发布到多个平台，支持定时发布
- 数据仪表盘：实时统计账号数、发布量、AI 生成次数等核心指标
- 模板中心：内置多平台内容模板，快速复用优质内容结构
- 模型配置：灵活切换 AI 模型提供方（OpenAI / DeepSeek / Moonshot / 智谱 AI）
- 权限管理：RBAC 角色权限体系，支持自定义角色、读写分离权限控制
- 审核工作流：内容审核、SQL 变更审核、用户创建审核

## 技术栈

- **前端**：Vue 3 + TypeScript + Vite + Arco Design Vue + Tailwind CSS
- **后端**：FastAPI + SQLAlchemy 2.0（异步）+ SQLite/PostgreSQL
- **任务队列**：Celery + Redis
- **AI 接入**：OpenAI 兼容 API
- **部署**：Docker + Docker Compose

## 快速开始

### Docker 部署（推荐）

```bash
git clone https://github.com/zhangming123456/ai-multi-platform-release.git
cd ai-multi-platform-release
docker-compose up -d
```

服务启动后：

- 前端：http://localhost:5173
- 后端 API：http://localhost:8000
- API 文档：http://localhost:8000/docs

### 本地开发

```bash
# 后端
cd backend
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000

# 前端
cd frontend
npm install
npm run dev
```

## 默认账号

| 账号 | 密码 | 角色 |
|------|------|------|
| admin@admin.com | admin123 | 超级管理员 |

## 技术文档

| 文档 | 说明 |
|------|------|
| [文档索引](./.trae/documents/README.md) | 全部技术文档导航 |
| [架构设计与技术选型](./.trae/documents/Architecture-Overview.md) | 系统架构、技术栈、部署架构 |
| [前端架构设计](./.trae/documents/Frontend-Architecture.md) | 目录结构、路由、数据流、组件通讯 |
| [后端 API 接口文档](./.trae/documents/Backend-API.md) | 全部 RESTful API 接口定义 |
| [数据模型与数据库设计](./.trae/documents/Database-Design.md) | ER 图、DDL、索引设计 |
| [账号管理与权限管理](./.trae/documents/Account-Permission-Management.md) | RBAC 权限体系、用户/角色/权限模型 |
| [产品需求文档](./.trae/documents/PRD.md) | 产品功能需求与规划 |

## License

MIT License
