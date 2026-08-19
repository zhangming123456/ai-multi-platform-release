package controllers

import (
	"net/http"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type InspectionMaterialsController struct {
	BaseController
}

type inspectionMaterialRequest struct {
	Category       string               `json:"category"`
	Title          string               `json:"title"`
	Standard       string               `json:"standard"`
	StandardImages []string             `json:"standard_images"`
	ScoreType      string               `json:"score_type"`
	MaxScore       int                  `json:"max_score"`
	ScoreOptions   []models.ScoreOption `json:"score_options"`
}

type inspectionMaterialView struct {
	ID             string               `json:"id"`
	Category       string               `json:"category"`
	Title          string               `json:"title"`
	Standard       string               `json:"standard"`
	StandardImages []string             `json:"standard_images"`
	ScoreType      string               `json:"score_type"`
	MaxScore       int                  `json:"max_score"`
	ScoreOptions   []models.ScoreOption `json:"score_options"`
	IsActive       bool                 `json:"is_active"`
	CreatedAt      string               `json:"created_at"`
	UpdatedAt      string               `json:"updated_at"`
}

// List GET /api/inspection-materials
func (c *InspectionMaterialsController) List() {
	if !c.CheckPermission("inspection:material:read") {
		return
	}
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.InspectionMaterial)).Filter("is_active", true)
	if keyword := strings.TrimSpace(c.GetQuery("keyword")); keyword != "" {
		cond := orm.NewCondition().
			Or("title__icontains", keyword).
			Or("category__icontains", keyword).
			Or("standard__icontains", keyword)
		qs = qs.SetCond(cond)
	}
	total, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询素材失败")
		return
	}
	var materials []models.InspectionMaterial
	if _, err := qs.OrderBy("-created_at").Limit(pageSize, (page-1)*pageSize).All(&materials); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询素材失败")
		return
	}
	items := make([]inspectionMaterialView, 0, len(materials))
	for _, m := range materials {
		items = append(items, materialViewFromModel(&m))
	}
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Create POST /api/inspection-materials
func (c *InspectionMaterialsController) Create() {
	if !c.CheckPermission("inspection:material:create:write") {
		return
	}
	var req inspectionMaterialRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		c.WriteError(http.StatusBadRequest, "素材标题不能为空")
		return
	}
	material := materialFromRequest(&req)
	if err := services.SaveInspectionMaterial(material); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存素材失败")
		return
	}
	c.Created(materialViewFromModel(material))
}

// Get GET /api/inspection-materials/:material_id
func (c *InspectionMaterialsController) Get() {
	if !c.CheckPermission("inspection:material:read") {
		return
	}
	material, err := services.FindInspectionMaterial(c.GetPathParam("material_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "素材不存在")
		return
	}
	c.OK(materialViewFromModel(material))
}

// Update PUT /api/inspection-materials/:material_id
func (c *InspectionMaterialsController) Update() {
	if !c.CheckPermission("inspection:material:update:write") {
		return
	}
	material, err := services.FindInspectionMaterial(c.GetPathParam("material_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "素材不存在")
		return
	}
	var req inspectionMaterialRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		c.WriteError(http.StatusBadRequest, "素材标题不能为空")
		return
	}
	material.Category = req.Category
	material.Title = strings.TrimSpace(req.Title)
	material.Standard = req.Standard
	images := req.StandardImages
	material.StandardImages = models.MarshalStringSlice(images)
	material.StandardImage = firstImage(images)
	material.ScoreType = normalizeScoreType(req.ScoreType)
	material.MaxScore = req.MaxScore
	if len(req.ScoreOptions) > 0 {
		material.ScoreOptions = models.MarshalScoreOptions(req.ScoreOptions)
	} else {
		material.ScoreOptions = models.MarshalScoreOptions(models.DefaultScoreOptions(material.ScoreType, material.MaxScore))
	}
	if err := services.SaveInspectionMaterial(material); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存素材失败")
		return
	}
	c.OK(materialViewFromModel(material))
}

// Delete DELETE /api/inspection-materials/:material_id
func (c *InspectionMaterialsController) Delete() {
	if !c.CheckPermission("inspection:material:delete:write") {
		return
	}
	if err := services.DeleteInspectionMaterial(c.GetPathParam("material_id")); err != nil {
		c.WriteError(http.StatusNotFound, "素材不存在")
		return
	}
	c.WriteNoContent()
}

func materialFromRequest(req *inspectionMaterialRequest) *models.InspectionMaterial {
	scoreType := normalizeScoreType(req.ScoreType)
	maxScore := req.MaxScore
	if maxScore <= 0 {
		maxScore = 5
	}
	options := req.ScoreOptions
	if len(options) == 0 {
		options = models.DefaultScoreOptions(scoreType, maxScore)
	}
	images := req.StandardImages
	material := &models.InspectionMaterial{
		Category:       req.Category,
		Title:          strings.TrimSpace(req.Title),
		Standard:       req.Standard,
		StandardImages: models.MarshalStringSlice(images),
		ScoreType:      scoreType,
		MaxScore:       maxScore,
		ScoreOptions:   models.MarshalScoreOptions(options),
	}
	material.StandardImage = firstImage(images)
	return material
}

func firstImage(images []string) string {
	if len(images) > 0 {
		return images[0]
	}
	return ""
}

func effectiveStandardImages(stored, legacy string) []string {
	if images := models.UnmarshalStringSlice(stored); len(images) > 0 {
		return images
	}
	if strings.TrimSpace(legacy) != "" {
		return []string{legacy}
	}
	return nil
}

func materialViewFromModel(m *models.InspectionMaterial) inspectionMaterialView {
	options := models.UnmarshalScoreOptions(m.ScoreOptions)
	if len(options) == 0 {
		options = models.DefaultScoreOptions(m.ScoreType, m.MaxScore)
	}
	images := effectiveStandardImages(m.StandardImages, m.StandardImage)
	return inspectionMaterialView{
		ID:             m.ID,
		Category:       m.Category,
		Title:          m.Title,
		Standard:       m.Standard,
		StandardImages: images,
		ScoreType:      m.ScoreType,
		MaxScore:       m.MaxScore,
		ScoreOptions:   options,
		IsActive:       m.IsActive,
		CreatedAt:      m.CreatedAt.Format(TimeFormat),
		UpdatedAt:      m.UpdatedAt.Format(TimeFormat),
	}
}
