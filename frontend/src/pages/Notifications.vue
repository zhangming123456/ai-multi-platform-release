<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconCheck, IconNotification, IconRefresh } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { useNotificationStore } from '@/stores/notification'
import { formatRelativeTime, formatDateTimeSec } from '@/utils/time'

const router = useRouter()
const notificationStore = useNotificationStore()

const filter = ref<'all' | 'unread'>('all')
const markingAllRead = ref(false)
const loading = ref(false)

const filteredNotifications = computed(() => {
  if (filter.value === 'unread') {
    return notificationStore.notifications.filter((n) => !n.is_read)
  }
  return notificationStore.notifications
})

const getNotificationIcon = (type: string) => {
  switch (type) {
    case 'review_submit':
      return '📝'
    case 'review_approved':
      return '✅'
    case 'review_rejected':
      return '❌'
    case 'role_updated':
      return '👤'
    case 'role_permissions_updated':
      return '🔐'
    default:
      return '🔔'
  }
}

const getNotificationTitle = (type: string) => {
  switch (type) {
    case 'review_submit':
      return '审核提交'
    case 'review_approved':
      return '审核通过'
    case 'review_rejected':
      return '审核驳回'
    case 'role_updated':
      return '角色更新'
    case 'role_permissions_updated':
      return '权限变更'
    default:
      return '通知'
  }
}

async function handleNotificationClick(id: string, relatedId: string | null, type: string) {
  try {
    await notificationStore.markAsRead(id)
    if (relatedId) {
      if (type === 'role_updated' || type === 'role_permissions_updated') {
        router.push('/profile')
      } else {
        router.push(`/review/${relatedId}`)
      }
    }
  } catch {
    Message.error('操作失败')
  }
}

async function handleMarkAllRead() {
  markingAllRead.value = true
  try {
    await notificationStore.markAllAsRead()
    Message.success('已全部标记为已读')
  } catch {
    Message.error('操作失败')
  } finally {
    markingAllRead.value = false
  }
}

async function refresh() {
  loading.value = true
  try {
    await notificationStore.fetchNotifications()
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  notificationStore.fetchNotifications()
})
</script>

<template>
  <div class="page-main">
    <PageHeader title="通知中心" subtitle="查看和管理你的通知消息">
      <template #actions>
        <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="refresh">
          <template #icon><IconRefresh :size="13" /></template>
          刷新
        </a-button>
        <a-button
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          :loading="markingAllRead"
          :disabled="notificationStore.unreadCount === 0"
          @click="handleMarkAllRead"
        >
          <template #icon><IconCheck :size="13" /></template>
          全部已读
        </a-button>
      </template>
    </PageHeader>

    <!-- 筛选 -->
    <div class="view-mode-toggle mb-5">
      <button class="toggle-btn" :class="{ active: filter === 'all' }" @click="filter = 'all'">
        全部 ({{ notificationStore.notifications.length }})
      </button>
      <button
        class="toggle-btn"
        :class="{ active: filter === 'unread' }"
        @click="filter = 'unread'"
      >
        未读 ({{ notificationStore.unreadCount }})
      </button>
    </div>

    <a-spin :loading="loading" tip="加载中..." class="w-full">
      <!-- 通知列表 -->
      <div v-if="filteredNotifications.length > 0" class="notification-list">
        <div
          v-for="notification in filteredNotifications"
          :key="notification.id"
          :class="['notification-item', { unread: !notification.is_read }]"
          @click="
            handleNotificationClick(notification.id, notification.related_id, notification.type)
          "
        >
          <div class="notification-icon">{{ getNotificationIcon(notification.type) }}</div>
          <div class="notification-content">
            <div class="notification-header">
              <span class="notification-title-text">{{
                getNotificationTitle(notification.type)
              }}</span>
              <span class="notification-time">{{
                formatRelativeTime(notification.created_at)
              }}</span>
            </div>
            <div class="notification-message">{{ notification.content }}</div>
            <div class="notification-time-detail">
              {{ formatDateTimeSec(notification.created_at) }}
            </div>
          </div>
          <div v-if="!notification.is_read" class="unread-dot"></div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <IconNotification :size="48" class="text-[#c9c9cc] mb-4" />
        <p class="text-[14px] text-[#86868B]">
          {{ filter === 'unread' ? '没有未读通知' : '暂无通知' }}
        </p>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.view-mode-toggle {
  display: inline-flex;
  align-items: center;
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 999px;
  padding: 3px;
  gap: 2px;
}

.toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  font-weight: 500;
  color: #86868b;
  background: transparent;
  border: none;
  border-radius: 999px;
  padding: 5px 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.toggle-btn:hover {
  color: #1d1d1f;
}

.toggle-btn.active {
  background: #f2f4f8;
  color: #1d1d1f;
}

.notification-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.notification-item {
  display: flex;
  gap: 14px;
  padding: 16px 20px;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 14px;
  transition: all 0.2s ease;
  position: relative;
}

.notification-item:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 24px -12px rgba(0, 0, 0, 0.1);
  border-color: rgba(0, 0, 0, 0.08);
}

.notification-item.unread {
  background: rgba(0, 122, 255, 0.03);
  border-color: rgba(0, 122, 255, 0.1);
}

.notification-icon {
  font-size: 24px;
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 12px;
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.notification-title-text {
  font-size: 14px;
  font-weight: 600;
  color: #1d1d1f;
}

.notification-time {
  font-size: 12px;
  color: #aeaeaf;
  flex-shrink: 0;
}

.notification-message {
  font-size: 13px;
  color: #636366;
  line-height: 1.5;
  margin-bottom: 4px;
}

.notification-time-detail {
  font-size: 11px;
  color: #c9c9cc;
}

.unread-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #007aff;
  flex-shrink: 0;
  margin-top: 6px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px 20px;
}
</style>
