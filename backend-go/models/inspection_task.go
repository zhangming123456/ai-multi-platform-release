package models

import (
	"time"
)

// 整改任务状态
const (
	TaskStatusPending      = "pending"       // 待整改
	TaskStatusRechecking   = "rechecking"    // 复核中（已提交整改，AI 复核中）
	TaskStatusRectified    = "rectified"     // AI 复核全部通过，自动闭环
	TaskStatusRectifying   = "rectifying"    // AI 复核存在未通过，打回继续整改
	TaskStatusManualReview = "manual_review" // 单项复核达上限，转人工审核
	TaskStatusConfirmed    = "confirmed"     // 人工确认整改到位，闭环
	TaskStatusRejected     = "rejected"      // 人工确认整改未到位，关闭
)

// 整改问题项状态
const (
	TaskItemStatusPending   = "pending"   // 待整改
	TaskItemStatusSubmitted = "submitted" // 已提交待复核
	TaskItemStatusFixed     = "fixed"     // AI 判定已修复
	TaskItemStatusNotFixed  = "not_fixed" // AI 判定未修复
	TaskItemStatusManual    = "manual"    // 复核达上限，转人工
	TaskItemStatusConfirmed = "confirmed" // 人工确认已修复
	TaskItemStatusRejected  = "rejected"  // 人工确认未修复
)

// 整改任务日志动作
const (
	TaskLogActionCreated       = "created"
	TaskLogActionSubmitted     = "submitted"
	TaskLogActionRecheckPassed = "recheck_passed"
	TaskLogActionRecheckFailed = "recheck_failed"
	TaskLogActionManualReview  = "manual_review"
	TaskLogActionConfirmed     = "confirmed"
	TaskLogActionRejected      = "rejected"
)

// MaxTaskItemRecheckCount 单项 AI 复核最大次数，达上限后转人工审核。
const MaxTaskItemRecheckCount = 2

// DefaultTaskDeadlineDays 整改任务默认期限（天）。
const DefaultTaskDeadlineDays = 3

type InspectionTask struct {
	ID              string    `orm:"column(id);pk;size(36)" json:"id"`
	InspectionID    string    `orm:"column(inspection_id);size(36);index" json:"inspection_id"`
	StoreID         string    `orm:"column(store_id);size(36);index" json:"store_id"`
	StoreName       string    `orm:"column(store_name);size(200)" json:"store_name"`
	Title           string    `orm:"column(title);size(200)" json:"title"`
	Status          string    `orm:"column(status);size(20);default(pending)" json:"status"`
	InspectorID     string    `orm:"column(inspector_id);size(36);null" json:"inspector_id"`
	InspectorName   string    `orm:"column(inspector_name);size(100);null" json:"inspector_name"`
	ResponsibleID   string    `orm:"column(responsible_id);size(36);null" json:"responsible_id"`
	ResponsibleName string    `orm:"column(responsible_name);size(100);null" json:"responsible_name"`
	Contact         string    `orm:"column(contact);size(100);null" json:"contact"`
	Phone           string    `orm:"column(phone);size(50);null" json:"phone"`
	Channel         string    `orm:"column(channel);size(20);default(internal)" json:"channel"`
	Deadline        time.Time `orm:"column(deadline);type(datetime);null" json:"deadline"`
	ClosedAt        time.Time `orm:"column(closed_at);type(datetime);null" json:"closed_at"`
	CreatedAt       time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt       time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (t *InspectionTask) TableName() string {
	return "inspection_tasks"
}

type InspectionTaskItem struct {
	ID             string    `orm:"column(id);pk;size(36)" json:"id"`
	TaskID         string    `orm:"column(task_id);size(36);index" json:"task_id"`
	InspectionID   string    `orm:"column(inspection_id);size(36);null" json:"inspection_id"`
	ScoreID        string    `orm:"column(score_id);size(36);null" json:"score_id"`
	ItemID         string    `orm:"column(item_id);size(36)" json:"item_id"`
	ItemName       string    `orm:"column(item_name);size(200)" json:"item_name"`
	Category       string    `orm:"column(category);size(100);null" json:"category"`
	Standard       string    `orm:"column(standard);type(text);null" json:"standard"`
	StandardImage  string    `orm:"column(standard_image);size(500);null" json:"-"`
	StandardImages string    `orm:"column(standard_images);type(text);null" json:"-"`
	ScoreType      string    `orm:"column(score_type);size(20);default(score)" json:"score_type"`
	MaxScore       int       `orm:"column(max_score);default(0)" json:"max_score"`
	Score          float64   `orm:"column(score);default(0)" json:"score"`
	Comment        string    `orm:"column(comment);type(text);null" json:"comment"`
	AISuggestion   string    `orm:"column(ai_suggestion);type(text);null" json:"ai_suggestion"`
	AIProblemDesc  string    `orm:"column(ai_problem_desc);type(text);null" json:"ai_problem_desc"`
	OriginalPhotos string    `orm:"column(original_photos);type(text);null" json:"-"`
	Status         string    `orm:"column(status);size(20);default(pending)" json:"status"`
	RectifyPhotos  string    `orm:"column(rectify_photos);type(text);null" json:"-"`
	RectifyComment string    `orm:"column(rectify_comment);type(text);null" json:"rectify_comment"`
	RecheckCount   int       `orm:"column(recheck_count);default(0)" json:"recheck_count"`
	AIResult       string    `orm:"column(ai_result);type(text);null" json:"-"`
	SubmittedAt    time.Time `orm:"column(submitted_at);type(datetime);null" json:"submitted_at"`
	RecheckedAt    time.Time `orm:"column(rechecked_at);type(datetime);null" json:"rechecked_at"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (t *InspectionTaskItem) TableName() string {
	return "inspection_task_items"
}

type InspectionTaskLog struct {
	ID           string    `orm:"column(id);pk;size(36)" json:"id"`
	TaskID       string    `orm:"column(task_id);size(36);index" json:"task_id"`
	ItemID       string    `orm:"column(item_id);size(36);null" json:"item_id"`
	Action       string    `orm:"column(action);size(30)" json:"action"`
	Content      string    `orm:"column(content);type(text);null" json:"content"`
	OperatorID   string    `orm:"column(operator_id);size(36);null" json:"operator_id"`
	OperatorName string    `orm:"column(operator_name);size(100);null" json:"operator_name"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (t *InspectionTaskLog) TableName() string {
	return "inspection_task_logs"
}
