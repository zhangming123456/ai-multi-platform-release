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
          @click="openDoc('schema')"
        >
          <template #icon><IconBook :size="13" /></template>
          说明
        </a-button>
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
            <div class="flex items-center justify-between mb-3 min-h-6">
              <div class="flex items-center gap-2">
                <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">测试数据（JSON）</h3>
                <a-tag v-if="dataRecords.length" size="small" color="arcoblue" class="!m-0">
                  {{ dataRecords.length }} 条
                </a-tag>
              </div>
              <div class="flex items-center gap-2">
                <a-button
                  v-for="preset in dataPresets"
                  :key="preset.key"
                  size="mini"
                  :type="selectedDataPresetKeys.includes(preset.key) ? 'primary' : 'secondary'"
                  @click="toggleDataPreset(preset)"
                >
                  {{ preset.label }}
                </a-button>
                <a-button
                  v-if="selectedDataPresetKeys.length"
                  size="mini"
                  type="text"
                  class="!text-[#007AFF] !px-1"
                  @click="clearDataPresets"
                >
                  清空
                </a-button>
              </div>
            </div>
            <ConditionJsonInput
              v-model="dataText"
              :field-options="fieldOptions"
              :field-groups="fieldGroups"
              :error="Boolean(dataError)"
              placeholder="输入用于校验的 JSON 对象，或用对象数组提供多条数据"
            />
            <div v-if="dataError" class="text-[12px] text-[#FF3B30] mt-2">{{ dataError }}</div>
            <div v-else class="text-[12px] text-[#86868B] mt-2">
              {{
                dataRecords.length
                  ? `共 ${dataRecords.length} 条测试数据，将逐条校验`
                  : '预设按钮支持多选，可一次校验多条数据'
              }}
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-2">
                <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">条件配置</h3>
                <a-tag size="small" color="arcoblue" class="!m-0">{{ itemCount }} 条</a-tag>
                <a-tag v-if="groupCount" size="small" color="purple" class="!m-0">
                  {{ groupCount }} 个分组
                </a-tag>
                <a-tooltip content="条件编辑说明">
                  <a-button
                    type="text"
                    size="mini"
                    class="!text-[#86868B] hover:!text-[#007AFF] !px-1 !h-auto"
                    @click="openDoc('condition')"
                  >
                    <template #icon><IconInfoCircle :size="14" /></template>
                  </a-button>
                </a-tooltip>
              </div>
              <span class="text-[12px] text-[#86868B]">变量共 {{ fieldOptions.length }} 个</span>
            </div>

            <div class="flex items-center justify-between gap-3 flex-wrap mb-3">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="text-[12px] text-[#86868B]">可用变量组</span>
                <a-button
                  v-for="fieldGroup in fieldGroupPresets"
                  :key="fieldGroup.key"
                  size="mini"
                  :type="selectedFieldGroupKeys.includes(fieldGroup.key) ? 'primary' : 'secondary'"
                  @click="toggleFieldGroup(fieldGroup)"
                >
                  {{ fieldGroup.label }}
                </a-button>
                <a-button
                  v-if="selectedFieldGroupKeys.length < fieldGroupPresets.length"
                  size="mini"
                  type="text"
                  class="!text-[#007AFF] !px-1"
                  @click="selectAllFieldGroups"
                >
                  全选
                </a-button>
              </div>
              <span class="text-[12px] text-[#86868B]">
                {{ selectedFieldGroupKeys.length }} / {{ fieldGroupPresets.length }} 组
              </span>
            </div>

            <ConditionBuilder
              v-model="group"
              :field-options="fieldOptions"
              :field-groups="fieldGroups"
              :scoped-groups="scopedGroups"
              :rule-field-options="allFieldOptions"
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
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <div class="flex items-center gap-2 mb-3">
              <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">互斥与关联规则</h3>
              <a-tag size="small" color="arcoblue" class="!m-0">{{ rules.length }} 条</a-tag>
              <a-tooltip content="互斥与关联规则说明">
                <a-button
                  type="text"
                  size="mini"
                  class="!text-[#86868B] hover:!text-[#007AFF] !px-1 !h-auto"
                  @click="openDoc('rule')"
                >
                  <template #icon><IconInfoCircle :size="14" /></template>
                </a-button>
              </a-tooltip>
              <span class="text-[12px] text-[#86868B] ml-auto">
                变量范围：全量可用变量（{{ allFieldOptions.length }} 个）
              </span>
            </div>

            <ConditionRuleEditor
              v-model:rules="rules"
              :field-options="allFieldOptions"
              :field-groups="allFieldGroups"
              :field-effects="ruleContext.fieldEffects"
              :rule-states="ruleStates"
            />
          </div>
        </div>

        <div class="flex flex-col gap-5 lg:flex-1 lg:min-w-[400px]">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0 mb-3 min-h-6">v-model 输出</h3>
            <pre class="json-output json-output--fixed">{{ modelJson }}</pre>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0 mb-3">表达式与校验</h3>

            <div class="text-[12px] text-[#86868B] mb-1">生成表达式</div>
            <div class="expression">{{ expression || '（暂无有效条件）' }}</div>

            <div class="flex items-center gap-2 mt-4 mb-2">
              <span class="text-[13px] font-semibold text-[#1D1D1F]">校验结果</span>
              <a-tag v-if="dataResults.length > 1" size="small" color="arcoblue" class="!m-0">
                {{ dataResults.length }} 条数据
              </a-tag>
            </div>

            <div v-if="!hasConditions" class="py-4">
              <a-empty description="暂无可校验的条件" />
            </div>
            <div v-else-if="!dataResults.length" class="py-4">
              <a-empty description="暂无可校验的测试数据" />
            </div>
            <div v-else class="data-results">
              <div v-for="result in dataResults" :key="result.key" class="data-result">
                <div class="data-result__head">
                  <span v-if="result.label" class="data-result__label">{{ result.label }}</span>
                  <a-tag
                    :color="result.evaluation.passed ? 'green' : 'red'"
                    size="small"
                    class="!m-0"
                  >
                    {{ result.evaluation.passed ? '命中' : '未命中' }}
                  </a-tag>
                </div>
                <div class="result-table">
                  <div class="result-row result-row--head">
                    <span>变量</span>
                    <span>运算符</span>
                    <span>期望值</span>
                    <span>实际值</span>
                    <span>结果</span>
                  </div>
                  <div
                    v-for="row in result.evaluation.results"
                    :key="row.item.id"
                    class="result-row"
                  >
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
              </div>
            </div>

            <div class="flex items-center gap-2 mt-5 mb-2">
              <span class="text-[13px] font-semibold text-[#1D1D1F]">互斥与关联命中结果</span>
              <a-tag v-if="rules.length" :color="ruleResultTagColor" class="!m-0">
                {{ ruleResultTagText }}
              </a-tag>
              <span class="text-[11px] text-[#86868B]">仅取决于条件配置，与测试数据无关</span>
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
                <a-tag color="gray" class="!m-0">{{ CONDITION_SQL_TABLE }}</a-tag>
                <a-tag :color="sqlStatement.hasConditions ? 'arcoblue' : 'gray'" class="!m-0">
                  {{ sqlStatement.params.length }} 个参数
                </a-tag>
                <a-tooltip content="SQL 翻译说明">
                  <a-button
                    type="text"
                    size="mini"
                    class="!text-[#86868B] hover:!text-[#007AFF] !px-1 !h-auto"
                    @click="openDoc('sql')"
                  >
                    <template #icon><IconInfoCircle :size="14" /></template>
                  </a-button>
                </a-tooltip>
              </div>
              <a-button
                type="text"
                size="mini"
                class="!text-[#007AFF] !px-0 !h-auto"
                :disabled="!sqlStatement.hasConditions"
                @click="copySql"
              >
                <template #icon><IconCopy :size="13" /></template>
                复制 SQL
              </a-button>
            </div>

            <template v-if="sqlStatement.hasConditions">
              <div class="text-[12px] text-[#86868B] mb-1">
                完整 SQL（contents 单表查询 · 参数化，复制后按 ? 顺序绑定参数即可执行）
              </div>
              <pre class="sql-output sql-output--statement">{{ sqlText }}</pre>
              <div v-if="sqlStatement.skipped.length" class="sql-skipped">
                <IconExclamationCircle :size="14" />
                <span>
                  已跳过 {{ sqlStatement.skipped.length }} 个无对应真实列的变量：{{
                    sqlStatement.skipped.join('、')
                  }}
                </span>
              </div>
              <div class="text-[12px] text-[#86868B] mt-3 mb-1">参数数组（按 ? 顺序）</div>
              <pre class="sql-output">{{ sqlParamsText }}</pre>
            </template>
            <div v-else-if="sqlStatement.skipped.length" class="py-4">
              <a-empty
                :description="`当前条件全部使用了无真实列的变量：${sqlStatement.skipped.join('、')}，无法生成 SQL`"
              />
            </div>
            <div v-else class="py-4">
              <a-empty description="暂无可转换的条件" />
            </div>
          </div>

          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
            <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0 mb-3">规则配置输出</h3>
            <pre class="json-output">{{ rulesJson }}</pre>
          </div>
        </div>
      </div>
    </div>

    <ConditionDocDrawer
      v-model:visible="docVisible"
      v-model:active-tab="docTab"
      :schema-fields="schemaFields"
      :operator-docs="operatorDocs"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconBook,
  IconCopy,
  IconExclamationCircle,
  IconInfoCircle,
  IconRefresh,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import ConditionBuilder from '@/components/ConditionBuilder/ConditionBuilder.vue'
