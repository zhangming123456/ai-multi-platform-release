<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconPlayArrow,
  IconDelete,
  IconStorage,
  IconCode,
  IconThunderbolt,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'
import { formatDateTimeSec as formatHistoryTime } from '@/utils/time'

interface SqlResult {
  success: boolean
  message: string
  columns: string[]
  rows: (string | number | null)[][]
  row_count: number
  total_count: number
  page_size: number
  is_query: boolean
  need_review?: boolean
  change_type?: string | null
}

interface TableColumnInfo {
  cid: number
  name: string
  type: string
  notnull: number
  dflt_value: string | null
  pk: number
}

interface SqlHistoryItem {
  id: string
  username: string
  sql_text: string
  is_success: boolean
  message: string
  created_at: string
}

interface SqlHistoryResponse {
  items: SqlHistoryItem[]
  total: number
  page: number
  page_size: number
}

interface Suggestion {
  text: string
  type: 'keyword' | 'table' | 'function' | 'operator'
  description: string
  category: string
}

const sqlInput = ref('')
const executing = ref(false)
const lastSql = ref('')
const result = ref<SqlResult | null>(null)
const history = ref<SqlHistoryItem[]>([])
const historyTotal = ref(0)
const historyPage = ref(1)
const historyPageSize = ref(20)
const showSuggestions = ref(false)
const suggestions = ref<Suggestion[]>([])
const selectedSuggestionIdx = ref(0)
const caretPosition = ref(0)
const suggestionStyle = ref({ top: '0px', left: '0px' })
const textareaEl = ref<InstanceType<typeof HTMLTextAreaElement> | null>(null)
const tableNames = ref<string[]>([])
const tableSchemas = ref<Record<string, TableColumnInfo[]>>({})
const expandedTables = ref<Set<string>>(new Set())
const loadingSchema = ref<Set<string>>(new Set())
const currentPage = ref(1)
const pageSize = ref(30)

const SQL_KEYWORDS: Suggestion[] = [
  { text: 'SELECT', type: 'keyword', description: '查询数据', category: 'DQL' },
  { text: 'FROM', type: 'keyword', description: '指定表名', category: 'DQL' },
  { text: 'WHERE', type: 'keyword', description: '条件过滤', category: 'DQL' },
  { text: 'AND', type: 'keyword', description: '逻辑与', category: '运算符' },
  { text: 'OR', type: 'keyword', description: '逻辑或', category: '运算符' },
  { text: 'ORDER BY', type: 'keyword', description: '排序', category: 'DQL' },
  { text: 'GROUP BY', type: 'keyword', description: '分组', category: 'DQL' },
  { text: 'HAVING', type: 'keyword', description: '分组过滤', category: 'DQL' },
  { text: 'LIMIT', type: 'keyword', description: '限制行数', category: 'DQL' },
  { text: 'OFFSET', type: 'keyword', description: '偏移量', category: 'DQL' },
  { text: 'JOIN', type: 'keyword', description: '连接表', category: 'DQL' },
  { text: 'LEFT JOIN', type: 'keyword', description: '左连接', category: 'DQL' },
  { text: 'INNER JOIN', type: 'keyword', description: '内连接', category: 'DQL' },
  { text: 'ON', type: 'keyword', description: '连接条件', category: 'DQL' },
  { text: 'AS', type: 'keyword', description: '别名', category: 'DQL' },
  { text: 'DISTINCT', type: 'keyword', description: '去重', category: 'DQL' },
  { text: 'INSERT INTO', type: 'keyword', description: '插入数据', category: 'DML' },
  { text: 'VALUES', type: 'keyword', description: '插入值', category: 'DML' },
  { text: 'UPDATE', type: 'keyword', description: '更新数据', category: 'DML' },
  { text: 'SET', type: 'keyword', description: '设置值', category: 'DML' },
  { text: 'DELETE FROM', type: 'keyword', description: '删除数据', category: 'DML' },
  { text: 'CREATE TABLE', type: 'keyword', description: '创建表', category: 'DDL' },
  { text: 'NOT NULL', type: 'keyword', description: '非空约束', category: 'DDL' },
  { text: 'PRIMARY KEY', type: 'keyword', description: '主键', category: 'DDL' },
  { text: 'DEFAULT', type: 'keyword', description: '默认值', category: 'DDL' },
  { text: 'INTEGER', type: 'keyword', description: '整数类型', category: '类型' },
  { text: 'TEXT', type: 'keyword', description: '文本类型', category: '类型' },
  { text: 'VARCHAR', type: 'keyword', description: '变长字符', category: '类型' },
  { text: 'BOOLEAN', type: 'keyword', description: '布尔类型', category: '类型' },
  { text: 'DATETIME', type: 'keyword', description: '日期时间', category: '类型' },
  { text: 'FLOAT', type: 'keyword', description: '浮点数', category: '类型' },
  { text: 'LIKE', type: 'operator', description: '模糊匹配', category: '运算符' },
  { text: 'IN', type: 'operator', description: '包含匹配', category: '运算符' },
  { text: 'BETWEEN', type: 'operator', description: '范围匹配', category: '运算符' },
  { text: 'IS NULL', type: 'operator', description: '空值判断', category: '运算符' },
  { text: 'IS NOT NULL', type: 'operator', description: '非空判断', category: '运算符' },
  { text: 'COUNT(', type: 'function', description: '计数', category: '聚合函数' },
  { text: 'SUM(', type: 'function', description: '求和', category: '聚合函数' },
  { text: 'AVG(', type: 'function', description: '平均值', category: '聚合函数' },
  { text: 'MAX(', type: 'function', description: '最大值', category: '聚合函数' },
  { text: 'MIN(', type: 'function', description: '最小值', category: '聚合函数' },
  { text: 'COALESCE(', type: 'function', description: '空值替换', category: '函数' },
  { text: 'PRAGMA', type: 'keyword', description: 'SQLite 配置', category: '系统' },
  { text: 'EXPLAIN', type: 'keyword', description: '查询计划', category: '系统' },
]

