package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/middleware"
	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

const TimeFormat = "2006-01-02 15:04:05"

type BaseController struct {
	web.Controller
}

func newID() string {
	return uuid.NewString()
}

func (c *BaseController) CurrentUser() *models.User {
	return middleware.CurrentUser(c.Ctx)
}

func (c *BaseController) WriteJSON(status int, data interface{}) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = middleware.MaskResponseData(data, c.Ctx.Input.URL())
	c.ServeJSON()
}

func (c *BaseController) WriteError(status int, detail string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = map[string]interface{}{"detail": detail}
	c.ServeJSON()
}

func (c *BaseController) WriteNoContent() {
	c.Ctx.Output.SetStatus(http.StatusNoContent)
}

func (c *BaseController) OK(data interface{}) {
	c.WriteJSON(http.StatusOK, data)
}

func (c *BaseController) Created(data interface{}) {
	c.WriteJSON(http.StatusCreated, data)
}

func (c *BaseController) CheckPermission(key string) bool {
	return middleware.RequirePermission(c.Ctx, key)
}

func (c *BaseController) CheckPermissionAny(keys []string) bool {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return false
	}
	for _, key := range keys {
		ok, err := services.HasPermission(user.ID, key, "read", nil)
		if err == nil && ok {
			return true
		}
	}
	c.WriteError(http.StatusForbidden, "权限不足：需要任一权限")
	return false
}

func (c *BaseController) ParseBody(target interface{}) error {
	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		return errors.New("请求体为空")
	}
	return json.Unmarshal(body, target)
}

func (c *BaseController) GetQuery(key string) string {
	return c.Ctx.Input.Query(key)
}

func (c *BaseController) GetPathParam(key string) string {
	return c.Ctx.Input.Param(":" + key)
}

func (c *BaseController) ParsePagination() (page, pageSize int) {
	page, _ = strconv.Atoi(c.Ctx.Input.Query("page"))
	pageSize, _ = strconv.Atoi(c.Ctx.Input.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return
}

type userInfo struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	Role      string    `json:"role"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

func toUserInfo(u *models.User) userInfo {
	return userInfo{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Nickname:  u.Nickname,
		Role:      u.Role,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
	}
}

func userRoles(userID string) []map[string]interface{} {
	o := services.GetOrm()
	var assignments []models.RBACUserRoleAssignment
	_, err := o.QueryTable(new(models.RBACUserRoleAssignment)).
		Filter("user_id", userID).
		All(&assignments)
	if err != nil || len(assignments) == 0 {
		return []map[string]interface{}{}
	}
	roleIDs := make([]string, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}
	var roles []models.RBACRole
	_, err = o.QueryTable(new(models.RBACRole)).
		Filter("id__in", roleIDs).
		All(&roles)
	if err != nil {
		return []map[string]interface{}{}
	}
	result := make([]map[string]interface{}, 0, len(roles))
	for _, r := range roles {
		result = append(result, map[string]interface{}{
			"id":           r.ID,
			"name":         r.Name,
			"display_name": r.DisplayName,
		})
	}
	return result
}

// userIDsWithPermission 返回所有拥有指定权限（或经继承推导）的用户 ID。
func userIDsWithPermission(permKey string) []string {
	o := services.GetOrm()
	var users []models.User
	_, err := o.QueryTable(new(models.User)).All(&users)
	if err != nil {
		return nil
	}
	ids := make([]string, 0)
	for _, u := range users {
		perms, err := services.GetUserEffectiveFlatPermissions(u.ID, nil)
		if err != nil {
			continue
		}
		if _, ok := perms[permKey]; ok {
			ids = append(ids, u.ID)
		}
	}
	return ids
}

// createNotification 写入通知并推送实时事件。
func createNotification(userID, ntype, title, content, relatedID string) error {
	return createNotificationWithChannel(userID, ntype, title, content, relatedID, models.NotificationChannelInternal)
}

func createNotificationWithChannel(userID, ntype, title, content, relatedID, channel string) error {
	o := services.GetOrm()
	n := &models.Notification{
		ID:        newID(),
		UserID:    userID,
		Type:      ntype,
		Title:     title,
		Content:   content,
		RelatedID: relatedID,
		Channel:   channel,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	if _, err := o.Insert(n); err != nil {
		return err
	}
	payload := services.NotificationPayload{
		ID:        n.ID,
		Type:      ntype,
		Title:     title,
		Content:   content,
		RelatedID: relatedID,
		IsRead:    false,
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05"),
	}
	services.GetBroadcaster().Broadcast(userID, payload)
	return nil
}

func notificationMap(n *models.Notification) map[string]interface{} {
	return map[string]interface{}{
		"id":         n.ID,
		"type":       n.Type,
		"title":      n.Title,
		"content":    n.Content,
		"related_id": n.RelatedID,
		"channel":    n.Channel,
		"is_read":    n.IsRead,
		"created_at": n.CreatedAt.Format("2006-01-02T15:04:05"),
	}
}
