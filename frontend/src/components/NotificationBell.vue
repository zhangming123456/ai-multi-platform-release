<template>
  <a-badge :count="notificationStore.unreadCount" :dot="false">
    <a-button type="text" size="small" class="notification-btn" @click="goToNotifications">
      <template #icon>
        <IconNotification :size="18" />
      </template>
    </a-button>
  </a-badge>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useNotificationStore } from '@/stores/notification'
import { IconNotification } from '@arco-design/web-vue/es/icon'

const notificationStore = useNotificationStore()
const router = useRouter()

onMounted(() => {
  notificationStore.fetchNotifications()
  notificationStore.fetchUnreadCount()
})

function goToNotifications() {
  router.push('/notifications')
}
</script>

<style scoped lang="scss">
.notification-btn {
  color: var(--color-text-2);
  transition: color 0.2s;

  &:hover {
    color: var(--color-text-1);
  }
}
</style>
