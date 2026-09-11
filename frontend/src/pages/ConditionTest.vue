<template>
  <div class="page-main">
    <PageHeader
      title="条件组件测试"
      subtitle="验证判断条件输入组件：变量提示输入、比较运算符、值编辑器、同层混用 且/或（且 优先）与括号分组嵌套，并即时生成表达式、SQL 与校验结果"
    >
      <template #actions>
        <a-button
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="resetPreset"
        >
          <template #icon><IconRefresh :size="13" /></template>
          重置示例
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <div class="flex flex-col lg:flex-row gap-5">
        <div class="flex flex-col gap-5 lg:min-w-[max-content] lg:shrink-0">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-2">
                <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">条件配置</h3>
                <a-tag size="small" color="arcoblue" class="!m-0">{{ itemCount }} 条</a-tag>
                <a-tag v-if="groupCount" size="small" color="purple" class="!m-0">
                  {{ groupCount }} 个分组
                </a-tag>
              </div>
              <span class="text-[12px] text-[#86868B]">变量共 {{ fieldOptions.length }} 个</span>
            </div>

            <ConditionBuilder
              v-model="group"
              :field-options="fieldOptions"
              :rules="rules"
              :max-items="8"
              :max-depth="3"
            />

            <div class="mt-5">
              <div class="text-[13px] font-semibold text-[#1D1D1F] mb-2">示例条件（点击载入）</div>
              <div class="flex flex-wrap gap-2 contain-inline-size">
                <a-button
                  v-for="preset in presets"
                  :key="preset.key"
                  size="mini"
                  @click="applyPreset(preset)"
                >
                  {{ preset.label }}
                </a-button>
              </div>
              <div class="text-[11px] text-[#86868B] mt-2">
                每行左侧可切换 且 / 或，且 优先级高于 或（与 SQL 一致）；子条件组最多嵌套 3
                层，支持解组或整体删除。
              </div>
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center gap-2 mb-3">
              <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">互斥与关联规则</h3>
              <a-tag size="small" color="arcoblue" class="!m-0">{{ rules.length }} 条</a-tag>
            </div>

            <ConditionRuleEditor
              v-model:rules="rules"
              :field-options="fieldOptions"
              :rule-states="ruleStates"
            />

            <div class="text-[11px] text-[#86868B] mt-3">
              规则命中后会禁用或过滤不可选的变量与取值；条件行尾的橙色感叹号可查看冲突原因。
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center justify-between mb-3">
              <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">测试数据（JSON）</h3>
              <div class="flex items-center gap-2">
                <a-button
                  v-for="preset in dataPresets"
                  :key="preset.key"
                  size="mini"
                  @click="dataText = JSON.stringify(preset.data, null, 2)"
                >
                  {{ preset.label }}
                </a-button>
              </div>
            </div>
            <a-textarea
              v-model="dataText"
              :auto-size="{ minRows: 6, maxRows: 14 }"
              :error="Boolean(dataError)"
              placeholder="输入用于校验的 JSON 对象"
            />
            <div v-if="dataError" class="text-[12px] text-[#FF3B30] mt-2">{{ dataError }}</div>
          </div>
        </div>

        <div class="flex flex-col gap-5 lg:flex-1 lg:min-w-[400px]">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0 mb-3">表达式与校验</h3>

            <div class="text-[12px] text-[#86868B] mb-1">生成表达式</div>
            <div class="expression">{{ expression || '（暂无有效条件）' }}</div>

            <div class="flex items-center gap-2 mt-4 mb-2">
              <span class="text-[13px] font-semibold text-[#1D1D1F]">校验结果</span>
              <a-tag
                v-if="evaluation.hasConditions"
                :color="evaluation.passed ? 'green' : 'red'"
                class="!m-0"
              >
                {{ evaluation.passed ? '命中' : '未命中' }}
              </a-tag>
              <a-tag v-else color="gray" class="!m-0">条件不完整</a-tag>
            </div>

            <div v-if="evaluation.results.length" class="result-table">
              <div class="result-row result-row--head">
                <span>变量</span>
                <span>运算符</span>
                <span>期望值</span>
                <span>实际值</span>
                <span>结果</span>
              </div>
              <div v-for="row in evaluation.results" :key="row.item.id" class="result-row">
                <span class="truncate" :style="{ paddingLeft: `${(row.depth - 1) * 16}px` }">
                  <span v-if="row.depth > 1" class="result-depth">└ </span>{{ row.fieldLabel }}
                </span>
                <span>{{ row.operatorLabel }}</span>
                <span class="truncate">{{ row.expected || '—' }}</span>
                <span class="truncate">{{ row.actual }}</span>
                <span :class="row.passed ? 'result-pass' : 'result-fail'">
                  {{ row.passed ? '通过' : '不通过' }}
                </span>
              </div>
            </div>
            <div v-else class="py-4">
              <a-empty description="暂无可校验的条件" />
            </div>

            <div class="flex items-center gap-2 mt-5 mb-2">
              <span class="text-[13px] font-semibold text-[#1D1D1F]">互斥与关联命中结果</span>
              <a-tag v-if="rules.length" :color="ruleResultTagColor" class="!m-0">
                {{ ruleResultTagText }}
              </a-tag>
            </div>

            <div v-if="ruleResultRows.length" class="result-table">
              <div class="result-row result-row--rule result-row--head">
                <span>规则</span>
                <span>类型</span>
                <span>状态</span>
                <span>说明</span>
              </div>
              <div v-for="row in ruleResultRows" :key="row.id" class="result-row result-row--rule">
                <span class="truncate">{{ row.name }}</span>
                <span>{{ row.typeLabel }}</span>
                <span :class="row.stateClass">{{ row.stateLabel }}</span>
                <span class="truncate">{{ row.detail }}</span>
              </div>
            </div>
            <div v-else class="py-4">
              <a-empty description="暂无规则配置" />
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">生成 SQL</h3>
                <a-tag :color="sqlResult.hasConditions ? 'arcoblue' : 'gray'" class="!m-0">
                  {{ sqlResult.params.length }} 个参数
                </a-tag>
              </div>
              <a-button
                type="text"
                size="mini"
                class="!text-[#007AFF] !px-0 !h-auto"
                :disabled="!sqlResult.hasConditions"
                @click="copySql"
              >
                <template #icon><IconCopy :size="13" /></template>
                复制 SQL
              </a-button>
            </div>

            <template v-if="sqlResult.hasConditions">
              <div class="text-[12px] text-[#86868B] mb-1">WHERE 条件（参数化）</div>
              <pre class="sql-output">{{ sqlText }}</pre>
              <div class="text-[12px] text-[#86868B] mt-3 mb-1">参数数组（按 ? 顺序）</div>
              <pre class="sql-output">{{ sqlParamsText }}</pre>
            </template>
            <div v-else class="py-4">
              <a-empty description="暂无可转换的条件" />
            </div>

            <div class="text-[11px] text-[#86868B] mt-3">
              字段使用变量编码；数值按数字绑定、布尔按 1/0
              绑定、文本/枚举/日期/日期时间/时间按字符串绑定；「包含 / 不包含」在字符串字段翻译为
              LIKE / NOT LIKE（多值用 OR / AND
              连接），在枚举、布尔、数值、日期、日期时间、时间字段翻译为 IN / NOT IN；「为空」翻译为
              (col IS NULL OR col = '')。
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0 mb-3">v-model 输出</h3>
            <pre class="json-output">{{ modelJson }}</pre>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0 mb-3">
              输出 JSON 结构字段说明
            </h3>
            <div class="schema-table">
              <div class="schema-row schema-row--head">
                <span>字段</span>
                <span>类型</span>
                <span>作用</span>
              </div>
              <div v-for="field in schemaFields" :key="field.name" class="schema-row">
                <span class="schema-name">{{ field.name }}</span>
                <span class="schema-type">{{ field.type }}</span>
                <span class="schema-desc">{{ field.desc }}</span>
              </div>
            </div>

            <div class="text-[12px] text-[#86868B] mt-4 mb-1">规则配置输出</div>
            <pre class="json-output">{{ rulesJson }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconCopy, IconRefresh } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import ConditionBuilder from '@/components/ConditionBuilder/ConditionBuilder.vue'
