package models

import (
	"time"
)

const (
	PublishTaskStatusPending    = "pending"
	PublishTaskStatusPublishing = "publishing"
	PublishTaskStatusPublished  = "published"
	PublishTaskStatusFailed     = "failed"
)

type PublishTask struct {
	ID           string     `orm:"column(id);pk;size(36)" json:"id"`
	ContentID    string     `orm:"column(content_id);size(36);index" json:"content_id"`
	AccountID    string     `orm:"column(account_id);size(36);index" json:"account_id"`
	Status       string     `orm:"column(status);size(20);default(pending)" json:"status"`
	ScheduledAt  *time.Time `orm:"column(scheduled_at);type(datetime);null" json:"scheduled_at"`
	PublishedAt  *time.Time `orm:"column(published_at);type(datetime);null" json:"published_at"`
	ErrorMessage string     `orm:"column(error_message);type(text);null" json:"error_message"`
	RetryCount   int        `orm:"column(retry_count);default(0)" json:"retry_count"`
	CreatedAt    time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt    time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (p *PublishTask) TableName() string {
	return "publish_tasks"
}
