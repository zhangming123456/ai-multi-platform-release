package services

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func floatPointer(value float64) *float64 { return &value }
func intPointer(value int) *int           { return &value }

func TestValidatePhotographyCoordinates(t *testing.T) {
	tests := []struct {
		name                string
		latitude, longitude float64
		wantErr             bool
	}{
		{name: "lower bounds", latitude: -90, longitude: -180},
		{name: "upper bounds", latitude: 90, longitude: 180},
		{name: "latitude below range", latitude: -90.0001, longitude: 0, wantErr: true},
		{name: "latitude above range", latitude: 90.0001, longitude: 0, wantErr: true},
		{name: "longitude below range", latitude: 0, longitude: -180.0001, wantErr: true},
		{name: "longitude above range", latitude: 0, longitude: 180.0001, wantErr: true},
		{name: "nan latitude", latitude: math.NaN(), longitude: 0, wantErr: true},
		{name: "infinite longitude", latitude: 0, longitude: math.Inf(1), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhotographyCoordinates(tt.latitude, tt.longitude)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePhotographyCoordinates() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !IsPhotographyValidationError(err) {
				t.Fatalf("expected a photography validation error, got %T", err)
			}
		})
	}
}

func TestValidatePhotographyDateUsesLocationCalendar(t *testing.T) {
	tests := []struct {
		name, date, timezone string
		now                  time.Time
		wantErr              bool
	}{
		{
			name: "positive timezone includes local today and day fifteen",
			date: "2026-09-30", timezone: "Asia/Shanghai",
			now: time.Date(2026, 9, 14, 16, 30, 0, 0, time.UTC),
		},
		{
			name: "positive timezone rejects day sixteen",
			date: "2026-10-01", timezone: "Asia/Shanghai",
			now:     time.Date(2026, 9, 14, 16, 30, 0, 0, time.UTC),
			wantErr: true,
		},
		{
			name: "negative timezone includes its local today",
			date: "2026-09-14", timezone: "America/New_York",
			now: time.Date(2026, 9, 15, 1, 0, 0, 0, time.UTC),
		},
		{
			name: "negative timezone rejects its local yesterday",
			date: "2026-09-13", timezone: "America/New_York",
			now:     time.Date(2026, 9, 15, 1, 0, 0, 0, time.UTC),
			wantErr: true,
		},
		{
			name: "invalid date format",
			date: "2026/09/15", timezone: "UTC",
			now:     time.Date(2026, 9, 15, 1, 0, 0, 0, time.UTC),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhotographyDate(tt.date, tt.timezone, tt.now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePhotographyDate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPhotographyScoreLevels(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{score: 100, want: "excellent"},
		{score: 75, want: "excellent"},
		{score: 74, want: "good"},
		{score: 55, want: "good"},
		{score: 54, want: "fair"},
		{score: 35, want: "fair"},
		{score: 34, want: "poor"},
		{score: 0, want: "poor"},
	}
	for _, tt := range tests {
		if got := photographyScoreLevel(tt.score); got != tt.want {
			t.Errorf("photographyScoreLevel(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}

	cases := []struct {
		name  string
		cloud float64
		rain  float64
		code  int
		score int
		level string
	}{
		{name: "ideal", cloud: 50, rain: 10, code: 0, score: 100, level: "excellent"},
		{name: "moderate", cloud: 19, rain: 20, code: 3, score: 60, level: "good"},
		{name: "fair", cloud: 81, rain: 40, code: 3, score: 45, level: "fair"},
		{name: "poor", cloud: 9, rain: 61, code: 95, score: 10, level: "poor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assessment := ScorePhotographyConditions(floatPointer(tc.cloud), floatPointer(tc.rain), intPointer(tc.code))
			if assessment.Score != tc.score || assessment.Level != tc.level {
				t.Fatalf("ScorePhotographyConditions() = %+v, want score %d level %q", assessment, tc.score, tc.level)
			}
			if len(assessment.Reasons) != 3 {
				t.Fatalf("expected three scoring reasons, got %d", len(assessment.Reasons))
			}
		})
	}
}

func TestSearchPhotographyLocationsParsesOpenMeteoResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("name"); got != "杭州西湖" {
			t.Errorf("name query = %q, want 杭州西湖", got)
		}
		if got := r.URL.Query().Get("count"); got != "8" {
			t.Errorf("count query = %q, want 8", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{{
				"name": "西湖", "country": "中国", "country_code": "CN", "admin1": "浙江省",
				"latitude": 30.242, "longitude": 120.148, "timezone": "Asia/Shanghai",
			}},
		})
	}))
	defer server.Close()
	t.Setenv("OPEN_METEO_GEOCODING_BASE_URL", server.URL)

	locations, err := SearchPhotographyLocations(" 杭州西湖 ")
	if err != nil {
		t.Fatalf("SearchPhotographyLocations() error = %v", err)
	}
	if len(locations) != 1 {
		t.Fatalf("got %d locations, want 1", len(locations))
	}
	if locations[0].DisplayName != "西湖 · 浙江省 · 中国" || locations[0].Timezone != "Asia/Shanghai" {
		t.Fatalf("location = %+v", locations[0])
	}
}

func TestFetchPhotographyForecastParsesAndAssessesResponse(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	localToday, err := time.ParseInLocation(photographyDateLayout, time.Now().In(location).Format(photographyDateLayout), location)
	if err != nil {
		t.Fatal(err)
	}
	dates := make([]string, PhotographyForecastDays)
	sunrises := make([]string, PhotographyForecastDays)
	sunsets := make([]string, PhotographyForecastDays)
	weatherCodes := make([]int, PhotographyForecastDays)
	precipitationMax := make([]float64, PhotographyForecastDays)
	for i := 0; i < PhotographyForecastDays; i++ {
		date := localToday.AddDate(0, 0, i).Format(photographyDateLayout)
		dates[i] = date
		sunrises[i] = date + "T06:00"
		sunsets[i] = date + "T18:00"
		weatherCodes[i] = 95
		precipitationMax[i] = 99
	}
	firstDate := dates[0]
	responseBody := map[string]interface{}{
		"latitude":  30.242,
		"longitude": 120.148,
		"timezone":  "Asia/Shanghai",
		"daily": map[string]interface{}{
			"time": dates, "sunrise": sunrises, "sunset": sunsets,
			"weather_code": weatherCodes, "precipitation_probability_max": precipitationMax,
		},
		"hourly": map[string]interface{}{
			"time": []string{
				firstDate + "T05:00", firstDate + "T06:00", firstDate + "T07:00",
				firstDate + "T17:00", firstDate + "T18:00", firstDate + "T19:00",
			},
			"cloud_cover":               []float64{20, 40, 60, 100, 100, 100},
			"precipitation_probability": []float64{10, 10, 10, 70, 70, 70},
			"precipitation":             []float64{0, 0, 0, 2, 2, 2},
			"weather_code":              []int{0, 0, 0, 95, 95, 95},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("forecast_days") != "16" || query.Get("timezone") != "auto" {
			t.Errorf("unexpected forecast query: %v", query)
		}
		if !strings.Contains(query.Get("daily"), "sunrise") || !strings.Contains(query.Get("hourly"), "cloud_cover") {
			t.Errorf("missing weather variables: %v", query)
		}
		_ = json.NewEncoder(w).Encode(responseBody)
	}))
	defer server.Close()
	t.Setenv("OPEN_METEO_FORECAST_BASE_URL", server.URL)

	forecast, err := FetchPhotographyForecast(30.242, 120.148, firstDate)
	if err != nil {
		t.Fatalf("FetchPhotographyForecast() error = %v", err)
	}
	if forecast.Timezone != "Asia/Shanghai" || len(forecast.Days) != PhotographyForecastDays {
		t.Fatalf("forecast metadata = timezone %q days %d", forecast.Timezone, len(forecast.Days))
	}
	if forecast.Days[0].SunriseScore != 100 || forecast.Days[0].SunriseLevel != "excellent" {
		t.Fatalf("sunrise assessment = %+v", forecast.Days[0])
	}
	if forecast.Days[0].SunsetScore != 10 || forecast.Days[0].SunsetLevel != "poor" {
		t.Fatalf("sunset assessment = %+v", forecast.Days[0])
	}
	if len(forecast.Days[0].SunriseReasons) != 3 || len(forecast.Days[0].SunsetReasons) != 3 {
		t.Fatalf("reasons were not parsed: %+v", forecast.Days[0])
	}
}

func TestFetchPhotographyForecastReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "{\"reason\":\"upstream unavailable\"}", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	t.Setenv("OPEN_METEO_FORECAST_BASE_URL", server.URL)

	target := time.Now().UTC().Format(photographyDateLayout)
	_, err := FetchPhotographyForecast(0, 0, target)
	if err == nil || !strings.Contains(err.Error(), "天气服务返回异常") {
		t.Fatalf("FetchPhotographyForecast() error = %v", err)
	}
	if IsPhotographyValidationError(err) {
		t.Fatalf("upstream error was incorrectly classified as validation error: %v", err)
	}
}

func TestPhotographyLogEndpointRemovesQuery(t *testing.T) {
	endpoint := "https://example.com/weather?apikey=secret&latitude=1"
	got := photographyLogEndpoint(endpoint)
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.RawQuery != "" || strings.Contains(got, "secret") {
		t.Fatalf("log endpoint leaked query: %q", got)
	}
}

func TestPhotographyWeatherSources(t *testing.T) {
	sources := ListPhotographyWeatherSources()
	if len(sources) < 4 {
		t.Fatalf("got %d weather sources, want at least 4", len(sources))
	}
	if NormalizePhotographyWeatherSource("") != PhotographyWeatherSourceBestMatch {
		t.Fatalf("empty source should use best match")
	}
	if NormalizePhotographyWeatherSource("invalid") != "" {
		t.Fatalf("invalid source should not be normalized")
	}
	if err := ValidatePhotographyWeatherSource(PhotographyWeatherSourceECMWF); err != nil {
		t.Fatalf("ValidatePhotographyWeatherSource() error = %v", err)
	}
}

