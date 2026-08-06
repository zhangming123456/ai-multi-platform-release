# 通知模块技术文档

## 1. 概述

通知模块负责系统内的消息通知推送与管理，支持以下能力：

- **实时推送**：基于 SSE（Server-Sent Events）+ 内存广播实现毫秒级通知推送
- **分级降级**：SSE → 长轮询 → 静默，根据网络状况自动切换
- **多渠道通知**：页面内 Arco 提醒框 + 浏览器 Notification API + PWA Badging API
- **权限联动**：角色/权限变更通知触发用户数据与权限即时刷新

---

## 2. 架构总览

```
┌─────────────────────────────────────────────────────────────┐
│                    通知架构                                  │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐ │
│  │ 角色变更  │  │ 权限变更  │  │ 内容审核  │  │ SQL 变更  │ │
│  │ rbac_users│  │ rbac_roles│  │ reviews  │  │ db_changes│ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬─────┘ │
│       │              │              │               │        │
│       └──────────────┴──────────────┴───────────────┘        │
│                          │                                   │
│                          ▼                                   │
│    ┌──────────────────────────────────────────┐             │
│    │         NotificationBroadcaster          │             │
│    │   subscribe() / broadcast() / unsubscribe│             │
│    │          asyncio.Queue per user          │             │
│    └──────────────┬───────────────────────────┘             │
│                   │                                          │
│          ┌────────┴────────┐                                 │
│          ▼                 ▼                                 │
│  ┌──────────────┐  ┌──────────────┐                          │
│  │ SSE /stream  │  │  DB 轮询      │                          │
│  │ 广播通道      │  │ 兜底通道      │                          │
│  └──────┬───────┘  └──────┬───────┘                          │
│         │                 │                                  │
│         └────────┬────────┘                                  │
│                  ▼                                           │
│  ┌──────────────────────────────────────────┐               │
│  │           前端 useNotificationRealtime    │               │
│  │  ┌──────────┐  ┌──────────┐  ┌────────┐ │               │
│  │  │ Notification│ │  Arco   │  │Badging │ │               │
│  │  │    API    │  │提醒框    │  │  API   │ │               │
│  │  └──────────┘  └──────────┘  └────────┘ │               │
│  └──────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 通知类型

### 3.1 类型枚举

| 类型值                     | 说明     | 触发场景                    | 提醒框颜色   |
| -------------------------- | -------- | --------------------------- | ------------ |
| `review_submit`            | 审核提交 | 用户提交内容/SQL 变更待审核 | 蓝 (info)    |
| `review_approved`          | 审核通过 | 内容/SQL 变更通过审核       | 绿 (success) |
| `review_rejected`          | 审核驳回 | 内容/SQL 变更被驳回         | 红 (error)   |
| `role_updated`             | 角色更新 | 用户角色发生变化            | 蓝 (info)    |
| `role_permissions_updated` | 权限变更 | 角色被分配/移除/替换        | 橙 (warning) |

### 3.2 通知消息模板

#### 角色变更 `role_permissions_updated`

| 场景     | 格式                                                     |
| -------- | -------------------------------------------------------- |
| 新增角色 | `你的角色已被 {操作人} 添加「{角色名}」`                 |
| 移除角色 | `你的角色已被 {操作人} 移除「{角色名}」`                 |
| 替换角色 | `你的角色已被 {操作人} 从「{旧角色}」更新为「{新角色}」` |

#### 个人信息变更 `role_updated`

| 场景       | 格式                                                                                   |
| ---------- | -------------------------------------------------------------------------------------- |
| 单字段修改 | `你的{字段}已被 {操作人} 修改：从「{旧值}」更新为「{新值}」`                           |
| 单字段新增 | `你的{字段}已被 {操作人} 修改：添加「{新值}」`                                         |
| 单字段清除 | `你的{字段}已被 {操作人} 修改：移除「{旧值}」`                                         |
| 多字段修改 | `你的个人信息已被 {操作人} 修改：昵称 从「旧」更新为「新」；邮箱 从「旧」更新为「新」` |

#### 审核相关

| 类型              | 格式                                       |
| ----------------- | ------------------------------------------ |
| `review_submit`   | `{操作人} 提交了内容「{标题}」待审核`      |
| `review_approved` | `您的内容「{标题}」已通过审核`             |
| `review_rejected` | `您的内容「{标题}」已被驳回，原因：{原因}` |

### 3.3 通知消息字典

管理通知消息中字段名和枚举值的显示名称，支持通过管理页面动态配置，无需修改代码。

**数据模型：** `notification_dict` 表

| 字段         | 说明                                                           |
| ------------ | -------------------------------------------------------------- |
| `category`   | `field`（字段名）或 `enum`（枚举值）                           |
| `group_key`  | 分组标识，如 `user_fields`、`notification_type`、`user_status` |
| `dict_key`   | 原始 Key，如 `nickname`、`review_submit`、`active`             |
| `dict_value` | 显示名称，如 `昵称`、`审核提交`、`启用`                        |
| `is_active`  | 是否启用，禁用后回退使用原始 key                               |

**管理系统：** 侧边栏 → 系统管理 → 通知消息字典

- 权限：`notification:enum:read` / `notification:enum:write`

**API 端点：**

| 方法     | 路径                                           | 说明               |
| -------- | ---------------------------------------------- | ------------------ |
| `GET`    | `/api/notification-dict/groups`                | 按分组获取全部字典 |
| `GET`    | `/api/notification-dict/?category=&group_key=` | 筛选查询           |
| `POST`   | `/api/notification-dict/`                      | 新增条目           |
| `PUT`    | `/api/notification-dict/{id}`                  | 编辑条目           |
| `DELETE` | `/api/notification-dict/{id}`                  | 删除条目           |

**消息解析流程：**

```
通知创建点 → resolve_field("user_fields", "nickname") → 查询 DB → 返回 "昵称"
```

- [notification_dict_service.py](file:///Users/zhangming/Desktop/project/ai-multi-platform-release/backend/app/services/notification_dict_service.py) 提供 `resolve_field()` 和 `load_dict_map()` 两个工具函数
- 数据库查不到的 key 会原样返回，确保不阻断消息生成
- 种子数据预置了 `user_fields`、`notification_type`、`user_status` 三组字典

---

## 4. 后端实现

### 4.1 文件清单

| 文件                                               | 说明                                                    |
| -------------------------------------------------- | ------------------------------------------------------- |
| `backend/app/models/notification.py`               | 数据模型 + 枚举定义                                     |
| `backend/app/schemas/notification.py`              | Pydantic 请求/响应 Schema                               |
| `backend/app/routers/notifications.py`             | RESTful API + SSE 端点                                  |
| `backend/app/services/notification_broadcaster.py` | 内存广播器（全局单例）                                  |
| `backend/app/routers/rbac_users.py`                | `_notify_user_role_change` + `_notify_user_info_change` |
| `backend/app/routers/rbac_roles.py`                | `_notify_role_users`                                    |
| `backend/app/routers/reviews.py`                   | 内容审核通知（submit/approve/reject）                   |
| `backend/app/routers/db_changes.py`                | SQL 变更通知（submit/approve/reject）                   |

### 4.2 数据模型

```python
# backend/app/models/notification.py

