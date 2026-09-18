package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 中国气象局（中国天气网）与深圳气象局（深圳市数据开放平台）适配。
// 两者都只提供逐日数据，落库前统一按天气现象估算云量与降水概率，评分口径与 Open-Meteo 保持一致。
const (
	photographyCMADays        = 5
	photographyShenzhenDays   = 10
	photographyChinaTimezone  = "Asia/Shanghai"
	photographyChinaReferer   = "http://www.weather.com.cn/weather1d/%s.shtml"
	photographyDefaultCMAHost = "http://d1.weather.com.cn"
)

const photographyEstimatedReason = "云量与降水概率由天气现象推算，仅供参考"

func cmaConfigured() bool {
	return photographyWeatherProviderConfigFor(photographyWeatherConfigCMA).Enabled
}

func shenzhenWeatherConfigured() bool {
	config := photographyWeatherProviderConfigFor(photographyWeatherConfigShenzhen)
	return config.Enabled && config.APIKey != "" && config.APIHost != ""
}

// —— 中国气象局（中国天气网） ——

type chinaWeatherForecastDay struct {
	DayWeatherCode   string `json:"fa"`
	NightWeatherCode string `json:"fb"`
	TempMax          string `json:"fc"`
	TempMin          string `json:"fd"`
	HumidityDay      string `json:"fm"`
	HumidityNight    string `json:"fn"`
	Date             string `json:"fi"`
}

type chinaWeatherForecastPayload struct {
	Days []chinaWeatherForecastDay `json:"f"`
}

func fetchCMAPhotographyForecast(latitude, longitude float64, targetDate string) (*PhotographyForecast, error) {
	if !isMainlandChinaCoordinate(latitude, longitude) {
		return nil, newPhotographyValidationError("中国气象局（中国天气网）数据源仅覆盖中国大陆范围")
	}
	cityName, err := reverseGeocodeChinaCityName(latitude, longitude)
	if err != nil {
		return nil, err
	}
	cityCode, err := searchChinaWeatherCityCode(cityName)
	if err != nil {
		return nil, err
	}
	endpoint := cmaWeatherIndexEndpoint(cityCode)
	headers := make(http.Header)
	headers.Set("Referer", fmt.Sprintf(photographyChinaReferer, cityCode))
	body, err := fetchPhotographyText("china-weather", endpoint, headers)
	if err != nil {
		return nil, err
	}
	raw, err := extractPhotographyJSONVar(body, "fc")
	if err != nil {
		return nil, fmt.Errorf("中国天气网未返回可解析的逐日预报：%w", err)
	}
	var payload chinaWeatherForecastPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("中国天气网逐日预报解析失败：%w", err)
	}
	solar, timezone := fetchPhotographySolarEvents(latitude, longitude)
	response, humidity := buildOpenMeteoResponseFromChinaWeather(payload, latitude, longitude, timezone, solar)
	days := buildPhotographyForecastDays(response)
	applyPhotographyDailyEstimates(days, humidity)
	if len(days) == 0 {
		return nil, photographyUpstreamError(endpoint, "中国天气网未返回有效日期数据")
	}
	forecast := &PhotographyForecast{
		Latitude: latitude, Longitude: longitude, Timezone: timezone,
		Source: PhotographyWeatherSourceCMA, SourceName: PhotographyWeatherSourceName(PhotographyWeatherSourceCMA),
		Days: days,
	}
	if !containsPhotographyDate(days, targetDate) {
		return nil, fmt.Errorf("目标日期不在中国气象局预报范围内（%s 最多支持未来 %d 天）", cityName, photographyCMADays)
	}
	return forecast, nil
}

func cmaWeatherIndexEndpoint(cityCode string) string {
	host := strings.TrimRight(strings.TrimSpace(photographyWeatherProviderConfigFor(photographyWeatherConfigCMA).APIHost), "/")
	if host == "" {
		host = photographyDefaultCMAHost
	}
	return host + "/weather_index/" + url.PathEscape(cityCode) + ".html"
}

