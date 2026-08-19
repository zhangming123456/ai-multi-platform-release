package controllers

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type InspectionsController struct {
	BaseController
}

type inspectionScoreRequest struct {
	ItemID       string   `json:"item_id"`
	Score        float64  `json:"score"`
	Comment      string   `json:"comment"`
	AIGenerated  bool     `json:"ai_generated"`
	AISuggestion string   `json:"ai_suggestion"`
	Photos       []string `json:"photos"`
}

type inspectionCreateRequest struct {
	StoreID     string                       `json:"store_id"`
	TemplateID  string                       `json:"template_id"`
	Title       string                       `json:"title"`
	Status      string                       `json:"status"`
	CheckedAt   string                       `json:"checked_at"`
	Issues      string                       `json:"issues"`
	Suggestion  string                       `json:"suggestion"`
	AISummary   services.InspectionAISummary `json:"ai_summary"`
	AIGenerated bool                         `json:"ai_generated"`
	Photos      []string                     `json:"photos"`
	Scores      []inspectionScoreRequest     `json:"scores"`
}

type inspectionUpdateRequest struct {
	StoreID     *string                       `json:"store_id"`
	TemplateID  *string                       `json:"template_id"`
	Title       *string                       `json:"title"`
	Status      *string                       `json:"status"`
	CheckedAt   *string                       `json:"checked_at"`
	Issues      *string                       `json:"issues"`
	Suggestion  *string                       `json:"suggestion"`
	AISummary   *services.InspectionAISummary `json:"ai_summary"`
	AIGenerated *bool                         `json:"ai_generated"`
	Photos      *[]string                     `json:"photos"`
	Scores      *[]inspectionScoreRequest     `json:"scores"`
}

const inspectionPassRatio = 0.8

// List GET /api/inspections/
func (c *InspectionsController) List() {
	if !c.CheckPermission("inspection:read") {
		return
	}
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.Inspection))
	if storeID := c.GetQuery("store_id"); storeID != "" {
		qs = qs.Filter("store_id", storeID)
	}
	if status := c.GetQuery("status"); status != "" {
		qs = qs.Filter("status", status)
	}
	total, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询巡店记录失败")
		return
	}
	var inspections []models.Inspection
	if _, err := qs.OrderBy("-checked_at").Limit(pageSize, (page-1)*pageSize).All(&inspections); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询巡店记录失败")
		return
	}
	items := buildInspectionViews(inspections)
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListItems GET /api/inspections/items
func (c *InspectionsController) ListItems() {
	if !c.CheckPermission("inspection:read") {
		return
	}
	items, err := services.GetActiveInspectionItems()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询检查项失败")
		return
	}
	c.OK(items)
}

// Create POST /api/inspections/
func (c *InspectionsController) Create() {
	if !c.CheckPermission("inspection:create:write") {
		return
	}
	var req inspectionCreateRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.StoreID) == "" {
		c.WriteError(http.StatusBadRequest, "请选择巡店门店")
		return
	}
	if _, err := findStore(req.StoreID); err != nil {
		c.WriteError(http.StatusBadRequest, "门店不存在")
		return
	}
	templateName := ""
	if req.TemplateID != "" {
		template, err := services.FindInspectionTemplate(req.TemplateID)
		if err != nil {
			c.WriteError(http.StatusBadRequest, "检查表模板不存在")
			return
		}
		templateName = template.Name
	}
	user := c.CurrentUser()
	checkedAt, err := parseCheckedAt(req.CheckedAt)
	if err != nil {
		c.WriteError(http.StatusBadRequest, "检查时间格式错误")
		return
	}
	status := normalizeInspectionStatus(req.Status)
	aiSummaryJSON := ""
	if raw, err := json.Marshal(req.AISummary); err == nil {
		aiSummaryJSON = string(raw)
	}
	inspection := &models.Inspection{
		ID:           newID(),
		StoreID:      req.StoreID,
		TemplateID:   req.TemplateID,
		TemplateName: templateName,
		Title:        req.Title,
		Status:       status,
		Issues:       req.Issues,
		Suggestion:   req.Suggestion,
		AISummary:    aiSummaryJSON,
		AIGenerated:  req.AIGenerated,
		Photos:       marshalPhotos(req.Photos),
		InspectorID:  user.ID,
		CheckedAt:    checkedAt,
	}
	scoreRows, err := buildScoreRows(inspection, req.Scores)
	if err != nil {
		if strings.HasPrefix(err.Error(), "validation:") {
			c.WriteError(http.StatusBadRequest, strings.TrimPrefix(err.Error(), "validation:"))
			return
		}
		if err.Error() == "template_required" {
			c.WriteError(http.StatusBadRequest, "该模板无可用检查项，请先完善模板")
			return
		}
		c.WriteError(http.StatusInternalServerError, "保存巡店记录失败")
		return
	}
	if err := saveInspectionWithScores(inspection, scoreRows); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存巡店记录失败")
		return
	}
	if status != models.InspectionStatusDraft {
		autoCreateInspectionTask(inspection)
	}
	c.Created(inspectionWithScores(inspection))
}