import ConditionDocDrawer from '@/components/ConditionBuilder/ConditionDocDrawer.vue'
import ConditionJsonInput from '@/components/ConditionBuilder/ConditionJsonInput.vue'
import ConditionRuleEditor from '@/components/ConditionBuilder/ConditionRuleEditor.vue'
import type {
  ConditionFieldGroup,
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
  createConditionItem,
  createScopedConditionGroup,
  evaluateConditionGroup,
  flattenFieldGroups,
  isConditionGroup,
  pruneInactiveScopedGroups,
} from '@/components/ConditionBuilder/conditionOperator'
import {
  CONDITION_RULE_STATE_LABELS,
  CONDITION_RULE_TYPE_LABELS,
  conditionRuleState,
  conditionRuleSummary,
  resolveConditionRules,
} from '@/components/ConditionBuilder/conditionRules'
import { buildConditionStatement } from '@/components/ConditionBuilder/conditionSql'
import { CONDITION_SQL_TABLE } from '@/components/ConditionBuilder/conditionSqlMapping'
import type {
  ConditionDataPreset,
  ConditionDataResult,
  ConditionDocTabKey,
  ConditionOperatorDoc,
  ConditionPreset,
  ConditionRuleState,
  ConditionSchemaField,
} from './ConditionTest.types'

