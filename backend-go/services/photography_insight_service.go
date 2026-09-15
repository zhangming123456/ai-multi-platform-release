package services

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type PhotographyTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
type PhotographyPhenomenonEstimate struct {
	Available   bool     `json:"available"`
	Score       *int     `json:"score"`
	Probability *int     `json:"probability"`
	Level       string   `json:"level"`
	Reasons     []string `json:"reasons"`
	Risks       []string `json:"risks"`
	Confidence  string   `json:"confidence"`
}
type PhotographyWeatherToday struct {
	Available                bool     `json:"available"`
	Temperature              *float64 `json:"temperature"`
	FeelsLike                *float64 `json:"feels_like"`
	TemperatureMax           *float64 `json:"temperature_max"`
	TemperatureMin           *float64 `json:"temperature_min"`
	RelativeHumidity         *float64 `json:"relative_humidity"`
	WindSpeed                *float64 `json:"wind_speed"`
	WindDirection            *float64 `json:"wind_direction"`
	PrecipitationProbability *float64 `json:"precipitation_probability"`
	Precipitation            *float64 `json:"precipitation"`
	Visibility               *float64 `json:"visibility"`
	CloudCover               *float64 `json:"cloud_cover"`
	WeatherCode              *int     `json:"weather_code"`
	WeatherDescription       string   `json:"weather_description"`
	Risk                     string   `json:"risk"`
}
type PhotographyTideEvent struct {
	Time   string  `json:"time"`
	Type   string  `json:"type"`
	Height float64 `json:"height"`
}
type PhotographyTideSummary struct {
	Available    bool                   `json:"available"`
	Source       string                 `json:"source"`
	Station      string                 `json:"station"`
	DistanceKm   *float64               `json:"distance_km"`
	CurrentLevel *float64               `json:"current_level"`
	NextHigh     *PhotographyTideEvent  `json:"next_high"`
	NextLow      *PhotographyTideEvent  `json:"next_low"`
	Events       []PhotographyTideEvent `json:"events"`
	Curve        []PhotographyTideEvent `json:"curve"`
	Message      string                 `json:"message"`
}
type PhotographyAuroraSummary struct {
	Available        bool     `json:"available"`
	Kp               *float64 `json:"kp"`
	MagneticLatitude *float64 `json:"magnetic_latitude"`
	VisibleStart     string   `json:"visible_start"`
	VisibleEnd       string   `json:"visible_end"`
	Score            *int     `json:"score"`
	Level            string   `json:"level"`
	Reasons          []string `json:"reasons"`
	Risks            []string `json:"risks"`
	Confidence       string   `json:"confidence"`
	Message          string   `json:"message"`
}
type PhotographyPhotographyDay struct {
	Date        string `json:"date"`
	Sunrise     string `json:"sunrise"`
	Sunset      string `json:"sunset"`
	GoldenHours struct {
		Morning *PhotographyTimeRange `json:"morning"`
		Evening *PhotographyTimeRange `json:"evening"`
	} `json:"golden_hours"`
	BlueHours struct {
		Morning *PhotographyTimeRange `json:"morning"`
		Evening *PhotographyTimeRange `json:"evening"`
	} `json:"blue_hours"`
	SunriseAssessment        PhotographyPhenomenonEstimate `json:"sunrise_assessment"`
	SunsetAssessment         PhotographyPhenomenonEstimate `json:"sunset_assessment"`
	CloudSea                 PhotographyPhenomenonEstimate `json:"cloud_sea"`
	StargazingIndex          int                           `json:"stargazing_index"`
	CloudCoverQuality        *int                          `json:"cloud_cover_quality"`
	CloudCover               *float64                      `json:"cloud_cover"`
	PrecipitationProbability *float64                      `json:"precipitation_probability"`
	WeatherCode              *int                          `json:"weather_code"`
	Confidence               string                        `json:"confidence"`
}
type PhotographyChartSeries struct {
	Name   string `json:"name"`
	Values []int  `json:"values"`
	Color  string `json:"color"`
}
type PhotographyCharts struct {
	Probability  []PhotographyChartSeries `json:"probability"`
	CloudQuality []PhotographyChartSeries `json:"cloud_quality"`
}
type PhotographyOverview struct {
	Location struct {
		Name        string   `json:"name"`
		Latitude    float64  `json:"latitude"`
		Longitude   float64  `json:"longitude"`
		Timezone    string   `json:"timezone"`
		CountryCode string   `json:"country_code"`
		Elevation   *float64 `json:"elevation"`
		MapProvider string   `json:"map_provider"`
	} `json:"location"`
	Today struct {
		Weather   PhotographyWeatherToday       `json:"weather"`
		Hours     []PhotographyForecastHour     `json:"hours"`
		Rainbow   PhotographyPhenomenonEstimate `json:"rainbow"`
		FrostRime PhotographyPhenomenonEstimate `json:"frost_rime"`
		Tide      PhotographyTideSummary        `json:"tide"`
		Aurora    PhotographyAuroraSummary      `json:"aurora"`
	} `json:"today"`
	Days    []PhotographyPhotographyDay `json:"days"`
	Charts  PhotographyCharts           `json:"charts"`
	Sources struct {
		Weather string `json:"weather"`
		Tide    string `json:"tide"`
		Aurora  string `json:"aurora"`
	} `json:"sources"`
	Warnings []string `json:"warnings"`
}
type photographyWindowMetrics struct {
	Cloud, Probability, Humidity, DewPoint, Visibility, WindSpeed, Temperature, Precipitation *float64
	WeatherCode                                                                               *int
}