// Get GET /api/inspections/:inspection_id
func (c *InspectionsController) Get() {
	if !c.CheckPermission("inspection:read") {
		return
	}
	inspection, err := findInspection(c.GetPathParam("inspection_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "巡店记录不存在")
		return
	}
	c.OK(inspectionWithScores(inspection))
}

// Update PUT /api/inspections/:inspection_id
func (c *InspectionsController) Update() {
	if !c.CheckPermission("inspection:update:write") {
		return
	}
	inspection, err := findInspection(c.GetPathParam("inspection_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "巡店记录不存在")
		return
	}
	if !models.InspectionStatusEditable(inspection.Status) {
		c.WriteError(http.StatusBadRequest, "当前巡店已进入整改流程，不可编辑")
		return
	}
	var req inspectionUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.StoreID != nil {
		if _, err := findStore(*req.StoreID); err != nil {
			c.WriteError(http.StatusBadRequest, "门店不存在")
			return
		}
		inspection.StoreID = *req.StoreID
	}
	if req.TemplateID != nil {
		if *req.TemplateID == "" {
			inspection.TemplateID = ""
			inspection.TemplateName = ""
		} else if template, err := services.FindInspectionTemplate(*req.TemplateID); err != nil {
			c.WriteError(http.StatusBadRequest, "检查表模板不存在")
			return
		} else {
			inspection.TemplateID = template.ID
			inspection.TemplateName = template.Name
		}
	}
	if req.Title != nil {
		inspection.Title = *req.Title
	}
	if req.Status != nil {
		inspection.Status = normalizeInspectionStatus(*req.Status)
	}
	if req.CheckedAt != nil {
		checkedAt, err := parseCheckedAt(*req.CheckedAt)
		if err != nil {
			c.WriteError(http.StatusBadRequest, "检查时间格式错误")
			return
		}
		inspection.CheckedAt = checkedAt
	}
	if req.Issues != nil {
		inspection.Issues = *req.Issues
	}
	if req.Suggestion != nil {
		inspection.Suggestion = *req.Suggestion
	}
	if req.AISummary != nil {
		if raw, err := json.Marshal(*req.AISummary); err == nil {
			inspection.AISummary = string(raw)
		}
	}
	if req.AIGenerated != nil {
		inspection.AIGenerated = *req.AIGenerated
	}
	if req.Photos != nil {
		inspection.Photos = marshalPhotos(*req.Photos)
	}
	scores := inspectionCurrentScores(inspection)
	if req.Scores != nil {
		scores = *req.Scores
	}
	scoreRows, err := buildScoreRows(inspection, scores)
	if err != nil {
		if strings.HasPrefix(err.Error(), "validation:") {
			c.WriteError(http.StatusBadRequest, strings.TrimPrefix(err.Error(), "validation:"))
			return
		}
		if err.Error() == "template_required" {
			c.WriteError(http.StatusBadRequest, "该模板无可用检查项，请先完善模板")
			return
		}
		c.WriteError(http.StatusInternalServerError, "保存巡店记录失败")
		return
	}
	if err := updateInspectionWithScores(inspection, scoreRows); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存巡店记录失败")
		return
	}
	autoCreateInspectionTask(inspection)
	c.OK(inspectionWithScores(inspection))
}

// Delete DELETE /api/inspections/:inspection_id
func (c *InspectionsController) Delete() {
	if !c.CheckPermission("inspection:delete:write") {
		return
	}
	inspection, err := findInspection(c.GetPathParam("inspection_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "巡店记录不存在")
		return
	}
	o := services.GetOrm()
	if _, err := o.QueryTable(new(models.InspectionScore)).
		Filter("inspection_id", inspection.ID).
		Delete(); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除巡店记录失败")
		return
	}
	if _, err := o.Delete(inspection); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除巡店记录失败")
		return
	}
	c.WriteNoContent()
}

