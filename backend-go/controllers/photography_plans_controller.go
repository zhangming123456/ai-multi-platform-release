package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type PhotographyPlansController struct {
	BaseController
}

type photographyPlanCreateRequest struct {
	Name          string  `json:"name"`
	Date          string  `json:"date"`
	Session       string  `json:"session"`
	LocationName  string  `json:"location_name"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	WeatherSource string  `json:"weather_source"`
	Note          string  `json:"note"`
}

type photographyPlanUpdateRequest struct {
	Name          *string  `json:"name"`
	Date          *string  `json:"date"`
	Session       *string  `json:"session"`
	LocationName  *string  `json:"location_name"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	WeatherSource *string  `json:"weather_source"`
	Note          *string  `json:"note"`
}

func (c *PhotographyPlansController) Geocode() {
	if !c.CheckPermission("photography_plan:read") {
		return
	}
	locations, err := services.SearchPhotographyLocations(c.GetQuery("query"))
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

func (c *PhotographyPlansController) WeatherSources() {
	if !c.CheckPermission("photography_plan:read") {
		return
	}
	c.OK(services.ListPhotographyWeatherSources())
}

func (c *PhotographyPlansController) Forecast() {
	if !c.CheckPermission("photography_plan:read") {
		return
	}
	latitude, longitude, date, source, err := parsePhotographyForecastQuery(c)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	forecast, err := services.FetchPhotographyForecastWithSource(latitude, longitude, date, source)
	if err != nil {
		writePhotographyForecastError(c, err)
		return
	}
	c.OK(forecast)
}

func (c *PhotographyPlansController) List() {
	if !c.CheckPermission("photography_plan:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	page, pageSize := c.ParsePagination()
	condition := orm.NewCondition().And("user_id", user.ID)
	if keyword := strings.TrimSpace(c.GetQuery("keyword")); keyword != "" {
		keywordCondition := orm.NewCondition().
			Or("name__icontains", keyword).
			Or("location_name__icontains", keyword).
			Or("note__icontains", keyword)
		condition = condition.AndCond(keywordCondition)
	}
	if session := strings.TrimSpace(c.GetQuery("session")); session != "" {
		if !isPhotographySession(session) {
			c.WriteError(http.StatusBadRequest, "拍摄时段不合法")
			return
		}
		condition = condition.And("session", session)
	}

	query := services.GetOrm().QueryTable(new(models.PhotographyPlan)).SetCond(condition)
	total, err := query.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询摄影计划失败")
		return
	}
	var plans []models.PhotographyPlan
	if _, err := query.OrderBy("date", "-created_at").Limit(pageSize).Offset((page - 1) * pageSize).All(&plans); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询摄影计划失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(plans))
	for i := range plans {
		items = append(items, photographyPlanView(&plans[i]))
	}
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *PhotographyPlansController) Get() {
	if !c.CheckPermission("photography_plan:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	plan, err := findOwnPhotographyPlan(c.GetPathParam("id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "摄影计划不存在")
		return
	}
	c.OK(photographyPlanView(plan))
}

func (c *PhotographyPlansController) Create() {
	if !c.CheckPermission("photography_plan:create:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req photographyPlanCreateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	source := services.NormalizePhotographyWeatherSource(req.WeatherSource)
	if source == "" {
		c.WriteError(http.StatusBadRequest, "天气数据源不合法")
		return
	}
	if err := validatePhotographyPlanFields(req.Name, req.Date, req.Session, req.LocationName, req.Latitude, req.Longitude); err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	forecast, err := services.FetchPhotographyForecastWithSource(req.Latitude, req.Longitude, req.Date, source)
	if err != nil {
		writePhotographyForecastError(c, err)
		return
	}
	now := time.Now()
	plan := &models.PhotographyPlan{
		ID:               newID(),
		UserID:           user.ID,
		Name:             strings.TrimSpace(req.Name),
		Date:             req.Date,
		Session:          req.Session,
		LocationName:     strings.TrimSpace(req.LocationName),
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		Timezone:         forecast.Timezone,
		WeatherSource:    forecast.Source,
		Note:             strings.TrimSpace(req.Note),
		ForecastSnapshot: services.SnapshotFromPhotographyForecast(forecast),
		LastSyncedAt:     &now,
	}
	if _, err := services.GetOrm().Insert(plan); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建摄影计划失败")
		return
	}
	c.Created(photographyPlanView(plan))
}

