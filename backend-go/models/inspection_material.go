package models

import (
	"time"
)

// InspectionMaterial 巡店素材库：可复用的检查标准图片与检查项配置
type InspectionMaterial struct {
	ID             string    `orm:"column(id);pk;size(36)" json:"id"`
	Category       string    `orm:"column(category);size(100);null" json:"category"`
	Title          string    `orm:"column(title);size(200)" json:"title"`
	Standard       string    `orm:"column(standard);type(text);null" json:"standard"`
	StandardImage  string    `orm:"column(standard_image);size(500);null" json:"standard_image"`
	StandardImages string    `orm:"column(standard_images);type(text);null" json:"-"`
	ScoreType      string    `orm:"column(score_type);size(20);default(score)" json:"score_type"`
	MaxScore       int       `orm:"column(max_score);default(0)" json:"max_score"`
	ScoreOptions   string    `orm:"column(score_options);type(text);null" json:"-"`
	IsActive       bool      `orm:"column(is_active);default(true)" json:"is_active"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (m *InspectionMaterial) TableName() string {
	return "inspection_materials"
}
