# 活动管理功能设计文档

> 模块：内容管理 · 活动管理
> 状态：设计稿（待评审）
> 关联页面：活动管理（新增）、创作内容（改造）、内容列表（增强）

## 1. 需求概述

### 1.1 背景与目标

当前创作内容页面仅支持「主题 + 关键词 + 平台 + 素材」的自由创作。当运营团队围绕某个营销活动（如新品首发、节日大促、门店开业）批量生产多平台内容时，每次都需要手动把活动信息（名称、介绍、卖点、地点等）粘贴进主题，导致：

- 活动信息在多个平台间反复复制，易遗漏、易错乱
- 活动宣发图无法随内容创作直接复用，素材重复上传
- 无法按活动维度统计/筛选已生成的内容

本次新增「活动管理」功能，将活动作为可维护、可复用的创作上下文：

1. 在「内容管理」菜单下新增**活动管理**页面，维护活动信息（名称、介绍、宣发图、地点、面向平台）
2. 在**创作内容**页面支持选择活动，AI 将结合活动信息与宣发图创作出符合活动调性的内容
3. 保存内容时记录所属活动（`campaign_id`），内容列表支持按活动筛选，形成「活动 → 内容」的关联闭环

### 1.2 用户故事

| 编号 | 角色   | 用户故事                                                                                    |
| ---- | ------ | ------------------------------------------------------------------------------------------- |
| US-1 | 运营者 | 我在活动管理页创建活动「夏季新品清凉季」，填写名称、介绍、上传宣发图、选择面向小红书/公众号 |
| US-2 | 运营者 | 我在创作内容页选择该活动，输入主题「防晒衣推荐」，AI 生成的文案自动融入活动介绍与宣发图信息 |
| US-3 | 运营者 | 我保存生成内容后，内容自动归属该活动；在内容列表可按活动筛选查看该活动下的所有内容          |
| US-4 | 运营者 | 活动结束后，我可将活动归档，归档后不再出现在创作内容的候选活动中                            |

### 1.3 功能范围

**本期包含：**

- 活动管理页面（列表 / 新建 / 编辑 / 详情 / 归档 / 删除）
- 活动信息维护：活动名称、活动介绍、活动宣发图（多图）、活动地点、面向平台（多选）
- 创作内容页的活动选择（单选，附加创作背景）
- AI 创作时注入活动信息（文本 + 宣发图多模态）
- 内容与活动关联（保存内容记录 `campaign_id`、内容列表按活动筛选）

**本期不包含：**

- 活动维度的发布任务聚合报表（活动下已发布内容的统计看板）
- 活动审批流、活动协作（多人共同编辑）
- 活动模板复制（从历史活动复制创建新活动）

## 2. 数据模型设计

### 2.1 campaigns 表（新增）

| 字段        | 类型                         | 说明                                                                        |
| ----------- | ---------------------------- | --------------------------------------------------------------------------- |
| id          | VARCHAR(36) PK               | UUID 主键                                                                   |
| user_id     | VARCHAR(36) INDEX            | 归属用户（创建者）                                                          |
| name        | VARCHAR(200)                 | 活动名称                                                                    |
| description | TEXT NULL                    | 活动介绍（可空，AI 创作上下文）                                             |
| media_urls  | TEXT NULL                    | 活动宣发图 URL 列表，JSON 数组字符串，如 `["/uploads/xxx.png"]`             |
| location    | VARCHAR(200) NULL            | 活动地点（线下门店 / 城市 / 线上等）                                        |
| platforms   | VARCHAR(100) NULL            | 面向平台，JSON 数组字符串，如 `["xiaohongshu","wechat_mp"]`，取值同平台常量 |
| status      | VARCHAR(20) DEFAULT 'active' | 状态：`active` 启用 / `archived` 归档                                       |
| created_at  | DATETIME                     | 创建时间                                                                    |
| updated_at  | DATETIME                     | 更新时间                                                                    |

模型定义位置：`models/campaign.go`，在 `InitModels()` 注册（`models/init.go`）。新表由 `RunSyncdb` 自动创建，无需手工迁移。

### 2.2 contents 表（新增字段）

| 字段        | 类型                   | 说明                                    |
| ----------- | ---------------------- | --------------------------------------- |
| campaign_id | VARCHAR(36) NULL INDEX | 所属活动 ID，可空（自由创作的内容为空） |