const pageSizeOptions = [10, 20, 30, 50, 100]

const tablePresets = [
  { label: '用户列表', icon: '👤', sql: 'SELECT id, username, nickname, role, email FROM users;' },
  { label: '平台账号', icon: '🔗', sql: 'SELECT id, nickname, platform, status FROM accounts;' },
  { label: '内容列表', icon: '📝', sql: 'SELECT id, title, platform, status FROM contents;' },
  { label: '发布任务', icon: '🚀', sql: 'SELECT id, title, platform, status FROM publish_tasks;' },
  {
    label: '所有表',
    icon: '📊',
    sql: "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;",
  },
  { label: '角色统计', icon: '📈', sql: 'SELECT role, COUNT(*) as cnt FROM users GROUP BY role;' },
]

const lastWord = computed(() => {
  const beforeCaret = sqlInput.value.substring(0, caretPosition.value)
  const words = beforeCaret.split(/[\s,;()]+/)
  return (words[words.length - 1] || '').toUpperCase()
})

const textBeforeCaret = computed(() => {
  return sqlInput.value.substring(0, caretPosition.value).toUpperCase()
})

const paginationConfig = computed(() => {
  if (!result.value || !result.value.is_query || result.value.total_count === 0) {
    return false
  }
  return {
    current: currentPage.value,
    pageSize: pageSize.value,
    total: result.value.total_count,
    showTotal: true,
    showPageSize: true,
    pageSizeOptions: pageSizeOptions,
    onChange: (page: number) => {
      currentPage.value = page
      executeSqlPage()
    },
    onPageSizeChange: (size: number) => {
      pageSize.value = size
      currentPage.value = 1
      executeSqlPage()
    },
  }
})

const historyPaginationConfig = computed(() => {
  if (historyTotal.value === 0) return false
  return {
    current: historyPage.value,
    pageSize: historyPageSize.value,
    total: historyTotal.value,
    showTotal: true,
    showPageSize: false,
    simple: true,
    size: 'mini' as const,
    onChange: (page: number) => {
      historyPage.value = page
      fetchHistory()
    },
  }
})

const activeTableName = ref('')

const tableColumnPresets = computed<Record<string, string[]>>(() => {
  const map: Record<string, string[]> = {}
  for (const [name, cols] of Object.entries(tableSchemas.value)) {
    map[name] = cols.map((c) => c.name)
  }
  return map
})

function needsTableSuggestion(): boolean {
  const before = textBeforeCaret.value.trim()
  return (
    before.endsWith('FROM') ||
    before.endsWith('JOIN') ||
    before.endsWith('INTO') ||
    before.endsWith('UPDATE') ||
    before.endsWith('TABLE')
  )
}

const filteredSuggestions = computed(() => {
  const word = lastWord.value
  if (!word) return []

  let items: Suggestion[]

  if (needsTableSuggestion()) {
    items = tableNames.value.map((name) => ({
      text: name,
      type: 'table' as const,
      description: `表: ${name}`,
      category: '数据表',
    }))
  } else {
    items = [...SQL_KEYWORDS]
  }

  const filtered = items.filter((s) => s.text.startsWith(word) && s.text !== word)

  return filtered.slice(0, 12)
})