const fieldGroupPresets: ConditionFieldGroup[] = [
  {
    key: 'contents',
    label: '内容属性',
    description: 'contents 表真实列，可直接生成 SQL',
    fields: [
      { value: 'title', label: '标题', type: 'string', description: 'contents.title · 内容标题' },
      { value: 'body', label: '正文', type: 'string', description: 'contents.body · 内容正文' },
      {
        value: 'platform',
        label: '发布平台',
        type: 'select',
        description: 'contents.platform · 发布渠道',
        options: [
          { label: '小红书', value: 'xiaohongshu' },
          { label: '抖音', value: 'douyin' },
          { label: '微信视频号', value: 'wechat_video' },
          { label: '微信公众号', value: 'wechat_mp' },
        ],
      },
      {
        value: 'status',
        label: '内容状态',
        type: 'select',
        description: 'contents.status · 内容生命周期状态',
        options: [
          { label: '草稿', value: 'draft' },
          { label: '待发布', value: 'ready' },
          { label: '待审核', value: 'pending_review' },
          { label: '已驳回', value: 'rejected' },
          { label: '已发布', value: 'published' },
        ],
      },
      {
        value: 'ai_generated',
        label: 'AI 生成',
        type: 'boolean',
        description: 'contents.ai_generated · 是否由 AI 生成',
      },
      {
        value: 'created_at',
        label: '创建时间',
        type: 'datetime',
        description: 'contents.created_at · 格式 YYYY-MM-DD HH:mm:ss',
      },
      {
        value: 'updated_at',
        label: '更新时间',
        type: 'datetime',
        description: 'contents.updated_at · 格式 YYYY-MM-DD HH:mm:ss',
      },
    ],
  },
  {
    key: 'metrics',
    label: '数据指标（演示）',
    description: '真实表暂无对应列，SQL 生成时会跳过并在下方标注',
    fields: [
      {
        value: 'fans_count',
        label: '粉丝数',
        type: 'number',
        description: '账号当前粉丝数量',
        queryable: false,
      },
      {
        value: 'like_count',
        label: '点赞数',
        type: 'number',
        description: '内容累计点赞数',
        queryable: false,
      },
      {
        value: 'comment_count',
        label: '评论数',
        type: 'number',
        description: '内容累计评论数',
        queryable: false,
      },
      {
        value: 'share_count',
        label: '转发数',
        type: 'number',
        description: '内容累计转发次数',
        queryable: false,
      },
      {
        value: 'collect_count',
        label: '收藏数',
        type: 'number',
        description: '内容累计收藏次数',
        queryable: false,
      },
      {
        value: 'view_count',
        label: '阅读量',
        type: 'number',
        description: '内容累计阅读 / 播放量',
        queryable: false,
      },
    ],
  },
  {
    key: 'account',
    label: '账号属性（演示）',
    description: '真实表暂无对应列，SQL 生成时会跳过并在下方标注',
    fields: [
      {
        value: 'is_verified',
        label: '是否认证',
        type: 'boolean',
        description: '账号是否已完成认证',
        queryable: false,
      },
      {
        value: 'account_level',
        label: '账号等级',
        type: 'select',
        description: '账号的运营等级',
        options: [
          { label: 'L1', value: 'L1' },
          { label: 'L2', value: 'L2' },
          { label: 'L3', value: 'L3' },
          { label: 'L4', value: 'L4' },
          { label: 'L5', value: 'L5' },
        ],
        queryable: false,
      },
      {
        value: 'follower_tier',
        label: '粉丝层级',
        type: 'select',
        description: '账号粉丝量级分层',
        options: [
          { label: '素人', value: 'micro' },
          { label: '腰部', value: 'mid' },
          { label: '头部', value: 'top' },
        ],
        queryable: false,
      },
      {
        value: 'region',
        label: '账号地区',
        type: 'string',
        description: '账号所属地区',
        queryable: false,
      },
    ],
  },
  {
    key: 'schedule',
    label: '发布时间（演示）',
    description: '真实表暂无对应列，SQL 生成时会跳过并在下方标注',
    fields: [
      {
        value: 'published_at',
        label: '发布时间',
        type: 'date',
        description: '内容实际发布时间，格式 YYYY-MM-DD',
        queryable: false,
      },
      {
        value: 'publish_slot',
        label: '发布时段',
        type: 'time',
        description: '每日计划发布的时刻，格式 HH:mm:ss',
        queryable: false,
      },
      {
        value: 'scheduled_at',
        label: '计划发布时间',
        type: 'datetime',
        description: '定时发布的计划时刻，格式 YYYY-MM-DD HH:mm:ss',
        queryable: false,
      },
    ],
  },
]