通过 `migrateSchema()` 增量 `ALTER TABLE contents ADD COLUMN campaign_id` 添加（`services/database.go` 的迁移列表中追加一项）。外键逻辑上指向 campaigns.id，但沿用本项目「不加物理外键、靠应用层保证」的惯例。

### 2.3 ai_generation_records 表（新增字段，可选）

| 字段        | 类型                   | 说明                                              |
| ----------- | ---------------------- | ------------------------------------------------- |
| campaign_id | VARCHAR(36) NULL INDEX | 生成记录归属的活动 ID，用于后续统计活动内容生产量 |

若本期不做活动内容量统计，可暂缓该字段，仅记录 contents.campaign_id。

### 2.4 平台枚举

沿用现有平台常量，不新增平台维度：

| 值           | 名称   |
| ------------ | ------ |
| wechat_mp    | 公众号 |
| xiaohongshu  | 小红书 |
| douyin       | 抖音   |
| wechat_video | 视频号 |

## 3. API 设计

### 3.1 活动 CRUD

| 方法   | 路径                          | 权限                    | 说明                                                                                |
| ------ | ----------------------------- | ----------------------- | ----------------------------------------------------------------------------------- |
| GET    | `/api/campaigns/`             | `campaign:read`         | 分页列表 + 关键字/平台/状态筛选                                                     |
| POST   | `/api/campaigns/`             | `campaign:create:write` | 创建活动                                                                            |
| GET    | `/api/campaigns/:campaign_id` | `campaign:read`         | 活动详情                                                                            |
| PUT    | `/api/campaigns/:campaign_id` | `campaign:update:write` | 编辑活动（指针字段，仅更新非空）                                                    |
| DELETE | `/api/campaigns/:campaign_id` | `campaign:delete:write` | 删除活动（软删除：置 archived；存在关联内容时拒绝物理删除）                         |
| GET    | `/api/campaigns/options`      | `campaign:read`         | 候选活动下拉（仅 `active`，字段精简：id/name/platforms/location），供创作内容页使用 |

请求体示例（创建）：

```json
{
  "name": "夏季新品清凉季",
  "description": "主推防晒衣与冰感面料系列，主打「轻薄防晒、上身冰凉」卖点",
  "media_urls": ["/uploads/ab12cd34.png"],
  "location": "全国门店 + 线上商城",
  "platforms": ["xiaohongshu", "wechat_mp"]
}
```

路由注册（`main.go` 的 `registerRoutes()`，遵循「静态路径先于 `:id`」约定，且路径参数名与 `GetPathParam` 保持一致）：

```go
campaigns := &controllers.CampaignsController{}
web.Router("/api/campaigns/options", campaigns, "get:Options")
web.Router("/api/campaigns/", campaigns, "get:List;post:Create")
web.Router("/api/campaigns/:campaign_id", campaigns, "get:Get;put:Update;delete:Delete")
```

### 3.2 内容模块扩展

| 方法 | 路径                               | 变更                          | 说明                                 |
| ---- | ---------------------------------- | ----------------------------- | ------------------------------------ |
| GET  | `/api/contents/`                   | 新增 `campaign_id` 查询参数   | 按活动筛选内容                       |
| POST | `/api/contents/`                   | 请求体新增 `campaign_id` 字段 | 保存内容时记录所属活动               |
| POST | `/api/contents/ai-generate-stream` | 请求体新增 `campaign_id` 字段 | AI 创作时注入活动上下文（见第 5 节） |

`aiGenerateStreamRequest` 扩展：

```go
type aiGenerateStreamRequest struct {
    Topic      string                  `json:"topic"`
    Platforms  []string                `json:"platforms"`
    Style      string                  `json:"style"`
    Keywords   []string                `json:"keywords"`
    PlanID     string                  `json:"plan_id"`
    ModelID    string                  `json:"model_id"`
    Files      []services.UploadedFile `json:"files"`
    CampaignID string                  `json:"campaign_id"` // 新增
}
```

### 3.3 图片上传

复用现有 `POST /api/uploads`（multipart，字段名 `file`，返回 `{url:"/uploads/xxx.png"}`）。需在其权限白名单中追加 `campaign:create:write`、`campaign:update:write`，使运营者可为活动上传宣发图。

