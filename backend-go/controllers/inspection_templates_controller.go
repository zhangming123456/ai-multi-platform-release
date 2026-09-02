package controllers

import (
	"net/http"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type InspectionTemplatesController struct {
	BaseController
}

type inspectionTemplateItemRequest struct {
	Category                    string               `json:"category"`
	Title                       string               `json:"title"`
	Standard                    string               `json:"standard"`
	StandardImages              []string             `json:"standard_images"`
	ScoreType                   string               `json:"score_type"`
	MaxScore                    int                  `json:"max_score"`
	ScoreOptions                []models.ScoreOption `json:"score_options"`
	RequireRemark               bool                 `json:"require_remark"`
	RequirePhoto                bool                 `json:"require_photo"`
	ShowRemark                  bool                 `json:"show_remark"`
	ShowPhoto                   bool                 `json:"show_photo"`
	CategoryPreconditionEnabled bool                 `json:"category_precondition_enabled"`
	CategoryPrecondition        string               `json:"category_precondition"`
}

type inspectionTemplateCreateRequest struct {
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	IsActive    bool                            `json:"is_active"`
	ScoringMode string                          `json:"scoring_mode"`
	Items       []inspectionTemplateItemRequest `json:"items"`
}

type inspectionTemplateUpdateRequest struct {
	Name        *string                          `json:"name"`
	Description *string                          `json:"description"`
	IsActive    *bool                            `json:"is_active"`
	ScoringMode *string                          `json:"scoring_mode"`
	Items       *[]inspectionTemplateItemRequest `json:"items"`
}

// List GET /api/inspection-templates/
func (c *InspectionTemplatesController) List() {
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.InspectionTemplate))
	if keyword := strings.TrimSpace(c.GetQuery("keyword")); keyword != "" {
		cond := orm.NewCondition().
			Or("name__icontains", keyword).
			Or("description__icontains", keyword)
		qs = qs.SetCond(cond)
	}
	total, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询模板失败")
		return
	}
	var templates []models.InspectionTemplate
	if _, err := qs.OrderBy("-created_at").Limit(pageSize, (page-1)*pageSize).All(&templates); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询模板失败")
		return
	}
	items := buildTemplateViews(templates)
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListAll GET /api/inspection-templates/all
func (c *InspectionTemplatesController) ListAll() {
	templates, err := services.GetActiveTemplates()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询模板失败")
		return
	}
	items := buildTemplateViews(templates)
	c.OK(items)
}

// aiGenerateTemplateItemRequest AI 智能添加检查项请求。
type aiGenerateTemplateItemRequest struct {
	PlanID      string `json:"plan_id"`
	ModelID     string `json:"model_id"`
	Description string `json:"description"`
	Photos      []struct {
		Data     string `json:"data"`
		MimeType string `json:"mime_type"`
	} `json:"photos"`
	ExistingCategories []string `json:"existing_categories"`
}

// AIGenerateItems POST /api/inspection-templates/ai-generate-items
func (c *InspectionTemplatesController) AIGenerateItems() {
	if !c.CheckPermission("inspection:template:ai_create") {
		return
	}
	var req aiGenerateTemplateItemRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求参数错误")
		return
	}
	svcReq, errMsg := buildAIGenerateSvcReq(&req)
	if errMsg != "" {
		c.WriteError(http.StatusBadRequest, errMsg)
		return
	}
	items, err := services.GenerateTemplateItemsByAI(svcReq)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, err.Error())
		return
	}
	c.OK(map[string]interface{}{"items": items})
}

// AIGenerateItemsStream POST /api/inspection-templates/ai-generate-items-stream
func (c *InspectionTemplatesController) AIGenerateItemsStream() {
	if !c.CheckPermission("inspection:template:ai_create") {
		return
	}
	var req aiGenerateTemplateItemRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求参数错误")
		return
	}
	svcReq, errMsg := buildAIGenerateSvcReq(&req)
	if errMsg != "" {
		c.WriteError(http.StatusBadRequest, errMsg)
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

	write("log", map[string]interface{}{
		"level":   "req",
		"message": "开始 AI 智能添加检查项…",
	})

	events := make(chan services.AIStreamEvent, 32)
	go func() {
		defer close(events)
		services.GenerateTemplateItemsByAIStream(svcReq, events)
	}()

	for evt := range events {
		switch evt.Event {
		case "log":
			level := evt.Level
			if level == "" {
				level = "info"
			}
			write("log", map[string]interface{}{
				"level":   level,
				"message": evt.Message,
			})
		case "chunk":
			write("chunk", map[string]interface{}{
				"text": evt.Text,
			})
		case "result":
			write("result", evt.Data)
		case "error":
			write("error", map[string]interface{}{
				"message":         evt.Message,
				"available_plans": evt.AvailablePlans,
			})
		}
	}
	write("complete", map[string]interface{}{"message": "AI 智能添加检查项完成"})
}