func chinaWeatherCitySearchEndpoint(cityName string) string {
	host := strings.TrimRight(strings.TrimSpace(os.Getenv("CMA_CITY_SEARCH_BASE_URL")), "/")
	if host == "" {
		host = "http://toy1.weather.com.cn"
	}
	params := url.Values{}
	params.Set("cityname", cityName)
	params.Set("callback", "cb")
	return host + "/search?" + params.Encode()
}

// searchChinaWeatherCityCode 用城市名换中国天气网的 101xxxxxx 城市编码。
func searchChinaWeatherCityCode(cityName string) (string, error) {
	cityName = strings.TrimSpace(cityName)
	if cityName == "" {
		return "", newPhotographyValidationError("未能根据经纬度识别城市，无法查询中国气象局数据")
	}
	for _, candidate := range chinaWeatherCityNameCandidates(cityName) {
		code, found, err := queryChinaWeatherCityCode(candidate)
		if err != nil {
			return "", err
		}
		if found {
			return code, nil
		}
	}
	return "", fmt.Errorf("中国天气网未找到城市「%s」对应的城市编码", cityName)
}

// 高德返回「深圳市」，中国天气网检索需要「深圳」。
func chinaWeatherCityNameCandidates(cityName string) []string {
	candidates := []string{cityName}
	for _, suffix := range []string{"特别行政区", "自治州", "地区", "市", "区", "县", "盟", "省"} {
		if strings.HasSuffix(cityName, suffix) && len(cityName) > len(suffix) {
			candidates = append(candidates, strings.TrimSuffix(cityName, suffix))
			break
		}
	}
	return candidates
}

func queryChinaWeatherCityCode(cityName string) (string, bool, error) {
	endpoint := chinaWeatherCitySearchEndpoint(cityName)
	headers := make(http.Header)
	headers.Set("Referer", "http://www.weather.com.cn/")
	body, err := fetchPhotographyText("china-weather-search", endpoint, headers)
	if err != nil {
		return "", false, err
	}
	start := strings.Index(body, "(")
	end := strings.LastIndex(body, ")")
	if start < 0 || end <= start {
		return "", false, fmt.Errorf("中国天气网城市检索返回异常")
	}
	var results []struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal([]byte(body[start+1:end]), &results); err != nil {
		return "", false, fmt.Errorf("中国天气网城市检索解析失败：%w", err)
	}
	for _, item := range results {
		code := strings.TrimSpace(strings.SplitN(item.Ref, "~", 2)[0])
		if isChinaWeatherCityCode(code) {
			return code, true, nil
		}
	}
	return "", false, nil
}

