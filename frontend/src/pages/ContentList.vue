<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { IconPlus, IconEdit, IconDelete, IconStar } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import api from '@/utils/api'

const router = useRouter()

const searchQuery = ref('')
const platformFilter = ref('all')
const statusFilter = ref('all')
const loading = ref(false)

interface Content {
  id: number
  user_id: number
  title: string
  body: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  status: 'draft' | 'ready' | 'published'
  media_urls: string[]
  ai_generated: boolean
  original_content_id: number | null
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

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

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

async function removeContent(id: number) {
  try {
    await api.delete(`/contents/${id}`)
    contents.value = contents.value.filter((c) => c.id !== id)
    Message.success('删除成功')
  } catch (e) {
    Message.error('删除失败')
  }
}
</script>

<template>
  <div>
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
            <a-button type="text" size="small" title="编辑">
              <template #icon><IconEdit /></template>
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
  </div>
</template>
