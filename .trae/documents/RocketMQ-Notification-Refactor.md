# 消息模块 RocketMQ 改造方案

> **版本**：v0.3（定稿）· **更新**：2026-08-17 · **状态**：已确认，待实施
> 本文档为消息（通知）模块引入 RocketMQ 的改造设计稿。所有关键决策均已由需求方确认（见「第 0 章 决策摘要」），可按「第 9 章 任务拆分」进入实施。

## 0. 决策摘要

| #   | 决策项       | 结论                                                                                   |
| --- | ------------ | -------------------------------------------------------------------------------------- |
| 1   | 部署形态     | **Docker 单机 RocketMQ 5.x**（nameserver + broker + proxy）                            |
| 2   | 客户端选型   | **官方 gRPC 客户端** `apache/rocketmq-clients/golang`（5.x，长期维护）                 |
| 3   | 改造范围     | **通知推送链路 + 发布任务 worker + AI 复核异步化**                                     |
| 4   | 消费者部署   | **独立消费者进程**（新 Go 二进制，与 API 服务分离）                                    |
| 5   | 可靠性       | **标准可靠**：同步发送 + 发送重试 + 消费重试 + 死信队列（DLQ）                         |
| 6   | Topic 组织   | **每业务域一个 Topic**：`notification` / `publish_task` / `ai_recheck`，Tag 按类型细分 |
| 7   | 生产端即时性 | **保留内存广播双写**（单实例下 RocketMQ/Redis 故障仍即时推送）                         |
| 8   | 跨进程桥接   | **Redis Pub/Sub**（通道 `notification:push`，已有 Redis 基础设施）                     |
| 9   | 顺序性       | **不要求严格顺序**（消息带 `created_at`，DB 为最终排序来源）                           |

> **架构要点**：选择「独立消费者进程」后，RocketMQ 消费端与 SSE 连接（进程内内存广播）**不在同一进程**。因此必须引入**跨进程推送桥接**（Redis Pub/Sub），由独立消费者进程把消息发布到 Redis，API 服务器各实例订阅后转发到本地 Broadcaster → SSE。Redis 已在现有 `docker-compose.yml` 中部署（redis:7-alpine），无需新增基础设施，仅需在 Go 侧引入 `go-redis` 依赖。

---

## 1. 背景与目标

当前消息（通知）模块的实时推送链路为：

```
业务控制器 → createNotificationWithChannel（写库 + 进程内内存广播）→ SSE → 前端
```

存在以下问题：

| 问题                   | 说明                                                                                                           |
| ---------------------- | -------------------------------------------------------------------------------------------------------------- |
| **无跨实例能力**       | 进程内内存广播仅单实例有效；多实例部署时，其他实例产生的通知无法实时到达本实例 SSE，只能依赖 500ms DB 轮询兜底 |
| **创建与推送同步耦合** | 通知创建时同步「先写库、再内存广播」，缺少异步削峰、重试与持久化中间层；批量通知场景下阻塞请求线程             |
| **扩展性受限**         | 接收者列表（`userIDsWithPermission`）在创建时同步全量计算，通知量大时成为性能瓶颈                              |
| **无异步任务承载**     | 后端没有任何持久化队列/worker/定时任务；发布任务只有「登记」没有「执行者」，AI 复核完全同步串行等待            |

**改造目标**：

1. 引入 RocketMQ 5.x 作为消息中间件，将「通知创建」与「实时推送」解耦；
2. 消息异步化、持久化、可靠投递，支持重试与死信处理；
3. 实现多实例水平扩展下的实时通知一致到达；
4. 以 RocketMQ 为统一异步底座，逐步承载发布任务、AI 复核等长耗时场景；
5. 对外保持 SSE 接口与前端交互**完全不变**（前端零改动）。

---

## 2. 现状分析

### 2.1 核心文件

| 层级         | 文件                                                                                    | 职责                                                                |
| ------------ | --------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| 模型         | `backend-go/models/notification.go`                                                     | `Notification`（notifications 表）、`NotificationDict`              |
| 统一创建入口 | `backend-go/controllers/base.go` `createNotification` / `createNotificationWithChannel` | 写库 + 内存广播                                                     |
| 内存广播     | `backend-go/services/notification_broadcaster.go`                                       | 全局单例 `map[userID]map[chan]bool`                                 |
| SSE 端点     | `backend-go/controllers/notifications_controller.go` `Stream`                           | SSE 长连接 + 500ms DB 轮询兜底 + knownIDs 去重                      |
| API 端点     | 同上                                                                                    | List / UnreadCount / TypeCounts / MarkRead / MarkAllRead / 字典管理 |
| 通知生产者   | reviews / db_changes / rbac_roles / rbac_users / inspection_tasks 控制器                | 20 处调用创建通知                                                   |
| 前端实时     | `frontend/src/composables/useNotificationRealtime.ts`                                   | EventSource 连接 SSE，失败降级 15s 轮询                             |
| 前端 UI      | NotificationBell / Notifications / NotificationDictManage                               | 铃铛角标、消息中心、字典管理                                        |