// AIAnalyze POST /api/inspections/ai-analyze
func (c *InspectionsController) AIAnalyze() {
	if !c.CheckPermission("inspection:ai:write") {
		return
	}
	var req struct {
		StoreID        string                     `json:"store_id"`
		TemplateID     string                     `json:"template_id"`
		PlanID         string                     `json:"plan_id"`
		ModelID        string                     `json:"model_id"`
		Photos         []services.UploadedFile    `json:"photos"`
		Keywords       string                     `json:"keywords"`
		Skills         []services.InspectionSkill `json:"skills"`
		ResponseSchema map[string]interface{}     `json:"response_schema"`
	}
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	storeName := ""
	if req.StoreID != "" {
		if store, err := findStore(req.StoreID); err == nil {
			storeName = store.Name
		}
	}
	aiItems, err := loadAIItems(req.TemplateID)
	if err != nil {
		if err.Error() == "template_required" {
			c.WriteError(http.StatusBadRequest, "所选检查表模板无可用检查项，请先完善模板")
			return
		}
		c.WriteError(http.StatusInternalServerError, "获取检查项失败")
		return
	}
	var skillSpec *services.InspectionSkillSpec
	if len(req.Skills) > 0 {
		skillSpec = &services.InspectionSkillSpec{
			Skills:         req.Skills,
			ResponseSchema: req.ResponseSchema,
		}
	}
	result, err := services.AnalyzeInspection(storeName, aiItems, req.Photos, req.Keywords, req.PlanID, req.ModelID, skillSpec)
	if err != nil {
		if aiErr, ok := err.(*services.AIGenerationError); ok {
			c.WriteJSON(http.StatusBadRequest, map[string]interface{}{
				"detail":          aiErr.Message,
				"available_plans": aiErr.AvailablePlans,
			})
			return
		}
		c.WriteError(http.StatusInternalServerError, "AI 巡店分析失败")
		return
	}
	scores := make([]map[string]interface{}, 0, len(aiItems))
	for _, it := range aiItems {
		score := 0.0
		if v, ok := result.Scores[it.ID]; ok {
			if len(it.ScoreOptions) > 0 {
				score = quantizeScore(v, it.ScoreOptions)
			} else {
				score = clampScore(v, it.MaxScore)
			}
		}
		comment := ""
		if result.Comments != nil {
			comment = result.Comments[it.ID]
		}
		scores = append(scores, map[string]interface{}{
			"item_id":       it.ID,
			"item_name":     it.Name,
			"category":      "",
			"standard":      it.Standard,
			"score_type":    it.ScoreType,
			"max_score":     it.MaxScore,
			"score_options": it.ScoreOptions,
			"score":         score,
			"comment":       comment,
		})
	}
	totalScore := 0.0
	for _, s := range scores {
		if s["score_type"] == "pass_fail" {
			continue
		}
		totalScore += s["score"].(float64)
	}
	c.OK(map[string]interface{}{
		"store_id":    req.StoreID,
		"store_name":  storeName,
		"template_id": req.TemplateID,
		"scores":      scores,
		"total_score": totalScore,
		"issues":      result.Issues,
		"suggestion":  result.Suggestion,
	})
}