const selectedFieldGroupKeys = ref<string[]>(fieldGroupPresets.map((group) => group.key))

const fieldGroups = computed(() =>
  fieldGroupPresets.filter((group) => selectedFieldGroupKeys.value.includes(group.key)),
)

const fieldOptions = computed<ConditionFieldOption[]>(() => flattenFieldGroups(fieldGroups.value))

const scopedGroups = computed<ConditionFieldGroup[]>(() =>
  fieldGroupPresets.map((group) => ({
    ...group,
    active: selectedFieldGroupKeys.value.includes(group.key),
  })),
)

const allFieldGroups = computed<ConditionFieldGroup[]>(() => fieldGroupPresets)

const allFieldOptions = computed<ConditionFieldOption[]>(() =>
  flattenFieldGroups(fieldGroupPresets),
)

const sampleData: Record<string, unknown> = {
  title: '周末不跑远！城市露营攻略🏕️超全装备清单+玩法推荐✨',
  body: '城市露营装备清单与玩法推荐，适合周末短途出行。',
  platform: 'xiaohongshu',
  status: 'draft',
  ai_generated: false,
  created_at: '2026-08-19 10:54:17',
  updated_at: '2026-08-19 10:54:17',
  fans_count: 12800,
  like_count: 860,
  comment_count: 42,
  share_count: 96,
  collect_count: 158,
  view_count: 42300,
  is_verified: false,
  account_level: 'L2',
  follower_tier: 'mid',
  region: '广东 深圳',
  published_at: '2026-08-19',
  publish_slot: '09:30:00',
  scheduled_at: '2026-08-19 09:30:00',
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
    key: 'real-full',
    label: '真实表完整案例：平台 IN + 标题包含 + 状态 IN + AI 生成 + 时间范围',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeItem('platform', 'contains', 'xiaohongshu, douyin'),
        makeItem('title', 'contains', '露营, 攻略'),
        makeItem('status', 'contains', 'draft, ready'),
        makeItem('ai_generated', 'eq', 'false'),
        makeItem('created_at', 'gte', '2026-08-01 00:00:00'),
        makeItem('updated_at', 'lt', '2026-10-01 00:00:00'),
      ]),
    ]),
  },
  {
    key: 'nested-or',
    label: '标题 包含 (露营) 或 (平台 = 小红书 且 状态 = draft)',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeItem('title', 'contains', '露营'),
        makeGroup(
          [makeItem('platform', 'eq', 'xiaohongshu'), makeItem('status', 'eq', 'draft')],
          'or',
        ),
      ]),
    ]),
  },
  {
    key: 'body-like',
    label: '正文 包含 露营, 装备 且 正文 不包含 广告, 推广（LIKE / NOT LIKE）',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeItem('body', 'contains', '露营, 装备'),
        makeItem('body', 'not_contains', '广告, 推广'),
      ]),
    ]),
  },
  {
    key: 'status-in',
    label: '内容状态 IN (draft, ready) 且 AI 生成 = 否（枚举 IN）',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeItem('status', 'contains', 'draft, ready'),
        makeItem('ai_generated', 'eq', 'false'),
      ]),
    ]),
  },
  {
    key: 'status-exclude',
    label: '内容状态 不包含 已驳回, 已发布 且 标题 不包含 广告（NOT IN / NOT LIKE）',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeItem('status', 'not_contains', 'rejected, published'),
        makeItem('title', 'not_contains', '广告'),
      ]),
    ]),
  },
  {
    key: 'datetime-range',
    label: '创建时间 ≥ 2026-08-01 00:00:00 且 更新时间 ≤ 2026-09-30 23:59:59',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeItem('created_at', 'gte', '2026-08-01 00:00:00'),
        makeItem('updated_at', 'lte', '2026-09-30 23:59:59'),
      ]),
    ]),
  },
  {
    key: 'datetime-or',
    label: '创建时间 < 2026-08-20 00:00:00 或 更新时间 ≥ 2026-09-01 00:00:00',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeItem('created_at', 'lt', '2026-08-20 00:00:00'),
        makeItem('updated_at', 'gte', '2026-09-01 00:00:00', 'or'),
      ]),
    ]),
  },
  {
    key: 'deep',
    label: '三层嵌套：(状态 = draft 或 AI 生成 = 是) 且 (平台 包含 抖音, 小红书 或 标题 包含 中秋)',
    group: makeGroup([
      createScopedConditionGroup('contents', [
        makeGroup([
          makeItem('status', 'eq', 'draft'),
          makeItem('ai_generated', 'eq', 'true', 'or'),
        ]),
        makeGroup([
          makeItem('platform', 'contains', 'douyin, xiaohongshu'),
          makeItem('title', 'contains', '中秋', 'or'),
        ]),
      ]),
    ]),
  },
  {
    key: 'is-null',
    label: '正文 为空',
    group: makeGroup([createScopedConditionGroup('contents', [makeItem('body', 'is_null', '')])]),
  },
  {
    key: 'demo-skip',
    label: '演示变量（无真实列，SQL 会跳过）：内容属性 平台 = 小红书｜账号指标 粉丝数 ≥ 10000',
    group: makeGroup([
      createScopedConditionGroup('contents', [makeItem('platform', 'eq', 'xiaohongshu')]),
      createScopedConditionGroup('metrics', [makeItem('fans_count', 'gte', '10000')]),
    ]),
  },
]

