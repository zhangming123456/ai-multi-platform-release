<template>
  <a-drawer
    :visible="props.visible"
    title="说明"
    :width="620"
    :drawer-style="{ maxWidth: '100%' }"
    :footer="false"
    placement="right"
    unmount-on-close
    @cancel="close"
  >
    <a-tabs :active-key="props.activeTab" type="line" @change="onTabChange">
      <a-tab-pane key="schema" title="输出 JSON 结构字段说明">
        <div class="schema-table">
          <div class="schema-row schema-row--head">
            <span>字段</span>
            <span>类型</span>
            <span>作用</span>
          </div>
          <div v-for="field in props.schemaFields" :key="field.name" class="schema-row">
            <span class="schema-name">{{ field.name }}</span>
            <span class="schema-type">{{ field.type }}</span>
            <span class="schema-desc">{{ field.desc }}</span>
          </div>
        </div>
      </a-tab-pane>

      <a-tab-pane key="operator" title="运算符说明">
        <div class="op-table">
          <div class="op-row op-row--head">
            <span>运算符</span>
            <span>标签</span>
            <span>符号</span>
            <span>支持类型</span>
            <span>需取值</span>
            <span>SQL 翻译</span>
          </div>
          <div v-for="doc in props.operatorDocs" :key="doc.value" class="op-row">
            <code class="op-code">{{ doc.value }}</code>
            <span>{{ doc.label }}</span>
            <span class="op-symbol">{{ doc.symbol }}</span>
            <span class="op-types">{{ doc.types }}</span>
            <span>{{ doc.needsValue ? '是' : '否' }}</span>
            <span class="op-sql">{{ doc.sql }}</span>
          </div>
        </div>
        <ul class="doc-list">
          <li>比较类（大于 / 大于等于 / 小于 / 小于等于）仅对数值与日期时间类字段可用</li>
          <li>
            「包含 / 不包含」在文本字段翻译为 LIKE / NOT LIKE（多值分别用 OR / AND
            连接），其他类型翻译为 IN / NOT IN
          </li>
          <li>「为空」翻译为 (col IS NULL OR col = '')</li>
        </ul>
      </a-tab-pane>

      <a-tab-pane key="condition" title="条件编辑说明">
        <ul class="doc-list">
          <li>每行左侧可切换「且 / 或」，「且」优先级高于「或」（与 SQL 一致）</li>
          <li>子条件组最多嵌套 3 层，支持解组或整体删除</li>
          <li>条件行尾的「+ 并且满足」会把该条件就地变成条件组，并自动追加一个「且」空条件</li>
          <li>
            「可用变量组」固定为 4 组，不可新增或删除，且至少保留 1 组；勾选 2
            组及以上时，条件编辑器会为每个已勾选组固定生成一个子组（标题为变量组名）
          </li>
          <li>固定子组不可解组或删除，组内条件只能在所属变量组的变量中选择</li>
          <li>
            启用「可用变量组」后，根级不再显示「添加子条件组」；嵌套子组需在变量组子组内用条件行尾的「+
            并且满足」创建
          </li>
          <li>「清空」下放到各变量组子组，只清空该组内的条件</li>
          <li>删除条件后若所在条件组只剩 1 个条件，该条件组会自动还原为单条件</li>
          <li>
            只勾选 1 个变量组时不套固定子组，条件直接平铺（与分组功能上线前一致），v-model
            输出同样是平铺结构（不含 scope 子组节点）
          </li>
          <li>
            取消勾选某个变量组时不再输出其子组，其内条件平铺保留在根级并标记为无效；重新勾选后按变量归属自动归位回对应子组
          </li>
          <li>
            根级若存在无法归属任何可用变量组的普通条件组，会在归一化时解包平铺（保留其条件、去掉条件组外壳）
          </li>
          <li>
            fieldGroups / scopedGroups 均为非必填；不传或只给 1 个变量组时，v-model
            按「单一可用变量组」的平铺结构输出
          </li>
        </ul>
      </a-tab-pane>

      <a-tab-pane key="rule" title="互斥与关联规则说明">
        <ul class="doc-list">
          <li>规则使用全量可用变量，不受「可用变量组」勾选影响</li>
          <li>规则命中后会禁用或过滤不可选的变量与取值</li>
          <li>
            规则已激活、但它要约束的目标变量不在当前可用变量组内（不可选）时，标记为「激活未生效」；仅部分目标不可选时为「激活部分未生效」
          </li>
          <li>
            上述警示词会出现在规则列表状态区、规则编辑器选择变量时，以及条件编辑器受影响的变量候选项上；命中结果状态显示为「未生效
            / 部分未生效」
          </li>
          <li>条件行尾的橙色感叹号可查看冲突原因</li>
        </ul>
      </a-tab-pane>

      <a-tab-pane key="sql" title="SQL 翻译说明">
        <ul class="doc-list">
          <li>
            「生成 SQL」针对真实表 contents 输出完整同表查询：SELECT 显式列出可用变量列 → FROM
            contents → WHERE 条件 → ORDER BY created_at DESC → LIMIT 100
          </li>
          <li>
            变量与真实列的映射集中在 conditionSqlMapping.ts：仅「内容属性」组映射到 contents
            真实列；账号指标等演示变量没有对应列，生成 SQL 时会被自动跳过并在结果下方标注
          </li>
          <li>字段使用变量编码</li>
          <li>数值按数字绑定、布尔按 1/0 绑定、文本 / 枚举 / 日期 / 日期时间 / 时间按字符串绑定</li>
          <li>
            「包含 / 不包含」在字符串字段翻译为 LIKE / NOT LIKE（多值用 OR / AND
            连接），在枚举、布尔、数值、日期、日期时间、时间字段翻译为 IN / NOT IN
          </li>
          <li>「为空」翻译为 (col IS NULL OR col = '')</li>
        </ul>
      </a-tab-pane>
    </a-tabs>
  </a-drawer>
</template>

<script setup lang="ts">
import type {
  ConditionDocDrawerEmits,
  ConditionDocDrawerProps,
  ConditionDocTabKey,
} from './ConditionBuilder.types'

const props = defineProps<ConditionDocDrawerProps>()
const emit = defineEmits<ConditionDocDrawerEmits>()

function close(): void {
  emit('update:visible', false)
}

function onTabChange(key: string | number): void {
  emit('update:activeTab', key as ConditionDocTabKey)
}
</script>

<style scoped>
.doc-list {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 1.9;
  color: #1d1d1f;
}

.doc-list li + li {
  margin-top: 2px;
}

.schema-table {
  border: 1px solid #e5e5ea;
  border-radius: 10px;
  overflow: hidden;
}

.schema-row {
  display: grid;
  grid-template-columns: 110px 150px 1fr;
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

.op-table {
  border: 1px solid #e5e5ea;
  border-radius: 10px;
  overflow-x: auto;
}

.op-row {
  display: grid;
  grid-template-columns: 92px 56px 44px minmax(90px, 1fr) 44px minmax(110px, 1.3fr);
  gap: 8px;
  align-items: start;
  padding: 8px 12px;
  font-size: 12px;
  color: #1d1d1f;
  border-bottom: 1px solid #e5e5ea;
}

.op-row:last-child {
  border-bottom: none;
}

.op-row--head {
  background: #f5f5f7;
  font-weight: 600;
  color: #86868b;
}

.op-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  color: #5856d6;
}

.op-symbol {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  color: #1d1d1f;
}

.op-types {
  color: #86868b;
}

.op-sql {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  color: #1d1d1f;
  word-break: break-all;
}
</style>