### 2.2 当前调用链

```
业务动作（审核/角色变更/SQL变更/巡店任务）
   │
   ▼
业务控制器（createNotificationWithChannel）
   ├─ ① o.Insert(&Notification{...})              → 写入 notifications 表
   └─ ② services.GetBroadcaster().Broadcast(...)   → 进程内内存广播
         ▼
controllers/notifications_controller.go Stream()
   ├─ 内存 chan 事件（即时）
   ├─ 500ms DB 轮询（兜底，knownIDs 去重）
   └─ context.Done（断开清理）
         ▼
SSE "event: notification"
         ▼
前端 useNotificationRealtime.ts（EventSource + 轮询降级）
```

### 2.3 现状特征与异步场景调查结论

- **未引入任何消息队列**（无 RocketMQ / Kafka / Redis 客户端），`go.mod` 仅 5 个直接依赖；根目录 `.env.example` 中的 `REDIS_URL` 与 `docker-compose.yml` 中的 redis 服务均属旧 Python 后端遗留，**Go 代码目前未使用 Redis**；
- 通知创建点共 20 处，接收者通过 RBAC 生效权限实时计算；
- `notifications` 表已含 `channel` 字段（默认 `internal`），可作为消息渠道扩展点；
- **配置方式**：Go 后端统一通过 Beego `conf/app.conf` 读取（如 `web.AppConfig.DefaultString("ai_api_key", ...)`），新增配置应追加到该文件；

**其他异步场景调查结果**：

| 场景                                                       | 现状                                                                                                                  | 结论                                                                                   |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| 发布任务（publish_task）                                   | **纯「登记制」**：`CreateTask` 仅写 DB 记录（publishing/published/failed 状态），**没有执行者**，`RetryTask` 仅改状态 | 可改造为「登记 → RocketMQ 消息 → 独立执行 worker 消费 → 回写状态 + 发通知」            |
| AI 生成（contents_controller AIGenerate/AIGenerateStream） | 请求内 goroutine + SSE 流式返回，一次性请求                                                                           | 改造需改为「提交 → 后台生成 → 完成后通知」，**会改变前端流式体验**，影响面大，建议暂缓 |
| 巡店整改 AI 复核（inspection_recheck_service）             | **完全同步串行**：提交整改后同步等待 AI 复核结果                                                                      | 可改造为「提交 → 后台复核 → 完成后通知 + 状态回写」，前端改为轮询/等待通知             |
| Redis                                                      | 仅 Python 侧 / docker-compose 有部署，Go 未接入                                                                       | 引入 `go-redis` 作为跨进程桥接依赖（基础设施已存在）                                   |

---

## 3. 目标架构

```
业务动作（审核/角色变更/SQL变更/巡店任务）
   │
   ▼
业务控制器 createNotificationWithChannel（改造）
   ├─ ① 写库 notifications 表（保持原有逻辑）
   ├─ ② RocketMQ Producer 同步发送（3 次重试，失败不阻塞）
   └─ ③ 内存广播双写（已定稿保留，单实例即时兜底）
         │
         ▼
  RocketMQ 5.x Broker（持久化 / 重试 / DLQ）
         │
         ▼
  独立消费者进程 notification-consumer（新 Go 二进制，cmd/notification-consumer）
   ├─ 幂等去重（msg_id 滑动窗口 + DB 存在性校验）
   ├─ [其他异步场景] 执行发布/AI 复核等业务逻辑并回写状态
   └─ Redis Pub/Sub 发布 channel: notification:push
         │
         ▼
  API 服务器进程（每实例一个 Redis 桥接订阅器 services/notification_bridge.go）
   └─ 反序列化 → 本地 Broadcaster.Broadcast(userID, payload)
         │
         ▼
  SSE /api/notifications/stream?token=（保持原样，knownIDs 去重）
         │
         ▼
  前端 useNotificationRealtime（保持原样，轮询降级保留）
```

