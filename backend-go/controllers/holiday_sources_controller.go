package controllers

import (
	"net/http"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type HolidaySourcesController struct {
	BaseController
}

type holidaySourceRequest struct {
	Name            string `json:"name"`
	Color           string `json:"color"`
	URL             string `json:"url"`
	Enabled         *bool  `json:"enabled"`
	RefreshInterval string `json:"refresh_interval"`
}

type holidaySourceView struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Color           string  `json:"color"`
	URL             string  `json:"url"`
	Enabled         bool    `json:"enabled"`
	RefreshInterval string  `json:"refresh_interval"`
	LastRefreshedAt *string `json:"last_refreshed_at"`
	LastAttemptAt   *string `json:"last_attempt_at"`
	LastStatus      string  `json:"last_status"`
	LastError       string  `json:"last_error"`
	EventCount      int     `json:"event_count"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type holidaySourceListResponse struct {
	Items           []holidaySourceView             `json:"items"`
	IntervalOptions []services.HolidayRefreshOption `json:"interval_options"`
}

func (r holidaySourceRequest) enabledValue() bool {
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

func formatHolidayLocalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Local().Format("2006-01-02T15:04:05")
	return &formatted
}

func toHolidaySourceView(source models.HolidaySource) holidaySourceView {
	return holidaySourceView{
		ID:              source.ID,
		Name:            source.Name,
		Color:           source.Color,
		URL:             source.URL,
		Enabled:         source.Enabled,
		RefreshInterval: source.RefreshInterval,
		LastRefreshedAt: formatHolidayLocalTime(source.LastRefreshedAt),
		LastAttemptAt:   formatHolidayLocalTime(source.LastAttemptAt),
		LastStatus:      source.LastStatus,
		LastError:       source.LastError,
		EventCount:      source.EventCount,
		CreatedAt:       source.CreatedAt.Local().Format("2006-01-02T15:04:05"),
		UpdatedAt:       source.UpdatedAt.Local().Format("2006-01-02T15:04:05"),
	}
}

func toHolidaySourceViews(sources []models.HolidaySource) []holidaySourceView {
	views := make([]holidaySourceView, 0, len(sources))
	for i := range sources {
		views = append(views, toHolidaySourceView(sources[i]))
	}
	return views
}

// List GET /api/holiday-sources
func (c *HolidaySourcesController) List() {
	if !c.CheckPermission("holiday_source:read") {
		return
	}
	sources, err := services.ListHolidaySources()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "获取订阅源列表失败")
		return
	}
	c.OK(holidaySourceListResponse{
		Items:           toHolidaySourceViews(sources),
		IntervalOptions: services.HolidayRefreshIntervalOptions(),
	})
}

// Create POST /api/holiday-sources
func (c *HolidaySourcesController) Create() {
	if !c.CheckPermission("holiday_source:write") {
		return
	}
	var req holidaySourceRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	source, err := services.CreateHolidaySource(req.Name, req.URL, req.Color, req.enabledValue(), req.RefreshInterval)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	c.Created(toHolidaySourceView(*source))
}

// Update PUT /api/holiday-sources/:id
func (c *HolidaySourcesController) Update() {
	if !c.CheckPermission("holiday_source:write") {
		return
	}
	id := c.GetPathParam("id")
	var req holidaySourceRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	source, err := services.UpdateHolidaySource(id, req.Name, req.URL, req.Color, req.enabledValue(), req.RefreshInterval)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	c.OK(toHolidaySourceView(*source))
}

// Delete DELETE /api/holiday-sources/:id
func (c *HolidaySourcesController) Delete() {
	if !c.CheckPermission("holiday_source:write") {
		return
	}
	id := c.GetPathParam("id")
	if err := services.DeleteHolidaySource(id); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除订阅源失败")
		return
	}
	c.WriteNoContent()
}

// Refresh POST /api/holiday-sources/:id/refresh
func (c *HolidaySourcesController) Refresh() {
	if !c.CheckPermission("holiday_source:write") {
		return
	}
	id := c.GetPathParam("id")
	source, err := services.RefreshHolidaySource(id)
	if err != nil {
		if source != nil {
			c.WriteError(http.StatusBadGateway, "刷新失败："+err.Error())
			return
		}
		c.WriteError(http.StatusNotFound, err.Error())
		return
	}
	c.OK(toHolidaySourceView(*source))
}

// RefreshAll POST /api/holiday-sources/refresh-all
func (c *HolidaySourcesController) RefreshAll() {
	if !c.CheckPermission("holiday_source:write") {
		return
	}
	count := services.RefreshAllHolidaySources()
	c.OK(map[string]interface{}{"refreshed": count})
}
