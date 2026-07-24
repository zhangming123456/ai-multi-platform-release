<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useNotificationStore } from '@/stores/notification'
import { formatDateTime } from '@/utils/time'
import { IconNotification } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

const notificationStore = useNotificationStore()
const router = useRouter()
const visible = ref(false)

onMounted(() => {
  notificationStore.fetchNotifications()
  notificationStore.fetchUnreadCount()
})

const handleNotificationClick = async (id: number, relatedId: number | null) => {
  await notificationStore.markAsRead(id)
  if (relatedId) {
    router.push(`/review/${relatedId}`)
  }
  visible.value = false
}

const handleMarkAllRead = async () => {
  await notificationStore.markAllAsRead()
}

const getNotificationIcon = (type: string) => {
  switch (type) {
    case 'review_submit':
      return '📝'
    case 'review_approved':
      return '✅'
    case 'review_rejected':
      return '❌'
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
    default:
      return '通知'
  }
}
</script>

<template>
  <a-popover
    v-model:popupVisible="visible"
    trigger="click"
    position="br"
    :content-style="{ width: '360px', padding: '0' }"
  >
    <a-badge :count="notificationStore.unreadCount" :dot="false">
      <a-button type="text" size="small" class="notification-btn">
        <template #icon>
          <IconNotification :size="18" />
        </template>
      </a-button>
    </a-badge>
    
    <template #content>
      <div class="notification-popover">
        <div class="notification-header">
          <span class="notification-title">通知</span>
          <a-button
            v-if="notificationStore.unreadCount > 0"
            type="text"
            size="mini"
            @click="handleMarkAllRead"
          >
            全部已读
          </a-button>
        </div>
        
        <div class="notification-list">
          <div
            v-for="notification in notificationStore.notifications"
            :key="notification.id"
            :class="['notification-item', { unread: !notification.is_read }]"
            @click="handleNotificationClick(notification.id, notification.related_id)"
          >
            <div class="notification-icon">
              {{ getNotificationIcon(notification.type) }}
            </div>
            <div class="notification-content">
              <div class="notification-title-text">
                {{ getNotificationTitle(notification.type) }}
              </div>
              <div class="notification-message">
                {{ notification.content }}
              </div>
              <div class="notification-time">
                {{ formatDateTime(notification.created_at) }}
              </div>
            </div>
          </div>
          
          <div v-if="notificationStore.notifications.length === 0" class="notification-empty">
            暂无通知
          </div>
        </div>
      </div>
    </template>
  </a-popover>
</template>

<style scoped lang="scss">
.notification-btn {
  color: var(--color-text-2);
  transition: color 0.2s;
  
  &:hover {
    color: var(--color-text-1);
  }
}

.notification-popover {
  max-height: 480px;
  display: flex;
  flex-direction: column;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--color-border-2);
  
  .notification-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--color-text-1);
  }
}

.notification-list {
  overflow-y: auto;
  max-height: 400px;
}

.notification-item {
  display: flex;
  gap: 12px;
  padding: 16px;
  cursor: pointer;
  transition: background-color 0.2s;
  border-bottom: 1px solid var(--color-border-1);
  
  &:hover {
    background-color: var(--color-fill-1);
  }
  
  &.unread {
    background-color: var(--color-primary-light-1);
    
    &:hover {
      background-color: var(--color-primary-light-2);
    }
  }
}

.notification-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-title-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
  margin-bottom: 4px;
}

.notification-message {
  font-size: 13px;
  color: var(--color-text-2);
  margin-bottom: 4px;
  line-height: 1.5;
}

.notification-time {
  font-size: 12px;
  color: var(--color-text-3);
}

.notification-empty {
  padding: 40px 16px;
  text-align: center;
  color: var(--color-text-3);
  font-size: 14px;
}
</style>
