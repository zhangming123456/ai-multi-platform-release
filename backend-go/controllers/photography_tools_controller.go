package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"ai-multi-platform-release/backend-go/services"
)

type PhotographyToolsController struct{ BaseController }

func (c *PhotographyToolsController) Overview() {
	if !c.CheckPermission("photography_tool:read") {
		return
	}
	latitude, longitude, err := parsePhotographyToolCoordinates(c)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	overview, err := services.FetchPhotographyOverview(latitude, longitude, strings.TrimSpace(c.GetQuery("date")), c.GetQuery("weather_source"), c.GetQuery("tide_source"), c.GetQuery("location_name"), c.GetQuery("country_code"))
	if err != nil {
		status := http.StatusBadGateway
		if services.IsPhotographyValidationError(err) {
			status = http.StatusBadRequest
		}
		c.WriteError(status, err.Error())
		return
	}
	c.OK(overview)
}

func (c *PhotographyToolsController) Geocode() {
	if !c.CheckPermission("photography_tool:read") {
		return
	}
	locations, err := services.SearchPhotographyLocations(c.GetQuery("query"), c.GetQuery("country_code"))
	if err != nil {
		status := http.StatusBadGateway
		if services.IsPhotographyValidationError(err) {
			status = http.StatusBadRequest
		}
		c.WriteError(status, err.Error())
		return
	}
	c.OK(locations)
}

func (c *PhotographyToolsController) MapConfig() {
	if !c.CheckPermission("photography_tool:read") {
		return
	}
	c.OK(services.GetPhotographyMapConfig(c.GetQuery("country_code")))
}

// Spatial 返回以中心点为中心的空间网格，供概率图/云量质量图的地图模式渲染。
func (c *PhotographyToolsController) Spatial() {
	if !c.CheckPermission("photography_tool:read") {
		return
	}
	latitude, longitude, err := parsePhotographyToolCoordinates(c)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	grid, err := services.FetchPhotographySpatialGrid(latitude, longitude, c.GetQuery("date"), c.GetQuery("period"), c.GetQuery("weather_source"))
	if err != nil {
		status := http.StatusBadGateway
		if services.IsPhotographyValidationError(err) {
			status = http.StatusBadRequest
		}
		c.WriteError(status, err.Error())
		return
	}
	c.OK(grid)
}

// Timezone 供前端高德 JS API 搜索完成后，按经纬度补齐 Open-Meteo 地点时区。
func (c *PhotographyToolsController) Timezone() {
	if !c.CheckPermission("photography_tool:read") {
		return
	}
	latitude, longitude, err := parsePhotographyToolCoordinates(c)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	timezone, err := services.FetchPhotographyTimezone(latitude, longitude)
	if err != nil {
		c.WriteError(http.StatusBadGateway, err.Error())
		return
	}
	c.OK(map[string]string{"timezone": timezone})
}

func parsePhotographyToolCoordinates(c *PhotographyToolsController) (float64, float64, error) {
	latitude, err := strconv.ParseFloat(strings.TrimSpace(c.GetQuery("latitude")), 64)
	if err != nil {
		return 0, 0, photographyToolError("latitude 参数不合法")
	}
	longitude, err := strconv.ParseFloat(strings.TrimSpace(c.GetQuery("longitude")), 64)
	if err != nil {
		return 0, 0, photographyToolError("longitude 参数不合法")
	}
	if err := services.ValidatePhotographyCoordinates(latitude, longitude); err != nil {
		return 0, 0, err
	}
	return latitude, longitude, nil
}

type photographyToolError string

func (e photographyToolError) Error() string { return string(e) }