func isChinaWeatherCityCode(code string) bool {
	if len(code) != 9 || !strings.HasPrefix(code, "101") {
		return false
	}
	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// reverseGeocodeChinaCityName 用高德 Web 服务把经纬度换成城市名，直辖市/计划单列市回退到省份。
func reverseGeocodeChinaCityName(latitude, longitude float64) (string, error) {
	key := photographyIntegrationSecretsForUse().AMapWebServiceKey
	if key == "" {
		return "", newPhotographyValidationError("中国气象局数据源需要先配置高德『Web服务』Key，用于把经纬度换算成城市")
	}
	endpoint := strings.TrimSpace(os.Getenv("AMAP_REGEO_BASE_URL"))
	if endpoint == "" {
		endpoint = "https://restapi.amap.com/v3/geocode/regeo"
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("高德逆地理编码地址无效: %w", err)
	}
	params := parsed.Query()
	params.Set("location", formatPhotographyCoordinate(longitude)+","+formatPhotographyCoordinate(latitude))
	params.Set("key", key)
	params.Set("output", "JSON")
	parsed.RawQuery = params.Encode()

	var response struct {
		Status    string `json:"status"`
		Info      string `json:"info"`
		Regeocode struct {
			AddressComponent struct {
				Province json.RawMessage `json:"province"`
				City     json.RawMessage `json:"city"`
			} `json:"addressComponent"`
		} `json:"regeocode"`
	}
	if err := fetchPhotographyJSONWithHeaders("china-weather-regeo", parsed.String(), &response, nil); err != nil {
		return "", err
	}
	if response.Status != "1" {
		message := strings.TrimSpace(response.Info)
		if message == "" {
			message = "高德逆地理编码未返回有效结果"
		}
		return "", fmt.Errorf("高德逆地理编码失败: %s", amapGeocodingErrorMessage(message))
	}
	city := parsePhotographyMapText(response.Regeocode.AddressComponent.City)
	province := parsePhotographyMapText(response.Regeocode.AddressComponent.Province)
	name := firstPhotographyNonEmpty(city, province)
	if name == "" {
		return "", newPhotographyValidationError("未识别到所在城市，无法查询中国气象局数据")
	}
	return name, nil
}

func buildOpenMeteoResponseFromChinaWeather(payload chinaWeatherForecastPayload, latitude, longitude float64, timezone string, solar map[string]photographySolarEvent) (openMeteoForecastResponse, map[string]*float64) {
	response := openMeteoForecastResponse{Latitude: latitude, Longitude: longitude, Timezone: timezone}
	humidity := map[string]*float64{}
	location := photographyLocationFor(timezone)
	year := time.Now().In(location).Year()
	for _, item := range payload.Days {
		date := chinaWeatherForecastDate(item.Date, year, location)
		if date == "" {
			continue
		}
		dayCode := chinaWeatherCodeToWMO(item.DayWeatherCode)
		nightCode := chinaWeatherCodeToWMO(item.NightWeatherCode)
		response.Daily.Time = append(response.Daily.Time, date)
		response.Daily.WeatherCode = append(response.Daily.WeatherCode, worstWMOCode(dayCode, nightCode))
		response.Daily.TemperatureMax = append(response.Daily.TemperatureMax, photographyFloatValue(item.TempMax))
		response.Daily.TemperatureMin = append(response.Daily.TemperatureMin, photographyFloatValue(item.TempMin))
		response.Daily.Sunrise = append(response.Daily.Sunrise, solar[date].Sunrise)
		response.Daily.Sunset = append(response.Daily.Sunset, solar[date].Sunset)
		if value, ok := averagePhotographyFloats(item.HumidityDay, item.HumidityNight); ok {
			copied := value
			humidity[date] = &copied
		}
	}
	return response, humidity
}

// 中国天气网日期是 9/16 这种月/日，按地点本地年份补全，跨年时顺延一年。
func chinaWeatherForecastDate(value string, year int, location *time.Location) string {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 {
		return ""
	}
	month, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return ""
	}
	day, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return ""
	}
	candidate := time.Date(year, time.Month(month), day, 12, 0, 0, 0, location)
	now := time.Now().In(location)
	if candidate.Before(now.AddDate(0, 0, -180)) {
		candidate = candidate.AddDate(1, 0, 0)
	}
	return candidate.Format(photographyDateLayout)
}

// —— 深圳气象局（深圳市数据开放平台） ——

