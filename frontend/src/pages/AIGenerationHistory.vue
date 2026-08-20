<template>
  <div class="page-main">
    <PageHeader title="生成历史" subtitle="AI 文案生成记录、质量评分与批量导出">
      <template #actions>
        <a-space :size="8">
          <a-button
            type="text"
            size="mini"
            class="!text-[#007AFF] !px-0 !h-auto"
            @click="exportCsv"
          >
            <template #icon><IconExport :size="13" /></template>
            导出 CSV
          </a-button>
          <a-button
            type="text"
            size="mini"
            class="!text-[#007AFF] !px-0 !h-auto"
            @click="exportJson"
          >
            <template #icon><IconCode :size="13" /></template>
            导出 JSON
          </a-button>
          <a-button
            type="text"
            size="mini"
            class="!text-[#007AFF] !px-0 !h-auto"
            @click="router.push('/content/create')"
          >
            <template #icon><IconPlus :size="13" /></template>
            新建创作
          </a-button>
        </a-space>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-space wrap :size="12" class="mb-5">
        <a-input-search
          v-model="searchQuery"
          placeholder="搜索主题或标题"
          allow-clear
          style="width: 260px"
        />
        <a-select v-model="platformFilter" placeholder="全部平台" style="width: 140px">
          <a-option value="all">全部平台</a-option>
          <a-option value="wechat_mp">微信公众号</a-option>
          <a-option value="xiaohongshu">小红书</a-option>
          <a-option value="douyin">抖音</a-option>
          <a-option value="wechat_video">视频号</a-option>
          <a-option value="wechat_moments">朋友圈</a-option>
          <a-option value="weibo">微博</a-option>
        </a-select>
        <a-select v-model="qualityFilter" placeholder="全部评分" style="width: 130px">
          <a-option value="all">全部评分</a-option>
          <a-option value="excellent">优秀 (≥90)</a-option>
          <a-option value="good">良好 (75-89)</a-option>
          <a-option value="pass">达标 (60-74)</a-option>
          <a-option value="poor">待优化 (&lt;60)</a-option>
        </a-select>
      </a-space>

      <a-spin :loading="loading" tip="加载中...">
        <a-table
          :columns="columns"
          :data="filteredRecords"
          :bordered="false"
          :hoverable="true"
          :pagination="{
            showTotal: true,
            defaultPageSize: 20,
            pageSizeOptions: [10, 20, 50],
          }"
        >
          <template #platform="{ record }">
            <PlatformIcon :platform="record.platform as any" size="sm" />
          </template>
          <template #topic="{ record }">
            <a-space :size="8" align="center">
              <span>{{ record.topic || '—' }}</span>
              <a-tag v-if="record.campaign_id" color="arcoblue" size="small">活动</a-tag>
            </a-space>
          </template>
          <template #title="{ record }">
            <span class="record-title" :title="record.title">{{ record.title }}</span>
          </template>
          <template #score="{ record }">
            <a-tooltip :content="scoreTooltip(record)">
              <a-tag :color="scoreColor(record.score.total)" size="small" class="!m-0">
                {{ record.score.total }} · {{ record.score.comment }}
              </a-tag>
            </a-tooltip>
          </template>
          <template #model="{ record }">
            <span class="record-model">{{ record.model || '—' }}</span>
          </template>
          <template #createdAt="{ record }">
            {{ formatDate(record.created_at) }}
          </template>
          <template #actions="{ record }">
            <a-space :size="4">
              <a-button type="text" size="small" title="复制文案" @click="copyRecord(record)">
                <template #icon><IconCopy /></template>
              </a-button>
              <a-button type="text" size="small" title="保存为内容" @click="saveToContent(record)">
                <template #icon><IconSave /></template>
              </a-button>
              <a-button type="text" size="small" title="查看详情" @click="openDetail(record)">
                <template #icon><IconEye /></template>
              </a-button>
            </a-space>
          </template>
        </a-table>
      </a-spin>
    </div>

    <a-modal
      v-model:visible="detailVisible"
      :title="detail?.title || '生成详情'"
      :footer="false"
      :width="640"
    >
      <div v-if="detail" class="detail-wrap">
        <a-descriptions :column="2" size="small" class="mb-4">
          <a-descriptions-item label="创作主题">
            {{ detail.topic || '—' }}
          </a-descriptions-item>
          <a-descriptions-item label="目标平台">
            <PlatformIcon :platform="detail.platform as any" size="sm" />
          </a-descriptions-item>
          <a-descriptions-item label="生成模型">
            {{ detail.model || '—' }}
          </a-descriptions-item>
          <a-descriptions-item label="质量评分">
            <a-tag :color="scoreColor(detail.score.total)" size="small">
              {{ detail.score.total }} · {{ detail.score.comment }}
            </a-tag>
          </a-descriptions-item>
        </a-descriptions>
        <div class="detail-section">
          <div class="detail-label">标题</div>
          <div class="detail-title">{{ detail.title }}</div>
        </div>
        <div class="detail-section">
          <div class="detail-label">正文</div>
          <div class="detail-body">{{ detail.body }}</div>
        </div>
        <div v-if="detail.hashtags.length" class="detail-section">
          <div class="detail-label">话题标签</div>
          <a-space wrap :size="6">
            <a-tag v-for="t in detail.hashtags" :key="t" color="arcoblue" size="small">
              #{{ t }}
            </a-tag>
          </a-space>
        </div>
        <div class="detail-section">
          <div class="detail-label">评分详情</div>
          <a-space wrap :size="6">
            <a-tag v-for="t in detail.score.tags" :key="t" size="small">{{ t }}</a-tag>
          </a-space>
        </div>
        <div class="flex items-center justify-end gap-2 mt-5">
          <a-button @click="copyRecord(detail)">复制文案</a-button>
          <a-button type="primary" @click="saveToContent(detail)">保存为内容</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconCopy,
  IconSave,
  IconEye,
  IconPlus,
  IconExport,
  IconCode,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import api from '@/utils/api'