async function fetchTableNames() {
  try {
    const res = await api.post<SqlResult>('/db/execute', {
      sql: "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;",
    })
    if (res.data.success && res.data.rows) {
      tableNames.value = res.data.rows.map((row) => String(row[0]))
    }
  } catch {
    tableNames.value = ['users', 'accounts', 'contents', 'publish_tasks', 'model_configs']
  }
}

function toggleTableSchema(tableName: string) {
  if (activeTableName.value === tableName) {
    activeTableName.value = ''
    return
  }

  activeTableName.value = tableName

  if (tableSchemas.value[tableName]) return

  loadTableSchema(tableName)
}

async function loadTableSchema(tableName: string) {
  loadingSchema.value.add(tableName)
  try {
    const res = await api.post<SqlResult>('/db/execute', {
      sql: `PRAGMA table_info(${tableName});`,
    })
    if (res.data.success && res.data.rows) {
      const cols: TableColumnInfo[] = res.data.rows.map((row) => ({
        cid: Number(row[0]),
        name: String(row[1]),
        type: String(row[2]),
        notnull: Number(row[3]),
        dflt_value: row[4] as string | null,
        pk: Number(row[5]),
      }))
      tableSchemas.value[tableName] = cols
    }
  } catch {
    Message.error('获取表结构失败')
  } finally {
    loadingSchema.value.delete(tableName)
  }
}

function selectTableSql(tableName: string) {
  sqlInput.value = `SELECT * FROM ${tableName} LIMIT 50;`
  nextTick(() => {
    textareaEl.value?.focus()
  })
}

function getCaretCoordinates(el: HTMLTextAreaElement, position: number) {
  const div = document.createElement('div')
  const style = window.getComputedStyle(el)
  const properties = [
    'fontFamily',
    'fontSize',
    'fontWeight',
    'letterSpacing',
    'wordSpacing',
    'lineHeight',
    'paddingTop',
    'paddingLeft',
    'paddingBottom',
    'paddingRight',
    'boxSizing',
    'whiteSpace',
    'textTransform',
    'width',
    'borderLeftWidth',
  ]

  div.style.position = 'absolute'
  div.style.visibility = 'hidden'
  div.style.whiteSpace = 'pre-wrap'
  div.style.overflowWrap = 'break-word'
  div.style.top = '0'
  div.style.left = '0'

  properties.forEach((prop) => {
    ;(div.style as any)[prop] = style.getPropertyValue(prop)
  })

  const textBefore = el.value.substring(0, position)
  const textAfter = el.value.substring(position)

  div.textContent = textBefore
  const span = document.createElement('span')
  span.textContent = textAfter || '.'
  div.appendChild(span)

  document.body.appendChild(div)
  const { offsetTop: spanTop, offsetLeft: spanLeft } = span
  const { offsetTop: divTop, offsetLeft: divLeft } = div
  const lineHeight = parseFloat(style.lineHeight) || parseFloat(style.fontSize) * 1.4

  document.body.removeChild(div)

  const rect = el.getBoundingClientRect()
  return {
    top: rect.top + (spanTop - divTop) + lineHeight + 6 + window.scrollY,
    left: rect.left + (spanLeft - divLeft) + window.scrollX,
  }
}

function updateSuggestions() {
  const el = textareaEl.value
  if (!el) return

  caretPosition.value = el.selectionStart ?? 0
  const filtered = filteredSuggestions.value

  if (filtered.length > 0 && lastWord.value.length >= 1) {
    suggestions.value = filtered
    selectedSuggestionIdx.value = 0
    showSuggestions.value = true

    const coords = getCaretCoordinates(el, caretPosition.value)
    suggestionStyle.value = {
      top: `${coords.top}px`,
      left: `${coords.left}px`,
    }
  } else {
    showSuggestions.value = false
  }
}

function acceptSuggestion(suggestion: Suggestion) {
  const el = textareaEl.value
  if (!el) return

  const before = sqlInput.value.substring(0, caretPosition.value)
  const after = sqlInput.value.substring(caretPosition.value)

  const lastWordMatch = before.match(/(\S+)$/)
  if (lastWordMatch) {
    const start = before.length - lastWordMatch[0].length
    sqlInput.value = before.substring(0, start) + suggestion.text + after
  } else {
    sqlInput.value = before + suggestion.text + after
  }

  showSuggestions.value = false
  nextTick(() => {
    const pos =
      (before.match(/(\S+)$/) ? before.length - before.match(/(\S+)$/)![0].length : before.length) +
      suggestion.text.length
    el.focus()
    el.setSelectionRange(pos, pos)
  })
}

