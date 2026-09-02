import { ref, onUnmounted } from 'vue'
import { Notification as ArcoNotification } from '@arco-design/web-vue'
import { useNotificationStore } from '@/stores/notification'
import { usePermissionStore } from '@/stores/permission'
import { useUserStore } from '@/stores/user'

type BadgeNavigator = Navigator & {
  setAppBadge?: (count: number) => Promise<void>
  clearAppBadge?: () => Promise<void>
  setClientBadge?: (count: number) => Promise<void>
  clearClientBadge?: () => Promise<void>
}

interface NotificationItem {
  id: string
  type: string
  title: string
  content: string | null
  related_id: string | null
  is_read: boolean
  created_at: string
}

const NOTIFICATION_ICONS: Record<string, string> = {
  review_submit: '📝',
  review_approved: '✅',
  review_rejected: '❌',
  role_updated: '👤',
  role_permissions_updated: '🔐',
}

const TYPE_NOTIFICATION_KIND: Record<string, 'info' | 'success' | 'warning' | 'error'> = {
  review_submit: 'info',
  review_approved: 'success',
  review_rejected: 'error',
  role_updated: 'info',
  role_permissions_updated: 'warning',
}

const CONTENT_MAX_LEN = 60

const LOG_PREFIX = '[通知实时]'

const SSE_CONNECT_TIMEOUT = 8000
const SSE_IDLE_TIMEOUT = 45000
const POLL_INTERVAL = 15000
const POLL_UPGRADE_INTERVAL = 5

let globalInstance: ReturnType<typeof createRealtimeInstance> | null = null