import ConditionRuleEditor from '@/components/ConditionBuilder/ConditionRuleEditor.vue'
import type {
  ConditionFieldOption,
  ConditionGroup,
  ConditionItem,
  ConditionLogic,
  ConditionNode,
  ConditionOperator,
  ConditionRule,
} from '@/components/ConditionBuilder/ConditionBuilder.types'
import {
  buildConditionExpression,
  cloneConditionGroup,
  collectConditionGroups,
  collectConditionItems,
  createConditionGroup,
  createConditionId,
  evaluateConditionGroup,
} from '@/components/ConditionBuilder/conditionOperator'
import {
  CONDITION_RULE_STATE_LABELS,
  CONDITION_RULE_TYPE_LABELS,
  conditionRuleState,
  conditionRuleSummary,
  resolveConditionRules,
} from '@/components/ConditionBuilder/conditionRules'
import { buildConditionWhere, formatConditionSql } from '@/components/ConditionBuilder/conditionSql'
import type {
  ConditionDataPreset,
  ConditionPreset,
  ConditionRuleState,
  ConditionSchemaField,
} from './ConditionTest.types'

const fieldOptions: ConditionFieldOption[] = [
  {
    value: 'platform',
    label: '平台',
    type: 'select',
    description: '内容发布所在平台',
    options: [
      { label: '小红书', value: 'xiaohongshu' },
      { label: '抖音', value: 'douyin' },
      { label: '微信视频号', value: 'channels' },
      { label: '微信公众号', value: 'wechat_mp' },
    ],
  },
  { value: 'fans_count', label: '粉丝数', type: 'number', description: '账号当前粉丝数量' },
  { value: 'like_count', label: '点赞数', type: 'number', description: '内容累计点赞数' },
  { value: 'comment_count', label: '评论数', type: 'number', description: '内容累计评论数' },
  { value: 'title', label: '标题', type: 'string', description: '内容标题文本' },
  {
    value: 'published_at',
    label: '发布时间',
    type: 'date',
    description: '内容发布时间，格式 YYYY-MM-DD',
  },
  {
    value: 'created_at',
    label: '创建时间',
    type: 'datetime',
    description: '内容创建时间，格式 YYYY-MM-DD HH:mm:ss',
  },
  {
    value: 'publish_slot',
    label: '发布时段',
    type: 'time',
    description: '每日计划发布的时刻，格式 HH:mm:ss',
  },
  { value: 'is_verified', label: '是否认证', type: 'boolean', description: '账号是否已完成认证' },
  {
    value: 'account_level',
    label: '账号等级',
    type: 'select',
    description: '账号的运营等级',
    options: [
      { label: 'L1', value: 'L1' },
      { label: 'L2', value: 'L2' },
      { label: 'L3', value: 'L3' },
    ],
  },
]