func FetchPhotographyOverview(latitude, longitude float64, baseDate, weatherSource, tideSource, locationName, countryCode string) (*PhotographyOverview, error) {
	if err := ValidatePhotographyCoordinates(latitude, longitude); err != nil {
		return nil, err
	}
	weatherSource = NormalizePhotographyWeatherSource(weatherSource)
	if err := ValidatePhotographyWeatherSource(weatherSource); err != nil {
		return nil, err
	}
	raw, forecast, err := fetchPhotographyInsightWeather(latitude, longitude, baseDate, weatherSource)
	if err != nil {
		return nil, err
	}
	timezone := "UTC"
	if raw != nil && raw.Timezone != "" {
		timezone = raw.Timezone
	}
	if forecast != nil && forecast.Timezone != "" {
		timezone = forecast.Timezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
		timezone = "UTC"
	}
	if strings.TrimSpace(baseDate) == "" {
		baseDate = time.Now().In(location).Format(photographyDateLayout)
	}
	effectiveWeatherSource := weatherSource
	if forecast != nil && forecast.Source != "" {
		effectiveWeatherSource = forecast.Source
	}
	maxDays := PhotographyForecastDays
	if effectiveWeatherSource == PhotographyWeatherSourceQWeather {
		maxDays = photographyQWeatherDays
	}
	if err := ValidatePhotographyDateWithDays(baseDate, timezone, time.Now(), maxDays); err != nil {
		return nil, err
	}
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if countryCode == "" {
		countryCode = inferPhotographyCountryCode(latitude, longitude)
	}
	if strings.TrimSpace(locationName) == "" {
		locationName = fmt.Sprintf("自定义坐标（%.4f, %.4f）", latitude, longitude)
	}
	o := &PhotographyOverview{}
	o.Location.Name, o.Location.Latitude, o.Location.Longitude, o.Location.Timezone, o.Location.CountryCode, o.Location.MapProvider = locationName, latitude, longitude, timezone, countryCode, SelectPhotographyMapProvider(countryCode)
	if raw != nil {
		o.Location.Elevation = raw.Elevation
	}
	o.Sources.Weather, o.Sources.Tide, o.Sources.Aurora = PhotographyWeatherSourceName(effectiveWeatherSource), normalizeTideSource(tideSource), "NOAA SWPC"
	if forecast != nil && forecast.FallbackReason != "" {
		o.Warnings = append(o.Warnings, forecast.FallbackReason)
	}
	if raw != nil {
		o.Days = buildPhotographyInsightDays(raw, baseDate)
		o.Today.Weather = buildPhotographyWeatherToday(raw, baseDate)
		o.Today.Hours = buildPhotographyForecastHours(*raw, baseDate)
		o.Today.Rainbow = estimatePhotographyRainbow(raw, baseDate)
		o.Today.FrostRime = estimatePhotographyFrostRime(raw, baseDate, raw.Elevation)
	} else {
		o.Days = buildPhotographyInsightDaysFromForecast(forecast, baseDate)
		o.Today.Weather = buildPhotographyWeatherTodayFromForecast(forecast, baseDate)
		o.Today.Hours = photographyForecastHoursForDate(forecast, baseDate)
		o.Today.Rainbow = unavailablePhotographyEstimate("当前天气源未提供逐小时降水、太阳高度和能见度数据")
		o.Today.FrostRime = unavailablePhotographyEstimate("当前天气源未提供温度、露点和湿度数据")
	}
	if len(o.Days) < 3 {
		o.Warnings = append(o.Warnings, "天气服务未返回完整的今天、明天、后天数据")
	}
	o.Charts = buildPhotographyCharts(o.Days)
	if tideSource != "none" && tideSource != "disabled" {
		o.Today.Tide, err = FetchPhotographyTideSummary(latitude, longitude, baseDate, tideSource, timezone)
		if err != nil {
			o.Warnings = append(o.Warnings, "潮汐数据暂不可用："+err.Error())
			o.Today.Tide = unavailableTide(normalizeTideSource(tideSource), err.Error())
		}
	} else {
		o.Today.Tide = unavailableTide("none", "未选择潮汐数据源")
	}
	o.Today.Aurora, err = FetchPhotographyAuroraSummary(latitude, timezone, baseDate, firstDayCloud(o.Days), o.Days)
	if err != nil {
		o.Warnings = append(o.Warnings, "极光数据暂不可用："+err.Error())
	}
	normalizePhotographyOverviewArrays(o)
	return o, nil
}

func normalizePhotographyOverviewArrays(overview *PhotographyOverview) {
	if overview == nil {
		return
	}
	if overview.Days == nil {
		overview.Days = []PhotographyPhotographyDay{}
	}
	if overview.Today.Hours == nil {
		overview.Today.Hours = []PhotographyForecastHour{}
	}
	if overview.Charts.Probability == nil {
		overview.Charts.Probability = []PhotographyChartSeries{}
	}
	if overview.Charts.CloudQuality == nil {
		overview.Charts.CloudQuality = []PhotographyChartSeries{}
	}
	if overview.Warnings == nil {
		overview.Warnings = []string{}
	}
	normalizePhotographyTideArrays(&overview.Today.Tide)
	normalizePhotographyEstimateArrays(&overview.Today.Rainbow)
	normalizePhotographyEstimateArrays(&overview.Today.FrostRime)
	if overview.Today.Aurora.Reasons == nil {
		overview.Today.Aurora.Reasons = []string{}
	}
	if overview.Today.Aurora.Risks == nil {
		overview.Today.Aurora.Risks = []string{}
	}
	for index := range overview.Days {
		normalizePhotographyEstimateArrays(&overview.Days[index].SunriseAssessment)
		normalizePhotographyEstimateArrays(&overview.Days[index].SunsetAssessment)
		normalizePhotographyEstimateArrays(&overview.Days[index].CloudSea)
	}
}

func normalizePhotographyTideArrays(tide *PhotographyTideSummary) {
	if tide == nil {
		return
	}
	if tide.Events == nil {
		tide.Events = []PhotographyTideEvent{}
	}
	if tide.Curve == nil {
		tide.Curve = []PhotographyTideEvent{}
	}
}

func normalizePhotographyEstimateArrays(estimate *PhotographyPhenomenonEstimate) {
	if estimate == nil {
		return
	}
	if estimate.Reasons == nil {
		estimate.Reasons = []string{}
	}
	if estimate.Risks == nil {
		estimate.Risks = []string{}
	}
}