**核心思路**：

- **生产端**：通知落库后，将通知载荷发送到 RocketMQ；发送失败**不阻塞主流程**（通知已落库，SSE 由 DB 轮询兜底补偿）；
- **消费端**：独立进程消费消息，做幂等去重与业务处理，再经 Redis Pub/Sub 桥接回 API 进程的内存 Broadcaster，最终推送到对应用户的 SSE 连接；
- **降级保障**：RocketMQ / Redis 任一不可用时，原有 500ms DB 轮询兜底通道仍可工作（实时性降低但功能不丢失）；生产端内存广播双写可保证单实例下 RocketMQ 故障时仍即时推送。

---

## 4. 详细设计

### 4.1 部署形态（已定稿：方案 B · Docker 5.x）

| 方案                              | 说明                                                                             | 结论                   |
| --------------------------------- | -------------------------------------------------------------------------------- | ---------------------- |
| A. Docker 单机 4.x                | nameserver + broker，客户端 `apache/rocketmq-client-go/v2`（TCP）                | ✗ 客户端已停止维护     |
| **B. Docker 单机 5.x**            | **nameserver + broker + proxy，客户端 `apache/rocketmq-clients/golang`（gRPC）** | **✓ 选定**             |
| C. 云上 RocketMQ（阿里云/腾讯云） | 托管服务，走 HTTP/gRPC 接入                                                      | 生产托管备选，架构兼容 |

### 4.2 Go 客户端选型（已定稿：官方 gRPC）

| 客户端                               | 协议                   | RocketMQ 版本 | 维护状态               | 结论       |
| ------------------------------------ | ---------------------- | ------------- | ---------------------- | ---------- |
| `apache/rocketmq-client-go/v2`       | TCP（Remoting）        | 4.x           | 已停止维护（社区存档） | ✗          |
| **`apache/rocketmq-clients/golang`** | **gRPC（Proxy 8081）** | **5.x**       | **官方维护**           | **✓ 选定** |

生产/消费代码形态（示意）：

```go
// 生产者
producer, _ := golang.NewProducer(&golang.ProducerOptions{
    ClientOptions: golang.ClientOptions{
        Endpoint: "127.0.0.1:8081",
        Credentials: &credentials.SessionCredentials{AccessKey: "rocketmq", SecretKey: "rocketmq"},
    },
    Topic: "notification",
}, nil)

// 消费者（独立进程内）
consumer, _ := golang.NewSimpleConsumer(&golang.SimpleConsumerOptions{
    ClientOptions: golang.ClientOptions{Endpoint: "127.0.0.1:8081", Credentials: ...},
    ConsumerGroup: "notification_consumer",
    Subscription:  &golang.Subscription{Topics: []string{"notification"}, Expression: golang.NewFilterExpression("*")},
}, nil)
```

### 4.3 Topic / Tag 设计（已定稿：每业务域一个 Topic）

```
Topic: notification       → 通知推送（消费组 notification_consumer）
Topic: publish_task       → 发布任务执行（消费组 publish_worker）
Topic: ai_recheck         → AI 复核异步化（消费组 ai_recheck_worker）
Tag：按业务类型细分（通知：review_submit / role_updated / inspection_task_* ...）
```

### 4.4 消息体结构

```json
{
  "msg_id": "uuid", // 消息唯一 ID（幂等去重）
  "user_id": "uuid", // 接收者
  "type": "review_submit", // 通知类型
  "title": "标题",
  "content": "内容",
  "related_id": "uuid", // 关联业务 ID
  "channel": "internal", // 渠道
  "created_at": "2026-08-17T10:00:00Z"
}
```

其他异步场景消息体在各自 Topic 下独立定义（如 publish_task 携带任务 ID，由消费端回查 DB 获取完整任务）。

### 4.5 生产者改造（backend-go/controllers/base.go）

```go
func createNotificationWithChannel(userID, ntype, title, content, relatedID, channel string) error {
    // ① 写库（保持原有逻辑）
    n := &models.Notification{...}
    if _, err := o.Insert(n); err != nil {
        return err
    }
    // ② 投递 RocketMQ（同步发送 + 重试；仍失败仅记日志，不阻塞主流程）
    payload := services.NotificationPayload{...}
    _ = services.SendNotificationMessage(payload) // 失败容忍，DB 轮询兜底
    // ③ 内存广播双写（已定稿保留：单实例即时兜底；SSE 侧 knownIDs 去重）
    services.GetBroadcaster().Broadcast(userID, payload)
    return nil
}
```