func TestFetchPhotographyForecastWithSourceSelectsModel(t *testing.T) {
	target := time.Now().UTC().Format(photographyDateLayout)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("models"); got != "ecmwf_ifs025" {
			t.Errorf("models query = %q, want ecmwf_ifs025", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"latitude":  30.242,
			"longitude": 120.148,
			"timezone":  "UTC",
			"daily": map[string]interface{}{
				"time": []string{target}, "sunrise": []string{target + "T06:00"}, "sunset": []string{target + "T18:00"},
				"weather_code": []int{0}, "precipitation_probability_max": []float64{0},
			},
			"hourly": map[string]interface{}{
				"time":        []string{target + "T06:00", target + "T18:00"},
				"cloud_cover": []float64{50, 50}, "precipitation_probability": []float64{0, 0},
				"precipitation": []float64{0, 0}, "weather_code": []int{0, 0},
			},
		})
	}))
	defer server.Close()
	t.Setenv("OPEN_METEO_FORECAST_BASE_URL", server.URL)

	forecast, err := FetchPhotographyForecastWithSource(30.242, 120.148, target, PhotographyWeatherSourceECMWF)
	if err != nil {
		t.Fatalf("FetchPhotographyForecastWithSource() error = %v", err)
	}
	if forecast.Source != PhotographyWeatherSourceECMWF || forecast.SourceName == "" {
		t.Fatalf("forecast source = %q/%q", forecast.Source, forecast.SourceName)
	}
}