func fetchPhotographyInsightWeather(latitude, longitude float64, baseDate, source string) (*openMeteoForecastResponse, *PhotographyForecast, error) {
	if source == PhotographyWeatherSourceQWeather {
		if strings.TrimSpace(baseDate) == "" {
			baseDate = time.Now().UTC().Format(photographyDateLayout)
		}
		forecast, err := FetchPhotographyForecastWithSource(latitude, longitude, baseDate, source)
		return nil, forecast, err
	}
	openMeteoConfig := photographyWeatherProviderConfigFor(photographyWeatherConfigOpenMeteo)
	var openMeteoErr error
	if openMeteoConfig.Enabled {
		raw, forecast, err := fetchOpenMeteoPhotographyData(latitude, longitude, baseDate, source)
		if err == nil {
			return raw, forecast, nil
		}
		openMeteoErr = err
	} else {
		openMeteoErr = errors.New("Open-Meteo 已停用")
	}
	if source == PhotographyWeatherSourceBestMatch && qweatherConfigured() {
		if strings.TrimSpace(baseDate) == "" {
			baseDate = time.Now().UTC().Format(photographyDateLayout)
		}
		forecast, err := fetchQWeatherPhotographyForecast(latitude, longitude, baseDate)
		if err == nil {
			forecast.FallbackReason = fmt.Sprintf("Open-Meteo 不可用，已切换至和风天气：%s", summarizePhotographyError(openMeteoErr))
			return nil, forecast, nil
		}
		return nil, nil, fmt.Errorf("自动匹配天气服务失败：Open-Meteo：%v；和风天气：%v", openMeteoErr, err)
	}
	return nil, nil, fmt.Errorf("Open-Meteo 天气服务不可用：%w；如需自动切换，请配置和风天气 API Key 或 JWT Token", openMeteoErr)
}

func inferPhotographyCountryCode(latitude, longitude float64) string {
	if latitude >= 3.5 && latitude <= 53.8 && longitude >= 73.0 && longitude <= 135.2 {
		return "CN"
	}
	return ""
}

func buildPhotographyInsightDays(response *openMeteoForecastResponse, baseDate string) []PhotographyPhotographyDay {
	result := make([]PhotographyPhotographyDay, 0, 3)
	for offset := 0; offset < 3; offset++ {
		date := addPhotographyDate(baseDate, offset)
		i := indexOfString(response.Daily.Time, date)
		if i < 0 {
			continue
		}
		sunrise, sunset := stringAt(response.Daily.Sunrise, i), stringAt(response.Daily.Sunset, i)
		morning, evening := eventPhotographyMetrics(response, date, sunrise), eventPhotographyMetrics(response, date, sunset)
		cloud := averageHourlyValueForDate(response.Hourly.Time, response.Hourly.CloudCover, date)
		if cloud == nil {
			cloud = morning.Cloud
			if cloud == nil {
				cloud = evening.Cloud
			}
		}
		probability, code := valuePtr(response.Daily.PrecipitationProbabilityMax, i), intPtr(response.Daily.WeatherCode, i)
		d := PhotographyPhotographyDay{Date: date, Sunrise: sunrise, Sunset: sunset, CloudCover: cloud, PrecipitationProbability: probability, WeatherCode: code, Confidence: photographyMetricsConfidence(morning, evening)}
		d.SunriseAssessment = buildPhenomenonFromAssessment(ScorePhotographyConditions(morning.Cloud, morning.Probability, morning.WeatherCode), morning, "朝霞")
		d.SunsetAssessment = buildPhenomenonFromAssessment(ScorePhotographyConditions(evening.Cloud, evening.Probability, evening.WeatherCode), evening, "晚霞")
		d.CloudSea = estimatePhotographyCloudSea(response, date, response.Elevation)
		d.CloudCoverQuality = cloudQualityPointer(cloud)
		d.StargazingIndex = calculatePhotographyStargazing(response, date, sunset)
		d.GoldenHours.Morning, d.GoldenHours.Evening = approximateGoldenHour(sunrise, true), approximateGoldenHour(sunset, false)
		d.BlueHours.Morning, d.BlueHours.Evening = approximateBlueHour(sunrise, true), approximateBlueHour(sunset, false)
		result = append(result, d)
	}
	return result
}
func buildPhotographyInsightDaysFromForecast(forecast *PhotographyForecast, baseDate string) []PhotographyPhotographyDay {
	if forecast == nil {
		return nil
	}
	result := make([]PhotographyPhotographyDay, 0, 3)
	for offset := 0; offset < 3; offset++ {
		date := addPhotographyDate(baseDate, offset)
		var source *PhotographyForecastDay
		for i := range forecast.Days {
			if forecast.Days[i].Date == date {
				source = &forecast.Days[i]
				break
			}
		}
		if source == nil {
			continue
		}
		d := PhotographyPhotographyDay{Date: source.Date, Sunrise: source.Sunrise, Sunset: source.Sunset, CloudCover: source.CloudCover, PrecipitationProbability: source.PrecipitationProbability, WeatherCode: source.WeatherCode, Confidence: "low"}
		d.SunriseAssessment = phenomenonFromLegacy(source.SunriseScore, source.SunriseLevel, source.SunriseReasons, "朝霞")
		d.SunsetAssessment = phenomenonFromLegacy(source.SunsetScore, source.SunsetLevel, source.SunsetReasons, "晚霞")
		d.CloudSea = unavailablePhotographyEstimate("当前数据源缺少湿度、露点和地形数据")
		d.CloudCoverQuality = cloudQualityPointer(source.CloudCover)
		d.GoldenHours.Morning, d.GoldenHours.Evening = approximateGoldenHour(source.Sunrise, true), approximateGoldenHour(source.Sunset, false)
		d.BlueHours.Morning, d.BlueHours.Evening = approximateBlueHour(source.Sunrise, true), approximateBlueHour(source.Sunset, false)
		result = append(result, d)
	}
	return result
}

func photographyForecastHoursForDate(forecast *PhotographyForecast, date string) []PhotographyForecastHour {
	if forecast == nil {
		return []PhotographyForecastHour{}
	}
	for _, day := range forecast.Days {
		if day.Date == date {
			return day.Hours
		}
	}
	return []PhotographyForecastHour{}
}

