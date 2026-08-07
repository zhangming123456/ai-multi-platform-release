package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type ContentsController struct {
	BaseController
}

type contentCreateRequest struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Platform  string `json:"platform"`
	Status    string `json:"status"`
	MediaURLs string `json:"media_urls"`
}

type contentUpdateRequest struct {
	Title     *string `json:"title"`
	Body      *string `json:"body"`
	Platform  *string `json:"platform"`
	Status    *string `json:"status"`
	MediaURLs *string `json:"media_urls"`
}

type aiGenerateRequest struct {
	Topic    string   `json:"topic"`
	Platform string   `json:"platform"`
	Style    string   `json:"style"`
	Keywords []string `json:"keywords"`
	Count    int      `json:"count"`
	PlanID   string   `json:"plan_id"`
}

type aiGenerateStreamRequest struct {
	Topic     string                  `json:"topic"`
	Platforms []string                `json:"platforms"`
	Style     string                  `json:"style"`
	Keywords  []string                `json:"keywords"`
	PlanID    string                  `json:"plan_id"`
	ModelID   string                  `json:"model_id"`
	Files     []services.UploadedFile `json:"files"`
}

var platformLabels = map[string]string{
	"wechat_mp":    "公众号",
	"xiaohongshu":  "小红书",
	"douyin":       "抖音",
	"wechat_video": "视频号",
}

func sseEvent(event string, data interface{}) string {
	payload, _ := json.Marshal(data)
	return "event: " + event + "\ndata: " + string(payload) + "\n\n"
}

func findOwnContent(contentID, userID string) (*models.Content, error) {
	var content models.Content
	err := services.GetOrm().QueryTable(new(models.Content)).
		Filter("id", contentID).
		Filter("user_id", userID).
		One(&content)
	if err != nil {
		return nil, err
	}
	return &content, nil
}

func aiRecordMap(r *models.AIGenerationRecord) map[string]interface{} {
	hashtags := []string{}
	if r.Hashtags != "" {
		_ = json.Unmarshal([]byte(r.Hashtags), &hashtags)
	}
	return map[string]interface{}{
		"id":         r.ID,
		"user_id":    r.UserID,
		"topic":      r.Topic,
		"platform":   r.Platform,
		"plan_id":    r.PlanID,
		"model":      r.Model,
		"title":      r.Title,
		"body":       r.Body,
		"hashtags":   hashtags,
		"created_at": r.CreatedAt,
	}
}

// List GET /api/contents/
func (c *ContentsController) List() {
	if !c.CheckPermission("content:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.Content)).
		Filter("user_id", user.ID)
	if platform := c.GetQuery("platform"); platform != "" {
		qs = qs.Filter("platform", platform)
	}
	if status := c.GetQuery("status"); status != "" {
		qs = qs.Filter("status", status)
	}
	var contents []models.Content
	_, err := qs.OrderBy("-created_at").Limit(pageSize).Offset((page - 1) * pageSize).All(&contents)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询内容失败")
		return
	}
	c.OK(contents)
}

// Create POST /api/contents/
func (c *ContentsController) Create() {
	if !c.CheckPermission("content:create:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req contentCreateRequest
	if err := c.ParseBody(&req); err != nil || req.Title == "" {
		c.WriteError(http.StatusBadRequest, "标题不能为空")
		return
	}
	status := req.Status
	if status == "" {
		status = models.ContentStatusDraft
	}
	content := &models.Content{
		ID:        newID(),
		UserID:    user.ID,
		Title:     req.Title,
		Body:      req.Body,
		Platform:  req.Platform,
		Status:    status,
		MediaURLs: req.MediaURLs,
	}
	if _, err := services.GetOrm().Insert(content); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建内容失败")
		return
	}
	c.Created(content)
}

// Get GET /api/contents/:content_id
func (c *ContentsController) Get() {
	if !c.CheckPermission("content:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	content, err := findOwnContent(c.GetPathParam("content_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "内容不存在")
		return
	}
	c.OK(content)
}

// Update PUT /api/contents/:content_id
func (c *ContentsController) Update() {
	if !c.CheckPermission("content:update:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	content, err := findOwnContent(c.GetPathParam("content_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "内容不存在")
		return
	}
	var req contentUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Title != nil {
		content.Title = *req.Title
	}
	if req.Body != nil {
		content.Body = *req.Body
	}
	if req.Platform != nil {
		content.Platform = *req.Platform
	}
	if req.Status != nil {
		content.Status = *req.Status
	}
	if req.MediaURLs != nil {
		content.MediaURLs = *req.MediaURLs
	}
	if _, err := services.GetOrm().Update(content); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新内容失败")
		return
	}
	c.OK(content)
}

// Delete DELETE /api/contents/:content_id
func (c *ContentsController) Delete() {
	if !c.CheckPermission("content:delete:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	content, err := findOwnContent(c.GetPathParam("content_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "内容不存在")
		return
	}
	if _, err := services.GetOrm().Delete(content); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除内容失败")
		return
	}
	c.WriteNoContent()
}