const dataPresets: ConditionDataPreset[] = [
  { key: 'camping', label: '真实：露营攻略(小红书)', data: sampleData },
  {
    key: 'midautumn',
    label: '真实：中秋倒计时(抖音)',
    data: {
      title: '🥮中秋倒计时！今年别再错过了！月圆人团圆',
      body: '中秋节日内容策划与选题思路。',
      platform: 'douyin',
      status: 'draft',
      ai_generated: false,
      created_at: '2026-08-20 02:31:20',
      updated_at: '2026-08-20 02:31:20',
      fans_count: 56000,
      like_count: 2400,
      comment_count: 0,
      share_count: 320,
      collect_count: 410,
      view_count: 128000,
      is_verified: true,
      account_level: 'L3',
      follower_tier: 'top',
      region: '上海',
      published_at: '2026-08-20',
      publish_slot: '07:45:00',
      scheduled_at: '2026-08-20 07:45:00',
    },
  },
  {
    key: 'missing-body',
    label: '正文缺失',
    data: {
      title: '记录一场绿光里的“荒野爱人”💚🎤',
      platform: 'xiaohongshu',
      status: 'ready',
      ai_generated: true,
      created_at: '2026-09-07 03:32:42',
      updated_at: '2026-09-07 03:32:42',
      fans_count: 99000,
      like_count: 5000,
      comment_count: 128,
      share_count: 640,
      collect_count: 980,
      view_count: 256000,
      is_verified: true,
      account_level: 'L5',
      follower_tier: 'top',
      region: '北京',
      published_at: '2026-09-07',
      publish_slot: '12:00:00',
      scheduled_at: '2026-09-07 12:00:00',
    },
  },
]