func (c *PhotographyPlansController) Update() {
	if !c.CheckPermission("photography_plan:update:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	plan, err := findOwnPhotographyPlan(c.GetPathParam("id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "摄影计划不存在")
		return
	}
	var req photographyPlanUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}

	name := plan.Name
	date := plan.Date
	session := plan.Session
	locationName := plan.LocationName
	latitude := plan.Latitude
	longitude := plan.Longitude
	source := services.NormalizePhotographyWeatherSource(plan.WeatherSource)
	note := plan.Note
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Date != nil {
		date = strings.TrimSpace(*req.Date)
	}
	if req.Session != nil {
		session = strings.TrimSpace(*req.Session)
	}
	if req.LocationName != nil {
		locationName = strings.TrimSpace(*req.LocationName)
	}
	if req.Latitude != nil {
		latitude = *req.Latitude
	}
	if req.Longitude != nil {
		longitude = *req.Longitude
	}
	if req.WeatherSource != nil {
		source = services.NormalizePhotographyWeatherSource(*req.WeatherSource)
	}
	if req.Note != nil {
		note = strings.TrimSpace(*req.Note)
	}
	if source == "" {
		c.WriteError(http.StatusBadRequest, "天气数据源不合法")
		return
	}
	if err := validatePhotographyPlanFields(name, date, session, locationName, latitude, longitude); err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}

	locationChanged := date != plan.Date || latitude != plan.Latitude || longitude != plan.Longitude || source != services.NormalizePhotographyWeatherSource(plan.WeatherSource)
	plan.Name = name
	plan.Date = date
	plan.Session = session
	plan.LocationName = locationName
	plan.Latitude = latitude
	plan.Longitude = longitude
	plan.WeatherSource = source
	plan.Note = note
	if locationChanged {
		if err := services.ValidatePhotographyWeatherSource(source); err != nil {
			c.WriteError(http.StatusBadRequest, err.Error())
			return
		}
		forecast, fetchErr := services.FetchPhotographyForecastWithSource(latitude, longitude, date, source)
		if fetchErr != nil {
			writePhotographyForecastError(c, fetchErr)
			return
		}
		now := time.Now()
		plan.Timezone = forecast.Timezone
		plan.ForecastSnapshot = services.SnapshotFromPhotographyForecast(forecast)
		plan.LastSyncedAt = &now
		plan.SyncError = ""
	}
	if _, err := services.GetOrm().Update(plan); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新摄影计划失败")
		return
	}
	c.OK(photographyPlanView(plan))
}

func (c *PhotographyPlansController) Delete() {
	if !c.CheckPermission("photography_plan:delete:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	plan, err := findOwnPhotographyPlan(c.GetPathParam("id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "摄影计划不存在")
		return
	}
	if _, err := services.GetOrm().Delete(plan); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除摄影计划失败")
		return
	}
	c.WriteNoContent()
}

func (c *PhotographyPlansController) Refresh() {
	if !c.CheckPermission("photography_plan:update:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	plan, err := findOwnPhotographyPlan(c.GetPathParam("id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "摄影计划不存在")
		return
	}
	source := services.NormalizePhotographyWeatherSource(plan.WeatherSource)
	forecast, fetchErr := services.FetchPhotographyForecastWithSource(plan.Latitude, plan.Longitude, plan.Date, source)
	if fetchErr != nil {
		plan.SyncError = truncatePhotographyError(fetchErr.Error())
		_, _ = services.GetOrm().Update(plan, "SyncError")
		status := http.StatusBadGateway
		if services.IsPhotographyValidationError(fetchErr) {
			status = http.StatusBadRequest
		}
		c.WriteError(status, "刷新天气失败："+fetchErr.Error())
		return
	}
	now := time.Now()
	plan.Timezone = forecast.Timezone
	plan.WeatherSource = forecast.Source
	plan.ForecastSnapshot = services.SnapshotFromPhotographyForecast(forecast)
	plan.LastSyncedAt = &now
	plan.SyncError = ""
	if _, err := services.GetOrm().Update(plan); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存天气快照失败")
		return
	}
	c.OK(photographyPlanView(plan))
}

