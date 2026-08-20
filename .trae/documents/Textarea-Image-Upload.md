# 文本域与图片上传结合技术文档

> **版本**：v1.5 · **更新**：2026-08-20 · **核心文件**：`AttachmentInputArea.vue` / `ContentCreate.vue` / `InspectionEdit.vue` / `useFileUpload.ts` / `useUrlExtractor.ts` / `ai_service.go` / `ImageViewerModal.vue` / `ImageEditorModal.vue`
>
> **v1.5 变更**：
>
> 1. **后端章节修正为 Go 实现**：原 v1.4 的后端示例代码为 Python（`backend/app/...`），与项目实际后端（`backend-go`）不符，已整体重写为 Go（`services/ai_service.go` / `controllers/contents_controller.go`）。
> 2. **组件 Props/Emits 对齐实际代码**：修正 `AttachmentInputArea` 的 Props（`modelValue` 为文本、`fileList` 为附件数组）与 Emits（`update:modelValue` / `update:fileList`）。
> 3. **示例代码同步实际实现**：更新 ContentCreate 接入绑定（`v-model:file-list`）、ImageViewer 缩放参数、ImageEditor 画布尺寸与裁剪/压缩/导出逻辑。
> 4. **上传接口补充**：`POST /api/uploads` 由 `UploadsController.Upload` 实现（v1.4 标注「未找到显式定义」，现已确认）。
>
> **v1.4 变更**：
>
> 1. **卡片式附件区域**：附件展示从网格缩略图改为横向卡片条。每张卡片固定 68px 高、220px 宽、8px 圆角，左侧 52×52px 方形缩略图/视频首帧/文件图标，右侧文件名 + 扩展名/类型 · 大小，右上角固定删除按钮（hover 额外显示图片编辑按钮）。支持横向滚动，light/dark 主题均适配。
>
> **v1.3 变更**：
>
> 1. **上传模式双轨制**：组件支持 `uploadMode: 'auto' | 'manual'`（默认 **manual**）。手动模式下选中文件仅暂存（显示"待上传"徽标），由父级在表单提交前调用 `flushPending()` 批量上传后提交；`auto` 模式保留选中即上传的旧行为。
> 2. **多文件类型**：新增 `fileTypes: ('image' | 'video' | 'file')[]` 配置（默认 `['image']`），按类型生成 accept、大小限制（图片 10MB / 视频 50MB / 其它 20MB）与工具栏图标；类型化预览（图片缩略图 / 视频首帧 / 其它文件图标+文件名+大小）。
> 3. **ContentCreate 接入组件**：创作内容页改用公共组件（`theme="dark"` 深色主题、`enterBehavior="send"` 回车发送），保留模型选择 + 发送按钮视觉交互；组件在未传 `upload` 时仅收集文件，通过 `getPendingFiles()` 取 File 列表走 base64 直发 AI。
>
> **v1.2 变更**：将原 `FileAttachmentArea.vue` 重构为公共组件 `AttachmentInputArea.vue`，按「创作内容风格一体化输入区」的样式与交互，供素材编辑、模板编辑、巡店编辑、整改弹窗等场景复用；上传函数由业务页面通过 props 注入，文本/图片支持独立显隐开关，操作按钮改为 icon + tooltip 提示。

## 1. 概述

项目采用**"创作内容风格一体化输入区"**模式，将文本域（textarea）与图片/视频上传深度结合，提供类似 ChatGPT 的输入体验。图片不是插入到文本中，而是作为附件与文本一起提交给 AI 进行多模态分析。

### 1.1 设计原则

- **原生实现**：不依赖富文本编辑器（如 TinyMCE、Quill），使用 Arco Design 的 `a-textarea` + 原生 `<input type="file">`
- **多入口上传**：支持点击工具栏按钮、拖拽文件到输入区、粘贴剪贴板图片
- **URL 自动提取**：从文本中自动识别图片/文件 URL，转为附件并移除原文
- **图片压缩**：上传前使用 Canvas 压缩大尺寸图片，减少传输体积
- **Base64 编码**：文件以 base64 形式发送给 AI 接口，支持多模态分析
- **图片查看与编辑**：基于 Fabric.js 提供图片查看（缩放/拖拽）与编辑（裁剪/旋转/压缩）能力

### 1.2 核心页面

| 页面         | 文件                         | 特点                                                                                                                                                                       |
| ------------ | ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **创作内容** | `ContentCreate.vue`          | 使用 `AttachmentInputArea`（`theme="dark"` + `enter-behavior="send"`）：textarea + 拖拽 + 粘贴 + 工具栏上传 + URL 提取 + 多文件类型 + base64 直发（不传 `upload`，仅收集） |
| **素材编辑** | `InspectionMaterialEdit.vue` | 使用 `AttachmentInputArea`（手动模式）："检查标准与标准图"一体化输入区，提交前 `flushPending()`                                                                            |
| **模板编辑** | `InspectionTemplateEdit.vue` | 使用 `AttachmentInputArea`（手动模式）：每个检查项一个一体化输入区，v-for 内函数 ref + Map 收集，保存前统一 `flushPending()`                                               |
| **巡店编辑** | `InspectionEdit.vue`         | 使用 `AttachmentInputArea`（手动模式）：每个检查项"反馈问题 + 巡店图片"，支持 `show-textarea` / `show-images` 独立开关，保存前 `flushPending()`                            |
| **整改弹窗** | `InspectionTaskDetail.vue`   | 使用 `AttachmentInputArea`（手动模式）："整改说明 + 整改图片"，提交整改前 `flushPending()`                                                                                 |

---

## 2. 前端实现

### 2.1 一体化输入区结构

以 `ContentCreate.vue` 为例，输入区由三部分组成：

```
┌─────────────────────────────────────────┐
│  附件预览区（aia-cards）                 │
│  ├─ 横向卡片条，超出可横向滚动           │
│  ├─ 每张卡片：左侧 52×52 缩略图/视频首帧/文件图标 │
│  │             右侧文件名 + 扩展名/类型 · 大小   │
│  │             右上角固定删除按钮        │
│  │             hover 图片卡片显示编辑按钮 │
│  ├─ pending / uploading：缩略图叠加 spinner │
│  └─ 点击整张卡片查看（图片→ImageViewerModal │
│      视频→播放弹窗，文件→新窗口下载）    │
├─────────────────────────────────────────┤
│  文本输入区（a-textarea）                │
│  ├─ 自动高度调整（minRows: 3, maxRows: 5）│
│  ├─ 粘贴图片自动识别                    │
│  └─ 文本中 URL 自动提取                 │
├─────────────────────────────────────────┤
│  工具栏（aia-toolbar）                   │
│  ├─ 上传按钮（按 fileTypes 生成 accept） │
│  ├─ 数量计数 + 提示文案                  │
│  └─ 发送/业务自定义按钮                  │
└─────────────────────────────────────────┘
```