func TestValidatePhotographyDateWithCustomForecastDays(t *testing.T) {
	now := time.Date(2026, 9, 15, 1, 0, 0, 0, time.UTC)
	if err := ValidatePhotographyDateWithDays("2026-09-24", "UTC", now, 10); err != nil {
		t.Fatalf("day nine should be accepted: %v", err)
	}
	if err := ValidatePhotographyDateWithDays("2026-09-25", "UTC", now, 10); err == nil {
		t.Fatal("day ten should be rejected for a ten-day source")
	}
}

func TestQWeatherIconMapping(t *testing.T) {
	cases := map[string]int{"100": 0, "101": 2, "104": 3, "300": 61, "400": 71, "500": 45, "999": 3}
	for icon, want := range cases {
		if got := qWeatherIconToWMO(icon); got != want {
			t.Errorf("qWeatherIconToWMO(%q) = %d, want %d", icon, got, want)
		}
	}
}

func TestFetchQWeatherPhotographyForecast(t *testing.T) {
	target := time.Now().UTC().Format(photographyDateLayout)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-QW-Api-Key") != "test-key" {
			t.Errorf("missing qweather api key header")
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/geo/v2/city/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": "200", "location": []map[string]string{{"tz": "UTC"}}})
		case strings.HasPrefix(r.URL.Path, "/weather/v1/daily/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": "200", "days": []map[string]interface{}{{"forecastStartTime": target + "T00:00", "astro": map[string]string{"sunrise": target + "T06:00", "sunset": target + "T18:00"}, "daytime": map[string]interface{}{"condition": map[string]string{"code": "100"}, "cloudCover": 0.5, "precipitation": map[string]interface{}{"probability": 0.1}}, "nighttime": map[string]interface{}{"condition": map[string]string{"code": "101"}, "cloudCover": 0.5, "precipitation": map[string]interface{}{"probability": 0.1}}}}})
		case strings.HasPrefix(r.URL.Path, "/weather/v1/hourly/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": "200", "hours": []map[string]interface{}{{"forecastTime": target + "T06:00", "condition": map[string]string{"code": "100"}, "cloudCover": 0.5, "precipitation": map[string]interface{}{"probability": 0.1, "amount": map[string]float64{"value": 0}}}, {"forecastTime": target + "T18:00", "condition": map[string]string{"code": "100"}, "cloudCover": 0.5, "precipitation": map[string]interface{}{"probability": 0.1, "amount": map[string]float64{"value": 0}}}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("QWEATHER_API_HOST", server.URL)
	t.Setenv("QWEATHER_API_KEY", "test-key")
	t.Setenv("QWEATHER_API_TOKEN", "")
	forecast, err := FetchPhotographyForecastWithSource(30, 120, target, PhotographyWeatherSourceQWeather)
	if err != nil {
		t.Fatalf("FetchPhotographyForecastWithSource() error = %v", err)
	}
	if forecast.Source != PhotographyWeatherSourceQWeather || forecast.Timezone != "UTC" || len(forecast.Days) != 1 {
		t.Fatalf("unexpected qweather forecast: %+v", forecast)
	}
	if forecast.Days[0].SunriseScore < 0 || forecast.Days[0].SunriseScore > 100 {
		t.Fatalf("invalid sunrise score: %+v", forecast.Days[0])
	}
}

func TestPhotographyWeatherConfigViewsMaskCredentials(t *testing.T) {
	t.Setenv("OPEN_METEO_API_KEY", "open-meteo-secret")
	t.Setenv("QWEATHER_API_KEY", "qweather-secret")
	t.Setenv("QWEATHER_API_TOKEN", "")
	t.Setenv("QWEATHER_API_HOST", "https://example.qweather.test")

	configs := ListPhotographyWeatherConfigs()
	if len(configs) != 2 {
		t.Fatalf("got %d weather configs, want 2", len(configs))
	}

	openMeteo := configs[0]
	if openMeteo.Source != "open-meteo" || !openMeteo.Configured {
		t.Fatalf("unexpected Open-Meteo config: %+v", openMeteo)
	}
	if openMeteo.APIKeyMasked != "open****cret" {
		t.Fatalf("Open-Meteo key was not masked as expected: %q", openMeteo.APIKeyMasked)
	}

	qweather := configs[1]
	if qweather.Source != PhotographyWeatherSourceQWeather || !qweather.Configured {
		t.Fatalf("unexpected QWeather config: %+v", qweather)
	}
	if qweather.APIKeyMasked != "qwea****cret" || qweather.APIHost != "https://example.qweather.test" {
		t.Fatalf("QWeather credentials were not represented safely: %+v", qweather)
	}
}

func TestPhotographyWeatherTokenTakesPrecedence(t *testing.T) {
	t.Setenv("QWEATHER_API_KEY", "api-key")
	t.Setenv("QWEATHER_API_TOKEN", "jwt-token")
	t.Setenv("QWEATHER_API_HOST", "")

	configs := ListPhotographyWeatherConfigs()
	qweather := configs[1]
	if qweather.CredentialType != "token" || qweather.APIKeyMasked != "jwt-****oken" {
		t.Fatalf("JWT token should be the active QWeather credential: %+v", qweather)
	}
}

func TestSavePhotographyWeatherConfigValidatesBeforeDatabaseAccess(t *testing.T) {
	if err := SavePhotographyWeatherConfig("unknown", "api_key", "secret", "", false); err == nil {
		t.Fatal("unknown weather source should be rejected")
	}
	if err := SavePhotographyWeatherConfig("qweather", "basic_auth", "secret", "", false); err == nil {
		t.Fatal("unknown credential type should be rejected")
	}
}

func TestSelectPhotographyMapProvider(t *testing.T) {
	if got := SelectPhotographyMapProvider("CN"); got != "amap" {
		t.Fatalf("SelectPhotographyMapProvider(CN) = %q, want amap", got)
	}
	if got := SelectPhotographyMapProvider("cn"); got != "amap" {
		t.Fatalf("SelectPhotographyMapProvider(cn) = %q, want amap", got)
	}
	if got := SelectPhotographyMapProvider("US"); got != "google" {
		t.Fatalf("SelectPhotographyMapProvider(US) = %q, want google", got)
	}
}

func TestQWeatherFormalResponseConversion(t *testing.T) {
	daily := []qWeatherDailyItem{{
		FxDate: "2026-09-15", Sunrise: "05:30", Sunset: "18:40", IconDay: "100", IconNight: "150",
		Pop: "20", Precip: "0.4", TempMax: "30", TempMin: "22",
	}}
	hourly := []qWeatherHourlyItem{{
		FxTime: "2026-09-15T05:30+08:00", Temp: "24", Humidity: "70", Dew: "18", Vis: "10",
		Cloud: "50", Pop: "10", Precip: "0", WindSpeed: "3", Icon: "100",
	}}
	response := buildOpenMeteoResponseFromQWeather(30, 120, "Asia/Shanghai", daily, hourly)
	if len(response.Daily.Time) != 1 || response.Daily.Time[0] != "2026-09-15" {
		t.Fatalf("unexpected daily dates: %+v", response.Daily.Time)
	}
	if response.Daily.Sunrise[0] != "2026-09-15T05:30" || response.Daily.Sunset[0] != "2026-09-15T18:40" {
		t.Fatalf("unexpected sunrise/sunset: %+v / %+v", response.Daily.Sunrise, response.Daily.Sunset)
	}
	if len(response.Hourly.Time) != 1 || response.Hourly.CloudCover[0] != 50 || response.Hourly.PrecipitationProbability[0] != 10 {
		t.Fatalf("unexpected hourly conversion: %+v", response.Hourly)
	}
}

func TestBuildPhotographyForecastDaysIncludesDailyWeatherAndHourlyData(t *testing.T) {
	date := "2026-09-15"
	response := openMeteoForecastResponse{}
	response.Daily.Time = []string{date}
	response.Daily.Sunrise = []string{date + "T05:30"}
	response.Daily.Sunset = []string{date + "T18:40"}
	response.Daily.WeatherCode = []int{1}
	response.Daily.TemperatureMax = []float64{30}
	response.Daily.TemperatureMin = []float64{22}
	response.Daily.PrecipitationProbabilityMax = []float64{20}
	for hour := 0; hour < 24; hour++ {
		response.Hourly.Time = append(response.Hourly.Time, fmt.Sprintf("%sT%02d:00", date, hour))
		response.Hourly.Temperature = append(response.Hourly.Temperature, float64(20+hour))
		response.Hourly.WeatherCode = append(response.Hourly.WeatherCode, 0)
		response.Hourly.RelativeHumidity = append(response.Hourly.RelativeHumidity, 60)
	}
	response.Hourly.Time = append(response.Hourly.Time, "2026-09-16T00:00")
	response.Hourly.Temperature = append(response.Hourly.Temperature, 19)
	response.Hourly.WeatherCode = append(response.Hourly.WeatherCode, 95)

	days := buildPhotographyForecastDays(response)
	if len(days) != 1 {
		t.Fatalf("got %d forecast days, want 1", len(days))
	}
	day := days[0]
	if day.TemperatureMax == nil || *day.TemperatureMax != 30 || day.TemperatureMin == nil || *day.TemperatureMin != 22 {
		t.Fatalf("daily temperatures = %v/%v, want 30/22", day.TemperatureMax, day.TemperatureMin)
	}
	if day.WeatherText != "少云" {
		t.Fatalf("daily weather text = %q, want 少云", day.WeatherText)
	}
	if len(day.Hours) != 24 {
		t.Fatalf("got %d hourly points, want 24", len(day.Hours))
	}
	if day.Hours[0].Time != date+"T00:00" || day.Hours[0].Temperature == nil || *day.Hours[0].Temperature != 20 {
		t.Fatalf("first hourly point = %+v", day.Hours[0])
	}
	if day.Hours[0].WeatherText != "晴朗" || day.Hours[23].Time != date+"T23:00" {
		t.Fatalf("hourly weather mapping = %+v / %+v", day.Hours[0], day.Hours[23])
	}
}

func TestStormglassSeaLevelValueParsing(t *testing.T) {
	var response stormglassSeaLevelResponse
	payload := `{"data":[{"time":"2026-09-15T00:00:00+00:00","sg":{"value":1.25,"source":"sg"}},{"time":"2026-09-15T01:00:00+00:00","value":0.8}]}`
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("unmarshal Stormglass response: %v", err)
	}
	if len(response.Data) != 2 {
		t.Fatalf("got %d Stormglass data points, want 2", len(response.Data))
	}
	if value, ok := stormglassSeaLevelValueOf(response.Data[0]); !ok || value != 1.25 {
		t.Fatalf("nested Stormglass sea level = %v, %v; want 1.25, true", value, ok)
	}
	if value, ok := stormglassSeaLevelValueOf(response.Data[1]); !ok || value != 0.8 {
		t.Fatalf("flat Stormglass sea level = %v, %v; want 0.8, true", value, ok)
	}
}