function onInput() {
  updateSuggestions()
}

function onTextareaKeydown(e: KeyboardEvent) {
  if (showSuggestions.value) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedSuggestionIdx.value = Math.min(
        selectedSuggestionIdx.value + 1,
        suggestions.value.length - 1,
      )
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedSuggestionIdx.value = Math.max(selectedSuggestionIdx.value - 1, 0)
      return
    }
    if (e.key === 'Enter' || e.key === 'Tab') {
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault()
        acceptSuggestion(suggestions.value[selectedSuggestionIdx.value])
        return
      }
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      showSuggestions.value = false
      return
    }
  }

  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
    e.preventDefault()
    executeSql()
  }
}

function onTextareaClick() {
  updateSuggestions()
}

function onTextareaBlur() {
  setTimeout(() => {
    showSuggestions.value = false
  }, 200)
}

async function fetchHistory() {
  try {
    const res = await api.get<SqlHistoryResponse>('/db/history', {
      params: { page: historyPage.value, page_size: historyPageSize.value },
    })
    if (res.data?.items) {
      history.value = res.data.items
      historyTotal.value = res.data.total
    }
  } catch {
    // 忽略获取历史失败
  }
}

async function executeSql() {
  const sql = sqlInput.value.trim()
  if (!sql) return

  executing.value = true
  result.value = null
  currentPage.value = 1
  lastSql.value = sql
  try {
    const res = await api.post<SqlResult>('/db/execute', { sql, page: 1, page_size: pageSize.value })
    result.value = res.data
    if (res.data.need_review) {
      confirmSubmitReview(sql, res.data.change_type || 'delete')
    } else if (res.data.success) {
      if (res.data.page_size > 0 && res.data.is_query) {
        pageSize.value = res.data.page_size
      }
      Message.success(res.data.message)
    } else {
      Message.error(res.data.message)
    }
    await fetchTableNames()
    historyPage.value = 1
    await fetchHistory()
  } catch (e: any) {
    const msg = e.response?.data?.detail || '请求失败'
    result.value = {
      success: false,
      message: msg,
      columns: [],
      rows: [],
      row_count: 0,
      total_count: 0,
      page_size: 0,
      is_query: false,
    }
    Message.error(msg)
  } finally {
    executing.value = false
  }
}

function confirmSubmitReview(sql: string, changeType: string) {
  const typeLabel = changeType === 'update' ? '修改（UPDATE）' : '删除（DELETE）'
  Modal.warning({
    title: '二次确认 - 提交审核',
    content: `您即将提交一条 SQL ${typeLabel}操作审核。\n\n${sql}\n\n提交后需要至少 2 名审核员审核通过才会自动执行，确定提交吗？`,
    hideCancel: false,
    okText: '确认提交审核',
    cancelText: '取消',
    onOk: async () => {
      try {
        await api.post('/db-changes/submit', {
          sql,
          change_type: changeType,
        })
        Message.success('已提交审核，等待审核员审批')
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '提交审核失败')
      }
    },
  })
}

function getRowPkCondition(record: Record<string, unknown>): { column: string; value: unknown } | null {
  const keys = Object.keys(record)
  const pkKey = keys.find((k) => k.toLowerCase() === 'id' || k.toLowerCase().endsWith('_id'))
  if (!pkKey) return null
  return { column: pkKey, value: record[pkKey] }
}

function getTableNameFromSql(sql: string): string | null {
  const m = sql.match(/FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)/i)
  return m ? m[1] : null
}

function requestRowDelete(record: Record<string, unknown>) {
  const table = getTableNameFromSql(lastSql.value)
  const pk = getRowPkCondition(record)
  if (!table || !pk) {
    Message.warning('无法确定该行所属表或主键，无法删除')
    return
  }

  const pkValue = typeof pk.value === 'number' ? pk.value : `'${String(pk.value).replace(/'/g, "''")}'`
  const deleteSql = `DELETE FROM ${table} WHERE ${pk.column} = ${pkValue};`

  Modal.warning({
    title: '二次确认 - 删除该行数据',
    content: `确定要删除该行数据吗？\n\n将提交以下 SQL 审核：\n${deleteSql}\n\n需至少 2 名审核员通过后自动执行。`,
    hideCancel: false,
    okText: '提交删除审核',
    cancelText: '取消',
    onOk: async () => {
      try {
        await api.post('/db-changes/submit', {
          sql: deleteSql,
          change_type: 'row_delete',
          description: `删除表 ${table} 中 ${pk.column}=${pk.value} 的记录`,
        })
        Message.success('已提交删除审核')
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '提交删除审核失败')
      }
    },
  })
}