const rules = ref<ConditionRule[]>([
  {
    id: 'rule-platform-exclusive',
    type: 'mutual_exclusive',
    name: '平台取值互斥',
    description: '「发布平台」不可同时包含 抖音 与 小红书',
    when: { field: 'platform', operator: 'contains', value: 'douyin' },
    targets: [{ field: 'platform', operator: 'contains', value: 'xiaohongshu' }],
    message: '「发布平台」不可同时包含 抖音 与 小红书',
  },
  {
    id: 'rule-status-prerequisite',
    type: 'prerequisite',
    name: '内容状态先决',
    description: '配置「内容状态」前必须先添加「发布平台」条件',
    when: { field: 'platform' },
    targets: [{ field: 'status' }],
    message: '配置「内容状态」前需先添加「发布平台」条件',
  },
  {
    id: 'rule-douyin-status-linkage',
    type: 'linkage',
    name: '抖音内容状态联动',
    description: '平台包含 抖音 时，「内容状态」仅可选 草稿 / 待发布',
    when: { field: 'platform', operator: 'contains', value: 'douyin' },
    field: 'status',
    operator: 'contains',
    allowedValues: ['draft', 'ready'],
    message: '平台含抖音时「内容状态」仅可选 草稿 / 待发布',
  },
  {
    id: 'rule-fans-count-nonnegative',
    type: 'linkage',
    name: '粉丝数不得为负',
    description: '「粉丝数」的取值不能小于 0（演示变量，不参与 SQL）',
    when: { field: 'fans_count' },
    field: 'fans_count',
    operator: 'gte',
    allowedValues: ['0'],
    message: '「粉丝数」不能小于 0',
  },
  {
    id: 'rule-verified-exclusive',
    type: 'mutual_exclusive',
    name: '抖音不可同为认证账号',
    description: '平台含抖音时不可同时为认证账号（目标是账号属性组，该组未勾选时=激活未生效）',
    when: { field: 'platform', operator: 'contains', value: 'douyin' },
    targets: [{ field: 'is_verified', operator: 'eq', value: 'true' }],
    message: '平台含抖音时不可同时为认证账号',
  },
  {
    id: 'rule-douyin-extra-exclusive',
    type: 'mutual_exclusive',
    name: '抖音附加互斥（跨组）',
    description: '平台含抖音时，不可同时使用「内容状态=待审核」与「账号等级≥L3」',
    when: { field: 'platform', operator: 'contains', value: 'douyin' },
    targets: [
      { field: 'status', operator: 'contains', value: 'pending_review' },
      { field: 'account_level', operator: 'gte', value: 'L3' },
    ],
    message: '平台含抖音时不可同时使用「待审核」与「L3 及以上」',
  },
  {
    id: 'rule-published-slot-linkage',
    type: 'linkage',
    name: '已发布内容发布时段联动',
    description: '内容状态含已发布时，「发布时段」仅可选 09:00:00 / 20:00:00（目标是发布时间组）',
    when: { field: 'status', operator: 'contains', value: 'published' },
    field: 'publish_slot',
    operator: 'contains',
    allowedValues: ['09:00:00', '20:00:00'],
    message: '已发布内容「发布时段」仅可选 09:00:00 / 20:00:00',
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

const operatorDocs: ConditionOperatorDoc[] = [
  { value: 'eq', label: '等于', symbol: '=', types: '全部类型', needsValue: true, sql: 'col = ?' },
  {
    value: 'ne',
    label: '不等于',
    symbol: '≠',
    types: '全部类型',
    needsValue: true,
    sql: 'col != ?',
  },
  {
    value: 'gt',
    label: '大于',
    symbol: '>',
    types: '数值 / 日期 / 日期时间 / 时间',
    needsValue: true,
    sql: 'col > ?',
  },
  {
    value: 'gte',
    label: '大于等于',
    symbol: '≥',
    types: '数值 / 日期 / 日期时间 / 时间',
    needsValue: true,
    sql: 'col >= ?',
  },
  {
    value: 'lt',
    label: '小于',
    symbol: '<',
    types: '数值 / 日期 / 日期时间 / 时间',
    needsValue: true,
    sql: 'col < ?',
  },
  {
    value: 'lte',
    label: '小于等于',
    symbol: '≤',
    types: '数值 / 日期 / 日期时间 / 时间',
    needsValue: true,
    sql: 'col <= ?',
  },
  {
    value: 'contains',
    label: '包含',
    symbol: 'IN',
    types: '全部类型',
    needsValue: true,
    sql: '文本 LIKE ?（OR）；其他 IN (?)',
  },
  {
    value: 'not_contains',
    label: '不包含',
    symbol: 'NOT IN',
    types: '全部类型',
    needsValue: true,
    sql: '文本 NOT LIKE ?（AND）；其他 NOT IN (?)',
  },
  {
    value: 'is_null',
    label: '为空',
    symbol: '为空',
    types: '全部类型',
    needsValue: false,
    sql: "col IS NULL OR col = ''",
  },
]

function createInitialGroup(): ConditionGroup {
  return createConditionGroup(
    fieldGroupPresets.map((entry) =>
      createScopedConditionGroup(entry.key, [createConditionItem()]),
    ),
  )
}

const group = ref<ConditionGroup>(createInitialGroup())
const dataText = ref(JSON.stringify(sampleData, null, 2))

const docVisible = ref(false)
const docTab = ref<ConditionDocTabKey>('schema')

const visibleGroup = computed(() =>
  pruneInactiveScopedGroups(group.value, (scope) => selectedFieldGroupKeys.value.includes(scope)),
)

const itemCount = computed(() => collectConditionItems(visibleGroup.value).length)

const groupCount = computed(() => collectConditionGroups(visibleGroup.value).length)

const dataState = computed<{ value?: unknown; error: string }>(() => {
  if (!dataText.value.trim()) return { error: '' }
  try {
    const value: unknown = JSON.parse(dataText.value)
    if (Array.isArray(value)) {
      if (!value.every((entry) => isPlainRecord(entry))) {
        return { error: '测试数据数组的每一项都必须是 JSON 对象' }
      }
      return { value, error: '' }
    }
    if (!isPlainRecord(value)) return { error: '测试数据必须是 JSON 对象或对象数组' }
    return { value, error: '' }
  } catch (error) {
    return { error: `JSON 解析失败：${error instanceof Error ? error.message : '未知错误'}` }
  }
})

const dataError = computed(() => dataState.value.error)

const dataRecords = computed<Record<string, unknown>[]>(() => {
  const { value, error } = dataState.value
  if (error || value === undefined) return []
  if (Array.isArray(value)) return value as Record<string, unknown>[]
  return [value as Record<string, unknown>]
})

const selectedDataPresetKeys = computed(() =>
  dataPresets
    .filter((preset) => {
      const signature = recordSignature(preset.data)
      return dataRecords.value.some((record) => recordSignature(record) === signature)
    })
    .map((preset) => preset.key),
)

const expression = computed(() => buildConditionExpression(visibleGroup.value, fieldOptions.value))

const hasConditions = computed(
  () => evaluateConditionGroup(visibleGroup.value, {}, fieldOptions.value).hasConditions,
)

const dataResults = computed<ConditionDataResult[]>(() =>
  dataRecords.value.map((record, index) => {
    const signature = recordSignature(record)
    const preset = dataPresets.find((entry) => recordSignature(entry.data) === signature)
    return {
      key: `${index}-${signature}`,
      label: preset ? preset.label : dataRecords.value.length > 1 ? `数据 ${index + 1}` : '',
      evaluation: evaluateConditionGroup(visibleGroup.value, record, fieldOptions.value),
    }
  }),
)

const sqlStatement = computed(() => buildConditionStatement(visibleGroup.value, fieldOptions.value))

const sqlText = computed(() => sqlStatement.value.sql)

const sqlParamsText = computed(() => JSON.stringify(sqlStatement.value.params))

const modelJson = computed(() => JSON.stringify(group.value, null, 2))

const ruleContext = computed(() =>
  resolveConditionRules(
    visibleGroup.value,
    rules.value,
    allFieldOptions.value,
    fieldOptions.value.map((field) => field.value),
  ),
)

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
  partial: 'result-warn',
  ineffective: 'result-warn',
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
      detail: violation?.message ?? conditionRuleSummary(rule, allFieldOptions.value),
    }
  }),
)

