package models

import (
	"time"
)

type AIGenerationRecord struct {
	ID          string    `orm:"column(id);pk;size(36)" json:"id"`
	UserID      string    `orm:"column(user_id);size(36);index" json:"user_id"`
	Topic       string    `orm:"column(topic);size(500)" json:"topic"`
	Platform    string    `orm:"column(platform);size(20)" json:"platform"`
	PlanID      string    `orm:"column(plan_id);size(64);null" json:"plan_id"`
	Model       string    `orm:"column(model);size(100);null" json:"model"`
	Title       string    `orm:"column(title);size(500)" json:"title"`
	Body        string    `orm:"column(body);type(text)" json:"body"`
	Hashtags    string    `orm:"column(hashtags);type(text);null" json:"hashtags"`
	ContentForm string    `orm:"column(content_form);size(30);null;index" json:"content_form"`
	CampaignID  string    `orm:"column(campaign_id);size(36);null;index" json:"campaign_id"`
	CreatedAt   time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (a *AIGenerationRecord) TableName() string {
	return "ai_generation_records"
}