async function executeSqlPage() {
  const sql = lastSql.value
  if (!sql) return

  executing.value = true
  try {
    const res = await api.post<SqlResult>('/db/execute', {
      sql,
      page: currentPage.value,
      page_size: pageSize.value,
    })
    result.value = res.data
    if (res.data.success) {
      Message.success(res.data.message)
    } else {
      Message.error(res.data.message)
    }
  } catch (e: any) {
    const msg = e.response?.data?.detail || '请求失败'
    Message.error(msg)
  } finally {
    executing.value = false
  }
}

function loadHistory(item: { sql_text: string }) {
  sqlInput.value = item.sql_text
  showSuggestions.value = false
  nextTick(() => {
    textareaEl.value?.focus()
  })
}

function applyPreset(sql: string) {
  sqlInput.value = sql
  nextTick(() => {
    textareaEl.value?.focus()
  })
}

function formatCell(value: string | number | null): string {
  if (value === null) return '<NULL>'
  if (value === '') return '<空>'
  return String(value)
}

function clearResult() {
  result.value = null
}

function getSuggestionIcon(type: string): string {
  switch (type) {
    case 'keyword':
      return 'λ'
    case 'table':
      return '⊡'
    case 'function':
      return '𝑓'
    case 'operator':
      return '⊕'
    default:
      return '·'
  }
}

const badgeType = (type: string): string => {
  switch (type) {
    case 'keyword':
      return 'keyword'
    case 'table':
      return 'table'
    case 'function':
      return 'function'
    case 'operator':
      return 'operator'
    default:
      return 'keyword'
  }
}

onMounted(() => {
  fetchTableNames()
  fetchHistory()
})
</script>