### 2.1.1 公共组件 AttachmentInputArea（v1.2 新增）

为统一「文本 + 图片」一体化输入体验，将原 `FileAttachmentArea.vue` 重构为公共组件 `AttachmentInputArea.vue`（`frontend/src/components/AttachmentInputArea.vue`）。组件自带 textarea、图片缩略图预览、工具栏与操作按钮，业务页面通过 **props（含上传函数）+ slot** 定制不同场景。

```
┌─────────────────────────────────────────────┐
│  #title 插槽（可选，标题说明区）              │
├─────────────────────────────────────────────┤
│  #empty 插槽 / 附件卡片区（aia-cards）        │
│  ├─ 横向卡片条，超出可滚动                    │
│  ├─ 每张卡片：左侧 52×52 缩略图              │
│  │             右侧文件名 + 扩展名/类型 · 大小 │
│  │             右上角固定删除按钮             │
│  │             hover 时额外显示图片编辑按钮    │
│  ├─ pending 项：缩略图叠加 spinner + "待上传"  │
│  └─ uploading 项：缩略图叠加 spinner + "上传中..."│
├─────────────────────────────────────────────┤
│  文本输入区（a-textarea，透明无边框）         │
│  ├─ auto-size 自动高度（minRows/maxRows）     │
│  ├─ 粘贴图片自动识别、URL 自动提取            │
│  └─ 连续换行守卫（Enter 防连续空行）          │
├─────────────────────────────────────────────┤
│  工具栏（aia-toolbar）                        │
│  ├─ #toolbar-left：上传按钮 + 计数 + 提示     │
│  └─ #toolbar-right：业务自定义按钮（发送等）   │
└─────────────────────────────────────────────┘
```

#### Props

| Prop                  | 类型                                           | 默认值                 | 说明                                                                                                                |
| --------------------- | ---------------------------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `modelValue`          | `string`                                       | `''`                   | 文本内容（`v-model` 绑定）                                                                                          |
| `fileList`            | `string[]`                                     | `undefined`            | 附件 URL 数组（`v-model:file-list` 绑定）                                                                           |
| `upload`              | `(file: File, onProgress?) => Promise<string>` | 可选（默认 undefined） | 文件上传函数，由业务页面注入；**不传时组件仅收集文件**（供 `getPendingFiles()` 取 File 走 base64 直发），不执行上传 |
| `uploadMode`          | `'auto' \| 'manual'`                           | `'manual'`             | 上传模式：`auto` 选中即上传；`manual` 暂存 pending，提交前父级调 `flushPending()` 批量上传                          |
| `fileTypes`           | `('image' \| 'video' \| 'file')[]`             | `['image']`            | 允许的文件类型集合，驱动 accept、大小限制与工具栏图标                                                               |
| `maxCount`            | `number`                                       | `0`（不限）            | 最大附件数量                                                                                                        |
| `disabled`            | `boolean`                                      | `false`                | 禁用态（隐藏操作按钮、上传按钮禁用）                                                                                |
| `bordered`            | `boolean`                                      | `true`                 | 是否显示卡片边框                                                                                                    |
| `compact`             | `boolean`                                      | `false`                | 紧凑模式（缩略图 60px）                                                                                             |
| `placeholder`         | `string`                                       | 内置默认               | textarea 占位符                                                                                                     |
| `hint`                | `string`                                       | `''`                   | 工具栏提示文字（空时按 fileTypes 自动生成）                                                                         |
| `maxLength`           | `number`                                       | `0`（不限）            | 文本最大字数（>0 时显示字数统计）                                                                                   |
| `showWordLimit`       | `boolean`                                      | `true`                 | 是否显示字数统计                                                                                                    |
| `minRows` / `maxRows` | `number`                                       | `2` / `5`              | textarea 自动高度行数范围                                                                                           |
| `showTextarea`        | `boolean`                                      | `true`                 | 独立开关：是否显示文本输入区                                                                                        |
| `showImages`          | `boolean`                                      | `true`                 | 独立开关：是否显示附件预览区                                                                                        |
| `extractUrls`         | `boolean`                                      | `true`                 | 是否从文本中自动提取图片/视频/文件 URL 并转为缩略图（内置逻辑）                                                     |
| `accept`              | `string`                                       | 由 `fileTypes` 生成    | 文件选择器 accept（可覆盖）                                                                                         |
| `theme`               | `'light' \| 'dark'`                            | `'light'`              | 主题：`dark` 深色背景适配创作内容页                                                                                 |
| `enterBehavior`       | `'newline' \| 'send'`                          | `'newline'`            | 回车行为：`send` 时 Enter 触发 `@enter`，Cmd/Ctrl+Enter 换行                                                        |

#### Emits

| Event               | 参数                                 | 说明                                        |
| ------------------- | ------------------------------------ | ------------------------------------------- |
| `update:modelValue` | `string`                             | 文本内容变化（含 URL 提取后的文本清理）     |
| `update:fileList`   | `string[]`                           | 附件数组变化（添加/删除/编辑替换/URL 提取） |
| `enter`             | `无`                                 | `enterBehavior="send"` 时回车触发           |
| `change`            | `{ total: number, pending: number }` | 附件数量变化（总数 / 待上传数）             |

#### Slots

| Slot             | 作用域 | 说明                                   |
| ---------------- | ------ | -------------------------------------- |
| `#title`         | 无     | 标题/说明区，渲染于输入区顶部          |
| `#empty`         | 无     | 空状态占位（无附件时）                 |
| `#toolbar-left`  | 无     | 工具栏左侧扩展（上传按钮之后）         |
| `#toolbar-right` | 无     | 工具栏右侧扩展（发送按钮、模型选择等） |

#### 上传函数注入

组件不直接请求后端，由业务页面通过 `upload` prop 注入（可复用 `useFileUpload.ts` 的 `uploadImageFile`）：

```typescript
// composables/useFileUpload.ts
export async function uploadImageFile(file: File): Promise<string> {
  const formData = new FormData()
  formData.append('file', file, file.name)
  const res = await api.post('/uploads', formData)
  return res.data?.url || ''
}
```

