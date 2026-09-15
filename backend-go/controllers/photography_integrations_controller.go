package controllers

import (
	"ai-multi-platform-release/backend-go/services"
	"net/http"
)

type PhotographyIntegrationsController struct{ BaseController }

func (c *PhotographyIntegrationsController) Get() {
	if !c.CheckPermission("photography_tool:config:read") {
		return
	}
	c.OK(services.ListPhotographyIntegrationConfig())
}

func (c *PhotographyIntegrationsController) Update() {
	if !c.CheckPermission("photography_tool:config:update") {
		return
	}
	var payload services.PhotographyIntegrationConfigPayload
	if err := c.ParseBody(&payload); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	user := c.CurrentUser()
	updatedBy := ""
	if user != nil {
		updatedBy = user.ID
	}
	if err := services.SavePhotographyIntegrationConfig(payload, updatedBy); err != nil {
		status := http.StatusInternalServerError
		if services.IsPhotographyValidationError(err) {
			status = http.StatusBadRequest
		}
		c.WriteError(status, err.Error())
		return
	}
	c.OK(services.ListPhotographyIntegrationConfig())
}
