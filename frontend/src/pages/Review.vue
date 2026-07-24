<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import api from '@/utils/api'
import { formatDateTime } from '@/utils/time'
import { useNotificationStore } from '@/stores/notification'

interface ReviewItem {
  id: number
  title: string
  body: string
  platform: string
  status: string
  created_at: string
  user_id: number
  username: string
}

const loading = ref(false)
const reviews = ref<ReviewItem[]>([])
const rejectModalVisible = ref(false)
const currentReviewId = ref<number | null>(null)
const rejectReason = ref('')
const notificationStore = useNotificationStore()

const fetchReviews = async () => {
  loading.value = true
  try {
    const res = await api.get('/reviews/')
    reviews.value = res.data
  } catch (error) {
    Message.error('获取审核列表失败')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const handleApprove = async (id: number) => {
  try {
    await api.post(`/reviews/${id}/approve`)
    Message.success('审核通过')
    await fetchReviews()
    await notificationStore.fetchNotifications()
    await notificationStore.fetchUnreadCount()
  } catch (error) {
    Message.error('审核失败')
    console.error(error)
  }
}

const handleReject = (id: number) => {
  currentReviewId.value = id
  rejectReason.value = ''
  rejectModalVisible.value = true
}

const confirmReject = async () => {
  if (!rejectReason.value.trim()) {
    Message.warning('请输入驳回原因')
    return
  }
  
  try {
    await api.post(`/reviews/${currentReviewId.value}/reject`, {
      reason: rejectReason.value
    })
    Message.success('已驳回')
    rejectModalVisible.value = false
    await fetchReviews()
    await notificationStore.fetchNotifications()
    await notificationStore.fetchUnreadCount()
  } catch (error) {
    Message.error('驳回失败')
    console.error(error)
  }
}

const getPlatformName = (platform: string) => {
  const map: Record<string, string> = {
    'xiaohongshu': '小红书',
    'douyin': '抖音',
    'wechat_video': '微信视频号',
    'wechat_mp': '微信公众号',
    'weibo': '微博',
    'bilibili': 'B站'
  }
  return map[platform] || platform
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'pending_review':
      return 'orange'
    case 'approved':
      return 'green'
    case 'rejected':
      return 'red'
    default:
      return 'gray'
  }
}

const getStatusText = (status: string) => {
  switch (status) {
    case 'pending_review':
      return '待审核'
    case 'approved':
      return '已通过'
    case 'rejected':
      return '已驳回'
    default:
      return status
  }
}

onMounted(() => {
  fetchReviews()
})
</script>

<template>
  <div class="review-page">
    <div class="page-header">
      <h2>审核管理</h2>
      <p class="page-desc">审核用户提交的内容</p>
    </div>

    <a-card :bordered="false" class="review-card">
      <a-table
        :data="reviews"
        :loading="loading"
        :pagination="false"
        :bordered="false"
        row-key="id"
      >
        <template #columns>
          <a-table-column title="标题" data-index="title" :width="200">
            <template #cell="{ record }">
              <div class="title-cell">
                <div class="title-text">{{ record.title }}</div>
                <div class="username-text">提交人：{{ record.username }}</div>
              </div>
            </template>
          </a-table-column>

          <a-table-column title="内容" data-index="body" :width="300">
            <template #cell="{ record }">
              <div class="body-cell">{{ record.body }}</div>
            </template>
          </a-table-column>

          <a-table-column title="平台" data-index="platform" :width="120">
            <template #cell="{ record }">
              <a-tag size="small" color="arcoblue">
                {{ getPlatformName(record.platform) }}
              </a-tag>
            </template>
          </a-table-column>

          <a-table-column title="状态" data-index="status" :width="100">
            <template #cell="{ record }">
              <a-tag size="small" :color="getStatusColor(record.status)">
                {{ getStatusText(record.status) }}
              </a-tag>
            </template>
          </a-table-column>

          <a-table-column title="提交时间" data-index="created_at" :width="160">
            <template #cell="{ record }">
              {{ formatDateTime(record.created_at) }}
            </template>
          </a-table-column>

          <a-table-column title="操作" :width="160" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-button
                  type="primary"
                  size="small"
                  @click="handleApprove(record.id)"
                >
                  通过
                </a-button>
                <a-button
                  type="secondary"
                  status="danger"
                  size="small"
                  @click="handleReject(record.id)"
                >
                  驳回
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>

        <template #empty>
          <a-empty description="暂无待审核内容" />
        </template>
      </a-table>
    </a-card>

    <!-- 驳回原因弹窗 -->
    <a-modal
      v-model:visible="rejectModalVisible"
      title="驳回内容"
      :ok-loading="false"
      @ok="confirmReject"
      @cancel="rejectModalVisible = false"
    >
      <a-form :model="{ reason: rejectReason }" layout="vertical">
        <a-form-item label="驳回原因" required>
          <a-textarea
            v-model="rejectReason"
            placeholder="请输入驳回原因"
            :max-length="500"
            show-word-limit
            :auto-size="{ minRows: 3, maxRows: 6 }"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped lang="scss">
.review-page {
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;

  h2 {
    font-size: 24px;
    font-weight: 600;
    color: #1d1d1f;
    margin: 0 0 8px 0;
  }

  .page-desc {
    font-size: 14px;
    color: #86868b;
    margin: 0;
  }
}

.review-card {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
  border-radius: 16px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.title-cell {
  .title-text {
    font-weight: 500;
    color: #1d1d1f;
    margin-bottom: 4px;
  }

  .username-text {
    font-size: 12px;
    color: #86868b;
  }
}

.body-cell {
  color: #636366;
  font-size: 13px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