class Notification(Base):
    __tablename__ = "notifications"

    id          Mapped[str]          # UUID 主键
    user_id     Mapped[str]          # FK → users.id，通知接收者
    type        Mapped[NotificationType]  # 枚举：review_submit / review_approved / ...
    title       Mapped[str]          # 通知标题（≤200 字符）
    content     Mapped[Optional[str]]# 通知正文（TEXT）
    related_id  Mapped[Optional[str]]# 关联业务 ID
    is_read     Mapped[bool]         # 已读标记，默认 False
    created_at  Mapped[datetime]     # 创建时间
```

### 4.3 API 端点

| 方法   | 路径                               | 说明                     |
| ------ | ---------------------------------- | ------------------------ |
| `GET`  | `/api/notifications/`              | 分页获取当前用户通知列表 |
| `GET`  | `/api/notifications/unread-count`  | 获取未读通知数量         |
| `POST` | `/api/notifications/{id}/read`     | 标记单条通知为已读       |
| `POST` | `/api/notifications/read-all`      | 标记所有通知为已读       |
| `GET`  | `/api/notifications/stream?token=` | SSE 实时推送端点         |

### 4.4 SSE 实时推送机制

#### 连接流程

```
Client ──GET /stream?token=xxx──▶ Server
                                   │
                                   ├─ decode_access_token(token)
                                   ├─ broadcaster.subscribe(user_id)  ── 注册 asyncio.Queue
                                   ├─ yield "event: connected"       ── 连接确认
                                   │
          ┌──asyncio.wait(FIRST_COMPLETED)──┐
          │                                  │
     task_broadcast                       task_poll
     (sub_queue.get())                   (_poll_db)
          │                                  │
    通知创建处 broadcast()               DB 轮询兜底
    立即放入 Queue                       每轮检查新通知
          │                                  │
          └──────── 谁先返回用谁 ──────────────┘
                   │
                   ├─ yield "event: notification"
                   └─ 循环……

