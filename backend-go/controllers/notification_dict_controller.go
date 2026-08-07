package controllers

import (
	"net/http"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type NotificationDictController struct {
	BaseController
}

type notificationDictCreateRequest struct {
	Category  string `json:"category"`
	GroupKey  string `json:"group_key"`
	DictKey   string `json:"dict_key"`
	DictValue string `json:"dict_value"`
	IsActive  bool   `json:"is_active"`
	IsPrivate bool   `json:"is_private"`
	SortOrder int    `json:"sort_order"`
}

type notificationDictUpdateRequest struct {
	Category  *string `json:"category"`
	GroupKey  *string `json:"group_key"`
	DictKey   *string `json:"dict_key"`
	DictValue *string `json:"dict_value"`
	IsActive  *bool   `json:"is_active"`
	IsPrivate *bool   `json:"is_private"`
	SortOrder *int    `json:"sort_order"`
}

// ListGroups GET /api/notification-dict/groups
func (c *NotificationDictController) ListGroups() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var items []models.NotificationDict
	_, err := services.GetOrm().QueryTable(new(models.NotificationDict)).
		OrderBy("sort_order", "created_at").
		All(&items)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询字典失败")
		return
	}
	groups := []map[string]interface{}{}
	index := map[string]int{}
	for i := range items {
		item := &items[i]
		idx, ok := index[item.GroupKey]
		if !ok {
			idx = len(groups)
			index[item.GroupKey] = idx
			groups = append(groups, map[string]interface{}{
				"group_key": item.GroupKey,
				"items":     []map[string]interface{}{},
			})
		}
		groups[idx]["items"] = append(groups[idx]["items"].([]map[string]interface{}), dictEntryMap(item))
	}
	c.OK(groups)
}

// List GET /api/notification-dict/
func (c *NotificationDictController) List() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	qs := services.GetOrm().QueryTable(new(models.NotificationDict)).
		OrderBy("sort_order", "created_at")
	if category := c.GetQuery("category"); category != "" {
		qs = qs.Filter("category", category)
	}
	if groupKey := c.GetQuery("group_key"); groupKey != "" {
		qs = qs.Filter("group_key", groupKey)
	}
	var items []models.NotificationDict
	if _, err := qs.All(&items); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询字典失败")
		return
	}
	result := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		result = append(result, dictEntryMap(&items[i]))
	}
	c.OK(map[string]interface{}{"items": result, "total": len(result)})
}

// Create POST /api/notification-dict/
func (c *NotificationDictController) Create() {
	if !c.CheckPermission("notification:enum:write") {
		return
	}
	var req notificationDictCreateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Category != "field" && req.Category != "enum" {
		c.WriteError(http.StatusBadRequest, "category 必须为 field 或 enum")
		return
	}
	exists := services.GetOrm().QueryTable(new(models.NotificationDict)).
		Filter("group_key", req.GroupKey).
		Filter("dict_key", req.DictKey).
		Exist()
	if exists {
		c.WriteError(http.StatusConflict, "该分组下已存在相同的 dict_key")
		return
	}
	entry := &models.NotificationDict{
		ID:        newID(),
		Category:  req.Category,
		GroupKey:  req.GroupKey,
		DictKey:   req.DictKey,
		DictValue: req.DictValue,
		IsActive:  req.IsActive,
		IsPrivate: req.IsPrivate,
		SortOrder: req.SortOrder,
	}
	if _, err := services.GetOrm().Insert(entry); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建字典条目失败")
		return
	}
	c.OK(dictEntryMap(entry))
}

func findDictEntry(entryID string) (*models.NotificationDict, error) {
	var entry models.NotificationDict
	err := services.GetOrm().QueryTable(new(models.NotificationDict)).
		Filter("id", entryID).
		One(&entry)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// Update PUT /api/notification-dict/:entry_id
func (c *NotificationDictController) Update() {
	if !c.CheckPermission("notification:enum:write") {
		return
	}
	entry, err := findDictEntry(c.GetPathParam("entry_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "字典条目不存在")
		return
	}
	var req notificationDictUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Category != nil {
		if *req.Category != "field" && *req.Category != "enum" {
			c.WriteError(http.StatusBadRequest, "category 必须为 field 或 enum")
			return
		}
		entry.Category = *req.Category
	}
	if req.GroupKey != nil {
		entry.GroupKey = *req.GroupKey
	}
	if req.DictKey != nil {
		entry.DictKey = *req.DictKey
	}
	if req.DictValue != nil {
		entry.DictValue = *req.DictValue
	}
	if req.IsActive != nil {
		entry.IsActive = *req.IsActive
	}
	if req.IsPrivate != nil {
		entry.IsPrivate = *req.IsPrivate
	}
	if req.SortOrder != nil {
		entry.SortOrder = *req.SortOrder
	}
	if _, err := services.GetOrm().Update(entry); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新字典条目失败")
		return
	}
	c.OK(dictEntryMap(entry))
}

// Delete DELETE /api/notification-dict/:entry_id
func (c *NotificationDictController) Delete() {
	if !c.CheckPermission("notification:enum:write") {
		return
	}
	entry, err := findDictEntry(c.GetPathParam("entry_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "字典条目不存在")
		return
	}
	if _, err := services.GetOrm().Delete(entry); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除字典条目失败")
		return
	}
	c.OK(map[string]string{"message": "删除成功"})
}

func dictEntryMap(e *models.NotificationDict) map[string]interface{} {
	return map[string]interface{}{
		"id":         e.ID,
		"category":   e.Category,
		"group_key":  e.GroupKey,
		"dict_key":   e.DictKey,
		"dict_value": e.DictValue,
		"is_active":  e.IsActive,
		"is_private": e.IsPrivate,
		"sort_order": e.SortOrder,
		"created_at": e.CreatedAt,
		"updated_at": e.UpdatedAt,
	}
}