<template>
  <div class="db-page">
    <PageHeader title="数据库管理" subtitle="执行 SQL 命令管理数据库（仅超级管理员可用）">
      <template #actions>
        <a-tag color="red" size="small">
          <template #icon><IconStorage /></template>
          超级管理员
        </a-tag>
      </template>
    </PageHeader>

    <div class="db-layout">
      <div class="db-sidebar">
        <a-card :bordered="false" title="数据表" size="small" class="db-schema-card">
          <a-spin :loading="tableNames.length === 0" class="db-schema-body">
            <div v-if="tableNames.length === 0" class="text-center py-5 text-[13px] text-[#AEAEB2]">
              暂无数据表
            </div>
            <div
              v-for="tableName in tableNames"
              :key="tableName"
              class="db-schema-table"
              :class="{ 'is-active': activeTableName === tableName }"
            >
              <div class="db-schema-table-row" @click="toggleTableSchema(tableName)">
                <span
                  class="db-schema-chevron"
                  :class="{ 'is-open': activeTableName === tableName }"
                  >▸</span
                >
                <span class="db-schema-dot" />
                <span class="db-schema-table-name">{{ tableName }}</span>
                <span v-if="loadingSchema.has(tableName)" class="db-schema-spinner" />
                <a-button
                  type="text"
                  size="mini"
                  class="db-schema-query-btn"
                  @click.stop="selectTableSql(tableName)"
                >
                  查询
                </a-button>
              </div>

              <div
                v-if="activeTableName === tableName && tableSchemas[tableName]"
                class="db-schema-columns"
              >
                <div v-for="col in tableSchemas[tableName]" :key="col.cid" class="db-schema-col">
                  <span class="db-schema-col-name" :class="{ 'is-pk': col.pk === 1 }">{{
                    col.name
                  }}</span>
                  <span class="db-schema-col-type">{{ col.type || '—' }}</span>
                  <a-tag v-if="col.pk === 1" color="orange" size="small" class="db-schema-col-tag"
                    >PK</a-tag
                  >
                </div>
              </div>
            </div>
          </a-spin>
        </a-card>

        <a-card :bordered="false" title="执行历史" size="small" class="db-history-card">
          <div class="db-history-body">
            <a-empty v-if="history.length === 0" description="暂无执行记录" />
            <div v-else class="db-history-list">
              <div
                v-for="item in history"
                :key="item.id"
                class="db-history-item"
                :class="{ 'is-error': !item.is_success }"
                @click="loadHistory(item)"
              >
                <div class="db-history-item__meta">
                  <span class="db-history-item__user">{{ item.username }}</span>
                  <span class="db-history-item__time">{{ formatHistoryTime(item.created_at) }}</span>
                </div>
                <code class="db-history-item__sql">{{ item.sql_text }}</code>
              </div>
              <div v-if="historyTotal > historyPageSize" class="db-history-pagination">
                <a-pagination v-bind="historyPaginationConfig" />
              </div>
            </div>
          </div>
        </a-card>
      </div>

      <div class="db-main">
        <a-card :bordered="false" class="db-editor-card">
          <template #title>
            <div class="flex items-center gap-2">
              <IconCode :size="16" class="text-[#007AFF]" />
              SQL 编辑器
            </div>
          </template>
          <template #extra>
            <span class="text-[11px] text-[#AEAEB2]">Ctrl/Cmd + Enter 执行</span>
          </template>

          <div class="db-textarea-wrap">
            <textarea
              ref="textareaEl"
              v-model="sqlInput"
              class="db-textarea"
              placeholder="输入 SQL 命令… 输入时自动提示关键词和表名"
              rows="6"
              spellcheck="false"
              @input="onInput"
              @keydown="onTextareaKeydown"
              @click="onTextareaClick"
              @blur="onTextareaBlur"
            />

            <Teleport to="body">
              <div
                v-if="showSuggestions && suggestions.length > 0"
                class="db-suggest-panel"
                :style="suggestionStyle"
              >
                <div class="db-suggest-header">
                  <IconThunderbolt :size="12" />
                  智能提示
                </div>
                <div
                  v-for="(item, idx) in suggestions"
                  :key="item.text"
                  class="db-suggest-item"
                  :class="{ 'is-active': idx === selectedSuggestionIdx }"
                  @mousedown.prevent="acceptSuggestion(item)"
                  @mouseenter="selectedSuggestionIdx = idx"
                >
                  <span class="db-suggest-icon">{{ getSuggestionIcon(item.type) }}</span>
                  <div class="db-suggest-content">
                    <span class="db-suggest-text">{{ item.text }}</span>
                    <span class="db-suggest-desc">{{ item.description }}</span>
                  </div>
                  <span
                    class="db-suggest-badge"
                    :class="`db-suggest-badge--${badgeType(item.type)}`"
                  >
                    {{ item.category }}
                  </span>
                </div>
              </div>
            </Teleport>
          </div>

          <div class="db-toolbar">
            <div class="db-presets">
              <a-tag
                v-for="preset in tablePresets"
                :key="preset.sql"
                color="arcoblue"
                class="db-preset-tag"
                @click="applyPreset(preset.sql)"
              >
                <span class="text-[12px] mr-1">{{ preset.icon }}</span>
                {{ preset.label }}
              </a-tag>
            </div>
            <a-space :size="8">
              <a-button :disabled="!sqlInput" @click="sqlInput = ''">
                <template #icon><IconDelete /></template>
                清空
              </a-button>
              <a-button type="primary" :loading="executing" @click="executeSql">
                <template #icon><IconPlayArrow /></template>
                执行
              </a-button>
            </a-space>
          </div>
        </a-card>

        <a-card v-if="result" :bordered="false" class="db-result-card">
          <template #title>
            <div class="flex items-center gap-2">
              <span class="db-result-dot" :class="result.success ? 'is-ok' : 'is-err'" />
              {{ result.is_query ? `查询结果 · ${result.row_count} 行` : '执行结果' }}
            </div>
          </template>
          <template #extra>
            <a-button type="text" size="mini" @click="clearResult">
              <template #icon><IconDelete /></template>
            </a-button>
          </template>

          <a-alert :type="result.success ? 'success' : 'error'" class="mb-4">
            {{ result.message }}
          </a-alert>

          <template v-if="result.is_query && result.columns.length > 0">
            <a-table
              :columns="[
                ...result.columns.map((c) => ({ title: c, dataIndex: c, ellipsis: true, width: 160 })),
                { title: '操作', slotName: 'rowActions', width: 80, fixed: 'right' as const },
              ]"
              :data="
                result.rows.map((row) => {
                  const obj: Record<string, unknown> = {}
                  result!.columns.forEach((col, i) => {
                    obj[col] = row[i]
                  })
                  return obj
                })
              "
              :bordered="false"
              :hoverable="true"
              :pagination="paginationConfig"
              size="small"
              :scroll="{ x: 'max-content' }"
            >
              <template #bodyCell="{ column, record }">
                <span
                  class="db-cell"
                  :class="{
                    'db-cell--null': record[column.dataIndex] === null,
                    'db-cell--empty': record[column.dataIndex] === '',
                  }"
                >
                  {{ formatCell(record[column.dataIndex] as string | number | null) }}
                </span>
              </template>
              <template #rowActions="{ record }">
                <a-button
                  type="text"
                  status="danger"
                  size="mini"
                  title="删除该行（需审核）"
                  @click="requestRowDelete(record)"
                >
                  <template #icon><IconDelete /></template>
                </a-button>
              </template>
              <template #empty>
                <a-empty description="无数据" />
              </template>
            </a-table>
          </template>
        </a-card>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.db-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.db-layout {
  display: flex;
  gap: 14px;
  align-items: flex-start;

  @media (max-width: 900px) {
    flex-direction: column;
  }
}