const sampleData: Record<string, unknown> = {
  platform: 'xiaohongshu',
  fans_count: 12800,
  like_count: 860,
  comment_count: 42,
  title: '双十一必买清单｜平价好物合集',
  published_at: '2026-09-01',
  created_at: '2026-09-01 10:30:00',
  publish_slot: '09:30:00',
  is_verified: true,
  account_level: 'L2',
}

function makeItem(
  field: string,
  operator: ConditionOperator,
  value: string,
  logic: ConditionLogic = 'and',
): ConditionItem {
  return { id: createConditionId('item'), nodeType: 'item', logic, field, operator, value }
}

function makeGroup(children: ConditionNode[], logic: ConditionLogic = 'and'): ConditionGroup {
  return { id: createConditionId('group'), nodeType: 'group', logic, children }
}

const presets: ConditionPreset[] = [
  {
    key: 'all-in-one',
    label: '最全案例：9 种运算符 · 7 类字段 · 3 层嵌套 · 且/或混用',
    group: makeGroup([
      makeItem('platform', 'contains', 'xiaohongshu'),
      makeItem('title', 'contains', '双十一, 好物'),
      makeItem('title', 'not_contains', '广告, 推广'),
      makeItem('comment_count', 'is_null', ''),
      makeGroup(
        [
          makeItem('like_count', 'gt', '1000'),
          makeGroup([makeItem('account_level', 'ne', 'L1'), makeItem('is_verified', 'eq', 'true')]),
        ],
        'or',
      ),
      makeItem('published_at', 'gte', '2026-09-01'),
      makeItem('created_at', 'lt', '2026-10-01 00:00:00'),
      makeItem('publish_slot', 'lte', '18:00:00'),
    ]),
  },
  {
    key: 'mixed',
    label: '标题 包含 (双十一) 且 标题 不包含 (平) 或 平台 IN (douyin, xiaohongshu)',
    group: makeGroup([
      makeItem('title', 'contains', '双十一'),
      makeItem('title', 'not_contains', '平'),
      makeItem('platform', 'contains', 'douyin, xiaohongshu', 'or'),
    ]),
  },
  {
    key: 'nested',
    label: '点赞数 > 888 或 (平台 包含 小红书, 抖音 且 粉丝数 = 12800)',
    group: makeGroup([
      makeItem('like_count', 'gt', '888'),
      makeGroup(
        [
          makeItem('platform', 'contains', 'xiaohongshu, douyin'),
          makeItem('fans_count', 'eq', '12800'),
        ],
        'or',
      ),
    ]),
  },
  {
    key: 'and-number',
    label: '粉丝数 ≥ 10000 且 平台 = 小红书',
    group: makeGroup([
      makeItem('fans_count', 'gte', '10000'),
      makeItem('platform', 'eq', 'xiaohongshu'),
    ]),
  },
  {
    key: 'date-boolean',
    label: '发布时间 ≥ 2026-09-01 且 是否认证 = 是',
    group: makeGroup([
      makeItem('published_at', 'gte', '2026-09-01'),
      makeItem('is_verified', 'eq', 'true'),
    ]),
  },
  {
    key: 'datetime-range',
    label: '创建时间 ≥ 2026-09-01 00:00:00 且 发布时段 ≤ 12:00:00',
    group: makeGroup([
      makeItem('created_at', 'gte', '2026-09-01 00:00:00'),
      makeItem('publish_slot', 'lte', '12:00:00'),
    ]),
  },
  {
    key: 'datetime-mixed',
    label: '创建时间 < 2026-10-01 00:00:00 或 发布时段 ≥ 18:00:00',
    group: makeGroup([
      makeItem('created_at', 'lt', '2026-10-01 00:00:00'),
      makeItem('publish_slot', 'gte', '18:00:00', 'or'),
    ]),
  },
  {
    key: 'deep',
    label: '三层嵌套：(评论数 > 10 或 点赞数 > 1000) 且 (平台 包含 抖音, 小红书 或 账号等级 = L3)',
    group: makeGroup([
      makeGroup([
        makeItem('comment_count', 'gt', '10'),
        makeItem('like_count', 'gt', '1000', 'or'),
      ]),
      makeGroup([
        makeItem('platform', 'contains', 'douyin, xiaohongshu'),
        makeItem('account_level', 'eq', 'L3', 'or'),
      ]),
    ]),
  },
  {
    key: 'contains',
    label: '平台 包含 小红书, 抖音 且 账号等级 不包含 L1, L3（枚举 IN）',
    group: makeGroup([
      makeItem('platform', 'contains', 'xiaohongshu, douyin'),
      makeItem('account_level', 'not_contains', 'L1, L3'),
    ]),
  },
  {
    key: 'string-like',
    label: '标题 包含 双十一, 好物 且 标题 不包含 广告, 推广（字符串 LIKE）',
    group: makeGroup([
      makeItem('title', 'contains', '双十一, 好物'),
      makeItem('title', 'not_contains', '广告, 推广'),
    ]),
  },
  {
    key: 'in-number',
    label: '点赞数 包含 860, 1200（数值 IN）',
    group: makeGroup([makeItem('like_count', 'contains', '860, 1200')]),
  },
  {
    key: 'is-null',
    label: '评论数 为空',
    group: makeGroup([makeItem('comment_count', 'is_null', '')]),
  },
]