业务页面使用（手动模式示例）：

```vue
<script setup lang="ts">
import { ref } from 'vue'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import { uploadImageFile } from '@/composables/useFileUpload'

const files = ref<string[]>([])
const content = ref('')
const areaRef = ref<{ flushPending: () => Promise<boolean> }>()

async function handleSave() {
  if (areaRef.value && !(await areaRef.value.flushPending())) return
  // 此时 pending 文件已上传，files 中为正式 URL，继续提交表单
}
</script>

<template>
  <AttachmentInputArea
    ref="areaRef"
    v-model="content"
    v-model:file-list="files"
    :upload="uploadImageFile"
    :max-count="10"
    :max-length="300"
    :min-rows="3"
    :max-rows="8"
    placeholder="（选填）请输入反馈问题，300 字内"
  >
    <template #toolbar-right>
      <a-button size="mini" type="text" @click="handleAI">AI 生成</a-button>
    </template>
  </AttachmentInputArea>
</template>
```

> **注意**：`v-model` 绑定文本内容，`v-model:file-list` 绑定附件 URL 数组，两者相互独立。URL 提取逻辑已内置（`extractUrls` 默认开启），父级无需再自行提取。

#### 组件暴露方法（defineExpose）

| 方法                | 说明                                                                                                                     |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `handlePaste(e)`    | 处理剪贴板粘贴图片，返回 `boolean`（是否处理了图片粘贴）                                                                 |
| `pickFiles()`       | 触发隐藏的文件选择器                                                                                                     |
| `addFiles(files)`   | 添加文件（校验 → 压缩 → 按 uploadMode 上传或暂存）                                                                       |
| `flushPending()`    | **手动模式**：批量上传全部 pending 文件（upload 未传时直接跳过并返回 `true`），全部成功返回 `true`，任一失败返回 `false` |
| `getPendingFiles()` | **手动模式**：返回待上传的 `{ file, type, name, size }[]`（供 base64 直发场景取 File）                                   |

### 2.2 核心类型定义（组件内部）

`AttachmentInputArea.vue` 内部使用「附件项 + 待上传项」两类模型：

```typescript
type AttachmentFileType = 'image' | 'video' | 'file'

// 待上传项（手动模式暂存，upload 未传时也用于收集 File）
interface PendingItem {
  key: string // 唯一标识
  file: File
  type: AttachmentFileType
  name: string
  size: number
}

// 预览附件项（displayItems computed 合并 images 与 pending）
interface DisplayItem {
  key: string
  type: AttachmentFileType
  name: string
  size?: number
  url: string // 已上传 URL
  pending?: boolean // true 表示未上传（待上传徽标）
  file?: File // pending 时持有原始 File
}
```

- `fileList`（`v-model:file-list` 绑定的附件 URL 数组）与 `pendingItems`（待上传项）合并为 `displayItems` 统一渲染
- 按类型分类的 computed：`imageItems` / `videoItems` / `fileItems`，驱动查看器、视频播放弹窗与文件项展示

### 2.3 文件上传触发

> **v1.3 说明**：以下 2.3 ~ 2.6 节展示的上传触发、文件校验、拖拽、粘贴逻辑原为 `ContentCreate.vue` 独立实现；现已被 `AttachmentInputArea.vue` 公共组件内聚（`pickFiles()` / `addFiles()` / 拖拽 / 粘贴）。各使用页面通过组件复用，不再各自实现。

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

### 2.7 URL 自动提取（useUrlExtractor）

从文本中识别图片/视频/文件 URL，转为附件并从文本中移除。该逻辑已抽取至 `useUrlExtractor.ts` 并被组件（`extractUrls` 默认开启）与 ContentCreate 复用：

```typescript
// composables/useUrlExtractor.ts
const IMAGE_URL_PATTERN = /\.(png|jpe?g|gif|webp|avif|svg|bmp|ico)(\?.*)?$/i
const VIDEO_URL_PATTERN = /\.(mp4|mov|m4v|webm|avi|mkv)(\?.*)?$/i
const FILE_URL_EXT_PATTERN = /\.(pdf|docx?|xlsx?|pptx?|zip|rar|7z|txt|md|csv|json)(\?.*)?$/i

// 根据 URL 扩展名推断类型：'image' | 'video' | 'file' | null
function typeFromUrl(url: string): AttachmentFileType | null

// 从文本中提取指定类型的 URL 列表（去重、清洗尾部标点）
function extractUrlsFromText(text: string, types: AttachmentFileType[]): string[]

// 从文本中移除指定类型的 URL
function cleanUrlsFromText(text: string, types: AttachmentFileType[]): string

// 从 URL 提取文件名（解码）
function urlFileName(url: string): string
```

组件内置的提取流程（`extractUrls` 开启时）：

```typescript
const urlTypes = props.fileTypes // 按配置的 fileTypes 过滤 URL
const urls = extractUrlsFromText(text, urlTypes)
const cleaned = cleanUrlsFromText(text, urlTypes)
// 新 URL 追加到 images，text 同步清理
```

> 兼容保留：`extractImageUrlsFromText(text)` 为仅提取图片 URL 的旧签名。

### 2.7.1 手动上传模式与 flushPending（v1.3）

`uploadMode: 'manual'`（默认）下，选中文件**只进入 `pendingItems` 暂存**，不发起上传；父级在表单提交前调用 `flushPending()` 将全部 pending 文件批量上传为正式 URL，再提交表单，避免「选择即上传」造成孤儿文件。

组件内部关键逻辑：

```typescript
// 手动模式：addFiles 只暂存，不调用 upload
function addFiles(files: File[]) {
  for (const file of accept) {
    pendingItems.value.push({
      key: genKey(),
      file,
      type: typeFromFile(file),
      name: file.name,
      size: file.size,
    })
  }
  emitChange()
}

// 提交前批量上传：全部成功返回 true，任一失败返回 false（父级中断提交）
async function flushPending(): Promise<boolean> {
  const targets = pendingItems.value
  if (targets.length === 0) return true
  if (!props.upload) {
    pendingItems.value = [] // 未配置上传函数：仅收集场景，直接清空暂存
    emitChange()
    return true
  }
  const results = await Promise.all(
    targets.map(async (item) => {
      const url = await props.upload(item.file)
      if (!url) return false
      images.value.push(url)
      URL.revokeObjectURL(previewUrl(item))
      return true
    }),
  )
  if (results.every(Boolean)) pendingItems.value = []
  emitChange()
  return results.every(Boolean)
}

// 供 base64 直发场景取 File 列表
function getPendingFiles(): PendingItem[] {
  return pendingItems.value
}
```

