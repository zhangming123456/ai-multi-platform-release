package models

import (
	"time"
)

const (
	CampaignStatusActive   = "active"
	CampaignStatusArchived = "archived"
)

type Campaign struct {
	ID          string    `orm:"column(id);pk;size(36)" json:"id"`
	UserID      string    `orm:"column(user_id);size(36);index" json:"user_id"`
	Name        string    `orm:"column(name);size(200)" json:"name"`
	Description string    `orm:"column(description);type(text);null" json:"description"`
	MediaURLs   string    `orm:"column(media_urls);type(text);null" json:"media_urls"`
	Location    string    `orm:"column(location);size(200);null" json:"location"`
	Platforms   string    `orm:"column(platforms);size(100);null" json:"platforms"`
	Status      string    `orm:"column(status);size(20);default(active)" json:"status"`
	CreatedAt   time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt   time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (c *Campaign) TableName() string {
	return "campaigns"
}