Client ──断开──▶ Server
                   └─ broadcaster.unsubscribe(user_id, queue)
```

#### 关键设计

- **双通道竞速**：`asyncio.wait` + `FIRST_COMPLETED` → 广播通道与 DB 轮询通道竞速，谁先返回就用谁
- **广播通道**：通知创建处通过 `broadcaster.broadcast()` 将通知 dict 放入对应用户的 `asyncio.Queue`，SSE 协程立即获取并 yield
- **DB 轮询通道**：`_poll_db()` 查询数据库，兜底非本进程创建的通知（多实例部署场景）
- **去重**：`known_ids` 集合跟踪已推送的通知 ID，避免重复推送

### 4.5 NotificationBroadcaster 广播器

```python
# backend/app/services/notification_broadcaster.py

class NotificationBroadcaster:
    _queues: dict[str, list[asyncio.Queue[dict]]]  # user_id → [Queue, ...]

    subscribe(user_id)    → asyncio.Queue[dict]    # SSE 连接时注册
    unsubscribe(user_id, queue) → None             # SSE 断开时清理
    broadcast(user_id, notification_dict) → None   # 通知创建时推送
```

**特性：**

- 全局单例（`broadcaster` 实例）
- 每用户可持有多个 Queue（支持多标签页同时连接）
- 队列满时自动丢弃最旧消息，放入最新消息
- 自动清理已死亡的队列

### 4.6 通知创建点

所有通知创建点均遵循以下流程：

```python
# 1. 创建通知对象
notification = Notification(user_id=..., type=..., title=..., ...)
db.add(notification)

# 2. flush 获取 ID
await db.flush()

# 3. 广播即时推送
await broadcaster.broadcast(user_id, _notification_dict(notification))

# 4. 提交事务
await db.commit()
```

**通知创建点汇总：**

| 文件            | 函数/位置                  | 通知类型                   | 接收者           |
| --------------- | -------------------------- | -------------------------- | ---------------- |
| `rbac_users.py` | `_notify_user_role_change` | `role_permissions_updated` | 被修改角色的用户 |
| `rbac_users.py` | `_notify_user_info_change` | `role_updated`             | 被修改信息的用户 |
| `rbac_roles.py` | `_notify_role_users`       | `role_permissions_updated` | 角色下所有用户   |
| `reviews.py`    | 提交审核                   | `review_submit`            | 审核员           |
| `reviews.py`    | 审核通过                   | `review_approved`          | 内容作者         |
| `reviews.py`    | 审核驳回                   | `review_rejected`          | 内容作者         |
| `db_changes.py` | SQL 变更提交               | `review_submit`            | 审核员           |
| `db_changes.py` | SQL 变更执行/进度          | `review_approved`          | 提交者           |
| `db_changes.py` | SQL 变更驳回               | `review_rejected`          | 提交者           |

---

## 5. 前端实现

### 5.1 文件清单

| 文件                                                  | 说明                       |
| ----------------------------------------------------- | -------------------------- |
| `frontend/src/stores/notification.ts`                 | Pinia 状态管理             |
| `frontend/src/composables/useNotificationRealtime.ts` | 实时通知核心逻辑           |
| `frontend/src/components/NotificationBell.vue`        | 顶部导航栏通知铃铛         |
| `frontend/src/pages/Notifications.vue`                | 通知中心页面               |
| `frontend/src/components/layout/AppLayout.vue`        | 布局组件（初始化实时通知） |

### 5.2 Pinia Store

```typescript
// frontend/src/stores/notification.ts