**父级接入模式**（v-for 内多组件场景）：

```typescript
// 模板：:ref="(el) => setAreaRef(item, el)" —— 函数 ref + Map 收集（比字符串 ref 更可靠）
const areaRefs = new Map<string, { flushPending: () => Promise<boolean> }>()
function setAreaRef(item: { id: string }, el: unknown) {
  if (el) areaRefs.set(item.id, el as { flushPending: () => Promise<boolean> })
  else areaRefs.delete(item.id)
}

// 提交表单前统一 flush
async function handleSave() {
  for (const ref of areaRefs.values()) {
    if (!(await ref.flushPending())) return // 任一上传失败则中断提交
  }
  // 继续提交表单（此时 v-model 数组已含全部正式 URL）
}
```

### 2.7.2 多文件类型配置（v1.3）

`fileTypes: ('image' | 'video' | 'file')[]` 数组（默认 `['image']`）驱动组件三类能力：

| fileTypes   | 生成 accept                       | 大小限制       | 工具栏图标             | 预览样式                   |
| ----------- | --------------------------------- | -------------- | ---------------------- | -------------------------- |
| `['image']` | `image/jpeg,image/png,image/webp` | 图片 10MB      | IconImage + 提示       | 图片缩略图（点击查看大图） |
| `['video']` | `video/*`                         | 视频 50MB      | IconVideoCamera + 提示 | 视频首帧（点击播放弹窗）   |
| `['file']`  | 通用文件 MIME（audio/pdf/doc 等） | 其它 20MB      | IconPlus + 提示        | 文件图标 + 文件名 + 大小   |
| 多类型混合  | 上述 MIME 拼接                    | 按类型分别限制 | IconPlus + 提示        | 类型化混合渲染             |

```typescript
// 类型判定与校验
function typeFromFile(file: File): AttachmentFileType {
  if (file.type.startsWith('image/')) return 'image'
  if (file.type.startsWith('video/')) return 'video'
  return 'file'
}

const FILE_TYPE_LIMITS: Record<AttachmentFileType, number> = {
  image: 10 * 1024 * 1024, // 10MB
  video: 50 * 1024 * 1024, // 50MB
  file: 20 * 1024 * 1024, // 20MB
}
```

类型化预览（统一以横向卡片展示）：

- **图片**：`<img>` 缩略图置于卡片左侧，点击整张卡片打开 `ImageViewerModal`（滚轮缩放、拖拽平移、多图翻页）
- **视频**：`<video preload="metadata" muted>` 首帧置于卡片左侧，点击整张卡片打开视频播放弹窗
- **其它文件**：卡片左侧显示 `IconFile`，点击整张卡片在新窗口打开/下载
- 所有卡片右上角始终显示删除按钮；hover 时图片卡片额外显示编辑按钮

### 2.7.3 卡片式附件区域（v1.4）

附件展示区由网格缩略图改为横向卡片条，核心结构与样式：

```vue
<div class="aia-cards">
  <div
    v-for="(item, index) in displayItems"
    :key="item.key"
    class="aia-card"
    :class="{ 'aia-card--pending': item.pending, 'aia-card--uploading': isUploading(item.key) }"
    @click="openViewer(index)"
  >
    <div class="aia-card__thumb">
      <!-- image / video / file icon -->
      <a-spin v-if="item.pending || isUploading(item.key)" :size="16" />
    </div>
    <div class="aia-card__info">
      <span class="aia-card__name">{{ item.name }}</span>
      <span class="aia-card__meta">{{ cardMeta(item) }}</span>
    </div>
    <div class="aia-card__actions">
      <button v-if="item.type === 'image'" @click.stop="openEditor(index)">编辑</button>
    </div>
    <button class="aia-card__delete" @click.stop="removeItem(index)">删除</button>
  </div>
</div>
```

**尺寸规范**：

| 元素                         | 尺寸 / 样式                                            |
| ---------------------------- | ------------------------------------------------------ |
| 卡片容器 `.aia-card`         | 宽 220px，高 68px，圆角 8px，间距 12px                 |
| 缩略图 `.aia-card__thumb`    | 52×52px，圆角 6px                                      |
| 文件名 `.aia-card__name`     | 13px，单行截断，深灰/浅色（dark 为 #f2f2f7）           |
| 元信息 `.aia-card__meta`     | 11px，格式为 `扩展名 · 大小` 或 `待上传` / `上传中...` |
| 编辑按钮 `.aia-card__action` | 20×20px，圆形白底，阴影，默认隐藏，hover 卡片时显示    |
| 删除按钮 `.aia-card__delete` | 20×20px，圆形白底，阴影，始终可见                      |
| 滚动容器 `.aia-cards`        | `flex-wrap: nowrap; overflow-x: auto;`                 |

**状态样式**：

- **正常已上传**：白色背景（light）/ #3a3a3c（dark），显示扩展名 + 文件大小
- **pending（手动模式）**：显示蓝色 `待上传` 文案 + 缩略图 spinner
- **uploading（auto 模式）**：显示蓝色 `上传中...` 文案 + 缩略图 spinner
- **hover**：卡片背景微亮，操作按钮淡入

**交互**：

- 点击整张卡片触发查看（图片 → 查看器 / 视频 → 播放弹窗 / 文件 → 新窗口）
- 删除按钮始终可见（图中为右上角固定 ×）；编辑按钮仅在图片卡片 hover 时出现
- 横向排列，超出容器宽度时底部出现滚动条

### 2.7.4 ContentCreate 接入与 base64 直发（v1.3）

创作内容页不再独立实现输入区，改用公共组件（视觉保持）：

```vue
<AttachmentInputArea
  ref="createAreaRef"
  v-model="promptText"
  v-model:file-list="fileUrls"
  theme="dark"
  :file-types="['image', 'video']"
  :max-count="MAX_UPLOAD_FILES"
  :enter-behavior="'send'"
  placeholder="输入创作需求，使用 #标签 添加关键词，例如：写一篇小红书文案 #穿搭 #夏季"
  @enter="generate"
  @change="onAreaChange"
>
  <template #toolbar-right>
    <!-- 模型选择 a-select + 发送按钮（视觉与旧实现一致） -->
  </template>
</AttachmentInputArea>
```

