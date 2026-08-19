package models

import (
	"time"
)

// 巡店记录状态
const (
	InspectionStatusDraft      = "draft"      // 草稿，可编辑
	InspectionStatusPending    = "pending"    // 待整改
	InspectionStatusRectifying = "rectifying" // 整改中
	InspectionStatusClosed     = "closed"     // 已闭环
)

// InspectionStatusEditable 返回巡店记录是否仍可编辑。
func InspectionStatusEditable(status string) bool {
	return status == InspectionStatusDraft || status == ""
}

type Inspection struct {
	ID           string    `orm:"column(id);pk;size(36)" json:"id"`
	StoreID      string    `orm:"column(store_id);size(36)" json:"store_id"`
	TemplateID   string    `orm:"column(template_id);size(36);null" json:"template_id"`
	TemplateName string    `orm:"column(template_name);size(200);null" json:"template_name"`
	Title        string    `orm:"column(title);size(200)" json:"title"`
	Status       string    `orm:"column(status);size(20);default(draft)" json:"status"`
	TotalScore   float64   `orm:"column(total_score);default(0)" json:"total_score"`
	Passed       bool      `orm:"column(passed);default(false)" json:"passed"`
	Issues       string    `orm:"column(issues);type(text);null" json:"issues"`
	Suggestion   string    `orm:"column(suggestion);type(text);null" json:"suggestion"`
	AISummary    string    `orm:"column(ai_summary);type(text);null" json:"-"`
	AIGenerated  bool      `orm:"column(ai_generated);default(false)" json:"ai_generated"`
	Photos       string    `orm:"column(photos);type(text);null" json:"photos"`
	InspectorID  string    `orm:"column(inspector_id);size(36)" json:"inspector_id"`
	CheckedAt    time.Time `orm:"column(checked_at);type(datetime)" json:"checked_at"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt    time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (i *Inspection) TableName() string {
	return "inspections"
}

type InspectionScore struct {
	ID             string    `orm:"column(id);pk;size(36)" json:"id"`
	InspectionID   string    `orm:"column(inspection_id);size(36)" json:"inspection_id"`
	ItemID         string    `orm:"column(item_id);size(36)" json:"item_id"`
	ItemName       string    `orm:"column(item_name);size(200)" json:"item_name"`
	Category       string    `orm:"column(category);size(50);null" json:"category"`
	Standard       string    `orm:"column(standard);type(text);null" json:"standard"`
	StandardImage  string    `orm:"column(standard_image);size(500);null" json:"-"`
	StandardImages string    `orm:"column(standard_images);type(text);null" json:"-"`
	ScoreType      string    `orm:"column(score_type);size(20);default(score)" json:"score_type"`
	MaxScore       int       `orm:"column(max_score)" json:"max_score"`
	ScoreOptions   string    `orm:"column(score_options);type(text);null" json:"-"`
	Score          float64   `orm:"column(score);default(0)" json:"score"`
	RequireRemark  bool      `orm:"column(require_remark);default(false)" json:"require_remark"`
	RequirePhoto   bool      `orm:"column(require_photo);default(false)" json:"require_photo"`
	ShowRemark     bool      `orm:"column(show_remark);default(true)" json:"show_remark"`
	ShowPhoto      bool      `orm:"column(show_photo);default(true)" json:"show_photo"`
	Comment        string    `orm:"column(comment);type(text);null" json:"comment"`
	AIGenerated    bool      `orm:"column(ai_generated);default(false)" json:"ai_generated"`
	AISuggestion   string    `orm:"column(ai_suggestion);type(text);null" json:"ai_suggestion"`
	Photos         string    `orm:"column(photos);type(text);null" json:"photos"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (s *InspectionScore) TableName() string {
	return "inspection_scores"
}
