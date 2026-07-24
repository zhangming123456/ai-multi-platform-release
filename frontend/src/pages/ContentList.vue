<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconStar,
  IconCopy,
  IconEye,
  IconSend,
} from '@arco-design/web-vue/es/icon'
import { Message, Modal } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import api from '@/utils/api'
import { formatDateTime as formatDate } from '@/utils/time'

const router = useRouter()

const searchQuery = ref('')
const platformFilter = ref('all')
const statusFilter = ref('all')
const loading = ref(false)

interface Content {
  id: string
  user_id: string
  title: string
  body: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  status: 'draft' | 'ready' | 'published' | 'pending_review' | 'rejected'
  media_urls: string[]
  ai_generated: boolean
  original_content_id: string | null
  created_at: string
  updated_at: string
}

const contents = ref<Content[]>([])

const filteredContents = computed(() => {
  return contents.value.filter((c) => {
    const matchSearch = !searchQuery.value || c.title.includes(searchQuery.value)
    const matchPlatform = platformFilter.value === 'all' || c.platform === platformFilter.value
    const matchStatus = statusFilter.value === 'all' || c.status === statusFilter.value
    return matchSearch && matchPlatform && matchStatus
  })
})

const columns = [
  { title: '标题', dataIndex: 'title', slotName: 'title' },
  { title: '平台', dataIndex: 'platform', slotName: 'platform' },
  { title: '状态', dataIndex: 'status', slotName: 'status' },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt' },
  { title: '操作', slotName: 'actions', align: 'right' as const },
]

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/contents/')
    contents.value = res.data
  } catch (e) {
    Message.error('加载内容失败')
  } finally {
    loading.value = false
  }
})

const editVisible = ref(false)
const editSaving = ref(false)
const editingContent = ref<Content | null>(null)
const editForm = ref({
  title: '',
  body: '',
  platform: 'xiaohongshu' as Content['platform'],
  status: 'draft' as Content['status'],
})

function openEdit(content: Content) {
  editingContent.value = content
  editForm.value = {
    title: content.title,
    body: content.body,
    platform: content.platform,
    status: content.status,
  }
  editVisible.value = true
}

async function saveEdit() {
  if (!editingContent.value) return
  editSaving.value = true
  try {
    const res = await api.put(`/contents/${editingContent.value.id}`, editForm.value)
    const idx = contents.value.findIndex((c) => c.id === editingContent.value!.id)
    if (idx !== -1) {
      contents.value[idx] = { ...contents.value[idx], ...res.data }
    }
    Message.success('内容已更新')
    editVisible.value = false
  } catch (e) {
    Message.error('更新失败')
  } finally {
    editSaving.value = false
  }
}

const detailVisible = ref(false)
const detailContent = ref<Content | null>(null)

function openDetail(content: Content) {
  detailContent.value = content
  detailVisible.value = true
}

async function copyContent(content: Content) {
  const text = `${content.title}\n\n${content.body}`
  try {
    await navigator.clipboard.writeText(text)
    Message.success('内容已复制到剪贴板')
  } catch {
    Message.error('复制失败，请手动选择文本复制')
  }
}

async function removeContent(id: string) {
  Modal.warning({
    title: '确认删除',
    content: '删除后不可恢复，确定要删除这条内容吗？',
    hideCancel: false,
    onOk: async () => {
      try {
        await api.delete(`/contents/${id}`)
        contents.value = contents.value.filter((c) => c.id !== id)
        Message.success('删除成功')
      } catch (e) {
        Message.error('删除失败')
      }
    },
  })
}

async function submitForReview(id: string) {
  try {
    await api.post(`/reviews/${id}/submit`)
    const idx = contents.value.findIndex(c => c.id === id)
    if (idx !== -1) {
      contents.value[idx].status = 'pending_review'
    }
    Message.success('已提交审核')
  } catch (e) {
    Message.error('提交审核失败')
  }
}
</script>