.db-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.db-sidebar {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;

  @media (max-width: 900px) {
    width: 100%;
  }
}

.db-schema-card {
  :deep(.arco-card-header) {
    font-size: 13px;
    font-weight: 600;
    color: var(--apple-text-secondary);
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }
  :deep(.arco-card-body) {
    padding: 8px 12px;
  }
}

.db-schema-body {
  max-height: 340px;
  overflow: auto;
  width: 100%;
}

.db-schema-table {
  margin-bottom: 1px;

  &.is-active {
    background: rgba(0, 122, 255, 0.04);
    border-radius: 10px;
  }
}

.db-schema-table-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.15s;
  user-select: none;

  &:hover {
    background: rgba(0, 122, 255, 0.04);
  }
}

.db-schema-chevron {
  font-size: 10px;
  color: var(--apple-text-tertiary);
  transition: transform 0.15s cubic-bezier(0.25, 0.1, 0.25, 1);
  flex-shrink: 0;
  line-height: 1;

  &.is-open {
    transform: rotate(90deg);
  }
}

.db-schema-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--apple-green);
  flex-shrink: 0;
}

.db-schema-table-name {
  font-size: 13px;
  color: var(--apple-text-primary);
  font-weight: 500;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.db-schema-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid rgba(0, 122, 255, 0.15);
  border-top-color: var(--apple-blue);
  border-radius: 50%;
  animation: db-spin 0.6s linear infinite;
  flex-shrink: 0;
}

.db-schema-query-btn {
  opacity: 0;
  transform: translateX(-4px);
  transition: all 0.15s cubic-bezier(0.25, 0.1, 0.25, 1);

  .db-schema-table-row:hover & {
    opacity: 1;
    transform: translateX(0);
  }
}

.db-schema-columns {
  margin: 2px 0 4px 22px;
  border-left: 1px solid rgba(0, 0, 0, 0.06);
  padding-left: 12px;
}

.db-schema-col {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 0;
  font-size: 12px;
}

.db-schema-col-name {
  color: var(--apple-text-primary);
  font-weight: 500;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, monospace;

  &.is-pk {
    color: var(--apple-orange);
  }
}

.db-schema-col-type {
  color: var(--apple-text-tertiary);
  font-size: 11px;
  margin-left: auto;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, monospace;
}

.db-schema-col-tag {
  transform: scale(0.75);
  transform-origin: center;
}

.db-history-card {
  :deep(.arco-card-header) {
    font-size: 13px;
    font-weight: 600;
    color: var(--apple-text-secondary);
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }
  :deep(.arco-card-body) {
    padding: 6px 8px;
  }
}

.db-history-body {
  max-height: 280px;
  overflow-y: auto;
}

.db-history-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.db-history-item {
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.15s;

  &:hover {
    background: rgba(0, 122, 255, 0.04);
  }

  &.is-error {
    .db-history-item__sql {
      color: var(--apple-red);
    }
  }
}

.db-history-item__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 2px;
}

.db-history-item__user {
  font-size: 10px;
  color: var(--apple-text-secondary);
  font-weight: 500;
}

.db-history-item__time {
  font-size: 10px;
  color: var(--apple-text-tertiary);
  font-variant-numeric: tabular-nums;
}