const router = useRouter()

interface ScoreInfo {
  total: number
  title_score: number
  body_score: number
  tag_score: number
  extra_score: number
  tags: string[]
  comment: string
}

interface GenerationRecord {
  id: string
  user_id: string
  topic: string
  platform: string
  plan_id: string
  model: string
  title: string
  body: string
  hashtags: string[]
  campaign_id: string
  created_at: string
  score: ScoreInfo
}

const loading = ref(false)
const records = ref<GenerationRecord[]>([])
const searchQuery = ref('')
const platformFilter = ref('all')
const qualityFilter = ref('all')

const columns = [
  { title: '平台', dataIndex: 'platform', slotName: 'platform', width: 90 },
  { title: '主题', dataIndex: 'topic', slotName: 'topic' },
  { title: '标题', dataIndex: 'title', slotName: 'title', width: 200 },
  { title: '评分', dataIndex: 'score', slotName: 'score', width: 130 },
  { title: '模型', dataIndex: 'model', slotName: 'model', width: 140 },
  { title: '生成时间', dataIndex: 'created_at', slotName: 'createdAt', width: 150 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 130 },
]

onMounted(fetchRecords)

async function fetchRecords() {
  loading.value = true
  try {
    const res = await api.get<GenerationRecord[]>('/contents/ai-generations')
    records.value = Array.isArray(res.data) ? res.data : []
  } catch {
    Message.error('加载生成记录失败')
  } finally {
    loading.value = false
  }
}