func fetchShenzhenPhotographyForecast(latitude, longitude float64, targetDate string) (*PhotographyForecast, error) {
	config := photographyWeatherProviderConfigFor(photographyWeatherConfigShenzhen)
	if !config.Enabled || config.APIKey == "" || config.APIHost == "" {
		return nil, newPhotographyValidationError("深圳气象局数据源需要配置数据开放平台的数据集服务地址与 AppKey，请先在系统管理 > 天气数据源 / 模型中配置")
	}
	if !isShenzhenCoordinate(latitude, longitude) {
		return nil, newPhotographyValidationError("深圳气象局数据源仅覆盖深圳市范围")
	}
	endpoint, err := shenzhenWeatherEndpoint(config.APIHost, config.APIKey)
	if err != nil {
		return nil, err
	}
	records, err := fetchShenzhenWeatherRecords(endpoint)
	if err != nil {
		return nil, err
	}
	solar, timezone := fetchPhotographySolarEvents(latitude, longitude)
	response, humidity := buildOpenMeteoResponseFromShenzhenRecords(records, latitude, longitude, timezone, solar)
	days := buildPhotographyForecastDays(response)
	applyPhotographyDailyEstimates(days, humidity)
	if len(days) == 0 {
		return nil, photographyUpstreamError(endpoint, "深圳气象局接口未返回可解析的预报数据，请确认数据集的字段包含日期与天气/温度")
	}
	forecast := &PhotographyForecast{
		Latitude: latitude, Longitude: longitude, Timezone: timezone,
		Source: PhotographyWeatherSourceShenzhen, SourceName: PhotographyWeatherSourceName(PhotographyWeatherSourceShenzhen),
		Days: days,
	}
	if !containsPhotographyDate(days, targetDate) {
		return nil, fmt.Errorf("目标日期不在深圳气象局预报范围内（最多支持未来 %d 天）", photographyShenzhenDays)
	}
	return forecast, nil
}

func shenzhenWeatherEndpoint(host, appKey string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(host))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", newPhotographyValidationError("深圳气象局 API Host 需要是完整的数据集服务地址")
	}
	params := parsed.Query()
	params.Set("appKey", appKey)
	if params.Get("page") == "" {
		params.Set("page", "1")
	}
	if params.Get("rows") == "" {
		params.Set("rows", "200")
	}
	parsed.RawQuery = params.Encode()
	return parsed.String(), nil
}

