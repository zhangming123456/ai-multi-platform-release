# 文本域与图片上传结合技术文档

> **版本**：v1.0 · **更新**：2026-07-31 · **核心文件**：`ContentCreate.vue` / `InspectionEdit.vue` / `ai_service.py`

## 1. 概述

项目采用**"创作内容风格一体化输入区"**模式，将文本域（textarea）与图片/视频上传深度结合，提供类似 ChatGPT 的输入体验。图片不是插入到文本中，而是作为附件与文本一起提交给 AI 进行多模态分析。

### 1.1 设计原则

- **原生实现**：不依赖富文本编辑器（如 TinyMCE、Quill），使用 Arco Design 的 `a-textarea` + 原生 `<input type="file">`
- **多入口上传**：支持点击工具栏按钮、拖拽文件到输入区、粘贴剪贴板图片
- **URL 自动提取**：从文本中自动识别图片/文件 URL，转为附件并移除原文
- **图片压缩**：上传前使用 Canvas 压缩大尺寸图片，减少传输体积
- **Base64 编码**：文件以 base64 形式发送给 AI 接口，支持多模态分析

### 1.2 核心页面

| 页面           | 文件                         | 特点                                                                                |
| -------------- | ---------------------------- | ----------------------------------------------------------------------------------- |
| **创作内容**   | `ContentCreate.vue`          | 最完整实现：textarea + 拖拽 + 粘贴 + 工具栏上传 + URL 提取 + 图片压缩 + base64 编码 |
| **巡店编辑**   | `InspectionEdit.vue`         | 每个检查项独立的 textarea + 图片上传，支持粘贴/URL提取/AI分析                       |
| **素材标准图** | `InspectionMaterialEdit.vue` | textarea 填写检查标准 + `a-upload` 上传标准图                                       |
| **模板编辑**   | `InspectionTemplateEdit.vue` | 多个检查项各自有 textarea + `a-upload`                                              |

---

## 2. 前端实现

### 2.1 一体化输入区结构

以 `ContentCreate.vue` 为例，输入区由三部分组成：

```
┌─────────────────────────────────────────┐
│  已上传文件预览区（chat-attachments）     │
│  ├─ 图片/视频缩略图 + 文件名            │
│  └─ 删除按钮                            │
├─────────────────────────────────────────┤
│  文本输入区（a-textarea）                │
│  ├─ 自动高度调整（minRows: 3, maxRows: 5）│
│  ├─ 粘贴图片自动识别                    │
│  └─ 文本中 URL 自动提取                 │
├─────────────────────────────────────────┤
│  工具栏（chat-toolbar）                  │
│  ├─ 上传文件按钮（accept: image/*,video/*）│
│  ├─ 图片按钮（accept: image/*）          │
│  ├─ 视频按钮（accept: video/*）          │
│  ├─ 文件大小提示                         │
│  └─ 发送按钮                             │
└─────────────────────────────────────────┘
```

### 2.2 核心类型定义

```typescript
// 上传项基础类型
interface UploadedItemBase {
  uid: string // 唯一标识
  name: string // 文件名
  url: string // 预览 URL（ObjectURL 或原始 URL）
  status: 'done' | 'init'
}

// 本地文件上传项（含 File 对象）
interface UploadedFileItem extends UploadedItemBase {
  kind: 'file'
  file: File
}

// URL 链接项（从文本中提取）
interface UploadedLinkItem extends UploadedItemBase {
  kind: 'link'
  url: string // 原始 URL
}

// 联合类型
type UploadedItem = UploadedFileItem | UploadedLinkItem
```

### 2.3 文件上传触发

```typescript
const acceptType = ref('image/*,video/*')
const fileInputRef = ref<HTMLInputElement | null>(null)

function triggerUpload(type: 'all' | 'image' | 'video') {
  acceptType.value = type === 'image' ? 'image/*' : type === 'video' ? 'video/*' : 'image/*,video/*'
  nextTick(() => {
    fileInputRef.value?.click()
  })
}

function onFileInputChange(e: Event) {
  const files = Array.from((e.target as HTMLInputElement).files || [])
  addFiles(files)
  // 清空 input，允许重复选择同一文件
  ;(e.target as HTMLInputElement).value = ''
}
```

### 2.4 文件校验与添加