func eventPhotographyMetrics(response *openMeteoForecastResponse, date, event string) photographyWindowMetrics {
	indices := photographyWindowIndices(response.Hourly.Time, date, event)
	return photographyWindowMetrics{Cloud: averageIndexedValue(response.Hourly.CloudCover, indices), Probability: maxIndexedValue(response.Hourly.PrecipitationProbability, indices), Humidity: averageIndexedValue(response.Hourly.RelativeHumidity, indices), DewPoint: averageIndexedValue(response.Hourly.DewPoint, indices), Visibility: averageIndexedValue(response.Hourly.Visibility, indices), WindSpeed: averageIndexedValue(response.Hourly.WindSpeed, indices), Temperature: averageIndexedValue(response.Hourly.Temperature, indices), Precipitation: averageIndexedValue(response.Hourly.Precipitation, indices), WeatherCode: worstIndexedWeatherCode(response.Hourly.WeatherCode, indices)}
}
func buildPhenomenonFromAssessment(a PhotographyAssessment, m photographyWindowMetrics, name string) PhotographyPhenomenonEstimate {
	score := a.Score
	risks := []string{}
	if m.Probability != nil && *m.Probability >= 60 {
		risks = append(risks, "降水概率较高")
	}
	if m.WeatherCode != nil && isWetWeatherCode(*m.WeatherCode) {
		risks = append(risks, "存在降雨、降雪或雷暴天气")
	}
	return PhotographyPhenomenonEstimate{Available: true, Score: &score, Probability: &score, Level: a.Level, Reasons: append(append([]string{}, a.Reasons...), name+"为天气条件估算"), Risks: risks, Confidence: metricsConfidence(m)}
}
func phenomenonFromLegacy(score int, level string, reasons []string, name string) PhotographyPhenomenonEstimate {
	return PhotographyPhenomenonEstimate{Available: true, Score: &score, Probability: &score, Level: level, Reasons: append(reasons, name+"为天气条件估算"), Confidence: "low"}
}
func unavailablePhotographyEstimate(reason string) PhotographyPhenomenonEstimate {
	return PhotographyPhenomenonEstimate{Available: false, Level: "unavailable", Reasons: []string{reason}, Risks: []string{"数据源未提供必要指标"}, Confidence: "low"}
}
func metricsConfidence(m photographyWindowMetrics) string {
	if m.Cloud != nil && m.Probability != nil && m.WeatherCode != nil {
		return "high"
	}
	if m.Cloud != nil || m.Probability != nil {
		return "medium"
	}
	return "low"
}
func photographyMetricsConfidence(a, b photographyWindowMetrics) string {
	if metricsConfidence(a) == "high" && metricsConfidence(b) == "high" {
		return "high"
	}
	if metricsConfidence(a) != "low" || metricsConfidence(b) != "low" {
		return "medium"
	}
	return "low"
}

