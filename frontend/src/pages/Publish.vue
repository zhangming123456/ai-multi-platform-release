<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { IconPlus, IconEye, IconRefresh, IconDelete } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import api from '@/utils/api'
import PageHeader from '@/components/layout/PageHeader.vue'
import SegmentedControl from '@/components/shared/SegmentedControl.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import Modal from '@/components/shared/Modal.vue'

type Platform = 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
type TaskStatus = 'pending' | 'publishing' | 'published' | 'failed'

interface PublishTask {
  id: number
  content_id: number
  account_id: number
  status: TaskStatus
  scheduled_at: string | null
  published_at: string | null
  error_message: string | null
  retry_count: number
  created_at: string
  updated_at: string
}

interface Content {
  id: number
  title: string
  platform: string
  status: string
}

interface Account {
  id: number
  platform: string
  nickname: string
  status: string
}

const activeTab = ref('all')
const showCreateModal = ref(false)
const loading = ref(false)
const submitting = ref(false)

const formData = ref({
  contentId: undefined as number | undefined,
  accountId: undefined as number | undefined,
  scheduledAt: undefined as string | undefined,
})

const tabs = [
  { key: 'all', label: '全部' },
  { key: 'pending', label: '待发布' },
  { key: 'publishing', label: '发布中' },
  { key: 'published', label: '已完成' },
  { key: 'failed', label: '失败' },
]

const platformChoices: Record<string, string> = {
  wechat_mp: '微信公众号',
  xiaohongshu: '小红书',
  douyin: '抖音',
  wechat_video: '视频号',
}

const publishTasks = ref<PublishTask[]>([])
const contents = ref<Content[]>([])
const accounts = ref<Account[]>([])

const filteredTasks = computed(() => {
  const enriched = publishTasks.value.map((task) => {
    const content = contents.value.find((c) => c.id === task.content_id)
    const account = accounts.value.find((a) => a.id === task.account_id)
    return {
      ...task,
      contentTitle: content?.title || `内容 #${task.content_id}`,
      accountName: account?.nickname || `账号 #${task.account_id}`,
      platform: (account?.platform || 'wechat_mp') as Platform,
    }
  })
  if (activeTab.value === 'all') return enriched
  return enriched.filter((t) => t.status === activeTab.value)
})

const columns = [
  { title: '内容标题', dataIndex: 'contentTitle' },
  { title: '平台', dataIndex: 'platform', slotName: 'platform' },
  { title: '目标账号', dataIndex: 'accountName' },
  { title: '状态', dataIndex: 'status', slotName: 'status' },
  { title: '计划时间', dataIndex: 'scheduled_at' },
  { title: '操作', slotName: 'actions', align: 'right' as const },
]

async function loadTasks() {
  loading.value = true
  try {
    const [tasksRes, contentsRes, accountsRes] = await Promise.all([
      api.get('/publish/tasks'),
      api.get('/contents/'),
      api.get('/accounts/'),
    ])
    publishTasks.value = tasksRes.data
    contents.value = contentsRes.data
    accounts.value = accountsRes.data
  } catch {
    Message.error('加载数据失败')
  } finally {
    loading.value = false
  }
}

async function createTask() {
  if (!formData.value.contentId || !formData.value.accountId) {
    Message.warning('请选择内容和账号')
    return
  }
  submitting.value = true
  try {
    await api.post('/publish/tasks', {
      content_id: formData.value.contentId,
      account_id: formData.value.accountId,
      scheduled_at: formData.value.scheduledAt || undefined,
    })
    Message.success('创建发布任务成功')
    showCreateModal.value = false
    formData.value = {
      contentId: undefined,
      accountId: undefined,
      scheduledAt: undefined,
    }
    await loadTasks()
  } catch {
    Message.error('创建发布任务失败')
  } finally {
    submitting.value = false
  }
}

async function retryTask(id: number) {
  try {
    await api.post(`/publish/tasks/${id}/retry`)
    Message.success('已重新触发发布')
    await loadTasks()
  } catch {
    Message.error('重试失败')
  }
}

onMounted(() => {
  loadTasks()
})
</script>

<template>
  <div>
    <PageHeader title="发布管理" subtitle="管理内容发布任务">
      <template #actions>
        <a-button type="primary" @click="showCreateModal = true">
          <template #icon><IconPlus /></template>
          新建发布
        </a-button>
      </template>
    </PageHeader>

    <div class="mb-5">
      <SegmentedControl v-model="activeTab" :options="tabs" />
    </div>

    <a-spin :loading="loading" style="width: 100%; display: block">
      <a-empty v-if="!loading && filteredTasks.length === 0" description="暂无发布任务" />
      <a-table
        v-else
        :columns="columns"
        :data="filteredTasks"
        :bordered="false"
        :hoverable="true"
        :pagination="false"
      >
        <template #platform="{ record }">
          <PlatformIcon :platform="record.platform" size="sm" />
        </template>
        <template #status="{ record }">
          <StatusBadge :status="record.status" />
        </template>
        <template #actions="{ record }">
          <a-space :size="4">
            <a-button type="text" size="small" title="查看">
              <template #icon><IconEye /></template>
            </a-button>
            <a-button
              v-if="record.status === 'failed'"
              type="text"
              status="warning"
              size="small"
              title="重试"
              @click="retryTask(record.id)"
            >
              <template #icon><IconRefresh /></template>
            </a-button>
            <a-button type="text" status="danger" size="small" title="删除">
              <template #icon><IconDelete /></template>
            </a-button>
          </a-space>
        </template>
      </a-table>
    </a-spin>

    <Modal v-model:visible="showCreateModal" title="新建发布" width="560px">
      <a-form :model="formData" layout="vertical">
        <a-form-item label="选择内容">
          <a-select v-model="formData.contentId" placeholder="请选择要发布的内容" allow-clear>
            <a-option v-for="content in contents" :key="content.id" :value="content.id">
              {{ content.title }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="选择账号">
          <a-select v-model="formData.accountId" placeholder="请选择目标账号" allow-clear>
            <a-option v-for="account in accounts" :key="account.id" :value="account.id">
              {{ account.nickname }} ({{ platformChoices[account.platform] || account.platform }})
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="发布时间">
          <a-date-picker
            v-model="formData.scheduledAt"
            show-time
            format="YYYY-MM-DD HH:mm"
            style="width: 100%"
          />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showCreateModal = false">取消</a-button>
          <a-button type="primary" :loading="submitting" @click="createTask">确认发布</a-button>
        </a-space>
      </template>
    </Modal>
  </div>
</template>
