package models

import (
	"time"
)

const (
	ContentStatusDraft         = "draft"
	ContentStatusReady         = "ready"
	ContentStatusPendingReview = "pending_review"
	ContentStatusRejected      = "rejected"
	ContentStatusPublished     = "published"
)

type Content struct {
	ID                string    `orm:"column(id);pk;size(36)" json:"id"`
	UserID            string    `orm:"column(user_id);size(36);index" json:"user_id"`
	Title             string    `orm:"column(title);size(500)" json:"title"`
	Body              string    `orm:"column(body);type(text)" json:"body"`
	Platform          string    `orm:"column(platform);size(20)" json:"platform"`
	Status            string    `orm:"column(status);size(20);default(draft)" json:"status"`
	MediaURLs         string    `orm:"column(media_urls);type(text);null" json:"media_urls"`
	AIGenerated       bool      `orm:"column(ai_generated);default(false)" json:"ai_generated"`
	OriginalContentID string    `orm:"column(original_content_id);size(36);null" json:"original_content_id"`
	CreatedAt         time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt         time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (c *Content) TableName() string {
	return "contents"
}