func estimatePhotographyCloudSea(response *openMeteoForecastResponse, date string, elevation *float64) PhotographyPhenomenonEstimate {
	indices := []int{}
	for i, v := range response.Hourly.Time {
		if strings.HasPrefix(v, date) && len(v) >= 13 {
			h, _ := strconv.Atoi(v[11:13])
			if h >= 3 && h <= 9 {
				indices = append(indices, i)
			}
		}
	}
	m := photographyWindowMetrics{Cloud: averageIndexedValue(response.Hourly.CloudCover, indices), Humidity: averageIndexedValue(response.Hourly.RelativeHumidity, indices), DewPoint: averageIndexedValue(response.Hourly.DewPoint, indices), Visibility: averageIndexedValue(response.Hourly.Visibility, indices), WindSpeed: averageIndexedValue(response.Hourly.WindSpeed, indices), Probability: maxIndexedValue(response.Hourly.PrecipitationProbability, indices), Temperature: averageIndexedValue(response.Hourly.Temperature, indices)}
	if m.Humidity == nil && m.Cloud == nil {
		return unavailablePhotographyEstimate("当前数据源未提供云海所需小时湿度数据")
	}
	score := 0
	reasons := []string{}
	risks := []string{}
	if m.Humidity != nil {
		if *m.Humidity >= 85 {
			score += 25
			reasons = append(reasons, "近地面湿度较高")
		} else if *m.Humidity >= 70 {
			score += 15
			reasons = append(reasons, "湿度一般")
		} else {
			risks = append(risks, "湿度偏低")
		}
	}
	if m.DewPoint != nil && m.Temperature != nil {
		gap := *m.Temperature - *m.DewPoint
		if gap < 0 {
			gap = -gap
		}
		if gap <= 3 {
			score += 25
			reasons = append(reasons, "温度与露点差较小")
		} else if gap <= 6 {
			score += 12
		} else {
			risks = append(risks, "温度与露点差偏大")
		}
	} else {
		risks = append(risks, "缺少温度或露点数据")
	}
	if m.Cloud != nil && *m.Cloud >= 35 && *m.Cloud <= 90 {
		score += 25
		reasons = append(reasons, "低云量处于可形成云海的范围")
	} else {
		risks = append(risks, "低云层条件不足")
	}
	if m.WindSpeed != nil && *m.WindSpeed <= 4 {
		score += 15
		reasons = append(reasons, "风速较低，云层较稳定")
	}
	if m.Probability != nil && *m.Probability >= 60 {
		score -= 15
		risks = append(risks, "降水概率较高")
	}
	if elevation == nil {
		risks = append(risks, "缺少海拔或地形数据，置信度最高为中")
	}
	confidence := metricsConfidence(m)
	if elevation == nil && confidence == "high" {
		confidence = "medium"
	}
	return estimateWithScore(score, reasons, risks, confidence)
}
func estimatePhotographyRainbow(response *openMeteoForecastResponse, date string) PhotographyPhenomenonEstimate {
	indices := []int{}
	for i, v := range response.Hourly.Time {
		if strings.HasPrefix(v, date) && len(v) >= 13 {
			h, _ := strconv.Atoi(v[11:13])
			if (h >= 5 && h <= 9) || (h >= 16 && h <= 20) {
				indices = append(indices, i)
			}
		}
	}
	m := photographyWindowMetrics{Cloud: averageIndexedValue(response.Hourly.CloudCover, indices), Probability: maxIndexedValue(response.Hourly.PrecipitationProbability, indices), Visibility: averageIndexedValue(response.Hourly.Visibility, indices), Precipitation: averageIndexedValue(response.Hourly.Precipitation, indices), WeatherCode: worstIndexedWeatherCode(response.Hourly.WeatherCode, indices)}
	if m.Probability == nil && m.Cloud == nil {
		return unavailablePhotographyEstimate("当前数据源未提供彩虹判断所需的逐小时数据")
	}
	score := 0
	reasons := []string{}
	risks := []string{}
	if m.Probability != nil && *m.Probability >= 20 && *m.Probability <= 70 {
		score += 35
		reasons = append(reasons, "日出或日落窗口存在降水机会")
	} else if m.Probability != nil && *m.Probability > 0 {
		score += 15
	} else {
		risks = append(risks, "缺少降水条件")
	}
	if m.Cloud != nil && *m.Cloud >= 20 && *m.Cloud <= 85 {
		score += 25
		reasons = append(reasons, "云量允许阳光穿透")
	} else {
		risks = append(risks, "云量过少或过厚")
	}
	if m.Visibility != nil && *m.Visibility >= 8000 {
		score += 25
		reasons = append(reasons, "能见度较好")
	} else if m.Visibility != nil && *m.Visibility < 3000 {
		risks = append(risks, "能见度偏低")
	}
	if m.WeatherCode != nil && isWetWeatherCode(*m.WeatherCode) {
		score += 15
		reasons = append(reasons, "存在降水天气信号")
	}
	return estimateWithScore(score, reasons, risks, metricsConfidence(m))
}
func estimatePhotographyFrostRime(response *openMeteoForecastResponse, date string, elevation *float64) PhotographyPhenomenonEstimate {
	day := findPhotographyDay(response, date)
	if day < 0 {
		return unavailablePhotographyEstimate("未找到本地今天天气")
	}
	m := eventPhotographyMetrics(response, date, stringAt(response.Daily.Sunrise, day))
	if m.Temperature == nil {
		return unavailablePhotographyEstimate("缺少雾凇判断所需的近地面温度数据")
	}
	score := 0
	reasons := []string{}
	risks := []string{}
	if *m.Temperature <= 0 {
		score += 35
		reasons = append(reasons, "温度低于冰点")
	} else if *m.Temperature <= 3 {
		score += 18
	} else {
		risks = append(risks, "温度高于雾凇常见范围")
	}
	if m.Humidity != nil && *m.Humidity >= 90 {
		score += 25
		reasons = append(reasons, "相对湿度很高")
	} else if m.Humidity == nil {
		risks = append(risks, "缺少相对湿度")
	} else {
		risks = append(risks, "湿度不足")
	}
	if m.DewPoint != nil {
		gap := *m.Temperature - *m.DewPoint
		if gap < 0 {
			gap = -gap
		}
		if gap <= 2 {
			score += 20
			reasons = append(reasons, "温度与露点接近")
		} else {
			risks = append(risks, "温度与露点差较大")
		}
	} else {
		risks = append(risks, "缺少露点数据")
	}
	if m.WindSpeed != nil && *m.WindSpeed <= 6 {
		score += 10
		reasons = append(reasons, "风速较低")
	}
	if elevation != nil && *elevation >= 500 {
		score += 10
		reasons = append(reasons, "海拔有利于结霜")
	}
	return estimateWithScore(score, reasons, risks, metricsConfidence(m))
}
func estimateWithScore(score int, reasons, risks []string, confidence string) PhotographyPhenomenonEstimate {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return PhotographyPhenomenonEstimate{Available: true, Score: &score, Probability: &score, Level: photographyScoreLevel(score), Reasons: append(reasons, "结果为天气条件估算，不代表自然现象一定发生"), Risks: risks, Confidence: confidence}
}

func buildPhotographyWeatherToday(response *openMeteoForecastResponse, date string) PhotographyWeatherToday {
	result := PhotographyWeatherToday{Available: true}
	day := findPhotographyDay(response, date)
	if day >= 0 {
		result.TemperatureMax = valuePtr(response.Daily.TemperatureMax, day)
		result.TemperatureMin = valuePtr(response.Daily.TemperatureMin, day)
		result.Precipitation = valuePtr(response.Daily.PrecipitationSum, day)
		result.WeatherCode = intPtr(response.Daily.WeatherCode, day)
	}
	if response.Current.Time != "" {
		result.Temperature = &response.Current.Temperature
		result.FeelsLike = &response.Current.ApparentTemperature
		result.RelativeHumidity = &response.Current.RelativeHumidity
		result.PrecipitationProbability = &response.Current.PrecipitationProbability
		result.CloudCover = &response.Current.CloudCover
		result.Visibility = &response.Current.Visibility
		result.WindSpeed = &response.Current.WindSpeed
		result.WindDirection = &response.Current.WindDirection
		code := response.Current.WeatherCode
		result.WeatherCode = &code
	}
	if result.Temperature == nil && day >= 0 {
		m := eventPhotographyMetrics(response, date, stringAt(response.Daily.Sunrise, day))
		result.Temperature = m.Temperature
		result.RelativeHumidity = m.Humidity
		result.Visibility = m.Visibility
		result.WindSpeed = m.WindSpeed
		result.PrecipitationProbability = m.Probability
		result.CloudCover = m.Cloud
		result.WeatherCode = m.WeatherCode
	}
	if result.WeatherCode != nil {
		result.WeatherDescription = photographyWeatherCodeDescription(*result.WeatherCode)
	}
	result.Risk = photographyWeatherRisk(result)
	return result
}
func buildPhotographyWeatherTodayFromForecast(forecast *PhotographyForecast, date string) PhotographyWeatherToday {
	if forecast == nil {
		return PhotographyWeatherToday{Risk: "天气数据暂不可用"}
	}
	for _, day := range forecast.Days {
		if day.Date == date {
			return PhotographyWeatherToday{Available: true, TemperatureMax: day.TemperatureMax, TemperatureMin: day.TemperatureMin, Precipitation: day.Precipitation, RelativeHumidity: day.RelativeHumidity, WindSpeed: day.WindSpeed, WindDirection: day.WindDirection, Visibility: day.Visibility, CloudCover: day.CloudCover, PrecipitationProbability: day.PrecipitationProbability, WeatherCode: day.WeatherCode, WeatherDescription: weatherCodeDescriptionPtr(day.WeatherCode), Risk: "当前数据源缺少实时体感等字段"}
		}
	}
	return PhotographyWeatherToday{Risk: "天气数据暂不可用"}
}
func weatherCodeDescriptionPtr(code *int) string {
	if code == nil {
		return ""
	}
	return photographyWeatherCodeDescription(*code)
}
func photographyWeatherRisk(w PhotographyWeatherToday) string {
	if w.PrecipitationProbability != nil && *w.PrecipitationProbability >= 60 {
		return "降水概率较高，户外拍摄请准备防雨装备"
	}
	if w.Visibility != nil && *w.Visibility < 3000 {
		return "能见度偏低，远景和星空拍摄可能受影响"
	}
	return "暂无明显天气风险"
}
func photographyWeatherCodeDescription(code int) string {
	switch {
	case code == 0:
		return "晴朗"
	case code <= 3:
		return "少云或阴天"
	case code >= 51 && code <= 67:
		return "毛毛雨或降雨"
	case code >= 71 && code <= 77:
		return "降雪"
	case code >= 80 && code <= 82:
		return "阵雨"
	case code >= 95:
		return "雷暴"
	default:
		return "雾或其他天气"
	}
}

