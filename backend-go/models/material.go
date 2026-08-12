package models

import (
	"time"
)

// Material 通用素材库（当前仅支持图片）
type Material struct {
	ID        string    `orm:"column(id);pk;size(36)" json:"id"`
	Name      string    `orm:"column(name);size(200)" json:"name"`
	URL       string    `orm:"column(url);size(500)" json:"url"`
	Type      string    `orm:"column(type);size(20);default(image)" json:"type"`
	Category  string    `orm:"column(category);size(100);null" json:"category"`
	IsActive  bool      `orm:"column(is_active);default(true)" json:"is_active"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (m *Material) TableName() string {
	return "materials"
}