```typescript
const MAX_IMAGE_SIZE = 10 * 1024 * 1024 // 10MB
const MAX_VIDEO_SIZE = 50 * 1024 * 1024 // 50MB
const MAX_FILES = 10

function addFiles(files: File[]) {
  const remaining = MAX_FILES - uploadedItems.value.length
  if (remaining <= 0) {
    Message.warning(`最多上传 ${MAX_FILES} 个文件`)
    return
  }
  const accept = files.slice(0, remaining)
  let hasOversize = false
  for (const file of accept) {
    const isImage = file.type.startsWith('image/')
    const isVideo = file.type.startsWith('video/')
    if (!isImage && !isVideo) continue
    if (isImage && file.size > MAX_IMAGE_SIZE) {
      hasOversize = true
      continue
    }
    if (isVideo && file.size > MAX_VIDEO_SIZE) {
      hasOversize = true
      continue
    }
    uploadedItems.value.push({
      kind: 'file',
      uid: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      name: file.name || (isImage ? 'image' : 'video'),
      url: URL.createObjectURL(file),
      status: 'init',
      file,
    })
  }
  if (hasOversize) Message.warning('图片不能超过 10MB，视频不能超过 50MB')
}
```

### 2.5 拖拽上传

```typescript
const isDragging = ref(false)
let dragDepth = 0

function onDragEnter(e: DragEvent) {
  if (!e.dataTransfer?.types.includes('Files')) return
  dragDepth++
  isDragging.value = true
}

function onDragLeave() {
  dragDepth--
  if (dragDepth <= 0) {
    dragDepth = 0
    isDragging.value = false
  }
}

function onDragOver(e: DragEvent) {
  if (!e.dataTransfer?.types.includes('Files')) return
  e.preventDefault()
}

function onDrop(e: DragEvent) {
  dragDepth = 0
  isDragging.value = false
  if (!e.dataTransfer?.types.includes('Files')) return
  e.preventDefault()
  const files = Array.from(e.dataTransfer.files)
  addFiles(files)
}
```

### 2.6 粘贴图片

```typescript
function handleTextareaPaste(e: ClipboardEvent) {
  const files = Array.from(e.clipboardData?.files || [])
  const imageFiles = files.filter((f) => f.type.startsWith('image/'))
  if (imageFiles.length > 0) {
    e.preventDefault()
    const remaining = MAX_FILES - uploadedItems.value.length
    if (remaining <= 0) {
      Message.warning(`最多上传 ${MAX_FILES} 个文件`)
      return
    }
    const accept = imageFiles.slice(0, remaining)
    let hasOversize = false
    for (const file of accept) {
      if (file.size > MAX_IMAGE_SIZE) {
        hasOversize = true
        continue
      }
      uploadedItems.value.push({
        kind: 'file',
        uid: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        name: file.name || 'pasted-image',
        url: URL.createObjectURL(file),
        status: 'init',
        file,
      })
    }
    if (hasOversize) Message.warning('单张图片大小不能超过 10MB')
    return
  }
  // 粘贴纯文本时，延迟提取 URL
  nextTick(() => {
    extractFileLinksFromText()
  })
}
```

### 2.7 URL 自动提取

从文本中识别图片/文件 URL，转为附件并从文本中移除：

```typescript
const FILE_URL_PATTERN = /https?:\/\/[^\s<>"'（）()，。；！？、]+/gi
const IMAGE_URL_PATTERN = /\.(png|jpe?g|gif|webp|avif|svg|bmp|ico)(\?.*)?$/i
const VIDEO_URL_PATTERN = /\.(mp4|mov|m4v|webm|avi|mkv)(\?.*)?$/i

function extractFileLinksFromText() {
  const text = promptText.value
  const urls = Array.from(new Set(text.match(FILE_URL_PATTERN) || []))
    .map((u) => u.replace(/[.,;:!?，。；：！？、]+$/, ''))
    .filter((u) => IMAGE_URL_PATTERN.test(u) || VIDEO_URL_PATTERN.test(u))
  if (urls.length === 0) return
  const existing = new Set(uploadedItems.value.map((i) => (i.kind === 'link' ? i.url : '')))
  const remaining = MAX_FILES - uploadedItems.value.length
  let addedCount = 0
  for (const url of urls) {
    if (existing.has(url)) continue
    if (addedCount >= remaining) break
    const path = url.split(/[?#]/)[0]
    let name = path.split('/').pop() || url
    try {
      name = decodeURIComponent(name)
    } catch {
      /* keep original */
    }
    uploadedItems.value.push({
      kind: 'link',
      uid: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      name,
      url,
      status: 'done',
    })
    existing.add(url)
    addedCount++
  }
  // 从文本中移除已提取的 URL
  const cleaned = text.replace(FILE_URL_PATTERN, (match) => {
    const url = match.replace(/[.,;:!?，。；：！？、]+$/, '')
    return IMAGE_URL_PATTERN.test(url) || VIDEO_URL_PATTERN.test(url) ? '' : match
  })
  promptText.value = cleaned.replace(/ +/g, ' ').replace(/\n{3,}/g, '\n\n')
}
```

### 2.8 图片压缩

使用 Canvas 压缩大尺寸图片（宽或高超过 1600px）：

