# 类型抽取重构规范（临时执行规范）

## 目标

把 `frontend/src/components` 下所有 TypeScript 类型声明抽取到独立类型文件，禁止在 `.vue` / `.ts` 中内联类型声明。

## 硬性规范

1. **命名与放置**：类型文件与模块主文件**同目录**，命名 `<模块主文件名>.types.ts`
   - 组件 `Foo.vue` → 同目录 `Foo.types.ts`
   - 实现文件 `bar.ts` → 同目录 `bar.types.ts`
2. **必须抽取**：
   - 所有 `interface` / `type` / `enum` 声明（含仅导出、未导出的本地类型）
   - 组件 Props / Emits / 插槽类型；`defineProps` / `defineEmits` 禁止内联对象字面量
   - 原本 `export` 的类型继续保持 `export`，并同步更新所有引用方 import 路径
3. **无需抽取**：内置 / 第三方 / 已抽取类型的类型标注与泛型实参（如 `ref<string[]>()`、`Record<string, number>`）
4. 类型文件只含类型与常量声明（允许 `as const`），不含副作用逻辑
5. 纯类型导入统一 `import type { ... } from './Foo.types'`（同目录用相对路径；跨目录按实际相对层级）
6. **禁止添加任何注释**
7. **行为 100% 不变**：不改类名、事件名、Props 名、默认值、运行时逻辑
8. **不要运行** `vue-tsc` / `eslint` / `build`（多个并行任务会互相干扰）
9. 代码风格：2 空格缩进、单引号、**不写分号**
10. 完成后用中文回报：新增文件清单、修改文件清单、以及任何你发现但未处理的跨目录引用

## 组件改造写法

```ts
// Foo.types.ts
export interface FooProps {
  a: string
  b?: number
}

export type FooEmits = {
  (e: 'change', v: string): void
}
```

```vue
<script setup lang="ts">
import type { FooProps, FooEmits } from './Foo.types'

const props = withDefaults(defineProps<FooProps>(), { b: 1 })
const emit = defineEmits<FooEmits>()
</script>
```

## 已完成的既有重命名（不要重复处理）

- `components/DropdownMenu/DropdownMenu.types.ts`（已含 `Option` / `DropdownMenuProps` / `DropdownMenuOptions` / `DropdownMenuEmits` / `MenuItem` / `RequestParams` 等；`DropdownMenu.vue` 已改造完成，任何组都**不要**再动 DropdownMenu）
- `components/shared/PlatformIcon.types.ts`（已含 `PlatformIconType`；B 组需把 `PlatformIcon.vue` 的 `PlatformIconProps` 追加进该文件）
- `components/ImageEditor/ImageEditor.types.ts`（模块级中心类型：`EditorMode`（含 `as const` 常量）/ `LayerType` / `BaseLayer` / `EditorState` / `Viewport` / `HistoryState` / `EngineCapabilities` / `EngineOptions` / `ExportOptions`；E/F 组新类型文件如引用这些类型，从它 `import type`）

## 分组任务（每组只做自己这一组，绝不触碰其他组文件）

### A 组 — 根级组件

| 文件                                     | 要求                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `components/AITemplateItemGenerator.vue` | 抽出 `AIGeneratedItem`、内联 props、内联 emits（其中 `confirm` 事件携带 `{ name; description; items }`，请为其定义具名类型）→ `components/AITemplateItemGenerator.types.ts`；**并同步更新外部引用** `pages/InspectionTemplateEdit.vue`（原 `import AITemplateItemGenerator, { type AIGeneratedItem } from '@/components/AITemplateItemGenerator.vue'` → 组件默认导入保留，类型改为从 `@/components/AITemplateItemGenerator.types` 导入） |
| `components/AttachmentInputArea.vue`     | 抽出 `AttachmentFileType`、`PendingItem`、`DisplayItem`、内联 props（`withDefaults(defineProps<{...}>(), {...})` 改为引用具名 `AttachmentInputAreaProps`）、内联 emits → `components/AttachmentInputArea.types.ts`                                                                                                                                                                                                                       |
| `components/ContentLogTerminal.vue`      | 抽出 `LogLevel`、内联 props、内联 emits → `components/ContentLogTerminal.types.ts`                                                                                                                                                                                                                                                                                                                                                       |
| `components/ContentStreamingPreview.vue` | 抽出内联 props → `components/ContentStreamingPreview.types.ts`                                                                                                                                                                                                                                                                                                                                                                           |

### B 组 — `components/shared/`

| 文件                          | 要求                                                                                                                                                                                                                                                                                                                                                                                |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `shared/PlatformIcon.vue`     | 把本地 `PlatformIconProps` 移入已存在的 `shared/PlatformIcon.types.ts`（与 `PlatformIconType` 同文件）                                                                                                                                                                                                                                                                              |
| `shared/ModelSelect.vue`      | 抽出 `ModelSelectValue`（保持 `export`）、`ModelOption`、内联 props、内联 emits → `shared/ModelSelect.types.ts`；**并同步更新外部引用** `pages/InspectionTaskDetail.vue` 与 `pages/InspectionEdit.vue`（原 `import ModelSelect, { type ModelSelectValue } from '@/components/shared/ModelSelect.vue'` → 组件默认导入保留，类型改为从 `@/components/shared/ModelSelect.types` 导入） |
| `shared/SegmentedControl.vue` | 抽出内联 props、内联 emits → `shared/SegmentedControl.types.ts`                                                                                                                                                                                                                                                                                                                     |
| `shared/Modal.vue`            | 抽出内联 props、内联 emits → `shared/Modal.types.ts`                                                                                                                                                                                                                                                                                                                                |
| `shared/StatCard.vue`         | 抽出内联 props → `shared/StatCard.types.ts`                                                                                                                                                                                                                                                                                                                                         |
| `shared/StatusBadge.vue`      | 抽出内联 props → `shared/StatusBadge.types.ts`                                                                                                                                                                                                                                                                                                                                      |

