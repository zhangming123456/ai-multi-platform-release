package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 空间分布图：以中心点为中心取一个 N×N 网格，一次多坐标请求拿到每格云量与降水概率，
// 复用既有评分逻辑给出「朝霞/晚霞概率」与「云量质量」，供前端地图色块渲染。
const (
	PhotographySpatialPeriodSunrise = "sunrise"
	PhotographySpatialPeriodSunset  = "sunset"

	photographySpatialRows = 11
	photographySpatialCols = 11
	// 20km 步长：11×11 = 121 个点，整体约 220km 见方，用于区域尺度的霞情分布。
	// 点数上限来自 URL 长度：Open-Meteo 多坐标查询 441 个点会被返回 414。
	photographySpatialStepKilometers = 20
	photographyKilometersPerDegree   = 111.32
	photographySpatialMaxCells       = 121
)

type PhotographySpatialCell struct {
	Latitude                 float64  `json:"latitude"`
	Longitude                float64  `json:"longitude"`
	CloudCover               *float64 `json:"cloud_cover"`
	PrecipitationProbability *float64 `json:"precipitation_probability"`
	Probability              *int     `json:"probability"`
	ProbabilityLevel         string   `json:"probability_level"`
	CloudQuality             *int     `json:"cloud_quality"`
	CloudQualityLevel        string   `json:"cloud_quality_level"`
}

type PhotographySpatialGrid struct {
	Date          string                   `json:"date"`
	Period        string                   `json:"period"`
	Rows          int                      `json:"rows"`
	Cols          int                      `json:"cols"`
	StepLatitude  float64                  `json:"step_latitude"`
	StepLongitude float64                  `json:"step_longitude"`
	Latitude      float64                  `json:"latitude"`
	Longitude     float64                  `json:"longitude"`
	Timezone      string                   `json:"timezone"`
	Source        string                   `json:"source"`
	SourceName    string                   `json:"source_name"`
	Message       string                   `json:"message"`
	Cells         []PhotographySpatialCell `json:"cells"`
}

func NormalizePhotographySpatialPeriod(period string) string {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case PhotographySpatialPeriodSunrise:
		return PhotographySpatialPeriodSunrise
	default:
		return PhotographySpatialPeriodSunset
	}
}

func FetchPhotographySpatialGrid(latitude, longitude float64, date, period, source string) (*PhotographySpatialGrid, error) {
	if err := ValidatePhotographyCoordinates(latitude, longitude); err != nil {
		return nil, err
	}
	period = NormalizePhotographySpatialPeriod(period)
	source = NormalizePhotographyWeatherSource(source)
	if err := ValidatePhotographyWeatherSource(source); err != nil {
		return nil, err
	}
	// 网格分布只走 Open-Meteo（原生支持一次多坐标查询），和风天气等单点数据源不参与。
	if !photographyWeatherProviderConfigFor(photographyWeatherConfigOpenMeteo).Enabled {
		return nil, errors.New("Open-Meteo 已停用，空间分布图暂不可用")
	}

	timezone, err := FetchPhotographyTimezone(latitude, longitude)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(date) == "" {
		date = time.Now().In(photographyTimezoneLocation(timezone)).Format(photographyDateLayout)
	}
	if err := ValidatePhotographyDateWithDays(date, timezone, time.Now(), PhotographyForecastDays); err != nil {
		return nil, err
	}

	centers, stepLatitude, stepLongitude := photographySpatialCenters(latitude, longitude)
	responses, err := fetchPhotographySpatialResponses(centers, timezone, date)
	if err != nil {
		return nil, err
	}

	grid := &PhotographySpatialGrid{
		Date:          date,
		Period:        period,
		Rows:          photographySpatialRows,
		Cols:          photographySpatialCols,
		StepLatitude:  stepLatitude,
		StepLongitude: stepLongitude,
		Latitude:      latitude,
		Longitude:     longitude,
		Timezone:      timezone,
		Source:        source,
		SourceName:    PhotographyWeatherSourceName(source),
		Cells:         make([]PhotographySpatialCell, 0, len(centers)),
	}
	for index, center := range centers {
		cell := PhotographySpatialCell{Latitude: center[0], Longitude: center[1]}
		if index < len(responses) {
			fillPhotographySpatialCell(&cell, &responses[index], date, period)
		}
		grid.Cells = append(grid.Cells, cell)
	}
	if len(grid.Cells) == 0 {
		return nil, errors.New("空间分布图未生成有效网格")
	}
	// 网格固定走 Open-Meteo 系数据源，选中和风天气等单点源时明确告知用户。
	if !strings.HasPrefix(source, "open-meteo") {
		grid.Message = "空间分布图固定使用 Open-Meteo 多坐标数据，与所选的单点数据源不同。"
	}
	return grid, nil
}