const ineffectiveCount = computed(
  () =>
    rules.value.filter((rule) => {
      const state = ruleStates.value[rule.id]
      return state === 'partial' || state === 'ineffective'
    }).length,
)

const ruleResultTagText = computed(() => {
  if (violationCount.value) return '存在冲突'
  if (ineffectiveCount.value) return '存在未生效'
  if (activeRuleCount.value) return '命中'
  return '未命中'
})

const ruleResultTagColor = computed(() => {
  if (violationCount.value) return 'red'
  if (ineffectiveCount.value) return 'orange'
  if (activeRuleCount.value) return 'green'
  return 'gray'
})

function presetScopes(node: ConditionNode, scopes: string[] = []): string[] {
  if (!isConditionGroup(node)) return scopes
  if (node.scope && !scopes.includes(node.scope)) scopes.push(node.scope)
  node.children.forEach((child) => presetScopes(child, scopes))
  return scopes
}

function applyPreset(preset: ConditionPreset): void {
  group.value = cloneConditionGroup(preset.group)
  const scopes = presetScopes(preset.group)
  selectedFieldGroupKeys.value = Array.from(new Set([...selectedFieldGroupKeys.value, ...scopes]))
}

async function copySql(): Promise<void> {
  if (!sqlStatement.value.hasConditions) return
  const text = `${sqlText.value}\n-- params: ${sqlParamsText.value}`
  try {
    await navigator.clipboard.writeText(text)
    Message.success('已复制到剪贴板')
  } catch {
    Message.error('复制失败，请手动选择文本复制')
  }
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function recordSignature(record: Record<string, unknown>): string {
  return JSON.stringify(
    Object.keys(record)
      .sort()
      .map((key) => [key, record[key]]),
  )
}

function serializeRecords(records: Record<string, unknown>[]): string {
  return records.length === 1
    ? JSON.stringify(records[0], null, 2)
    : JSON.stringify(records, null, 2)
}

function toggleDataPreset(preset: ConditionDataPreset): void {
  const signature = recordSignature(preset.data)
  const records = dataRecords.value
  const exists = records.some((record) => recordSignature(record) === signature)
  const next = exists
    ? records.filter((record) => recordSignature(record) !== signature)
    : [...records, preset.data]
  dataText.value = serializeRecords(next)
}

function clearDataPresets(): void {
  dataText.value = JSON.stringify([], null, 2)
}

function toggleFieldGroup(group: ConditionFieldGroup): void {
  const keys = selectedFieldGroupKeys.value
  if (!keys.includes(group.key)) {
    selectedFieldGroupKeys.value = [...keys, group.key]
    return
  }
  if (keys.length <= 1) {
    Message.warning('至少需要保留一个可用变量组')
    return
  }
  selectedFieldGroupKeys.value = keys.filter((key) => key !== group.key)
}

function selectAllFieldGroups(): void {
  selectedFieldGroupKeys.value = fieldGroupPresets.map((group) => group.key)
}

function resetPreset(): void {
  group.value = createInitialGroup()
  dataText.value = JSON.stringify(sampleData, null, 2)
  selectAllFieldGroups()
}

function openDoc(tab: ConditionDocTabKey): void {
  docTab.value = tab
  docVisible.value = true
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

.sql-output--statement {
  max-height: 340px;
  white-space: pre;
}

.sql-skipped {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-top: 8px;
  padding: 8px 10px;
  font-size: 12px;
  line-height: 1.5;
  color: #b7791f;
  background: rgba(255, 193, 7, 0.12);
  border: 1px solid rgba(255, 193, 7, 0.28);
  border-radius: 10px;
}

.sql-skipped :deep(svg) {
  flex: 0 0 auto;
  margin-top: 2px;
}

.data-results {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.data-result__head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.data-result__label {
  padding: 1px 8px;
  font-size: 12px;
  font-weight: 600;
  color: #007aff;
  background: rgba(0, 122, 255, 0.08);
  border-radius: 6px;
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

.result-warn {
  color: #d46b08;
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

.json-output--fixed {
  height: 300px;
}
</style>
