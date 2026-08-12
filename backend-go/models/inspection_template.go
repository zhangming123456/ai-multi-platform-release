package models

import (
	"encoding/json"
	"time"
)

type ScoreOption struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

func MarshalScoreOptions(options []ScoreOption) string {
	if options == nil {
		return ""
	}
	b, _ := json.Marshal(options)
	return string(b)
}

func UnmarshalScoreOptions(raw string) []ScoreOption {
	if raw == "" {
		return nil
	}
	var options []ScoreOption
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return nil
	}
	return options
}

func DefaultScoreOptions(scoreType string, maxScore int) []ScoreOption {
	if scoreType == "pass_fail" {
		return []ScoreOption{{Score: 1, Label: "合格"}, {Score: 2, Label: "不合格"}}
	}
	return []ScoreOption{{Score: 0, Label: "0分"}, {Score: 2, Label: "2分"}, {Score: 5, Label: "5分"}}
}

type InspectionTemplate struct {
	ID          string    `orm:"column(id);pk;size(36)" json:"id"`
	Name        string    `orm:"column(name);size(200)" json:"name"`
	Description string    `orm:"column(description);type(text);null" json:"description"`
	IsActive    bool      `orm:"column(is_active);default(true)" json:"is_active"`
	ScoringMode string    `orm:"column(scoring_mode);size(20);default(additive)" json:"scoring_mode"`
	CreatedAt   time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt   time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (t *InspectionTemplate) TableName() string {
	return "inspection_templates"
}

type InspectionTemplateItem struct {
	ID            string    `orm:"column(id);pk;size(36)" json:"id"`
	TemplateID    string    `orm:"column(template_id);size(36)" json:"template_id"`
	Category      string    `orm:"column(category);size(100)" json:"category"`
	Title         string    `orm:"column(title);size(200)" json:"title"`
	Standard      string    `orm:"column(standard);type(text);null" json:"standard"`
	StandardImage string    `orm:"column(standard_image);size(500);null" json:"standard_image"`
	ScoreType     string    `orm:"column(score_type);size(20);default(score)" json:"score_type"`
	MaxScore      int       `orm:"column(max_score);default(0)" json:"max_score"`
	ScoreOptions  string    `orm:"column(score_options);type(text);null" json:"-"`
	RequireRemark bool      `orm:"column(require_remark);default(false)" json:"require_remark"`
	RequirePhoto  bool      `orm:"column(require_photo);default(false)" json:"require_photo"`
	// 字段显示开关：默认开启，关闭后巡店时该字段不展示
	ShowRemark    bool      `orm:"column(show_remark);default(true)" json:"show_remark"`
	ShowPhoto     bool      `orm:"column(show_photo);default(true)" json:"show_photo"`
	// 分类级前置条件：同分类下所有 item 共享同一值（存于每行做冗余，读取时取首条）
	CategoryPreconditionEnabled bool   `orm:"column(category_precondition_enabled);default(false)" json:"category_precondition_enabled"`
	CategoryPrecondition        string `orm:"column(category_precondition);size(50);null" json:"category_precondition"`
	SortOrder     int       `orm:"column(sort_order);default(0)" json:"sort_order"`
	IsActive      bool      `orm:"column(is_active);default(true)" json:"is_active"`
	CreatedAt     time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt     time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (i *InspectionTemplateItem) TableName() string {
	return "inspection_template_items"
}
