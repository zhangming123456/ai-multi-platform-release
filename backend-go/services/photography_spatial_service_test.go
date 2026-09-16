package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 空间分布图测试：一次多坐标请求要能解析成数组，并按摄影时段算出概率与云量质量。
func TestFetchPhotographySpatialGridUsesMultiCoordinateResponse(t *testing.T) {
	date := time.Now().In(photographyTimezoneLocation("Asia/Shanghai")).Format(photographyDateLayout)
	var gridRequested bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("daily") == "" {
			// FetchPhotographyTimezone 只取时区。
			_, _ = w.Write([]byte(`{"timezone":"Asia/Shanghai"}`))
			return
		}
		gridRequested = true
		if got := r.URL.Query().Get("timezone"); got != "Asia/Shanghai" {
			t.Errorf("timezone query = %q, want Asia/Shanghai", got)
		}
		if got := len(strings.Split(r.URL.Query().Get("latitude"), ",")); got != photographySpatialRows*photographySpatialCols {
			t.Errorf("latitude count = %d, want %d", got, photographySpatialRows*photographySpatialCols)
		}
		_, _ = w.Write(mustPhotographySpatialFixture(t, date, photographySpatialRows*photographySpatialCols))
	}))
	defer server.Close()
	t.Setenv("OPEN_METEO_FORECAST_BASE_URL", server.URL)

	grid, err := FetchPhotographySpatialGrid(30.25, 120.15, date, PhotographySpatialPeriodSunset, PhotographyWeatherSourceBestMatch)
	if err != nil {
		t.Fatalf("FetchPhotographySpatialGrid() error = %v", err)
	}
	if !gridRequested {
		t.Fatal("expected a multi coordinate grid request")
	}
	if grid.Date != date || grid.Period != PhotographySpatialPeriodSunset || grid.Timezone != "Asia/Shanghai" {
		t.Fatalf("unexpected grid meta: %+v", grid)
	}
	// Open-Meteo 系数据源本身就是网格数据来源，不应提示来源差异。
	if grid.Message != "" {
		t.Fatalf("grid.Message = %q, want empty for Open-Meteo source", grid.Message)
	}
	if grid.StepLatitude != photographySpatialStepDegrees || grid.StepLongitude <= 0 {
		t.Fatalf("unexpected grid steps: %v / %v", grid.StepLatitude, grid.StepLongitude)
	}
	if len(grid.Cells) != photographySpatialRows*photographySpatialCols {
		t.Fatalf("cells = %d, want %d", len(grid.Cells), photographySpatialRows*photographySpatialCols)
	}
	cell := grid.Cells[0]
	if cell.Probability == nil || *cell.Probability != 100 || cell.ProbabilityLevel != "excellent" {
		t.Fatalf("unexpected probability cell: %+v", cell)
	}
	if cell.CloudQuality == nil || *cell.CloudQuality != 100 || cell.CloudQualityLevel != "excellent" {
		t.Fatalf("unexpected cloud quality cell: %+v", cell)
	}
	// 中心点应落在网格正中间。
	centerIndex := (photographySpatialRows*photographySpatialCols - 1) / 2
	if grid.Cells[centerIndex].Latitude != 30.25 || grid.Cells[centerIndex].Longitude != 120.15 {
		t.Fatalf("center cell = %+v", grid.Cells[centerIndex])
	}
}

func TestFetchPhotographySpatialGridRejectsOutOfRangeDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"timezone":"Asia/Shanghai"}`))
	}))
	defer server.Close()
	t.Setenv("OPEN_METEO_FORECAST_BASE_URL", server.URL)

	date := time.Now().In(photographyTimezoneLocation("Asia/Shanghai")).AddDate(0, 0, 20).Format(photographyDateLayout)
	if _, err := FetchPhotographySpatialGrid(30.25, 120.15, date, PhotographySpatialPeriodSunrise, PhotographyWeatherSourceBestMatch); err == nil {
		t.Fatal("expected an out of range date error")
	} else if !IsPhotographyValidationError(err) {
		t.Fatalf("expected a validation error, got %v", err)
	}
}

func TestFetchPhotographySpatialGridSurfacesUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("daily") == "" {
			_, _ = w.Write([]byte(`{"timezone":"Asia/Shanghai"}`))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"reason":"upstream down"}`))
	}))
	defer server.Close()
	t.Setenv("OPEN_METEO_FORECAST_BASE_URL", server.URL)

	date := time.Now().In(photographyTimezoneLocation("Asia/Shanghai")).Format(photographyDateLayout)
	if _, err := FetchPhotographySpatialGrid(30.25, 120.15, date, PhotographySpatialPeriodSunset, PhotographyWeatherSourceBestMatch); err == nil {
		t.Fatal("expected an upstream failure")
	}
}

// 与真实 Open-Meteo 多坐标响应一致：数组，每个元素含 timezone/daily/hourly。
func mustPhotographySpatialFixture(t *testing.T, date string, count int) []byte {
	t.Helper()
	hours := make([]string, 0, 24)
	cloud, rain, codes := make([]float64, 0, 24), make([]float64, 0, 24), make([]int, 0, 24)
	for hour := 0; hour < 24; hour++ {
		hours = append(hours, fmt.Sprintf("%sT%02d:00", date, hour))
		cloud = append(cloud, 50)
		rain = append(rain, 10)
		codes = append(codes, 1)
	}
	items := make([]map[string]any, 0, count)
	for index := 0; index < count; index++ {
		items = append(items, map[string]any{
			"latitude":  30.0 + float64(index)/100,
			"longitude": 120.0 + float64(index)/100,
			"timezone":  "Asia/Shanghai",
			"daily": map[string]any{
				"time":    []string{date},
				"sunrise": []string{date + "T05:30"},
				"sunset":  []string{date + "T18:00"},
			},
			"hourly": map[string]any{
				"time":                      hours,
				"cloud_cover":               cloud,
				"precipitation_probability": rain,
				"weather_code":              codes,
			},
		})
	}
	body, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("fixture marshal error = %v", err)
	}
	return body
}
