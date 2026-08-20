# Vue 图片编辑器 双引擎（原生 Canvas / Fabric.js）可选架构方案

> **版本**：v2.1 · **技术栈**：Vue 3 + TypeScript + 原生 Canvas 2D + Fabric.js（可选增强）
>
> **方案定位**：一套轻量化、高扩展、可商用的图片编辑框架。通过「双引擎可选架构」在**原生自研 Canvas 引擎**与 **Fabric.js 增强引擎**之间按需切换：业务层 API 统一、UI 面板共用，底层渲染内核可配置替换，兼顾**包体积轻量**与**交互能力强大**两个目标。
>
> **v2.1 变更**：基于「双引擎（原生 Canvas / Fabric.js）可选架构方案」整体重构 —— 新增 `engine/base/BaseEngine` 抽象基类、`engine/lightweight`（轻量引擎）、`engine/powerful`（Fabric 增强引擎）、`config/engine.config.ts` 全局配置、`engine/index.ts` 动态切换工厂；统一图层模型升级为 `scaleX / scaleY`；新增能力差异矩阵与差异抹平策略；历史记录基于业务图层快照，与引擎解耦。
>
> **版本号修正**：方案正文提及的 Fabric 版本为 `5.x`，本项目实际依赖为 **`fabric@^7.4.0`**（见 `frontend/package.json`），下文均以 7.x 为准（v6 起改为命名导出 `import { Canvas, Image as FabricImage, Rect } from 'fabric'`）。Fabric 6+ 已内置 TS 类型，**无需安装 `@types/fabric`**。

---

## 目录

