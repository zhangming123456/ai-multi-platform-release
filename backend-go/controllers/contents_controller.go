package controllers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type ContentsController struct {
	BaseController
}

type contentCreateRequest struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	Platform   string `json:"platform"`
	Status     string `json:"status"`
	MediaURLs  string `json:"media_urls"`
	CampaignID string `json:"campaign_id"`
}

type contentUpdateRequest struct {
	Title      *string `json:"title"`
	Body       *string `json:"body"`
	Platform   *string `json:"platform"`
	Status     *string `json:"status"`
	MediaURLs  *string `json:"media_urls"`
	CampaignID *string `json:"campaign_id"`
}

type aiGenerateRequest struct {
	Topic      string   `json:"topic"`
	Platform   string   `json:"platform"`
	Style      string   `json:"style"`
	Keywords   []string `json:"keywords"`
	Count      int      `json:"count"`
	PlanID     string   `json:"plan_id"`
	CampaignID string   `json:"campaign_id"`
}

type aiGenerateStreamRequest struct {
	Topic           string                  `json:"topic"`
	Platforms       []string                `json:"platforms"`
	Style           string                  `json:"style"`
	Keywords        []string                `json:"keywords"`
	PlanID          string                  `json:"plan_id"`
	ModelID         string                  `json:"model_id"`
	Files           []services.UploadedFile `json:"files"`
	CampaignID      string                  `json:"campaign_id"`
	GenerateVersion int                     `json:"generate_version"`
}

var platformLabels = map[string]string{
	"wechat_mp":      "公众号",
	"xiaohongshu":    "小红书",
	"douyin":         "抖音",
	"wechat_video":   "视频号",
	"wechat_moments": "朋友圈",
	"weibo":          "微博",
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
	variant := services.AIVariant{Title: r.Title, Body: r.Body, Hashtags: hashtags}
	score := services.ScoreVariant(variant, r.Platform)
	return map[string]interface{}{
		"id":          r.ID,
		"user_id":     r.UserID,
		"topic":       r.Topic,
		"platform":    r.Platform,
		"plan_id":     r.PlanID,
		"model":       r.Model,
		"title":       r.Title,
		"body":        r.Body,
		"hashtags":    hashtags,
		"campaign_id": r.CampaignID,
		"created_at":  r.CreatedAt,
		"score":       score,
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
	if campaignID := c.GetQuery("campaign_id"); campaignID != "" {
		qs = qs.Filter("campaign_id", campaignID)
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
		ID:         newID(),
		UserID:     user.ID,
		Title:      req.Title,
		Body:       req.Body,
		Platform:   req.Platform,
		Status:     status,
		MediaURLs:  req.MediaURLs,
		CampaignID: req.CampaignID,
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
	if req.CampaignID != nil {
		content.CampaignID = *req.CampaignID
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

// ExportGenerations GET /api/contents/ai-generations/export?format=csv|json
func (c *ContentsController) ExportGenerations() {
	if !c.CheckPermission("content:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	format := c.GetQuery("format")
	if format == "" {
		format = "json"
	}
	qs := services.GetOrm().QueryTable(new(models.AIGenerationRecord)).
		Filter("user_id", user.ID)
	if platform := c.GetQuery("platform"); platform != "" {
		qs = qs.Filter("platform", platform)
	}
	if topic := c.GetQuery("topic"); topic != "" {
		qs = qs.Filter("topic__icontains", topic)
	}
	var records []models.AIGenerationRecord
	if _, err := qs.OrderBy("-created_at").All(&records); err != nil {
		c.WriteError(http.StatusInternalServerError, "导出 AI 生成记录失败")
		return
	}

	if format == "csv" {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"ID", "主题", "平台", "模型", "标题", "正文", "话题标签", "质量评分", "评分等级", "生成时间"})
		for i := range records {
			r := &records[i]
			hashtags := []string{}
			if r.Hashtags != "" {
				_ = json.Unmarshal([]byte(r.Hashtags), &hashtags)
			}
			score := services.ScoreVariant(services.AIVariant{Title: r.Title, Body: r.Body, Hashtags: hashtags}, r.Platform)
			_ = w.Write([]string{
				r.ID,
				r.Topic,
				platformLabels[r.Platform],
				r.Model,
				r.Title,
				r.Body,
				strings.Join(hashtags, " "),
				fmt.Sprintf("%d", score.Total),
				score.Comment,
				r.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		w.Flush()
		filename := fmt.Sprintf("ai-generations-%s.csv", time.Now().Format("20060102"))
		c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
		c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		_, _ = c.Ctx.ResponseWriter.Write(buf.Bytes())
		return
	}

	rows := make([]map[string]interface{}, 0, len(records))
	for i := range records {
		rows = append(rows, aiRecordMap(&records[i]))
	}
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	c.OK(rows)
}
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
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Topic == "" && len(req.Keywords) == 0 {
		c.WriteError(http.StatusBadRequest, "内容主题、关键词至少填写一项")
		return
	}
	count := req.Count
	if count <= 0 {
		count = 3
	}
	variants, err := services.GenerateContentVariants(req.Topic, req.Platform, req.Style, req.Keywords, count, req.PlanID, "", req.CampaignID)
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
			ID:         newID(),
			UserID:     user.ID,
			Topic:      req.Topic,
			Platform:   req.Platform,
			PlanID:     req.PlanID,
			Model:      modelName,
			Title:      v.Title,
			Body:       v.Body,
			Hashtags:   string(hashtagsJSON),
			CampaignID: req.CampaignID,
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
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Topic == "" && len(req.Keywords) == 0 && len(req.Files) == 0 {
		c.WriteError(http.StatusBadRequest, "内容主题、关键词、上传文件至少填写一项")
		return
	}
	if len(req.Platforms) == 0 {
		c.WriteError(http.StatusBadRequest, "请至少选择一个目标平台")
		return
	}
	if len(req.Files) > 0 && !services.ModelSupportsFiles(req.PlanID, req.ModelID) {
		c.WriteError(http.StatusBadRequest, "当前模型不支持文件上传，请切换到支持视觉/图片/视频的模型配置")
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
	versionNum := req.GenerateVersion
	if versionNum <= 0 {
		versionNum = 1
	}
	if versionNum > 3 {
		versionNum = 3
	}
	for _, platform := range req.Platforms {
		label := platformLabels[platform]
		if label == "" {
			label = platform
		}
		write("log", map[string]interface{}{
			"level":   "req",
			"message": fmt.Sprintf("开始为「%s」生成内容（%d 个版本）…", label, versionNum),
		})

		events, err := services.GenerateContentStream(req.Topic, platform, req.Style, req.Keywords, req.PlanID, req.ModelID, req.Files, req.CampaignID, versionNum)
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
						ID:         newID(),
						UserID:     user.ID,
						Topic:      req.Topic,
						Platform:   platform,
						PlanID:     req.PlanID,
						Model:      evt.Model,
						Title:      evt.Variant.Title,
						Body:       evt.Variant.Body,
						Hashtags:   marshalHashtags(evt.Variant.Hashtags),
						CampaignID: req.CampaignID,
					}
					_, _ = o.Insert(record)
					write("done", map[string]interface{}{
						"platform":      platform,
						"variant":       evt.Variant,
						"variant_index": evt.VariantIndex,
						"score":         evt.Score,
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