func fetchShenzhenWeatherRecords(endpoint string) ([]map[string]interface{}, error) {
	var payload struct {
		Result struct {
			Data []map[string]interface{} `json:"data"`
		} `json:"result"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := fetchPhotographyJSONWithHeaders("shenzhen-weather", endpoint, &payload, nil); err != nil {
		return nil, err
	}
	records := payload.Result.Data
	if len(records) == 0 {
		records = payload.Data
	}
	if len(records) == 0 {
		return nil, photographyUpstreamError(endpoint, "深圳气象局接口未返回数据，请确认数据集服务地址与 AppKey 是否正确")
	}
	return records, nil
}

// 开放平台不同数据集的字段命名差异很大，这里按常见中英文字段名取第一个可用值。
func buildOpenMeteoResponseFromShenzhenRecords(records []map[string]interface{}, latitude, longitude float64, timezone string, solar map[string]photographySolarEvent) (openMeteoForecastResponse, map[string]*float64) {
	type dailyAggregate struct {
		date           string
		weatherCode    int
		temperatureMax float64
		temperatureMin float64
		humidityTotal  float64
		humidityCount  int
		hasTemperature bool
	}
	order := make([]string, 0, len(records))
	aggregates := map[string]*dailyAggregate{}
	for _, record := range records {
		date := shenzhenRecordDate(record)
		if date == "" {
			continue
		}
		aggregate, ok := aggregates[date]
		if !ok {
			aggregate = &dailyAggregate{date: date, weatherCode: 3}
			aggregates[date] = aggregate
			order = append(order, date)
		}
		if code, ok := shenzhenRecordWeatherCode(record); ok {
			aggregate.weatherCode = worstWMOCode(aggregate.weatherCode, code)
		}
		if value, ok := shenzhenRecordNumber(record, "temp", "temperature", "temperatureMax", "tem", "温度", "气温", "最高温度", "最高气温"); ok {
			if !aggregate.hasTemperature || value > aggregate.temperatureMax {
				aggregate.temperatureMax = value
			}
			if !aggregate.hasTemperature || value < aggregate.temperatureMin {
				aggregate.temperatureMin = value
			}
			aggregate.hasTemperature = true
		}
		if value, ok := shenzhenRecordNumber(record, "humidity", "rhu", "shidu", "湿度", "相对湿度"); ok {
			aggregate.humidityTotal += value
			aggregate.humidityCount++
		}
	}
	response := openMeteoForecastResponse{Latitude: latitude, Longitude: longitude, Timezone: timezone}
	humidity := map[string]*float64{}
	for _, date := range order {
		aggregate := aggregates[date]
		response.Daily.Time = append(response.Daily.Time, date)
		response.Daily.WeatherCode = append(response.Daily.WeatherCode, aggregate.weatherCode)
		response.Daily.TemperatureMax = append(response.Daily.TemperatureMax, aggregate.temperatureMax)
		response.Daily.TemperatureMin = append(response.Daily.TemperatureMin, aggregate.temperatureMin)
		response.Daily.Sunrise = append(response.Daily.Sunrise, solar[date].Sunrise)
		response.Daily.Sunset = append(response.Daily.Sunset, solar[date].Sunset)
		if aggregate.humidityCount > 0 {
			value := aggregate.humidityTotal / float64(aggregate.humidityCount)
			humidity[date] = &value
		}
	}
	return response, humidity
}

func shenzhenRecordDate(record map[string]interface{}) string {
	for _, key := range []string{"date", "Date", "dataTime", "datetime", "observationTime", "日期", "观测日期", "预报日期", "更新时间", "时间"} {
		value, ok := record[key]
		if !ok {
			continue
		}
		if date := parsePhotographyRecordDate(fmt.Sprintf("%v", value)); date != "" {
			return date
		}
	}
	return ""
}

func parsePhotographyRecordDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02", "20060102", "2006/01/02", "2006年01月02日"} {
		if parsed, err := time.Parse(layout, value[:min(len(value), len(layout))]); err == nil {
			return parsed.Format(photographyDateLayout)
		}
	}
	for _, layout := range []string{"2006年01月02日", "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format(photographyDateLayout)
		}
	}
	return ""
}

func shenzhenRecordWeatherCode(record map[string]interface{}) (int, bool) {
	for _, key := range []string{"weather", "weatherText", "wea", "weatherPhenomenon", "天气现象", "天气", "现象", "天气状况"} {
		value, ok := record[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" || text == "nil" {
			continue
		}
		return chinaWeatherTextToWMO(text), true
	}
	return 0, false
}

func shenzhenRecordNumber(record map[string]interface{}, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := record[key]
		if !ok {
			continue
		}
		if number, ok := parsePhotographyNumberText(fmt.Sprintf("%v", value)); ok {
			return number, true
		}
	}
	return 0, false
}

// —— 公共辅助 ——

// applyPhotographyDailyEstimates 给只有逐日数据的来源补上湿度、按天气现象估算云量与降水概率，并重算评分。
func applyPhotographyDailyEstimates(days []PhotographyForecastDay, humidity map[string]*float64) {
	for index := range days {
		day := &days[index]
		if day.RelativeHumidity == nil {
			day.RelativeHumidity = humidity[day.Date]
		}
		estimated := false
		if day.CloudCover == nil {
			day.CloudCover = photographyEstimatedCloudCover(day.WeatherCode)
			estimated = true
		}
		if day.PrecipitationProbability == nil {
			day.PrecipitationProbability = photographyEstimatedRainProbability(day.WeatherCode)
			estimated = true
		}
		if len(day.Hours) > 0 {
			// 有逐小时数据时评分按拍摄窗口计算，估算值只做展示兜底。
			continue
		}
		sunrise := ScorePhotographyConditions(day.CloudCover, day.PrecipitationProbability, day.WeatherCode)
		sunset := ScorePhotographyConditions(day.CloudCover, day.PrecipitationProbability, day.WeatherCode)
		day.SunriseScore, day.SunriseLevel, day.SunriseReasons = sunrise.Score, sunrise.Level, sunrise.Reasons
		day.SunsetScore, day.SunsetLevel, day.SunsetReasons = sunset.Score, sunset.Level, sunset.Reasons
		if estimated {
			day.SunriseReasons = append(day.SunriseReasons, photographyEstimatedReason)
			day.SunsetReasons = append(day.SunsetReasons, photographyEstimatedReason)
		}
	}
}

func photographyEstimatedCloudCover(code *int) *float64 {
	if code == nil {
		return nil
	}
	value := 55.0
	switch {
	case *code == 0:
		value = 8
	case *code == 1 || *code == 2:
		value = 35
	case *code == 3:
		value = 92
	case *code == 45 || *code == 48:
		value = 95
	case isWetWeatherCode(*code):
		value = 90
	}
	return &value
}

func photographyEstimatedRainProbability(code *int) *float64 {
	if code == nil {
		return nil
	}
	value := 10.0
	switch {
	case *code == 3:
		value = 25
	case *code == 45 || *code == 48:
		value = 20
	case isWetWeatherCode(*code):
		value = 75
	}
	return &value
}

// 中国天气网天气现象代码（00 晴、01 多云、02 阴、03 阵雨…）映射到 WMO 代码。
func chinaWeatherCodeToWMO(code string) int {
	trimmed := strings.TrimLeft(strings.TrimSpace(code), "0")
	if trimmed == "" {
		return 0
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 3
	}
	switch {
	case value == 0:
		return 0
	case value == 1:
		return 2
	case value == 2:
		return 3
	case value == 3:
		return 80
	case value == 4:
		return 95
	case value == 5:
		return 96
	case value == 6:
		return 68
	case value == 7 || value == 21:
		return 61
	case value == 8 || value == 22:
		return 63
	case value == 9 || value == 10 || value == 11 || value == 12 || value == 23 || value == 24 || value == 25:
		return 65
	case value == 13:
		return 85
	case value == 14 || value == 26:
		return 71
	case value == 15 || value == 27:
		return 73
	case value == 16 || value == 17 || value == 28:
		return 75
	case value == 19:
		return 66
	case value >= 29 && value <= 31:
		return 45
	default:
		return 3
	}
}

func chinaWeatherTextToWMO(text string) int {
	switch {
	case strings.Contains(text, "雷"):
		return 95
	case strings.Contains(text, "雪"):
		return 71
	case strings.Contains(text, "暴雨") || strings.Contains(text, "大雨"):
		return 65
	case strings.Contains(text, "中雨") || strings.Contains(text, "雨夹雪"):
		return 63
	case strings.Contains(text, "雨"):
		return 61
	case strings.Contains(text, "雾") || strings.Contains(text, "霾") || strings.Contains(text, "沙") || strings.Contains(text, "尘"):
		return 45
	case strings.Contains(text, "阴"):
		return 3
	case strings.Contains(text, "云"):
		return 2
	case strings.Contains(text, "晴"):
		return 0
	default:
		return 3
	}
}

type photographySolarEvent struct {
	Sunrise string
	Sunset  string
}

// 日出日落属于天文数据，各数据源差异极小，统一用 Open-Meteo 补齐；失败时留空，不影响评分兜底。
func fetchPhotographySolarEvents(latitude, longitude float64) (map[string]photographySolarEvent, string) {
	events := map[string]photographySolarEvent{}
	timezone := photographyChinaTimezone
	endpoint, err := url.Parse(photographyForecastBaseURL())
	if err != nil {
		return events, timezone
	}
	params := endpoint.Query()
	params.Set("latitude", formatPhotographyCoordinate(latitude))
	params.Set("longitude", formatPhotographyCoordinate(longitude))
	params.Set("daily", "sunrise,sunset")
	params.Set("timezone", "auto")
	params.Set("forecast_days", strconv.Itoa(PhotographyForecastDays))
	appendOpenMeteoAPIKey(params)
	endpoint.RawQuery = params.Encode()
	var response struct {
		Timezone string `json:"timezone"`
		Daily    struct {
			Time    []string `json:"time"`
			Sunrise []string `json:"sunrise"`
			Sunset  []string `json:"sunset"`
		} `json:"daily"`
	}
	if err := fetchPhotographyJSONWithHeaders("open-meteo", endpoint.String(), &response, nil); err != nil {
		return events, timezone
	}
	if value := strings.TrimSpace(response.Timezone); value != "" {
		timezone = value
	}
	for index, date := range response.Daily.Time {
		events[date] = photographySolarEvent{Sunrise: stringAt(response.Daily.Sunrise, index), Sunset: stringAt(response.Daily.Sunset, index)}
	}
	return events, timezone
}

func photographyLocationFor(timezone string) *time.Location {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil || location == nil {
		return time.FixedZone("CST", 8*3600)
	}
	return location
}

func photographyFloatValue(value string) float64 {
	if parsed, ok := parsePhotographyNumberText(value); ok {
		return parsed
	}
	return 0
}

func averagePhotographyFloats(values ...string) (float64, bool) {
	total, count := 0.0, 0
	for _, value := range values {
		if parsed, ok := parsePhotographyNumberText(value); ok {
			total += parsed
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

// "28.9℃"、"67%"、"<3级" 这类带单位的文本统一取数字部分。
func parsePhotographyNumberText(value string) (float64, bool) {
	matches := regexp.MustCompile(`-?\d+(\.\d+)?`).FindString(value)
	if matches == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(matches, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func extractPhotographyJSONVar(body, name string) (string, error) {
	marker := "var " + name + " ="
	index := strings.Index(body, marker)
	if index < 0 {
		return "", fmt.Errorf("缺少变量 %s", name)
	}
	rest := body[index+len(marker):]
	start := strings.Index(rest, "{")
	if start < 0 {
		return "", fmt.Errorf("变量 %s 不是对象", name)
	}
	depth, inString, escaped := 0, false, false
	for offset, char := range rest[start:] {
		switch {
		case escaped:
			escaped = false
		case char == '\\' && inString:
			escaped = true
		case char == '"':
			inString = !inString
		case inString:
		case char == '{':
			depth++
		case char == '}':
			depth--
			if depth == 0 {
				return rest[start : start+offset+1], nil
			}
		}
	}
	return "", fmt.Errorf("变量 %s 未闭合", name)
}

func fetchPhotographyText(source, endpoint string, headers http.Header) (string, error) {
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	if request.Header.Get("User-Agent") == "" {
		request.Header.Set("User-Agent", "ai-multi-platform-release/1.0")
	}
	response, err := photographyHTTPClient.Do(request)
	if err != nil {
		message := fmt.Sprintf("天气服务请求失败: %v", err)
		LogBackendError(source, http.MethodGet, photographyLogEndpoint(endpoint), http.StatusBadGateway, message)
		return "", fmt.Errorf("%s", message)
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if readErr != nil {
		message := fmt.Sprintf("天气服务响应读取失败: %v", readErr)
		LogBackendError(source, http.MethodGet, photographyLogEndpoint(endpoint), http.StatusBadGateway, message)
		return "", fmt.Errorf("%s", message)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := fmt.Sprintf("天气服务返回异常: %s", response.Status)
		LogBackendError(source, http.MethodGet, photographyLogEndpoint(endpoint), response.StatusCode, message)
		return "", fmt.Errorf("%s", message)
	}
	return string(body), nil
}

func isMainlandChinaCoordinate(latitude, longitude float64) bool {
	return latitude >= 18 && latitude <= 54 && longitude >= 73 && longitude <= 136
}

func isShenzhenCoordinate(latitude, longitude float64) bool {
	return latitude >= 22.3 && latitude <= 23 && longitude >= 113.6 && longitude <= 114.8
}
