# ConditionBuilder 条件构建器 · 技术说明与使用指南

> **版本**：v1.0 · **更新**：2026-09-17 · **技术栈**：Vue 3 (`<script setup>` + TypeScript) · Element Plus · Arco Design
>
> **文档目标**：说明 `ConditionBuilder` 的组件定位、数据模型、Props / 事件约定、字段与规则配置方式，以及如何在业务页面中集成，并给出表达式求值、SQL 翻译等派生能力的使用方法。

---

## 目录

1. [组件定位](#1-组件定位)
2. [目录与模块划分](#2-目录与模块划分)
3. [数据模型](#3-数据模型)
4. [快速开始](#4-快速开始)
5. [Props 参考](#5-props-参考)
6. [事件与实例方法](#6-事件与实例方法)
7. [字段与变量组配置](#7-字段与变量组配置)
8. [运算符与取值控件](#8-运算符与取值控件)
9. [逻辑模式与嵌套](#9-逻辑模式与嵌套)
10. [互斥 / 先决 / 联动 / 运算符限定规则](#10-互斥--先决--联动--运算符限定规则)
11. [派生工具函数](#11-派生工具函数)
12. [子组件职责](#12-子组件职责)
13. [页面集成参考](#13-页面集成参考)
14. [注意事项与边界](#14-注意事项与边界)

---

## 1. 组件定位

`ConditionBuilder` 是一个**可视化条件表达式编辑器**，把「变量 + 运算符 + 取值」的布尔组合编辑为结构化 JSON，而不是让用户手写字符串或 SQL。

它解决三个问题：

| 问题             | 组件的做法                                                                           |
| ---------------- | ------------------------------------------------------------------------------------ |
| 条件长什么样     | 嵌套的条件组（`and` / `or`）→ 结构化的 `ConditionGroup` 树，天然可序列化、可回显     |
| 条件能选什么     | 由页面通过 `fieldOptions` / `fieldGroups` 声明变量，组件负责类型与运算符联动         |
| 条件之间互相约束 | 通过 `rules` 声明互斥 / 先决 / 联动 / 运算符限定规则，组件实时锁定变量、取值与运算符 |

组件**不关心**条件最终被谁消费。它只产出结构，页面按需选择三种消费方式：

- 转成可读表达式：`buildConditionExpression`
- 本地对数据求值：`evaluateConditionGroup`
- 翻译成参数化 SQL：`buildConditionStatement`

---

## 2. 目录与模块划分

```
frontend/src/components/ConditionBuilder/
├── ConditionBuilder.vue            # 对外主组件（唯一推荐入口）
├── ConditionBuilder.types.ts       # 全部类型定义（Props / 节点 / 规则 / SQL）
├── ConditionGroupEditor.vue        # 条件组渲染与操作（可递归）
├── ConditionItemRow.vue            # 单条条件行（变量 / 运算符 / 取值）
├── ConditionConnector.vue          # 行间「且 / 或」连接器
├── ConditionValueControl.vue       # 取值控件（按字段类型切换）
├── ConditionRuleEditor.vue         # 互斥 / 先决 / 联动 / 运算符限定规则编辑器
├── ConditionJsonInput.vue          # JSON 编辑框（CodeMirror + 变量补全）
├── ConditionDocDrawer.vue          # 内置说明抽屉（字段 / 运算符 / 条件 / 规则 / SQL）
├── conditionOperator.ts            # 节点工具 + 运算符元数据 + 表达式 + 求值
├── conditionRules.ts               # 规则解析、锁定、取值限制、草稿转换
├── conditionSql.ts                 # 条件树 → 参数化 WHERE / 完整 SQL
└── conditionSqlMapping.ts          # 变量 ↔ 真实表字段的映射表
```

分层原则：**`.vue` 只负责渲染与交互，纯逻辑全部落在 `condition*.ts`**，因此表达式、求值、SQL 都可在页面或单测中脱离组件直接调用。

---

## 3. 数据模型

### 3.1 节点树

`v-model` 绑定的是**根条件组**，子节点为「条件行」或「嵌套条件组」：

```ts
type ConditionNode = ConditionItem | ConditionGroup

interface ConditionItem {
  id: string
  nodeType: 'item'
  logic: ConditionLogic // 该行与「上一行」的连接关系
  field: string // 变量编码
  operator: ConditionOperator
  value: string // 统一用字符串承载，按字段类型解析
  granularity?: ConditionValueGranularity // 日期时间字段的粒度
}

interface ConditionGroup {
  id: string
  nodeType: 'group'
  logic: ConditionLogic // 该组默认连接关系
  scope?: string // 固定变量组 key（组件维护）
  groupedByLock?: boolean // 因「同变量锁定」自动聚合出的子组
  children: ConditionNode[]
}
```

要点：

- **首个子节点的 `logic` 被忽略**（前面没有可连接的对象），第一个有效连接符从第二个子节点开始生效。
- 语义等价于「且优先于或」，与 SQL 一致；组件不做除此之外的隐式重排。
- `value` 恒为 `string`：数值按 `Number` 解析、布尔按 `true/1/yes/是` 归一化，多值用中英文逗号分隔，范围用 `~` 分隔（见 [8.3](#83-多值与范围)）。

### 3.2 结构示例

界面：`标题 包含 "发布" 且 ( 平台 等于 抖音 或 平台 等于 小红书 )`

```json
{
  "id": "condition-root",
  "nodeType": "group",
  "logic": "and",
  "children": [
    {
      "id": "cond_1",
      "nodeType": "item",
      "logic": "and",
      "field": "title",
      "operator": "contains",
      "value": "发布"
    },
    {
      "id": "cond_2",
      "nodeType": "group",
      "logic": "and",
      "children": [
        {
          "id": "cond_3",
          "nodeType": "item",
          "logic": "and",
          "field": "platform",
          "operator": "eq",
          "value": "douyin"
        },
        {
          "id": "cond_4",
          "nodeType": "item",
          "logic": "or",
          "field": "platform",
          "operator": "eq",
          "value": "xiaohongshu"
        }
      ]
    }
  ]
}
```

---

## 4. 快速开始

### 4.1 最小用法

```vue
<template>
  <ConditionBuilder v-model="group" :field-options="fieldOptions" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ConditionBuilder from '@/components/ConditionBuilder/ConditionBuilder.vue'
import type {
  ConditionFieldOption,
  ConditionGroup,
} from '@/components/ConditionBuilder/ConditionBuilder.types'
import {
  createConditionGroup,
  createConditionItem,
} from '@/components/ConditionBuilder/conditionOperator'

const fieldOptions: ConditionFieldOption[] = [
  { value: 'title', label: '标题', type: 'string' },
  {
    value: 'status',
    label: '状态',
    type: 'select',
    options: [{ label: '已发布', value: 'published' }],
  },
]

const group = ref<ConditionGroup>(createConditionGroup([createConditionItem('title')]))
</script>
```

> `modelValue` 可省略。省略或无条件时组件内部用一张空根组渲染，用户操作后通过 `update:modelValue` 回写。

### 4.2 推荐用法（变量组 + 远程搜索 + 规则 + 消费链）

```vue
<template>
  <ConditionBuilder
    v-model="group"
    :field-options="fieldOptions"
    :field-groups="fieldGroups"
    :scoped-groups="scopedGroups"
    :load-field-options="loadConditionFieldOptions"
    :rule-field-options="allFieldOptions"
    :rules="rules"
    :max-items="8"
    :max-depth="3"
    logic-mode="uniform"
  />

  <ConditionRuleEditor
    v-model:rules="rules"
    :field-options="allFieldOptions"
    :field-groups="allFieldGroups"
    :field-effects="ruleContext.fieldEffects"
    :rule-states="ruleStates"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import ConditionBuilder from '@/components/ConditionBuilder/ConditionBuilder.vue'
import ConditionRuleEditor from '@/components/ConditionBuilder/ConditionRuleEditor.vue'
import { resolveConditionRules } from '@/components/ConditionBuilder/conditionRules'
import { buildConditionStatement } from '@/components/ConditionBuilder/conditionSql'

// 规则上下文：把 rules 解析成「变量锁定 / 取值限制 / 运算符限制 / 冲突」，可直接喂给编辑器做态展示
const ruleContext = computed(() =>
  resolveConditionRules(
    group.value,
    rules.value,
    allFieldOptions,
    fieldOptions.map((f) => f.value),
  ),
)

// 消费：翻译成参数化 SQL
const statement = computed(() => buildConditionStatement(group.value, fieldOptions, { limit: 100 }))
</script>
```

完整可运行示例见 `frontend/src/pages/ConditionTest.vue`。

---

## 5. Props 参考

### 5.1 `ConditionBuilder`

| Prop               | 类型                                                        | 默认值                             | 说明                                                   |
| ------------------ | ----------------------------------------------------------- | ---------------------------------- | ------------------------------------------------------ |
| `modelValue`       | `ConditionGroup`                                            | —                                  | 根条件组，`v-model` 绑定；组件会归一化后回写           |
| `fieldOptions`     | `ConditionFieldOption[]`                                    | `[]`                               | 可用于「非变量组模式」的全部变量                       |
| `fieldGroups`      | `ConditionFieldGroup[]`                                     | `[]`                               | 变量分组，用于「可用变量」悬浮提示分组与远程搜索上下文 |
| `scopedGroups`     | `ConditionFieldGroup[]`                                     | `[]`                               | 启用「固定变量组」模式；`active: false` 表示未勾选     |
| `loadFieldOptions` | `(query, signal?, ctx?) => Promise<ConditionFieldOption[]>` | —                                  | 变量下拉远程搜索；`ctx.scope` 为变量组 key             |
| `ruleFieldOptions` | `ConditionFieldOption[]`                                    | 同 `fieldOptions`                  | 规则可选变量范围（通常传全量变量，不受变量组勾选影响） |
| `rules`            | `ConditionRule[]`                                           | `[]`                               | 互斥 / 先决 / 联动 / 运算符限定规则                    |
| `disabled`         | `boolean`                                                   | `false`                            | 只读态（整体降透明度 + 屏蔽交互）                      |
| `maxItems`         | `number`                                                    | `0`                                | 单组条件数量上限，`0` 表示不限制                       |
| `maxDepth`         | `number`                                                    | `3`                                | 子条件组嵌套上限（**根组不计入层数**）                 |
| `logicEditable`    | `boolean`                                                   | `true`                             | 是否允许用户切换「且 / 或」                            |
| `logicMode`        | `'uniform' \| 'mixed'`                                      | `'uniform'`                        | 逻辑编辑模式，见 [第 9 节](#9-逻辑模式与嵌套)          |
| `addText`          | `string`                                                    | `'添加条件'`                       | 添加条件按钮文案                                       |
| `addGroupText`     | `string`                                                    | `'添加子条件组'`                   | 添加子组按钮文案                                       |
| `clearText`        | `string`                                                    | `'清空'`                           | 清空按钮文案                                           |
| `emptyText`        | `string`                                                    | `'暂无判断条件，点击下方按钮添加'` | 空态文案                                               |
| `ungroupText`      | `string`                                                    | `'解组'`                           | 解组按钮文案                                           |

### 5.2 字段与变量组

```ts
interface ConditionFieldOption {
  value: string // 变量编码，即条件里的 field
  label?: string // 展示名，缺省时回退为 value
  type?: ConditionValueType // 决定可用运算符与取值控件
  description?: string // 悬浮提示中的说明
  options?: ConditionFieldOptionValue[] // 枚举静态候选项
  loadOptions?: ConditionFieldOptionLoader // 枚举远程搜索
  queryable?: boolean // false 表示仅用于编辑，不参与 SQL 生成
}

interface ConditionFieldGroup {
  key: string
  label: string
  description?: string
  active?: boolean // scopedGroups 中 false = 未勾选
  fields: ConditionFieldOption[]
}
```

`ConditionValueType` 支持：`string`（文本）、`number`（数值）、`boolean`（布尔）、`date`（日期）、`datetime`（日期时间）、`time`（时间）、`select`（枚举）。**类型决定运算符可选范围与取值控件形态**。

### 5.3 `ConditionRuleEditor`

| Prop           | 类型                                 | 默认值  | 说明                                   |
| -------------- | ------------------------------------ | ------- | -------------------------------------- |
| `rules`        | `ConditionRule[]`                    | `[]`    | `v-model:rules` 绑定                   |
| `fieldOptions` | `ConditionFieldOption[]`             | `[]`    | 规则可选变量                           |
| `fieldGroups`  | `ConditionFieldGroup[]`              | `[]`    | 变量选择下拉分组                       |
| `fieldEffects` | `ConditionRuleFieldEffect[]`         | `[]`    | 变量受影响状态（生效 / 部分 / 未生效） |
| `ruleStates`   | `Record<string, ConditionRuleState>` | `{}`    | 规则命中态，用于列表状态标签           |
| `disabled`     | `boolean`                            | `false` | 只读态                                 |

### 5.4 其余子组件

| 组件                 | 关键 Props                                                                         | 用途                                                                 |
| -------------------- | ---------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `ConditionJsonInput` | `modelValue` / `fieldOptions` / `fieldGroups` / `placeholder` / `height` / `error` | CodeMirror JSON 编辑器，带变量与枚举补全                             |
| `ConditionDocDrawer` | `visible` / `activeTab` / `schemaFields` / `operatorDocs`                          | 内置说明抽屉（`schema` / `operator` / `condition` / `rule` / `sql`） |

---

## 6. 事件与实例方法

| 事件                | 载荷             | 触发时机                             |
| ------------------- | ---------------- | ------------------------------------ |
| `update:modelValue` | `ConditionGroup` | 任意结构变化（含挂载时的归一化回写） |
| `change`            | `ConditionGroup` | 与 `update:modelValue` 同时触发      |

> 组件挂载时会执行一次**归一化**（变量组归档、锁定重排、失效组剪枝）。若结果与传入值不同，会立即回写 `modelValue`。因此「挂载即收到一次 `change`」是预期行为，页面**不要**在 `change` 里做重请求之类的副作用，除非做了去重。

通过 `ref` 暴露的方法：

```ts
const builderRef = ref<InstanceType<typeof ConditionBuilder>>()
builderRef.value?.addItem() // 等价于点击「添加条件」
builderRef.value?.clear() // 等价于点击「清空」
```

---

## 7. 字段与变量组配置

### 7.1 远程搜索

两处支持异步远程搜索，**都是 250ms 防抖**，并通过 `AbortSignal` 取消过期请求：

```ts
// 1) 变量下拉远程搜索（ConditionBuilder.loadFieldOptions）
async function loadConditionFieldOptions(
  query: string,
  signal?: AbortSignal,
  context?: { scope?: string },
) {
  // context.scope = 当前固定变量组 key；接口应按 scope 过滤，避免加载到组外变量
  const { data } = await api.get('/api/condition-fields', {
    params: { query, scope: context?.scope },
    signal,
  })
  return data as ConditionFieldOption[]
}

// 2) 枚举取值远程搜索（ConditionFieldOption.loadOptions）
const regionField: ConditionFieldOption = {
  value: 'region',
  label: '地区',
  type: 'select',
  loadOptions: async (query, signal) => {
    const { data } = await api.get('/api/regions', { params: { query }, signal })
    return data as ConditionFieldOptionValue[] // [{ label: '北京', value: 'beijing' }]
  },
}
```

实现要点（照做可避免踩坑）：

- 用请求序号 + `signal.aborted` 双保险丢弃过期响应，防止慢请求覆盖新结果。
- 异步枚举可以不配置初始 `options`，组件会按需拉取。

### 7.2 变量组（`scopedGroups`）行为

传入 `scopedGroups` 即进入「固定变量组」模式：

| 勾选情况    | 渲染与输出                                                            |
| ----------- | --------------------------------------------------------------------- |
| 0 组或 1 组 | 不套固定子组，条件**平铺**在根级（`scope` 子组不出现在 `v-model` 中） |
| ≥ 2 组      | 每个已勾选组固定生成一个子组，组头为变量组名，组内只能选该组的变量    |

- 固定子组**不可解组或删除**；「清空」下放到各组，只清空该组条件。
- 启用变量组后，根级不再显示「添加子条件组」；嵌套子组需在组内用条件行尾的「+ 并且满足」创建。
- 取消勾选某组：其子组不再输出，组内条件平铺保留在根级并标记为无效；重新勾选后按变量归属自动归位。
- 根级若存在无法归属任何可用变量组的普通条件组，归一化时会被解包平铺（保留条件、去掉外壳）。

### 7.3 同变量锁定（`groupedByLock`）

对**数值 / 日期 / 日期时间 / 时间**类型，同一条件组内出现 ≥2 条相同变量时，该组及其子组的变量名被锁定为这个变量且不可修改；删到不足 2 条时自动解锁。

锁定组行为：新增条件会追加空行并把原锁定条目聚成一个子组；在条件行点「+ 并且满足」则就地包成子组（组内自动填入锁定变量），不打散其他条目。组头实时显示该组完整表达式，过长省略、悬浮可查看全文。

---

## 8. 运算符与取值控件

### 8.1 运算符表

| value          | 标签     | 符号     | 适用类型                      | 需取值 | SQL 翻译                              |
| -------------- | -------- | -------- | ----------------------------- | ------ | ------------------------------------- |
| `eq`           | 等于     | `=`      | 全部                          | 是     | `col = ?`                             |
| `ne`           | 不等于   | `≠`      | 全部                          | 是     | `col != ?`                            |
| `gt`           | 大于     | `>`      | 数值 / 日期 / 日期时间 / 时间 | 是     | `col > ?`                             |
| `gte`          | 大于等于 | `≥`      | 同上                          | 是     | `col >= ?`                            |
| `lt`           | 小于     | `<`      | 同上                          | 是     | `col < ?`                             |
| `lte`          | 小于等于 | `≤`      | 同上                          | 是     | `col <= ?`                            |
| `contains`     | 包含     | `IN`     | 全部                          | 是     | `LIKE %?%` / `BETWEEN` / `IN`         |
| `not_contains` | 不包含   | `NOT IN` | 全部                          | 是     | `NOT LIKE` / `NOT BETWEEN` / `NOT IN` |
| `is_null`      | 为空     | 为空     | 全部                          | 否     | `(col IS NULL OR col = '')`           |

`contains` / `not_contains` 的翻译随字段类型变化：

- 文本 → `LIKE` / `NOT LIKE`（多值分别用 `OR` / `AND` 连接）
- 数值、日期、日期时间、时间 → **闭区间比较**，可只填一端
- 枚举、布尔 → `IN` / `NOT IN`

> 当命中「运算符限定」规则时，条件行只会展示规则集合内的运算符；已有条件不在限制内时会自动切换到第一个允许运算符；多条规则同时命中时取交集。

### 8.2 日期时间粒度

日期 / 日期时间 / 时间字段在运算符右侧可切换粒度：**日期时间 / 仅日期 / 仅时间**。切换时已填值会按新粒度截取（`2026-10-03 15:30:00` → `2026-10-03` 或 `15:30:00`）。生成 SQL 时列会包一层 `DATE(col)` / `TIME(col)` 再参与比较。

### 8.3 多值与范围

| 场景                         | 输入方式         | `value` 形态                           |
| ---------------------------- | ---------------- | -------------------------------------- |
| 枚举 / 布尔「等于 / 不等于」 | 多选             | `douyin,xiaohongshu`（中英文逗号分隔） |
| 数值「包含 / 不包含」        | 数值范围         | `10~100`，可只填一端                   |
| 日期 / 时间「包含 / 不包含」 | 日期（时间）范围 | `2026-09-26~2026-10-12`，可只填一端    |

> 多值枚举的「等于」在 SQL 中**逐个展开并 AND 连接**（`col = ? AND col = ?`），不是 `IN`——这是刻意保留的语义，与界面展示保持一致。

---

## 9. 逻辑模式与嵌套

`logicMode` 决定「且 / 或」如何被编辑：

| 模式              | 表现                                              | 归一化                                                                               |
| ----------------- | ------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `uniform`（默认） | 不显示行内连接符，仅每层底部提供「且 / 或」切换器 | 切换模式时按该层第一个子项的逻辑统一整层；点底部切换器一次性改写该层（不改节点结构） |
| `mixed`           | 每行左侧可切换「且 / 或」，保留连接符与 `IF` 标记 | 逐行独立，不做统一                                                                   |

从 `mixed` 切到 `uniform` 时会触发一次统一，之后不再改写。

嵌套约束：子条件组最多 `maxDepth` 层（默认 3，**根组不计层数**）。达到上限后「添加子条件组」隐藏、「+ 并且满足」置灰并提示。

删除行为：条件组只剩 1 个条件时自动还原为单条件（`removeNodeWithCollapse`）；「+ 并且满足」会就地把该行包成一个「且」子组并追加一条空条件。

---

## 10. 互斥 / 先决 / 联动 / 运算符限定规则

规则是**声明式**的：由页面维护 `ConditionRule[]`，组件负责解析成锁定与限制并实时反馈状态。

```ts
type ConditionRuleType = 'mutual_exclusive' | 'prerequisite' | 'linkage' | 'operator_limit'

// 互斥：when 命中时，targets 中的变量不可同时使用
interface ConditionMutualExclusiveRule {
  type: 'mutual_exclusive'
  id: string
  name: string
  when: ConditionRuleSelector
  targets: ConditionRuleSelector[]
}

// 先决：when 命中时，targets 中的变量才被允许使用
interface ConditionPrerequisiteRule {
  type: 'prerequisite'
  id: string
  name: string
  when: ConditionRuleSelector
  targets: ConditionRuleSelector[]
}

// 联动：when 命中时，field 的取值被限制在 allowedValues 内
interface ConditionLinkageRule {
  type: 'linkage'
  id: string
  name: string
  when: ConditionRuleSelector
  field: string
  operator: ConditionOperator
  allowedValues: string[]
}

// 运算符限定：when 命中时，field 的可用运算符被限制在 operators 内（至少一个）
interface ConditionOperatorLimitRule {
  type: 'operator_limit'
  id: string
  name: string
  when: ConditionRuleSelector
  field: string
  operators: ConditionOperator[]
}
```

完整配置示例：

```ts
import type { ConditionOperatorLimitRule } from '@/components/ConditionBuilder/ConditionBuilder.types'

const operatorLimitRule: ConditionOperatorLimitRule = {
  id: 'rule-douyin-title-operator-limit',
  type: 'operator_limit',
  name: '抖音标题运算符限定',
  description: '平台包含 抖音 时，「标题」仅允许使用 包含 / 不包含 运算符',
  when: { field: 'platform', operator: 'contains', value: 'douyin' },
  field: 'title',
  operators: ['contains', 'not_contains'],
  message: '平台含抖音时「标题」仅可使用 包含 / 不包含',
}
```

命中后，`title` 条件行的运算符下拉只保留 `contains` / `not_contains`；已有条件使用其他运算符时会自动切换为第一个允许项。多条运算符限定规则同时命中时取交集，冲突提示仍保留在条件行尾。

解析结果 `ConditionRuleContext` 包含五类信息，页面可直接消费：

| 字段                           | 含义                                              |
| ------------------------------ | ------------------------------------------------- |
| `fieldLocks`                   | 命中的变量锁定（原因、来源规则）                  |
| `valueLimits`                  | 命中的取值限制（可选项被过滤或禁用）              |
| `operatorLimits`               | 命中的运算符限制（运算符下拉项被过滤）            |
| `violations`                   | 当前条件与规则的冲突，附 `itemIds` 用于定位       |
| `fieldEffects` / `ruleEffects` | 生效状态：`effective` / `partial` / `ineffective` |

行为约定：

- 规则使用**全量可用变量**（`ruleFieldOptions`），不受变量组勾选影响。
- 规则已激活但目标变量不在当前可用变量组内时标记为「激活未生效」；仅部分目标不可选时为「激活部分未生效」。
- 数值 / 日期 / 时间的范围取值（`a~b`）在互斥与联动校验中分别校验最小值与最大值；只填一端时缺失边界按无穷处理。
- 多条「运算符限定」规则同时命中时取交集；交集为空时目标变量的运算符全部禁用，无法自动切换的冲突仍会保留提示。
- 单条规则命中且存在允许项时，已有条件会自动切换到第一个允许运算符，不再显示运算符限定提示图标。
- 冲突提示出现在条件行尾（橙色感叹号），悬浮可见原因。

---

## 11. 派生工具函数

这些函数不依赖组件实例，页面、Store、单测均可直接调用。

### 11.1 表达式与求值（`conditionOperator.ts`）

```ts
buildConditionExpression(group, fieldOptions) // → '标题 包含 发布 且 (平台 等于 抖音 或 平台 等于 小红书)'
evaluateConditionGroup(group, data, fieldOptions) // → ConditionEvaluation
```

`evaluateConditionGroup` 返回 `{ hasConditions, passed, results }`，其中 `results` 是**扁平化的逐条结果**（含 `fieldLabel` / `operatorLabel` / `expected` / `actual` / `passed` / `depth` / `isFirst`），适合渲染命中明细。

### 11.2 节点工具（`conditionOperator.ts`）

| 函数                                                                                      | 用途                                     |
| ----------------------------------------------------------------------------------------- | ---------------------------------------- |
| `createConditionGroup` / `createConditionItem` / `createConditionId`                      | 构造节点                                 |
| `cloneConditionGroup`                                                                     | 深拷贝（改前先拷贝，避免污染原值）       |
| `collectConditionItems` / `collectConditionGroups`                                        | 扁平收集全部节点                         |
| `updateGroupAt` / `removeNodeAt` / `ungroupNodeAt` / `patchChildItem` / `patchChildLogic` | 按 `path` 做不可变更新                   |
| `isConditionItemEffective`                                                                | 判断条件是否「有效」（变量与取值均已填） |
| `flattenFieldGroups` / `groupConditionFields` / `resolveConditionFieldType`               | 字段辅助                                 |

`path` 是从根出发的子节点下标数组（根为 `[]`）。所有更新函数**返回新对象**，未变化时返回原引用，便于 `computed` / `watch` 做引用比较。

### 11.3 SQL 翻译（`conditionSql.ts`）

```ts
buildConditionWhere(group, fieldOptions, columnMap) // → { hasConditions, where, params, skipped }
formatConditionSql(result) // → 'WHERE ...'
buildConditionStatement(group, fieldOptions, options) // → { hasConditions, sql, params, skipped }
```

`buildConditionStatement` 默认产出完整同表查询：

```sql
SELECT
  title,
  platform
FROM contents
WHERE (title LIKE ? AND platform = ?)
ORDER BY created_at DESC
LIMIT 100
```

可通过 `options` 覆盖 `table` / `columns` / `orderBy` / `limit` / `columnMap`。

**安全约定**：值一律走 `?` 占位符 + `params` 数组；标识符（表名、列名）只能来自 `conditionSqlMapping.ts` 的映射表，绝不拼接用户输入。不可查询的变量会被跳过并记录在 `skipped` 中，页面应显式提示。

默认映射（`CONDITION_SQL_COLUMN_MAP`）：仅「内容属性」组映射到 `contents` 真实列；`fans_count`、`like_count` 等演示变量 `queryable: false`，生成 SQL 时自动跳过。

### 11.4 规则（`conditionRules.ts`）

```ts
resolveConditionRules(group, rules, ruleFieldOptions, availableFields) // → ConditionRuleContext
conditionRuleState(rule, group, ...)                                   // → ConditionRuleState
conditionRuleSummary(rule, fieldOptions)                               // → 一句话描述
conditionRuleOperatorLimits(context, field)                            // → 当前字段的运算符限制
conditionRuleAllowedOperators(limits)                                  // → 交集结果；null 表示未限制
conditionRuleOperatorReason(limits)                                    // → 规则提示文案（条件行默认不展示）
createConditionRuleDraft / conditionRuleToDraft / conditionRuleFromDraft // 编辑器草稿互转
```

---

## 12. 子组件职责

| 组件                    | 职责                                                                     | 对外接口                    |
| ----------------------- | ------------------------------------------------------------------------ | --------------------------- |
| `ConditionBuilder`      | 归一化、变量组归档、规则上下文、命令分发、变量提示；对外唯一入口         | `v-model` + `addItem/clear` |
| `ConditionGroupEditor`  | 渲染一个条件组（含递归子组、连接符、层级逻辑切换、添加 / 清空）          | `@command`                  |
| `ConditionItemRow`      | 单行：变量选择、运算符选择、取值控件、锁定 / 冲突态                      | `@command`                  |
| `ConditionConnector`    | 「且 / 或」连接器（`uniform` 下不可编辑）                                | `@update:logic`             |
| `ConditionValueControl` | 按字段类型切换取值控件（文本 / 数值 / 布尔 / 日期 / 枚举 / 范围 / 多选） | `v-model`                   |
| `ConditionRuleEditor`   | 规则的增删改与状态展示                                                   | `v-model:rules`             |
| `ConditionJsonInput`    | JSON 编辑与校验，带变量 / 枚举补全                                       | `v-model`                   |
| `ConditionDocDrawer`    | 内置说明抽屉                                                             | `visible` / `activeTab`     |

**通信方式**：所有编辑操作由子组件发 `command`（`ConditionCommand`）向上冒泡，统一在 `ConditionBuilder.handleCommand` 中做不可变更新。新增交互时应**复用 `command` 通道**，不要在子组件里直接改 `props`。

```ts
type ConditionCommand =
  | { type: 'set-logic'; path: number[]; logic: ConditionLogic }
  | { type: 'set-level-logic'; path: number[]; logic: ConditionLogic }
  | { type: 'add-item'; path: number[] }
  | { type: 'add-group'; path: number[]; field?: string }
  | { type: 'clear-group'; path: number[] }
  | { type: 'remove-group'; path: number[] }
  | { type: 'ungroup-group'; path: number[] }
  | { type: 'update-item'; path: number[]; index: number; patch: Partial<ConditionItem> }
  | { type: 'remove-item'; path: number[]; index: number }
  | { type: 'wrap-item'; path: number[]; index: number; field?: string }
```

---

## 13. 页面集成参考

`frontend/src/pages/ConditionTest.vue` 是完整的端到端示例，覆盖：

- 变量组勾选与切换（`scopedGroups`）、嵌套上限、逻辑模式切换
- 远程变量搜索与远程枚举搜索（250ms 防抖 + `AbortSignal`）
- 规则编辑器 + 规则上下文（变量锁定、取值限制、运算符限制、冲突提示）
- 三条消费链：可读表达式、本地求值（含逐条命中明细）、参数化 SQL
- JSON 输入 / 输出与说明抽屉
- 示例条件预设一键载入

新页面接入时的推荐顺序：

1. 定义 `fieldOptions`（至少给 `value` / `label` / `type`）与可选的 `fieldGroups`。
2. `v-model` 绑一个 `ConditionGroup`，初始值用 `createConditionGroup(...)`。
3. 需要变量分组就加 `scopedGroups`；需要数据约束就加 `rules`。
4. 按落地方式选 `buildConditionExpression` / `evaluateConditionGroup` / `buildConditionStatement` 之一做消费。

---

## 14. 注意事项与边界

1. **`change` 会在挂载时触发一次**（归一化回写），页面勿在回调里做重副作用。
2. **组件会改写传入结构**：变量组归档、锁定重排、失效组剪枝均在归一化中进行。需要保留原始结构时先 `cloneConditionGroup`。
3. **不要直接改 `props`**：所有变更走 `command` 或工具函数返回的新对象。
4. **`value` 恒为字符串**：数值与布尔由消费方（求值 / SQL）按字段类型转换，页面里不要假设它是 `number`。
5. **`queryable: false` 的变量不参与 SQL**：生成语句时会被跳过，务必展示 `skipped` 提示，否则用户会以为条件已生效。
6. **标识符不来自用户输入**：SQL 的表名与列名只允许来自 `columnMap`；`buildConditionWhere` 只负责值参数化。
7. **`path` 是位置索引**：节点增删后既有 `path` 立即失效，不要跨渲染缓存。
8. **默认值即现状约定**：`maxDepth` 默认 3、`logicMode` 默认 `uniform`、防抖 250ms、默认 SQL 表 `contents` + `LIMIT 100`；改动前先确认调用方。

---

## 附：相关文件

| 文件                                         | 说明               |
| -------------------------------------------- | ------------------ |
| `frontend/src/components/ConditionBuilder/*` | 组件与纯逻辑实现   |
| `frontend/src/pages/ConditionTest.vue`       | 端到端示例页面     |
| `frontend/src/pages/ConditionTest.types.ts`  | 示例页面的辅助类型 |
