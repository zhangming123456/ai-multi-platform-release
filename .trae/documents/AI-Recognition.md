# AI 识别功能技术文档

> **版本**：v1.0 · **更新**：2026-08-19 · **后端**：Go（backend-go）· **前端**：Vue 3
>
> **文档目标**：梳理项目中所有使用 AI 进行「识别 / 视觉理解 / 多模态分析」的功能，说明其业务场景、接口、Prompt 设计、结构化输出与质量保障机制。

---

## 目录

1. [功能总览](#1-功能总览)
2. [通用技术底座](#2-通用技术底座)
3. [AI 巡店分析](#3-ai-巡店分析)
4. [AI 单项检查项分析](#4-ai-单项检查项分析)
5. [AI 智能添加检查项](#5-ai-智能添加检查项)
6. [AI 整改复核](#6-ai-整改复核)
7. [AI 多模态内容生成](#7-ai-多模态内容生成)
8. [识别质量保障机制](#8-识别质量保障机制)
9. [相关文件](#9-相关文件)

---

## 1. 功能总览

项目基于 OpenAI 兼容的 Chat Completions 接口（支持 vision / 多模态），共沉淀 **5 大 AI 识别能力**，全部由后端 `backend-go/services` 统一封装：

| #   | 识别功能              | 前端入口                      | 后端接口                                                  | 核心服务函数                        | 识别内容                                     |
| --- | --------------------- | ----------------------------- | --------------------------------------------------------- | ----------------------------------- | -------------------------------------------- |
| 1   | **AI 巡店分析**       | `InspectionEdit.vue`          | `POST /api/inspections/ai-analyze-stream`                 | `AnalyzeInspectionStream`           | 现场图 + 标准图对比、关键词 → 评分/问题/建议 |
| 2   | **AI 单项检查项分析** | `InspectionEdit.vue`（单行）  | `POST /api/inspections/ai-analyze-item-stream`            | `AnalyzeSingleInspectionItemStream` | 单项标准 + 现场图 → 评分/相关性              |
| 3   | **AI 智能添加检查项** | `AITemplateItemGenerator.vue` | `POST /api/inspection-templates/ai-generate-items-stream` | `GenerateTemplateItemsByAIStream`   | 现场图 / 文字描述 → 结构化检查项             |
| 4   | **AI 整改复核**       | `InspectionTaskDetail.vue`    | `POST /api/inspection-tasks/:task_id/recheck`             | `RecheckRectifyPhoto`               | 整改照片 → 是否已修复/新评分                 |
| 5   | **AI 多模态内容生成** | `ContentCreate.vue`           | `POST /api/contents/ai-generate-stream`                   | `GenerateContentStream`             | 图片/视频素材 + 主题 → 多平台文案            |

> 除第 4 项（整改复核）为同步接口外，其余均提供 **SSE 流式**版本，前端实时展示识别进度与增量文本。

### 1.1 通用调用链路

```
前端页面（组装 skills / photos / keywords）
        │  fetch + Authorization: Bearer <token>
        ▼
控制器（Controller：权限校验 + 请求解析 + SSE 输出）
        │
        ▼
服务层（Service：模型配置解析 → Prompt 构建 → 图片转 base64 → 调用 LLM）
        │
        ▼
识别结果（结构化 JSON / SSE 事件流）
        │
        ▼
前端回填（评分、问题、建议、检查项、复核结论）
```

---

## 2. 通用技术底座

### 2.1 模型配置解析

所有 AI 识别功能的第一步均为解析模型配置：

```go
func ResolveModelConfig(planID, modelID string) (apiKey, baseURL, model, planName string, supportsVision bool, err error)
```

- `plan_id`：模型配置（ModelConfig）ID，确定 `apiKey / baseURL / planName`
- `model_id`：可选，在配置的多个模型中挑选具体模型
- `supportsVision`：从模型字段的 `types` 中判断是否包含 `vision / image / video`，决定是否发送图片内容

### 2.2 图片加载与 Base64 转换

为屏蔽图片来源差异，后端提供统一加载器 `loadImageAsUploadedFile`：

| 图片来源                     | 处理方式                       |
| ---------------------------- | ------------------------------ |
| `http(s)://` 完整链接        | 后端下载图片内容 → base64 编码 |
| `/uploads/xxx.jpg` 相对路径  | 读取本地上传目录 → base64 编码 |
| 前端直传 base64（Data 非空） | 原样使用                       |

> 关键设计：**所有图片统一由后端转为 `data:image/xxx;base64,...` 再交给视觉模型**，避免跨域/鉴权/相对路径导致模型无法读取图片。

```go
func visionPartFromFile(f UploadedFile) interface{} {
    // 统一转换为 { type: "image_url", image_url: { url: dataURL } }
}
```

### 2.3 Vision Content 构建

```go
// 巡店场景：标准图在前 + 现场图在后，便于模型对比
func buildInspectionVisionContent(prompt string, standardImages, photos []UploadedFile) []interface{}

// 内容生成场景：支持 image_url 与 video_url
func buildVisionContent(prompt string, files []UploadedFile) []interface{}
```

`user` 消息为多模态数组：`[{type:"text"}, {type:"image_url"}...]`；非 vision 模型（或未提供图片）时退化为纯文本 prompt。

### 2.4 Function Calling 结构化输出

为保证识别结果格式稳定，巡店分析与检查项生成采用 **强制 function calling**：

```go
type toolDefinition struct {
    Type     string           `json:"type"` // 固定 "function"
    Function toolFunctionSpec `json:"function"`
}

// 强制指定调用工具，避免模型输出普通文本或被截断
toolChoice := map[string]interface{}{
    "type": "function",
    "function": map[string]interface{}{"name": "submit_inspection_result"},
}
```

| 工具名                     | 用途                                   |
| -------------------------- | -------------------------------------- |
| `submit_inspection_result` | 提交巡店评分、问题、整改建议、风险总结 |
| `submit_template_items`    | 提交 AI 识别的检查表模板与检查项列表   |

> 部分模型不支持强制 `tool_choice`（返回 400），服务端会自动降级为 `tool_choice: "auto"` 重试（见 `GenerateTemplateItemsByAIStream`）。

### 2.5 SSE 流式事件协议

流式接口统一输出 `text/event-stream`，事件类型如下：

| 事件            | 说明                                        | 关键字段                      |
| --------------- | ------------------------------------------- | ----------------------------- |
| `log`           | 识别进度日志（level: info/req/ok/warn/err） | `level` / `message`           |
| `chunk`         | 增量文本片段                                | `text`                        |
| `result`        | 最终结构化识别结果                          | `data`（JSON）                |
| `request_data`  | 调试用：发送给模型的请求摘要                | `message` / `detail`          |
| `response_data` | 调试用：模型响应摘要                        | `message` / `detail`          |
| `error`         | 错误信息 + 可用模型配置列表                 | `message` / `available_plans` |
| `complete`      | 流结束                                      | `message`                     |

---

## 3. AI 巡店分析

### 3.1 业务场景

巡店人员上传门店现场照片（可带关键词/现场描述），AI 依据检查表模板的检查项标准（含标准图）进行**对比识别**，输出每个检查项的评分、问题与整改建议，以及整体风险总结。

### 3.2 接口定义

```
POST /api/inspections/ai-analyze-stream
Authorization: Bearer <token>
权限：inspection:ai:write
```

### 3.3 请求参数

```json
{
  "store_id": "门店ID",
  "template_id": "检查表模板ID",
  "plan_id": "模型配置ID",
  "model_id": "模型ID",
  "photos": [{ "data": "base64", "mime_type": "image/jpeg" }],
  "keywords": "巡店关键词/现场描述",
  "skills": [
    {
      "item_id": "检查项ID",
      "name": "门头形象",
      "category": "形象",
      "standard": "门店招牌完整、干净、夜间亮灯正常",
      "standard_images": ["/uploads/standard.jpg"],
      "score_type": "score | pass_fail",
      "max_score": 10,
      "score_options": [{ "score": 0, "label": "0分" }],
      "require_remark": true,
      "require_photo": true
    }
  ],
  "response_schema": {}
}
```

### 3.4 输出结构（result 事件）

```json
{
  "scores": [{ "item_id": "检查项ID", "score": 8, "comment": "地面有污渍" }],
  "issues": "问题与备注，按检查项分条列出",
  "suggestion": "AI 整改建议",
  "summary": "整体一句话概括（30-60字）",
  "high_risk_problems": [
    { "item_id": "", "item_name": "消防安全", "level": "高", "desc": "灭火器过期" }
  ],
  "main_problems": [{ "item_id": "", "item_name": "", "level": "中", "desc": "" }],
  "priority_suggest": [{ "title": "优先整改建议", "desc": "说明" }],
  "business_suggest": [{ "title": "运营优化建议", "desc": "说明" }]
}
```

### 3.5 Prompt 设计要点

1. **技能规范文本化**：`buildInspectionSkillsPrompt` 将每个检查项的 ID、名称、分类、标准、标准图、评分方式（选项/分值）、必填项拼装为结构化文本，作为 AI 的评分依据
2. **输入提示动态化**：`buildInspectionInputHints` 根据是否有照片/关键词生成不同提示（基于照片客观判断 / 基于通用规范建议性检查）
3. **图片分段说明**：标准图标注「检查标准参照图，作为评分基准」，现场图标注「巡店现场图，与标准图对比分析」
4. **严格约束**：仅返回与关键词/图片相关检查项的评分，`score` 必须落在允许分值范围内，无关项不返回
5. **系统角色**：`你是专业的连锁门店巡店督导专家…必须且只能调用 submit_inspection_result 工具`

### 3.6 前端交互（InspectionEdit.vue）

- AI 设置面板：选择模型配置（`ModelSelect`，需支持视觉理解）
- 一键分析：收集检查项 skills、全部现场照片、巡店关键词 → SSE 实时展示日志与增量文本
- 结果回填：评分写入各检查项，问题/建议写入表单，并标记 `ai_generated`
- 二次调整：用户可手动修改，修改后自动取消 AI 生成标记

---

## 4. AI 单项检查项分析

### 4.1 业务场景

对**单个检查项**独立评分分析：结合检查标准、标准参照图、巡店人员当前评分与反馈、现场图片，生成更合理的评分与反馈，并判断**图片与检查项的相关性**（防错图）。

### 4.2 接口定义

```
POST /api/inspections/ai-analyze-item-stream
POST /api/inspections/ai-analyze-item
Authorization: Bearer <token>
权限：inspection:ai:write
```

### 4.3 请求 / 输出

```json
// 请求（关键字段）
{
  "item_id": "检查项ID", "item_name": "门头形象",
  "standard": "检查标准", "standard_images": ["/uploads/x.jpg"],
  "score_type": "score", "max_score": 10, "score_options": [],
  "comment": "巡店人员现场反馈", "current_score": 6,
  "photos": [{ "data": "base64", "mime_type": "image/jpeg" }],
  "keywords": "补充信息", "plan_id": "", "model_id": ""
}

// 输出
{ "score": 8, "comment": "问题与备注", "suggestion": "整改建议", "photo_relevance": true }
```

> `photo_relevance`：仅当提供现场图片时返回，用于判断图片内容是否与该检查项匹配，前端据此提示「照片与检查项不相关」。

---

## 5. AI 智能添加检查项

### 5.1 业务场景

在**检查表模板编辑**中，用户上传门店现场照片或填写文字描述，AI 自动识别并生成结构化检查项（含分类、标题、标准、评分方式），可直接加入模板。

### 5.2 接口定义

```
POST /api/inspection-templates/ai-generate-items-stream
POST /api/inspection-templates/ai-generate-items
Authorization: Bearer <token>
```

### 5.3 请求 / 输出

```json
// 请求
{
  "plan_id": "", "model_id": "",
  "description": "需要检查门店门头招牌、灯箱、地面…",
  "photos": [{ "data": "base64", "mime_type": "image/jpeg" }],
  "existing_categories": ["形象", "卫生"]
}

// 输出（result 事件 data）
{
  "template_name": "门店形象检查表",
  "template_description": "适用于日常门店形象检查",
  "items": [
    {
      "category": "形象",
      "title": "门头招牌",
      "standard": "招牌完整、干净、夜间亮灯正常",
      "standard_images": ["data:image/jpeg;base64,..."],
      "score_type": "score",
      "max_score": 5,
      "score_options": [{ "score": 0, "label": "0分" }, { "score": 5, "label": "5分" }]
    }
  ]
}
```

### 5.4 设计要点

- 已有分类复用：`existing_categories` 引导 AI 优先归入已有分类，保持分类统一
- 标准图回填：用户上传的图片可作为检查项 `standard_images`（data URL）
- 格式兜底：`parseIntRaw` 兼容数字/字符串形式的 `max_score`、`score`；`normalizeGeneratedItems` 清洗非法项
- 强制工具：`submit_template_items` + `tool_choice` 强制，400 时降级 `auto`

---

## 6. AI 整改复核

### 6.1 业务场景

门店负责人提交整改照片后，AI 对照**检查标准 + 原问题 + 整改建议 + 整改照片**，判断问题是否已修复，并给出修复后评分。超限未修复则转入人工复核。

### 6.2 接口定义

```
POST /api/inspection-tasks/:task_id/recheck
Authorization: Bearer <token>
权限：inspection:task:recheck:write
前提：任务状态为「复核中」（rechecking）
```

### 6.3 请求 / 输出

```json
// 请求
{
  "items": [{ "item_id": "问题项ID" }],
  "plan_id": "", "model_id": ""
}

// 输出
{
  "status": "rectified | rectifying",
  "items": [
    {
      "item_id": "问题项ID", "item_name": "门头形象",
      "fixed": true, "score": 9, "reason": "从整改照片观察到招牌已更换…",
      "status": "fixed | not_fixed | manual",
      "recheck_count": 1
    }
  ]
}
```

### 6.4 状态流转

| 复核结果             | 问题项状态  | 说明                |
| -------------------- | ----------- | ------------------- |
| 已修复（fixed=true） | `fixed`     | 复核通过            |
| 未修复且未超限       | `not_fixed` | 继续整改 → 再次复核 |
| 未修复且达最大次数   | `manual`    | 转入人工复核        |

- 判定原则：`若无法从照片确认已修复，应判定为未修复`（保守策略）
- 复核完成后写入任务日志（`addTaskLog`）并向负责人/巡店人发送通知

---

## 7. AI 多模态内容生成

### 7.1 业务场景

创作内容页支持**文本 + 图片/视频附件**一并提交，AI 识别素材内容并将关键信息融入多平台文案（公众号 / 小红书 / 抖音等）。

### 7.2 接口定义

```
POST /api/contents/ai-generate-stream
POST /api/contents/ai-generate
Authorization: Bearer <token>
```

### 7.3 请求 / 输出

```json
// 请求
{
  "topic": "创作需求文本",
  "platforms": ["wechat", "xiaohongshu"],
  "files": [{ "data": "base64", "mime_type": "image/jpeg" }]
}

// 输出（done 事件）
{ "title": "标题", "body": "正文", "hashtags": ["#标签1", "#标签2"] }
```

### 7.4 设计要点

- `buildVisionContent` 同时支持 `image_url` 与 `video_url`
- 图片压缩：前端上传前压缩（>1600px 质量 0.8），编辑后按滑块质量导出
- 输出解析：`ParseStreamedContent` 支持「【标题】【正文】【标签】」标记格式，并兼容 JSON / JSON 数组回退
- 能力预检：`ModelSupportsFiles` 用于前端展示「该模型是否支持文件上传」

---

## 8. 识别质量保障机制

### 8.1 结构化解析与回退链

为保证模型输出格式异常时仍能解析，服务层实现多级回退：

```
优先解析 tool_calls（submit_inspection_result 的 arguments）
        │ 失败
        ▼
解析 content（去代码围栏 stripCodeFence）
        │ 失败
        ▼
数组型 scores → 对象型 scores（兼容两种 schema）
        │ 失败
        ▼
返回 AIGenerationError（附可用模型配置 available_plans）
```

### 8.2 分数量化与边界裁剪

```go
// 允许的离散分值（如 0/0.5/1/2/3/4/5）就近量化，避免 AI 输出 7.3 这类非法值
func quantizeScore(v float64, options []models.ScoreOption) float64

// 连续分值裁剪到 [0, maxScore]
func clampScore(v float64, maxScore int) float64
```

### 8.3 错误处理

- 统一错误类型 `AIGenerationError`：包含用户可读的 `Message` 与 `AvailablePlans`（可切换的模型配置列表）
- 前端根据 `available_plans` 提示「请尝试切换到其他模型配置后重试」
- 所有 AI 请求/响应均通过 `writeAILog` 落盘日志（含 baseURL、model、请求体、状态码），便于排障

---

## 9. 相关文件

### 后端（backend-go）

| 文件                                             | 说明                                                           |
| ------------------------------------------------ | -------------------------------------------------------------- |
| `services/ai_service.go`                         | 模型配置解析、LLM 调用封装、内容生成、SSE 事件、AI 日志        |
| `services/inspection_service.go`                 | 巡店分析（整表 / 单项）、vision content、技能 prompt、结果解析 |
| `services/inspection_template_service.go`        | 模板 AI 生成检查项（tools / 解析 / 兜底）                      |
| `services/inspection_recheck_service.go`         | 整改照片复核                                                   |
| `controllers/inspections_controller.go`          | 巡店 AI 分析接口（同步 / SSE）、分数量化                       |
| `controllers/inspection_tasks_controller.go`     | 整改提交、AI 复核、状态流转、通知                              |
| `controllers/inspection_templates_controller.go` | 模板 AI 生成接口                                               |
| `main.go`                                        | 路由注册                                                       |

### 前端（frontend/src）

| 文件                                     | 说明                                         |
| ---------------------------------------- | -------------------------------------------- |
| `pages/InspectionEdit.vue`               | AI 巡店分析面板 / 单项 AI 评分 / 结果回填    |
| `components/AITemplateItemGenerator.vue` | AI 智能添加检查项 Drawer（SSE 进度）         |
| `pages/InspectionTaskDetail.vue`         | AI 复核模型选择与状态流转                    |
| `pages/ContentCreate.vue`                | 文本 + 图片/视频多模态提交                   |
| `components/shared/ModelSelect.vue`      | 模型选择（含 `require-vision` 视觉能力校验） |
| `components/AttachmentInputArea.vue`     | 图片/视频附件上传区（多页面复用）            |