1. [方案概述](#1-方案概述)
2. [技术栈选型](#2-技术栈选型)
3. [双引擎架构设计](#3-双引擎架构设计)
4. [项目目录规范（双引擎企业级标准）](#4-项目目录规范双引擎企业级标准)
5. [核心数据模型设计](#5-核心数据模型设计)
6. [双引擎能力差异矩阵](#6-双引擎能力差异矩阵)
7. [能力差异抹平策略](#7-能力差异抹平策略)
8. [双向适配器设计（FabricAdapter）](#8-双向适配器设计fabricadapter)
9. [历史记录与状态设计](#9-历史记录与状态设计)
10. [引擎工厂与按需加载](#10-引擎工厂与按需加载)
11. [落地实施步骤](#11-落地实施步骤)
12. [方案总结](#12-方案总结)
13. [附录：Fabric.js 进阶技术细节](#13-附录fabricjs-进阶技术细节)
14. [项目落地现状与相关文件](#14-项目落地现状与相关文件)

---

## 1. 方案概述

本方案基于 **Vue3 + TypeScript** 构建一套**双引擎可选**的图片编辑框架。框架采用**四层架构、图层驱动、状态中心化、功能插件化**设计，支持图片裁剪、旋转、缩放、涂鸦、文字、贴纸、滤镜、马赛克、撤销重做、导出等主流图片编辑能力。

核心设计理念：

- **双引擎可选**：原生 Canvas（轻量，零依赖）与 Fabric.js（增强，交互强大）按需切换，业务代码零感知
- **图层化设计**：图片 / 文字 / 涂鸦 / 贴纸全部为独立图层，支持层级、显隐、变换
- **插件式功能**：新增编辑功能无需改动核心引擎
- **引擎与业务解耦**：上层业务读写统一 `BaseLayer` 模型，引擎层通过适配器双向同步
- **完整历史快照**：基于业务图层做快照，撤销 / 重做与具体引擎无关
- **完整 TS 类型**：适合大型项目、团队协作、组件库封装

---

## 2. 技术栈选型

| 类别       | 选型                                           | 说明                                      |
| ---------- | ---------------------------------------------- | ----------------------------------------- |
| 框架核心   | Vue 3 + TypeScript                             | Composition API + `<script setup>`        |
| 状态管理   | Composables 模块化 + 局部响应式                | 轻量替代 Pinia，按模块拆分                |
| 渲染内核 A | 原生 Canvas 2D（轻量引擎，默认）               | 完全脱离 Vue，纯 TS 实现，零额外依赖      |
| 渲染内核 B | Fabric.js（`^7.4.0`，增强引擎，可选）          | 复杂自由编辑 / 海报设计场景的第二渲染内核 |
| UI 组件    | Element Plus / NaiveUI / Arco Design（可自选） | 面板、工具栏、弹窗                        |
| 文件处理   | 原生 File / Blob / URL                         | 图片导入与导出                            |
| 构建工具   | Vite                                           | 产物按需分包，Fabric 动态懒加载           |

> **Fabric.js 版本说明**：本项目实际使用 **`fabric@^7.4.0`**。v6 起 API 全面改为命名导出（`import { Canvas, Image as FabricImage, Rect } from 'fabric'`），且已内置 TypeScript 类型定义，无需再安装 `@types/fabric`。本方案所有示例代码均基于 7.x 语法。

---

## 3. 双引擎架构设计

### 3.1 四层架构（严格单向依赖）

整体采用**自上而下分层，严格单向依赖**：上层调用下层，下层不依赖上层。

```
┌─────────────────────────────────────────────────────────────┐
│                    1. 视图层（UI）                            │
│   编辑器入口 ImageEditor.vue · 工具栏 · 功能面板 · 画布容器    │
│   —— 只负责页面渲染、用户交互、面板展示（双引擎通用）         │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                 2. 业务逻辑层（Composables Hooks）            │
│   useEditor / useHistory / useCrop / useDraw / useText /     │
│   useFilter / useSticker / useFile                           │
│   —— Vue 业务胶水层：状态管理、用户操作、调用引擎（API 统一） │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│               3. 核心引擎层（可切换的双引擎内核）             │
│   BaseEngine（抽象基类）                                      │
│   ├── lightweight/ CanvasEngine + LayerManager + render/*    │
│   └── powerful/   FabricEngine + FabricAdapter + render/*    │
│   —— 画布初始化、图层管理、矩阵变换、渲染、导出               │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│               4. 工具基础层（Utils + Types）                  │
│   全局类型定义 · 坐标转换 · 图片处理 · 文件解析 · 通用工具     │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 引擎层职责划分

- **`engine/base/BaseEngine.ts`**：引擎抽象基类，定义统一接口规范，双引擎都必须实现
- **`engine/lightweight/`**：原生 Canvas 轻量引擎（默认），零额外依赖、包体积最小
- **`engine/powerful/`**：Fabric 增强引擎，交互能力完整，`FabricAdapter` 负责业务图层 ↔ Fabric 对象双向适配
- **`engine/index.ts`**：引擎统一入口，`createEngine()` 工厂根据配置动态创建对应引擎实例

### 3.3 核心能力架构图

```
用户操作 → UI组件 → useEditorHook → createEngine() → 具体引擎实例 → 画布渲染

用户操作 → 生成业务图层快照 → 存入历史栈 → 支持撤销重做（与引擎无关）

编辑完成 → 引擎统一接口 exportToBlob → 前端下载 / 回传业务接口
```

---

## 4. 项目目录规范（双引擎企业级标准）

```
src/components/ImageEditor
├── ImageEditor.vue          # 编辑器入口（双引擎自动适配，业务方唯一入口）
├── config/
│   └── engine.config.ts     # 引擎全局配置（切换开关、默认引擎）
├── components/              # 所有 UI 面板、工具栏（双引擎通用）
│   ├── EditorCanvas.vue     # 画布容器
│   ├── Toolbar.vue          # 顶部工具栏
│   ├── panel-crop.vue       # 裁剪面板
│   ├── panel-filter.vue     # 滤镜面板
│   ├── panel-text.vue       # 文字面板
│   ├── panel-draw.vue       # 涂鸦面板
│   └── panel-sticker.vue    # 贴纸面板
├── composables/             # 所有业务钩子（双引擎通用，API 统一）
│   ├── useEditor.ts         # 编辑器总控
│   ├── useHistory.ts        # 撤销重做
│   ├── useCrop.ts           # 裁剪
│   ├── useDraw.ts           # 涂鸦
│   ├── useText.ts           # 文字
│   ├── useFilter.ts         # 滤镜
│   ├── useSticker.ts        # 贴纸
│   └── useFile.ts           # 文件导入导出
├── engine/                  # 双引擎核心内核（无 Vue 依赖）
│   ├── base/
│   │   └── BaseEngine.ts    # 引擎抽象基类（统一接口规范）
│   ├── lightweight/         # 轻量自研 Canvas 引擎
│   │   ├── CanvasEngine.ts  # 引擎主类
│   │   ├── LayerManager.ts  # 图层管理器
│   │   └── render/          # 各图层渲染器
│   │       ├── renderImage.ts
│   │       ├── renderText.ts
│   │       ├── renderDraw.ts
│   │       └── renderSticker.ts
│   ├── powerful/            # 增强 Fabric 引擎
│   │   ├── FabricEngine.ts  # Fabric 引擎主类
│   │   ├── FabricAdapter.ts # 双向适配器：业务图层 ↔ Fabric 对象
│   │   ├── FabricEvent.ts   # 画布事件统一封装（选中、拖拽、变换）
│   │   └── render/          # 适配 Fabric 的各元素创建器
│   │       ├── createFabricImage.ts
│   │       ├── createFabricText.ts
│   │       ├── createFabricPath.ts
│   │       └── createFabricGroup.ts
│   └── index.ts             # 引擎统一入口、动态切换工厂 createEngine()
├── types/
│   └── index.ts             # 全局类型定义
└── utils/
    ├── image.ts             # 图片处理
    ├── canvas.ts            # canvas 工具
    └── coordinate.ts        # 坐标转换
```

### 4.1 引擎全局配置（engine.config.ts）

```typescript
// config/engine.config.ts
// ENGINE_TYPE：引擎类型
//   'lightweight' —— 原生 Canvas 轻量引擎（默认，零依赖、包体积最小）
//   'powerful'    —— Fabric.js 增强引擎（交互能力完整，按需动态加载）
export const ENGINE_TYPE = import.meta.env.VITE_EDITOR_ENGINE || 'lightweight'
```

- 默认轻量引擎：简单裁剪、压缩、滤镜场景直接使用，**不打包任何 Fabric 代码**
- 通过环境变量 `VITE_EDITOR_ENGINE=powerful` 切换为 Fabric 增强引擎，**业务代码零改动**

---

## 5. 核心数据模型设计

### 5.1 编辑模式枚举

定义当前编辑器交互状态：

```typescript
export enum EditorMode {
  Select = 'select', // 选择
  Crop = 'crop', // 裁剪
  Draw = 'draw', // 涂鸦
  Text = 'text', // 文字
  Sticker = 'sticker', // 贴纸
  Filter = 'filter', // 滤镜
}
```

### 5.2 图层统一模型（核心）

所有编辑元素统一图层结构，保证渲染、变换、层级逻辑统一。**这是业务层与引擎层之间的唯一数据契约**：

```typescript
export type LayerType = 'image' | 'text' | 'draw' | 'sticker'

export interface BaseLayer {
  // 基础持久化属性
  id: string
  type: LayerType
  visible: boolean
  zIndex: number
  locked: boolean

  // 空间变更属性
  x: number
  y: number
  width: number
  height: number
  rotate: number
  scaleX: number // v2.1：独立 X 轴缩放
  scaleY: number // v2.1：独立 Y 轴缩放

  // 类型专属属性
  image?: {
    src: string // 图片资源（base64 / URL）
    naturalWidth: number
    naturalHeight: number
  }
  text?: {
    content: string // 文字内容
    fontSize: number
    color: string
    bold: boolean
  }
  draw?: {
    points: number[][] // 路径点位
    strokeColor: string
    strokeWidth: number
    isEraser: boolean
  }
  sticker?: {
    url: string // 贴纸地址
  }
}
```

> **设计优势**：业务层只感知 `BaseLayer`，不感知具体引擎。无论底层是原生 Canvas 还是 Fabric，图层数据契约完全一致；新增元素类型只需扩展 `LayerType` 与对应渲染器，**无需修改引擎核心与业务层**。

### 5.3 编辑器全局状态

```typescript
export interface EditorState {
  canvasWidth: number // 画布尺寸
  canvasHeight: number
  mode: EditorMode // 当前编辑模式
  activeLayerId: string | null // 激活图层
  layers: BaseLayer[] // 图层数组
  history: HistoryState // 历史记录（undo/redo 栈）
  viewport: {
    // 画布缩放与平移
    scale: number
    offsetX: number
    offsetY: number
  }
}
```

---

## 6. 双引擎能力差异矩阵

| 能力项                     | 原生 Canvas 轻量引擎（lightweight） | Fabric.js 增强引擎（powerful） |
| -------------------------- | ----------------------------------- | ------------------------------ |
| 图片裁剪 / 旋转 / 压缩     | ✅ 支持                             | ✅ 支持                        |
| 滤镜 / 马赛克（像素级）    | ✅ 支持（ImageData 共用算法）       | ✅ 支持（ImageData 共用算法）  |
| 元素自游拖拽 / 缩放 / 旋转 | ✅ 基础支持                         | ✅ 完整支持（内置控制点）      |
| 多选 / 群组 / 图层锁定     | ❌ 不支持                           | ✅ 原生支持                    |
| 画布缩放 / 平移            | ✅ 基础支持                         | ✅ 原生支持（viewport）        |
| SVG 素材导入 / 矢量编辑    | ❌ 不支持                           | ✅ 原生支持                    |
| 框选批量操作               | ❌ 不支持                           | ✅ 原生支持                    |
| 精准锚点变换 / 等比例锁定  | ❌ 不支持                           | ✅ 原生支持                    |
| 元素双击编辑 / 悬浮高亮    | ❌ 不支持                           | ✅ 原生支持                    |
| 包体积                     | 极小（零额外依赖）                  | 增加（gzip 约 35KB，按需加载） |

> **共同点**：像素级处理（滤镜 / 马赛克）两引擎共用同一套 `ImageData` 像素算法，保证效果完全一致、无行为差异。

---

## 7. 能力差异抹平策略

双引擎能力不同，但**上层 UI 与业务必须表现一致**。采用「能力探测 + 面板按能力渲染」策略：

### 7.1 引擎能力描述

每个引擎实现 `BaseEngine.getCapabilities()`，返回自身能力清单：

```typescript
export interface EngineCapabilities {
  multiSelect: boolean // 多选
  group: boolean // 群组
  lockLayer: boolean // 图层锁定
  viewportZoom: boolean // 画布缩放平移
  svgImport: boolean // SVG 导入
  freeTransform: boolean // 自由变换
  doubleClickEdit: boolean // 双击编辑
}

// lightweight 引擎返回：{ multiSelect: false, group: false, ... }
// powerful 引擎返回：{ multiSelect: true, group: true, ... }
```

### 7.2 面板按能力渲染

```vue
<!-- 工具栏：仅当引擎支持时展示对应按钮 -->
<template>
  <ToolbarItem v-if="caps.group" label="群组" @click="groupActiveLayers" />
  <ToolbarItem v-if="caps.multiSelect" label="框选" @click="enableBoxSelect" />
  <ToolbarItem v-if="caps.svgImport" label="导入SVG" @click="importSvg" />
</template>
```

- **基础能力**（裁剪 / 旋转 / 压缩 / 滤镜 / 涂鸦 / 文字 / 贴纸）：双引擎统一支持，UI 完全一致
- **高阶能力**（多选 / 群组 / SVG / 自由变换）：仅增强引擎开放，轻量引擎**自动隐藏**对应面板与入口
- 切换引擎时由 `useEditor` 依据 `getCapabilities()` 动态计算可用功能，**避免出现「按钮存在但点了无效」的体验落差**

---

## 8. 双向适配器设计（FabricAdapter）

Fabric 引擎的核心是 `FabricAdapter`：实现**业务 `BaseLayer` 模型 ↔ Fabric 内置对象**的双向转换，彻底隔离两个世界。

```
业务层（BaseLayer[]）  ⇄  FabricAdapter  ⇄  Fabric 对象（Image / Text / Path / Group）
        │                                                      │
        │  更新业务状态（位置/尺寸/旋转/层级/显隐）              │  原生交互（拖拽/缩放/旋转/选中）
        └────────────── 双向同步 ◀─────────────┘
```

### 8.1 业务层 → Fabric 对象（渲染方向）

```typescript
import { Image as FabricImage, Text as FabricText, Path as FabricPath } from 'fabric'

export function toFabricObject(layer: BaseLayer): FabricImage | FabricText | FabricPath {
  switch (layer.type) {
    case 'image':
      return new FabricImage(document.getElementById(layer.image!.src) as HTMLImageElement, {
        left: layer.x,
        top: layer.y,
        width: layer.width,
        height: layer.height,
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
      })
    case 'text':
      return new FabricText(layer.text!.content, {
        left: layer.x,
        top: layer.y,
        fontSize: layer.text!.fontSize,
        fill: layer.text!.color,
        fontWeight: layer.text!.bold ? 'bold' : 'normal',
      })
    // ...draw / sticker 同理
  }
}
```

### 8.2 Fabric 对象 → 业务层（回写方向）

用户拖拽 / 缩放 / 旋转后，通过 `FabricEvent` 监听对象变化，回写业务图层：

```typescript
export function fromFabricObject(obj: FabricObject): Partial<BaseLayer> {
  return {
    x: obj.left,
    y: obj.top,
    width: obj.width,
    height: obj.height,
    rotate: obj.angle,
    scaleX: obj.scaleX,
    scaleY: obj.scaleY,
  }
}
```

### 8.3 事件封装（FabricEvent）

统一封装 Fabric 画布事件，只向业务层暴露 `onLayerChange` 语义化回调：

```typescript
export function bindFabricEvents(
  canvas: Canvas,
  onChange: (id: string, patch: Partial<BaseLayer>) => void,
) {
  canvas.on('object:modified', (e) => {
    const obj = e.target
    onChange(obj.get('id'), fromFabricObject(obj))
  })
  canvas.on('object:moving', (e) => {
    /* 节流回写 */
  })
  canvas.on('object:scaling', (e) => {
    /* 节流回写 */
  })
  canvas.on('object:rotating', (e) => {
    /* 节流回写 */
  })
  canvas.on('selection:created', (e) => {
    /* 更新 activeLayerId */
  })
}
```

---

## 9. 历史记录与状态设计

### 9.1 历史基于业务图层（与引擎无关）

历史快照存储的是**业务 `BaseLayer[]` 数组**，而非引擎实例 / Fabric 对象，因此：

- 撤销 / 重做逻辑**双引擎完全共用**，零适配成本
- Fabric 画布变更后通过适配器回写业务状态，再生成快照，**撤销重做与引擎完全解耦**
- 深浅拷贝用 `structuredClone` 快照，性能可控

```typescript
export function useHistory(max = 50) {
  const undoStack = ref<BaseLayer[][]>([])
  const redoStack = ref<BaseLayer[][]>([])

  function pushSnapshot(layers: BaseLayer[]) {
    const snap = structuredClone(layers)
    undoStack.value.push(snap)
    if (undoStack.value.length > max) undoStack.value.shift()
    redoStack.value = []
  }

  function undo(layers: Ref<BaseLayer[]>) {
    const snap = undoStack.value.pop()
    if (!snap) return
    redoStack.value.push(structuredClone(layers.value))
    layers.value = snap
  }

  function redo(layers: Ref<BaseLayer[]>) {
    const snap = redoStack.value.pop()
    if (!snap) return
    undoStack.value.push(structuredClone(layers.value))
    layers.value = snap
  }

  return { undoStack, redoStack, pushSnapshot, undo, redo }
}
```

### 9.2 撤销重做后的引擎同步

历史回退后，`useEditor` 将恢复的图层数组整体同步给当前引擎重建：

```typescript
// useEditor 内部
function applyLayers(layers: BaseLayer[]) {
  engine.clear()
  layers.forEach((layer) => engine.addLayer(layer))
  engine.render()
}
```

---

## 10. 引擎工厂与按需加载

### 10.1 引擎抽象基类（BaseEngine）

```typescript
// engine/base/BaseEngine.ts
export interface BaseEngine {
  init(): void // 初始化画布
  destroy(): void // 销毁，释放资源
  setSize(width: number, height: number): void // 调整画布尺寸
  addLayer(layer: BaseLayer): void // 新增图层
  updateLayer(id: string, patch: Partial<BaseLayer>): void // 更新图层
  removeLayer(id: string): void // 删除图层
  render(): void // 重绘全部图层
  exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob> // 导出
  clear(): void // 清空画布
  getCapabilities(): EngineCapabilities // 能力探测（差异抹平用）
}
```

`lightweight` 与 `powerful` 两个引擎都必须完整实现上述接口，业务层只依赖 `BaseEngine` 类型。

### 10.2 引擎统一入口与工厂（engine/index.ts）

```typescript
// engine/index.ts
import type { BaseEngine } from './base/BaseEngine'
import { ENGINE_TYPE } from '../config/engine.config'

export function createEngine(canvasEl: HTMLCanvasElement, w: number, h: number): BaseEngine {
  if (ENGINE_TYPE === 'powerful') {
    // 增强引擎：动态懒加载 Fabric，减少首屏包体积
    return import('./powerful/FabricEngine').then((m) => new m.FabricEngine(canvasEl, w, h))
  }
  // 轻量引擎：默认，零额外依赖
  return new (require('./lightweight/CanvasEngine').CanvasEngine)(canvasEl, w, h)
}
```

> 实际落地时使用 Vite 的 `import()` 动态导入 + 顶层 `await`（或异步工厂）实现 Fabric 的按需加载，避免轻量模式打包 Fabric 代码。

### 10.3 编辑器入口自动适配

```vue
<template>
  <ImageEditor :src="imageUrl" @export="handleExport" />
</template>

<script setup lang="ts">
import ImageEditor from '@/components/ImageEditor/ImageEditor.vue'

function handleExport(blob: Blob) {
  // 上传 / 下载 / 回传业务接口
}
</script>
```

`ImageEditor.vue` 内部通过 `createEngine()` 自动创建引擎，并根据 `getCapabilities()` 渲染可用面板 —— **业务方无需关心当前使用的是哪个引擎**。

### 10.4 切换引擎的兼容规则

- **视图层**：所有 UI 组件、工具栏、功能面板完全不用改（高阶能力按能力探测自动显隐）
- **业务逻辑层**：所有 Composables 钩子对外 API 保持一致，内部只调 `BaseEngine` 统一接口
- **核心引擎层**：仅替换 `createEngine()` 的返回实例
- **工具层、类型层**：小幅拓展适配，不改动原有核心定义
- **切换零业务成本**：轻量 ⇄ 增强 随时切换，业务代码与数据模型零改动

---

## 11. 落地实施步骤

1. **安装依赖**：`npm install fabric`（项目已装 `^7.4.0`；Fabric 6+ 自带类型，**无需 `@types/fabric`**）
2. **搭建目录骨架**：按第 4 章创建 `config/`、`engine/base/`、`engine/lightweight/`、`engine/powerful/`、`types/`、`utils/`
3. **定义数据契约**：`types/index.ts` 完成 `EditorMode`、`BaseLayer`（含 `scaleX/scaleY`）、`EditorState`、`EngineCapabilities`
4. **实现抽象基类**：`BaseEngine` 定义统一接口（init / destroy / setSize / addLayer / updateLayer / removeLayer / render / exportToBlob / clear / getCapabilities）
5. **实现轻量引擎**：`CanvasEngine` + `LayerManager` + `render/*`，完成基础裁剪、旋转、缩放、渲染、导出
6. **实现增强引擎**：`FabricEngine` + `FabricAdapter`（双向适配）+ `FabricEvent`（事件封装）+ `render/*`
7. **配置切换开关**：`engine.config.ts` 接入 `VITE_EDITOR_ENGINE` 环境变量，`engine/index.ts` 实现 `createEngine()` 工厂
8. **实现业务钩子**：`useEditor` / `useHistory` / `useCrop` / `useDraw` / `useText` / `useFilter` / `useSticker` / `useFile`，只依赖 `BaseEngine`
9. **搭建 UI 面板**：工具栏与各功能面板，按 `getCapabilities()` 动态渲染可用功能
10. **联调验证**：分别以 `lightweight` 与 `powerful` 两种模式跑通「导入 → 编辑 → 撤销重做 → 导出」全流程，验证差异抹平策略

---

## 12. 方案总结

本方案提供一套**双引擎可选、业务零感知、可按需切换**的 Vue 图片编辑框架：

- **轻量场景**（图片裁剪 / 旋转 / 压缩 / 基础滤镜）：使用原生 Canvas 引擎，零额外依赖、包体积最小
- **复杂场景**（海报设计 / 多元素自由编辑 / SVG / 群组）：一键切换 Fabric 增强引擎，交互能力完整
- **切换零成本**：统一 `BaseLayer` 数据契约 + `BaseEngine` 统一接口 + `createEngine` 工厂 + 能力差异抹平策略，业务代码完全复用

通过四层架构、图层驱动、双引擎可选、插件化功能设计，兼顾**轻量化与高扩展性**，可长期迭代并具备封装独立组件库的条件，满足企业级后台、H5、海报设计、头像编辑、图片处理工具等业务场景。

---

## 13. 附录：Fabric.js 进阶技术细节

> 本附录保留原 v1.x 文档的高阶能力设计，作为 Fabric 增强引擎（`powerful`）落地时的**技术参考**，重点覆盖 WebGL 调色、图层隔离、标注交互与高保真导出。

### 13.1 状态管理与图层隔离

Fabric.js 内部存在复杂的双向指针引用及循环结构。若直接使用 Vue 的 `ref()` 或 `reactive()` 深度代理，会造成 Proxy 拦截所有内部属性，引发显著的渲染卡顿与内存泄漏。因此采用以下隔离策略：

- **Canvas 实例存储**：统一采用 `shallowRef()` 或普通全局模块变量持有
- **状态单向流**：仅将图层元数据（x, y, w, h、旋转角度、滤镜数值、标注图层列表）抽离到 Pinia / Composables 中进行响应式管理

```
[UI Component Event]
        │
        ▼
[Pinia Action / Hook] ──► 更新编辑参数状态（非破坏性配置）
        │
        ▼
[Fabric Adapter API] ──► 批量调用 Fabric 实例原生方法并触发 requestRenderAll()
```

#### 13.1.1 图层分级标准

| 层级序号 | 图层名称             | 承载对象类型                  | 事件响应规则                            | 滤镜继承性                     |
| -------- | -------------------- | ----------------------------- | --------------------------------------- | ------------------------------ |
| Layer 0  | 底图层（Base）       | `FabricImage`                 | 禁止直接拖拽与缩放，仅响应底层 API 控制 | 承载所有 WebGL 调色着色器      |
| Layer 1  | 马赛克层（Mosaic）   | 克隆 `FabricImage` + clipPath | 响应拖拽、缩放、删除                    | 继承底图色彩，叠加像素化滤镜   |
| Layer 2  | 标注层（Annotation） | `Rect`, `Path`                | 响应标准控制点缩放与平移                | 独立矢量渲染，不受底图调色影响 |
| Layer 3  | 水印层（Watermark）  | `IText`, `FabricImage`        | 视模式支持自由拖动或全屏固定            | 独立半透明渲染，位于最顶层     |
| Layer 4  | 交互蒙版（Overlay）  | 裁剪框、九宫格线、悬浮条      | 捕获最高优先级 Pointer 事件             | 不参与最终图像合成导出         |

### 13.2 进阶色彩调节系统（WebGL Pipeline）

系统启用 WebGL 滤镜后端，通过**单阶段自定义片段着色器（Fragment Shader）**与 **1D 纹理查找表（LUT）** 实现多维度调色。

```typescript
import { WebglFilterBackend } from 'fabric'
// Fabric 7.x：filterBackend 设置方式
```

#### 13.2.1 核心着色器设计（色温、色调、色调分离、曲线映射）

将多个调色步骤合并到单个着色器中，避免多次 GPU 读写开销：

```typescript
import { Image, filters } from 'fabric'

export const MasterColorAdjustmentFilter = filters.BaseFilter.extend({
  type: 'MasterColorAdjustmentFilter',

  fragmentSource: `
    precision highp float;
    uniform sampler2D uSampler;
    uniform sampler2D uCurveLut;   // 256x1 曲线映射纹理（包含 RGB 通道计算结果）
    uniform float uTemperature;    // 色温: -1.0 ~ 1.0（蓝-黄偏移）
    uniform float uTint;           // 色调: -1.0 ~ 1.0（绿-洋红偏移）
    uniform vec3 uShadowTint;      // 阴影着色色相与强度
    uniform vec3 uHighlightTint;   // 高光着色色相与强度
    uniform float uBalance;        // 色调分离分界平衡: -1.0 ~ 1.0
    varying vec2 vTexCoord;

    void main() {
      vec4 color = texture2D(uSampler, vTexCoord);
      if (color.a == 0.0) {
        gl_FragColor = color;
        return;
      }

      // 1. 色温与色调（White Balance）
      color.r += uTemperature * 0.12 + uTint * 0.04;
      color.g -= uTint * 0.12;
      color.b -= uTemperature * 0.12 - uTint * 0.04;

      // 2. 亮度加权（Rec.709 Luminance）
      float luma = dot(color.rgb, vec3(0.2126, 0.7152, 0.0722));

      // 3. 高光 / 阴影色调分离（Split Toning）
      float shadowFactor = clamp((1.0 - (luma + uBalance * 0.5)) * 2.0, 0.0, 1.0);
      float highlightFactor = clamp(((luma + uBalance * 0.5) - 0.5) * 2.0, 0.0, 1.0);

      color.rgb = mix(color.rgb, color.rgb * uShadowTint, shadowFactor * 0.4);
      color.rgb = mix(color.rgb, color.rgb * uHighlightTint, highlightFactor * 0.4);

      // 4. 256 阶 RGB 曲线映射（LUT Lookup）
      color.r = texture2D(uCurveLut, vec2(clamp(color.r, 0.0, 1.0), 0.5)).r;
      color.g = texture2D(uCurveLut, vec2(clamp(color.g, 0.0, 1.0), 0.5)).g;
      color.b = texture2D(uCurveLut, vec2(clamp(color.b, 0.0, 1.0), 0.5)).b;

      gl_FragColor = vec4(clamp(color.rgb, 0.0, 1.0), color.a);
    }
  `,

  temperature: 0,
  tint: 0,
  shadowTint: [1, 1, 1],
  highlightTint: [1, 1, 1],
  balance: 0,
  curveLutTexture: null,

  applyToWebGL(options) {
    const gl = options.context
    const program = options.program
    this.bindUniforms(gl, program)
    gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4)
  },
})
```

#### 13.2.2 RGB 曲线单调三次样条算法（Monotone Cubic Spline）

为避免普通贝塞尔样条在极值点处产生的插值震荡（Runge's Phenomenon），采用**单调三次样条**计算 `0 ~ 255` 查找表：

```typescript
export function buildCurveLUT(controlPoints) {
  // controlPoints 格式: [{x: 0, y: 0}, {x: 128, y: 140}, {x: 255, y: 255}]
  const n = controlPoints.length
  const lut = new Uint8Array(256)
  if (n < 2) return lut

  const x = controlPoints.map((p) => p.x)
  const y = controlPoints.map((p) => p.y)

  // 1. 计算割线斜率与切线
  const dx = [],
    dy = [],
    m = []
  for (let i = 0; i < n - 1; i++) {
    dx.push(x[i + 1] - x[i])
    dy.push(y[i + 1] - y[i])
    m.push(dy[i] / dx[i])
  }

  const tangents = [m[0]]
  for (let i = 1; i < n - 1; i++) {
    if (m[i - 1] * m[i] <= 0) {
      tangents.push(0)
    } else {
      tangents.push((m[i - 1] + m[i]) / 2)
    }
  }
  tangents.push(m[m.length - 1])

  // 2. 生成 256 点插值映射
  let pIdx = 0
  for (let i = 0; i < 256; i++) {
    while (pIdx < n - 2 && i > x[pIdx + 1]) {
      pIdx++
    }
    const h = dx[pIdx]
    const t = (i - x[pIdx]) / h
    const t2 = t * t
    const t3 = t2 * t

    // Hermite 基函数插值
    const h00 = 2 * t3 - 3 * t2 + 1
    const h10 = t3 - 2 * t2 + t
    const h01 = -2 * t3 + 3 * t2
    const h11 = t3 - t2

    const val =
      h00 * y[pIdx] + h10 * h * tangents[pIdx] + h01 * y[pIdx + 1] + h11 * h * tangents[pIdx + 1]
    lut[i] = Math.max(0, Math.min(255, Math.round(val)))
  }
  return lut
}
```

### 13.3 关键交互功能技术方案

#### 13.3.1 重点画框（矩形 / 标注框）

在标注模式下捕获鼠标拖拽轨迹，绘制具有统一控制手柄风格的矢量矩形：

```typescript
import { Canvas, Rect } from 'fabric'

export function registerBoxAnnotation(canvas: Canvas) {
  let isDrawing = false
  let startX = 0,
    startY = 0
  let activeRect: Rect | null = null

  canvas.on('mouse:down', (e) => {
    if (!canvas.isAnnotatingBox) return
    isDrawing = true
    const pointer = canvas.getPointer(e.e)
    startX = pointer.x
    startY = pointer.y

    activeRect = new Rect({
      left: startX,
      top: startY,
      width: 0,
      height: 0,
      fill: 'transparent',
      stroke: '#FF3B30',
      strokeWidth: 3,
      strokeUniform: true, // 保持缩放时边框粗细一致
      cornerColor: '#FFFFFF',
      cornerStrokeColor: '#FF3B30',
      cornerSize: 8,
      transparentCorners: false,
    })
    canvas.add(activeRect)
  })

  canvas.on('mouse:move', (e) => {
    if (!isDrawing) return
    const pointer = canvas.getPointer(e.e)
    if (startX > pointer.x) activeRect!.set({ left: pointer.x })
    if (startY > pointer.y) activeRect!.set({ top: pointer.y })
    activeRect!.set({
      width: Math.abs(startX - pointer.x),
      height: Math.abs(startY - pointer.y),
    })
    canvas.requestRenderAll()
  })

  canvas.on('mouse:up', () => {
    if (!isDrawing) return
    isDrawing = false
    canvas.isAnnotatingBox = false
    activeRect!.setCoords()
    canvas.setActiveObject(activeRect!)
    canvas.requestRenderAll()
  })
}
```

#### 13.3.2 常用水印体系（文本 / Logo / 全屏防伪）

- **固定文本水印**：使用 `IText`，设置 `editable: true`、`fill: '#FFFFFF'`、`shadow`（增强对比防遮挡）
- **全屏倾斜防伪水印**：通过动态生成 Canvas 生成 Pattern 填充：

```typescript
import { Pattern } from 'fabric'

export function createTiledWatermarkPattern(text) {
  const patternCanvas = document.createElement('canvas')
  patternCanvas.width = 300
  patternCanvas.height = 200
  const ctx = patternCanvas.getContext('2d')

  ctx.rotate((-25 * Math.PI) / 180)
  ctx.font = '16px sans-serif'
  ctx.fillStyle = 'rgba(180, 180, 180, 0.25)'
  ctx.fillText(text, -50, 150)

  return new Pattern({
    source: patternCanvas,
    repeat: 'repeat',
  })
}
```

#### 13.3.3 几何旋转与视图重算（90° 步进与自由微调）

定角旋转涉及画布容器宽高对调及底图物理坐标重定位：

```typescript
import { Canvas } from 'fabric'

export function rotateCanvasClockwise(canvas: Canvas, baseImage) {
  const nextAngle = (baseImage.angle + 90) % 360
  baseImage.set('angle', nextAngle)

  // 90 度与 270 度时宽高维度调换
  const isPerpendicular = nextAngle === 90 || nextAngle === 270
  const targetWidth = isPerpendicular ? baseImage.getScaledHeight() : baseImage.getScaledWidth()
  const targetHeight = isPerpendicular ? baseImage.getScaledWidth() : baseImage.getScaledHeight()

  canvas.setDimensions({ width: targetWidth, height: targetHeight })
  baseImage.center()
  baseImage.setCoords()
  canvas.requestRenderAll()
}
```

#### 13.3.4 矢量化可重编辑局部马赛克

将马赛克实现为**克隆底图 + 像素化滤镜 + 裁剪路径蒙版**的复合体，支持用户随时点击移动、拉伸与删除：

```typescript
import { Image as FabricImage, Rect, filters } from 'fabric'

export async function addMosaicRegion(
  canvas: Canvas,
  baseImage: FabricImage,
  { left, top, width, height, blockSize = 8 },
) {
  const clonedImg = await baseImage.clone()
  // 1. 为克隆图层注入 Pixelate 滤镜
  clonedImg.filters = [new filters.Pixelate({ blocksize: blockSize })]
  clonedImg.applyFilters()

  // 2. 建立独立裁剪路径
  const clipRect = new Rect({
    left,
    top,
    width,
    height,
    absolutePositioned: true,
  })

  clonedImg.set({
    clipPath: clipRect,
    selectable: true,
    hasControls: true,
  })

  // 3. 关联拖拽：移动马赛克时同步更新 clipPath
  clonedImg.on('moving', () => {
    clipRect.set({ left: clonedImg.left, top: clonedImg.top })
  })

  canvas.add(clonedImg)
  canvas.setActiveObject(clonedImg)
  canvas.requestRenderAll()
}
```

### 13.4 高保真合成导出与压缩管线

#### 13.4.1 全分辨率还原导出机制

为保障大图（如 `4000 × 3000`）在视口缩放编辑后不失真，导出时依据原始物理像素动态计算放大因子 M：

```
M = Image.originWidth / Image.currentRenderWidth
```

```typescript
export function exportHighResolutionDataURL(canvas, baseImage, format = 'jpeg', quality = 0.92) {
  const originWidth = baseImage.getOriginalSize().width
  const currentDisplayWidth = baseImage.getScaledWidth()
  const multiplier = originWidth / currentDisplayWidth

  return canvas.toDataURL({
    format,
    quality,
    multiplier,
    enableRetinaScaling: false,
  })
}
```

#### 13.4.2 客户端二次压缩策略

采用 **Web Worker + 现代图像编码接口**进行多线程异步压缩，避免占用主线程渲染帧：

```
[Canvas 导出 DataURL / Blob]
              │
              ▼
[Web Worker 线程池]
   ├── 优先调用 OffscreenCanvas 转换为 image/webp
   └── 判定文件大小是否超标（如目标 < 500KB）
              │
              ├─► 超标：执行二分搜索质量参数（Quality Bisection: [0.5 ~ 0.95]）
              └─► 达标：输出最终 ArrayBuffer / File 对象
```

### 13.5 性能优化与边界防御清单

1. **纹理尺寸限制保护**：在载入超大图片（如超过 GPU `MAX_TEXTURE_SIZE`，通常为 4096px 或 8192px）时，前置做离屏等比降采样，防止 WebGL 纹理绑定失败崩溃。
2. **绘制节流与 RAF 调度**：调色滑块的 `@input` 事件严禁用全量 `renderAll()` 阻塞，统一使用 Fabric 内置的 `requestRenderAll()`，让更新对齐浏览器显示刷新率（60/120Hz）。
3. **内存即时释放**：在组件注销（`onBeforeUnmount`）或图片切换时，显式调用 `fabricCanvas.dispose()`，并遍历调用滤镜和 Pattern 的销毁逻辑，释放绑定的 WebGL 纹理单元（`gl.deleteTexture`）。

---

## 14. 项目落地现状与相关文件

> **落地现状**：本项目当前已落地一个基于 Fabric.js 的**轻量图片编辑器** `ImageEditorModal.vue`（裁剪 / 旋转 / 压缩 / 重置），以及图片查看器 `ImageViewerModal.vue`。本方案（双引擎架构 + 差异抹平）可作为**下一步框架化演进**的正式落地蓝图；`ImageEditorModal` 对应本方案 `powerful` 引擎（Fabric）能力子集，且未接入双引擎切换体系。

### 14.1 已落地组件

| 文件                                              | 说明                                                                          |
| ------------------------------------------------- | ----------------------------------------------------------------------------- |
| `frontend/src/components/ImageEditorModal.vue`    | **已落地**：Fabric.js 图片编辑器（裁剪 / 旋转 / 压缩 / 重置，`fabric@7.4.0`） |
| `frontend/src/components/ImageViewerModal.vue`    | **已落地**：图片查看器（缩放 / 拖拽 / 翻页）                                  |
| `frontend/src/components/AttachmentInputArea.vue` | **已落地**：附件输入区，查看 / 编辑入口及编辑结果替换                         |
| `frontend/src/pages/ContentCreate.vue`            | **已落地**：创作内容页，附件预览区接入查看 / 编辑入口                         |

### 14.2 本方案规划（框架化演进）

| 目录 / 文件                                                     | 说明                                                                         |
| --------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `src/components/ImageEditor/ImageEditor.vue`                    | **规划**：编辑器入口主组件（双引擎自动适配，第 4 章目录规范）                |
| `src/components/ImageEditor/config/engine.config.ts`            | **规划**：引擎全局配置（`VITE_EDITOR_ENGINE` 切换开关）                      |
| `src/components/ImageEditor/engine/base/BaseEngine.ts`          | **规划**：引擎抽象基类（统一接口规范）                                       |
| `src/components/ImageEditor/engine/lightweight/CanvasEngine.ts` | **规划**：原生 Canvas 轻量引擎（默认）                                       |
| `src/components/ImageEditor/engine/powerful/FabricEngine.ts`    | **规划**：Fabric 增强引擎（`FabricAdapter` 双向适配）                        |
| `src/components/ImageEditor/engine/index.ts`                    | **规划**：`createEngine()` 统一入口、动态切换工厂                            |
| `src/components/ImageEditor/composables/*.ts`                   | **规划**：useEditor / useHistory / useCrop 等业务钩子                        |
| `src/components/ImageEditor/types/index.ts`                     | **规划**：EditorMode / BaseLayer / EditorState / EngineCapabilities 全局类型 |