```typescript
async function compressImage(file: File, maxSize = 1600, quality = 0.8): Promise<File> {
  return new Promise((resolve) => {
    const img = new Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      URL.revokeObjectURL(url)
      let { width, height } = img
      if (width <= maxSize && height <= maxSize) {
        resolve(file)
        return
      }
      if (width > height) {
        height = Math.round((height * maxSize) / width)
        width = maxSize
      } else {
        width = Math.round((width * maxSize) / height)
        height = maxSize
      }
      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        resolve(file)
        return
      }
      ctx.drawImage(img, 0, 0, width, height)
      canvas.toBlob(
        (blob) => {
          if (!blob) {
            resolve(file)
            return
          }
          const compressed = new File([blob], file.name, { type: 'image/jpeg' })
          resolve(compressed)
        },
        'image/jpeg',
        quality,
      )
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      resolve(file)
    }
    img.src = url
  })
}
```

### 2.9 Base64 编码

将文件转为 base64 格式，用于发送给 AI 接口：

```typescript
function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      const commaIndex = result.indexOf(',')
      resolve(commaIndex >= 0 ? result.slice(commaIndex + 1) : result)
    }
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}
```

### 2.10 发送请求

将文本和文件一起发送给 AI 接口：

```typescript
async function startAiGeneration() {
  // 1. 收集本地文件并压缩
  const filesPayload: { data: string; mime_type: string }[] = []
  for (const item of uploadedItems.value) {
    if (item.kind === 'file') {
      const fileToUse =
        item.file.type.startsWith('image/') && item.file.size > 500 * 1024
          ? await compressImage(item.file)
          : item.file
      const base64 = await fileToBase64(fileToUse)
      filesPayload.push({ data: base64, mime_type: item.file.type })
    }
  }

  // 2. 构建请求体
  const body = {
    topic: promptText.value,
    platforms: selectedPlatforms.value,
    files: filesPayload.length > 0 ? filesPayload : undefined,
  }

  // 3. 发送 SSE 流式请求
  const response = await fetch('/api/contents/ai-generate-stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  })

  // 4. 处理 SSE 流
  const reader = response.body!.getReader()
  const decoder = new TextDecoder()
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    const text = decoder.decode(value)
    // 解析 SSE 事件...
  }
}
```

---

## 3. 后端实现

### 3.1 类型定义

```python
# backend/app/schemas/content.py

class UploadedFile(BaseModel):
    data: str          # base64 编码的文件数据
    mime_type: str     # MIME 类型（如 image/jpeg, video/mp4）

class AIGenerateStreamRequest(BaseModel):
    topic: str                              # 创作主题/需求
    platforms: list[str]                    # 目标平台列表
    style: Optional[str] = None             # 风格
    keywords: Optional[list[str]] = None    # 关键词
    plan_id: Optional[str] = None           # 模型配置 ID
    model_id: Optional[str] = None          # 模型 ID
    files: Optional[List[UploadedFile]] = None  # 可选的文件列表
```

### 3.2 AI 服务处理

```python
# backend/app/services/ai_service.py

async def ai_generate_stream(..., files: Optional[List[UploadedFile]] = None, ...):
    # 1. 统计文件类型
    img_count = sum(1 for f in files if f.mime_type.startswith("image/"))
    vid_count = sum(1 for f in files if f.mime_type.startswith("video/"))
    if img_count or vid_count:
        parts = []
        if img_count: parts.append(f"{img_count} 张图片")
        if vid_count: parts.append(f"{vid_count} 个视频")
        yield {"event": "log", "level": "info", "message": f"附带 {'、'.join(parts)}（多模态分析）"}

    # 2. 构建多模态消息
    user_content: list[dict] = [{"type": "text", "text": prompt}]
    for f in files or []:
        if f.mime_type.startswith("video/"):
            user_content.append({
                "type": "video_url",
                "video_url": {"url": f"data:{f.mime_type};base64,{f.data}"},
            })
        else:
            user_content.append({
                "type": "image_url",
                "image_url": {"url": f"data:{f.mime_type};base64,{f.data}"},
            })

    # 3. 调用 LLM API
    stream = await client.chat.completions.create(
        model=model,
        messages=[
            {"role": "system", "content": "你是一个专业的社交媒体内容创作助手..."},
            {"role": "user", "content": user_content if files else prompt},
        ],
        temperature=0.8,
        max_tokens=4000,
        stream=True,
    )

    # 4. 流式返回
    async for chunk in stream:
        if chunk.choices and chunk.choices[0].delta.content:
            yield {"event": "chunk", "text": chunk.choices[0].delta.content}
```

---

## 4. API 接口

### 4.1 文件上传接口

```
POST /api/uploads
Content-Type: multipart/form-data

参数：
- file: File  # 上传的文件

响应：
{
  "url": "https://example.com/uploads/xxx.jpg"
}
```