**关键差异：不传 `upload`** —— 组件仅收集文件（`pendingItems`），AI 直发时通过 `getPendingFiles()` 取 File 列表，与文本中提取的 URL 一起编码为 base64：

```typescript
interface PendingFileItem {
  file: File
  type: string
  name: string
  size: number
}

// pending 文件 → base64（图片先压缩）
async function fileToBase64(file: File): Promise<{ data: string; mime_type: string }>

// URL 链接 → fetch → blob → base64
async function fetchUrlToBase64(url: string): Promise<{ data: string; mime_type: string } | null>

async function buildFilesPayload(): Promise<{ data: string; mime_type: string }[] | undefined> {
  const pendingFiles = createAreaRef.value?.getPendingFiles() ?? []
  const urls = fileUrls.value
  if (pendingFiles.length + urls.length === 0) return undefined
  const tasks = [
    ...pendingFiles.map((p) => fileToBase64(p.file)),
    ...urls.map((url) => fetchUrlToBase64(url)),
  ]
  const results = await Promise.all(tasks)
  return results.filter((r): r is { data: string; mime_type: string } => r !== null)
}
```

**交互保留**：

- `enterBehavior="send"`：Enter 直接触发 `@enter="generate"`（发送），Cmd/Ctrl+Enter 换行；快捷键提示文案由 `hint` 传入
- `theme="dark"`：深色背景（#2c2c2e）与文字，适配创作页暗色卡片
- `#toolbar-right` slot 承载模型选择器 + 发送按钮，布局与旧实现一致
- 模型不支持文件上传时（`!store.selectedModelSupportsFiles`），`generate()` 提前拦截并提示

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

> **v1.3 说明**：以下代码为旧版 `ContentCreate.vue` 独立实现（遍历 `uploadedItems`）。当前创作页已改用公共组件，文件收集统一走 `buildFilesPayload()`（组件 `getPendingFiles()` + URL 转 base64，见 2.7.3），逻辑等价：本地文件先压缩再编码，URL 通过 fetch 转 base64，合并为 `{ data, mime_type }[]` 随 SSE 请求发送。

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

### 2.11 图片查看与编辑（Fabric.js）