function createRealtimeInstance() {
  const permissionGranted = ref(false)
  const badgingSupported = ref(false)
  const isSseActive = ref(false)
  const isPollActive = ref(false)

  let eventSource: EventSource | null = null
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let sseConnectTimeout: ReturnType<typeof setTimeout> | null = null
  let sseIdleTimer: ReturnType<typeof setTimeout> | null = null
  let destroyed = false
  let lastCount = 0
  let pollTick = 0

  const notificationStore = useNotificationStore()
  const permissionStore = usePermissionStore()
  const userStore = useUserStore()

  function isPageVisible(): boolean {
    return document.visibilityState === 'visible'
  }

  function initBadging() {
    if ('setAppBadge' in navigator) {
      badgingSupported.value = true
      console.log(`${LOG_PREFIX} Badging API 可用 ✅`)
    } else if ('setClientBadge' in navigator) {
      badgingSupported.value = true
      console.log(`${LOG_PREFIX} Badging API (experimental) 可用 ✅`)
    } else {
      console.warn(`${LOG_PREFIX} 浏览器不支持 Badging API`)
    }

    document.addEventListener('visibilitychange', onVisibilityChange)
  }

  function onVisibilityChange() {
    if (isPageVisible()) {
      clearBadge()
    }
  }

  async function updateBadge() {
    if (!badgingSupported.value) return

    try {
      const unread = notificationStore.unreadCount
      if (unread > 0) {
        if ('setAppBadge' in navigator) {
          await (navigator as BadgeNavigator).setAppBadge!(unread)
        } else if ('setClientBadge' in navigator) {
          await (navigator as BadgeNavigator).setClientBadge!(unread)
        }
        console.log(`${LOG_PREFIX} Badging API 已更新: ${unread}`)
      }
    } catch (err) {
      console.warn(`${LOG_PREFIX} Badging API 更新失败`, err)
    }
  }

  async function clearBadge() {
    if (!badgingSupported.value) return

    try {
      if ('clearAppBadge' in navigator) {
        await (navigator as BadgeNavigator).clearAppBadge!()
      } else if ('clearClientBadge' in navigator) {
        await (navigator as BadgeNavigator).clearClientBadge!()
      }
      console.log(`${LOG_PREFIX} Badging API 已清除`)
    } catch (err) {
      console.warn(`${LOG_PREFIX} Badging API 清除失败`, err)
    }
  }

  function resetIdleTimer() {
    if (sseIdleTimer) clearTimeout(sseIdleTimer)
    sseIdleTimer = setTimeout(() => {
      if (isPageVisible()) {
        resetIdleTimer()
        return
      }
      console.warn(
        `${LOG_PREFIX} SSE 长时间空闲且页面不可见（${SSE_IDLE_TIMEOUT / 1000}s），降级为轮询`,
      )
      disconnectSSE()
      startPolling()
    }, SSE_IDLE_TIMEOUT)
  }

  async function requestPermission() {
    if (!('Notification' in window)) {
      console.warn(`${LOG_PREFIX} 浏览器不支持 Notification API，将跳过系统弹窗`)
      permissionGranted.value = false
      return
    }

    const currentState = Notification.permission
    if (currentState === 'granted') {
      console.log(`${LOG_PREFIX} Notification API 权限已授予 ✅`)
      permissionGranted.value = true
    } else if (currentState === 'denied') {
      console.warn(`${LOG_PREFIX} Notification API 权限已被拒绝，将跳过系统弹窗`)
      permissionGranted.value = false
    } else {
      const result = await Notification.requestPermission()
      permissionGranted.value = result === 'granted'
      if (result === 'granted') {
        console.log(`${LOG_PREFIX} Notification API 权限已授予 ✅`)
      } else {
        console.warn(`${LOG_PREFIX} Notification API 权限被拒绝，将跳过系统弹窗`)
      }
    }

    watchPermissionChanges()
  }

  function watchPermissionChanges() {
    if (!('permissions' in navigator)) {
      console.warn(`${LOG_PREFIX} 浏览器不支持 Permissions API，无法监听权限实时变化`)
      return
    }

    navigator.permissions
      .query({ name: 'notifications' })
      .then((status) => {
        status.onchange = () => {
          const newState = status.state
          permissionGranted.value = newState === 'granted'
          if (newState === 'granted') {
            console.log(
              `${LOG_PREFIX} Notification API 权限已实时变更为：已授予 ✅（后台时弹系统通知，前台时弹页面提醒框）`,
            )
          } else if (newState === 'denied') {
            console.warn(
              `${LOG_PREFIX} Notification API 权限已实时变更为：已拒绝（回退页面提醒框）`,
            )
          } else {
            console.log(`${LOG_PREFIX} Notification API 权限已实时变更为：${newState}`)
          }
        }
      })
      .catch(() => {
        console.warn(`${LOG_PREFIX} 无法查询 Permissions API，权限变化将不会实时感知`)
      })
  }

  function showBrowserNotification(item: NotificationItem) {
    if (!permissionGranted.value) {
      return
    }

    if (isPageVisible()) {
      console.log(`${LOG_PREFIX} 页面可见，跳过系统弹窗（"${item.title}"）`, {
        title: item.title,
        body: item.content,
        data: { id: item.id, type: item.type, related_id: item.related_id },
      })
      return
    }

    const icon = NOTIFICATION_ICONS[item.type] || '🔔'
    const body = item.content || ''

    try {
      const n = new Notification(`${icon} ${item.title}`, {
        body,
        tag: item.id,
        requireInteraction: false,
      })
      console.log(`${LOG_PREFIX} 系统弹窗已弹出:`, {
        'notification.title': n.title,
        'notification.body': n.body,
        'notification.data': { id: item.id, type: item.type, related_id: item.related_id },
      })
      n.onclick = () => {
        window.focus()
        n.close()
        window.dispatchEvent(
          new CustomEvent('notification-clicked', {
            detail: { id: item.id, type: item.type, related_id: item.related_id },
          }),
        )
      }
    } catch (err) {
      console.warn(`${LOG_PREFIX} 系统弹窗创建失败`, err)
    }
  }

  function showAppNotification(item: NotificationItem) {
    const content = item.content || ''
    const summary =
      content.length > CONTENT_MAX_LEN ? content.slice(0, CONTENT_MAX_LEN) + '…' : content
    const kind = TYPE_NOTIFICATION_KIND[item.type] || 'info'

    ArcoNotification[kind]({
      title: item.title,
      content: summary,
      duration: 5000,
      closable: true,
    })
  }

  function showAppNotificationBatch(items: NotificationItem[]) {
    if (items.length === 0) return

    if (!isPageVisible()) {
      console.log(`${LOG_PREFIX} 页面不可见，跳过页面提醒框（由系统通知/Badging API 接管）`)
      return
    }

    if (items.length === 1) {
      showAppNotification(items[0])
      return
    }

    const titles = items.map((i) => i.title).join('、')
    ArcoNotification.info({
      title: `${items.length} 条新通知`,
      content: titles,
      duration: 6000,
      closable: true,
    })
  }

  function hasPermissionChange(items: NotificationItem[]): boolean {
    return items.some(
      (item) => item.type === 'role_permissions_updated' || item.type === 'role_updated',
    )
  }

  function handleNewNotifications(items: NotificationItem[]) {
    if (!items || items.length === 0) return

    const unreadItems = items.filter((item) => !item.is_read)
    if (unreadItems.length === 0) return

    for (const item of unreadItems) {
      showBrowserNotification(item)
    }
    showAppNotificationBatch(unreadItems)

    notificationStore.fetchNotifications()
    notificationStore.fetchUnreadCount()
    updateBadge()

    if (hasPermissionChange(unreadItems)) {
      console.log(`${LOG_PREFIX} 检测到角色/权限变更通知，立即同步权限与用户数据`)
      permissionStore.loadPermissions()
      userStore.fetchUserInfo()
    }
  }

  function connectSSE() {
    if (destroyed || eventSource) return

    const token = localStorage.getItem('token')
    if (!token) return

    const url = `/api/notifications/stream?token=${encodeURIComponent(token)}`
    const es = new EventSource(url)

    sseConnectTimeout = setTimeout(() => {
      if (!isSseActive.value) {
        console.warn(
          `${LOG_PREFIX} SSE 连接超时（${SSE_CONNECT_TIMEOUT / 1000}s），未收到 connected 事件，降级为轮询`,
        )
        disconnectSSE()
        startPolling()
      }
    }, SSE_CONNECT_TIMEOUT)

    es.addEventListener('connected', () => {
      console.log(`${LOG_PREFIX} SSE 连接成功 ✅`)
      isSseActive.value = true
      isPollActive.value = false
      stopPolling()
      clearSseConnectTimeout()
      resetIdleTimer()
    })

    es.addEventListener('notification', (event) => {
      clearSseConnectTimeout()
      resetIdleTimer()
      try {
        const items: NotificationItem[] = JSON.parse(event.data)
        if (Array.isArray(items) && items.length > 0) {
          console.log(
            `${LOG_PREFIX} SSE 推送 ${items.length} 条新通知`,
            items.map((i) => i.title),
          )
          handleNewNotifications(items)
        }
      } catch {
        // 解析失败，忽略
      }
    })

    es.addEventListener('error', () => {
      console.warn(`${LOG_PREFIX} SSE 连接异常/断开，降级为轮询`)
      clearSseConnectTimeout()
      disconnectSSE()
      if (!destroyed) {
        startPolling()
      }
    })

    es.onopen = () => {
      // HTTP 握手完成，connected 事件将标记业务就绪
    }

    eventSource = es
  }

  function disconnectSSE() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    clearSseConnectTimeout()
    if (sseIdleTimer) {
      clearTimeout(sseIdleTimer)
      sseIdleTimer = null
    }
    isSseActive.value = false
  }

  function clearSseConnectTimeout() {
    if (sseConnectTimeout) {
      clearTimeout(sseConnectTimeout)
      sseConnectTimeout = null
    }
  }

  function startPolling() {
    if (destroyed || pollTimer) return

    console.log(`${LOG_PREFIX} 启动轮询模式（间隔 ${POLL_INTERVAL / 1000}s）`)
    isPollActive.value = true
    lastCount = notificationStore.unreadCount
    pollTick = 0

    pollTimer = setInterval(async () => {
      try {
        pollTick++
        if (pollTick % POLL_UPGRADE_INTERVAL === 0 && isPageVisible()) {
          stopPolling()
          connectSSE()
          return
        }

        const prevCount = lastCount
        await notificationStore.fetchUnreadCount()
        const newCount = notificationStore.unreadCount
        if (newCount > prevCount) {
          await notificationStore.fetchNotifications()
          const unreadItems = notificationStore.notifications.filter((n) => !n.is_read)
          const newItems = unreadItems.slice(0, newCount - prevCount)
          const notificationItems: NotificationItem[] = newItems.map((item) => ({
            id: item.id,
            type: item.type,
            title: item.title,
            content: item.content,
            related_id: item.related_id,
            is_read: item.is_read,
            created_at: item.created_at,
          }))
          for (const ni of notificationItems) {
            showBrowserNotification(ni)
          }
          showAppNotificationBatch(notificationItems)
          updateBadge()

          if (hasPermissionChange(notificationItems)) {
            console.log(`${LOG_PREFIX} 检测到角色/权限变更通知，立即同步权限与用户数据`)
            permissionStore.loadPermissions()
            userStore.fetchUserInfo()
          }
        }
        lastCount = notificationStore.unreadCount
      } catch {
        // 轮询失败，忽略
      }
    }, POLL_INTERVAL)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    isPollActive.value = false
  }

  function destroy() {
    destroyed = true
    disconnectSSE()
    stopPolling()
    document.removeEventListener('visibilitychange', onVisibilityChange)
    clearBadge()
    globalInstance = null
  }

  function start() {
    initBadging()
    requestPermission().then(() => {
      connectSSE()
    })
  }

  return {
    permissionGranted,
    badgingSupported,
    isSseActive,
    isPollActive,
    start,
    destroy,
    requestPermission,
  }
}

export function useNotificationRealtime() {
  if (globalInstance) {
    return globalInstance
  }

  const instance = createRealtimeInstance()

  onUnmounted(() => {
    instance.destroy()
  })

  globalInstance = instance
  return instance
}