func fillPhotographySpatialCell(cell *PhotographySpatialCell, response *openMeteoForecastResponse, date, period string) {
	index := indexOfString(response.Daily.Time, date)
	if index < 0 {
		return
	}
	event := stringAt(response.Daily.Sunrise, index)
	if period == PhotographySpatialPeriodSunset {
		event = stringAt(response.Daily.Sunset, index)
	}
	metrics := eventPhotographyMetrics(response, date, event)
	cell.CloudCover = metrics.Cloud
	cell.PrecipitationProbability = metrics.Probability

	assessment := ScorePhotographyConditions(metrics.Cloud, metrics.Probability, metrics.WeatherCode)
	score := assessment.Score
	cell.Probability = &score
	cell.ProbabilityLevel = assessment.Level

	if metrics.Cloud != nil {
		quality := cloudQuality(*metrics.Cloud)
		cell.CloudQuality = &quality
		cell.CloudQualityLevel = photographyScoreLevel(quality)
	}
}

func photographySpatialCenters(latitude, longitude float64) (centers [][2]float64, stepLatitude, stepLongitude float64) {
	stepLatitude = photographySpatialStepKilometers / photographyKilometersPerDegree
	stepLongitude = stepLatitude / math.Max(0.25, math.Cos(latitude*math.Pi/180))
	halfRows := (photographySpatialRows - 1) / 2
	halfCols := (photographySpatialCols - 1) / 2
	for row := -halfRows; row <= halfRows; row++ {
		for col := -halfCols; col <= halfCols; col++ {
			cellLatitude := math.Max(-90, math.Min(90, latitude+float64(row)*stepLatitude))
			cellLongitude := normalizePhotographyLongitude(longitude + float64(col)*stepLongitude)
			centers = append(centers, [2]float64{cellLatitude, cellLongitude})
		}
	}
	if len(centers) > photographySpatialMaxCells {
		centers = centers[:photographySpatialMaxCells]
	}
	return centers, stepLatitude, stepLongitude
}

func normalizePhotographyLongitude(longitude float64) float64 {
	for longitude > 180 {
		longitude -= 360
	}
	for longitude < -180 {
		longitude += 360
	}
	return longitude
}

func fetchPhotographySpatialResponses(centers [][2]float64, timezone, date string) ([]openMeteoForecastResponse, error) {
	endpoint, err := url.Parse(photographyForecastBaseURL())
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		if err == nil {
			err = errors.New("地址必须是完整的 HTTP 或 HTTPS 地址")
		}
		return nil, fmt.Errorf("天气接口地址无效: %w", err)
	}
	latitudes := make([]string, 0, len(centers))
	longitudes := make([]string, 0, len(centers))
	for _, center := range centers {
		latitudes = append(latitudes, formatPhotographyCoordinate(center[0]))
		longitudes = append(longitudes, formatPhotographyCoordinate(center[1]))
	}
	params := endpoint.Query()
	params.Set("latitude", strings.Join(latitudes, ","))
	params.Set("longitude", strings.Join(longitudes, ","))
	// 多坐标查询必须使用固定时区（auto 只支持单点）。
	params.Set("timezone", timezone)
	params.Set("forecast_days", strconv.Itoa(photographySpatialForecastDays(date, timezone)))
	params.Set("daily", "sunrise,sunset")
	params.Set("hourly", "cloud_cover,precipitation_probability,weather_code")
	appendOpenMeteoAPIKey(params)
	endpoint.RawQuery = params.Encode()

	var raw json.RawMessage
	if err := fetchPhotographyJSON(endpoint.String(), &raw); err != nil {
		return nil, err
	}
	body := bytes.TrimSpace(raw)
	if len(body) == 0 {
		return nil, errors.New("天气接口未返回空间分布数据")
	}
	if body[0] == '[' {
		var list []openMeteoForecastResponse
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("天气接口空间分布数据解析失败: %w", err)
		}
		return list, nil
	}
	var single openMeteoForecastResponse
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, fmt.Errorf("天气接口空间分布数据解析失败: %w", err)
	}
	return []openMeteoForecastResponse{single}, nil
}

// 只请求覆盖到目标日期所需的天数，避免固定 16 天带来的大响应体。
func photographySpatialForecastDays(date, timezone string) int {
	location := photographyTimezoneLocation(timezone)
	today := time.Now().In(location).Format(photographyDateLayout)
	start, startErr := time.ParseInLocation(photographyDateLayout, today, location)
	target, targetErr := time.ParseInLocation(photographyDateLayout, date, location)
	if startErr != nil || targetErr != nil {
		return PhotographyForecastDays
	}
	days := int(target.Sub(start).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	if days > PhotographyForecastDays {
		days = PhotographyForecastDays
	}
	return days
}

func photographyTimezoneLocation(timezone string) *time.Location {
	if loaded, err := time.LoadLocation(strings.TrimSpace(timezone)); err == nil {
		return loaded
	}
	return time.Local
}
