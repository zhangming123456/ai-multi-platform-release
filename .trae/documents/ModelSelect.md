# 模型选择组件（ModelSelect）技术文档

> **版本**：v1.1 · **更新**：2026-08-18 · **核心文件**：`frontend/src/components/shared/ModelSelect.vue` / `frontend/src/pages/InspectionTaskDetail.vue` / `frontend/src/pages/InspectionEdit.vue` / `frontend/src/stores/tokenPlan.ts`
>
> **v1.0 变更**：将「整改任务详情」页的 AI 复核模型下拉选择抽取为公共组件 `ModelSelect`。组件内聚了模型选项构建（`enabledPlans` + `parseModelField`）、支持视觉能力过滤、模型配置自动加载、首个视觉模型自动选中与 `planId:modelId` 键值解析，页面仅保留卡片外壳与提交字段状态。
>
> **v1.1 变更**：组件增加 `require-vision` 与 `size` 两个配置项，并接入「发起巡店」（`InspectionEdit.vue`）的 AI 巡店表单（页面主体与抽屉两处复用）。发起巡店通过 `require-vision=false` 允许选择非视觉模型（AI 分析可仅依赖文字描述），通过 `model-value` 单向受控 + `auto-select=false` 保留其「`activePlanId` 方案首个模型」的自动选择策略；选项展示格式统一为「服务商 · 模型ID」。

## 1. 概述

`ModelSelect` 是 AI 分析场景的**模型下拉选择器**（不含卡片外壳），支持在模型列表中选择单个模型。组件自身完成以下能力，业务页面无需再关心模型数据来源与解析：

- **模型配置加载**：挂载时自动调用 `tokenPlanStore.loadPlans()`，从「模型服务商管理」读取启用中的模型方案
- **选项构建**：遍历 `enabledPlans`，对每个方案的 `model` 字段执行 `parseModelField` 展开为模型条目（过滤空 ID）
- **视觉能力过滤（可选）**：`require-vision` 开启时仅 `types` 包含 `vision` 的模型可选择，其余选项禁用并标注「（不支持视觉）」；关闭时全部模型可选，由业务页在提交时自行校验
- **自动选择（可选）**：`auto-select` 开启时，`require-vision=true` 自动选中第一个支持视觉的模型，`false` 则选中第一个模型
- **键值解析**：下拉值与提交值解耦 —— 内部使用 `planId:modelId` 作为键，通过 `change` 事件向父组件输出解析后的 `{ planId, modelId, hasVision }`

### 1.1 应用场景

| 场景                    | 页面                                                  | 配置                                                             |
| ----------------------- | ----------------------------------------------------- | ---------------------------------------------------------------- |
| AI 复核（必须图片分析） | 整改任务详情 `InspectionTaskDetail.vue`               | 默认（`require-vision=true`、`auto-select=true`）                |
| AI 巡店（可纯文字分析） | 发起/编辑巡店 `InspectionEdit.vue`（主体 + 抽屉复用） | `require-vision=false`、`auto-select=false` + `model-value` 受控 |

### 1.2 设计原则

- **纯选择器组件**：只负责「选模型」，不包含卡片容器、标题、提示文案等布局，由业务页面按自身排版自由组织
- **双向数据流**：`v-model` 绑定选中键（可选受控），`change` 事件输出结构化结果，父组件无需重复解析字符串
- **内部自动加载**：模型配置的加载由组件完成，父组件无需感知 `tokenPlan` store

---

## 2. 组件接口

### 2.1 Props

| Prop            | 类型                                       | 默认值     | 说明                                                                    |
| --------------- | ------------------------------------------ | ---------- | ----------------------------------------------------------------------- |
| `modelValue`    | `string`                                   | `''`       | 当前选中模型的键（`planId:modelId`），可选；受控模式下用 `v-model` 绑定 |
| `autoSelect`    | `boolean`                                  | `true`     | 模型列表就绪后是否自动选择（视觉模式下选首个视觉模型，否则选首个模型）  |
| `requireVision` | `boolean`                                  | `true`     | 是否仅允许选择支持视觉的模型（禁用非视觉选项并标注「（不支持视觉）」）  |
| `size`          | `'mini' \| 'small' \| 'medium' \| 'large'` | `'medium'` | 下拉选择器尺寸，透传 Arco `a-select`                                    |

