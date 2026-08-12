package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"

	"github.com/beego/beego/v2/client/orm"
)

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if strings.Contains(key, "****") {
		return key
	}
	runes := []rune(key)
	if len(runes) <= 8 {
		return "******"
	}
	return string(runes[:4]) + "****" + string(runes[len(runes)-4:])
}

func isMaskedAPIKey(key string) bool {
	return strings.Contains(key, "****")
}

type ModelConfigsController struct {
	BaseController
}

type modelConfigCreateRequest struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Provider       string `json:"provider"`
	Mode           string `json:"mode"`
	APIFormat      string `json:"api_format"`
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`
	FullURL        bool   `json:"full_url"`
	Model          string `json:"model"`
	Multimodal     bool   `json:"multimodal"`
	ModelSeries    string `json:"model_series"`
	ContextInput   int    `json:"context_input"`
	ContextOutput  int    `json:"context_output"`
	ToolCallRounds int    `json:"tool_call_rounds"`
	Enabled        bool   `json:"enabled"`
	MonthlyQuota   int    `json:"monthly_quota"`
	UsedTokens     int    `json:"used_tokens"`
}

type modelConfigUpdateRequest struct {
	Name           *string `json:"name"`
	DisplayName    *string `json:"display_name"`
	Provider       *string `json:"provider"`
	Mode           *string `json:"mode"`
	APIFormat      *string `json:"api_format"`
	APIKey         *string `json:"api_key"`
	BaseURL        *string `json:"base_url"`
	FullURL        *bool   `json:"full_url"`
	Model          *string `json:"model"`
	Multimodal     *bool   `json:"multimodal"`
	ModelSeries    *string `json:"model_series"`
	ContextInput   *int    `json:"context_input"`
	ContextOutput  *int    `json:"context_output"`
	ToolCallRounds *int    `json:"tool_call_rounds"`
	Enabled        *bool   `json:"enabled"`
	MonthlyQuota   *int    `json:"monthly_quota"`
	UsedTokens     *int    `json:"used_tokens"`
}

// List GET /api/model-configs
func (c *ModelConfigsController) List() {
	if !c.CheckPermission("token_plan:read") {
		return
	}
	var configs []models.ModelConfig
	_, err := services.GetOrm().QueryTable(new(models.ModelConfig)).
		OrderBy("sort_order", "created_at").
		All(&configs)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询模型配置失败")
		return
	}
	for i := range configs {
		configs[i].APIKey = maskAPIKey(configs[i].APIKey)
	}
	c.OK(map[string]interface{}{"data": configs})
}

// Create POST /api/model-configs
func (c *ModelConfigsController) Create() {
	if !c.CheckPermission("model_config:create:write") {
		return
	}
	var req modelConfigCreateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	configID := req.ID
	if configID == "" {
		configID = fmt.Sprintf("plan-%d", time.Now().UnixMilli())
	}
	if services.GetOrm().QueryTable(new(models.ModelConfig)).Filter("id", configID).Exist() {
		c.WriteError(http.StatusBadRequest, "配置 ID 已存在")
		return
	}
	config := &models.ModelConfig{
		ID:             configID,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Provider:       req.Provider,
		Mode:           req.Mode,
		APIFormat:      req.APIFormat,
		APIKey:         req.APIKey,
		BaseURL:        req.BaseURL,
		FullURL:        req.FullURL,
		Model:          req.Model,
		Multimodal:     req.Multimodal,
		ModelSeries:    req.ModelSeries,
		ContextInput:   req.ContextInput,
		ContextOutput:  req.ContextOutput,
		ToolCallRounds: req.ToolCallRounds,
		Enabled:        req.Enabled,
		MonthlyQuota:   req.MonthlyQuota,
		UsedTokens:     req.UsedTokens,
	}
	if config.APIFormat == "" {
		config.APIFormat = "openai_chat"
	}
	if config.ModelSeries == "" {
		config.ModelSeries = "default"
	}
	if config.ContextInput == 0 {
		config.ContextInput = 128000
	}
	if config.ContextOutput == 0 {
		config.ContextOutput = 4096
	}
	if config.ToolCallRounds == 0 {
		config.ToolCallRounds = 200
	}
	if config.MonthlyQuota == 0 {
		config.MonthlyQuota = 1000000
	}
	var allConfigs []models.ModelConfig
	if _, err := services.GetOrm().QueryTable(new(models.ModelConfig)).All(&allConfigs); err == nil {
		config.SortOrder = len(allConfigs)
	}
	if _, err := services.GetOrm().Insert(config); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建模型配置失败")
		return
	}
	config.APIKey = maskAPIKey(config.APIKey)
	c.Created(config)
}

func findModelConfig(configID string) (*models.ModelConfig, error) {
	var config models.ModelConfig
	config.ID = configID
	err := services.GetOrm().Read(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Update PUT /api/model-configs/:config_id
func (c *ModelConfigsController) Update() {
	if !c.CheckPermission("model_config:update:write") {
		return
	}
	config, err := findModelConfig(c.GetPathParam("config_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "配置不存在")
		return
	}
	var req modelConfigUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.DisplayName != nil {
		config.DisplayName = *req.DisplayName
	}
	if req.Provider != nil {
		config.Provider = *req.Provider
	}
	if req.Mode != nil {
		config.Mode = *req.Mode
	}
	if req.APIFormat != nil {
		config.APIFormat = *req.APIFormat
	}
	if req.APIKey != nil && !isMaskedAPIKey(*req.APIKey) {
		config.APIKey = *req.APIKey
	}
	if req.BaseURL != nil {
		config.BaseURL = *req.BaseURL
	}
	if req.FullURL != nil {
		config.FullURL = *req.FullURL
	}
	if req.Model != nil {
		config.Model = *req.Model
	}
	if req.Multimodal != nil {
		config.Multimodal = *req.Multimodal
	}
	if req.ModelSeries != nil {
		config.ModelSeries = *req.ModelSeries
	}
	if req.ContextInput != nil {
		config.ContextInput = *req.ContextInput
	}
	if req.ContextOutput != nil {
		config.ContextOutput = *req.ContextOutput
	}
	if req.ToolCallRounds != nil {
		config.ToolCallRounds = *req.ToolCallRounds
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.MonthlyQuota != nil {
		config.MonthlyQuota = *req.MonthlyQuota
	}
	if req.UsedTokens != nil {
		config.UsedTokens = *req.UsedTokens
	}
	if _, err := services.GetOrm().Update(config); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新模型配置失败")
		return
	}
	config.APIKey = maskAPIKey(config.APIKey)
	c.OK(config)
}

// Delete DELETE /api/model-configs/:config_id
func (c *ModelConfigsController) Delete() {
	if !c.CheckPermission("model_config:delete:write") {
		return
	}
	config, err := findModelConfig(c.GetPathParam("config_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "配置不存在")
		return
	}
	if _, err := services.GetOrm().Delete(config); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除模型配置失败")
		return
	}
	c.WriteNoContent()
}

type modelConfigReorderRequest struct {
	IDs []string `json:"ids"`
}

// Reorder PUT /api/model-configs/reorder
func (c *ModelConfigsController) Reorder() {
	if !c.CheckPermission("model_config:update:write") {
		return
	}
	var req modelConfigReorderRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if len(req.IDs) == 0 {
		c.WriteError(http.StatusBadRequest, "排序列表不能为空")
		return
	}
	o := services.GetOrm()
	for i, id := range req.IDs {
		if _, err := o.QueryTable(new(models.ModelConfig)).Filter("id", id).Update(orm.Params{"sort_order": i}); err != nil {
			c.WriteError(http.StatusInternalServerError, "更新排序失败")
			return
		}
	}
	c.OK(map[string]interface{}{"message": "排序已更新"})
}