## 4. 权限设计

在 `services/rbac_init_service.go` 的 `RBACResources` 追加：

| 权限键                | 名称     | 说明                             |
| --------------------- | -------- | -------------------------------- |
| campaign:read         | 查看活动 | 查看活动列表、详情，读取候选活动 |
| campaign:create:write | 创建活动 | 新建活动                         |
| campaign:update:write | 编辑活动 | 编辑活动信息                     |
| campaign:delete:write | 删除活动 | 删除/归档活动                    |

内置角色默认权限（`defaultRolePermissions`）建议：

- **manager**：追加以上 4 项
- **operator**（日常内容运营）：追加以上 4 项
- **reviewer**：追加 `campaign:read`

说明：活动管理属于日常运营能力，默认授予 manager / operator；reviewer 仅只读。超级管理员不受限制（`IsSuperAdmin` 恒放行）。

## 5. AI 创作集成设计

### 5.1 交互流程（创作内容页）

1. 创作内容表单新增「关联活动」选择器（单选下拉，数据源 `GET /api/campaigns/options`，仅展示 `active` 活动）
2. 选择活动后：
   - 该活动的信息不强制覆盖主题输入，主题仍可自由填写
   - 活动信息作为**创作背景**附加到 AI 请求（`campaign_id` 随请求体提交）
   - 若活动的面向平台已勾选，自动将平台选择同步为活动的面向平台（用户可再手动增删）
3. 未选择活动时行为与现状完全一致（不携带 `campaign_id`）

### 5.2 后端注入（ai_service.go）

在 `GenerateContentStream` / `GenerateContentVariants` 前新增活动上下文装配：

```go
// loadCampaignContext 加载活动并构造注入内容
func loadCampaignContext(campaignID string) (contextText string, files []UploadedFile, err error)
```

- 从 campaigns 表读取活动（name、description、location、platforms、media_urls）
- `contextText` 构造为结构化文本，追加进用户提示词：

```
【活动背景】
活动名称：夏季新品清凉季
活动介绍：主推防晒衣与冰感面料系列，主打「轻薄防晒、上身冰凉」卖点
活动地点：全国门店 + 线上商城
面向平台：小红书、公众号

请围绕以上活动背景进行创作，内容需贴合活动调性与卖点。
```

- 宣发图：遍历 `media_urls`，复用 `loadImageAsUploadedFile`（`services/inspection_service.go`）将 `/uploads/xxx.png` 加载为 `UploadedFile`（base64），追加到多模态 files，随现有 `buildContentVariantMessages` 的 vision 分支一起注入
- 装配顺序：`files` = 用户上传素材 + 活动宣发图；提示词 = 原有主题提示词 + 活动背景文本
- 若活动宣发图加载失败或为空，降级为仅文本注入（不阻断创作）

### 5.3 SSE 事件

沿用现有事件协议不变（`log / chunk / done / error / complete`），`done` 事件的 variant 结构不变（`{title, body, hashtags[]}`）。日志中可新增一条：`已注入活动「xxx」创作背景（含 N 张宣发图）`。

### 5.4 保存内容

创作结果保存时（`saveContent` → `POST /api/contents/`）携带 `campaign_id`，落库到 `contents.campaign_id`。

## 6. 前端页面设计

### 6.1 路由与菜单

| 路由         | 组件               | permKey       | sidebarType | sidebarOrder | icon |
| ------------ | ------------------ | ------------- | ----------- | ------------ | ---- |
| `/campaigns` | CampaignManage.vue | campaign:read | content     | 2            | gift |

调整「内容管理」组顺序：内容列表(0) → 创作内容(1) → **活动管理(2)** → 发布管理(3) → 模板管理(4)。

路由注册（`router/index.ts`），`icon` 需在 `AppSidebar.vue` 的 `iconRegistry` 中补充 `gift`（Arco `IconGift`）。

### 6.2 CampaignManage.vue（活动管理页）

参照 `StoreManage.vue` 的「列表 + 服务端分页 + 共用弹窗」范式，外层 `div.page-main` + `PageHeader`：