func buildAIGenerateSvcReq(req *aiGenerateTemplateItemRequest) (services.AIGenerateTemplateItemRequest, string) {
	if strings.TrimSpace(req.PlanID) == "" || strings.TrimSpace(req.ModelID) == "" {
		return services.AIGenerateTemplateItemRequest{}, "请选择 AI 模型"
	}
	if strings.TrimSpace(req.Description) == "" && len(req.Photos) == 0 {
		return services.AIGenerateTemplateItemRequest{}, "请上传图片或填写文字描述"
	}
	svcReq := services.AIGenerateTemplateItemRequest{
		PlanID:             req.PlanID,
		ModelID:            req.ModelID,
		Description:        req.Description,
		ExistingCategories: req.ExistingCategories,
	}
	for _, p := range req.Photos {
		if strings.TrimSpace(p.Data) == "" {
			continue
		}
		mimeType := p.MimeType
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		svcReq.Photos = append(svcReq.Photos, services.UploadedFile{
			Data:     p.Data,
			MimeType: mimeType,
		})
	}
	return svcReq, ""
}

// Create POST /api/inspection-templates/
func (c *InspectionTemplatesController) Create() {
	if !c.CheckPermission("inspection:template:create:write") {
		return
	}
	var req inspectionTemplateCreateRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.WriteError(http.StatusBadRequest, "模板名称不能为空")
		return
	}
	template := &models.InspectionTemplate{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		ScoringMode: normalizeScoringMode(req.ScoringMode),
	}
	entries, err := normalizeTemplateItems(req.Items)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	if err := services.SaveInspectionTemplate(template, entries); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存模板失败")
		return
	}
	c.Created(templateWithItems(template))
}

// Get GET /api/inspection-templates/:template_id
func (c *InspectionTemplatesController) Get() {
	template, err := services.FindInspectionTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "模板不存在")
		return
	}
	c.OK(templateWithItems(template))
}

// Update PUT /api/inspection-templates/:template_id
func (c *InspectionTemplatesController) Update() {
	if !c.CheckPermission("inspection:template:update:write") {
		return
	}
	template, err := services.FindInspectionTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "模板不存在")
		return
	}
	var req inspectionTemplateUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Description != nil {
		template.Description = *req.Description
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}
	if req.ScoringMode != nil {
		template.ScoringMode = normalizeScoringMode(*req.ScoringMode)
	}
	entries, err := normalizeTemplateItemsFromPtr(req.Items)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	if err := services.SaveInspectionTemplate(template, entries); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存模板失败")
		return
	}
	c.OK(templateWithItems(template))
}

// Delete DELETE /api/inspection-templates/:template_id
func (c *InspectionTemplatesController) Delete() {
	if !c.CheckPermission("inspection:template:delete:write") {
		return
	}
	templateID := c.GetPathParam("template_id")
	if err := services.DeleteInspectionTemplate(templateID); err != nil {
		if err.Error() == "模板不存在" {
			c.WriteError(http.StatusNotFound, "模板不存在")
			return
		}
		c.WriteError(http.StatusConflict, err.Error())
		return
	}
	c.WriteNoContent()
}

func normalizeTemplateItems(requests []inspectionTemplateItemRequest) ([]*models.InspectionTemplateItem, error) {
	if requests == nil {
		return []*models.InspectionTemplateItem{}, nil
	}
	entries := make([]*models.InspectionTemplateItem, 0, len(requests))
	for _, r := range requests {
		scoreType := normalizeScoreType(r.ScoreType)
		options := r.ScoreOptions
		if len(options) == 0 {
			options = models.DefaultScoreOptions(scoreType, 10)
		}
		if scoreType == "pass_fail" {
			if err := services.ValidatePassFailOptions(options); err != nil {
				return nil, err
			}
		} else {
			if err := services.ValidateScoreOptions(options); err != nil {
				return nil, err
			}
		}
		var maxScore int
		if scoreType == "pass_fail" {
			if r.MaxScore > 0 {
				maxScore = r.MaxScore
			} else {
				maxScore = 5
			}
		} else {
			maxScore = int(maxOptionScore(options))
			if maxScore <= 0 {
				maxScore = r.MaxScore
			}
			if maxScore <= 0 {
				maxScore = 5
			}
		}
		images := r.StandardImages
		item := &models.InspectionTemplateItem{
			Category:                    r.Category,
			Title:                       r.Title,
			Standard:                    r.Standard,
			StandardImages:              models.MarshalStringSlice(images),
			ScoreType:                   scoreType,
			MaxScore:                    maxScore,
			ScoreOptions:                models.MarshalScoreOptions(options),
			RequireRemark:               r.RequireRemark,
			RequirePhoto:                r.RequirePhoto,
			ShowRemark:                  r.ShowRemark,
			ShowPhoto:                   r.ShowPhoto,
			CategoryPreconditionEnabled: r.CategoryPreconditionEnabled,
			CategoryPrecondition:        r.CategoryPrecondition,
		}
		item.StandardImage = firstImage(images)
		entries = append(entries, item)
	}
	return entries, nil
}