func buildPhotographyCharts(days []PhotographyPhotographyDay) PhotographyCharts {
	result := PhotographyCharts{Probability: make([]PhotographyChartSeries, 0, 3), CloudQuality: make([]PhotographyChartSeries, 0, 2)}
	sunrise, sunset, cloudsea, morning, evening := []int{}, []int{}, []int{}, []int{}, []int{}
	for _, d := range days {
		sunrise = append(sunrise, estimateScore(d.SunriseAssessment))
		sunset = append(sunset, estimateScore(d.SunsetAssessment))
		cloudsea = append(cloudsea, estimateScore(d.CloudSea))
		morning = append(morning, estimateCloudQuality(d.CloudCover))
		evening = append(evening, estimateCloudQuality(d.CloudCover))
	}
	result.Probability = []PhotographyChartSeries{{Name: "朝霞", Values: sunrise, Color: "#F59E0B"}, {Name: "晚霞", Values: sunset, Color: "#3B82F6"}, {Name: "云海", Values: cloudsea, Color: "#14B8A6"}}
	result.CloudQuality = []PhotographyChartSeries{{Name: "日出时段云量质量", Values: morning, Color: "#F97316"}, {Name: "日落时段云量质量", Values: evening, Color: "#6366F1"}}
	return result
}
func estimateScore(e PhotographyPhenomenonEstimate) int {
	if e.Score == nil {
		return 0
	}
	return *e.Score
}
func estimateCloudQuality(v *float64) int {
	if v == nil {
		return 0
	}
	return cloudQuality(*v)
}
func firstDayCloud(days []PhotographyPhotographyDay) *float64 {
	if len(days) == 0 {
		return nil
	}
	return days[0].CloudCover
}
func cloudQualityPointer(v *float64) *int {
	if v == nil {
		return nil
	}
	x := cloudQuality(*v)
	return &x
}
func cloudQuality(v float64) int {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	x := int(math.Round(100 - math.Abs(v-50)*1.7))
	if x < 0 {
		x = 0
	}
	return x
}
func approximateGoldenHour(event string, morning bool) *PhotographyTimeRange {
	t, ok := parsePhotographyLocalTime(event)
	if !ok {
		return nil
	}
	_ = morning
	return &PhotographyTimeRange{Start: t.Add(-25 * time.Minute).Format("15:04"), End: t.Add(25 * time.Minute).Format("15:04")}
}
func approximateBlueHour(event string, morning bool) *PhotographyTimeRange {
	t, ok := parsePhotographyLocalTime(event)
	if !ok {
		return nil
	}
	if morning {
		return &PhotographyTimeRange{Start: t.Add(-45 * time.Minute).Format("15:04"), End: t.Add(-25 * time.Minute).Format("15:04")}
	}
	return &PhotographyTimeRange{Start: t.Add(25 * time.Minute).Format("15:04"), End: t.Add(45 * time.Minute).Format("15:04")}
}
func addPhotographyDate(date string, offset int) string {
	t, err := time.ParseInLocation(photographyDateLayout, date, time.UTC)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, offset).Format(photographyDateLayout)
}
func indexOfString(values []string, target string) int {
	for i, v := range values {
		if v == target {
			return i
		}
	}
	return -1
}
func findPhotographyDay(response *openMeteoForecastResponse, date string) int {
	return indexOfString(response.Daily.Time, date)
}
func calculatePhotographyStargazing(response *openMeteoForecastResponse, date, sunset string) int {
	m := eventPhotographyMetrics(response, date, sunset)
	score := 50
	if m.Cloud != nil {
		score += int(math.Round((100 - *m.Cloud) * 0.35))
	}
	if m.Probability != nil {
		score -= int(math.Round(*m.Probability * 0.35))
	}
	if m.Visibility != nil {
		if *m.Visibility >= 10000 {
			score += 15
		} else if *m.Visibility < 3000 {
			score -= 15
		}
	}
	if m.Humidity != nil && *m.Humidity > 90 {
		score -= 8
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func FetchPhotographyAuroraSummary(latitude float64, timezone, date string, cloud *float64, days []PhotographyPhotographyDay) (PhotographyAuroraSummary, error) {
	result := PhotographyAuroraSummary{Confidence: "low"}
	endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("NOAA_SWPC_KP_URL")), "/")
	if endpoint == "" {
		endpoint = "https://services.swpc.noaa.gov/products/noaa-planetary-k-index.json"
	}
	var raw [][]interface{}
	if err := fetchPhotographyJSON(endpoint, &raw); err != nil {
		result.Level = "unavailable"
		result.Message = "NOAA SWPC 数据暂不可用"
		return result, err
	}
	var kp float64
	for i := len(raw) - 1; i >= 1; i-- {
		if len(raw[i]) < 2 {
			continue
		}
		switch v := raw[i][1].(type) {
		case float64:
			kp = v
		case string:
			kp, _ = strconv.ParseFloat(v, 64)
		}
		break
	}
	result.Available = true
	result.Kp = &kp
	mag := math.Abs(latitude) + 5
	if mag > 90 {
		mag = 90
	}
	result.MagneticLatitude = &mag
	score := int(math.Round(kp * 12))
	if score > 60 {
		score = 60
	}
	if mag >= 55 {
		score += 25
	} else if mag >= 45 {
		score += 15
	} else {
		score += 5
	}
	if len(days) == 0 || days[0].Sunset == "" || days[0].Sunrise == "" {
		score -= 35
		result.Risks = append(result.Risks, "当地没有完整的日落/日出事件")
	}
	if cloud != nil && *cloud > 70 {
		score -= 25
		result.Risks = append(result.Risks, "云量较高，可能遮挡极光")
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	result.Score = &score
	result.Level = photographyScoreLevel(score)
	result.Confidence = "medium"
	result.Reasons = []string{fmt.Sprintf("当前 Kp 指数 %.1f", kp), fmt.Sprintf("估算磁纬度 %.1f°", mag), "结合黑夜和云量进行观测建议估算"}
	result.Message = "极光可见性为天气与空间天气条件估算"
	_ = timezone
	_ = date
	return result, nil
}

func normalizeTideSource(source string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	if source == "" {
		return "noaa-coops"
	}
	return source
}
func unavailableTide(source, message string) PhotographyTideSummary {
	return PhotographyTideSummary{
		Available: false,
		Source:    source,
		Events:    []PhotographyTideEvent{},
		Curve:     []PhotographyTideEvent{},
		Message:   message,
	}
}

type photographyNOAAStation struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}
type photographyNOAAStationsResponse struct {
	Stations []photographyNOAAStation `json:"stations"`
}
type photographyNOAAPrediction struct {
	Time  string `json:"t"`
	Value string `json:"v"`
}
type photographyNOAAPredictionsResponse struct {
	Predictions []photographyNOAAPrediction `json:"predictions"`
}