func writePhotographyForecastError(c *PhotographyPlansController, err error) {
	status := http.StatusBadGateway
	if services.IsPhotographyValidationError(err) {
		status = http.StatusBadRequest
	}
	c.WriteError(status, err.Error())
}

func parsePhotographyForecastQuery(c *PhotographyPlansController) (float64, float64, string, string, error) {
	latitude, err := strconv.ParseFloat(strings.TrimSpace(c.GetQuery("latitude")), 64)
	if err != nil {
		return 0, 0, "", "", photographyError("latitude 参数不合法")
	}
	longitude, err := strconv.ParseFloat(strings.TrimSpace(c.GetQuery("longitude")), 64)
	if err != nil {
		return 0, 0, "", "", photographyError("longitude 参数不合法")
	}
	date := strings.TrimSpace(c.GetQuery("date"))
	if date == "" {
		return 0, 0, "", "", photographyError("date 参数不能为空")
	}
	source := services.NormalizePhotographyWeatherSource(c.GetQuery("source"))
	if source == "" {
		return 0, 0, "", "", photographyError("天气数据源不合法")
	}
	if err := services.ValidatePhotographyCoordinates(latitude, longitude); err != nil {
		return 0, 0, "", "", err
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return 0, 0, "", "", photographyError("date 参数必须为 YYYY-MM-DD")
	}
	return latitude, longitude, date, source, nil
}

func validatePhotographyPlanFields(name, date, session, locationName string, latitude, longitude float64) error {
	if strings.TrimSpace(name) == "" {
		return photographyError("计划名称不能为空")
	}
	if len([]rune(strings.TrimSpace(name))) > 200 {
		return photographyError("计划名称不能超过 200 个字符")
	}
	if strings.TrimSpace(locationName) == "" {
		return photographyError("拍摄地点不能为空")
	}
	if len([]rune(strings.TrimSpace(locationName))) > 200 {
		return photographyError("拍摄地点不能超过 200 个字符")
	}
	if !isPhotographySession(session) {
		return photographyError("拍摄时段不合法")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return photographyError("日期必须为 YYYY-MM-DD")
	}
	return services.ValidatePhotographyCoordinates(latitude, longitude)
}

func isPhotographySession(value string) bool {
	return value == models.PhotographyPlanSessionSunrise || value == models.PhotographyPlanSessionSunset || value == models.PhotographyPlanSessionBoth
}

func findOwnPhotographyPlan(id, userID string) (*models.PhotographyPlan, error) {
	var plan models.PhotographyPlan
	err := services.GetOrm().QueryTable(new(models.PhotographyPlan)).
		Filter("id", id).
		Filter("user_id", userID).
		One(&plan)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func photographyPlanView(plan *models.PhotographyPlan) map[string]interface{} {
	snapshot := services.DecodePhotographyForecastSnapshot(plan.ForecastSnapshot)
	return map[string]interface{}{
		"id":                  plan.ID,
		"user_id":             plan.UserID,
		"name":                plan.Name,
		"date":                plan.Date,
		"session":             plan.Session,
		"location_name":       plan.LocationName,
		"latitude":            plan.Latitude,
		"longitude":           plan.Longitude,
		"timezone":            plan.Timezone,
		"weather_source":      services.NormalizePhotographyWeatherSource(plan.WeatherSource),
		"weather_source_name": services.PhotographyWeatherSourceName(plan.WeatherSource),
		"note":                plan.Note,
		"forecast_snapshot":   snapshot,
		"last_synced_at":      formatPhotographyTime(plan.LastSyncedAt),
		"sync_error":          plan.SyncError,
		"created_at":          plan.CreatedAt,
		"updated_at":          plan.UpdatedAt,
	}
}

func formatPhotographyTime(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return value.Local().Format(time.RFC3339)
}

func truncatePhotographyError(value string) string {
	const maxRunes = 500
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

type photographyError string

func (e photographyError) Error() string { return string(e) }
