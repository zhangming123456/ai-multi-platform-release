<template>
  <el-drawer
    :model-value="props.visible"
    title="说明"
    size="620px"
    direction="rtl"
    destroy-on-close
    @close="close"
  >
    <el-tabs :model-value="props.activeTab" @update:model-value="onTabChange">
      <el-tab-pane name="schema" label="输出 JSON 结构字段说明">
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
      </el-tab-pane>

      <el-tab-pane name="operator" label="运算符说明">
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
            日期时间类型变量可在条件行运算符右侧切换比较粒度：日期时间 / 仅日期 /
            仅时间；粒度只改变参与比较的部分，运算符与表达式位置不变
          </li>
          <li>
            「包含 / 不包含」在文本字段翻译为 LIKE / NOT LIKE（多值分别用 OR / AND 连接），在日期 /
            日期时间 / 时间字段为闭区间比较（可只填一端），其他类型翻译为 IN / NOT IN
          </li>
          <li>「为空」翻译为 (col IS NULL OR col = '')</li>
        </ul>
      </el-tab-pane>

      <el-tab-pane name="condition" label="条件编辑说明">
        <ul class="doc-list">
          <li>每行左侧可切换「且 / 或」，「且」优先级高于「或」（与 SQL 一致）</li>
          <li>
            组件参数 logicMode 可选 mixed / uniform（默认 uniform）：mixed 为每行各自切换「且 /
            或」，行内保留连接符与 IF
            标记；uniform（全且或）时不显示行内连接符，仅在层级底部显示一个 且 / 或 切换器；切换到
            uniform 时会按每层第一个子项的逻辑统一该层所有条件的
            logic，点底部切换器也会一次性改写该层（不改变节点结构）
          </li>
          <li>
            子条件组最多嵌套 n 层（默认
            3，可在页面「嵌套上限」调整；根组不计入层数），支持解组或整体删除；达到上限后「添加子条件组」不再显示、「+
            并且满足」置灰并提示
          </li>
          <li>条件行尾的「+ 并且满足」会把该条件就地变成条件组，并自动追加一个「且」空条件</li>
          <li>
            同一条件组内出现 2 条及以上相同变量时（仅数值 / 日期 / 日期时间 /
            时间类型），该组及其子组的变量名会锁定为该变量且不可修改；删除到不足 2 条时锁定自动解除
          </li>
          <li>
            在锁定组上「添加条件」会追加一条空条件行，原有锁定条目自动聚成一个子组，便于新行选择其他变量；在条件行点「+
            并且满足」则在该位置就地包成一个子组（组内自动填入锁定变量），不会打散其他条目
          </li>
          <li>
            锁定组内如混有其他变量（仅数值 / 日期 / 日期时间 /
            时间类型才会触发），会就地重整：锁定变量的条目聚成一个子组，放在它首个出现的位置；其余条目保持原位置，只有「同变量且原位置相邻」的连续段
            ≥2 条才包成子组，单条保持独立行
          </li>
          <li>
            上述聚合子组会递归处理；当子组内的同变量条目不足 2
            条（锁定解除）时，该子组会自动拆平并回退到原位置，无需手动解组
          </li>
          <li>
            锁定组的组头会在「变量锁定」旁实时显示该组（含子组）的完整表达式，过长时省略、悬浮可查看完整内容
          </li>
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
            日期时间变量可在运算符右侧切换「仅日期 / 仅时间」粒度，切换时已填的值会按新粒度截取（如
            2026-10-03 15:30:00 → 2026-10-03 或 15:30:00）
          </li>
          <li>
            日期 / 日期时间 / 时间字段选择「包含 / 不包含」时，值输入变为范围选择器（v-model 用 ~
            分隔起止，如 2026-09-26~2026-10-12，可只填一端）
          </li>
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
      </el-tab-pane>

      <el-tab-pane name="rule" label="互斥与关联规则说明">
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
      </el-tab-pane>

      <el-tab-pane name="sql" label="SQL 翻译说明">
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
            日期时间变量选择「仅日期 / 仅时间」粒度时，列会包一层 DATE(col) / TIME(col) 后再参与比较
          </li>
          <li>
            「包含 / 不包含」在字符串字段翻译为 LIKE / NOT LIKE（多值用 OR / AND 连接）；日期 /
            日期时间 / 时间字段为闭区间，两端填齐翻译为 BETWEEN / NOT
            BETWEEN，只填一端时退化为单边比较；枚举、布尔、数值翻译为 IN / NOT IN
          </li>
          <li>「为空」翻译为 (col IS NULL OR col = '')</li>
        </ul>
      </el-tab-pane>
    </el-tabs>
  </el-drawer>
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