type stormglassExtremesResponse struct {
	Data []struct {
		Time   string  `json:"time"`
		Type   string  `json:"type"`
		Height float64 `json:"height"`
	} `json:"data"`
}

type stormglassSeaLevelValue struct {
	Value  float64 `json:"value"`
	Source string  `json:"source"`
}

type stormglassSeaLevelItem struct {
	Time  string                   `json:"time"`
	SG    *stormglassSeaLevelValue `json:"sg"`
	NOAA  *stormglassSeaLevelValue `json:"noaa"`
	Value *float64                 `json:"value"`
}

type stormglassSeaLevelResponse struct {
	Data []stormglassSeaLevelItem `json:"data"`
}

func FetchPhotographyTideSummary(latitude, longitude float64, date, source, timezone string) (PhotographyTideSummary, error) {
	source = normalizeTideSource(source)
	secrets := photographyIntegrationSecretsForUse()
	if !secrets.TideEnabled {
		return unavailableTide(source, "潮汐服务未启用"), nil
	}
	if source == "stormglass" {
		if secrets.TideKey == "" {
			return unavailableTide(source, "未配置全球潮汐服务 Key"), nil
		}
		return fetchStormglassTideSummary(latitude, longitude, date, source, timezone, secrets.TideKey, secrets.TideHost)
	}
	if source != "noaa-coops" {
		return unavailableTide(source, "未识别潮汐数据源"), nil
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("NOAA_COOPS_BASE_URL")), "/")
	if base == "" {
		base = "https://api.tidesandcurrents.noaa.gov"
	}
	var stations photographyNOAAStationsResponse
	if err := fetchPhotographyJSON(base+"/mdapi/prod/webapi/stations.json?type=waterlevels", &stations); err != nil {
		return unavailableTide(source, "NOAA 潮汐站点服务不可用"), err
	}
	nearest, distance := nearestPhotographyStation(latitude, longitude, stations.Stations)
	if nearest.ID == "" || distance > 300 {
		return unavailableTide(source, "附近 300 公里内暂无 NOAA 潮汐站点"), nil
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	parsed, parseErr := time.ParseInLocation(photographyDateLayout, date, loc)
	if parseErr != nil {
		return unavailableTide(source, "潮汐日期不合法"), parseErr
	}
	q := url.Values{}
	q.Set("product", "predictions")
	q.Set("datum", "MLLW")
	q.Set("station", nearest.ID)
	q.Set("begin_date", parsed.Format("20060102"))
	q.Set("end_date", parsed.AddDate(0, 0, 1).Format("20060102"))
	q.Set("time_zone", "lst_ldt")
	q.Set("units", "metric")
	q.Set("format", "json")
	var predictions photographyNOAAPredictionsResponse
	if err := fetchPhotographyJSON(base+"/api/prod/datagetter?"+q.Encode(), &predictions); err != nil {
		return unavailableTide(source, "NOAA 潮汐预测服务不可用"), err
	}
	events := classifyPhotographyTides(predictions.Predictions, loc)
	distanceCopy := distance
	result := PhotographyTideSummary{Available: true, Source: source, Station: nearest.Name, DistanceKm: &distanceCopy, Events: events}
	for _, item := range predictions.Predictions {
		height, _ := strconv.ParseFloat(item.Value, 64)
		result.Curve = append(result.Curve, PhotographyTideEvent{Time: item.Time, Type: "level", Height: height})
	}
	if len(result.Curve) > 0 {
		result.CurrentLevel = &result.Curve[0].Height
	}
	for i := range events {
		if events[i].Type == "high" && result.NextHigh == nil {
			e := events[i]
			result.NextHigh = &e
		}
		if events[i].Type == "low" && result.NextLow == nil {
			e := events[i]
			result.NextLow = &e
		}
	}
	return result, nil
}