useNotificationStore:
  state:
    notifications: Notification[]   // 通知列表
    unreadCount: number             // 未读数量

  getters:
    unreadNotifications             // 未读通知列表

  actions:
    fetchNotifications()            // GET /api/notifications/
    fetchUnreadCount()              // GET /api/notifications/unread-count
    markAsRead(id)                  // POST /api/notifications/{id}/read
    markAllAsRead()                 // POST /api/notifications/read-all
```

### 5.3 useNotificationRealtime Composable

**核心职责：** 管理 SSE 连接生命周期，协调多层通知通道，处理通知响应。

#### 通知通道递推逻辑

```
┌──────────────────────────────────────────────────┐
│             handleNewNotifications               │
│                                                  │
│  1. showBrowserNotification(item) × N            │
│     ├─ permissionGranted && 后台 → 弹系统通知    │
│     └─ 否则 → 跳过                               │
│                                                  │
│  2. showAppNotificationBatch(items)              │
│     ├─ 页面不可见 → 跳过（系统通知接管）          │
│     ├─ 1 条 → 单条 Arco 提醒框（带类型颜色）      │
│     └─ N 条 → 合并 "N 条新通知"                   │
│                                                  │
│  3. updateBadge()                                │
│     └─ PWA 图标角标 + favicon 角标               │
│                                                  │
│  4. hasPermissionChange? → 立即同步权限+用户数据   │
│     ├─ permissionStore.loadPermissions()         │
│     └─ userStore.fetchUserInfo()                 │
└──────────────────────────────────────────────────┘
```

#### SSE 连接管理

| 配置                    | 值       | 说明                                        |
| ----------------------- | -------- | ------------------------------------------- |
| `SSE_CONNECT_TIMEOUT`   | 8s       | 连接超时，未收到 `connected` 事件则降级轮询 |
| `SSE_IDLE_TIMEOUT`      | 45s      | 空闲超时，仅页面不可见时触发降级            |
| `POLL_INTERVAL`         | 15s      | 轮询间隔                                    |
| `POLL_UPGRADE_INTERVAL` | 5 个周期 | 每 75s 尝试从轮询升级回 SSE                 |

#### SSE 空闲降级规则

```
SSE 空闲 45s 到期
  │
  ├─ isPageVisible() === true
  │    └─ resetIdleTimer()  // 用户活跃，不倒计时，不降级
  │
  └─ isPageVisible() === false
       └─ disconnectSSE() → startPolling()  // 降级
```

#### 轮询升级规则

```
每 POLL_UPGRADE_INTERVAL(5) 个轮询周期
  │
  ├─ isPageVisible() === true
  │    └─ stopPolling() → connectSSE()  // 尝试升级回 SSE
  │
  └─ isPageVisible() === false
       └─ 继续轮询（后台不需要 SSE）
```

#### Notification API 权限监听

通过 `navigator.permissions.query({ name: 'notifications' })` 注册 `onchange` 回调，实时感知权限变化并在控制台打印日志。

#### Badging API

- 检测 `navigator.setAppBadge` / `navigator.setClientBadge`
- 新通知到达 → `updateBadge()` 设置 PWA 角标 = 未读数
- 页面切回前台 → `visibilitychange` → `clearBadge()` 清除角标
- 组件销毁 → 移除监听 + 清除角标

### 5.4 NotificationBell 组件

顶部导航栏的通知铃铛图标，显示未读数量 Badge，点击跳转通知中心。

### 5.5 Notifications 页面

通知中心页面，功能：

- **筛选**：全部 / 未读，带计数显示
- **通知列表**：按时间倒序排列，未读项高亮显示
- **点击交互**：点击通知 → 标记已读 → 根据类型跳转关联页面
  - `role_updated` / `role_permissions_updated` → 跳转 `/profile`
  - 审核类（有 `related_id`）→ 跳转 `/review/{related_id}`
- **全部已读**：一键标记所有通知为已读
- **刷新**：手动拉取最新数据
- **空状态**：无通知时显示空状态提示

### 5.6 初始化流程

```
AppLayout.vue onMounted()
  │
  ├─ userStore.fetchUserInfo()     // 获取用户信息
  │
  └─ realtime.start()              // 启动实时通知
       │
       ├─ initBadging()            // 初始化 PWA 角标
       ├─ requestPermission()      // 请求 Notification 权限
       │    ├─ watchPermissionChanges()  // 注册权限变化监听
       │    └─ 打印权限状态日志
       │
       └─ connectSSE()             // 建立 SSE 连接
            ├─ 连接成功 → connected 事件
            ├─ 收到通知 → handleNewNotifications
            └─ 连接失败 → startPolling
