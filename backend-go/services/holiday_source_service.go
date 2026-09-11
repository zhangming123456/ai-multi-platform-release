package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/models"
)

const (
	HolidayRefreshManual  = "manual"
	HolidayRefreshHourly  = "hourly"
	HolidayRefreshDaily   = "daily"
	HolidayRefreshWeekly  = "weekly"
	HolidayRefreshMonthly = "monthly"
	HolidayRefreshYearly  = "yearly"

	HolidayStatusPending = "pending"
	HolidayStatusSuccess = "success"
	HolidayStatusFailed  = "failed"
)

const (
	holidaySchedulerTickInterval = time.Minute
	holidayFailureRetryInterval  = 10 * time.Minute
	holidaySourceErrorMaxRunes   = 200
	holidayDefaultCalendarTitle  = "节假日日历"
	holidaySourceNameMaxRunes    = 100
	holidaySourceDefaultColor    = "#007AFF"
)

var holidaySourceColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func normalizeHolidaySourceColor(value string) string {
	value = strings.TrimSpace(value)
	if holidaySourceColorPattern.MatchString(value) {
		return strings.ToUpper(value)
	}
	return holidaySourceDefaultColor
}

type HolidayRefreshOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func HolidayRefreshIntervalOptions() []HolidayRefreshOption {
	return []HolidayRefreshOption{
		{Value: HolidayRefreshManual, Label: "手动"},
		{Value: HolidayRefreshHourly, Label: "每小时"},
		{Value: HolidayRefreshDaily, Label: "每天"},
		{Value: HolidayRefreshWeekly, Label: "每周"},
		{Value: HolidayRefreshMonthly, Label: "每月"},
		{Value: HolidayRefreshYearly, Label: "每年"},
	}
}

func normalizeHolidayRefreshInterval(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case HolidayRefreshHourly:
		return HolidayRefreshHourly
	case HolidayRefreshDaily:
		return HolidayRefreshDaily
	case HolidayRefreshWeekly:
		return HolidayRefreshWeekly
	case HolidayRefreshMonthly:
		return HolidayRefreshMonthly
	case HolidayRefreshYearly:
		return HolidayRefreshYearly
	default:
		return HolidayRefreshManual
	}
}