func fetchStormglassTideSummary(latitude, longitude float64, date, source, timezone, key, host string) (PhotographyTideSummary, error) {
	base := strings.TrimRight(strings.TrimSpace(host), "/")
	if base == "" {
		base = "https://api.stormglass.io/v2"
	}
	if !strings.HasSuffix(base, "/v2") {
		base += "/v2"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	start, err := time.ParseInLocation(photographyDateLayout, date, loc)
	if err != nil {
		return unavailableTide(source, "潮汐日期不合法"), err
	}
	q := url.Values{}
	q.Set("lat", strconv.FormatFloat(latitude, 'f', 6, 64))
	q.Set("lng", strconv.FormatFloat(longitude, 'f', 6, 64))
	q.Set("start", start.UTC().Format(time.RFC3339))
	q.Set("end", start.AddDate(0, 0, 1).UTC().Format(time.RFC3339))
	q.Set("params", "sg")
	headers := http.Header{}
	headers.Set("Authorization", key)
	var extremes stormglassExtremesResponse
	if err := fetchPhotographyJSONWithHeaders("stormglass", base+"/tide/extremes/point?"+q.Encode(), &extremes, headers); err != nil {
		return unavailableTide(source, "全球潮汐极值服务不可用"), err
	}
	result := PhotographyTideSummary{Source: source, Station: "Stormglass 全球潮汐点", Events: []PhotographyTideEvent{}}
	now := time.Now().UTC()
	for _, item := range extremes.Data {
		parsed, parseErr := time.Parse(time.RFC3339, item.Time)
		if parseErr != nil {
			continue
		}
		typeName := strings.ToLower(strings.TrimSpace(item.Type))
		if typeName != "high" && typeName != "low" {
			continue
		}
		event := PhotographyTideEvent{Time: parsed.In(loc).Format(time.RFC3339), Type: typeName, Height: item.Height}
		result.Events = append(result.Events, event)
		if parsed.After(now) && typeName == "high" && result.NextHigh == nil {
			eventCopy := event
			result.NextHigh = &eventCopy
		}
		if parsed.After(now) && typeName == "low" && result.NextLow == nil {
			eventCopy := event
			result.NextLow = &eventCopy
		}
	}
	var sea stormglassSeaLevelResponse
	if err := fetchPhotographyJSONWithHeaders("stormglass", base+"/tide/sea-level/point?"+q.Encode(), &sea, headers); err == nil && len(sea.Data) > 0 {
		bestDistance := time.Duration(1<<63 - 1)
		var best float64
		foundBest := false
		seaCurve := make([]PhotographyTideEvent, 0, len(sea.Data))
		for _, item := range sea.Data {
			parsed, parseErr := time.Parse(time.RFC3339, item.Time)
			if parseErr != nil {
				continue
			}
			value, ok := stormglassSeaLevelValueOf(item)
			if !ok {
				continue
			}
			distance := parsed.Sub(time.Now().UTC())
			if distance < 0 {
				distance = -distance
			}
			if distance < bestDistance {
				bestDistance, best, foundBest = distance, value, true
			}
			seaCurve = append(seaCurve, PhotographyTideEvent{Time: parsed.In(loc).Format(time.RFC3339), Type: "level", Height: value})
		}
		if foundBest {
			result.CurrentLevel = &best
		}
		if len(seaCurve) > 0 {
			result.Curve = seaCurve
		}
	}
	result.Available = len(result.Events) > 0 || result.CurrentLevel != nil
	if !result.Available {
		result.Message = "全球潮汐服务未返回当前日期数据"
		return result, nil
	}
	result.Message = "潮汐数据由 Stormglass 全球潮汐服务提供"
	return result, nil
}

func stormglassSeaLevelValueOf(item stormglassSeaLevelItem) (float64, bool) {
	if item.SG != nil {
		return item.SG.Value, true
	}
	if item.NOAA != nil {
		return item.NOAA.Value, true
	}
	if item.Value != nil {
		return *item.Value, true
	}
	return 0, false
}

func nearestPhotographyStation(lat, lon float64, stations []photographyNOAAStation) (photographyNOAAStation, float64) {
	var nearest photographyNOAAStation
	distance := math.MaxFloat64
	for _, s := range stations {
		d := photographyDistanceKm(lat, lon, s.Lat, s.Lng)
		if d < distance {
			nearest, distance = s, d
		}
	}
	return nearest, distance
}
func photographyDistanceKm(a, b, c, d float64) float64 {
	const earth = 6371
	dlat := (c - a) * math.Pi / 180
	dlon := (d - b) * math.Pi / 180
	x := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(a*math.Pi/180)*math.Cos(c*math.Pi/180)*math.Sin(dlon/2)*math.Sin(dlon/2)
	return earth * 2 * math.Atan2(math.Sqrt(x), math.Sqrt(1-x))
}
func classifyPhotographyTides(items []photographyNOAAPrediction, loc *time.Location) []PhotographyTideEvent {
	result := []PhotographyTideEvent{}
	for i := 1; i < len(items)-1; i++ {
		left, _ := strconv.ParseFloat(items[i-1].Value, 64)
		value, err := strconv.ParseFloat(items[i].Value, 64)
		right, _ := strconv.ParseFloat(items[i+1].Value, 64)
		if err != nil {
			continue
		}
		kind := ""
		if value >= left && value >= right {
			kind = "high"
		}
		if value <= left && value <= right {
			kind = "low"
		}
		if kind != "" {
			t := items[i].Time
			if parsed, err := time.ParseInLocation("2006-01-02 15:04", t, loc); err == nil {
				t = parsed.Format(time.RFC3339)
			}
			result = append(result, PhotographyTideEvent{Time: t, Type: kind, Height: value})
		}
	}
	return result
}
