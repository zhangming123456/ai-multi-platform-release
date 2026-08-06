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

    <!-- 类型分组 Tab -->
    <div class="type-tabs-wrapper mb-4">
      <div class="type-tabs">
        <button class="type-tab" :class="{ active: activeType === '' }" @click="switchType('')">
          <span class="tab-label">全部</span>
          <span class="tab-count">{{ notificationStore.totalCount }}</span>
        </button>
        <button
          v-for="tab in typeTabs"
          :key="tab.key"
          class="type-tab"
          :class="{ active: activeType === tab.key }"
          @click="switchType(tab.key)"
        >
          <span class="tab-label">{{ tab.label }}</span>
          <span class="tab-count">{{ tab.count }}</span>
        </button>
      </div>
    </div>

    <!-- 已读/未读 切换 -->
    <div class="view-mode-toggle mb-5">
      <button class="toggle-btn" :class="{ active: filter === 'all' }" @click="filter = 'all'">
        全部 ({{ filteredNotifications.length }})
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
              <div class="flex items-center gap-2">
                <span class="notification-title-text">{{
                  getNotificationTitle(notification.type)
                }}</span>
                <a-tag
                  size="small"
                  :color="getTypeColor(notification.type)"
                  class="!m-0 !leading-none"
                >
                  {{ getNotificationTitle(notification.type) }}
                </a-tag>
              </div>
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

      <div v-else class="empty-state">
        <IconNotification :size="48" class="text-[#c9c9cc] mb-4" />
        <p class="text-[14px] text-[#86868B]">
          {{ filter === 'unread' ? '没有未读通知' : '暂无此类型通知' }}
        </p>
      </div>
    </a-spin>
  </div>
</template>

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

const activeType = ref('')
const filter = ref<'all' | 'unread'>('all')
const markingAllRead = ref(false)
const loading = ref(false)

const TYPE_LABELS: Record<string, string> = {
  review_submit: '审核提交',
  review_approved: '审核通过',
  review_rejected: '审核驳回',
  role_updated: '角色更新',
  role_permissions_updated: '权限变更',
}

const TYPE_ORDER = [
  'review_submit',
  'review_approved',
  'review_rejected',
  'role_updated',
  'role_permissions_updated',
]

const typeTabs = computed(() => {
  const counts = notificationStore.typeCounts || {}
  return TYPE_ORDER.filter((k) => (counts[k] || 0) > 0).map((k) => ({
    key: k,
    label: TYPE_LABELS[k] || k,
    count: counts[k] || 0,
  }))
})

const filteredNotifications = computed(() => {
  let list = notificationStore.notifications
  if (filter.value === 'unread') {
    list = list.filter((n) => !n.is_read)
  }
  return list
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
  return TYPE_LABELS[type] || '通知'
}

const getTypeColor = (type: string) => {
  switch (type) {
    case 'review_submit':
      return 'blue'
    case 'review_approved':
      return 'green'
    case 'review_rejected':
      return 'red'
    case 'role_updated':
      return 'blue'
    case 'role_permissions_updated':
      return 'orangered'
    default:
      return 'gray'
  }
}

async function switchType(type: string) {
  activeType.value = type
  loading.value = true
  try {
    await notificationStore.fetchNotifications(type || undefined)
  } finally {
    loading.value = false
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
    await notificationStore.fetchTypeCounts()
    await notificationStore.fetchNotifications(activeType.value || undefined)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await notificationStore.fetchTypeCounts()
    await notificationStore.fetchNotifications()
  } finally {
    loading.value = false
  }
})
</script>

<style scoped lang="scss">
.type-tabs-wrapper {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;

  &::-webkit-scrollbar {
    display: none;
  }
}

.type-tabs {
  display: inline-flex;
  align-items: center;
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 999px;
  padding: 3px;
  gap: 2px;
  white-space: nowrap;
}

.type-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
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

.type-tab:hover {
  color: #1d1d1f;
}

.type-tab.active {
  background: #f2f4f8;
  color: #1d1d1f;
}

.tab-count {
  font-size: 11px;
  font-weight: 600;
  color: #aeaeb2;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 8px;
  padding: 1px 7px;
  min-width: 20px;
  text-align: center;
}

.type-tab.active .tab-count {
  background: rgba(0, 122, 255, 0.1);
  color: #007aff;
}

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
