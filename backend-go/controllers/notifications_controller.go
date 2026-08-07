package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
	"ai-multi-platform-release/backend-go/utils"
)

type NotificationsController struct {
	BaseController
}

const isoFormat = "2006-01-02T15:04:05"

// List GET /api/notifications/
func (c *NotificationsController) List() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.Notification)).
		Filter("user_id", user.ID)
	if ntype := c.GetQuery("type"); ntype != "" {
		qs = qs.Filter("type", ntype)
	}
	total, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询通知失败")
		return
	}
	var notifications []models.Notification
	_, err = qs.OrderBy("-created_at").Limit(pageSize).Offset((page - 1) * pageSize).All(&notifications)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询通知失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(notifications))
	for i := range notifications {
		items = append(items, notificationMap(&notifications[i]))
	}
	c.OK(map[string]interface{}{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     items,
	})
}

// UnreadCount GET /api/notifications/unread-count
func (c *NotificationsController) UnreadCount() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	count, err := services.GetOrm().QueryTable(new(models.Notification)).
		Filter("user_id", user.ID).
		Filter("is_read", false).
		Count()
	if err != nil {
		count = 0
	}
	c.OK(map[string]interface{}{"count": count})
}

// TypeCounts GET /api/notifications/type-counts
func (c *NotificationsController) TypeCounts() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var notifications []models.Notification
	_, err := services.GetOrm().QueryTable(new(models.Notification)).
		Filter("user_id", user.ID).
		All(&notifications)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询通知失败")
		return
	}
	counts := map[string]int64{}
	for _, n := range notifications {
		counts[n.Type]++
	}
	c.OK(map[string]interface{}{
		"total":  int64(len(notifications)),
		"counts": counts,
	})
}

// MarkRead POST /api/notifications/:id/read
func (c *NotificationsController) MarkRead() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var notification models.Notification
	err := services.GetOrm().QueryTable(new(models.Notification)).
		Filter("id", c.GetPathParam("id")).
		Filter("user_id", user.ID).
		One(&notification)
	if err != nil {
		c.WriteError(http.StatusNotFound, "通知不存在")
		return
	}
	notification.IsRead = true
	if _, err := services.GetOrm().Update(&notification); err != nil {
		c.WriteError(http.StatusInternalServerError, "标记已读失败")
		return
	}
	c.OK(map[string]string{"message": "已标记为已读"})
}

// MarkAllRead POST /api/notifications/read-all
func (c *NotificationsController) MarkAllRead() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	_, err := services.GetOrm().QueryTable(new(models.Notification)).
		Filter("user_id", user.ID).
		Filter("is_read", false).
		Update(map[string]interface{}{"is_read": true})
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "标记已读失败")
		return
	}
	c.OK(map[string]string{"message": "已全部标记为已读"})
}

// Stream GET /api/notifications/stream
func (c *NotificationsController) Stream() {
	token := c.GetQuery("token")
	claims, err := utils.DecodeAccessToken(token)
	if err != nil || claims == nil || claims.UserID == "" {
		c.WriteError(http.StatusUnauthorized, "令牌无效")
		return
	}
	userID := claims.UserID

	broadcaster := services.GetBroadcaster()
	ch := broadcaster.Subscribe(userID)
	defer broadcaster.Unsubscribe(userID, ch)

	w := c.Ctx.ResponseWriter
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if !ok {
		c.WriteError(http.StatusInternalServerError, "SSE 不受支持")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"ok\"}\n\n"))
	flusher.Flush()

	knownIDs := map[string]bool{}

	poll := func() {
		newItems := pollNotifications(userID, knownIDs)
		if len(newItems) > 0 {
			data, _ := json.Marshal(newItems)
			_, _ = w.Write([]byte("event: notification\ndata: " + string(data) + "\n\n"))
			flusher.Flush()
		}
	}
	poll()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Ctx.Request.Context().Done():
			return
		case item, ok := <-ch:
			if !ok {
				return
			}
			if knownIDs[item.ID] {
				continue
			}
			knownIDs[item.ID] = true
			data, _ := json.Marshal([]services.NotificationPayload{item})
			_, _ = w.Write([]byte("event: notification\ndata: " + string(data) + "\n\n"))
			flusher.Flush()
		case <-ticker.C:
			poll()
		}
	}
}

func pollNotifications(userID string, knownIDs map[string]bool) []map[string]interface{} {
	var notifications []models.Notification
	_, err := services.GetOrm().QueryTable(new(models.Notification)).
		Filter("user_id", userID).
		Filter("is_read", 0).
		OrderBy("-created_at").
		Limit(20).
		All(&notifications)
	if err != nil {
		return nil
	}
	var newItems []map[string]interface{}
	for i := range notifications {
		n := &notifications[i]
		if knownIDs[n.ID] {
			continue
		}
		knownIDs[n.ID] = true
		newItems = append(newItems, notificationMap(n))
	}
	return newItems
}
