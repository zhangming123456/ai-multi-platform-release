package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChinaWeatherCodeAndTextMapping(t *testing.T) {
	cases := []struct {
		code string
		want int
	}{
		{"00", 0},
		{"01", 2},
		{"02", 3},
		{"04", 95},
		{"07", 61},
		{"", 0},
		{"unknown", 3},
	}
	for _, item := range cases {
		if got := chinaWeatherCodeToWMO(item.code); got != item.want {
			t.Errorf("chinaWeatherCodeToWMO(%q) = %d, want %d", item.code, got, item.want)
		}
	}
	if got := chinaWeatherTextToWMO("雷阵雨"); got != 95 {
		t.Errorf("雷阵雨 should map to thunderstorm, got %d", got)
	}
	if got := chinaWeatherTextToWMO("晴天"); got != 0 {
		t.Errorf("晴天 should map to clear, got %d", got)
	}
}

func TestExtractPhotographyJSONVar(t *testing.T) {
	body := `var cityDZ = {};var fc = {"f":[{"fa":"02","fb":"01","fc":"31","fd":"25","fi":"9/16","fj":"今天"}]};var other = 1;`
	raw, err := extractPhotographyJSONVar(body, "fc")
	if err != nil {
		t.Fatalf("extractPhotographyJSONVar() error = %v", err)
	}
	var payload chinaWeatherForecastPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal fc payload: %v", err)
	}
	if len(payload.Days) != 1 || payload.Days[0].DayWeatherCode != "02" || payload.Days[0].Date != "9/16" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if _, err := extractPhotographyJSONVar(body, "missing"); err == nil {
		t.Fatal("missing variable should return error")
	}
}

func TestApplyPhotographyDailyEstimatesRecomputesScores(t *testing.T) {
	code := 3
	days := []PhotographyForecastDay{{Date: "2026-09-16", WeatherCode: &code}}
	applyPhotographyDailyEstimates(days, nil)

	day := days[0]
	if day.CloudCover == nil || day.PrecipitationProbability == nil {
		t.Fatalf("estimated values should be filled: %+v", day)
	}
	if day.SunriseScore <= 0 || day.SunriseLevel == "" || day.SunsetLevel == "" {
		t.Fatalf("scores should be recalculated: %+v", day)
	}
	if len(day.SunriseReasons) == 0 || day.SunriseReasons[len(day.SunriseReasons)-1] != photographyEstimatedReason {
		t.Fatalf("estimated reason should be appended: %+v", day.SunriseReasons)
	}
}

func TestPhotographyChinaCoverageRanges(t *testing.T) {
	if !isMainlandChinaCoordinate(22.54, 114.05) || !isShenzhenCoordinate(22.54, 114.05) {
		t.Fatal("Shenzhen coordinate should be inside both coverage ranges")
	}
	if isMainlandChinaCoordinate(35.68, 139.69) {
		t.Fatal("Tokyo coordinate should not be inside mainland China coverage")
	}
	if isShenzhenCoordinate(23.13, 113.26) {
		t.Fatal("Guangzhou coordinate should not be inside Shenzhen coverage")
	}
}

func TestShenzhenWeatherEndpointKeepsExistingQuery(t *testing.T) {
	endpoint, err := shenzhenWeatherEndpoint("https://opendata.sz.gov.cn/api/weather?page=1", "app-key")
	if err != nil {
		t.Fatalf("shenzhenWeatherEndpoint() error = %v", err)
	}
	for _, fragment := range []string{"page=1", "rows=200", "appKey=app-key"} {
		if !strings.Contains(endpoint, fragment) {
			t.Fatalf("endpoint %q should contain %q", endpoint, fragment)
		}
	}
	if _, err := shenzhenWeatherEndpoint("", "app-key"); err == nil {
		t.Fatal("empty host should return error")
	}
}

func TestChinaWeatherCityNameCandidates(t *testing.T) {
	cases := map[string][]string{
		"深圳市":     {"深圳市", "深圳"},
		"深圳":      {"深圳"},
		"香港特别行政区": {"香港特别行政区", "香港"},
		"湘西自治州":   {"湘西自治州", "湘西"},
	}
	for input, want := range cases {
		got := chinaWeatherCityNameCandidates(input)
		if len(got) != len(want) {
			t.Fatalf("chinaWeatherCityNameCandidates(%q) = %v, want %v", input, got, want)
		}
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("chinaWeatherCityNameCandidates(%q) = %v, want %v", input, got, want)
			}
		}
	}
}
