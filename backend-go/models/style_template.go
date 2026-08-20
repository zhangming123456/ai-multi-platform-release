package models

import (
	"time"
)

const (
	StyleTemplateStatusActive   = "active"
	StyleTemplateStatusArchived = "archived"
)

type StyleTemplate struct {
	ID          string    `orm:"column(id);pk;size(36)" json:"id"`
	Name        string    `orm:"column(name);size(200)" json:"name"`
	Platform    string    `orm:"column(platform);size(50);index" json:"platform"`
	Description string    `orm:"column(description);type(text);null" json:"description"`
	Keywords    string    `orm:"column(keywords);type(text);null" json:"keywords"`
	Status      string    `orm:"column(status);size(20);default(active)" json:"status"`
	CreatedAt   time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt   time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (s *StyleTemplate) TableName() string {
	return "style_templates"
}
