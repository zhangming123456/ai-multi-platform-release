package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type StyleTemplatesController struct {
	BaseController
}

type styleTemplateCreateRequest struct {
	Name        string   `json:"name"`
	Platform    string   `json:"platform"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}

type styleTemplateUpdateRequest struct {
	Name        *string   `json:"name"`
	Platform    *string   `json:"platform"`
	Description *string   `json:"description"`
	Keywords    *[]string `json:"keywords"`
	Status      *string   `json:"status"`
}

func styleTemplateMap(s *models.StyleTemplate) map[string]interface{} {
	keywords := []string{}
	if s.Keywords != "" {
		_ = json.Unmarshal([]byte(s.Keywords), &keywords)
	}
	return map[string]interface{}{
		"id":          s.ID,
		"name":        s.Name,
		"platform":    s.Platform,
		"description": s.Description,
		"keywords":    keywords,
		"status":      s.Status,
		"created_at":  s.CreatedAt,
		"updated_at":  s.UpdatedAt,
	}
}

// List GET /api/style-templates/
func (c *StyleTemplatesController) List() {
	if !c.CheckPermission("templates:read") {
		return
	}
	qs := services.GetOrm().QueryTable(new(models.StyleTemplate))
	if platform := c.GetQuery("platform"); platform != "" {
		qs = qs.Filter("platform", platform)
	}
	if status := c.GetQuery("status"); status != "" {
		qs = qs.Filter("status", status)
	}
	var templates []models.StyleTemplate
	if _, err := qs.OrderBy("-created_at").All(&templates); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询风格模板失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(templates))
	for i := range templates {
		items = append(items, styleTemplateMap(&templates[i]))
	}
	c.OK(items)
}

// Create POST /api/style-templates/
func (c *StyleTemplatesController) Create() {
	if !c.CheckPermission("templates:create:write") {
		return
	}
	var req styleTemplateCreateRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.WriteError(http.StatusBadRequest, "模板名称不能为空")
		return
	}
	if req.Platform == "" {
		c.WriteError(http.StatusBadRequest, "目标平台不能为空")
		return
	}
	t := &models.StyleTemplate{
		ID:          newID(),
		Name:        strings.TrimSpace(req.Name),
		Platform:    req.Platform,
		Description: req.Description,
		Keywords:    marshalStringList(req.Keywords),
		Status:      models.StyleTemplateStatusActive,
	}
	if _, err := services.GetOrm().Insert(t); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建风格模板失败")
		return
	}
	c.Created(styleTemplateMap(t))
}

// Update PUT /api/style-templates/:template_id
func (c *StyleTemplatesController) Update() {
	if !c.CheckPermission("templates:update:write") {
		return
	}
	t, err := findStyleTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "风格模板不存在")
		return
	}
	var req styleTemplateUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体无效")
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.WriteError(http.StatusBadRequest, "模板名称不能为空")
			return
		}
		t.Name = name
	}
	if req.Platform != nil {
		t.Platform = *req.Platform
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Keywords != nil {
		t.Keywords = marshalStringList(*req.Keywords)
	}
	if req.Status != nil {
		if *req.Status != models.StyleTemplateStatusActive && *req.Status != models.StyleTemplateStatusArchived {
			c.WriteError(http.StatusBadRequest, "无效的模板状态")
			return
		}
		t.Status = *req.Status
	}
	if _, err := services.GetOrm().Update(t); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新风格模板失败")
		return
	}
	c.OK(styleTemplateMap(t))
}

// Delete DELETE /api/style-templates/:template_id
func (c *StyleTemplatesController) Delete() {
	if !c.CheckPermission("templates:delete:write") {
		return
	}
	t, err := findStyleTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "风格模板不存在")
		return
	}
	t.Status = models.StyleTemplateStatusArchived
	if _, err := services.GetOrm().Update(t); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除风格模板失败")
		return
	}
	c.WriteNoContent()
}

func findStyleTemplate(id string) (*models.StyleTemplate, error) {
	var t models.StyleTemplate
	err := services.GetOrm().QueryTable(new(models.StyleTemplate)).
		Filter("id", id).
		One(&t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type PromptTemplatesController struct {
	BaseController
}

type promptTemplateCreateRequest struct {
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	Role         string `json:"role"`
	Rules        string `json:"rules"`
	OutputFormat string `json:"output_format"`
	IsDefault    bool   `json:"is_default"`
}

type promptTemplateUpdateRequest struct {
	Name         *string `json:"name"`
	Platform     *string `json:"platform"`
	Role         *string `json:"role"`
	Rules        *string `json:"rules"`
	OutputFormat *string `json:"output_format"`
	IsDefault    *bool   `json:"is_default"`
	Status       *string `json:"status"`
}

func promptTemplateMap(p *models.PromptTemplate) map[string]interface{} {
	return map[string]interface{}{
		"id":            p.ID,
		"name":          p.Name,
		"platform":      p.Platform,
		"role":          p.Role,
		"rules":         p.Rules,
		"output_format": p.OutputFormat,
		"is_default":    p.IsDefault,
		"status":        p.Status,
		"created_at":    p.CreatedAt,
		"updated_at":    p.UpdatedAt,
	}
}

// List GET /api/prompt-templates/
func (c *PromptTemplatesController) List() {
	if !c.CheckPermission("templates:read") {
		return
	}
	qs := services.GetOrm().QueryTable(new(models.PromptTemplate))
	if platform := c.GetQuery("platform"); platform != "" {
		qs = qs.Filter("platform", platform)
	}
	if status := c.GetQuery("status"); status != "" {
		qs = qs.Filter("status", status)
	}
	var templates []models.PromptTemplate
	if _, err := qs.OrderBy("-is_default", "-created_at").All(&templates); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询 Prompt 模板失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(templates))
	for i := range templates {
		items = append(items, promptTemplateMap(&templates[i]))
	}
	c.OK(items)
}

// Create POST /api/prompt-templates/
func (c *PromptTemplatesController) Create() {
	if !c.CheckPermission("templates:create:write") {
		return
	}
	var req promptTemplateCreateRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.WriteError(http.StatusBadRequest, "模板名称不能为空")
		return
	}
	if req.Platform == "" {
		c.WriteError(http.StatusBadRequest, "目标平台不能为空")
		return
	}
	p := &models.PromptTemplate{
		ID:           newID(),
		Name:         strings.TrimSpace(req.Name),
		Platform:     req.Platform,
		Role:         req.Role,
		Rules:        req.Rules,
		OutputFormat: req.OutputFormat,
		IsDefault:    req.IsDefault,
		Status:       models.PromptTemplateStatusActive,
	}
	if req.IsDefault {
		if err := clearDefaultPromptTemplate(req.Platform); err != nil {
			c.WriteError(http.StatusInternalServerError, "设置默认模板失败")
			return
		}
	}
	if _, err := services.GetOrm().Insert(p); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建 Prompt 模板失败")
		return
	}
	c.Created(promptTemplateMap(p))
}

// Update PUT /api/prompt-templates/:template_id
func (c *PromptTemplatesController) Update() {
	if !c.CheckPermission("templates:update:write") {
		return
	}
	p, err := findPromptTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "Prompt 模板不存在")
		return
	}
	var req promptTemplateUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体无效")
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.WriteError(http.StatusBadRequest, "模板名称不能为空")
			return
		}
		p.Name = name
	}
	if req.Platform != nil {
		p.Platform = *req.Platform
	}
	if req.Role != nil {
		p.Role = *req.Role
	}
	if req.Rules != nil {
		p.Rules = *req.Rules
	}
	if req.OutputFormat != nil {
		p.OutputFormat = *req.OutputFormat
	}
	if req.IsDefault != nil && *req.IsDefault {
		if err := clearDefaultPromptTemplate(p.Platform); err != nil {
			c.WriteError(http.StatusInternalServerError, "设置默认模板失败")
			return
		}
		p.IsDefault = true
	}
	if req.Status != nil {
		if *req.Status != models.PromptTemplateStatusActive && *req.Status != models.PromptTemplateStatusArchived {
			c.WriteError(http.StatusBadRequest, "无效的模板状态")
			return
		}
		p.Status = *req.Status
	}
	if _, err := services.GetOrm().Update(p); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新 Prompt 模板失败")
		return
	}
	c.OK(promptTemplateMap(p))
}

// Delete DELETE /api/prompt-templates/:template_id
func (c *PromptTemplatesController) Delete() {
	if !c.CheckPermission("templates:delete:write") {
		return
	}
	p, err := findPromptTemplate(c.GetPathParam("template_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "Prompt 模板不存在")
		return
	}
	p.Status = models.PromptTemplateStatusArchived
	if _, err := services.GetOrm().Update(p); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除 Prompt 模板失败")
		return
	}
	c.WriteNoContent()
}

func findPromptTemplate(id string) (*models.PromptTemplate, error) {
	var p models.PromptTemplate
	err := services.GetOrm().QueryTable(new(models.PromptTemplate)).
		Filter("id", id).
		One(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func clearDefaultPromptTemplate(platform string) error {
	o := services.GetOrm()
	var templates []models.PromptTemplate
	_, err := o.QueryTable(new(models.PromptTemplate)).
		Filter("platform", platform).
		Filter("is_default", true).
		All(&templates)
	if err != nil {
		return err
	}
	for i := range templates {
		templates[i].IsDefault = false
		if _, err := o.Update(&templates[i]); err != nil {
			return err
		}
	}
	return nil
}
