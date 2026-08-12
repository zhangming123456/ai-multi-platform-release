package controllers

import (
	"net/http"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type MaterialsController struct {
	BaseController
}

type materialRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	Category string `json:"category"`
}

type materialView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Type      string `json:"type"`
	Category  string `json:"category"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// List GET /api/materials
func (c *MaterialsController) List() {
	if !c.CheckPermission("material:read") {
		return
	}
	page, pageSize := c.ParsePagination()
	materialType := strings.TrimSpace(c.GetQuery("type"))
	if materialType == "" {
		materialType = "image"
	}
	category := strings.TrimSpace(c.GetQuery("category"))
	keyword := strings.TrimSpace(c.GetQuery("keyword"))
	materials, total, err := services.ListMaterials(materialType, category, keyword, page, pageSize)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询素材失败")
		return
	}
	items := make([]materialView, 0, len(materials))
	for _, m := range materials {
		items = append(items, buildMaterialView(&m))
	}
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Categories GET /api/materials/categories
func (c *MaterialsController) Categories() {
	if !c.CheckPermission("material:read") {
		return
	}
	materialType := strings.TrimSpace(c.GetQuery("type"))
	if materialType == "" {
		materialType = "image"
	}
	categories, err := services.ListMaterialCategories(materialType)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询分类失败")
		return
	}
	c.OK(categories)
}

// Create POST /api/materials
func (c *MaterialsController) Create() {
	if !c.CheckPermission("material:create:write") {
		return
	}
	var req materialRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.URL) == "" {
		c.WriteError(http.StatusBadRequest, "素材名称和地址不能为空")
		return
	}
	material := buildMaterialFromRequest(&req)
	if err := services.SaveMaterial(material); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存素材失败")
		return
	}
	c.Created(buildMaterialView(material))
}

// Get GET /api/materials/:material_id
func (c *MaterialsController) Get() {
	if !c.CheckPermission("material:read") {
		return
	}
	material, err := services.FindMaterial(c.GetPathParam("material_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "素材不存在")
		return
	}
	c.OK(buildMaterialView(material))
}

// Update PUT /api/materials/:material_id
func (c *MaterialsController) Update() {
	if !c.CheckPermission("material:update:write") {
		return
	}
	material, err := services.FindMaterial(c.GetPathParam("material_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "素材不存在")
		return
	}
	var req materialRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.URL) == "" {
		c.WriteError(http.StatusBadRequest, "素材名称和地址不能为空")
		return
	}
	material.Name = strings.TrimSpace(req.Name)
	material.URL = strings.TrimSpace(req.URL)
	material.Type = normalizeMaterialType(req.Type)
	material.Category = strings.TrimSpace(req.Category)
	if err := services.SaveMaterial(material); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存素材失败")
		return
	}
	c.OK(buildMaterialView(material))
}

// Delete DELETE /api/materials/:material_id
func (c *MaterialsController) Delete() {
	if !c.CheckPermission("material:delete:write") {
		return
	}
	if err := services.DeleteMaterial(c.GetPathParam("material_id")); err != nil {
		c.WriteError(http.StatusNotFound, "素材不存在")
		return
	}
	c.WriteNoContent()
}

func buildMaterialFromRequest(req *materialRequest) *models.Material {
	return &models.Material{
		Name:     strings.TrimSpace(req.Name),
		URL:      strings.TrimSpace(req.URL),
		Type:     normalizeMaterialType(req.Type),
		Category: strings.TrimSpace(req.Category),
	}
}

func normalizeMaterialType(t string) string {
	if t == "image" {
		return "image"
	}
	return "image"
}

func buildMaterialView(m *models.Material) materialView {
	return materialView{
		ID:        m.ID,
		Name:      m.Name,
		URL:       m.URL,
		Type:      m.Type,
		Category:  m.Category,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt.Format(TimeFormat),
		UpdatedAt: m.UpdatedAt.Format(TimeFormat),
	}
}