附件预览区为每张图片提供「查看」与「编辑」两个入口。编辑能力基于 [Fabric.js](https://fabricjs.com/) 实现，支持裁剪、旋转、压缩。

> **默认不编辑**：图片上传后**默认直接使用原始图片**。用户可通过「编辑」按钮进入编辑器。图片编辑为可选操作，非提交环节的必经步骤。

#### 2.11.1 功能说明

- **Fabric.js 集成**：利用 Canvas 技术实现前端图片裁剪框选、角度旋转，并根据调整后的预览图实时导出数据。
- **裁剪（Crop）**：通过交互式 `Rect` 矩形框定义裁剪区域，支持拖拽调节框选范围，裁剪结果按区域导出后重新加载到画布。
- **旋转（Rotate）**：支持 90° 步进旋转（左转 / 右转），直接旋转画布中的图片对象。
- **压缩（Compress）**：提供质量滑块（Quality Slider，0.3 ~ 1），导出前将画布背景置白，通过 `canvas.toBlob` 按质量压缩为 JPEG。

#### 2.11.2 交互流程

1. **上传/粘贴**：图片以横向卡片形式进入附件预览区。
2. **点击卡片**：打开 `ImageViewerModal` 查看大图（滚轮缩放、拖拽平移、多图翻页）。
3. **hover 悬浮**：图片卡片 hover 时显示「编辑」图标；删除按钮始终可见。
4. **编辑入口**：点击「编辑」，打开 `ImageEditorModal`。
5. **画布调整**：在 Fabric.js 画布内进行裁剪/旋转操作。
6. **实时预览**：编辑器提供即时预览效果。
7. **确认应用**：点击确认，导出编辑后的图片数据流（Blob/Base64）并覆盖本地附件原图。
8. **取消/关闭**：丢弃所有临时编辑，保留原图不变。

#### 2.11.3 图片查看器

查看器无需 Fabric.js，使用 CSS `transform: scale() + translate()` 实现缩放与拖拽，性能更优：

```typescript
const scale = ref(1)
const offset = ref({ x: 0, y: 0 })
const isDragging = ref(false)
let startPos = { x: 0, y: 0 }

function onWheel(e: WheelEvent) {
  const delta = e.deltaY > 0 ? -0.12 : 0.12
  scale.value = Math.min(4, Math.max(0.25, scale.value + delta))
}

function zoomBy(delta: number) {
  scale.value = Math.min(4, Math.max(0.25, scale.value + delta))
}

function onDragStart(e: MouseEvent) {
  isDragging.value = true
  startPos = { x: e.clientX, y: e.clientY }
}

function onDragMove(e: MouseEvent) {
  if (!isDragging.value) return
  offset.value = {
    x: offset.value.x + (e.clientX - startPos.x),
    y: offset.value.y + (e.clientY - startPos.y),
  }
  startPos = { x: e.clientX, y: e.clientY }
}

function onDragEnd() {
  isDragging.value = false
}
```

模板绑定（查看器支持多图 `items` + `index`，通过 `prev` / `next` 事件翻页）：

```vue
<img
  :src="currentItem.url"
  :style="{
    transform: `translate(${offset.x}px, ${offset.y}px) scale(${scale})`,
    transition: isDragging ? 'none' : 'transform 0.15s ease',
  }"
  @wheel.prevent="onWheel"
  @mousedown="onDragStart"
  @mousemove="onDragMove"
  @mouseup="onDragEnd"
  @mouseleave="onDragEnd"
/>
```

#### 2.11.4 图片编辑器（Fabric.js 画布初始化）

```typescript
import { Canvas, Image as FabricImage, Rect } from 'fabric'

const CANVAS_W = 820
const CANVAS_H = 540

const canvasRef = ref<HTMLCanvasElement | null>(null)
let canvas: Canvas | null = null
let image: FabricImage | null = null
let cropRect: Rect | null = null
const isCropMode = ref(false)
const isExporting = ref(false)
const quality = ref(0.85)

// 组件打开时懒初始化画布（仅创建一次），并加载图片
watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    nextTick(async () => {
      if (!canvasRef.value) return
      if (!canvas) {
        canvas = new Canvas(canvasRef.value, {
          width: CANVAS_W,
          height: CANVAS_H,
          selection: false,
          preserveObjectStacking: true,
        })
      }
      await loadImage(props.url)
    })
  },
)

onBeforeUnmount(() => {
  canvas?.dispose()
  canvas = null
})

async function loadImage(url: string) {
  if (!canvas) return
  canvas.clear()
  image = null
  cropRect = null
  isCropMode.value = false
  try {
    const img = await FabricImage.fromURL(url)
    const width = img.width || 1
    const height = img.height || 1
    // 等比缩放到画布内（留 32px 边距，且不超过原图尺寸）
    const scale = Math.min((CANVAS_W - 32) / width, (CANVAS_H - 32) / height, 1)
    img.scale(scale)
    img.set({
      originX: 'center',
      originY: 'center',
      left: CANVAS_W / 2,
      top: CANVAS_H / 2,
    })
    canvas.add(img)
    canvas.setActiveObject(img)
    image = img
    canvas.renderAll()
  } catch (err) {
    Message.error('图片加载失败，无法进入编辑模式')
  }
}
```

> **编辑器能力**：工具栏提供「裁剪 / 旋转（左 90°、右 90°）/ 重置 / 质量滑块（0.3 ~ 1）/ 确认导出」。组件由 `AttachmentInputArea` 以 `defineAsyncComponent` 异步加载，Fabric.js 仅在首次打开编辑弹窗时按需进入包体。
>
> 组件 Props：`visible` / `url` / `name`；Emits：`close` / `confirm({ blob, name })`。

#### 2.11.5 裁剪

Fabric.js v6 无内置裁剪工具，采用「`Rect` 框选 + `toDataURL` 区域导出」方案。裁剪进入独立模式（`isCropMode`，状态声明见 2.11.4），支持取消与应用：

```typescript
function startCrop() {
  if (!canvas || !image || cropRect) return
  const b = image.getBoundingRect()
  cropRect = new Rect({
    left: b.left,
    top: b.top,
    width: b.width,
    height: b.height,
    fill: 'rgba(0, 122, 255, 0.06)',
    stroke: '#007AFF',
    strokeWidth: 1.5,
    strokeUniform: true,
    strokeDashArray: [6, 4],
    cornerColor: '#007AFF',
    cornerSize: 12,
    transparentCorners: false,
    lockRotation: true,
    borderColor: '#007AFF',
  })
  canvas.add(cropRect)
  canvas.setActiveObject(cropRect)
  isCropMode.value = true
  canvas.renderAll()
}

function cancelCrop() {
  if (!canvas || !cropRect) return
  canvas.remove(cropRect)
  cropRect = null
  isCropMode.value = false
  canvas.renderAll()
}

async function applyCrop() {
  if (!canvas || !cropRect) return
  const b = cropRect.getBoundingRect()
  canvas.remove(cropRect)
  cropRect = null
  isCropMode.value = false
  // 按裁剪区域导出并重新加载到画布
  const dataUrl = exportDataUrl({ left: b.left, top: b.top, width: b.width, height: b.height })
  if (dataUrl) await loadImage(dataUrl)
}
```

> **注意**：`cropRect` 仅在与 `image` 同坐标系下使用 `getBoundingRect()` 计算实际裁剪区域；若图片有旋转，需先应用旋转再裁剪（或重置裁剪框），避免区域错位。

#### 2.11.6 旋转

支持 90° 步进旋转（左转 / 右转），直接旋转画布中的图片对象：

```typescript
function rotate(degrees: number) {
  if (!image) return
  image.rotate((image.angle || 0) + degrees)
  canvas?.renderAll()
}

// 使用示例：左转 90° / 右转 90°
rotate(-90)
rotate(90)
```

> **注意**：旋转后 `getBoundingRect()` 会随角度变化，因此旋转后建议先「重置」（重新加载原图）再进入裁剪，避免裁剪区域与旋转后的图片错位。

#### 2.11.7 压缩

通过 `canvas.toBlob` 的 `quality` 参数控制压缩质量（质量滑块范围 0.3 ~ 1，步进 0.05），导出前将画布背景切换为白色以得到干净的 JPEG：

```typescript
const quality = ref(0.85)
const isExporting = ref(false)

async function confirm() {
  if (!canvas) return
  isExporting.value = true
  try {
    const prevBg = canvas.backgroundColor
    canvas.backgroundColor = '#ffffff'
    canvas.renderAll()
    const blob = await canvas.toBlob({
      format: 'jpeg',
      quality: quality.value, // 0.3 ~ 1
      multiplier: 1,
    })
    canvas.backgroundColor = prevBg
    canvas.renderAll()
    if (!blob) return
    const baseName = props.name.replace(/\.[^.]+$/, '') || 'image'
    emit('confirm', { blob, name: `${baseName}_edited.jpg` })
    close()
  } finally {
    isExporting.value = false
  }
}
```

> 裁剪区域导出复用同一 `exportDataUrl()`：临时设置白色背景 → `canvas.toDataURL({ format: 'jpeg', quality, multiplier: 1, ...region })` → 恢复背景。

#### 2.11.8 与上传流程集成

图片编辑入口与流程已内聚至 `AttachmentInputArea.vue`：编辑按钮仅在 `item.type === 'image'` 且（`pending` 或已配置 `upload`）时显示；编辑确认后由组件内部 `handleEditorConfirm` 处理替换，父级无需介入：

```typescript
async function handleEditorConfirm(result: { blob: Blob; name: string }) {
  const item = displayItems.value[editorIndex.value]
  if (!item || item.type !== 'image') return
  const file = new File([result.blob], result.name, { type: result.blob.type || 'image/jpeg' })

  // 分支一：待上传图片 —— 直接替换 PendingItem.file，随后 flushPending()/getPendingFiles() 使用编辑后文件
  if (item.pending && item.file) {
    const i = pendingItems.value.findIndex((p) => p.file === item.file)
    if (i >= 0) {
      const next = [...pendingItems.value]
      next[i] = { ...next[i], file, name: file.name, size: file.size }
      pendingItems.value = next
    }
    Message.success('图片编辑已保存')
    emitChange()
    return
  }

  // 分支二：已上传图片 —— 用编辑结果重新上传，替换 fileList 中的 URL
  if (!props.upload) return
  const url = await props.upload(file)
  if (url) {
    const i = fileList.value.indexOf(item.url || '')
    if (i >= 0) {
      const next = [...fileList.value]
      next[i] = url
      fileList.value = next
    }
    urlSizeMap.set(url, file.size)
    Message.success('图片编辑已保存')
  }
  emitChange()
}
```

> **默认不编辑场景**：用户从未点击「编辑」时，图片保持上传时的原始文件。创作页 `buildFilesPayload()` 直接对原始 File 执行上传前压缩与 base64 编码，完全绕过 Fabric.js 编辑器。
>
> **AI 内容生成路径**：`ContentCreate.vue` 未传 `upload`（仅收集文件），`getPendingFiles()` 返回的编辑后 File 随 SSE 请求发送给 AI 接口。

---

## 3. 后端实现（Go）

> **说明**：项目后端为 **Go**（`backend-go`），非 Python。本节基于 `services/ai_service.go` 与 `controllers/contents_controller.go` 的实际实现整理。

### 3.1 类型定义

```go
// backend-go/services/ai_service.go

type UploadedFile struct {
	Data     string `json:"data"`      // base64 编码的文件数据
	MimeType string `json:"mime_type"` // MIME 类型（如 image/jpeg, video/mp4）
}

// backend-go/controllers/contents_controller.go

type aiGenerateStreamRequest struct {
	Topic     string                  `json:"topic"`      // 创作主题/需求
	Platforms []string                `json:"platforms"`  // 目标平台列表
	Style     string                  `json:"style"`      // 风格
	Keywords  []string                `json:"keywords"`   // 关键词
	PlanID    string                  `json:"plan_id"`    // 模型配置 ID
	ModelID   string                  `json:"model_id"`   // 模型 ID
	Files     []services.UploadedFile `json:"files"`      // 可选的文件列表
}
```

### 3.2 AI 服务处理（GenerateContentStream）

```go
// backend-go/services/ai_service.go

func GenerateContentStream(topic, platform, style string, keywords []string, planID, modelID string, files []UploadedFile) (<-chan AIStreamEvent, error) {
	events := make(chan AIStreamEvent)
	go func() {
		defer close(events)
		apiKey, baseURL, model, planName, _, err := ResolveModelConfig(planID, modelID)
		if err != nil { events <- aiError(err); return }

		// 1. 统计文件类型并记录日志
		imgCount, vidCount := 0, 0
		for _, f := range files {
			if strings.HasPrefix(f.MimeType, "image/") { imgCount++ }
			if strings.HasPrefix(f.MimeType, "video/") { vidCount++ }
		}
		if imgCount > 0 || vidCount > 0 {
			parts := make([]string, 0, 2)
			if imgCount > 0 { parts = append(parts, fmt.Sprintf("%d 张图片", imgCount)) }
			if vidCount > 0 { parts = append(parts, fmt.Sprintf("%d 个视频", vidCount)) }
			events <- AIStreamEvent{Event: "log", Level: "info",
				Message: fmt.Sprintf("附带 %s（多模态分析）", strings.Join(parts, "、"))}
		}

		// 2. 构建消息：有文件时 user 为多模态数组（buildVisionContent），否则纯文本
		messages, tools, toolChoice := buildContentMessages(topic, platform, style, keywords, files)
		userContent := messages[len(messages)-1].Content
		if len(files) > 0 {
			// userContent 已为 [{type:"text"}, {type:"image_url"|"video_url"}...]
		}

		// 3. 调用 LLM（tool_choice 强制结构化输出）
		resp, err := callChatCompletionsWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 8000)
		// 400 且包含 tool_choice 字样时降级为 auto 重试
		if resp.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(body), "tool_choice") {
			resp, err = callChatCompletionsWithTools(apiKey, baseURL, model, messages, tools, nil, 8000)
		}

		// 4. 流式解析 tool_calls delta，回退解析 parseContentVariantsArgs / parseContentVariantArgs
		for ... { events <- AIStreamEvent{Event: "chunk", Text: delta.Content} }
		events <- AIStreamEvent{Event: "done", Variant: v, Model: model}
	}()
	return events, nil
}
```

> 流式事件统一通过 `AIStreamEvent` channel 返回：`log` / `chunk` / `done` / `error` / `result` 等，控制器负责写入 `text/event-stream` 响应。多平台生成在控制器 `AIGenerateStream` 中循环调用，最后发送 `complete` 事件。

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

> **说明**：该接口由 `backend-go/controllers/uploads_controller.go` 的 `UploadsController.Upload` 实现（路由 `main.go: web.Router("/api/uploads", uploads, "post:Upload")`）。上传目录为 `services.GetUploadDir()`，并通过 `web.SetStaticPath("/uploads", uploadDir)` 提供静态访问。

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
  "store_id": "门店ID",
  "template_id": "检查表模板ID",
  "plan_id": "模型配置ID",
  "model_id": "模型ID",
  "photos": [
    {
      "data": "base64编码的图片数据",
      "mime_type": "image/jpeg"
    }
  ],
  "keywords": "巡店关键词/现场描述",
  "skills": [
    {
      "item_id": "检查项ID",
      "name": "门头形象",
      "standard": "检查标准",
      "standard_images": ["/uploads/standard.jpg"],
      "score_type": "score | pass_fail",
      "max_score": 10,
      "score_options": [{ "score": 0, "label": "0分" }]
    }
  ],
  "response_schema": {}
}