// ListGenerations GET /api/contents/ai-generations
func (c *ContentsController) ListGenerations() {
	if !c.CheckPermission("content:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.AIGenerationRecord)).
		Filter("user_id", user.ID)
	if platform := c.GetQuery("platform"); platform != "" {
		qs = qs.Filter("platform", platform)
	}
	var records []models.AIGenerationRecord
	_, err := qs.OrderBy("-created_at").Limit(pageSize).Offset((page - 1) * pageSize).All(&records)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询 AI 生成记录失败")
		return
	}
	result := make([]map[string]interface{}, 0, len(records))
	for i := range records {
		result = append(result, aiRecordMap(&records[i]))
	}
	c.OK(result)
}

// AIGenerate POST /api/contents/ai-generate
func (c *ContentsController) AIGenerate() {
	if !c.CheckPermission("content:ai_generate:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req aiGenerateRequest
	if err := c.ParseBody(&req); err != nil || req.Topic == "" {
		c.WriteError(http.StatusBadRequest, "主题不能为空")
		return
	}
	count := req.Count
	if count <= 0 {
		count = 3
	}
	variants, err := services.GenerateContentVariants(req.Topic, req.Platform, req.Style, req.Keywords, count, req.PlanID, "")
	if err != nil {
		if aiErr, ok := err.(*services.AIGenerationError); ok {
			c.WriteJSON(http.StatusBadGateway, map[string]interface{}{
				"detail": map[string]interface{}{
					"message":         aiErr.Message,
					"available_plans": aiErr.AvailablePlans,
				},
			})
			return
		}
		c.WriteError(http.StatusInternalServerError, "AI 生成失败")
		return
	}

	modelName := ""
	if req.PlanID != "" {
		var plan models.ModelConfig
		plan.ID = req.PlanID
		if err := services.GetOrm().Read(&plan); err == nil {
			modelName = plan.Model
		}
	}

	o := services.GetOrm()
	for _, v := range variants {
		hashtagsJSON, _ := json.Marshal(v.Hashtags)
		record := &models.AIGenerationRecord{
			ID:       newID(),
			UserID:   user.ID,
			Topic:    req.Topic,
			Platform: req.Platform,
			PlanID:   req.PlanID,
			Model:    modelName,
			Title:    v.Title,
			Body:     v.Body,
			Hashtags: string(hashtagsJSON),
		}
		if _, err := o.Insert(record); err != nil {
			c.WriteError(http.StatusInternalServerError, "保存 AI 生成记录失败")
			return
		}
	}
	c.OK(map[string]interface{}{"variants": variants})
}

// AIGenerateStream POST /api/contents/ai-generate-stream
func (c *ContentsController) AIGenerateStream() {
	if !c.CheckPermission("content:ai_generate:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req aiGenerateStreamRequest
	if err := c.ParseBody(&req); err != nil || req.Topic == "" || len(req.Platforms) == 0 {
		c.WriteError(http.StatusBadRequest, "主题和平台不能为空")
		return
	}

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

	write := func(event string, data interface{}) {
		_, _ = w.Write([]byte(sseEvent(event, data)))
		flusher.Flush()
	}

	o := services.GetOrm()
	for _, platform := range req.Platforms {
		label := platformLabels[platform]
		if label == "" {
			label = platform
		}
		write("log", map[string]interface{}{
			"level":   "req",
			"message": fmt.Sprintf("开始为「%s」生成内容…", label),
		})

		events, err := services.GenerateContentStream(req.Topic, platform, req.Style, req.Keywords, req.PlanID, req.ModelID, req.Files)
		if err != nil {
			write("error", map[string]interface{}{
				"platform":        platform,
				"message":         err.Error(),
				"available_plans": []interface{}{},
			})
			continue
		}
		for evt := range events {
			switch evt.Event {
			case "chunk":
				write("chunk", map[string]interface{}{
					"platform": platform,
					"text":     evt.Text,
				})
			case "done":
				if evt.Variant != nil {
					record := &models.AIGenerationRecord{
						ID:       newID(),
						UserID:   user.ID,
						Topic:    req.Topic,
						Platform: platform,
						PlanID:   req.PlanID,
						Model:    evt.Model,
						Title:    evt.Variant.Title,
						Body:     evt.Variant.Body,
						Hashtags: marshalHashtags(evt.Variant.Hashtags),
					}
					_, _ = o.Insert(record)
					write("done", map[string]interface{}{
						"platform": platform,
						"variant":  evt.Variant,
					})
				}
			case "error":
				write("error", map[string]interface{}{
					"platform":        platform,
					"message":         evt.Message,
					"available_plans": evt.AvailablePlans,
				})
			case "log":
				level := evt.Level
				if level == "" {
					level = "info"
				}
				write("log", map[string]interface{}{
					"platform": platform,
					"level":    level,
					"message":  evt.Message,
				})
			}
		}
	}
	write("complete", map[string]interface{}{"message": "全部生成完成"})
}

func marshalHashtags(tags []string) string {
	b, _ := json.Marshal(tags)
	return string(b)
}

func normalizePlatform(p string) string {
	return strings.TrimSpace(p)
}