func validateHolidaySourceURL(raw string) error {
	if raw == "" {
		return errors.New("订阅地址不能为空")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("订阅地址格式不正确")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("订阅地址必须以 http:// 或 https:// 开头")
	}
	if parsed.Host == "" {
		return errors.New("订阅地址缺少主机名")
	}
	return nil
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func ListHolidaySources() ([]models.HolidaySource, error) {
	o := GetOrm()
	sources := make([]models.HolidaySource, 0)
	_, err := o.QueryTable(new(models.HolidaySource)).OrderBy("created_at").All(&sources)
	for i := range sources {
		sources[i].Color = normalizeHolidaySourceColor(sources[i].Color)
	}
	return sources, err
}

func GetHolidaySource(id string) (*models.HolidaySource, error) {
	o := GetOrm()
	source := &models.HolidaySource{ID: id}
	if err := o.Read(source); err != nil {
		return nil, errors.New("订阅源不存在")
	}
	return source, nil
}

func CreateHolidaySource(name, sourceURL, color string, enabled bool, interval string) (*models.HolidaySource, error) {
	name = truncateRunes(strings.TrimSpace(name), holidaySourceNameMaxRunes)
	if name == "" {
		return nil, errors.New("订阅源名称不能为空")
	}
	sourceURL = strings.TrimSpace(sourceURL)
	if err := validateHolidaySourceURL(sourceURL); err != nil {
		return nil, err
	}

	days, err := fetchHolidayDays(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("订阅源校验失败：%s", err.Error())
	}

	now := time.Now()
	source := &models.HolidaySource{
		ID:              uuid.NewString(),
		Name:            name,
		Color:           normalizeHolidaySourceColor(color),
		URL:             sourceURL,
		Enabled:         enabled,
		RefreshInterval: normalizeHolidayRefreshInterval(interval),
		LastRefreshedAt: &now,
		LastAttemptAt:   &now,
		LastStatus:      HolidayStatusSuccess,
		EventCount:      len(days),
	}
	o := GetOrm()
	if _, err := o.Insert(source); err != nil {
		return nil, err
	}
	if err := replaceHolidayEvents(o, source.ID, days); err != nil {
		return nil, err
	}
	return source, nil
}

func UpdateHolidaySource(id, name, sourceURL, color string, enabled bool, interval string) (*models.HolidaySource, error) {
	source, err := GetHolidaySource(id)
	if err != nil {
		return nil, err
	}
	name = truncateRunes(strings.TrimSpace(name), holidaySourceNameMaxRunes)
	if name == "" {
		return nil, errors.New("订阅源名称不能为空")
	}
	sourceURL = strings.TrimSpace(sourceURL)
	if err := validateHolidaySourceURL(sourceURL); err != nil {
		return nil, err
	}

	urlChanged := sourceURL != source.URL
	if urlChanged {
		if _, err := fetchHolidayDays(sourceURL); err != nil {
			return nil, fmt.Errorf("订阅源校验失败：%s", err.Error())
		}
	}

	o := GetOrm()
	source.Name = name
	source.Color = normalizeHolidaySourceColor(color)
	source.URL = sourceURL
	source.Enabled = enabled
	source.RefreshInterval = normalizeHolidayRefreshInterval(interval)
	if _, err := o.Update(source); err != nil {
		return nil, err
	}
	if urlChanged {
		return RefreshHolidaySource(id)
	}
	return source, nil
}

func DeleteHolidaySource(id string) error {
	o := GetOrm()
	if _, err := o.QueryTable(new(models.HolidayEvent)).Filter("source_id", id).Delete(); err != nil {
		return err
	}
	if _, err := o.QueryTable(new(models.HolidaySource)).Filter("id", id).Delete(); err != nil {
		return err
	}
	return nil
}

func RefreshHolidaySource(id string) (*models.HolidaySource, error) {
	source, err := GetHolidaySource(id)
	if err != nil {
		return nil, err
	}

	o := GetOrm()
	now := time.Now()
	days, fetchErr := fetchHolidayDays(source.URL)
	if fetchErr != nil {
		source.LastStatus = HolidayStatusFailed
		source.LastError = truncateRunes(fetchErr.Error(), holidaySourceErrorMaxRunes)
		source.LastAttemptAt = &now
		if _, err := o.Update(source, "LastStatus", "LastError", "LastAttemptAt"); err != nil {
			return nil, err
		}
		return source, fetchErr
	}

	if err := replaceHolidayEvents(o, source.ID, days); err != nil {
		return nil, err
	}
	source.LastStatus = HolidayStatusSuccess
	source.LastError = ""
	source.EventCount = len(days)
	source.LastRefreshedAt = &now
	source.LastAttemptAt = &now
	if _, err := o.Update(source, "LastStatus", "LastError", "EventCount", "LastRefreshedAt", "LastAttemptAt"); err != nil {
		return nil, err
	}
	return source, nil
}

func RefreshAllHolidaySources() int {
	sources, err := ListHolidaySources()
	if err != nil {
		return 0
	}
	count := 0
	for i := range sources {
		if _, err := RefreshHolidaySource(sources[i].ID); err == nil {
			count++
		}
	}
	return count
}

func replaceHolidayEvents(o orm.Ormer, sourceID string, days []HolidayDay) error {
	if _, err := o.QueryTable(new(models.HolidayEvent)).Filter("source_id", sourceID).Delete(); err != nil {
		return err
	}
	if len(days) == 0 {
		return nil
	}
	events := make([]models.HolidayEvent, 0, len(days))
	for _, day := range days {
		encoded, err := json.Marshal(day.Labels)
		if err != nil {
			encoded = []byte("[]")
		}
		events = append(events, models.HolidayEvent{
			ID:       uuid.NewString(),
			SourceID: sourceID,
			Date:     day.Date,
			Name:     day.Name,
			Kind:     day.Kind,
			Rest:     day.Rest,
			Work:     day.Work,
			Labels:   string(encoded),
		})
	}
	if _, err := o.InsertMulti(len(events), events); err != nil {
		return err
	}
	return nil
}

func GetHolidayCalendar() (*HolidayCalendar, error) {
	o := GetOrm()
	sources := make([]models.HolidaySource, 0)
	if _, err := o.QueryTable(new(models.HolidaySource)).Filter("enabled", true).OrderBy("created_at").All(&sources); err != nil {
		return nil, err
	}

	calendar := &HolidayCalendar{
		Title:     holidayDefaultCalendarTitle,
		UpdatedAt: time.Now().Format("2006-01-02T15:04:05"),
		Days:      make([]HolidayDay, 0),
	}
	if len(sources) == 0 {
		return calendar, nil
	}

	sourceIDs := make([]string, 0, len(sources))
	sourceURLs := make([]string, 0, len(sources))
	var latest time.Time
	for i := range sources {
		sourceIDs = append(sourceIDs, sources[i].ID)
		sourceURLs = append(sourceURLs, sources[i].URL)
		if sources[i].LastRefreshedAt != nil && sources[i].LastRefreshedAt.After(latest) {
			latest = *sources[i].LastRefreshedAt
		}
	}

	events := make([]models.HolidayEvent, 0)
	if _, err := o.QueryTable(new(models.HolidayEvent)).Filter("source_id__in", sourceIDs).All(&events); err != nil {
		return nil, err
	}

	byDate := map[string]*holidayAgg{}
	for i := range events {
		event := events[i]
		agg := byDate[event.Date]
		if agg == nil {
			agg = &holidayAgg{}
			byDate[event.Date] = agg
		}
		if event.Rest {
			agg.rest = true
		}
		if event.Work {
			agg.work = true
		}
		agg.entries = append(agg.entries, holidayLabel{name: event.Name, kind: event.Kind})
	}

	days := holidayDaysFromAgg(byDate)
	calendar.Source = strings.Join(sourceURLs, "、")
	calendar.Days = days
	calendar.Total = len(days)
	if !latest.IsZero() {
		calendar.UpdatedAt = latest.Local().Format("2006-01-02T15:04:05")
	}
	if len(days) > 0 {
		calendar.Start = days[0].Date
		calendar.End = days[len(days)-1].Date
	}
	return calendar, nil
}

func EnsureDefaultHolidaySource() error {
	o := GetOrm()
	if o.QueryTable(new(models.HolidaySource)).Exist() {
		return nil
	}
	source := &models.HolidaySource{
		ID:              uuid.NewString(),
		Name:            "中国大陆节假日（iCloud）",
		Color:           holidaySourceDefaultColor,
		URL:             HolidayICSURL,
		Enabled:         true,
		RefreshInterval: HolidayRefreshDaily,
		LastStatus:      HolidayStatusPending,
	}
	_, err := o.Insert(source)
	return err
}

func StartHolidayRefreshScheduler() {
	go func() {
		refreshDueHolidaySources()
		ticker := time.NewTicker(holidaySchedulerTickInterval)
		defer ticker.Stop()
		for range ticker.C {
			refreshDueHolidaySources()
		}
	}()
}

func refreshDueHolidaySources() {
	sources, err := ListHolidaySources()
	if err != nil {
		return
	}
	now := time.Now()
	for i := range sources {
		if !isHolidaySourceDue(sources[i], now) {
			continue
		}
		_, _ = RefreshHolidaySource(sources[i].ID)
	}
}

func isHolidaySourceDue(source models.HolidaySource, now time.Time) bool {
	if !source.Enabled {
		return false
	}
	interval := normalizeHolidayRefreshInterval(source.RefreshInterval)
	if interval == HolidayRefreshManual {
		return false
	}
	if source.LastAttemptAt == nil {
		return true
	}
	if source.LastStatus == HolidayStatusFailed {
		return now.Sub(*source.LastAttemptAt) >= holidayFailureRetryInterval
	}
	next, ok := holidayNextRefreshTime(interval, *source.LastAttemptAt)
	if !ok {
		return false
	}
	return !now.Before(next)
}

func holidayNextRefreshTime(interval string, base time.Time) (time.Time, bool) {
	switch interval {
	case HolidayRefreshHourly:
		return base.Add(time.Hour), true
	case HolidayRefreshDaily:
		return base.Add(24 * time.Hour), true
	case HolidayRefreshWeekly:
		return base.Add(7 * 24 * time.Hour), true
	case HolidayRefreshMonthly:
		return base.AddDate(0, 1, 0), true
	case HolidayRefreshYearly:
		return base.AddDate(1, 0, 0), true
	default:
		return time.Time{}, false
	}
}
