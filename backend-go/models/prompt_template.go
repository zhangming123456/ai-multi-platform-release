package models

import (
	"time"
)

const (
	PromptTemplateStatusActive   = "active"
	PromptTemplateStatusArchived = "archived"
)

type PromptTemplate struct {
	ID           string    `orm:"column(id);pk;size(36)" json:"id"`
	Name         string    `orm:"column(name);size(200)" json:"name"`
	Platform     string    `orm:"column(platform);size(50);index" json:"platform"`
	Role         string    `orm:"column(role);type(text);null" json:"role"`
	Rules        string    `orm:"column(rules);type(text);null" json:"rules"`
	OutputFormat string    `orm:"column(output_format);type(text);null" json:"output_format"`
	IsDefault    bool      `orm:"column(is_default);default(false)" json:"is_default"`
	Status       string    `orm:"column(status);size(20);default(active)" json:"status"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt    time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (p *PromptTemplate) TableName() string {
	return "prompt_templates"
}