### 4.6 独立消费者进程设计（新增 backend-go/cmd/notification-consumer/main.go）

- **形态**：独立 Go 二进制，与 API 服务分离部署（复用现有 `cmd/seed_stores` 的多入口模式）；通过 Beego `conf/app.conf` 读取配置，需要访问 **同一 DB**（幂等校验、状态回写）与 **Redis**（推送桥接）；
- **启动流程**：读取配置 → 初始化 DB → 初始化 Redis 客户端 → 初始化 RocketMQ SimpleConsumer → 订阅各 Topic → 循环 `Receive` / 处理 / `Ack`；
- **通知消息处理**：
  1. 反序列化 `NotificationMessage`；
  2. 幂等去重（`msg_id` 滑动窗口内存去重 + 可选 DB 存在性校验）；
  3. 发布到 Redis `notification:push` 通道（payload 含 `user_id`，API 侧按订阅者过滤）；
  4. 处理成功 `Ack`，失败按重试策略；
- **消费并发**：默认 16 路并发，可通过配置调整；不同 Topic 使用独立消费组与并发数；
- **进程治理**：消费处理循环 + 优雅退出（signal 监听 + 停止订阅）；提供健康检查端口（可选）。

### 4.7 跨进程推送桥接：Redis Pub/Sub（已定稿）

> 因 SSE 长连接持有在 API 服务器进程内，而消费者是独立进程，二者之间必须跨进程投递。**已确认使用 Redis Pub/Sub 作为桥接通道**（Redis 已在现有 docker-compose 部署）。

- **消费侧（notification-consumer）**：处理完消息后 `redis.Publish("notification:push", jsonBytes)`；
- **API 侧（新增 backend-go/services/notification_bridge.go）**：
  - `main.go` 启动时（`ROCKETMQ_ENABLED=true`）拉起 `StartNotificationBridge()` goroutine；
  - 通过 `redis.Subscribe("notification:push")` 建立长订阅，断线按退避策略自动重连；
  - 收到消息 → 反序列化 `NotificationPayload` → `GetBroadcaster().Broadcast(userID, payload)`（Broadcaster 天然按 userID 过滤，非本实例订阅者零开销丢弃）；
  - **Redis 不可用时仅影响实时推送**，500ms DB 轮询兜底照常工作；
- **通道设计**：单通道 `notification:push`（消息量小，全量广播 + 本地过滤即可；如需优化可拆分 `notification:push:{userID}` 或改用 Redis Streams，非当前必须）；
- **依赖**：Go 侧新增 `github.com/redis/go-redis/v9`（唯一新增第三方依赖之一）。

### 4.8 可靠性设计（已定稿：标准可靠）

| 环节         | 策略                                                                                                            |
| ------------ | --------------------------------------------------------------------------------------------------------------- |
| 发送失败     | **同步发送 + 重试（3 次）**；仍失败记录日志，依赖内存广播双写 + DB 轮询兜底补偿                                 |
| 消费失败     | RocketMQ 消息阶梯重试（默认 16 次），重试耗尽进 **DLQ：`%DLQ%notification_consumer`**（按消费组自动生成）       |
| 幂等         | 消费端以 `msg_id` 去重（滑动窗口内存去重 + DB 存在性校验可选）；SSE 侧 `knownIDs` 二次去重，防双写重复推送      |
| 乱序         | 单用户通知无强顺序要求（消息带 `created_at`，DB 是最终排序来源）；如需严格顺序可按 `user_id` 分片消费（见 §11） |
| 降级         | RocketMQ / Redis 不可用 → 500ms DB 轮询兜底通道照常工作，功能不丢失                                             |
| 至少一次语义 | RocketMQ 保证至少一次投递，重复投递由幂等去重兜住                                                               |

### 4.9 配置管理（backend-go/conf/app.conf）

```
rocketmq_enabled = true                      # 总开关（false 回退纯内存广播 + DB 轮询）
rocketmq_endpoint = 127.0.0.1:8081           # 5.x gRPC Proxy 地址
rocketmq_topic_notification = notification   # 各 Topic 名
rocketmq_topic_publish_task = publish_task
rocketmq_topic_ai_recheck = ai_recheck
rocketmq_consumer_group = notification_consumer
rocketmq_access_key = rocketmq
rocketmq_secret_key = rocketmq
redis_addr = 127.0.0.1:6379
redis_password =
redis_db = 0
redis_channel_push = notification:push
```

---