> 不传 `modelValue` 时组件内部自持选中状态，父组件仅通过 `change` 事件获取结果即可。

### 2.2 Emits

| Event               | 参数                                                      | 说明                                     |
| ------------------- | --------------------------------------------------------- | ---------------------------------------- |
| `update:modelValue` | `string`（选中键 `planId:modelId`）                       | 选中键变化（配合 `v-model` 受控使用）    |
| `change`            | `{ planId: string, modelId: string, hasVision: boolean }` | 选择变化时的结构化结果（含视觉能力标记） |

`change` 载荷类型导出为 `ModelSelectValue`：

```typescript
export interface ModelSelectValue {
  planId: string // 模型方案 ID（提交接口的 plan_id）
  modelId: string // 模型 ID（提交接口的 model_id）
  hasVision: boolean // 是否支持视觉理解（提交前校验用）
}
```

### 2.3 内部实现要点

组件数据流：

```
tokenPlanStore.loadPlans() ──► enabledPlans ──► options(computed)
                                                     │  key = `${plan.id}:${model.id}`
                                                     │  hasVision = model.types.includes('vision')
                                                     ▼
              autoSelect watch ──► requireVision ? 首个视觉模型
                                                    : 首个模型 ──► emit change
                                                     ▲
                             用户选择 a-select ──► handleChange ──► emit update:modelValue + change
```

关键逻辑：

```typescript
// 选项构建（与页面原有实现一致）
const options = computed<ModelOption[]>(() => {
  const list: ModelOption[] = []
  for (const p of tokenPlanStore.enabledPlans) {
    const planName = p.displayName || p.name
    for (const m of parseModelField(p.model)) {
      list.push({
        key: `${p.id}:${m.id}`,
        planName,
        modelId: m.id,
        hasVision: m.types.includes('vision'),
      })
    }
  }
  return list
})

// 自动选择：列表就绪后按 requireVision 策略选中（可关闭）
watch(
  options,
  (list) => {
    if (!props.autoSelect) return
    if (list.some((o) => o.key === selectedKey.value)) return
    if (props.requireVision) {
      const first = list.find((o) => o.hasVision)
      if (first) select(first)
    } else if (list.length > 0) {
      select(list[0])
    }
  },
  { immediate: true },
)

// 键值解析：planId:modelId → 提交字段
function parseKey(key: string): { planId: string; modelId: string } {
  const idx = key.indexOf(':')
  if (idx <= 0) return { planId: '', modelId: '' }
  return { planId: key.slice(0, idx), modelId: key.slice(idx + 1) }
}

// 选择变化：统一出口，携带结构化结果
function select(opt: ModelOption) {
  selectedKey.value = opt.key
  emit('update:modelValue', opt.key)
  const { planId, modelId } = parseKey(opt.key)
  emit('change', { planId, modelId, hasVision: opt.hasVision })
}
```

> **行为说明**：
>
> - `options` 为空（加载失败或无启用方案）时不产生选择，`change` 不触发
> - 点击清空（`allow-clear`）时 Arco 传值为 `undefined`，组件保持内部键清空、不触发 `change`，与抽取前页面行为一致
> - `modelValue` 作为受控输入时，父组件值变化会同步覆盖内部选中键

---

## 3. 页面接入

### 3.1 整改任务详情（InspectionTaskDetail.vue）

页面保留卡片外壳（圆角容器、标题、提示文案），内部改用组件：

```vue
<div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
  <div class="text-[15px] font-semibold text-[#1D1D1F] mb-3">AI 复核模型</div>
  <ModelSelect @change="onModelChange" />
  <div class="text-[12px] text-[#86868b] mt-2">AI 复核需选择支持视觉理解的模型。</div>
</div>
```

脚本侧：删除原 `modelOptions` computed、`selectedModelKey` 与键值解析逻辑，保留提交字段并新增视觉标记：