const filteredRecords = computed(() => {
  return records.value.filter((r) => {
    const q = searchQuery.value.trim().toLowerCase()
    const matchSearch = !q || r.topic.toLowerCase().includes(q) || r.title.toLowerCase().includes(q)
    const matchPlatform = platformFilter.value === 'all' || r.platform === platformFilter.value
    let matchQuality = true
    if (qualityFilter.value === 'excellent') matchQuality = r.score.total >= 90
    else if (qualityFilter.value === 'good')
      matchQuality = r.score.total >= 75 && r.score.total < 90
    else if (qualityFilter.value === 'pass')
      matchQuality = r.score.total >= 60 && r.score.total < 75
    else if (qualityFilter.value === 'poor') matchQuality = r.score.total < 60
    return matchSearch && matchPlatform && matchQuality
  })
})

function formatDate(value: string) {
  if (!value) return '—'
  const d = new Date(value)
  if (isNaN(d.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function scoreColor(total: number) {
  if (total >= 90) return 'green'
  if (total >= 75) return 'arcoblue'
  if (total >= 60) return 'orange'
  return 'red'
}

function scoreTooltip(record: GenerationRecord) {
  const s = record.score
  return `标题 ${s.title_score} · 正文 ${s.body_score} · 标签 ${s.tag_score} · 附加 ${s.extra_score}`
}

async function copyRecord(record: GenerationRecord) {
  const tags = (record.hashtags || [])
    .map((t) => t.trim().replace(/^#+/, ''))
    .filter(Boolean)
    .map((t) => `#${t}`)
    .join(' ')
  const text = `${record.title}\n\n${record.body}${tags ? `\n${tags}` : ''}`
  try {
    await navigator.clipboard.writeText(text)
    Message.success('已复制到剪贴板')
  } catch {
    Message.error('复制失败，请手动选择文本复制')
  }
}

async function saveToContent(record: GenerationRecord) {
  const tags = (record.hashtags || [])
    .map((t) => t.trim().replace(/^#+/, ''))
    .filter(Boolean)
    .map((t) => `#${t}`)
    .join(' ')
  try {
    await api.post('/contents/', {
      title: record.title,
      body: `${record.body}${tags ? `\n${tags}` : ''}`,
      platform: record.platform,
      status: 'draft',
      ...(record.campaign_id ? { campaign_id: record.campaign_id } : {}),
    })
    Message.success('已保存为内容草稿')
  } catch {
    Message.error('保存失败，请重试')
  }
}

const detailVisible = ref(false)
const detail = ref<GenerationRecord | null>(null)

function openDetail(record: GenerationRecord) {
  detail.value = record
  detailVisible.value = true
}

function exportUrl(format: 'csv' | 'json') {
  const params = new URLSearchParams({ format })
  if (platformFilter.value !== 'all') params.set('platform', platformFilter.value)
  if (searchQuery.value.trim()) params.set('topic', searchQuery.value.trim())
  return `/api/contents/ai-generations/export?${params.toString()}`
}

async function exportCsv() {
  download(exportUrl('csv'), `ai-generations-${Date.now()}.csv`)
}

async function exportJson() {
  download(exportUrl('json'), `ai-generations-${Date.now()}.json`)
}

function download(url: string, filename: string) {
  const token = localStorage.getItem('token')
  fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
    .then((res) => {
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      return res.blob()
    })
    .then((blob) => {
      const objectUrl = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = objectUrl
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(objectUrl)
      Message.success('导出成功')
    })
    .catch(() => Message.error('导出失败'))
}
</script>

<style scoped lang="scss">
.record-title {
  display: block;
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.record-model {
  font-size: 12px;
  color: #86909c;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.detail-wrap {
  padding-top: 4px;
}

.detail-section {
  margin-bottom: 16px;
}

.detail-label {
  font-size: 12px;
  color: #86909c;
  font-weight: 600;
  margin-bottom: 6px;
}

.detail-title {
  font-size: 16px;
  font-weight: 600;
}

.detail-body {
  white-space: pre-wrap;
  font-size: 14px;
  line-height: 1.7;
  color: #1d2129;
}
</style>