响应：SSE 流（log / chunk / result / error / complete，result 事件携带 scores / issues / suggestion / summary 等）
```

---

## 5. 使用指南

### 5.1 创作内容页面（AttachmentInputArea）

1. **输入创作需求**：在输入区 textarea 中输入文本，支持 `#标签` 提取关键词
2. **添加图片/视频/文件**：
   - 点击工具栏上传按钮（多类型为 IconPlus）选择文件
   - 拖拽文件到输入区
   - 从剪贴板粘贴图片（Ctrl+V）
   - 在文本中粘贴图片/视频/文件 URL，自动提取为附件并清理文本
3. **查看附件**：点击横向卡片打开对应预览（图片 → 大图 / 视频 → 播放弹窗 / 文件 → 新窗口）
4. **编辑/删除**：右上角固定删除按钮；图片卡片 hover 时额外显示编辑按钮
5. **回车发送**：Enter 直接触发 AI 生成，Cmd/Ctrl+Enter 换行
6. **选择模型与平台**：工具栏右侧模型选择器切换模型，勾选目标平台
7. **点击发送**：本地文件与 URL 统一转为 base64（`buildFilesPayload()`）随 SSE 请求发送，AI 基于文本与附件生成多平台适配内容

### 5.2 巡店编辑页面（AttachmentInputArea，手动模式）