// AIAnalyzeStream POST /api/inspections/ai-analyze-stream
func (c *InspectionsController) AIAnalyzeStream() {
	if !c.CheckPermission("inspection:ai:write") {
		return
	}
	var req struct {
		StoreID        string                     `json:"store_id"`
		TemplateID     string                     `json:"template_id"`
		PlanID         string                     `json:"plan_id"`
		ModelID        string                     `json:"model_id"`
		Photos         []services.UploadedFile    `json:"photos"`
		Keywords       string                     `json:"keywords"`
		Skills         []services.InspectionSkill `json:"skills"`
		ResponseSchema map[string]interface{}     `json:"response_schema"`
	}
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	storeName := ""
	if req.StoreID != "" {
		if store, err := findStore(req.StoreID); err == nil {
			storeName = store.Name
		}
	}
	aiItems, err := loadAIItems(req.TemplateID)
	if err != nil {
		if err.Error() == "template_required" {
			c.WriteError(http.StatusBadRequest, "所选检查表模板无可用检查项，请先完善模板")
			return
		}
		c.WriteError(http.StatusInternalServerError, "获取检查项失败")
		return
	}
	var skillSpec *services.InspectionSkillSpec
	if len(req.Skills) > 0 {
		skillSpec = &services.InspectionSkillSpec{
			Skills:         req.Skills,
			ResponseSchema: req.ResponseSchema,
		}
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
		"message": "开始 AI 巡店分析…",
	})

	events, err := services.AnalyzeInspectionStream(storeName, aiItems, req.Photos, req.Keywords, req.PlanID, req.ModelID, skillSpec)
	if err != nil {
		write("error", map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

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
		case "request_data", "response_data":
			write(evt.Event, map[string]interface{}{
				"message": evt.Message,
				"detail":  evt.Detail,
			})
		case "result":
			data := map[string]interface{}{
				"issues":     "",
				"suggestion": "",
				"scores":     []interface{}{},
			}
			if m, ok := evt.Data.(map[string]interface{}); ok {
				data = m
			}
			write("result", data)
		case "error":
			write("error", map[string]interface{}{
				"message":         evt.Message,
				"available_plans": evt.AvailablePlans,
			})
		}
	}
	write("complete", map[string]interface{}{"message": "AI 巡店分析结束"})
}

// AIAnalyzeItem POST /api/inspections/ai-analyze-item
func (c *InspectionsController) AIAnalyzeItem() {
	if !c.CheckPermission("inspection:ai:write") {
		return
	}
	var req services.SingleItemAnalysisRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.ItemName == "" {
		c.WriteError(http.StatusBadRequest, "检查项名称不能为空")
		return
	}
	result, err := services.AnalyzeSingleInspectionItem(req)
	if err != nil {
		if aiErr, ok := err.(*services.AIGenerationError); ok {
			c.WriteJSON(http.StatusBadRequest, map[string]interface{}{
				"detail":          aiErr.Message,
				"available_plans": aiErr.AvailablePlans,
			})
			return
		}
		c.WriteError(http.StatusInternalServerError, "AI 单项分析失败")
		return
	}
	c.WriteJSON(http.StatusOK, map[string]interface{}{
		"score":      result.Score,
		"comment":    result.Comment,
		"suggestion": result.Suggestion,
	})
}

