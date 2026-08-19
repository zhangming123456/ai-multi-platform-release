# 基于 Vue 3 + Fabric.js 的高保真图片编辑系统技术设计文档

> **版本**：v1.1 · **技术栈**：Vue 3（Composition API）+ Fabric.js（WebGL Backend）
>
> **能力范围**：图片裁剪、进阶调色、结构化标注（画框 / 水印 / 马赛克）、客户端高性能压缩

---

## 目录

1. [架构总览](#1-架构总览)
2. [状态管理与图层隔离](#2-状态管理与图层隔离)
3. [进阶色彩调节系统（WebGL Pipeline）](#3-进阶色彩调节系统webgl-pipeline)
4. [关键交互功能技术方案](#4-关键交互功能技术方案)
5. [高保真合成导出与压缩管线](#5-高保真合成导出与压缩管线)
6. [Vue 3 核心单文件组件集成示例](#6-vue-3-核心单文件组件集成示例)
7. [性能优化与边界防御清单](#7-性能优化与边界防御清单)
8. [相关文件](#8-相关文件)

---

## 1. 架构总览

本方案基于 Vue 3（Composition API）+ Fabric.js（WebGL Backend）构建，专为**纯图片裁剪、进阶调色、结构化标注（画框/水印/马赛克）及客户端高性能压缩**而设计。

系统核心原则：

- **数据与视图解耦**：编辑参数状态与 Fabric 实例分离管理
- **非破坏性编辑流程（Non-destructive Pipeline）**：所有编辑操作通过参数状态驱动，可撤销/重做
- **多图层合成机制**：底图、马赛克、标注、水印、交互蒙版分层管理

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Vue 3 UI Layer                            │
│  [顶部控制栏]（撤销/重做/比例/对比/导出）  [右侧/底部面板]（调色/裁剪/标注/水印） │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                    （响应式同步：Pinia / Store）
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Fabric.js Engine Core                        │
│                                                                     │
│  [Annotation Layer]    重点画框（Rect）/ 水印（IText, Image）          │
│  [Pixelate Layer]      动态克隆蒙版 / 局部像素化块                     │
│  [Interaction Overlay] 九宫格裁剪框 / 吸附参考线 / 旋转手柄            │
│  [Base Image Layer]    原图适配 / 几何变换（Angle/Flip）              │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Rendering & Export Pipeline                    │
│                                                                     │
│  [WebGL Shader Backend] → [Offscreen Multiplier] → [Wasm/Worker]    │
│  （色温/色调分离/RGB曲线LUT）  （原图分辨率高保真渲染）   （WebP/MozJPEG压缩）│
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. 状态管理与图层隔离

### 2.1 响应式隔离方案

Fabric.js 内部存在复杂的双向指针引用及循环结构。若直接使用 Vue 的 `ref()` 或 `reactive()` 深度代理，会造成 Proxy 拦截所有内部属性，引发显著的渲染卡顿与内存泄漏。因此采用以下隔离策略：

- **Canvas 实例存储**：统一采用 `shallowRef()` 或普通全局模块变量持有
- **状态单向流**：仅将图层元数据（x, y, w, h、旋转角度、滤镜数值、标注图层列表）抽离到 Pinia 中进行响应式管理

```
[UI Component Event]
        │
        ▼
[Pinia Action] ──► 更新编辑参数状态（非破坏性配置）
        │
        ▼
[Fabric Adapter API] ──► 批量调用 Fabric 实例原生方法并触发 requestRenderAll()
```

### 2.2 图层分级标准

| 层级序号 | 图层名称             | 承载对象类型                   | 事件响应规则                            | 滤镜继承性                     |
| -------- | -------------------- | ------------------------------ | --------------------------------------- | ------------------------------ |
| Layer 0  | 底图层（Base）       | `fabric.Image`                 | 禁止直接拖拽与缩放，仅响应底层 API 控制 | 承载所有 WebGL 调色着色器      |
| Layer 1  | 马赛克层（Mosaic）   | 克隆 `fabric.Image` + clipPath | 响应拖拽、缩放、删除                    | 继承底图色彩，叠加像素化滤镜   |
| Layer 2  | 标注层（Annotation） | `fabric.Rect`, `fabric.Path`   | 响应标准控制点缩放与平移                | 独立矢量渲染，不受底图调色影响 |
| Layer 3  | 水印层（Watermark）  | `fabric.IText`, `fabric.Image` | 视模式支持自由拖动或全屏固定            | 独立半透明渲染，位于最顶层     |
| Layer 4  | 交互蒙版（Overlay）  | 裁剪框、九宫格线、悬浮条       | 捕获最高优先级 Pointer 事件             | 不参与最终图像合成导出         |

---

## 3. 进阶色彩调节系统（WebGL Pipeline）

系统启用 WebGL 滤镜后端，通过**单阶段自定义片段着色器（Fragment Shader）**与 **1D 纹理查找表（LUT）** 实现多维度调色。

```typescript
fabric.filterBackend = new fabric.WebglFilterBackend()
```

### 3.1 核心着色器设计（色温、色调、色调分离、曲线映射）

将多个调色步骤合并到单个着色器中，避免多次 GPU 读写开销：

```typescript
import { fabric } from 'fabric'

export const MasterColorAdjustmentFilter = fabric.util.createClass(
  fabric.Image.filters.BaseFilter,
  {
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
  },
)
```

### 3.2 RGB 曲线单调三次样条算法（Monotone Cubic Spline）

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

---

## 4. 关键交互功能技术方案

### 4.1 重点画框（矩形/标注框）

在标注模式下捕获鼠标拖拽轨迹，绘制具有统一控制手柄风格的矢量矩形：

```typescript
export function registerBoxAnnotation(canvas) {
  let isDrawing = false
  let startX = 0,
    startY = 0
  let activeRect = null

  canvas.on('mouse:down', (e) => {
    if (!canvas.isAnnotatingBox) return
    isDrawing = true
    const pointer = canvas.getPointer(e.e)
    startX = pointer.x
    startY = pointer.y

    activeRect = new fabric.Rect({
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
    if (startX > pointer.x) activeRect.set({ left: pointer.x })
    if (startY > pointer.y) activeRect.set({ top: pointer.y })
    activeRect.set({
      width: Math.abs(startX - pointer.x),
      height: Math.abs(startY - pointer.y),
    })
    canvas.requestRenderAll()
  })

  canvas.on('mouse:up', () => {
    if (!isDrawing) return
    isDrawing = false
    canvas.isAnnotatingBox = false
    activeRect.setCoords()
    canvas.setActiveObject(activeRect)
    canvas.requestRenderAll()
  })
}
```

### 4.2 常用水印体系（文本 / Logo / 全屏防伪）

- **固定文本水印**：使用 `fabric.IText`，设置 `editable: true`、`fill: '#FFFFFF'`、`shadow`（增强对比防遮挡）
- **全屏倾斜防伪水印**：通过动态生成 Canvas 生成 Pattern 填充：

```typescript
export function createTiledWatermarkPattern(text) {
  const patternCanvas = document.createElement('canvas')
  patternCanvas.width = 300
  patternCanvas.height = 200
  const ctx = patternCanvas.getContext('2d')

  ctx.rotate((-25 * Math.PI) / 180)
  ctx.font = '16px sans-serif'
  ctx.fillStyle = 'rgba(180, 180, 180, 0.25)'
  ctx.fillText(text, -50, 150)

  return new fabric.Pattern({
    source: patternCanvas,
    repeat: 'repeat',
  })
}
```

### 4.3 几何旋转与视图重算（90° 步进与自由微调）

定角旋转涉及画布容器宽高对调及底图物理坐标重定位：

```typescript
export function rotateCanvasClockwise(canvas, baseImage) {
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

### 4.4 矢量化可重编辑局部马赛克

将马赛克实现为**克隆底图 + 像素化滤镜 + 裁剪路径蒙版**的复合体，支持用户随时点击移动、拉伸与删除：

```typescript
export async function addMosaicRegion(
  canvas,
  baseImage,
  { left, top, width, height, blockSize = 8 },
) {
  baseImage.clone((clonedImg) => {
    // 1. 为克隆图层注入 Pixelate 滤镜
    clonedImg.filters = [new fabric.Image.filters.Pixelate({ blocksize: blockSize })]
    clonedImg.applyFilters()

    // 2. 建立独立裁剪路径
    const clipRect = new fabric.Rect({
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
  })
}
```

---

## 5. 高保真合成导出与压缩管线

### 5.1 全分辨率还原导出机制

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

### 5.2 客户端二次压缩策略

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

---

## 6. Vue 3 核心单文件组件集成示例

```vue
<script setup>
import { onMounted, onBeforeUnmount, shallowRef, reactive } from 'vue'
import { fabric } from 'fabric'
import { MasterColorAdjustmentFilter, buildCurveLUT } from './filters/MasterFilter'
import { rotateCanvasClockwise } from './utils/transform'

const canvasEl = shallowRef(null)
let fabricCanvas = null
let baseImage = null

const editState = reactive({
  temperature: 0,
  tint: 0,
  brightness: 0,
  contrast: 0,
  balance: 0,
})

onMounted(() => {
  // 1. 初始化 WebGL 滤镜后端
  fabric.filterBackend = new fabric.WebglFilterBackend()

  // 2. 创建 Canvas
  fabricCanvas = new fabric.Canvas(canvasEl.value, {
    preserveObjectStacking: true,
    selectionColor: 'rgba(64, 158, 255, 0.15)',
    selectionBorderColor: '#409EFF',
  })

  // 3. 载入并适配底图
  fabric.Image.fromURL('/example.jpg', (img) => {
    baseImage = img
    baseImage.set({
      selectable: false,
      hoverCursor: 'default',
      crossOrigin: 'anonymous',
    })

    // 初始化画布与底图同尺寸适配
    fabricCanvas.setDimensions({ width: img.width * 0.5, height: img.height * 0.5 })
    baseImage.scale(0.5)
    fabricCanvas.add(baseImage)
    fabricCanvas.renderAll()
  })
})

// 响应式调色管线同步
function updateColorPipeline() {
  if (!baseImage) return

  const defaultCurvePoints = [
    { x: 0, y: 0 },
    { x: 255, y: 255 },
  ]
  const lut = buildCurveLUT(defaultCurvePoints)

  const masterFilter = new MasterColorAdjustmentFilter({
    temperature: editState.temperature / 100,
    tint: editState.tint / 100,
    balance: editState.balance / 100,
  })

  baseImage.filters = [masterFilter]
  baseImage.applyFilters()
  fabricCanvas.requestRenderAll()
}

function handleRotate() {
  if (fabricCanvas && baseImage) {
    rotateCanvasClockwise(fabricCanvas, baseImage)
  }
}

onBeforeUnmount(() => {
  fabricCanvas?.dispose()
})
</script>

<template>
  <div class="editor-container">
    <div class="toolbar">
      <button @click="handleRotate">顺时针旋转 90°</button>
      <div class="slider-group">
        <label>色温: {{ editState.temperature }}</label>
        <input
          type="range"
          min="-100"
          max="100"
          v-model.number="editState.temperature"
          @input="updateColorPipeline"
        />
      </div>
      <div class="slider-group">
        <label>色调: {{ editState.tint }}</label>
        <input
          type="range"
          min="-100"
          max="100"
          v-model.number="editState.tint"
          @input="updateColorPipeline"
        />
      </div>
    </div>
    <div class="canvas-viewport">
      <canvas ref="canvasEl"></canvas>
    </div>
  </div>
</template>

<style scoped>
.editor-container {
  display: flex;
  height: 100vh;
  background-color: #1a1a1a;
  color: #fff;
}
.toolbar {
  width: 300px;
  padding: 16px;
  background-color: #242424;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.canvas-viewport {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
</style>
```

---

## 7. 性能优化与边界防御清单

1. **纹理尺寸限制保护**：在载入超大图片（如超过 GPU `MAX_TEXTURE_SIZE`，通常为 4096px 或 8192px）时，前置做离屏等比降采样，防止 WebGL 纹理绑定失败崩溃。
2. **绘制节流与 RAF 调度**：调色滑块的 `@input` 事件严禁用全量 `renderAll()` 阻塞，统一使用 Fabric 内置的 `requestRenderAll()`，让更新对齐浏览器显示刷新率（60/120Hz）。
3. **内存即时释放**：在组件注销（`onBeforeUnmount`）或图片切换时，显式调用 `fabricCanvas.dispose()`，并遍历调用滤镜和 Pattern 的销毁逻辑，释放绑定的 WebGL 纹理单元（`gl.deleteTexture`）。

---

## 8. 相关文件

| 文件                                           | 说明                                                |
| ---------------------------------------------- | --------------------------------------------------- |
| `frontend/src/components/ImageEditorModal.vue` | 项目实际落地图片编辑器（裁剪/旋转/压缩，Fabric.js） |
| `frontend/src/components/ImageViewerModal.vue` | 项目实际落地图片查看器（缩放/拖拽/翻页）            |
| `frontend/src/pages/ContentCreate.vue`         | 附件预览区接入查看/编辑入口                         |
| `filters/MasterFilter.ts`                      | 主调色滤镜（着色器 + LUT）                          |
| `utils/transform.ts`                           | 几何变换工具（旋转、画布重算）                      |