## 5. 其他异步场景改造（范围待确认，见 §11）

> 背景：当前后端**无任何持久化队列/worker**。以下候选场景统一收敛到 RocketMQ 底座。改造影响差异较大，需确认纳入范围。

### 5.1 发布任务执行 worker（推荐纳入）

| 项       | 内容                                                                                                                            |
| -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| 现状     | `PublishController.CreateTask` 仅登记 DB 记录；`publishing/published/failed` 状态无执行者                                       |
| 改造     | 创建任务时落库 + 发 `publish_task` 消息；`publish_worker` 消费组执行实际发布（复用现有发布逻辑），完成后回写状态 + 发送完成通知 |
| 新增文件 | `cmd/publish-worker/main.go`、`services/publish_worker.go`                                                                      |
| 影响面   | 后端新增执行逻辑，**前端不变**（任务列表/状态/重试按钮语义保持）                                                                |

### 5.2 AI 复核异步化（已确认纳入）

| 项       | 内容                                                                                                                                                                                |
| -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 现状     | 整改提交后同步串行等待 AI 复核（`inspection_recheck_service.go`），请求长时间阻塞                                                                                                   |
| 改造     | 提交整改时落库 + 发 `ai_recheck` 消息；`ai_recheck_worker` 消费组后台执行复核，完成后回写复核结果 + 发送「复核通过/不通过」通知（复用现有通知类型 `recheck_passed/recheck_failed`） |
| 新增文件 | `cmd/ai-recheck-worker/main.go`、`services/ai_recheck_worker.go`                                                                                                                    |
| 影响面   | 前端提交整改后改为「等待通知/轮询状态」，需小改前端交互提示                                                                                                                         |

### 5.3 AI 生成异步化（不建议本期纳入）

| 项   | 内容                                                                                        |
| ---- | ------------------------------------------------------------------------------------------- |
| 现状 | `AIGenerateStream` 请求内 goroutine + SSE 流式返回                                          |
| 改造 | 改为后台生成 + 完成后通知，**丢失流式体验**，前端改动大（生成进度、取消、重试）             |
| 建议 | 本期仅将「生成完成 → 通知」链路接入通知 Topic（若生成入口已有通知），**不重构生成执行方式** |

---

## 6. 部署方案（Docker Compose）

### 6.1 新增 `docker-compose.rocketmq.yml`（5.x 单机，gRPC 接入）

```yaml
services:
  rocketmq-namesrv:
    image: apache/rocketmq:5.3.1
    container_name: rocketmq-namesrv
    command: sh mqnamesrv
    ports:
      - '9876:9876'
    environment:
      - TZ=Asia/Shanghai
      - JAVA_OPT_EXT=-Xms512m -Xmx512m -Xmn256m
    volumes:
      - rocketmq_namesrv_logs:/home/rocketmq/logs

  rocketmq-broker:
    image: apache/rocketmq:5.3.1
    container_name: rocketmq-broker
    command: sh mqbroker -n rocketmq-namesrv:9876 --enable-proxy -c /home/rocketmq/conf/broker.conf
    depends_on:
      - rocketmq-namesrv
    ports:
      - '10909:10909'
      - '10911:10911'
      - '8081:8081' # gRPC Proxy（5.x 客户端接入端口）
    environment:
      - TZ=Asia/Shanghai
      - NAMESRV_ADDR=rocketmq-namesrv:9876
      - JAVA_OPT_EXT=-Xms1g -Xmx1g -Xmn512m
    volumes:
      - ./docker/rocketmq/broker.conf:/home/rocketmq/conf/broker.conf
      - rocketmq_broker_store:/home/rocketmq/store
      - rocketmq_broker_logs:/home/rocketmq/logs

volumes:
  rocketmq_namesrv_logs:
  rocketmq_broker_store:
  rocketmq_broker_logs:
```

### 6.2 新增 `docker/rocketmq/broker.conf`

```properties
brokerClusterName=DefaultCluster
brokerName=broker-a
brokerId=0
deleteWhen=04
fileReservedTime=48
brokerRole=ASYNC_MASTER
flushDiskType=ASYNC_FLUSH
autoCreateTopicEnable=true
autoCreateSubscriptionGroup=true
```

### 6.3 消费者进程启动方式

- 开发：`go run ./backend-go/cmd/notification-consumer`；
- 生产：随 API 服务一同部署的独立 systemd/容器进程，或 `docker-compose.yml` 追加 `notification-consumer` 服务（与 API 共用 DB / Redis 网络）。

