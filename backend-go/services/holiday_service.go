package services

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	HolidayICSURL = "https://calendars.icloud.com/holidays/cn_zh.ics"

	HolidayKindHoliday   = "holiday"
	HolidayKindWorkday   = "workday"
	HolidayKindFestival  = "festival"
	HolidayKindSolarTerm = "solar_term"
)

const (
	holidayDateLayout     = "20060102"
	holidayMaxOccurrences = 12
	holidayMaxRangeDays   = 60
)

var holidaySolarTerms = map[string]struct{}{
	"立春": {}, "雨水": {}, "惊蛰": {}, "春分": {}, "清明": {}, "谷雨": {},
	"立夏": {}, "小满": {}, "芒种": {}, "夏至": {}, "小暑": {}, "大暑": {},
	"立秋": {}, "处暑": {}, "白露": {}, "秋分": {}, "寒露": {}, "霜降": {},
	"立冬": {}, "小雪": {}, "大雪": {}, "冬至": {}, "小寒": {}, "大寒": {},
}

type HolidayDay struct {
	Date   string   `json:"date"`
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`
	Rest   bool     `json:"rest"`
	Work   bool     `json:"work"`
	Labels []string `json:"labels"`
}

type HolidayCalendar struct {
	Source    string       `json:"source"`
	Title     string       `json:"title"`
	UpdatedAt string       `json:"updated_at"`
	Start     string       `json:"start"`
	End       string       `json:"end"`
	Total     int          `json:"total"`
	Days      []HolidayDay `json:"days"`
}

type rawHolidayEvent struct {
	start time.Time
	end   time.Time
	name  string
	kind  string
	rest  bool
	work  bool
	rrule string
}

type holidayRRule struct {
	freq     string
	count    int
	interval int
	until    time.Time
}

type holidayLabel struct {
	name string
	kind string
}

type holidayAgg struct {
	rest    bool
	work    bool
	entries []holidayLabel
}

var holidayHTTPClient = &http.Client{Timeout: 20 * time.Second}

func FilterHolidayByYear(cal *HolidayCalendar, year int) *HolidayCalendar {
	prefix := fmt.Sprintf("%04d-", year)
	filtered := &HolidayCalendar{
		Source:    cal.Source,
		Title:     cal.Title,
		UpdatedAt: cal.UpdatedAt,
		Days:      make([]HolidayDay, 0),
	}
	for _, d := range cal.Days {
		if strings.HasPrefix(d.Date, prefix) {
			filtered.Days = append(filtered.Days, d)
		}
	}
	filtered.Total = len(filtered.Days)
	if filtered.Total > 0 {
		filtered.Start = filtered.Days[0].Date
		filtered.End = filtered.Days[filtered.Total-1].Date
	}
	return filtered
}

func fetchHolidayDays(sourceURL string) ([]HolidayDay, error) {
	body, err := fetchHolidayICSBody(sourceURL)
	if err != nil {
		return nil, err
	}
	events, _ := parseHolidayICS(body)
	return buildHolidayDays(events), nil
}

func fetchHolidayICSBody(sourceURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ai-multi-platform-release/1.0")

	resp, err := holidayHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("上游返回状态码 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parseHolidayICS(content string) ([]rawHolidayEvent, string) {
	lines := unfoldICSLines(content)
	events := make([]rawHolidayEvent, 0, 256)
	title := ""

	var current map[string]string
	for _, line := range lines {
		key, value := splitICSProperty(line)
		switch key {
		case "X-WR-CALNAME":
			if title == "" {
				title = value
			}
		case "BEGIN":
			if value == "VEVENT" {
				current = map[string]string{}
			}
		case "END":
			if value == "VEVENT" && current != nil {
				if ev, ok := buildRawEvent(current); ok {
					events = append(events, ev)
				}
				current = nil
			}
		default:
			if current != nil && key != "" {
				current[key] = value
			}
		}
	}
	return events, title
}

func unfoldICSLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	raw := strings.Split(content, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if line == "" {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && len(out) > 0 {
			out[len(out)-1] += line[1:]
			continue
		}
		out = append(out, line)
	}
	return out
}

func splitICSProperty(line string) (string, string) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", ""
	}
	raw := line[:idx]
	value := line[idx+1:]
	if semi := strings.IndexByte(raw, ';'); semi >= 0 {
		raw = raw[:semi]
	}
	return strings.ToUpper(strings.TrimSpace(raw)), unescapeICSText(value)
}

func unescapeICSText(value string) string {
	if !strings.Contains(value, "\\") {
		return value
	}
	replacer := strings.NewReplacer("\\n", "\n", "\\N", "\n", "\\,", ",", "\\;", ";", "\\\\", "\\")
	return replacer.Replace(value)
}

func buildRawEvent(fields map[string]string) (rawHolidayEvent, bool) {
	start, ok := parseICSDate(fields["DTSTART"])
	if !ok {
		return rawHolidayEvent{}, false
	}

	end, endOK := parseICSDate(fields["DTEND"])
	if !endOK || !end.After(start) {
		end = start.AddDate(0, 0, 1)
	}

	name := cleanHolidayName(fields["SUMMARY"])
	if name == "" {
		return rawHolidayEvent{}, false
	}

	ev := rawHolidayEvent{
		start: start,
		end:   end,
		name:  name,
		rrule: strings.TrimSpace(fields["RRULE"]),
	}

	switch strings.ToUpper(strings.TrimSpace(fields["X-APPLE-SPECIAL-DAY"])) {
	case "WORK-HOLIDAY":
		ev.kind = HolidayKindHoliday
		ev.rest = true
	case "ALTERNATE-WORKDAY":
		ev.kind = HolidayKindWorkday
		ev.work = true
	default:
		ev.kind = HolidayKindFestival
	}

	if ev.kind == HolidayKindFestival {
		if _, isTerm := holidaySolarTerms[ev.name]; isTerm {
			ev.kind = HolidayKindSolarTerm
		}
	}
	return ev, true
}

func parseICSDate(value string) (time.Time, bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		return time.Time{}, false
	}
	if len(v) > 8 {
		v = v[:8]
	}
	t, err := time.ParseInLocation(holidayDateLayout, v, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func cleanHolidayName(name string) string {
	s := strings.TrimSpace(name)
	for _, suffix := range []string{"（休）", "（班）", "(休)", "(班)"} {
		s = strings.TrimSuffix(s, suffix)
	}
	return strings.TrimSpace(s)
}

func parseHolidayRRule(raw string) *holidayRRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	rule := &holidayRRule{interval: 1}
	for _, part := range strings.Split(raw, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "FREQ":
			rule.freq = strings.ToUpper(val)
		case "INTERVAL":
			if n, err := strconv.Atoi(val); err == nil {
				rule.interval = n
			}
		case "COUNT":
			if n, err := strconv.Atoi(val); err == nil {
				rule.count = n
			}
		case "UNTIL":
			if t, ok := parseICSDate(val); ok {
				rule.until = t
			}
		}
	}
	if rule.freq == "" {
		return nil
	}
	return rule
}

func expandHolidayOccurrences(ev rawHolidayEvent) []rawHolidayEvent {
	rule := parseHolidayRRule(ev.rrule)
	if rule == nil || rule.freq != "YEARLY" {
		return []rawHolidayEvent{ev}
	}

	count := rule.count
	if count <= 0 || count > holidayMaxOccurrences {
		count = holidayMaxOccurrences
	}
	interval := rule.interval
	if interval < 1 {
		interval = 1
	}
	duration := ev.end.Sub(ev.start)

	out := make([]rawHolidayEvent, 0, count)
	for i := 0; i < count; i++ {
		start := ev.start.AddDate(i*interval, 0, 0)
		if !rule.until.IsZero() && start.After(rule.until) {
			break
		}
		item := ev
		item.start = start
		item.end = start.Add(duration)
		if !item.end.After(item.start) {
			item.end = item.start.AddDate(0, 0, 1)
		}
		item.rrule = ""
		out = append(out, item)
	}
	return out
}

func buildHolidayDays(events []rawHolidayEvent) []HolidayDay {
	byDate := map[string]*holidayAgg{}

	for _, ev := range events {
		for _, occ := range expandHolidayOccurrences(ev) {
			guard := 0
			for cur := occ.start; cur.Before(occ.end) && guard < holidayMaxRangeDays; cur, guard = cur.AddDate(0, 0, 1), guard+1 {
				date := cur.Format("2006-01-02")
				agg := byDate[date]
				if agg == nil {
					agg = &holidayAgg{}
					byDate[date] = agg
				}
				if ev.rest {
					agg.rest = true
				}
				if ev.work {
					agg.work = true
				}
				agg.entries = append(agg.entries, holidayLabel{name: ev.name, kind: ev.kind})
			}
		}
	}

	return holidayDaysFromAgg(byDate)
}

func holidayDaysFromAgg(byDate map[string]*holidayAgg) []HolidayDay {
	dates := make([]string, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	days := make([]HolidayDay, 0, len(dates))
	for _, date := range dates {
		agg := byDate[date]
		sort.SliceStable(agg.entries, func(i, j int) bool {
			return holidayKindRank(agg.entries[i].kind) > holidayKindRank(agg.entries[j].kind)
		})

		day := HolidayDay{Date: date, Rest: agg.rest, Work: agg.work, Labels: make([]string, 0, len(agg.entries))}
		for _, entry := range agg.entries {
			if entry.name == "" || containsHolidayLabel(day.Labels, entry.name) {
				continue
			}
			day.Labels = append(day.Labels, entry.name)
		}
		if len(agg.entries) > 0 {
			day.Name = agg.entries[0].name
			day.Kind = agg.entries[0].kind
		}
		if day.Kind == "" {
			switch {
			case day.Rest:
				day.Kind = HolidayKindHoliday
			case day.Work:
				day.Kind = HolidayKindWorkday
			default:
				day.Kind = HolidayKindFestival
			}
		}
		if day.Name == "" && len(day.Labels) > 0 {
			day.Name = day.Labels[0]
		}
		days = append(days, day)
	}
	return days
}

func holidayKindRank(kind string) int {
	switch kind {
	case HolidayKindHoliday:
		return 4
	case HolidayKindWorkday:
		return 3
	case HolidayKindFestival:
		return 2
	case HolidayKindSolarTerm:
		return 1
	}
	return 0
}

func containsHolidayLabel(labels []string, name string) bool {
	for _, label := range labels {
		if label == name {
			return true
		}
	}
	return false
}