> **注意**：该接口在当前代码库中未找到显式定义，可能由外部服务（如 Nginx、对象存储网关）处理。

### 4.2 AI 生成流式接口

```
POST /api/contents/ai-generate-stream
Content-Type: application/json
Authorization: Bearer <token>

请求体：
{
  "topic": "创作需求文本",
  "platforms": ["wechat", "xiaohongshu"],
  "files": [
    {
      "data": "base64编码的文件数据",
      "mime_type": "image/jpeg"
    }
  ]
}

响应：SSE 流
event: log
data: {"level": "info", "message": "附带 2 张图片（多模态分析）"}

event: chunk
data: {"text": "生成的内容片段"}

event: done
data: {"variant": {...}, "model": "xxx", "plan_name": "xxx"}

event: error
data: {"message": "错误信息", "available_plans": [...]}
```

### 4.3 AI 巡店分析流式接口

```
POST /api/inspections/ai-analyze-stream
Content-Type: application/json
Authorization: Bearer <token>

请求体：
{
  "inspection_id": "xxx",
  "photos": [
    {
      "data": "base64编码的图片数据",
      "mime_type": "image/jpeg"
    }
  ],
  "skills": [...]
}

响应：SSE 流（同上）
```

---

## 5. 使用指南

### 5.1 创作内容页面

1. **输入创作需求**：在 textarea 中输入文本
2. **添加图片/视频**：
   - 点击工具栏"图片"或"视频"按钮选择文件
   - 拖拽文件到输入区
   - 从剪贴板粘贴图片（Ctrl+V）
   - 在文本中粘贴图片 URL，自动提取为附件
3. **选择目标平台**：勾选需要发布的平台
4. **点击发送**：AI 将基于文本和图片生成多平台适配内容

### 5.2 巡店编辑页面

1. **填写检查项反馈**：在每个检查项的 textarea 中输入备注
2. **上传巡店图片**：
   - 点击"图片"按钮选择文件
   - 粘贴剪贴板图片
   - 粘贴图片 URL 自动提取
3. **AI 巡店分析**：上传图片后，AI 自动分析并生成评分和建议

---

## 6. 最佳实践

### 6.1 图片处理

- **压缩策略**：宽或高超过 1600px 的图片自动压缩，质量 0.8
- **大小限制**：图片 10MB，视频 50MB
- **数量限制**：最多 10 个文件
- **格式支持**：
  - 图片：png, jpg, jpeg, gif, webp, avif, svg, bmp, ico
  - 视频：mp4, mov, m4v, webm, avi, mkv

### 6.2 URL 提取

- **自动识别**：文本中的 http/https URL，根据扩展名判断类型
- **去重处理**：已存在的 URL 不会重复添加
- **文本清理**：提取后从文本中移除 URL，避免重复

### 6.3 错误处理

- **文件超限**：提示"图片不能超过 10MB，视频不能超过 50MB"
- **数量超限**：提示"最多上传 10 个文件"
- **AI 调用失败**：提示错误信息并展示可用模型列表

### 6.4 性能优化

- **ObjectURL 释放**：删除文件时调用 `URL.revokeObjectURL` 释放内存
- **延迟提取**：粘贴文本时使用 `nextTick` 延迟提取 URL，避免阻塞
- **流式响应**：AI 生成使用 SSE 流式返回，实时展示生成进度

---

## 7. 扩展建议

### 7.1 可复用组件

建议将以下逻辑抽取为公共组件/工具函数：

- `useFileUpload`：文件上传逻辑（校验、压缩、base64 编码）
- `useUrlExtractor`：URL 自动提取逻辑
- `FileAttachmentArea`：一体化输入区组件（textarea + 工具栏 + 预览）

### 7.2 功能增强

- **进度显示**：上传/压缩过程中显示进度条
- **断点续传**：大文件分片上传
- **图片编辑**：裁剪、旋转、滤镜
- **拖拽排序**：调整附件顺序
- **批量操作**：全选、批量删除

---

## 8. 相关文件

| 文件                                            | 说明                       |
| ----------------------------------------------- | -------------------------- |
| `frontend/src/pages/ContentCreate.vue`          | 创作内容页（最完整实现）   |
| `frontend/src/pages/InspectionEdit.vue`         | 巡店编辑页（检查项级输入） |
| `frontend/src/pages/InspectionMaterialEdit.vue` | 素材标准图编辑             |
| `frontend/src/pages/InspectionTemplateEdit.vue` | 模板编辑                   |
| `backend/app/schemas/content.py`                | 后端类型定义               |
| `backend/app/services/ai_service.py`            | AI 服务处理                |
| `frontend/vite.config.ts`                       | Vite 代理配置              |