func normalizeTemplateItemsFromPtr(requests *[]inspectionTemplateItemRequest) ([]*models.InspectionTemplateItem, error) {
	if requests == nil {
		return nil, nil
	}
	return normalizeTemplateItems(*requests)
}

func maxOptionScore(options []models.ScoreOption) float64 {
	var max float64
	for _, opt := range options {
		if opt.Score > max {
			max = opt.Score
		}
	}
	return max
}

func normalizeScoreType(scoreType string) string {
	if scoreType == "pass_fail" {
		return "pass_fail"
	}
	return "score"
}

func normalizeScoringMode(mode string) string {
	if mode == "deductive" {
		return "deductive"
	}
	return "additive"
}

type inspectionTemplateView struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	IsActive    bool                         `json:"is_active"`
	ScoringMode string                       `json:"scoring_mode"`
	ItemCount   int                          `json:"item_count"`
	Items       []inspectionTemplateItemView `json:"items,omitempty"`
	CreatedAt   string                       `json:"created_at"`
	UpdatedAt   string                       `json:"updated_at"`
}

type inspectionTemplateItemView struct {
	ID                          string               `json:"id"`
	Category                    string               `json:"category"`
	Title                       string               `json:"title"`
	Standard                    string               `json:"standard"`
	StandardImages              []string             `json:"standard_images"`
	ScoreType                   string               `json:"score_type"`
	MaxScore                    int                  `json:"max_score"`
	ScoreOptions                []models.ScoreOption `json:"score_options"`
	RequireRemark               bool                 `json:"require_remark"`
	RequirePhoto                bool                 `json:"require_photo"`
	ShowRemark                  bool                 `json:"show_remark"`
	ShowPhoto                   bool                 `json:"show_photo"`
	CategoryPreconditionEnabled bool                 `json:"category_precondition_enabled"`
	CategoryPrecondition        string               `json:"category_precondition"`
}

func buildTemplateViews(templates []models.InspectionTemplate) []inspectionTemplateView {
	views := make([]inspectionTemplateView, 0, len(templates))
	for _, t := range templates {
		views = append(views, templateViewFromModel(&t))
	}
	return views
}

func templateViewFromModel(t *models.InspectionTemplate) inspectionTemplateView {
	count, _ := services.CountTemplateItems(t.ID)
	view := inspectionTemplateView{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		IsActive:    t.IsActive,
		ScoringMode: normalizeScoringMode(t.ScoringMode),
		ItemCount:   int(count),
		CreatedAt:   t.CreatedAt.Format(TimeFormat),
		UpdatedAt:   t.UpdatedAt.Format(TimeFormat),
	}
	return view
}

func templateWithItems(t *models.InspectionTemplate) map[string]interface{} {
	items, _ := services.GetInspectionTemplateItems(t.ID)
	itemViews := make([]inspectionTemplateItemView, 0, len(items))
	for _, it := range items {
		images := effectiveStandardImages(it.StandardImages, it.StandardImage)
		itemViews = append(itemViews, inspectionTemplateItemView{
			ID:                          it.ID,
			Category:                    it.Category,
			Title:                       it.Title,
			Standard:                    it.Standard,
			StandardImages:              images,
			ScoreType:                   it.ScoreType,
			MaxScore:                    it.MaxScore,
			ScoreOptions:                services.EffectiveScoreOptions(&it),
			RequireRemark:               it.RequireRemark,
			RequirePhoto:                it.RequirePhoto,
			ShowRemark:                  it.ShowRemark,
			ShowPhoto:                   it.ShowPhoto,
			CategoryPreconditionEnabled: it.CategoryPreconditionEnabled,
			CategoryPrecondition:        it.CategoryPrecondition,
		})
	}
	view := map[string]interface{}{
		"id":           t.ID,
		"name":         t.Name,
		"description":  t.Description,
		"is_active":    t.IsActive,
		"scoring_mode": normalizeScoringMode(t.ScoringMode),
		"items":        itemViews,
		"created_at":   t.CreatedAt.Format(TimeFormat),
		"updated_at":   t.UpdatedAt.Format(TimeFormat),
	}
	return view
}
