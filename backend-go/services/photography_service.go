package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-multi-platform-release/backend-go/models"
)

const (
	PhotographyForecastDays = 16
	photographyQWeatherDays = 10
	photographyDateLayout   = "2006-01-02"
	photographyHTTPTimeout  = 10 * time.Second
	photographyHTTPAttempts = 3
)

var photographyHTTPClient = &http.Client{
	Timeout: photographyHTTPTimeout,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   photographyHTTPTimeout,
		ResponseHeaderTimeout: photographyHTTPTimeout,
		IdleConnTimeout:       30 * time.Second,
		ForceAttemptHTTP2:     false,
	},
}

type PhotographyLocation struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Detail      string   `json:"detail"`
	Country     string   `json:"country"`
	CountryCode string   `json:"country_code"`
	Admin1      string   `json:"admin1"`
	CityCode    string   `json:"city_code"`
	AdCode      string   `json:"adcode"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Timezone    string   `json:"timezone"`
	Elevation   *float64 `json:"elevation"`
	MapProvider string   `json:"map_provider"`
}

type PhotographyAssessment struct {
	Score   int      `json:"score"`
	Level   string   `json:"level"`
	Reasons []string `json:"reasons"`
}

type PhotographyForecastHour struct {
	Time        string   `json:"time"`
	Temperature *float64 `json:"temperature"`
	WeatherCode *int     `json:"weather_code"`
	WeatherText string   `json:"weather_text"`
}

type PhotographyForecastDay struct {
	Date                     string                    `json:"date"`
	Sunrise                  string                    `json:"sunrise"`
	Sunset                   string                    `json:"sunset"`
	SunriseScore             int                       `json:"sunrise_score"`
	SunriseLevel             string                    `json:"sunrise_level"`
	SunriseReasons           []string                  `json:"sunrise_reasons"`
	SunsetScore              int                       `json:"sunset_score"`
	SunsetLevel              string                    `json:"sunset_level"`
	SunsetReasons            []string                  `json:"sunset_reasons"`
	CloudCover               *float64                  `json:"cloud_cover"`
	PrecipitationProbability *float64                  `json:"precipitation_probability"`
	WeatherCode              *int                      `json:"weather_code"`
	WeatherText              string                    `json:"weather_text"`
	TemperatureMax           *float64                  `json:"temperature_max"`
	TemperatureMin           *float64                  `json:"temperature_min"`
	Precipitation            *float64                  `json:"precipitation"`
	RelativeHumidity         *float64                  `json:"relative_humidity"`
	WindSpeed                *float64                  `json:"wind_speed"`
	WindDirection            *float64                  `json:"wind_direction"`
	Visibility               *float64                  `json:"visibility"`
	Hours                    []PhotographyForecastHour `json:"hours"`
}

type PhotographyForecastSnapshot struct {
	Days []PhotographyForecastDay `json:"days"`
}

const (
	PhotographyWeatherSourceBestMatch = "open-meteo-best-match"
	PhotographyWeatherSourceECMWF     = "open-meteo-ecmwf"
	PhotographyWeatherSourceGFS       = "open-meteo-gfs"
	PhotographyWeatherSourceICON      = "open-meteo-icon"
	PhotographyWeatherSourceQWeather  = "qweather"
)

type PhotographyWeatherConfigView struct {
	Source         string `json:"source"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	Description    string `json:"description"`
	APIKeyMasked   string `json:"api_key_masked"`
	APIHost        string `json:"api_host"`
	CredentialType string `json:"credential_type"`
	Configured     bool   `json:"configured"`
	Enabled        bool   `json:"enabled"`
	RequiresKey    bool   `json:"requires_key"`
	SupportsHost   bool   `json:"supports_host"`
}

type photographyWeatherConfigDefinition struct {
	Source       string
	Name         string
	Provider     string
	Description  string
	RequiresKey  bool
	SupportsHost bool
}

type photographyWeatherProviderConfig struct {
	APIKey         string
	APIHost        string
	CredentialType string
	Enabled        bool
}

const (
	photographyWeatherConfigOpenMeteo = "open-meteo"
	photographyWeatherConfigQWeather  = PhotographyWeatherSourceQWeather
)

func photographyWeatherConfigDefinitions() []photographyWeatherConfigDefinition {
	return []photographyWeatherConfigDefinition{
		{Source: photographyWeatherConfigOpenMeteo, Name: "Open-Meteo", Provider: "Open-Meteo", Description: "Open-Meteo 模型共享此配置；API Key 可选。", RequiresKey: false, SupportsHost: false},
		{Source: photographyWeatherConfigQWeather, Name: "和风天气", Provider: "QWeather", Description: "用于和风天气逐日、逐小时接口。", RequiresKey: true, SupportsHost: true},
	}
}

func ListPhotographyWeatherConfigs() []PhotographyWeatherConfigView {
	items := make([]PhotographyWeatherConfigView, 0, len(photographyWeatherConfigDefinitions()))
	for _, definition := range photographyWeatherConfigDefinitions() {
		config := photographyWeatherProviderConfigFor(definition.Source)
		items = append(items, PhotographyWeatherConfigView{
			Source: definition.Source, Name: definition.Name, Provider: definition.Provider, Description: definition.Description,
			APIKeyMasked: maskPhotographyAPIKey(config.APIKey), APIHost: config.APIHost, CredentialType: config.CredentialType,
			Configured: config.APIKey != "" && config.Enabled, Enabled: config.Enabled,
			RequiresKey: definition.RequiresKey, SupportsHost: definition.SupportsHost,
		})
	}
	return items
}

func SavePhotographyWeatherConfig(source, credentialType, apiKey, apiHost string, clearAPIKey bool) error {
	definition, ok := findPhotographyWeatherConfigDefinition(source)
	if !ok {
		return newPhotographyValidationError("天气配置数据源不合法")
	}
	credentialType = strings.TrimSpace(credentialType)
	if credentialType == "" {
		credentialType = "api_key"
	}
	if credentialType != "api_key" && credentialType != "token" {
		return newPhotographyValidationError("认证方式必须为 api_key 或 token")
	}
	if !definition.RequiresKey {
		credentialType = "api_key"
	}
	apiHost = strings.TrimSpace(apiHost)
	if len([]rune(apiHost)) > 500 {
		return newPhotographyValidationError("API 地址不能超过 500 个字符")
	}
	apiKey = strings.TrimSpace(apiKey)
	if strings.Contains(apiKey, "****") {
		apiKey = ""
	}
	payload := PhotographyIntegrationConfigPayload{}
	enabled := true
	if source == PhotographyWeatherSourceQWeather {
		payload.QWeatherKey, payload.QWeatherHost, payload.QWeatherCredentialType, payload.QWeatherEnabled = apiKey, apiHost, credentialType, &enabled
		payload.ClearQWeatherKey = clearAPIKey
	} else {
		payload.OpenMeteoKey, payload.OpenMeteoHost, payload.OpenMeteoEnabled = apiKey, apiHost, &enabled
		payload.ClearOpenMeteoKey = clearAPIKey
	}
	return SavePhotographyIntegrationConfig(payload, "legacy-weather-settings")
}

func findPhotographyWeatherConfigDefinition(source string) (photographyWeatherConfigDefinition, bool) {
	for _, definition := range photographyWeatherConfigDefinitions() {
		if definition.Source == source {
			return definition, true
		}
	}
	return photographyWeatherConfigDefinition{}, false
}

func photographyWeatherProviderConfigFor(source string) photographyWeatherProviderConfig {
	config := photographyWeatherProviderConfig{Enabled: true, CredentialType: "api_key"}
	if source == photographyWeatherConfigOpenMeteo {
		config.APIKey = strings.TrimSpace(os.Getenv("OPEN_METEO_API_KEY"))
	}
	if source == photographyWeatherConfigQWeather {
		config.APIKey = strings.TrimSpace(os.Getenv("QWEATHER_API_KEY"))
		config.APIHost = strings.TrimSpace(os.Getenv("QWEATHER_API_HOST"))
		if token := strings.TrimSpace(os.Getenv("QWEATHER_API_TOKEN")); token != "" {
			config.APIKey = token
			config.CredentialType = "token"
		}
	}
	// 旧表只作为迁移兼容回退。新配置优先，避免继续依赖明文配置。
	if o := GetOrm(); o != nil && config.APIKey == "" {
		stored := models.PhotographyWeatherConfig{ID: source}
		if err := o.Read(&stored); err == nil && stored.APIKey != "" {
			config.APIKey = stored.APIKey
			if stored.APIHost != "" {
				config.APIHost = stored.APIHost
			}
			if stored.CredentialType != "" {
				config.CredentialType = stored.CredentialType
			}
			config.Enabled = stored.Enabled
		}
	}
	integration := photographyIntegrationProviderConfig(source)
	if integration.APIKey != "" {
		config.APIKey = integration.APIKey
		config.CredentialType = integration.CredentialType
	}
	if integration.APIHost != "" {
		config.APIHost = integration.APIHost
	}
	if integration.APIKey != "" || !integration.Enabled {
		config.Enabled = integration.Enabled
	}
	return config
}