- **页头**：title「活动管理」、subtitle「维护营销活动信息，供 AI 内容创作引用」，`#actions` 放置「新建活动」按钮（`v-perm="'campaign:create:write'"`）
- **筛选区**：搜索框（按名称）、平台筛选（`a-select`：all + 4 平台）、状态筛选（all/active/archived）
- **表格列**：活动名称、面向平台（PlatformIcon 多图标）、活动地点、宣发图数、状态（active=启用/archived=已归档，StatusBadge 或 a-tag）、创建时间、操作
- **操作**：查看详情（弹窗展示介绍 + 宣发图 + 生成内容列表）、编辑（`campaign:update:write`）、归档/启用（`campaign:update:write`）、删除（`campaign:delete:write`，`a-popconfirm` 二次确认）
- **新建/编辑弹窗**（共用 `a-modal`，`editingId` 区分）字段：
  - 活动名称：`a-input`（必填，`maxLength=200`）
  - 活动介绍 + 活动宣发图：**复用 `AttachmentInputArea`**（`v-model` 绑定介绍文本，`v-model:file-list` 绑定宣发图），`theme="light"`、`file-types=['image']`、`max-count=9`、`min-rows=3`，图片选择后本地以 data URL 预览
  - 活动地点：`a-input`
  - 面向平台：`a-checkbox-group`（公众号 / 小红书 / 抖音 / 视频号，多选）

**宣发图保存逻辑**：弹窗确认保存时，遍历 `file-list`——data URL 项先调 `POST /api/uploads` 上传获取 `/uploads/xxx` URL；已是 `/uploads/` URL 的项直接保留；最终 URL 数组 JSON 序列化存入 `media_urls`。

**活动详情弹窗**：`a-descriptions` 展示活动信息 + 宣发图缩略图（`a-image` 多图）+ 底部「查看该活动内容」跳转 `/content?campaign_id=xxx`（或内嵌该活动内容列表）。

### 6.3 ContentCreate.vue（创作内容页改造）

- 表单新增 `campaignId`（默认空）与候选活动列表（`onMounted` 拉取 `GET /api/campaigns/options`）
- 输入区新增「关联活动」`a-select`（可清空，placeholder「不关联活动，自由创作」），候选仅 `active` 活动，显示名称 + 地点
- 选择活动后：`selectedPlatforms` 自动同步为活动 platforms（追加式：保留用户已选平台并加入活动平台，保证非空）
- 生成请求体携带 `campaign_id`（未选择时省略）
- `saveContent` 携带 `campaign_id`
- 预览区「生成预览」卡片标题旁可显示当前关联活动 tag（`a-tag color="arcoblue"`），便于确认创作背景

### 6.4 ContentList.vue（内容列表增强）

- 筛选区新增「关联活动」`a-select`（数据源同 `/api/campaigns/options`，含"全部"），选中后请求 `GET /api/contents/?campaign_id=xxx`
- 表格新增「所属活动」列（`campaign_id` 非空时显示活动名称 tag，点击可筛选）

### 6.5 TypeScript 类型（types/index.ts 或页面内）

```ts
export interface Campaign {
  id: string
  user_id: string
  name: string
  description: string
  media_urls: string[]
  location: string
  platforms: PlatformValue[]
  status: 'active' | 'archived'
  created_at: string
  updated_at: string
}
```

## 7. 实施计划 · 每日任务清单

> 约定：每个任务完成后在「验收标准」处打勾；当日全部验收通过视为该天完成。后端优先推进（Day 1-3），前端从 Day 4 接入，Day 3 与 Day 4 可并行。

### 7.0 阶段总览

| 天    | 阶段                  | 交付物                                                      | 对应原步骤 |
| ----- | --------------------- | ----------------------------------------------------------- | ---------- |
| Day 1 | 后端数据模型          | campaigns 表建成，contents / ai_generation_records 字段迁移 | 步骤 1     |
| Day 2 | 后端权限 + 活动 CRUD  | 活动接口可 curl 自测、权限生效                              | 步骤 2、3  |
| Day 3 | 后端 AI 集成          | 活动上下文注入 AI 创作、内容保存带 campaign_id              | 步骤 4     |
| Day 4 | 前端活动管理页        | 活动管理页面可浏览 / 新建 / 编辑 / 归档                     | 步骤 5、6  |
| Day 5 | 前端联动 + 全链路验证 | 创作页选活动生成、内容列表筛选、回归通过                    | 步骤 7、8  |