func TestPhotographyDayJSONUsesDistinctAssessmentFields(t *testing.T) {
	day := PhotographyPhotographyDay{Date: "2026-09-15", Sunrise: "06:00", Sunset: "18:00"}
	day.SunriseAssessment = PhotographyPhenomenonEstimate{Available: true, Level: "excellent"}
	day.SunsetAssessment = PhotographyPhenomenonEstimate{Available: true, Level: "good"}
	encoded, err := json.Marshal(day)
	if err != nil {
		t.Fatalf("marshal PhotographyPhotographyDay: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode PhotographyPhotographyDay: %v", err)
	}
	for _, key := range []string{"sunrise", "sunset", "sunrise_assessment", "sunset_assessment"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing JSON field %q in %s", key, encoded)
		}
	}
}

func TestPhotographyIntegrationSecretEncryptionAndMasking(t *testing.T) {
	t.Setenv("INTEGRATION_ENCRYPTION_KEY", "test-integration-key")
	ciphertext, err := encryptPhotographySecret("super-secret-token")
	if err != nil {
		t.Fatalf("encryptPhotographySecret() error = %v", err)
	}
	if ciphertext == "super-secret-token" || ciphertext == "" {
		t.Fatalf("secret was not encrypted: %q", ciphertext)
	}
	plaintext, err := decryptPhotographySecret(ciphertext)
	if err != nil {
		t.Fatalf("decryptPhotographySecret() error = %v", err)
	}
	if plaintext != "super-secret-token" {
		t.Fatalf("decrypted secret = %q, want original value", plaintext)
	}
	if got := maskPhotographyAPIKey("super-secret-token"); got != "supe****oken" {
		t.Fatalf("maskPhotographyAPIKey() = %q, want supe****oken", got)
	}
	if got := maskPhotographyAPIKey("short"); got != "******" {
		t.Fatalf("short key mask = %q, want ******", got)
	}
}