### C 组 — `components/layout/`

| 文件                        | 要求                                                                                                                   |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `layout/AppLayout.vue`      | 若有任何类型声明/内联 props/emits，抽出 → `layout/AppLayout.types.ts`（无则跳过并说明）                                |
| `layout/AppHeader.vue`      | 抽出内联 props、内联 emits → `layout/AppHeader.types.ts`                                                               |
| `layout/AppSidebar.vue`     | 抽出 `SidebarGroupConfig`、`MenuEntry`、`MenuGroup`、`MenuItem`、内联 props、内联 emits → `layout/AppSidebar.types.ts` |
| `layout/PageHeader.vue`     | 抽出内联 props → `layout/PageHeader.types.ts`                                                                          |
| `layout/RegionSwitcher.vue` | 若有任何类型声明/内联 props/emits，抽出 → `layout/RegionSwitcher.types.ts`（无则跳过并说明）                           |

### D 组 — `components/rbac/`

| 文件                              | 要求                                                                                                                     |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `rbac/RoleInheritanceTree.vue`    | 抽出 `RoleRef`、`InheritanceNode`、`InheritanceData`、内联 props、内联 emits → `rbac/RoleInheritanceTree.types.ts`       |
| `rbac/PermissionModuleCard.vue`   | 抽出 `ModuleItem`、`Props`（改名为 `PermissionModuleCardProps`）、内联 emits → `rbac/PermissionModuleCard.types.ts`      |
| `rbac/ResourcePermissionCard.vue` | 抽出 `Role`、`Resource`、`Permission`、`RolePermission`、内联 props、内联 emits → `rbac/ResourcePermissionCard.types.ts` |

### E 组 — `components/ImageEditor/` 顶层与面板

| 文件                                        | 要求                                                                           |
| ------------------------------------------- | ------------------------------------------------------------------------------ |
| `ImageEditor/ImageEditor.vue`               | 抽出内联 props、内联 emits → 追加进已存在的 `ImageEditor/ImageEditor.types.ts` |
| `ImageEditor/ImageEditorModal.vue`          | 抽出内联 props、内联 emits → `ImageEditor/ImageEditorModal.types.ts`           |
| `ImageEditor/components/panel-crop.vue`     | 抽出内联 props、内联 emits → `ImageEditor/components/panel-crop.types.ts`      |
| `ImageEditor/components/panel-text.vue`     | 同上 → `panel-text.types.ts`                                                   |
| `ImageEditor/components/panel-draw.vue`     | 同上 → `panel-draw.types.ts`                                                   |
| `ImageEditor/components/panel-filter.vue`   | 同上 → `panel-filter.types.ts`                                                 |
| `ImageEditor/components/panel-sticker.vue`  | 同上 → `panel-sticker.types.ts`                                                |
| `ImageEditor/components/panel-compress.vue` | 同上 → `panel-compress.types.ts`                                               |

### F 组 — `components/ImageEditor/` 引擎 / composables / config

| 文件                                                                                                                                    | 要求                                                                                                                                                                                                                                                                                                             |
| --------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ImageEditor/engine/base/BaseEngine.ts`                                                                                                 | 把 `export interface BaseEngine` 抽到 `engine/base/BaseEngine.types.ts`（其引用的 `BaseLayer` / `EditorMode` / `EngineCapabilities` / `EngineOptions` / `Viewport` 从 `../../ImageEditor.types` import type）；`BaseEngine.ts` 保留 `AbstractEngine` 类并 `import type { BaseEngine } from './BaseEngine.types'` |
| `ImageEditor/engine/index.ts`                                                                                                           | 原 `export type { BaseEngine } from './base/BaseEngine'` 改为 `from './base/BaseEngine.types'`；`EngineType` 改为从 `../config/engine.config.types` 导入                                                                                                                                                         |
| `ImageEditor/config/engine.config.ts`                                                                                                   | 把 `export type EngineType` 抽到 `config/engine.config.types.ts`；`engine.config.ts` 保留 `ENGINE_TYPE` 常量并从该类型文件 `import type { EngineType }`                                                                                                                                                          |
| `ImageEditor/engine/powerful/FabricAdapter.ts`                                                                                          | 抽出本地 `type TaggedObject` → `engine/powerful/FabricAdapter.types.ts`                                                                                                                                                                                                                                          |
| `ImageEditor/engine/lightweight/CanvasEngine.ts`                                                                                        | 抽出本地 `interface DragState` → `engine/lightweight/CanvasEngine.types.ts`                                                                                                                                                                                                                                      |
| `ImageEditor/composables/useEditor.ts`                                                                                                  | 抽出 `export interface TextLayerOptions` → `composables/useEditor.types.ts`（若该类型被其它文件引用，一并更新引用路径）                                                                                                                                                                                          |
| `ImageEditor/engine/powerful/FabricEngine.ts`、`FabricEvent.ts`、`engine/lightweight/LayerManager.ts`、`engine/lightweight/render/*.ts` | 若因上述类型迁移导致 import 路径需要更新，一并修正；这些文件本身若含类型声明也按规范抽出                                                                                                                                                                                                                         |