func maskPhotographyAPIKey(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 8 {
		return "******"
	}
	return string(runes[:4]) + "****" + string(runes[len(runes)-4:])
}

type PhotographyWeatherSource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	MaxDays     int    `json:"max_days"`
}

type PhotographyForecast struct {
	Latitude       float64                  `json:"latitude"`
	Longitude      float64                  `json:"longitude"`
	Timezone       string                   `json:"timezone"`
	Source         string                   `json:"source"`
	SourceName     string                   `json:"source_name"`
	FallbackReason string                   `json:"fallback_reason,omitempty"`
	Days           []PhotographyForecastDay `json:"days"`
}

type openMeteoGeocodingResponse struct {
	Results []openMeteoGeocodingResult `json:"results"`
}

type openMeteoGeocodingResult struct {
	Name        string   `json:"name"`
	Country     string   `json:"country"`
	CountryCode string   `json:"country_code"`
	Admin1      string   `json:"admin1"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Timezone    string   `json:"timezone"`
	Elevation   *float64 `json:"elevation"`
}

type amapGeocodingResponse struct {
	Status   string `json:"status"`
	Info     string `json:"info"`
	Geocodes []struct {
		FormattedAddress string          `json:"formatted_address"`
		Country          json.RawMessage `json:"country"`
		Province         json.RawMessage `json:"province"`
		City             json.RawMessage `json:"city"`
		District         json.RawMessage `json:"district"`
		Location         string          `json:"location"`
		AdCode           string          `json:"adcode"`
		CityCode         string          `json:"citycode"`
	} `json:"geocodes"`
}

type googleGeocodingResponse struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	Results      []struct {
		FormattedAddress  string `json:"formatted_address"`
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		Geometry struct {
			Location struct {
				Latitude  float64 `json:"lat"`
				Longitude float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
}

type openMeteoForecastResponse struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Timezone  string   `json:"timezone"`
	Elevation *float64 `json:"elevation"`
	Current   struct {
		Time                     string  `json:"time"`
		Temperature              float64 `json:"temperature_2m"`
		RelativeHumidity         float64 `json:"relative_humidity_2m"`
		ApparentTemperature      float64 `json:"apparent_temperature"`
		PrecipitationProbability float64 `json:"precipitation_probability"`
		Precipitation            float64 `json:"precipitation"`
		WeatherCode              int     `json:"weather_code"`
		CloudCover               float64 `json:"cloud_cover"`
		Visibility               float64 `json:"visibility"`
		WindSpeed                float64 `json:"wind_speed_10m"`
		WindDirection            float64 `json:"wind_direction_10m"`
		IsDay                    int     `json:"is_day"`
	} `json:"current"`
	Daily struct {
		Time                        []string  `json:"time"`
		Sunrise                     []string  `json:"sunrise"`
		Sunset                      []string  `json:"sunset"`
		WeatherCode                 []int     `json:"weather_code"`
		PrecipitationProbabilityMax []float64 `json:"precipitation_probability_max"`
		TemperatureMax              []float64 `json:"temperature_2m_max"`
		TemperatureMin              []float64 `json:"temperature_2m_min"`
		PrecipitationSum            []float64 `json:"precipitation_sum"`
	} `json:"daily"`
	Hourly struct {
		Time                     []string  `json:"time"`
		Temperature              []float64 `json:"temperature_2m"`
		RelativeHumidity         []float64 `json:"relative_humidity_2m"`
		DewPoint                 []float64 `json:"dew_point_2m"`
		ApparentTemperature      []float64 `json:"apparent_temperature"`
		CloudCover               []float64 `json:"cloud_cover"`
		PrecipitationProbability []float64 `json:"precipitation_probability"`
		Precipitation            []float64 `json:"precipitation"`
		Rain                     []float64 `json:"rain"`
		Snowfall                 []float64 `json:"snowfall"`
		WeatherCode              []int     `json:"weather_code"`
		Visibility               []float64 `json:"visibility"`
		WindSpeed                []float64 `json:"wind_speed_10m"`
		WindDirection            []float64 `json:"wind_direction_10m"`
	} `json:"hourly"`
}

// SearchPhotographyLocations 搜索地点，并按经纬度用 Open-Meteo 补齐时区。
func SearchPhotographyLocations(query string, countryCodes ...string) ([]PhotographyLocation, error) {
	locations, err := searchPhotographyLocations(query, countryCodes...)
	if err != nil {
		return locations, err
	}
	fillPhotographyLocationTimezones(locations)
	return locations, nil
}

// 高德 / Google 地理编码都不返回时区，统一用经纬度向 Open-Meteo 查询补齐；
// 单个地点失败只留空时区，不影响搜索结果。
// ponytail: 无缓存的逐点查询，并发 4；搜索结果命中率高了再加 TTL 缓存。
func fillPhotographyLocationTimezones(locations []PhotographyLocation) {
	semaphore := make(chan struct{}, 4)
	var waitGroup sync.WaitGroup
	for index := range locations {
		if strings.TrimSpace(locations[index].Timezone) != "" {
			continue
		}
		waitGroup.Add(1)
		semaphore <- struct{}{}
		go func(target int) {
			defer waitGroup.Done()
			defer func() { <-semaphore }()
			timezone, err := fetchOpenMeteoTimezone(locations[target].Latitude, locations[target].Longitude)
			if err != nil || timezone == "" {
				return
			}
			locations[target].Timezone = timezone
		}(index)
	}
	waitGroup.Wait()
}

func fetchOpenMeteoTimezone(latitude, longitude float64) (string, error) {
	endpoint, err := url.Parse(photographyForecastBaseURL())
	if err != nil {
		return "", fmt.Errorf("Open-Meteo 天气接口地址无效: %w", err)
	}
	params := endpoint.Query()
	params.Set("latitude", formatPhotographyCoordinate(latitude))
	params.Set("longitude", formatPhotographyCoordinate(longitude))
	params.Set("timezone", "auto")
	params.Set("forecast_days", "1")
	appendOpenMeteoAPIKey(params)
	endpoint.RawQuery = params.Encode()

	var response struct {
		Timezone string `json:"timezone"`
	}
	if err := fetchPhotographyJSON(endpoint.String(), &response); err != nil {
		return "", err
	}
	timezone := strings.TrimSpace(response.Timezone)
	if timezone == "" {
		return "", fmt.Errorf("Open-Meteo 未返回地点时区")
	}
	return timezone, nil
}

// FetchPhotographyTimezone 暴露给接口层：仅按经纬度取地点时区，用于高德 JS API 搜索结果补全。
func FetchPhotographyTimezone(latitude, longitude float64) (string, error) {
	if err := ValidatePhotographyCoordinates(latitude, longitude); err != nil {
		return "", err
	}
	return fetchOpenMeteoTimezone(latitude, longitude)
}

func searchPhotographyLocations(query string, countryCodes ...string) ([]PhotographyLocation, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 {
		return nil, newPhotographyValidationError("地点关键词至少需要 2 个字符")
	}
	countryCode := ""
	if len(countryCodes) > 0 {
		countryCode = strings.TrimSpace(countryCodes[0])
	}
	provider := SelectPhotographyMapProvider(countryCode)
	locations, configured, err := searchPhotographyLocationsByMapProvider(query, provider, countryCode)
	if configured {
		if err != nil {
			LogBackendError("http", "/api/photography-tools/geocode", provider+"-geocode", 502, err.Error())
		}
		return locations, err
	}
	if provider == "amap" {
		// 中国大陆（含未指定地区）强制使用高德地理编码：不降级到其他数据源，
		// 避免同一地区出现来源不一致、精度不一的地址结果。
		return nil, fmt.Errorf("未配置高德 Web服务 Key，请在「系统管理 → 气象数据源 → 地图服务」中配置高德『Web服务』类型的 Key")
	}

	// 地图服务未配置 Key 时保留 Open-Meteo 作为无 Key 降级方案，避免天气查询入口完全不可用。
	return searchPhotographyLocationsByOpenMeteo(query)
}

// amapGeocodingErrorMessage 把高德接口返回的错误码翻译成可执行的配置提示。
func amapGeocodingErrorMessage(info string) string {
	switch info {
	case "USERKEY_PLAT_NOMATCH", "INVALID_USER_SCODE":
		return "Key 与接口平台不匹配，地址搜索需要在系统管理中配置高德『Web服务』类型的 Key"
	case "INVALID_USER_KEY", "USER_KEY_RECYCLED":
		return "高德 Web服务 Key 无效或已删除，请在系统管理中重新配置"
	case "SERVICE_NOT_AVAILABLE":
		return "高德 Web服务 Key 未开通地理编码服务"
	case "DAILY_QUERY_OVER_LIMIT", "USER_DAILY_QUERY_OVER_LIMIT":
		return "高德地址搜索今日调用量已用尽"
	default:
		return info
	}
}

func searchPhotographyLocationsByMapProvider(query, provider, countryCode string) ([]PhotographyLocation, bool, error) {
	secrets := photographyIntegrationSecretsForUse()
	var endpoint, key, source string
	switch provider {
	case "amap":
		// 高德「Web端(JS API)」Key 与「Web服务」Key 平台不互通，
		// 服务端地理编码必须使用 Web服务 Key，否则会返回 USERKEY_PLAT_NOMATCH。
		key, source = secrets.AMapWebServiceKey, "amap-geocoding"
		endpoint = strings.TrimSpace(os.Getenv("AMAP_GEOCODING_BASE_URL"))
		if endpoint == "" {
			endpoint = "https://restapi.amap.com/v3/geocode/geo"
		}
	case "google":
		key, source = secrets.GoogleMapsKey, "google-geocoding"
		endpoint = strings.TrimSpace(os.Getenv("GOOGLE_GEOCODING_BASE_URL"))
		if endpoint == "" {
			endpoint = "https://maps.googleapis.com/maps/api/geocode/json"
		}
	default:
		return nil, false, nil
	}
	if key == "" {
		return nil, false, nil
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, true, fmt.Errorf("%s 地址搜索地址无效: %w", source, err)
	}
	params := parsed.Query()
	params.Set("address", query)
	params.Set("key", key)
	if provider == "amap" {
		params.Set("output", "JSON")
	} else {
		params.Set("language", "zh-CN")
		if countryCode != "" {
			params.Set("region", strings.ToLower(countryCode))
		}
	}
	parsed.RawQuery = params.Encode()

	if provider == "amap" {
		var response amapGeocodingResponse
		if err := fetchPhotographyJSONWithHeaders(source, parsed.String(), &response, nil); err != nil {
			return nil, true, err
		}
		if response.Status != "1" {
			message := strings.TrimSpace(response.Info)
			if message == "" {
				message = "高德地址搜索未返回有效结果"
			}
			return nil, true, fmt.Errorf("高德地址搜索失败: %s", amapGeocodingErrorMessage(message))
		}
		locations := make([]PhotographyLocation, 0, len(response.Geocodes))
		for _, result := range response.Geocodes {
			latitude, longitude, ok := parsePhotographyMapLocation(result.Location)
			if !ok {
				continue
			}
			// 高德在城市级结果里会把 province / city / district 返回成空数组，统一按文本解析。
			country := parsePhotographyMapText(result.Country)
			province := parsePhotographyMapText(result.Province)
			city := parsePhotographyMapText(result.City)
			district := parsePhotographyMapText(result.District)
			name := firstPhotographyNonEmpty(district, city, province, result.FormattedAddress)
			locations = append(locations, PhotographyLocation{
				Name: name, DisplayName: buildLocationDisplayName(name, province, country),
				Detail:  strings.TrimSpace(result.FormattedAddress),
				Country: country, CountryCode: "CN", Admin1: province,
				CityCode: strings.TrimSpace(result.CityCode), AdCode: strings.TrimSpace(result.AdCode),
				Latitude: latitude, Longitude: longitude, MapProvider: "amap",
			})
		}
		return locations, true, nil
	}

	var response googleGeocodingResponse
	if err := fetchPhotographyJSONWithHeaders(source, parsed.String(), &response, nil); err != nil {
		return nil, true, err
	}
	if response.Status == "ZERO_RESULTS" {
		return []PhotographyLocation{}, true, nil
	}
	if response.Status != "OK" {
		message := strings.TrimSpace(response.ErrorMessage)
		if message == "" {
			message = response.Status
		}
		return nil, true, fmt.Errorf("Google 地址搜索失败: %s", message)
	}
	locations := make([]PhotographyLocation, 0, len(response.Results))
	for _, result := range response.Results {
		country, resultCountryCode, admin1, cityCode := "", "", "", ""
		for _, component := range result.AddressComponents {
			if containsPhotographyString(component.Types, "country") {
				country, resultCountryCode = component.LongName, component.ShortName
			}
			if containsPhotographyString(component.Types, "administrative_area_level_1") {
				admin1, cityCode = component.LongName, component.ShortName
			}
		}
		if resultCountryCode == "" {
			resultCountryCode = strings.ToUpper(countryCode)
		}
		locations = append(locations, PhotographyLocation{
			Name: result.FormattedAddress, DisplayName: result.FormattedAddress,
			Detail:  strings.TrimSpace(result.FormattedAddress),
			Country: country, CountryCode: resultCountryCode, Admin1: admin1,
			CityCode: cityCode,
			Latitude: result.Geometry.Location.Latitude, Longitude: result.Geometry.Location.Longitude,
			MapProvider: "google",
		})
	}
	return locations, true, nil
}

func searchPhotographyLocationsByOpenMeteo(query string) ([]PhotographyLocation, error) {
	parsed, err := url.Parse(photographyGeocodingBaseURL())
	if err != nil {
		return nil, fmt.Errorf("地理编码地址无效: %w", err)
	}
	params := parsed.Query()
	params.Set("name", query)
	params.Set("count", "8")
	params.Set("language", "zh")
	params.Set("format", "json")
	appendOpenMeteoAPIKey(params)
	parsed.RawQuery = params.Encode()

	var response openMeteoGeocodingResponse
	if err := fetchPhotographyJSON(parsed.String(), &response); err != nil {
		return nil, err
	}
	locations := make([]PhotographyLocation, 0, len(response.Results))
	for _, result := range response.Results {
		locations = append(locations, PhotographyLocation{
			Name: result.Name, DisplayName: buildLocationDisplayName(result.Name, result.Admin1, result.Country),
			Country: result.Country, CountryCode: result.CountryCode, Admin1: result.Admin1,
			Latitude: result.Latitude, Longitude: result.Longitude, Timezone: result.Timezone,
			Elevation: result.Elevation, MapProvider: SelectPhotographyMapProvider(result.CountryCode),
		})
	}
	return locations, nil
}

func parsePhotographyMapLocation(value string) (float64, float64, bool) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	longitude, longitudeErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	latitude, latitudeErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if longitudeErr != nil || latitudeErr != nil || ValidatePhotographyCoordinates(latitude, longitude) != nil {
		return 0, 0, false
	}
	return latitude, longitude, true
}

func parsePhotographyMapText(value json.RawMessage) string {
	var text string
	if json.Unmarshal(value, &text) == nil {
		return strings.TrimSpace(text)
	}
	var values []string
	if json.Unmarshal(value, &values) == nil {
		return strings.TrimSpace(strings.Join(values, " "))
	}
	return ""
}

func firstPhotographyNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func ListPhotographyWeatherSources() []PhotographyWeatherSource {
	secrets := photographyIntegrationSecretsForUse()
	return []PhotographyWeatherSource{
		{ID: PhotographyWeatherSourceBestMatch, Name: "自动匹配（推荐）", Provider: "Open-Meteo / 和风天气", Description: "优先使用 Open-Meteo，失败时切换已配置的备用来源", Available: secrets.OpenMeteoEnabled || qweatherConfigured(), MaxDays: PhotographyForecastDays},
		{ID: PhotographyWeatherSourceECMWF, Name: "ECMWF 欧洲中期天气预报", Provider: "Open-Meteo", Model: "ecmwf_ifs025", Description: "适合全球中期趋势判断", Available: secrets.OpenMeteoEnabled, MaxDays: PhotographyForecastDays},
		{ID: PhotographyWeatherSourceGFS, Name: "NOAA GFS", Provider: "Open-Meteo", Model: "gfs_seamless", Description: "美国 NOAA 全球预报模型", Available: secrets.OpenMeteoEnabled, MaxDays: PhotographyForecastDays},
		{ID: PhotographyWeatherSourceICON, Name: "DWD ICON", Provider: "Open-Meteo", Model: "icon_seamless", Description: "德国气象局全球预报模型", Available: secrets.OpenMeteoEnabled, MaxDays: PhotographyForecastDays},
		{ID: PhotographyWeatherSourceQWeather, Name: "和风天气", Provider: "QWeather", Description: "逐日与逐小时天气数据，最多支持未来 10 天", Available: qweatherConfigured(), MaxDays: photographyQWeatherDays},
	}
}

func NormalizePhotographyWeatherSource(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return PhotographyWeatherSourceBestMatch
	}
	for _, source := range ListPhotographyWeatherSources() {
		if source.ID == value {
			return value
		}
	}
	return ""
}

func ValidatePhotographyWeatherSource(value string) error {
	sourceID := NormalizePhotographyWeatherSource(value)
	if sourceID == "" {
		return newPhotographyValidationError("天气数据源不合法")
	}
	for _, source := range ListPhotographyWeatherSources() {
		if source.ID == sourceID && !source.Available {
			return newPhotographyValidationError("和风天气未配置凭据，请先在系统管理 > 天气数据源 / 模型中配置 API Key 或 JWT Token")
		}
	}
	return nil
}

func PhotographyWeatherSourceName(value string) string {
	sourceID := NormalizePhotographyWeatherSource(value)
	for _, source := range ListPhotographyWeatherSources() {
		if source.ID == sourceID {
			return source.Name
		}
	}
	return "自动匹配（推荐）"
}

func FetchPhotographyForecast(latitude, longitude float64, targetDate string) (*PhotographyForecast, error) {
	return FetchPhotographyForecastWithSource(latitude, longitude, targetDate, PhotographyWeatherSourceBestMatch)
}

func FetchPhotographyForecastWithSource(latitude, longitude float64, targetDate, source string) (*PhotographyForecast, error) {
	source = NormalizePhotographyWeatherSource(source)
	if err := ValidatePhotographyWeatherSource(source); err != nil {
		return nil, err
	}
	if err := ValidatePhotographyCoordinates(latitude, longitude); err != nil {
		return nil, err
	}
	if _, err := time.Parse(photographyDateLayout, targetDate); err != nil {
		return nil, newPhotographyValidationError("日期格式必须为 YYYY-MM-DD")
	}
	if source == PhotographyWeatherSourceQWeather {
		return fetchQWeatherPhotographyForecast(latitude, longitude, targetDate)
	}
	if source != PhotographyWeatherSourceBestMatch {
		return fetchOpenMeteoPhotographyForecast(latitude, longitude, targetDate, source)
	}

	openMeteoConfig := photographyWeatherProviderConfigFor(photographyWeatherConfigOpenMeteo)
	var openMeteoErr error
	if openMeteoConfig.Enabled {
		if forecast, err := fetchOpenMeteoPhotographyForecast(latitude, longitude, targetDate, source); err == nil {
			return forecast, nil
		} else {
			openMeteoErr = err
		}
	} else {
		openMeteoErr = errors.New("Open-Meteo 已停用")
	}
	if qweatherConfigured() {
		if forecast, err := fetchQWeatherPhotographyForecast(latitude, longitude, targetDate); err == nil {
			forecast.FallbackReason = fmt.Sprintf("Open-Meteo 不可用，已切换至和风天气：%s", summarizePhotographyError(openMeteoErr))
			return forecast, nil
		} else {
			return nil, fmt.Errorf("自动匹配天气服务失败：Open-Meteo：%v；和风天气：%v", openMeteoErr, err)
		}
	}
	return nil, fmt.Errorf("Open-Meteo 天气服务不可用：%w；如需自动切换，请配置和风天气 API Key 或 JWT Token", openMeteoErr)
}

func fetchOpenMeteoPhotographyForecast(latitude, longitude float64, targetDate, source string) (*PhotographyForecast, error) {
	_, forecast, err := fetchOpenMeteoPhotographyData(latitude, longitude, targetDate, source)
	return forecast, err
}

func fetchOpenMeteoPhotographyData(latitude, longitude float64, targetDate, source string) (*openMeteoForecastResponse, *PhotographyForecast, error) {
	parsed, err := url.Parse(photographyForecastBaseURL())
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		if err == nil {
			err = errors.New("地址必须是完整的 HTTP 或 HTTPS 地址")
		}
		return nil, nil, fmt.Errorf("天气接口地址无效: %w", err)
	}
	params := parsed.Query()
	params.Set("latitude", strconv.FormatFloat(latitude, 102, 6, 64))
	params.Set("longitude", strconv.FormatFloat(longitude, 102, 6, 64))
	params.Set("timezone", "auto")
	params.Set("forecast_days", strconv.Itoa(PhotographyForecastDays))
	if model := photographyWeatherSourceModel(source); model != "" {
		params.Set("models", model)
	}
	params.Set("current", "temperature_2m,relative_humidity_2m,apparent_temperature,precipitation_probability,precipitation,weather_code,cloud_cover,visibility,wind_speed_10m,wind_direction_10m,is_day")
	params.Set("daily", "sunrise,sunset,weather_code,precipitation_probability_max,temperature_2m_max,temperature_2m_min,precipitation_sum")
	params.Set("hourly", "temperature_2m,relative_humidity_2m,dew_point_2m,apparent_temperature,cloud_cover,precipitation_probability,precipitation,rain,snowfall,weather_code,visibility,wind_speed_10m,wind_direction_10m")
	appendOpenMeteoAPIKey(params)
	parsed.RawQuery = params.Encode()

	var response openMeteoForecastResponse
	if err := fetchPhotographyJSON(parsed.String(), &response); err != nil {
		return nil, nil, err
	}
	if response.Timezone == "" {
		response.Timezone = "UTC"
	}
	if targetDate != "" {
		if err := ValidatePhotographyDateWithDays(targetDate, response.Timezone, time.Now(), PhotographyForecastDays); err != nil {
			return nil, nil, err
		}
	}

	forecast := &PhotographyForecast{
		Latitude:   response.Latitude,
		Longitude:  response.Longitude,
		Timezone:   response.Timezone,
		Source:     source,
		SourceName: PhotographyWeatherSourceName(source),
		Days:       buildPhotographyForecastDays(response),
	}
	if len(forecast.Days) == 0 {
		return nil, nil, fmt.Errorf("天气接口未返回有效日期数据")
	}
	if targetDate != "" && !containsPhotographyDate(forecast.Days, targetDate) {
		return nil, nil, fmt.Errorf("目标日期不在未来 %d 天预报范围内", PhotographyForecastDays)
	}
	return &response, forecast, nil
}

type qWeatherGeoResponse struct {
	Code     string
	Location []struct{ TZ string }
}

type qWeatherDailyResponse struct {
	Code  string
	Days  []qWeatherDailyItem
	Daily []qWeatherDailyItem
}

type qWeatherDailyItem struct {
	ForecastStartTime string
	FxDate            string
	Sunrise           string
	Sunset            string
	IconDay           string
	IconNight         string
	Cloud             string
	Pop               string
	Precip            string
	Vis               string
	TempMax           string
	TempMin           string
	Astro             struct {
		Sunrise string
		Sunset  string
	}
	Daytime   qWeatherPeriod
	Nighttime qWeatherPeriod
}

type qWeatherPeriod struct {
	Condition     struct{ Code string }
	CloudCover    float64
	Precipitation struct {
		Probability float64
		Amount      struct{ Value float64 }
	}
}

type qWeatherHourlyResponse struct {
	Code   string
	Hours  []qWeatherHourlyItem
	Hourly []qWeatherHourlyItem
}

type qWeatherHourlyItem struct {
	ForecastTime string
	FxTime       string
	Temp         string
	Humidity     string
	Dew          string
	Vis          string
	Cloud        string
	Pop          string
	Precip       string
	WindSpeed    string
	Condition    struct {
		Code string
		Icon string
	}
	Icon          string
	CloudCover    float64
	Precipitation struct {
		Probability float64
		Amount      struct{ Value float64 }
	}
}

func fetchQWeatherPhotographyForecast(latitude, longitude float64, targetDate string) (*PhotographyForecast, error) {
	if !qweatherConfigured() {
		return nil, newPhotographyValidationError("和风天气未配置凭据，请先在系统管理 > 天气数据源 / 模型中配置 API Key 或 JWT Token")
	}
	if forecast, err := fetchQWeatherFormalForecast(latitude, longitude, targetDate); err == nil {
		return forecast, nil
	} else if !qWeatherEndpointNotFound(err) {
		return nil, err
	}
	return fetchQWeatherLegacyForecast(latitude, longitude, targetDate)
}

func fetchQWeatherFormalForecast(latitude, longitude float64, targetDate string) (*PhotographyForecast, error) {
	timezone, err := fetchQWeatherTimezone(latitude, longitude)
	if err != nil {
		return nil, err
	}
	if err := ValidatePhotographyDateWithDays(targetDate, timezone, time.Now(), photographyQWeatherDays); err != nil {
		return nil, err
	}
	dailyURL, err := url.Parse(qweatherAPIEndpoint("/v7/weather/10d"))
	if err != nil {
		return nil, fmt.Errorf("和风天气逐日接口地址无效: %w", err)
	}
	dailyParams := dailyURL.Query()
	dailyParams.Set("location", formatPhotographyCoordinate(longitude)+","+formatPhotographyCoordinate(latitude))
	dailyParams.Set("lang", "zh")
	dailyParams.Set("unit", "m")
	appendQWeatherCredential(dailyParams)
	dailyURL.RawQuery = dailyParams.Encode()
	var daily qWeatherDailyResponse
	if err := fetchQWeatherJSON(dailyURL.String(), &daily); err != nil {
		return nil, err
	}
	if daily.Code != "" && daily.Code != "200" {
		return nil, photographyUpstreamError(dailyURL.String(), fmt.Sprintf("和风天气逐日接口返回异常: %s", daily.Code))
	}
	items := daily.Daily
	if len(items) == 0 {
		items = daily.Days
	}
	if len(items) == 0 {
		return nil, photographyUpstreamError(dailyURL.String(), "和风天气未返回有效逐日数据")
	}
	hourlyURL, err := url.Parse(qweatherAPIEndpoint("/v7/weather/24h"))
	if err != nil {
		return nil, fmt.Errorf("和风天气逐小时接口地址无效: %w", err)
	}
	hourlyParams := hourlyURL.Query()
	hourlyParams.Set("location", formatPhotographyCoordinate(longitude)+","+formatPhotographyCoordinate(latitude))
	hourlyParams.Set("lang", "zh")
	hourlyParams.Set("unit", "m")
	appendQWeatherCredential(hourlyParams)
	hourlyURL.RawQuery = hourlyParams.Encode()
	var hourly qWeatherHourlyResponse
	if err := fetchQWeatherJSON(hourlyURL.String(), &hourly); err != nil {
		return nil, err
	}
	if hourly.Code != "" && hourly.Code != "200" {
		return nil, photographyUpstreamError(hourlyURL.String(), fmt.Sprintf("和风天气逐小时接口返回异常: %s", hourly.Code))
	}
	hours := hourly.Hourly
	if len(hours) == 0 {
		hours = hourly.Hours
	}
	response := buildOpenMeteoResponseFromQWeather(latitude, longitude, timezone, items, hours)
	forecast := &PhotographyForecast{Latitude: latitude, Longitude: longitude, Timezone: timezone, Source: PhotographyWeatherSourceQWeather, SourceName: PhotographyWeatherSourceName(PhotographyWeatherSourceQWeather), Days: buildPhotographyForecastDays(response)}
	if len(forecast.Days) == 0 {
		return nil, photographyUpstreamError(hourlyURL.String(), "和风天气未返回有效日期数据")
	}
	if !containsPhotographyDate(forecast.Days, targetDate) {
		return nil, fmt.Errorf("目标日期不在未来 %d 天预报范围内", photographyQWeatherDays)
	}
	return forecast, nil
}

func fetchQWeatherLegacyForecast(latitude, longitude float64, targetDate string) (*PhotographyForecast, error) {
	timezone, err := fetchQWeatherTimezone(latitude, longitude)
	if err != nil {
		return nil, err
	}
	if err := ValidatePhotographyDateWithDays(targetDate, timezone, time.Now(), photographyQWeatherDays); err != nil {
		return nil, err
	}
	dailyURL, err := url.Parse(qweatherAPIEndpoint("/weather/v1/daily/" + formatPhotographyCoordinate(latitude) + "/" + formatPhotographyCoordinate(longitude)))
	if err != nil {
		return nil, fmt.Errorf("和风天气逐日接口地址无效: %w", err)
	}
	dailyParams := dailyURL.Query()
	dailyParams.Set("days", strconv.Itoa(photographyQWeatherDays))
	dailyParams.Set("lang", "zh")
	dailyParams.Set("localTime", "true")
	appendQWeatherCredential(dailyParams)
	dailyURL.RawQuery = dailyParams.Encode()
	var daily qWeatherDailyResponse
	if err := fetchQWeatherJSON(dailyURL.String(), &daily); err != nil {
		return nil, err
	}
	if daily.Code != "" && daily.Code != "200" {
		return nil, photographyUpstreamError(dailyURL.String(), fmt.Sprintf("和风天气逐日接口返回异常: %s", daily.Code))
	}
	if len(daily.Days) == 0 {
		return nil, photographyUpstreamError(dailyURL.String(), "和风天气未返回有效逐日数据")
	}
	hourlyURL, err := url.Parse(qweatherAPIEndpoint("/weather/v1/hourly/" + formatPhotographyCoordinate(latitude) + "/" + formatPhotographyCoordinate(longitude)))
	if err != nil {
		return nil, fmt.Errorf("和风天气逐小时接口地址无效: %w", err)
	}
	hourlyParams := hourlyURL.Query()
	hourlyParams.Set("hours", strconv.Itoa(photographyQWeatherDays*24))
	hourlyParams.Set("lang", "zh")
	hourlyParams.Set("localTime", "true")
	appendQWeatherCredential(hourlyParams)
	hourlyURL.RawQuery = hourlyParams.Encode()
	var hourly qWeatherHourlyResponse
	if err := fetchQWeatherJSON(hourlyURL.String(), &hourly); err != nil {
		return nil, err
	}
	if hourly.Code != "" && hourly.Code != "200" {
		return nil, photographyUpstreamError(hourlyURL.String(), fmt.Sprintf("和风天气逐小时接口返回异常: %s", hourly.Code))
	}
	response := buildOpenMeteoResponseFromQWeather(latitude, longitude, timezone, daily.Days, hourly.Hours)
	forecast := &PhotographyForecast{Latitude: latitude, Longitude: longitude, Timezone: timezone, Source: PhotographyWeatherSourceQWeather, SourceName: PhotographyWeatherSourceName(PhotographyWeatherSourceQWeather), Days: buildPhotographyForecastDays(response)}
	if len(forecast.Days) == 0 {
		return nil, photographyUpstreamError(hourlyURL.String(), "和风天气未返回有效日期数据")
	}
	if !containsPhotographyDate(forecast.Days, targetDate) {
		return nil, fmt.Errorf("目标日期不在未来 %d 天预报范围内", photographyQWeatherDays)
	}
	return forecast, nil
}

func qWeatherEndpointNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "404") || strings.Contains(message, "not found")
}

func appendQWeatherCredential(params url.Values) {
	config := photographyWeatherProviderConfigFor(PhotographyWeatherSourceQWeather)
	if config.CredentialType == "api_key" && config.APIKey != "" {
		params.Set("key", config.APIKey)
	}
}

func fetchQWeatherTimezone(latitude, longitude float64) (string, error) {
	endpoint, err := url.Parse(qweatherGeoAPIEndpoint("/geo/v2/city/lookup"))
	if err != nil {
		return "", fmt.Errorf("和风天气地点接口地址无效: %w", err)
	}
	params := endpoint.Query()
	params.Set("location", formatPhotographyCoordinate(longitude)+","+formatPhotographyCoordinate(latitude))
	params.Set("number", "1")
	params.Set("lang", "zh")
	appendQWeatherCredential(params)
	endpoint.RawQuery = params.Encode()
	var response qWeatherGeoResponse
	if err := fetchQWeatherJSON(endpoint.String(), &response); err != nil {
		return "", err
	}
	if response.Code != "" && response.Code != "200" {
		return "", fmt.Errorf("和风天气地点接口返回异常: %s", response.Code)
	}
	if len(response.Location) == 0 || strings.TrimSpace(response.Location[0].TZ) == "" {
		return "", fmt.Errorf("和风天气未返回地点时区")
	}
	return strings.TrimSpace(response.Location[0].TZ), nil
}

func buildOpenMeteoResponseFromQWeather(latitude, longitude float64, timezone string, daily []qWeatherDailyItem, hourly []qWeatherHourlyItem) openMeteoForecastResponse {
	response := openMeteoForecastResponse{Latitude: latitude, Longitude: longitude, Timezone: timezone}
	for _, item := range daily {
		date := strings.TrimSpace(item.FxDate)
		if date == "" {
			date = qWeatherDateFromTime(item.Astro.Sunrise)
		}
		if date == "" {
			date = qWeatherDateFromTime(item.ForecastStartTime)
		}
		if date == "" {
			continue
		}
		sunrise := qWeatherEventTime(date, item.Sunrise)
		if sunrise == "" {
			sunrise = qWeatherEventTime(date, item.Astro.Sunrise)
		}
		sunset := qWeatherEventTime(date, item.Sunset)
		if sunset == "" {
			sunset = qWeatherEventTime(date, item.Astro.Sunset)
		}
		response.Daily.Time = append(response.Daily.Time, date)
		response.Daily.Sunrise = append(response.Daily.Sunrise, sunrise)
		response.Daily.Sunset = append(response.Daily.Sunset, sunset)
		dayCode := qWeatherIconToWMO(item.IconDay)
		if dayCode == 3 && item.Daytime.Condition.Code != "" {
			dayCode = qWeatherIconToWMO(item.Daytime.Condition.Code)
		}
		nightCode := qWeatherIconToWMO(item.IconNight)
		if nightCode == 3 && item.Nighttime.Condition.Code != "" {
			nightCode = qWeatherIconToWMO(item.Nighttime.Condition.Code)
		}
		response.Daily.WeatherCode = append(response.Daily.WeatherCode, worstWMOCode(dayCode, nightCode))
		probability, ok := qWeatherStringNumber(item.Pop)
		if !ok {
			probability = qWeatherPercentValue(item.Daytime.Precipitation.Probability)
		}
		if nightProbability := qWeatherPercentValue(item.Nighttime.Precipitation.Probability); nightProbability > probability {
			probability = nightProbability
		}
		response.Daily.PrecipitationProbabilityMax = append(response.Daily.PrecipitationProbabilityMax, probability)
		if value, ok := qWeatherStringNumber(item.TempMax); ok {
			response.Daily.TemperatureMax = append(response.Daily.TemperatureMax, value)
		} else {
			response.Daily.TemperatureMax = append(response.Daily.TemperatureMax, 0)
		}
		if value, ok := qWeatherStringNumber(item.TempMin); ok {
			response.Daily.TemperatureMin = append(response.Daily.TemperatureMin, value)
		} else {
			response.Daily.TemperatureMin = append(response.Daily.TemperatureMin, 0)
		}
		if value, ok := qWeatherStringNumber(item.Precip); ok {
			response.Daily.PrecipitationSum = append(response.Daily.PrecipitationSum, value)
		} else {
			response.Daily.PrecipitationSum = append(response.Daily.PrecipitationSum, 0)
		}
	}
	for _, item := range hourly {
		forecastTime := item.FxTime
		if forecastTime == "" {
			forecastTime = item.ForecastTime
		}
		if forecastTime == "" {
			continue
		}
		response.Hourly.Time = append(response.Hourly.Time, forecastTime)
		cloud, ok := qWeatherStringNumber(item.Cloud)
		if !ok {
			cloud = qWeatherPercentValue(item.CloudCover)
		}
		response.Hourly.CloudCover = append(response.Hourly.CloudCover, qWeatherPercentValue(cloud))
		probability, ok := qWeatherStringNumber(item.Pop)
		if !ok {
			probability = qWeatherPercentValue(item.Precipitation.Probability)
		}
		response.Hourly.PrecipitationProbability = append(response.Hourly.PrecipitationProbability, probability)
		precipitation, ok := qWeatherStringNumber(item.Precip)
		if !ok {
			precipitation = item.Precipitation.Amount.Value
		}
		response.Hourly.Precipitation = append(response.Hourly.Precipitation, precipitation)
		code := item.Condition.Code
		if code == "" {
			code = item.Icon
		}
		response.Hourly.WeatherCode = append(response.Hourly.WeatherCode, qWeatherIconToWMO(code))
		if value, ok := qWeatherStringNumber(item.Temp); ok {
			response.Hourly.Temperature = append(response.Hourly.Temperature, value)
		} else {
			response.Hourly.Temperature = append(response.Hourly.Temperature, 0)
		}
		if value, ok := qWeatherStringNumber(item.Humidity); ok {
			response.Hourly.RelativeHumidity = append(response.Hourly.RelativeHumidity, value)
		} else {
			response.Hourly.RelativeHumidity = append(response.Hourly.RelativeHumidity, 0)
		}
		if value, ok := qWeatherStringNumber(item.Dew); ok {
			response.Hourly.DewPoint = append(response.Hourly.DewPoint, value)
		} else {
			response.Hourly.DewPoint = append(response.Hourly.DewPoint, 0)
		}
		if value, ok := qWeatherStringNumber(item.Vis); ok {
			response.Hourly.Visibility = append(response.Hourly.Visibility, value*1000)
		} else {
			response.Hourly.Visibility = append(response.Hourly.Visibility, 0)
		}
		if value, ok := qWeatherStringNumber(item.WindSpeed); ok {
			response.Hourly.WindSpeed = append(response.Hourly.WindSpeed, value)
		} else {
			response.Hourly.WindSpeed = append(response.Hourly.WindSpeed, 0)
		}
	}
	return response
}

func qWeatherEventTime(date, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, date) {
		return value
	}
	if len(value) >= 5 && value[2] == ':' {
		return date + "T" + value[:5]
	}
	return value
}

func qWeatherStringNumber(value string) (float64, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	if value == "" || value == "-" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func qWeatherDateFromTime(value string) string {
	if len(value) < len(photographyDateLayout) {
		return ""
	}
	return value[:len(photographyDateLayout)]
}

func qWeatherPercentValue(value float64) float64 {
	if value >= 0 && value <= 1 {
		value *= 100
	}
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func qweatherConfigured() bool {
	config := photographyWeatherProviderConfigFor(photographyWeatherConfigQWeather)
	return config.Enabled && config.APIKey != ""
}

func qweatherGeoAPIEndpoint(path string) string {
	host := strings.TrimRight(strings.TrimSpace(os.Getenv("QWEATHER_GEO_API_HOST")), "/")
	if host == "" {
		host = strings.TrimRight(strings.TrimSpace(os.Getenv("QWEATHER_API_HOST")), "/")
	}
	if host == "" {
		host = "https://geoapi.qweather.com"
	}
	return host + "/" + strings.TrimLeft(path, "/")
}

func qweatherAPIEndpoint(path string) string {
	host := strings.TrimRight(photographyWeatherProviderConfigFor(photographyWeatherConfigQWeather).APIHost, "/")
	if host == "" {
		host = "https://devapi.qweather.com"
	}
	return host + "/" + strings.TrimLeft(path, "/")
}

func formatPhotographyCoordinate(value float64) string { return strconv.FormatFloat(value, 'f', 6, 64) }

func photographyFloat64Pointer(value float64) *float64 { return &value }

func qWeatherPercent(value string) *float64 {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	if value == "" || value == "-" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	if parsed >= 0 && parsed <= 1 {
		parsed *= 100
	}
	if parsed < 0 {
		parsed = 0
	}
	if parsed > 100 {
		parsed = 100
	}
	return &parsed
}

func qWeatherNumber(value string) *float64 {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	if value == "" || value == "-" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func qWeatherIconToWMO(icon string) int {
	code, err := strconv.Atoi(strings.TrimSpace(icon))
	if err != nil {
		return 3
	}
	switch {
	case code == 100 || code == 150:
		return 0
	case (code >= 101 && code <= 103) || (code >= 151 && code <= 153):
		return 2
	case code == 104 || code == 154:
		return 3
	case code >= 300 && code <= 399:
		return 61
	case code >= 400 && code <= 499:
		return 71
	case code >= 500 && code <= 515:
		return 45
	case code == 900 || code == 901:
		return 3
	default:
		return 3
	}
}

func worstWMOCode(left, right int) int {
	if weatherCodeSeverity(right) > weatherCodeSeverity(left) {
		return right
	}
	return left
}

func fetchQWeatherJSON(endpoint string, target interface{}) error {
	return fetchPhotographyJSONWithHeaders("qweather", endpoint, target, qWeatherHeaders())
}

func photographyUpstreamError(endpoint, message string) error {
	LogBackendError("qweather", http.MethodGet, photographyLogEndpoint(endpoint), http.StatusBadGateway, message)
	return fmt.Errorf("%s", message)
}

func qWeatherHeaders() http.Header {
	headers := make(http.Header)
	headers.Set("User-Agent", "ai-multi-platform-release/1.0")
	config := photographyWeatherProviderConfigFor(photographyWeatherConfigQWeather)
	if config.CredentialType == "token" && config.APIKey != "" {
		headers.Set("Authorization", "Bearer "+config.APIKey)
	} else if config.APIKey != "" {
		headers.Set("X-QW-Api-Key", config.APIKey)
	}
	return headers
}

func ValidatePhotographyCoordinates(latitude, longitude float64) error {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 {
		return newPhotographyValidationError("纬度必须在 -90 至 90 之间")
	}
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return newPhotographyValidationError("经度必须在 -180 至 180 之间")
	}
	return nil
}

func ValidatePhotographyDate(dateValue, timezoneName string, now time.Time) error {
	return ValidatePhotographyDateWithDays(dateValue, timezoneName, now, PhotographyForecastDays)
}

func ValidatePhotographyDateWithDays(dateValue, timezoneName string, now time.Time, forecastDays int) error {
	if forecastDays < 1 {
		forecastDays = PhotographyForecastDays
	}
	location := time.Local
	if timezoneName != "" {
		if loaded, loadErr := time.LoadLocation(timezoneName); loadErr == nil {
			location = loaded
		}
	}
	if _, err := time.ParseInLocation(photographyDateLayout, dateValue, location); err != nil {
		return newPhotographyValidationError("日期格式必须为 YYYY-MM-DD")
	}

	localNow := now.In(location)
	localToday := localNow.Format(photographyDateLayout)
	maxDate := localNow.AddDate(0, 0, forecastDays-1).Format(photographyDateLayout)
	if dateValue < localToday || dateValue > maxDate {
		return newPhotographyValidationError(fmt.Sprintf("日期仅支持所选地点今天起 %d 天内", forecastDays))
	}
	return nil
}

func ScorePhotographyConditions(cloudCover, precipitationProbability *float64, weatherCode *int) PhotographyAssessment {
	cloudScore := 10
	if cloudCover != nil {
		switch {
		case *cloudCover >= 20 && *cloudCover <= 80:
			cloudScore = 40
		case *cloudCover >= 10 && *cloudCover <= 90:
			cloudScore = 25
		}
	}

	rainScore := 25
	if precipitationProbability != nil {
		switch {
		case *precipitationProbability < 20:
			rainScore = 40
		case *precipitationProbability < 40:
			rainScore = 25
		case *precipitationProbability <= 60:
			rainScore = 10
		default:
			rainScore = 0
		}
	}

	weatherScore := 10
	if weatherCode != nil {
		switch {
		case isClearWeatherCode(*weatherCode):
			weatherScore = 20
		case isWetWeatherCode(*weatherCode):
			weatherScore = 0
		}
	}

	score := cloudScore + rainScore + weatherScore
	assessment := PhotographyAssessment{Score: score, Level: photographyScoreLevel(score)}
	assessment.Reasons = append(assessment.Reasons, photographyCloudReason(cloudCover))
	assessment.Reasons = append(assessment.Reasons, photographyRainReason(precipitationProbability))
	assessment.Reasons = append(assessment.Reasons, photographyWeatherReason(weatherCode))
	return assessment
}

func photographyScoreLevel(score int) string {
	switch {
	case score >= 75:
		return "excellent"
	case score >= 55:
		return "good"
	case score >= 35:
		return "fair"
	default:
		return "poor"
	}
}

func buildPhotographyForecastDays(response openMeteoForecastResponse) []PhotographyForecastDay {
	count := len(response.Daily.Time)
	days := make([]PhotographyForecastDay, 0, count)
	for i := 0; i < count; i++ {
		date := response.Daily.Time[i]
		sunrise := stringAt(response.Daily.Sunrise, i)
		sunset := stringAt(response.Daily.Sunset, i)
		cloudCover := averageHourlyValueForDate(response.Hourly.Time, response.Hourly.CloudCover, date)
		precipitationProbability := valuePtr(response.Daily.PrecipitationProbabilityMax, i)
		weatherCode := intPtr(response.Daily.WeatherCode, i)
		temperatureMax := valuePtr(response.Daily.TemperatureMax, i)
		temperatureMin := valuePtr(response.Daily.TemperatureMin, i)
		precipitation := valuePtr(response.Daily.PrecipitationSum, i)
		relativeHumidity := averageHourlyValueForDate(response.Hourly.Time, response.Hourly.RelativeHumidity, date)
		windSpeed := averageHourlyValueForDate(response.Hourly.Time, response.Hourly.WindSpeed, date)
		windDirection := averageHourlyValueForDate(response.Hourly.Time, response.Hourly.WindDirection, date)
		visibility := averageHourlyValueForDate(response.Hourly.Time, response.Hourly.Visibility, date)

		sunriseAssessment := buildPhotographyAssessmentForEvent(response, date, sunrise, cloudCover, precipitationProbability, weatherCode)
		sunsetAssessment := buildPhotographyAssessmentForEvent(response, date, sunset, cloudCover, precipitationProbability, weatherCode)
		days = append(days, PhotographyForecastDay{
			Date:                     date,
			Sunrise:                  sunrise,
			Sunset:                   sunset,
			SunriseScore:             sunriseAssessment.Score,
			SunriseLevel:             sunriseAssessment.Level,
			SunriseReasons:           sunriseAssessment.Reasons,
			SunsetScore:              sunsetAssessment.Score,
			SunsetLevel:              sunsetAssessment.Level,
			SunsetReasons:            sunsetAssessment.Reasons,
			CloudCover:               cloudCover,
			PrecipitationProbability: precipitationProbability,
			WeatherCode:              weatherCode,
			WeatherText:              photographyForecastWeatherText(weatherCode),
			TemperatureMax:           temperatureMax,
			TemperatureMin:           temperatureMin,
			Precipitation:            precipitation,
			RelativeHumidity:         relativeHumidity,
			WindSpeed:                windSpeed,
			WindDirection:            windDirection,
			Visibility:               visibility,
			Hours:                    buildPhotographyForecastHours(response, date),
		})
	}
	return days
}

func buildPhotographyForecastHours(response openMeteoForecastResponse, date string) []PhotographyForecastHour {
	hours := make([]PhotographyForecastHour, 0, 24)
	for i, value := range response.Hourly.Time {
		if !strings.HasPrefix(value, date) {
			continue
		}
		weatherCode := intPtr(response.Hourly.WeatherCode, i)
		hours = append(hours, PhotographyForecastHour{
			Time:        value,
			Temperature: valuePtr(response.Hourly.Temperature, i),
			WeatherCode: weatherCode,
			WeatherText: photographyForecastWeatherText(weatherCode),
		})
	}
	return hours
}

func photographyForecastWeatherText(code *int) string {
	if code == nil {
		return "暂无数据"
	}
	switch {
	case *code == 0:
		return "晴朗"
	case *code == 1 || *code == 2:
		return "少云"
	case *code == 3:
		return "阴天"
	case *code == 45 || *code == 48:
		return "雾"
	case *code >= 51 && *code <= 67:
		return "降雨"
	case *code >= 71 && *code <= 77:
		return "降雪"
	case *code >= 80 && *code <= 86:
		return "阵雨或阵雪"
	case *code >= 95:
		return "雷暴"
	default:
		return "多云"
	}
}

func buildPhotographyAssessmentForEvent(response openMeteoForecastResponse, date, event string, fallbackCloud, fallbackProbability *float64, fallbackWeatherCode *int) PhotographyAssessment {
	indices := photographyWindowIndices(response.Hourly.Time, date, event)
	cloud := averageIndexedValue(response.Hourly.CloudCover, indices)
	probability := maxIndexedValue(response.Hourly.PrecipitationProbability, indices)
	weatherCode := worstIndexedWeatherCode(response.Hourly.WeatherCode, indices)
	if cloud == nil {
		cloud = fallbackCloud
	}
	if probability == nil {
		probability = fallbackProbability
	}
	if weatherCode == nil {
		weatherCode = fallbackWeatherCode
	}
	return ScorePhotographyConditions(cloud, probability, weatherCode)
}

func photographyWindowIndices(times []string, date, event string) []int {
	if event == "" {
		return nil
	}
	eventTime, ok := parsePhotographyLocalTime(event)
	if !ok {
		return nil
	}
	indices := make([]int, 0, 3)
	nearestIndex := -1
	nearestDistance := time.Duration(math.MaxInt64)
	for i, value := range times {
		if !strings.HasPrefix(value, date) {
			continue
		}
		candidate, candidateOK := parsePhotographyLocalTime(value)
		if !candidateOK {
			continue
		}
		distance := candidate.Sub(eventTime)
		if distance < 0 {
			distance = -distance
		}
		if distance <= time.Hour {
			indices = append(indices, i)
		}
		if distance < nearestDistance {
			nearestIndex = i
			nearestDistance = distance
		}
	}
	if len(indices) == 0 && nearestIndex >= 0 {
		indices = append(indices, nearestIndex)
	}
	return indices
}

func parsePhotographyLocalTime(value string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02T15:04", time.RFC3339, "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func averageHourlyValueForDate(times []string, values []float64, date string) *float64 {
	indices := make([]int, 0)
	for i, value := range times {
		if strings.HasPrefix(value, date) && i < len(values) {
			indices = append(indices, i)
		}
	}
	return averageIndexedValue(values, indices)
}

func averageIndexedValue(values []float64, indices []int) *float64 {
	if len(indices) == 0 {
		return nil
	}
	var total float64
	count := 0
	for _, index := range indices {
		if index >= 0 && index < len(values) {
			total += values[index]
			count++
		}
	}
	if count == 0 {
		return nil
	}
	value := total / float64(count)
	return &value
}

func maxIndexedValue(values []float64, indices []int) *float64 {
	var result *float64
	for _, index := range indices {
		if index < 0 || index >= len(values) {
			continue
		}
		value := values[index]
		if result == nil || value > *result {
			copied := value
			result = &copied
		}
	}
	return result
}

func worstIndexedWeatherCode(values []int, indices []int) *int {
	var result *int
	for _, index := range indices {
		if index < 0 || index >= len(values) {
			continue
		}
		value := values[index]
		if result == nil || weatherCodeSeverity(value) > weatherCodeSeverity(*result) {
			copied := value
			result = &copied
		}
	}
	return result
}

func weatherCodeSeverity(code int) int {
	switch {
	case code >= 95:
		return 4
	case code >= 51 && code <= 86:
		return 3
	case code == 45 || code == 48 || code == 3:
		return 2
	default:
		return 1
	}
}

func isClearWeatherCode(code int) bool {
	return code == 0 || code == 1 || code == 2
}

func isWetWeatherCode(code int) bool {
	return (code >= 51 && code <= 86) || code >= 95
}

func photographyCloudReason(value *float64) string {
	if value == nil {
		return "云量数据不足，按中性条件估算"
	}
	switch {
	case *value >= 20 && *value <= 80:
		return fmt.Sprintf("云量 %.0f%%，层次条件较好", *value)
	case *value < 20:
		return fmt.Sprintf("云量 %.0f%%，天空可能偏空", *value)
	default:
		return fmt.Sprintf("云量 %.0f%%，云层可能偏厚", *value)
	}
}

func photographyRainReason(value *float64) string {
	if value == nil {
		return "降水概率数据不足，建议关注临近天气"
	}
	switch {
	case *value < 20:
		return fmt.Sprintf("降水概率 %.0f%%，降雨风险较低", *value)
	case *value <= 60:
		return fmt.Sprintf("降水概率 %.0f%%，建议准备防雨装备", *value)
	default:
		return fmt.Sprintf("降水概率 %.0f%%，不建议作为首选拍摄时段", *value)
	}
}

func photographyWeatherReason(value *int) string {
	if value == nil {
		return "天气状况数据不足，评分仅供参考"
	}
	if isClearWeatherCode(*value) {
		return "天气以晴朗或少云为主"
	}
	if isWetWeatherCode(*value) {
		return "存在降雨、降雪或雷暴天气信号"
	}
	return "存在阴天或雾天气信号"
}

func valuePtr(values []float64, index int) *float64 {
	if index < 0 || index >= len(values) {
		return nil
	}
	value := values[index]
	return &value
}

func intPtr(values []int, index int) *int {
	if index < 0 || index >= len(values) {
		return nil
	}
	value := values[index]
	return &value
}

func stringAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

func containsPhotographyDate(days []PhotographyForecastDay, target string) bool {
	for _, day := range days {
		if day.Date == target {
			return true
		}
	}
	return false
}

func buildLocationDisplayName(name, admin1, country string) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{name, admin1, country} {
		part = strings.TrimSpace(part)
		if part == "" || containsPhotographyString(parts, part) {
			continue
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " · ")
}

func containsPhotographyString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func fetchPhotographyJSON(endpoint string, target interface{}) error {
	return fetchPhotographyJSONWithHeaders("open-meteo", endpoint, target, nil)
}

func fetchPhotographyJSONWithHeaders(source, endpoint string, target interface{}, headers http.Header) error {
	logEndpoint := photographyLogEndpoint(endpoint)
	var lastErr error
	for attempt := 0; attempt < photographyHTTPAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			LogBackendError(source, http.MethodGet, logEndpoint, http.StatusBadGateway, err.Error())
			return err
		}
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "ai-multi-platform-release/1.0")
		}
		response, err := photographyHTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("天气服务请求失败: %w", err)
			if attempt+1 < photographyHTTPAttempts && retryablePhotographyHTTPError(err) {
				time.Sleep(photographyRetryDelay(attempt))
				continue
			}
			LogBackendError(source, http.MethodGet, logEndpoint, http.StatusBadGateway, lastErr.Error())
			return lastErr
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 8<<20))
		response.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("天气服务响应读取失败: %w", readErr)
			if attempt+1 < photographyHTTPAttempts && retryablePhotographyHTTPError(readErr) {
				time.Sleep(photographyRetryDelay(attempt))
				continue
			}
			LogBackendError(source, http.MethodGet, logEndpoint, http.StatusBadGateway, lastErr.Error())
			return lastErr
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			message := strings.TrimSpace(string(body))
			if message == "" {
				message = response.Status
			}
			lastErr = fmt.Errorf("天气服务返回异常: %s", message)
			if attempt+1 < photographyHTTPAttempts && retryablePhotographyHTTPStatus(response.StatusCode) {
				time.Sleep(photographyRetryDelay(attempt))
				continue
			}
			LogBackendError(source, http.MethodGet, logEndpoint, response.StatusCode, lastErr.Error())
			return lastErr
		}
		if err := json.Unmarshal(body, target); err != nil {
			lastErr = fmt.Errorf("天气服务响应解析失败: %w", err)
			LogBackendError(source, http.MethodGet, logEndpoint, http.StatusBadGateway, lastErr.Error())
			return lastErr
		}
		return nil
	}
	return lastErr
}

func retryablePhotographyHTTPError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"eof", "timeout", "connection reset", "broken pipe", "temporary"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func retryablePhotographyHTTPStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func photographyRetryDelay(attempt int) time.Duration {
	return time.Duration(250*(1<<attempt)) * time.Millisecond
}

func summarizePhotographyError(err error) string {
	if err == nil {
		return "未知错误"
	}
	message := strings.TrimSpace(err.Error())
	if len([]rune(message)) > 160 {
		return string([]rune(message)[:160]) + "…"
	}
	return message
}

func photographyLogEndpoint(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	parsed.RawQuery = ""
	return parsed.String()
}

func photographyWeatherSourceModel(source string) string {
	for _, item := range ListPhotographyWeatherSources() {
		if item.ID == source {
			return item.Model
		}
	}
	return ""
}

func appendOpenMeteoAPIKey(params url.Values) {
	if key := photographyWeatherProviderConfigFor(photographyWeatherConfigOpenMeteo).APIKey; key != "" {
		params.Set("apikey", key)
	}
}

func photographyForecastBaseURL() string {
	if value := strings.TrimSpace(photographyIntegrationSecretsForUse().OpenMeteoHost); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("OPEN_METEO_FORECAST_BASE_URL")); value != "" {
		return value
	}
	return "https://api.open-meteo.com/v1/forecast"
}

func photographyGeocodingBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("OPEN_METEO_GEOCODING_BASE_URL")); value != "" {
		return value
	}
	return "https://geocoding-api.open-meteo.com/v1/search"
}

type photographyValidationError struct {
	message string
}

func (e *photographyValidationError) Error() string {
	return e.message
}

func newPhotographyValidationError(message string) error {
	return &photographyValidationError{message: message}
}

func IsPhotographyValidationError(err error) bool {
	var validationErr *photographyValidationError
	return errors.As(err, &validationErr)
}

func SnapshotFromPhotographyForecast(forecast *PhotographyForecast) string {
	if forecast == nil {
		return ""
	}
	encoded, err := json.Marshal(PhotographyForecastSnapshot{Days: forecast.Days})
	if err != nil {
		return ""
	}
	return string(encoded)
}

func DecodePhotographyForecastSnapshot(raw string) PhotographyForecastSnapshot {
	var snapshot PhotographyForecastSnapshot
	if strings.TrimSpace(raw) == "" {
		return snapshot
	}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return PhotographyForecastSnapshot{}
	}
	return snapshot
}
