package models

import (
	"time"
)

type InspectionItem struct {
	ID        string    `orm:"column(id);pk;size(36)" json:"id"`
	Name      string    `orm:"column(name);size(200)" json:"name"`
	Category  string    `orm:"column(category);size(50);null" json:"category"`
	MaxScore  int       `orm:"column(max_score)" json:"max_score"`
	SortOrder int       `orm:"column(sort_order);default(0)" json:"sort_order"`
	IsActive  bool      `orm:"column(is_active);default(true)" json:"is_active"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (i *InspectionItem) TableName() string {
	return "inspection_items"
}