---

## 7. 前端改动

**无改动**。SSE 端点、消息结构、轮询降级策略均保持不变，前端对 RocketMQ / Redis 无感知。

---

## 8. 兼容性与降级

1. `ROCKETMQ_ENABLED=false`（默认关闭时）走原有「写库 + 内存广播 + DB 轮询」链路，功能等价，灰度切换；
2. 生产端发送失败不阻塞主流程（通知已落库）；
3. 独立消费者进程未启动 / 崩溃 → 仅影响实时推送，DB 轮询兜底持续可用；
4. Redis 桥接不可用 → 同上，DB 轮询兜底；
5. SSE 端点与 knownIDs 去重逻辑保持不变，双写路径重复推送被去重。

---

## 9. 改造范围与任务拆分

| #   | 任务                                      | 涉及文件                                                         |
| --- | ----------------------------------------- | ---------------------------------------------------------------- |
| 1   | RocketMQ / go-redis 依赖引入与配置读取    | `go.mod`、`conf/app.conf`                                        |
| 2   | 消息体定义与序列化                        | `services/notification_message.go`                               |
| 3   | 生产者 `SendNotificationMessage`          | `services/notification_producer.go`                              |
| 4   | 创建链路改造（写库后投递 + 内存广播双写） | `controllers/base.go`                                            |
| 5   | 独立消费者进程（通知链路）                | `cmd/notification-consumer/main.go`                              |
| 6   | Redis 桥接（API 侧订阅转发）              | `services/notification_bridge.go`、`main.go`                     |
| 7   | Docker Compose（RocketMQ 5.x）            | `docker-compose.rocketmq.yml`、`docker/rocketmq/broker.conf`     |
| 8   | 发布任务 worker                           | `cmd/publish-worker/main.go`、`services/publish_worker.go`       |
| 9   | AI 复核 worker                            | `cmd/ai-recheck-worker/main.go`、`services/ai_recheck_worker.go` |
| 10  | 联调验证（单实例/多实例 SSE、故障注入）   | 测试脚本                                                         |

---

## 10. 风险与注意事项

1. **客户端生态**：`rocketmq-clients/golang` 较新（5.x），需验证与 5.3.x Broker/Proxy 的兼容性与稳定性；建议先做最小发送/消费冒烟测试；
2. **多进程部署复杂度**：独立消费者进程需要访问同一 DB 与 Redis，配置与部署节奏需与 API 服务同步管理（同仓库、同一配置源）；
3. **Redis Pub/Sub 即发即弃**：通道无订阅者时消息丢失（消费端 Publish 时 API 侧恰未连上）——因 SSE 为临时连接且 DB 轮询兜底，丢失可接受；如需不丢可升级为 Redis Streams（非当前必须）；
4. **重复通知**：RocketMQ 至少一次投递 + 双写路径可能导致重复推送，靠 SSE 侧 `knownIDs` + 消费端 `msg_id` 双重去重；
5. **时序**：内存广播双写先于 RocketMQ 消费到达，SSE 侧已 `knownIDs` 去重，不会重复推送；
6. **资源开销**：RocketMQ Broker 内存要求较高（建议 ≥1GB，compose 中已限制），开发机需评估；
7. **升级节奏**：RocketMQ 依赖作为独立服务引入，建议 `.env` 侧不破坏现有启动方式（`dev:backend` 脚本仅启动 Go 服务，RocketMQ 需显式 `docker compose -f docker-compose.rocketmq.yml up -d`）。

---

## 11. 决策确认记录

以下决策均已确认，进入实施后不再变更（如需调整请重新评审本文档）：

| #   | 决策项           | 确认结论                                                               | 影响章节 |
| --- | ---------------- | ---------------------------------------------------------------------- | -------- |
| Q1  | 其他异步场景范围 | 发布任务执行 worker + AI 复核异步化，**均纳入本期**；AI 生成异步化暂缓 | §5       |
| Q2  | Topic 组织方式   | **每业务域一个 Topic**（notification / publish_task / ai_recheck）     | §4.3     |
| Q3  | 生产端即时性     | **保留内存广播双写**（单实例即时兜底，knownIDs 去重防重复）            | §4.5     |
| Q4  | 跨进程桥接       | **Redis Pub/Sub**（通道 `notification:push`，API 侧桥接订阅器转发）    | §4.7     |
| Q5  | 顺序性           | **不要求严格顺序**（消息带 `created_at`，DB 为最终排序来源）           | §4.8     |