.db-history-item__sql {
  display: block;
  font-size: 11px;
  color: var(--apple-text-secondary);
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.db-history-pagination {
  display: flex;
  justify-content: center;
  padding: 8px 0 4px;
}

.db-editor-card {
  :deep(.arco-card-body) {
    padding: 20px;
  }
}

.db-textarea-wrap {
  position: relative;
}

.db-textarea {
  width: 100%;
  min-height: 150px;
  padding: 14px 16px;
  resize: vertical;
  border: 1px solid rgba(0, 0, 0, 0.07);
  border-radius: 10px;
  background: var(--apple-bg);
  color: var(--apple-text-primary);
  font-family: 'SF Mono', 'JetBrains Mono', 'Fira Code', ui-monospace, Menlo, Monaco, monospace;
  font-size: 14px;
  line-height: 1.65;
  outline: none;
  transition:
    border-color 0.18s,
    box-shadow 0.18s,
    background 0.18s;

  &::placeholder {
    color: var(--apple-text-tertiary);
  }

  &:hover {
    border-color: rgba(0, 0, 0, 0.14);
  }

  &:focus {
    border-color: var(--apple-blue);
    box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.16);
    background: var(--apple-card);
  }
}

.db-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-top: 14px;
  flex-wrap: wrap;
}

.db-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex: 1;
}

.db-preset-tag {
  cursor: pointer;
  transition: transform 0.15s cubic-bezier(0.25, 0.1, 0.25, 1);

  &:hover {
    transform: translateY(-1px);
  }
}

.db-suggest-panel {
  position: absolute;
  z-index: 9999;
  min-width: 320px;
  max-width: 480px;
  max-height: 280px;
  overflow-y: auto;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.96);
  border: 1px solid rgba(0, 0, 0, 0.06);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.12),
    0 0 0 1px rgba(0, 0, 0, 0.02) inset;
  backdrop-filter: blur(20px) saturate(180%);
}

.db-suggest-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  font-size: 10px;
  font-weight: 600;
  color: var(--apple-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.db-suggest-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  cursor: pointer;
  transition: background 0.1s;

  &.is-active {
    background: rgba(0, 122, 255, 0.06);
  }
}

.db-suggest-icon {
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  font-size: 12px;
  color: var(--apple-blue);
  background: rgba(0, 122, 255, 0.06);
  flex-shrink: 0;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, monospace;
}

.db-suggest-content {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex: 1;
  min-width: 0;
}

.db-suggest-text {
  font-size: 13px;
  color: var(--apple-text-primary);
  font-weight: 500;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, monospace;
}

.db-suggest-desc {
  font-size: 11px;
  color: var(--apple-text-tertiary);
}

.db-suggest-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 600;
  letter-spacing: 0.04em;
  flex-shrink: 0;

  &--keyword {
    color: var(--apple-blue);
    background: rgba(0, 122, 255, 0.06);
  }

  &--table {
    color: var(--apple-green);
    background: rgba(52, 199, 89, 0.06);
  }

  &--function {
    color: var(--apple-orange);
    background: rgba(255, 149, 0, 0.06);
  }

  &--operator {
    color: var(--apple-indigo);
    background: rgba(88, 86, 214, 0.06);
  }
}

.db-result-card {
  :deep(.arco-card-body) {
    padding: 16px 20px;
    overflow-x: auto;
  }
}

.db-result-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;

  &.is-ok {
    background: var(--apple-green);
  }

  &.is-err {
    background: var(--apple-red);
  }
}

.db-cell {
  font-size: 13px;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, monospace;

  &--null {
    color: var(--apple-red);
    font-style: italic;
  }

  &--empty {
    color: var(--apple-text-tertiary);
  }
}

@keyframes db-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 480px) {
  .db-editor-card :deep(.arco-card-body) {
    padding: 14px 16px;
  }
  .db-result-card :deep(.arco-card-body) {
    padding: 12px 14px;
  }
  .db-textarea {
    font-size: 13px;
    min-height: 120px;
    padding: 12px 14px;
  }
  .db-schema-card :deep(.arco-card-header) {
    font-size: 11px;
  }
  .db-history-card :deep(.arco-card-header) {
    font-size: 11px;
  }
  .db-suggest-panel {
    min-width: 260px;
    max-width: calc(100vw - 24px);
  }
}

@media (max-width: 248px) {
  .db-editor-card :deep(.arco-card-body) {
    padding: 10px 12px;
  }
  .db-result-card :deep(.arco-card-body) {
    padding: 10px 12px;
  }
  .db-textarea {
    font-size: 12px;
    min-height: 100px;
    padding: 10px 12px;
  }
  .db-toolbar {
    gap: 10px;
    margin-top: 10px;
  }
  .db-suggest-panel {
    min-width: 200px;
  }
  .db-suggest-text {
    font-size: 12px;
  }
  .db-suggest-desc {
    font-size: 10px;
  }
}
</style>
