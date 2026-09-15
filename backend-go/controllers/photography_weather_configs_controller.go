package controllers

import (
	"net/http"
	"strings"

	"ai-multi-platform-release/backend-go/services"
)

type PhotographyWeatherConfigsController struct {
	BaseController
}

type photographyWeatherConfigUpdateRequest struct {
	CredentialType string `json:"credential_type"`
	APIKey         string `json:"api_key"`
	APIHost        string `json:"api_host"`
	ClearAPIKey    bool   `json:"clear_api_key"`
}

func (c *PhotographyWeatherConfigsController) List() {
	if !c.CheckPermission("photography_weather_config:read") {
		return
	}
	c.OK(services.ListPhotographyWeatherConfigs())
}

func (c *PhotographyWeatherConfigsController) Update() {
	if !c.CheckPermission("photography_weather_config:update:write") {
		return
	}
	var req photographyWeatherConfigUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	source := strings.TrimSpace(c.GetPathParam("source"))
	if err := services.SavePhotographyWeatherConfig(source, req.CredentialType, req.APIKey, req.APIHost, req.ClearAPIKey); err != nil {
		status := http.StatusInternalServerError
		if services.IsPhotographyValidationError(err) {
			status = http.StatusBadRequest
		}
		c.WriteError(status, err.Error())
		return
	}
	for _, item := range services.ListPhotographyWeatherConfigs() {
		if item.Source == source {
			c.OK(item)
			return
		}
	}
	c.WriteError(http.StatusNotFound, "天气配置不存在")
}