const dataPresets: ConditionDataPreset[] = [
  { key: 'default', label: '默认数据', data: sampleData },
  {
    key: 'low-fans',
    label: '低粉丝账号',
    data: {
      platform: 'douyin',
      fans_count: 320,
      like_count: 12,
      comment_count: 0,
      title: '开箱测评',
      published_at: '2026-08-12',
      created_at: '2026-08-12 18:05:00',
      publish_slot: '20:00:00',
      is_verified: false,
      account_level: 'L1',
    },
  },
  {
    key: 'missing-comment',
    label: '评论数缺失',
    data: {
      platform: 'channels',
      fans_count: 56000,
      like_count: 2400,
      title: '双十一好物推荐',
      published_at: '2026-10-20',
      created_at: '2026-10-20 08:15:00',
      publish_slot: '07:45:00',
      is_verified: true,
      account_level: 'L3',
    },
  },
]

const rules = ref<ConditionRule[]>([
  {
    id: 'rule-platform-exclusive',
    type: 'mutual_exclusive',
    name: '平台取值互斥',
    description: '「平台」不可同时包含 抖音 与 小红书',
    when: { field: 'platform', operator: 'contains', value: 'douyin' },
    targets: [{ field: 'platform', operator: 'contains', value: 'xiaohongshu' }],
    message: '「平台」不可同时包含 抖音 与 小红书',
  },
  {
    id: 'rule-account-level-prerequisite',
    type: 'prerequisite',
    name: '账号等级先决',
    description: '配置「账号等级」前必须先添加「平台」条件',
    when: { field: 'platform' },
    targets: [{ field: 'account_level' }],
    message: '配置「账号等级」前需先添加「平台」条件',
  },
  {
    id: 'rule-douyin-level-linkage',
    type: 'linkage',
    name: '抖音账号等级联动',
    description: '平台包含 抖音 时，「账号等级」仅可选 L1 / L3',
    when: { field: 'platform', operator: 'contains', value: 'douyin' },
    field: 'account_level',
    operator: 'contains',
    allowedValues: ['L1', 'L3'],
    message: '平台含抖音时「账号等级」仅可选 L1 / L3',
  },
])