func TestUnavailableTideSerializesEmptyArrays(t *testing.T) {
	payload, err := json.Marshal(unavailableTide("none", "未选择潮汐数据源"))
	if err != nil {
		t.Fatalf("marshal unavailable tide: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode unavailable tide: %v", err)
	}
	for _, key := range []string{"events", "curve"} {
		value, ok := decoded[key]
		if !ok || string(value) != "[]" {
			t.Fatalf("unavailable tide %s = %s, want []", key, value)
		}
	}
}

func TestNormalizePhotographyOverviewArrays(t *testing.T) {
	overview := &PhotographyOverview{}
	overview.Today.Tide.Events = nil
	overview.Today.Tide.Curve = nil
	overview.Today.Rainbow.Reasons = nil
	overview.Today.Rainbow.Risks = nil
	overview.Today.Aurora.Reasons = nil
	overview.Today.Aurora.Risks = nil
	overview.Days = nil
	overview.Charts.Probability = nil
	overview.Charts.CloudQuality = nil
	overview.Warnings = nil

	normalizePhotographyOverviewArrays(overview)
	if overview.Days == nil || overview.Charts.Probability == nil || overview.Charts.CloudQuality == nil || overview.Warnings == nil {
		t.Fatal("overview top-level arrays should be non-nil after normalization")
	}
	if overview.Today.Tide.Events == nil || overview.Today.Tide.Curve == nil {
		t.Fatal("tide arrays should be non-nil after normalization")
	}
	if overview.Today.Rainbow.Reasons == nil || overview.Today.Rainbow.Risks == nil || overview.Today.Aurora.Reasons == nil || overview.Today.Aurora.Risks == nil {
		t.Fatal("estimate arrays should be non-nil after normalization")
	}
}

func TestFetchPhotographyJSONRetriesTransientStatus(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "temporary unavailable", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	var response map[string]string
	if err := fetchPhotographyJSON(server.URL, &response); err != nil {
		t.Fatalf("fetchPhotographyJSON() error = %v", err)
	}
	if attempts != 3 || response["status"] != "ok" {
		t.Fatalf("attempts=%d response=%v, want three attempts and success", attempts, response)
	}
}

func TestPhotographyForecastUsesOpenMeteoEnvironmentHost(t *testing.T) {
	t.Setenv("OPEN_METEO_FORECAST_BASE_URL", "https://env.example.test/v1/forecast")
	if got := photographyForecastBaseURL(); got != "https://env.example.test/v1/forecast" {
		t.Fatalf("photographyForecastBaseURL() = %q", got)
	}
}
