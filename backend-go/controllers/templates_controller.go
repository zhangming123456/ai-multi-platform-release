package controllers

import (
	"net/http"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type TemplatesController struct {
	BaseController
}

type templateCreateRequest struct {
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	ThumbnailURL string `json:"thumbnail_url"`
	Config       string `json:"config"`
}

type templateUpdateRequest struct {
	Name         *string `json:"name"`
	Platform     *string `json:"platform"`
	ThumbnailURL *string `json:"thumbnail_url"`
	Config       *string `json:"config"`
}

// List GET /api/templates/
func (c *TemplatesController) List() {
	if !c.CheckPermission("templates:read") {
		return
	}
	qs := services.GetOrm().QueryTable(new(models.Template))
	if platform := c.GetQuery("platform"); platform != "" {
		qs = qs.Filter("platform", platform)
	}
	var templates []models.Template
	_, err := qs.OrderBy("-created_at").All(&templates)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询模板失败")
		return
	}
	c.OK(templates)
}

// Create POST /api/templates/
func (c *TemplatesController) Create() {
	if !c.CheckPermission("templates:create:write") {
		return
	}
	var req templateCreateRequest
	if err := c.ParseBody(&req); err != nil || req.Name == "" {
		c.WriteError(http.StatusBadRequest, "模板名称不能为空")
		return
	}
	template := &models.Template{
		ID:           newID(),
		Name:         req.Name,
		Platform:     req.Platform,
		ThumbnailURL: req.ThumbnailURL,
		Config:       req.Config,
	}
	if _, err := services.GetOrm().Insert(template); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建模板失败")
		return
	}
	c.Created(template)
}

func findTemplate(templateID string) (*models.Template, error) {
	var template models.Template
	err := services.GetOrm().QueryTable(new(models.Template)).
		Filter("id", templateID).
		One(&template)
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// Get GET /api/templates/:template_id
func (c *TemplatesController) Get() {
	if !c.CheckPermission("templates:read") {
		return
	}
	template, err := findTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "模板不存在")
		return
	}
	c.OK(template)
}

// Update PUT /api/templates/:template_id
func (c *TemplatesController) Update() {
	if !c.CheckPermission("templates:update:write") {
		return
	}
	template, err := findTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "模板不存在")
		return
	}
	var req templateUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Platform != nil {
		template.Platform = *req.Platform
	}
	if req.ThumbnailURL != nil {
		template.ThumbnailURL = *req.ThumbnailURL
	}
	if req.Config != nil {
		template.Config = *req.Config
	}
	if _, err := services.GetOrm().Update(template); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新模板失败")
		return
	}
	c.OK(template)
}

// Delete DELETE /api/templates/:template_id
func (c *TemplatesController) Delete() {
	if !c.CheckPermission("templates:delete:write") {
		return
	}
	template, err := findTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "模板不存在")
		return
	}
	if _, err := services.GetOrm().Delete(template); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除模板失败")
		return
	}
	c.WriteNoContent()
}