### Day 1 · 后端数据模型

| ID   | 任务                                                                                                                                                                                                             | 验收标准                                                           |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| D1-1 | 新建 `models/campaign.go`，定义 `Campaign` 结构体（字段见 2.1：id / user_id / name / description / media_urls / location / platforms / status / created_at / updated_at，UUID 主键）                             | 字段与 2.1 完全一致，orm tag 正确                                  |
| D1-2 | 在 `models/init.go` 的 `InitModels()` 注册 `new(Campaign)`                                                                                                                                                       | 编译通过                                                           |
| D1-3 | 在 `services/database.go` 的 `migrateSchema()` 迁移列表追加两条：`ALTER TABLE contents ADD COLUMN campaign_id`、`ALTER TABLE ai_generation_records ADD COLUMN campaign_id`（含索引，先查列存在再执行，保证幂等） | 迁移函数存在且可重复执行                                           |
| D1-4 | `go build ./...` + 启动后端，确认建表与迁移成功                                                                                                                                                                  | 启动日志无错；`campaigns` 表存在，`contents` 表含 `campaign_id` 列 |

### Day 2 · 后端权限与活动 CRUD

| ID   | 任务                                                                                                                                                                                                                                                                                      | 验收标准                                                            |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| D2-1 | `services/rbac_init_service.go`：`RBACResources` 追加 `campaign:read` / `campaign:create:write` / `campaign:update:write` / `campaign:delete:write` 四条（含名称与描述）                                                                                                                  | 启动后 `rbac_resources`、`rbac_permissions` 出现 4 条 campaign 记录 |
| D2-2 | `defaultRolePermissions` 配置：manager / operator 追加全部 4 键，reviewer 追加 `campaign:read`                                                                                                                                                                                            | 权限管理页可见新权限键且各角色默认勾选正确                          |
| D2-3 | `controllers/uploads_controller.go`：`CheckPermissionAny` 白名单追加 `campaign:create:write`、`campaign:update:write`                                                                                                                                                                     | 活动保存宣发图上传不返回 403                                        |
| D2-4 | 新建 `controllers/campaigns_controller.go`：`CampaignsController` 实现 `List`（分页 + name / platforms / status 筛选）、`Create`、`Get`、`Update`（指针字段）、`Delete`（软删除：置 archived；存在关联内容时禁止物理删除）、`Options`（仅 active，返回 id / name / location / platforms） | 各方法权限检查正确、数据归属当前用户                                |
| D2-5 | `main.go` `registerRoutes()` 注册 3 条路由（`/api/campaigns/options` 静态路径优先注册；`/api/campaigns/`；`/api/campaigns/:campaign_id`，路径参数名与 `GetPathParam("campaign_id")` 严格一致）                                                                                            | curl 全接口自测通过；无权限用户访问返回 403                         |

### Day 3 · 后端 AI 集成

| ID   | 任务                                                                                                                                                                                                                        | 验收标准                                     |
| ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| D3-1 | `controllers/contents_controller.go`：`contentCreateRequest` 增加 `campaign_id`；`List` 支持 `campaign_id` 查询参数过滤；`aiGenerateStreamRequest` 增加 `campaign_id`                                                       | 保存 / 筛选 / 生成请求均可携带 campaign_id   |
| D3-2 | `services/ai_service.go`：新增 `loadCampaignContext(campaignID)` — 读取活动，组装 `contextText`（活动名称 / 介绍 / 地点 / 平台）与宣发图 `[]UploadedFile`（复用 `loadImageAsUploadedFile` 加载 `/uploads/` 图片）           | 活动不存在或宣发图加载失败时返回空值但不报错 |
| D3-3 | `GenerateContentStream` / `GenerateContentVariants` 注入改造：提示词追加活动背景文本；`files` = 用户素材 + 活动宣发图合并；所选模型不支持视觉时自动降级为仅文本注入；日志输出「已注入活动「xxx」创作背景（含 N 张宣发图）」 | 请求体内含活动上下文；无视觉模型时不报错     |
| D3-4 | `AIGenerateStream` / `AIGenerate`：写 `AIGenerationRecord` 时携带 campaign_id                                                                                                                                               | 生成记录落库含 campaign_id                   |
| D3-5 | `go build ./...` + `go vet ./services/` + 用 MiniMax 实测 SSE                                                                                                                                                               | 日志出现活动注入记录；正文与话题标签正常解析 |