const schemaFields: ConditionSchemaField[] = [
  { name: 'id', type: 'string', desc: '节点唯一标识，用于 diff 与精确定位删除' },
  { name: 'nodeType', type: "'item' | 'group'", desc: '节点类型：item=单条条件，group=嵌套条件组' },
  {
    name: 'logic',
    type: "'and' | 'or'",
    desc: '相对同级前一节点的连接符（首项忽略）；and=且、or=或',
  },
  { name: 'field', type: 'string', desc: '引用的变量编码，对应「可用变量」中的 code' },
  {
    name: 'operator',
    type: 'ConditionOperator',
    desc: '比较运算符：eq / ne / gt / gte / lt / lte / contains / not_contains / is_null',
  },
  { name: 'value', type: 'string', desc: '比较值；多值以逗号分隔；is_null 时为空字符串' },
  { name: 'children', type: 'ConditionNode[]', desc: '仅 group 节点：子节点数组（最多嵌套 3 层）' },
]

const group = ref<ConditionGroup>(createConditionGroup())
const dataText = ref(JSON.stringify(sampleData, null, 2))

const itemCount = computed(() => collectConditionItems(group.value).length)

const groupCount = computed(() => collectConditionGroups(group.value).length)

const parsedData = computed<Record<string, unknown>>(() => {
  try {
    const parsed: unknown = JSON.parse(dataText.value)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>
    }
    return {}
  } catch {
    return {}
  }
})

const dataError = computed(() => {
  try {
    const parsed: unknown = JSON.parse(dataText.value)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return '测试数据必须是 JSON 对象'
    }
    return ''
  } catch (error) {
    return `JSON 解析失败：${error instanceof Error ? error.message : '未知错误'}`
  }
})

const expression = computed(() => buildConditionExpression(group.value, fieldOptions))

const evaluation = computed(() =>
  evaluateConditionGroup(group.value, parsedData.value, fieldOptions),
)

const sqlResult = computed(() => buildConditionWhere(group.value, fieldOptions))

const sqlText = computed(() => formatConditionSql(sqlResult.value))

const sqlParamsText = computed(() => JSON.stringify(sqlResult.value.params))

const modelJson = computed(() => JSON.stringify(group.value, null, 2))

const ruleContext = computed(() => resolveConditionRules(group.value, rules.value, fieldOptions))

const ruleStates = computed<Record<string, ConditionRuleState>>(() => {
  const states: Record<string, ConditionRuleState> = {}
  for (const rule of rules.value) {
    states[rule.id] = conditionRuleState(ruleContext.value, rule.id)
  }
  return states
})

