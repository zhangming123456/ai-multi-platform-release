package models

import (
	"time"
)

type HolidaySource struct {
	ID              string     `orm:"column(id);pk;size(36)" json:"id"`
	Name            string     `orm:"column(name);size(100)" json:"name"`
	Color           string     `orm:"column(color);size(20);default(#007AFF)" json:"color"`
	URL             string     `orm:"column(url);size(500)" json:"url"`
	Enabled         bool       `orm:"column(enabled);default(true)" json:"enabled"`
	RefreshInterval string     `orm:"column(refresh_interval);size(20);default(manual)" json:"refresh_interval"`
	LastRefreshedAt *time.Time `orm:"column(last_refreshed_at);type(datetime);null" json:"last_refreshed_at"`
	LastAttemptAt   *time.Time `orm:"column(last_attempt_at);type(datetime);null" json:"last_attempt_at"`
	LastStatus      string     `orm:"column(last_status);size(20);default(pending);index" json:"last_status"`
	LastError       string     `orm:"column(last_error);size(500);null" json:"last_error"`
	EventCount      int        `orm:"column(event_count);default(0)" json:"event_count"`
	CreatedAt       time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt       time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (m *HolidaySource) TableName() string {
	return "holiday_sources"
}

type HolidayEvent struct {
	ID        string    `orm:"column(id);pk;size(36)" json:"id"`
	SourceID  string    `orm:"column(source_id);size(36);index" json:"source_id"`
	Date      string    `orm:"column(date);size(10);index" json:"date"`
	Name      string    `orm:"column(name);size(100)" json:"name"`
	Kind      string    `orm:"column(kind);size(20);index" json:"kind"`
	Rest      bool      `orm:"column(rest);default(false)" json:"rest"`
	Work      bool      `orm:"column(work);default(false)" json:"work"`
	Labels    string    `orm:"column(labels);type(text);null" json:"labels"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (m *HolidayEvent) TableName() string {
	return "holiday_events"
}