### Day 4 · 前端活动管理页

| ID   | 任务                                                                                                                                                                                                                                                            | 验收标准                                             |
| ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| D4-1 | `router/index.ts`：注册 `/campaigns` 路由（name=CampaignManage、permKey=campaign:read、sidebarType=content、sidebarOrder=2、icon=gift）                                                                                                                         | 菜单出现在「内容管理」组，无权限用户不可见           |
| D4-2 | `components/layout/AppSidebar.vue`：`iconRegistry` 补充 `gift` → `IconGift`                                                                                                                                                                                     | 侧边栏图标正常渲染                                   |
| D4-3 | 新建 `pages/CampaignManage.vue`：页面骨架（PageHeader + 搜索 / 平台 / 状态筛选 + a-table + a-pagination），数据走 `GET /api/campaigns/` 服务端分页                                                                                                              | 列表可浏览、筛选生效、分页正确                       |
| D4-4 | 新建 / 编辑共用弹窗：活动名称（必填）+ 活动介绍 + 宣发图（复用 `AttachmentInputArea`，file-types=image、max-count=9、data URL 本地预览）+ 活动地点 + 面向平台（a-checkbox-group）；保存时 data URL 项先调 `POST /api/uploads` 换 URL，已 `/uploads/` 项直接保留 | 新建 / 编辑 / 宣发图上传全流程可用，刷新后宣发图可见 |
| D4-5 | 详情弹窗（a-descriptions + 宣发图 a-image 多图 + 「查看该活动内容」跳转 `/content?campaign_id=xxx`）；行内归档 / 启用、删除（a-popconfirm + v-perm）                                                                                                            | 操作按钮权限正确、二次确认有效                       |
| D4-6 | `npx eslint src/pages/CampaignManage.vue` + `npx vue-tsc -b`                                                                                                                                                                                                    | 本文件无新增 lint / 类型错误                         |

### Day 5 · 前端联动 + 全链路验证

| ID   | 任务                                                                                                                                                                                                                                            | 验收标准                                                         |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| D5-1 | `ContentCreate.vue`：输入区新增「关联活动」a-select（onMounted 拉取 `GET /api/campaigns/options`，可清空）；选择后 `selectedPlatforms` 自动同步活动 platforms；生成请求体携带 campaign_id；saveContent 携带 campaign_id；预览区显示关联活动 tag | 选活动后平台自动联动；生成日志含「已注入活动」；保存内容归属活动 |
| D5-2 | `ContentList.vue`：筛选区新增「关联活动」a-select；请求带 `campaign_id` 参数；表格新增「所属活动」列（点击可筛选）                                                                                                                              | 按活动筛选结果正确                                               |
| D5-3 | 全链路浏览器验证：建活动（含宣发图）→ 创作页选活动生成 → 保存 → 内容列表按活动筛选 → 活动详情查看内容                                                                                                                                           | 全链路无报错、SSE 正常                                           |
| D5-4 | 回归验证：不选活动自由创作行为与改造前一致；检查项生成、模板、素材、发布等既有功能不受影响                                                                                                                                                      | 回归通过                                                         |
| D5-5 | 按实现实际更新本文档（字段 / 接口 / 页面如有偏差同步修正）                                                                                                                                                                                      | 文档与实现一致                                                   |

## 8. 风险与备注

- **活动宣发图体积**：`/api/uploads` 限制单图 ≤10MB，活动宣发图建议前端沿用 `compressImage`（maxWidth 1920、quality 0.8）压缩后再上传
- **AI 多模态限制**：注入宣发图要求所选模型支持 vision（现有 `ModelSupportsFiles` 校验可复用）；若模型不支持视觉，自动降级为仅文本注入
- **删除保护**：活动存在关联内容时仅允许归档（软删除），不允许物理删除，避免内容列表活动信息悬挂
- **既有问题规避**：contents 现有 `/api/contents/:id` 路由参数名与 `GetPathParam("content_id")` 不一致，活动路由一律采用 `:campaign_id` 与 `GetPathParam("campaign_id")` 严格一致，规避同类问题