const activeRuleCount = computed(() => ruleContext.value.activeRuleIds.length)

const violationCount = computed(() => ruleContext.value.violations.length)

const rulesJson = computed(() => JSON.stringify(rules.value, null, 2))

const RULE_STATE_CLASSES: Record<ConditionRuleState, string> = {
  idle: 'result-idle',
  active: 'result-pass',
  violation: 'result-fail',
}

const ruleResultRows = computed(() =>
  rules.value.map((rule) => {
    const state = ruleStates.value[rule.id] ?? 'idle'
    const violation = ruleContext.value.violations.find((entry) => entry.ruleId === rule.id)
    return {
      id: rule.id,
      name: rule.name,
      typeLabel: CONDITION_RULE_TYPE_LABELS[rule.type],
      stateLabel: CONDITION_RULE_STATE_LABELS[state],
      stateClass: RULE_STATE_CLASSES[state],
      detail: violation?.message ?? conditionRuleSummary(rule, fieldOptions),
    }
  }),
)

const ruleResultTagText = computed(() => {
  if (violationCount.value) return '存在冲突'
  if (activeRuleCount.value) return '命中'
  return '未命中'
})

const ruleResultTagColor = computed(() => {
  if (violationCount.value) return 'red'
  if (activeRuleCount.value) return 'green'
  return 'gray'
})

function applyPreset(preset: ConditionPreset): void {
  group.value = cloneConditionGroup(preset.group)
}

async function copySql(): Promise<void> {
  if (!sqlResult.value.hasConditions) return
  const text = `${sqlText.value}\n-- params: ${sqlParamsText.value}`
  try {
    await navigator.clipboard.writeText(text)
    Message.success('已复制到剪贴板')
  } catch {
    Message.error('复制失败，请手动选择文本复制')
  }
}

function resetPreset(): void {
  group.value = createConditionGroup()
  dataText.value = JSON.stringify(sampleData, null, 2)
}
</script>

<style scoped>
.expression {
  padding: 10px 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  color: #1d1d1f;
  background: #f5f5f7;
  border-radius: 10px;
  word-break: break-all;
}

.sql-output {
  margin: 0;
  max-height: 220px;
  overflow: auto;
  padding: 10px 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #1d1d1f;
  background: #f5f5f7;
  border-radius: 10px;
  word-break: break-all;
}

.result-table {
  border: 1px solid #e5e5ea;
  border-radius: 10px;
  overflow: hidden;
}

.result-row {
  display: grid;
  grid-template-columns: 1.1fr 1fr 1.2fr 1.2fr 0.7fr;
  gap: 8px;
  align-items: center;
  padding: 8px 12px;
  font-size: 12px;
  color: #1d1d1f;
  border-bottom: 1px solid #e5e5ea;
}

.result-row:last-child {
  border-bottom: none;
}

.result-row--head {
  background: #f5f5f7;
  font-weight: 600;
  color: #86868b;
}

.result-depth {
  color: #5856d6;
  font-weight: 600;
}

.result-pass {
  color: #34c759;
  font-weight: 600;
}

.result-fail {
  color: #ff3b30;
  font-weight: 600;
}

.result-idle {
  color: #86868b;
  font-weight: 600;
}

.result-row--rule {
  grid-template-columns: 1fr 0.6fr 0.6fr 1.8fr;
}

.json-output {
  margin: 0;
  max-height: 300px;
  overflow: auto;
  padding: 10px 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  line-height: 1.6;
  color: #1d1d1f;
  background: #f5f5f7;
  border-radius: 10px;
}

.schema-table {
  border: 1px solid #e5e5ea;
  border-radius: 10px;
  overflow: hidden;
}

.schema-row {
  display: grid;
  grid-template-columns: 96px 132px 1fr;
  gap: 8px;
  align-items: start;
  padding: 8px 12px;
  font-size: 12px;
  color: #1d1d1f;
  border-bottom: 1px solid #e5e5ea;
}

.schema-row:last-child {
  border-bottom: none;
}

.schema-row--head {
  background: #f5f5f7;
  font-weight: 600;
  color: #86868b;
}

.schema-name {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  color: #5856d6;
}

.schema-type {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  color: #86868b;
}

.schema-desc {
  color: #1d1d1f;
}
</style>
