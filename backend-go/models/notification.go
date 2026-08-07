package models

import (
	"time"
)

const (
	NotificationTypeReviewSubmit           = "review_submit"
	NotificationTypeReviewApproved         = "review_approved"
	NotificationTypeReviewRejected         = "review_rejected"
	NotificationTypeRoleUpdated            = "role_updated"
	NotificationTypeRolePermissionsUpdated = "role_permissions_updated"
)

type Notification struct {
	ID        string    `orm:"column(id);pk;size(36)" json:"id"`
	UserID    string    `orm:"column(user_id);size(36);index" json:"user_id"`
	Type      string    `orm:"column(type);size(30)" json:"type"`
	Title     string    `orm:"column(title);size(200)" json:"title"`
	Content   string    `orm:"column(content);type(text);null" json:"content"`
	RelatedID string    `orm:"column(related_id);size(36);null" json:"related_id"`
	IsRead    bool      `orm:"column(is_read);default(false)" json:"is_read"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (n *Notification) TableName() string {
	return "notifications"
}

type NotificationDict struct {
	ID        string    `orm:"column(id);pk;size(36)" json:"id"`
	Category  string    `orm:"column(category);size(20);index" json:"category"`
	GroupKey  string    `orm:"column(group_key);size(100);index" json:"group_key"`
	DictKey   string    `orm:"column(dict_key);size(200)" json:"dict_key"`
	DictValue string    `orm:"column(dict_value);size(200)" json:"dict_value"`
	IsActive  bool      `orm:"column(is_active);default(true)" json:"is_active"`
	IsPrivate bool      `orm:"column(is_private);default(false)" json:"is_private"`
	SortOrder int       `orm:"column(sort_order);default(0)" json:"sort_order"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (n *NotificationDict) TableName() string {
	return "notification_dict"
}