```

AppLayout.vue `onUnmounted()` → `realtime.destroy()`

- 断开 SSE + 停止轮询 + 移除监听 + 清除 Badge

---

## 6. 完整数据流

### 场景：管理员修改用户角色

```
 管理后台操作 "更新用户角色"
        │
        ▼
 PUT /api/users/{id}
        │
        ▼
 _notify_user_role_change()
        │
        ├─ 比较新旧角色集合 → 无变化则 return
        │
        ├─ db.add(Notification)           ← 写入数据库
        ├─ await db.flush()               ← 获取 ID
        ├─ await broadcaster.broadcast()  ← 放入 Queue
        └─ await db.commit()              ← 提交事务
               │
     ┌─────────┴─────────┐
     ▼                   ▼
  数据库写入            Queue 即时推送
     │                   │
     │         SSE /stream 协程
     │            │
     │         asyncio.wait 竞速
     │            │
     │         yield "event: notification"
     │            │
     │        前端 EventSource
     │            │
     │         handleNewNotifications
     │            │
     │    ┌───────┼────────┬──────────┐
     │    ▼       ▼         ▼          ▼
     │  系统通知  Arco提醒框  Badge角标  权限同步
     │    │       │         │          │
     │    ▼       ▼         ▼          ▼
     │  后台弹窗  页面提醒   PWA角标    loadPermissions()
     │                                  fetchUserInfo()
     │
     ▼
  轮询兜底（SSE 断开时）
  _poll_db() 每 10s 检查数据库新通知
```

---

## 7. 控制台日志

所有日志以 `[通知实时]` 为前缀，可在浏览器 DevTools 控制台查看：

| 日志                                                                           | 含义                                 |
| ------------------------------------------------------------------------------ | ------------------------------------ |
| `Badging API 可用 ✅`                                                          | PWA 角标功能可用                     |
| `Notification API 权限已授予 ✅`                                               | 浏览器通知权限已开启                 |
| `Notification API 权限被拒绝`                                                  | 浏览器通知权限被拒绝                 |
| `SSE 连接成功 ✅`                                                              | SSE 长连接建立 + 收到 connected 事件 |
| `SSE 推送 N 条新通知 ["标题1", "标题2"]`                                       | 通过 SSE 收到新通知                  |
| `SSE 连接超时（8s），降级为轮询`                                               | SSE 超时降级                         |
| `SSE 长时间空闲且页面不可见（45s），降级为轮询`                                | 空闲超时降级                         |
| `启动轮询模式（间隔 15s）`                                                     | 降级到轮询                           |
| `尝试升级为 SSE 模式`                                                          | 从轮询尝试升级回 SSE                 |
| `页面可见，跳过系统弹窗（"XXX"）`                                              | 前台不弹系统通知                     |
| `系统弹窗已弹出: { notification.title, notification.body, notification.data }` | 系统通知弹窗成功                     |
| `检测到角色/权限变更通知，立即同步权限与用户数据`                              | 触发权限+用户数据刷新                |
| `Badging API 已更新: 3`                                                        | PWA 角标更新                         |
| `Badging API 已清除`                                                           | PWA 角标清除                         |
| `Notification API 权限已实时变更为：已授予 ✅`                                 | 权限实时变化                         |