1. **填写检查项反馈**：每个检查项的"反馈问题 + 巡店图片"一体化输入区内填写
2. **添加巡店图片**：
   - 点击工具栏 "+" 按钮选择文件，文件以卡片形式进入"待上传"状态（**不会立即上传**）
   - 拖拽图片到输入区、粘贴剪贴板图片
   - 在文本中粘贴图片 URL，自动提取为卡片并清理文本
3. **提交表单**：点击保存/开始 AI 巡店时，页面先对每个输入区调用 `flushPending()` 批量上传"待上传"文件（任一失败则中断），再提交表单/发起分析
4. **查看/编辑/删除**：点击卡片查看图片；hover 时图片卡片显示编辑按钮；右上角固定删除按钮
5. **独立开关**：检查项按配置仅显示文本区或仅显示图片区（`show-textarea` / `show-images`）

### 5.3 素材编辑 / 模板编辑页面（AttachmentInputArea，手动模式）

1. **填写检查标准**：在"检查标准与标准图"一体化输入区（素材）或检查项输入区（模板）填写标准文本
2. **添加标准图**：点击 "+"、拖拽、粘贴图片，或粘贴图片 URL 自动识别；文件先以卡片形式进入"待上传"状态
3. **保存触发上传**：点击保存时统一 `flushPending()` 上传"待上传"文件后再提交
4. **数量限制**：素材页最多 5 张（`max-count="5"`）、模板页按模板配置限制

### 5.4 整改弹窗（AttachmentInputArea，手动模式）

1. **填写整改说明**：在整改弹窗的"整改说明 + 整改图片"一体化输入区内填写（选填）
2. **添加整改图片**：点击 "+"、拖拽、粘贴图片，或粘贴图片 URL 自动识别；文件先以卡片形式进入"待上传"状态
3. **提交整改触发上传**：点击提交整改时先 `flushPending()` 上传"待上传"文件，再提交整改
4. **必填校验**：根据问题项的整改要求，弹窗提交前校验整改说明/图片是否已提供

---

## 6. 最佳实践

### 6.1 图片处理

- **上传前压缩**：宽或高超过 1600px 的图片自动压缩，质量 0.8
- **编辑后压缩**：Fabric.js 编辑器导出时按 `quality`（0.3 ~ 1）滑块控制压缩质量
- **大小限制**（按 `fileTypes` 类型分别限制）：图片 10MB，视频 50MB，其它文件 20MB
- **数量限制**：由 `max-count` 控制（创作内容 10、素材 5、模板按配置、巡店 10、整改 9）
- **格式支持**：
  - 图片：png, jpg, jpeg, gif, webp, avif, svg, bmp, ico
  - 视频：mp4, mov, m4v, webm, avi, mkv
  - 其它文件：pdf, doc, docx, xls, xlsx, ppt, pptx, zip, rar, 7z, txt, md, csv, json 等
- **编辑输出**：裁剪/旋转/压缩统一导出为 JPEG 格式（`image/jpeg`），文件名追加 `_edited` 后缀
- **多文件类型**：`fileTypes` 数组驱动 accept 生成与工具栏图标（单图片 IconImage / 单视频 IconVideoCamera / 多类型与其它文件 IconPlus）

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

已抽取的公共能力：

- `useFileUpload`：文件上传逻辑（校验、压缩、上传函数、base64 编码）
- `useUrlExtractor`：URL 自动提取逻辑（图片/视频/文件类型推断、文本清理）
- `AttachmentInputArea`：**一体化输入区公共组件**（textarea + 工具栏 + 类型化预览 + 查看/编辑/删除），支持 `uploadMode`（auto/manual）、`fileTypes` 多文件类型、`theme`（light/dark）、`enterBehavior`（newline/send）；上传函数由业务页面注入，已应用于创作内容、素材编辑、模板编辑、巡店编辑、整改弹窗五个场景
- `ImageEditorModal`：Fabric.js 编辑器（画布初始化、裁剪、旋转、压缩导出），由 `AttachmentInputArea` 以异步组件方式按需加载

### 7.2 功能增强

- **进度显示**：上传/压缩过程中显示进度条
- **断点续传**：大文件分片上传
- **图片滤镜**：亮度、对比度、饱和度调节（Fabric.js 内置 `filters` 能力）
- **裁剪预设比例**：1:1、4:3、16:9 等常用比例裁剪框
- **拖拽排序**：调整附件顺序
- **批量操作**：全选、批量删除、批量压缩

---

## 8. 相关文件

| 文件                                              | 说明                                                                                        |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `frontend/src/components/AttachmentInputArea.vue` | **公共一体化输入区组件**（textarea + 类型化预览 + 工具栏 + 查看/编辑/删除 + 手动/自动上传） |
| `frontend/src/composables/useFileUpload.ts`       | 文件上传工具（校验/压缩/uploadImageFile/base64）                                            |
| `frontend/src/composables/useUrlExtractor.ts`     | URL 提取与文本清理工具（图片/视频/文件）                                                    |
| `frontend/src/pages/ContentCreate.vue`            | 创作内容页（使用组件，dark 主题 + 回车发送 + base64 直发）                                  |
| `frontend/src/pages/InspectionEdit.vue`           | 巡店编辑页（检查项级输入，保存前 flushPending）                                             |
| `frontend/src/pages/InspectionMaterialEdit.vue`   | 素材标准图编辑（保存前 flushPending）                                                       |
| `frontend/src/pages/InspectionTemplateEdit.vue`   | 模板编辑（v-for 内 Map refs，保存前 flushPending）                                          |
| `frontend/src/pages/InspectionTaskDetail.vue`     | 整改弹窗（提交前 flushPending）                                                             |
| `frontend/src/components/ImageViewerModal.vue`    | 图片查看器（缩放/拖拽/翻页）                                                                |
| `frontend/src/components/ImageEditorModal.vue`    | 图片编辑器（Fabric.js 裁剪/旋转/压缩）                                                      |
| `backend-go/services/ai_service.go`               | AI 服务处理（GenerateContentStream、buildVisionContent、模型配置解析）                      |
| `backend-go/controllers/contents_controller.go`   | 内容 AI 生成接口（AIGenerateStream，SSE 输出）                                              |
| `backend-go/controllers/uploads_controller.go`    | 文件上传接口（Upload）                                                                      |
| `frontend/vite.config.ts`                         | Vite 代理配置                                                                               |