// AIAnalyzeItemStream POST /api/inspections/ai-analyze-item-stream
func (c *InspectionsController) AIAnalyzeItemStream() {
	if !c.CheckPermission("inspection:ai:write") {
		return
	}
	var req services.SingleItemAnalysisRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.ItemName == "" {
		c.WriteError(http.StatusBadRequest, "检查项名称不能为空")
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

	events, err := services.AnalyzeSingleInspectionItemStream(req)
	if err != nil {
		write("error", map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

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
		case "request_data", "response_data":
			write(evt.Event, map[string]interface{}{
				"message": evt.Message,
				"detail":  evt.Detail,
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
	write("complete", map[string]interface{}{"message": "AI 分析结束"})
}

func loadAIItems(templateID string) ([]services.InspectionAIItem, error) {
	if templateID != "" {
		items, err := services.GetInspectionTemplateItems(templateID)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, errors.New("template_required")
		}
		aiItems := make([]services.InspectionAIItem, 0, len(items))
		for _, it := range items {
			aiItems = append(aiItems, services.InspectionAIItem{
				ID:             it.ID,
				Name:           it.Title,
				Standard:       it.Standard,
				StandardImages: effectiveStandardImages(it.StandardImages, it.StandardImage),
				ScoreType:      it.ScoreType,
				MaxScore:       it.MaxScore,
				ScoreOptions:   services.EffectiveScoreOptions(&it),
			})
		}
		return aiItems, nil
	}
	items, err := services.GetActiveInspectionItems()
	if err != nil {
		return nil, err
	}
	aiItems := make([]services.InspectionAIItem, 0, len(items))
	for _, it := range items {
		aiItems = append(aiItems, services.InspectionAIItem{
			ID:           it.ID,
			Name:         it.Name,
			ScoreType:    "score",
			MaxScore:     it.MaxScore,
			ScoreOptions: models.DefaultScoreOptions("score", it.MaxScore),
		})
	}
	return aiItems, nil
}

func clampScore(v float64, maxScore int) float64 {
	if v < 0 {
		return 0
	}
	if v > float64(maxScore) {
		return float64(maxScore)
	}
	return v
}

func quantizeScore(v float64, options []models.ScoreOption) float64 {
	if len(options) == 0 {
		return v
	}
	best := options[0]
	bestDiff := math.Abs(v - float64(best.Score))
	for _, opt := range options[1:] {
		diff := math.Abs(v - float64(opt.Score))
		if diff < bestDiff {
			best = opt
			bestDiff = diff
		}
	}
	return float64(best.Score)
}

func effectiveSnapshotScoreOptions(scoreType string, maxScore int, raw string) []models.ScoreOption {
	options := models.UnmarshalScoreOptions(raw)
	if len(options) == 0 {
		return models.DefaultScoreOptions(scoreType, maxScore)
	}
	return options
}

func parseCheckedAt(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local)
	if err != nil {
		t2, err2 := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
		if err2 != nil {
			return time.Now(), err
		}
		return t2, nil
	}
	return t, nil
}

func findInspection(inspectionID string) (*models.Inspection, error) {
	var inspection models.Inspection
	err := services.GetOrm().QueryTable(new(models.Inspection)).
		Filter("id", inspectionID).
		One(&inspection)
	if err != nil {
		return nil, err
	}
	return &inspection, nil
}

func inspectionCurrentScores(inspection *models.Inspection) []inspectionScoreRequest {
	var list []models.InspectionScore
	if _, err := services.GetOrm().QueryTable(new(models.InspectionScore)).
		Filter("inspection_id", inspection.ID).
		OrderBy("created_at").
		All(&list); err != nil {
		return nil
	}
	result := make([]inspectionScoreRequest, 0, len(list))
	for _, s := range list {
		result = append(result, inspectionScoreRequest{
			ItemID:       s.ItemID,
			Score:        s.Score,
			Comment:      s.Comment,
			AIGenerated:  s.AIGenerated,
			AISuggestion: s.AISuggestion,
			Photos:       unmarshalPhotos(s.Photos),
		})
	}
	return result
}

func saveInspectionWithScores(inspection *models.Inspection, scoreRows []*models.InspectionScore) error {
	o := services.GetOrm()
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Insert(inspection); err != nil {
		tx.Rollback()
		return err
	}
	for _, s := range scoreRows {
		if _, err := tx.Insert(s); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func updateInspectionWithScores(inspection *models.Inspection, scoreRows []*models.InspectionScore) error {
	o := services.GetOrm()
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Update(inspection); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.QueryTable(new(models.InspectionScore)).
		Filter("inspection_id", inspection.ID).
		Delete(); err != nil {
		tx.Rollback()
		return err
	}
	for _, s := range scoreRows {
		if _, err := tx.Insert(s); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type scoreItemSource struct {
	ID             string
	Name           string
	Category       string
	Standard       string
	StandardImages []string
	ScoreType      string
	MaxScore       int
	ScoreOptions   []models.ScoreOption
	RequireRemark  bool
	RequirePhoto   bool
	ShowRemark     bool
	ShowPhoto      bool
}

func loadScoreItemSources(templateID string) ([]scoreItemSource, error) {
	if templateID != "" {
		items, err := services.GetInspectionTemplateItems(templateID)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, errors.New("template_required")
		}
		sources := make([]scoreItemSource, 0, len(items))
		for _, it := range items {
			sources = append(sources, scoreItemSource{
				ID:             it.ID,
				Name:           it.Title,
				Category:       it.Category,
				Standard:       it.Standard,
				StandardImages: effectiveStandardImages(it.StandardImages, it.StandardImage),
				ScoreType:      it.ScoreType,
				MaxScore:       it.MaxScore,
				ScoreOptions:   services.EffectiveScoreOptions(&it),
				RequireRemark:  it.RequireRemark,
				RequirePhoto:   it.RequirePhoto,
				ShowRemark:     it.ShowRemark,
				ShowPhoto:      it.ShowPhoto,
			})
		}
		return sources, nil
	}
	items, err := services.GetActiveInspectionItems()
	if err != nil {
		return nil, err
	}
	sources := make([]scoreItemSource, 0, len(items))
	for _, it := range items {
		sources = append(sources, scoreItemSource{
			ID:           it.ID,
			Name:         it.Name,
			Category:     it.Category,
			ScoreType:    "score",
			MaxScore:     it.MaxScore,
			ScoreOptions: models.DefaultScoreOptions("score", it.MaxScore),
			ShowRemark:   true,
			ShowPhoto:    true,
		})
	}
	return sources, nil
}

func buildScoreRows(inspection *models.Inspection, reqScores []inspectionScoreRequest) ([]*models.InspectionScore, error) {
	sources, err := loadScoreItemSources(inspection.TemplateID)
	if err != nil {
		return nil, err
	}
	if inspection.ID != "" {
		var existing []models.InspectionScore
		if _, err := services.GetOrm().QueryTable(new(models.InspectionScore)).
			Filter("inspection_id", inspection.ID).
			All(&existing); err == nil {
			snapshotByID := make(map[string]models.InspectionScore, len(existing))
			for _, s := range existing {
				snapshotByID[s.ItemID] = s
			}
			for i := range sources {
				if snap, ok := snapshotByID[sources[i].ID]; ok {
					sources[i].ScoreType = snap.ScoreType
					sources[i].MaxScore = snap.MaxScore
					opts := models.UnmarshalScoreOptions(snap.ScoreOptions)
					if len(opts) > 0 {
						sources[i].ScoreOptions = opts
					} else {
						sources[i].ScoreOptions = nil
					}
				}
			}
		}
	}
	itemByID := make(map[string]scoreItemSource, len(sources))
	for _, it := range sources {
		itemByID[it.ID] = it
	}
	now := time.Now()
	rows := make([]*models.InspectionScore, 0, len(reqScores))
	for _, rs := range reqScores {
		item, ok := itemByID[rs.ItemID]
		if !ok {
			continue
		}
		score := rs.Score
		if item.ScoreType == "pass_fail" {
			if len(item.ScoreOptions) > 0 {
				score = quantizeScore(score, item.ScoreOptions)
			}
		} else {
			if len(item.ScoreOptions) > 0 {
				score = quantizeScore(score, item.ScoreOptions)
			}
			score = clampScore(score, item.MaxScore)
			if !isFullScore(score, item.MaxScore) {
				if item.ShowRemark && item.RequireRemark && strings.TrimSpace(rs.Comment) == "" {
					return nil, errors.New("validation:「" + item.Name + "」未得满分，请填写问题描述")
				}
				if item.ShowPhoto && item.RequirePhoto && len(rs.Photos) == 0 {
					return nil, errors.New("validation:「" + item.Name + "」未得满分，请上传巡店图片")
				}
			}
		}
		rows = append(rows, &models.InspectionScore{
			ID:             newID(),
			InspectionID:   inspection.ID,
			ItemID:         item.ID,
			ItemName:       item.Name,
			Category:       item.Category,
			Standard:       item.Standard,
			StandardImages: models.MarshalStringSlice(item.StandardImages),
			StandardImage:  firstImage(item.StandardImages),
			ScoreType:      item.ScoreType,
			MaxScore:       item.MaxScore,
			ScoreOptions:   models.MarshalScoreOptions(item.ScoreOptions),
			Score:          score,
			RequireRemark:  item.RequireRemark,
			RequirePhoto:   item.RequirePhoto,
			ShowRemark:     item.ShowRemark,
			ShowPhoto:      item.ShowPhoto,
			Comment:        rs.Comment,
			AIGenerated:    rs.AIGenerated,
			AISuggestion:   rs.AISuggestion,
			Photos:         marshalPhotos(rs.Photos),
			CreatedAt:      now,
		})
	}
	if inspection.TemplateID != "" && len(rows) == 0 {
		return nil, errors.New("template_required")
	}
	totalScore := 0.0
	maxTotal := 0
	for _, s := range rows {
		if s.ScoreType == "pass_fail" {
			continue
		}
		totalScore += s.Score
		maxTotal += s.MaxScore
	}
	inspection.TotalScore = totalScore
	if maxTotal > 0 {
		inspection.Passed = totalScore >= float64(maxTotal)*inspectionPassRatio
	}
	return rows, nil
}

func isFullScore(score float64, maxScore int) bool {
	if maxScore <= 0 {
		return true
	}
	return score >= float64(maxScore)
}

func marshalPhotos(photos []string) string {
	if photos == nil {
		return "[]"
	}
	b, _ := json.Marshal(photos)
	return string(b)
}

func unmarshalPhotos(raw string) []string {
	var photos []string
	if raw == "" {
		return []string{}
	}
	_ = json.Unmarshal([]byte(raw), &photos)
	if photos == nil {
		return []string{}
	}
	return photos
}

type inspectionView struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Status        string  `json:"status"`
	StoreID       string  `json:"store_id"`
	StoreName     string  `json:"store_name"`
	StoreCode     string  `json:"store_code"`
	TemplateID    string  `json:"template_id"`
	TemplateName  string  `json:"template_name"`
	InspectorID   string  `json:"inspector_id"`
	InspectorName string  `json:"inspector_name"`
	TotalScore    float64 `json:"total_score"`
	Passed        bool    `json:"passed"`
	AIGenerated   bool    `json:"ai_generated"`
	CheckedAt     string  `json:"checked_at"`
	CreatedAt     string  `json:"created_at"`
}

func buildInspectionViews(inspections []models.Inspection) []inspectionView {
	o := services.GetOrm()
	storeIDs := make([]string, 0, len(inspections))
	userIDs := make([]string, 0, len(inspections))
	for _, ins := range inspections {
		storeIDs = append(storeIDs, ins.StoreID)
		userIDs = append(userIDs, ins.InspectorID)
	}
	var stores []models.Store
	if len(storeIDs) > 0 {
		if _, err := o.QueryTable(new(models.Store)).Filter("id__in", storeIDs).All(&stores); err != nil {
			stores = nil
		}
	}
	storeByName := make(map[string]models.Store, len(stores))
	for _, s := range stores {
		storeByName[s.ID] = s
	}
	var users []models.User
	if len(userIDs) > 0 {
		if _, err := o.QueryTable(new(models.User)).Filter("id__in", userIDs).All(&users); err != nil {
			users = nil
		}
	}
	userByID := make(map[string]models.User, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}
	timeFormat := "2006-01-02 15:04:05"
	result := make([]inspectionView, 0, len(inspections))
	for _, ins := range inspections {
		view := inspectionView{
			ID:           ins.ID,
			Title:        ins.Title,
			Status:       ins.Status,
			StoreID:      ins.StoreID,
			TemplateID:   ins.TemplateID,
			TemplateName: ins.TemplateName,
			InspectorID:  ins.InspectorID,
			TotalScore:   ins.TotalScore,
			Passed:       ins.Passed,
			AIGenerated:  ins.AIGenerated,
			CheckedAt:    ins.CheckedAt.Format(timeFormat),
			CreatedAt:    ins.CreatedAt.Format(timeFormat),
		}
		if store, ok := storeByName[ins.StoreID]; ok {
			view.StoreName = store.Name
			view.StoreCode = store.Code
		}
		if u, ok := userByID[ins.InspectorID]; ok {
			view.InspectorName = u.Nickname
		}
		result = append(result, view)
	}
	return result
}

func inspectionWithScores(inspection *models.Inspection) map[string]interface{} {
	o := services.GetOrm()
	var scores []models.InspectionScore
	if _, err := o.QueryTable(new(models.InspectionScore)).
		Filter("inspection_id", inspection.ID).
		OrderBy("created_at").
		All(&scores); err != nil {
		scores = nil
	}
	scoreList := make([]map[string]interface{}, 0, len(scores))
	for _, s := range scores {
		scoreList = append(scoreList, map[string]interface{}{
			"item_id":         s.ItemID,
			"item_name":       s.ItemName,
			"category":        s.Category,
			"standard":        s.Standard,
			"standard_images": effectiveStandardImages(s.StandardImages, s.StandardImage),
			"score_type":      s.ScoreType,
			"max_score":       s.MaxScore,
			"score_options":   effectiveSnapshotScoreOptions(s.ScoreType, s.MaxScore, s.ScoreOptions),
			"score":           s.Score,
			"require_remark":  s.RequireRemark,
			"require_photo":   s.RequirePhoto,
			"show_remark":     s.ShowRemark,
			"show_photo":      s.ShowPhoto,
			"comment":         s.Comment,
			"ai_generated":    s.AIGenerated,
			"ai_suggestion":   s.AISuggestion,
			"photos":          unmarshalPhotos(s.Photos),
		})
	}
	aiSummary := services.ParseInspectionAISummary(inspection.AISummary)
	view := map[string]interface{}{
		"id":            inspection.ID,
		"title":         inspection.Title,
		"status":        inspection.Status,
		"store_id":      inspection.StoreID,
		"template_id":   inspection.TemplateID,
		"template_name": inspection.TemplateName,
		"inspector_id":  inspection.InspectorID,
		"total_score":   inspection.TotalScore,
		"passed":        inspection.Passed,
		"issues":        inspection.Issues,
		"suggestion":    inspection.Suggestion,
		"ai_summary":    aiSummary,
		"ai_generated":  inspection.AIGenerated,
		"photos":        unmarshalPhotos(inspection.Photos),
		"scores":        scoreList,
		"checked_at":    inspection.CheckedAt.Format("2006-01-02T15:04:05"),
		"created_at":    inspection.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":    inspection.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	store, err := findStore(inspection.StoreID)
	if err == nil {
		view["store_name"] = store.Name
		view["store_code"] = store.Code
		view["store_address"] = store.Address
	} else {
		view["store_name"] = ""
		view["store_code"] = ""
		view["store_address"] = ""
	}
	user := loadUserByID(inspection.InspectorID)
	view["inspector_name"] = user
	return view
}

func loadUserByID(userID string) string {
	var user models.User
	if err := services.GetOrm().QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		return ""
	}
	if user.Nickname != "" {
		return user.Nickname
	}
	return user.Username
}

// normalizeInspectionStatus 统一前端传入的状态值：completed（旧值）兼容映射为 pending。
func normalizeInspectionStatus(status string) string {
	switch status {
	case models.InspectionStatusDraft, models.InspectionStatusPending,
		models.InspectionStatusRectifying, models.InspectionStatusClosed:
		return status
	case "completed":
		return models.InspectionStatusPending
	default:
		return models.InspectionStatusDraft
	}
}
