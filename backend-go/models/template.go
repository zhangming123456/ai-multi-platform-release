package models

import (
	"time"
)

type Template struct {
	ID           string    `orm:"column(id);pk;size(36)" json:"id"`
	Name         string    `orm:"column(name);size(200)" json:"name"`
	Platform     string    `orm:"column(platform);size(20)" json:"platform"`
	ThumbnailURL string    `orm:"column(thumbnail_url);size(500);null" json:"thumbnail_url"`
	Config       string    `orm:"column(config);type(text);null" json:"config"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt    time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (t *Template) TableName() string {
	return "templates"
}
