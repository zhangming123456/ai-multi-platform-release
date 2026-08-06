import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/utils/api'

interface Notification {
  id: string
  type: string
  title: string
  content: string
  related_id: string
  is_read: boolean
  created_at: string
}

export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref<Notification[]>([])
  const unreadCount = ref(0)
  const typeCounts = ref<Record<string, number>>({})
  const totalCount = ref(0)

  const unreadNotifications = computed(() => notifications.value.filter((n) => !n.is_read))

  async function fetchNotifications(type?: string) {
    try {
      const params: Record<string, string> = {}
      if (type) params.type = type
      const res = await api.get('/notifications/', { params })
      notifications.value = res.data.items
      await fetchUnreadCount()
    } catch (error) {
      console.error('Failed to fetch notifications:', error)
    }
  }

  async function fetchTypeCounts() {
    try {
      const res = await api.get('/notifications/type-counts')
      typeCounts.value = res.data.counts || {}
      totalCount.value = res.data.total || 0
    } catch (error) {
      console.error('Failed to fetch type counts:', error)
    }
  }

  async function fetchUnreadCount() {
    try {
      const res = await api.get('/notifications/unread-count')
      unreadCount.value = res.data.count
    } catch (error) {
      console.error('Failed to fetch unread count:', error)
    }
  }

  async function markAsRead(id: string) {
    try {
      await api.post(`/notifications/${id}/read`)
      const notification = notifications.value.find((n) => n.id === id)
      if (notification) {
        notification.is_read = true
      }
      await fetchUnreadCount()
    } catch (error) {
      console.error('Failed to mark notification as read:', error)
      throw error
    }
  }

  async function markAllAsRead() {
    try {
      await api.post('/notifications/read-all')
      notifications.value.forEach((n) => (n.is_read = true))
      unreadCount.value = 0
    } catch (error) {
      console.error('Failed to mark all as read:', error)
      throw error
    }
  }

  return {
    notifications,
    unreadCount,
    typeCounts,
    totalCount,
    unreadNotifications,
    fetchNotifications,
    fetchTypeCounts,
    fetchUnreadCount,
    markAsRead,
    markAllAsRead,
  }
})