<template>
  <div class="content-list">
    <PageHeader title="内容工坊" subtitle="创作并管理适配各平台的内容">
      <template #actions>
        <a-button type="primary" @click="router.push('/content/create')">
          <template #icon><IconPlus /></template>
          创建内容
        </a-button>
      </template>
    </PageHeader>

    <a-space wrap :size="12" class="mb-5">
      <a-input-search
        v-model="searchQuery"
        placeholder="搜索内容标题"
        allow-clear
        style="width: 260px"
      />
      <a-select v-model="platformFilter" placeholder="全部平台" style="width: 140px">
        <a-option value="all">全部平台</a-option>
        <a-option value="wechat_mp">微信公众号</a-option>
        <a-option value="xiaohongshu">小红书</a-option>
        <a-option value="douyin">抖音</a-option>
        <a-option value="wechat_video">视频号</a-option>
      </a-select>
      <a-select v-model="statusFilter" placeholder="全部状态" style="width: 120px">
        <a-option value="all">全部状态</a-option>
        <a-option value="draft">草稿</a-option>
        <a-option value="ready">待发布</a-option>
        <a-option value="published">已发布</a-option>
      </a-select>
    </a-space>

    <a-spin :loading="loading" tip="加载中...">
      <a-table
        :columns="columns"
        :data="filteredContents"
        :bordered="false"
        :hoverable="true"
        :pagination="false"
      >
        <template #title="{ record }">
          <a-space :size="8" align="center">
            <span>{{ record.title }}</span>
            <a-tag v-if="record.ai_generated" color="purple" size="small">
              <template #icon><IconStar /></template>
              AI
            </a-tag>
          </a-space>
        </template>
        <template #platform="{ record }">
          <PlatformIcon :platform="record.platform" size="sm" />
        </template>
        <template #status="{ record }">
          <StatusBadge :status="record.status" />
        </template>
        <template #createdAt="{ record }">
          {{ formatDate(record.created_at) }}
        </template>
        <template #actions="{ record }">
          <a-space :size="4">
            <a-button type="text" size="small" title="查看" @click="openDetail(record)">
              <template #icon><IconEye /></template>
            </a-button>
            <a-button type="text" size="small" title="复制" @click="copyContent(record)">
              <template #icon><IconCopy /></template>
            </a-button>
            <a-button type="text" size="small" title="编辑" @click="openEdit(record)">
              <template #icon><IconEdit /></template>
            </a-button>
            <a-button
              v-if="['draft', 'rejected'].includes(record.status)"
              type="text"
              size="small"
              title="提交审核"
              @click="submitForReview(record.id)"
            >
              <template #icon><IconSend /></template>
            </a-button>
            <a-button
              type="text"
              status="danger"
              size="small"
              title="删除"
              @click="removeContent(record.id)"
            >
              <template #icon><IconDelete /></template>
            </a-button>
          </a-space>
        </template>
        <template #empty>
          <a-empty description="暂无内容" />
        </template>
      </a-table>
    </a-spin>

    <a-modal
      v-model:visible="editVisible"
      title="编辑内容"
      :width="640"
      @ok="saveEdit"
      :ok-loading="editSaving"
      ok-text="保存"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="标题">
          <a-input v-model="editForm.title" placeholder="内容标题" />
        </a-form-item>
        <a-form-item label="正文">
          <a-textarea
            v-model="editForm.body"
            placeholder="内容正文"
            :auto-size="{ minRows: 4, maxRows: 10 }"
          />
        </a-form-item>
        <a-form-item label="平台">
          <a-select v-model="editForm.platform">
            <a-option value="wechat_mp">微信公众号</a-option>
            <a-option value="xiaohongshu">小红书</a-option>
            <a-option value="douyin">抖音</a-option>
            <a-option value="wechat_video">视频号</a-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="detailVisible"
      title="内容详情"
      :width="640"
      :footer="false"
    >
      <template v-if="detailContent">
        <a-descriptions :column="2" bordered size="small" class="mb-4">
          <a-descriptions-item label="平台">
            <PlatformIcon :platform="detailContent.platform" size="sm" />
          </a-descriptions-item>
          <a-descriptions-item label="状态">
            <StatusBadge :status="detailContent.status" />
          </a-descriptions-item>
          <a-descriptions-item label="创建时间">
            {{ formatDate(detailContent.created_at) }}
          </a-descriptions-item>
          <a-descriptions-item label="更新时间">
            {{ formatDate(detailContent.updated_at) }}
          </a-descriptions-item>
          <a-descriptions-item label="AI 生成" :span="2">
            <a-tag v-if="detailContent.ai_generated" color="purple" size="small">
              <template #icon><IconStar /></template>
              是
            </a-tag>
            <span v-else class="text-[#aeaeb2] text-[13px]">否</span>
          </a-descriptions-item>
        </a-descriptions>
        <div class="mb-3">
          <div class="text-[11px] font-semibold uppercase tracking-[0.06em] text-[#aeaeb2] mb-1.5">标题</div>
          <div class="text-[15px] font-semibold text-[#1d1d1f]">{{ detailContent.title }}</div>
        </div>
        <div>
          <div class="text-[11px] font-semibold uppercase tracking-[0.06em] text-[#aeaeb2] mb-1.5">正文</div>
          <div class="text-[14px] text-[#3a3a3c] leading-relaxed whitespace-pre-wrap bg-[#f5f5f7] rounded-lg p-4">{{ detailContent.body }}</div>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<style scoped lang="scss">
.content-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

@media (max-width: 248px) {
  :deep(.arco-card-header) {
    padding: 8px 10px !important;
  }
  :deep(.arco-card-body) {
    padding: 10px !important;
  }
  :deep(.arco-card-header-title) {
    font-size: 13px !important;
  }
  :deep(.arco-table-th),
  :deep(.arco-table-td) {
    padding: 6px 8px !important;
    font-size: 11px !important;
  }
  :deep(.arco-table-cell) {
    font-size: 11px !important;
  }
  :deep(.arco-pagination-item) {
    min-width: 24px !important;
    height: 24px !important;
    line-height: 22px !important;
    font-size: 11px !important;
  }
}
</style>