```typescript
import ModelSelect, { type ModelSelectValue } from '@/components/shared/ModelSelect.vue'

const selectedPlanId = ref('')
const selectedModelId = ref('')
const selectedHasVision = ref(false)

function onModelChange(val: ModelSelectValue) {
  selectedPlanId.value = val.planId
  selectedModelId.value = val.modelId
  selectedHasVision.value = val.hasVision
}

async function handleRecheck() {
  if (!selectedPlanId.value || !selectedModelId.value) {
    Message.warning('请选择 AI 复核模型')
    return
  }
  if (!selectedHasVision.value) {
    Message.warning('请选择支持视觉理解的模型')
    return
  }
  await api.post(`/inspection-tasks/${task.value?.id}/recheck`, {
    items: submittedItems.value.map((it) => ({ item_id: it.id })),
    plan_id: selectedPlanId.value,
    model_id: selectedModelId.value,
  })
}
```

### 3.2 发起/编辑巡店（InspectionEdit.vue）

发起巡店与整改任务详情的约束不同：AI 巡店分析允许纯文字描述，因此需要放开视觉限制；同时页面原有「`activePlanId` 方案首个模型」的自动选择策略需要保留。接入方式为**单向受控 + 关闭组件内部自动选择**：

```vue
<ModelSelect
  :model-value="activeModelKey"
  :auto-select="false"
  :require-vision="false"
  :size="(selectSize as any)"
  @change="onModelChange"
/>
```

脚本侧：`activeModelKey` computed（`${selectedPlanId.value}:${selectedModelId.value}`）负责向组件下发选中键；`onMounted` 中基于 `activePlanId` 的自动选择逻辑保留在页面内；选择变化仅需写入提交字段：

```typescript
import ModelSelect, { type ModelSelectValue } from '@/components/shared/ModelSelect.vue'

const activeModelKey = computed(() => {
  if (!selectedPlanId.value || !selectedModelId.value) return ''
  return `${selectedPlanId.value}:${selectedModelId.value}`
})

function onModelChange(val: ModelSelectValue) {
  selectedPlanId.value = val.planId
  selectedModelId.value = val.modelId
}
```

> 页面主体与抽屉两处表单通过 `DefineAIInspectForm` / `ReuseAIInspectForm` 复用，两处实例的模型选择行为完全一致。

### 3.3 接入前后对比

以下对比以整改任务详情为例，发起巡店的接入方式见 3.2：

| 关注点   | 抽取前（页面内实现）                              | 抽取后（组件）                                  |
| -------- | ------------------------------------------------- | ----------------------------------------------- |
| 模型加载 | `onMounted` 中 `await tokenPlanStore.loadPlans()` | 组件挂载自动加载                                |
| 选项构建 | 页面内 `modelOptions` computed                    | 组件内 `options` computed                       |
| 自动选择 | `onMounted` 中手动查找首个视觉模型                | 组件内 `watch(options, { immediate: true })`    |
| 键值解析 | 页面内 `onModelChange(val: unknown)` 字符串切分   | 组件内 `parseKey` + `change` 事件输出结构化对象 |
| 提交校验 | 通过 `selectedModelKey` 反查 `hasVision`          | 直接使用 `selectedHasVision`                    |

---

## 4. 使用指南

1. **引入组件**：`import ModelSelect from '@/components/shared/ModelSelect.vue'`
2. **放置位置**：放在需要模型选择的位置（组件不含外壳，按页面需要包裹容器）
3. **接收结果**：监听 `@change` 保存 `planId` / `modelId` / `hasVision`
4. **受控模式（可选）**：`v-model="key"` 绑定选中键；`auto-select=false` 时需外部提供初始值
5. **提交前校验**：`planId`、`modelId` 非空且 `hasVision` 为 `true`

---

## 5. 相关文件

| 文件                                             | 说明                                                                               |
| ------------------------------------------------ | ---------------------------------------------------------------------------------- |
| `frontend/src/components/shared/ModelSelect.vue` | **公共模型选择组件**（选项构建 / 视觉过滤 / 自动选择 / 键值解析）                  |
| `frontend/src/pages/InspectionTaskDetail.vue`    | 整改任务详情页（卡片外壳 + 提交字段，消费 `ModelSelect`）                          |
| `frontend/src/pages/InspectionEdit.vue`          | 发起/编辑巡店页（AI 巡店表单主体 + 抽屉两处复用，`require-vision=false` 受控接入） |
| `frontend/src/stores/tokenPlan.ts`               | 模型方案 store（`enabledPlans` / `loadPlans` / `parseModelField`）                 |
