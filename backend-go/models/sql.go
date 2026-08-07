package models

import (
	"time"
)

const (
	SqlChangeTypeUpdate    = "update"
	SqlChangeTypeDelete    = "delete"
	SqlChangeTypeRowDelete = "row_delete"

	SqlChangeStatusPending       = "pending"
	SqlChangeStatusApproved      = "approved"
	SqlChangeStatusRejected      = "rejected"
	SqlChangeStatusExecuted      = "executed"
	SqlChangeStatusExecuteFailed = "execute_failed"
)

type SqlHistory struct {
	ID        string    `orm:"column(id);pk;size(36)" json:"id"`
	UserID    string    `orm:"column(user_id);size(36);index" json:"user_id"`
	SQLText   string    `orm:"column(sql_text);type(text)" json:"sql_text"`
	IsSuccess bool      `orm:"column(is_success);default(true)" json:"is_success"`
	Message   string    `orm:"column(message);size(500);default()" json:"message"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (s *SqlHistory) TableName() string {
	return "sql_history"
}

type SqlChangeRequest struct {
	ID                string    `orm:"column(id);pk;size(36)" json:"id"`
	RequesterID       string    `orm:"column(requester_id);size(36);index" json:"requester_id"`
	ChangeType        string    `orm:"column(change_type);size(20)" json:"change_type"`
	SQLText           string    `orm:"column(sql_text);type(text)" json:"sql_text"`
	Description       string    `orm:"column(description);size(500);null" json:"description"`
	Status            string    `orm:"column(status);size(20);default(pending)" json:"status"`
	Approvals         int       `orm:"column(approvals);default(0)" json:"approvals"`
	RequiredApprovals int       `orm:"column(required_approvals);default(2)" json:"required_approvals"`
	ApprovedBy        string    `orm:"column(approved_by);size(500);default()" json:"approved_by"`
	RejectReason      string    `orm:"column(reject_reason);size(500);null" json:"reject_reason"`
	ExecuteMessage    string    `orm:"column(execute_message);size(500);null" json:"execute_message"`
	CreatedAt         time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt         time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (s *SqlChangeRequest) TableName() string {
	return "sql_change_requests"
}

const (
	UserCreationStatusPending  = "pending"
	UserCreationStatusApproved = "approved"
	UserCreationStatusRejected = "rejected"
)

type UserCreationRequest struct {
	ID             string    `orm:"column(id);pk;size(36)" json:"id"`
	RequesterID    string    `orm:"column(requester_id);size(36);index" json:"requester_id"`
	Username       string    `orm:"column(username);size(100)" json:"username"`
	Email          string    `orm:"column(email);size(255);null" json:"email"`
	HashedPassword string    `orm:"column(hashed_password);size(255)" json:"-"`
	Nickname       string    `orm:"column(nickname);size(100)" json:"nickname"`
	Role           string    `orm:"column(role);size(50);default(operator)" json:"role"`
	AvatarURL      string    `orm:"column(avatar_url);size(500);null" json:"avatar_url"`
	Status         string    `orm:"column(status);size(20);default(pending);index" json:"status"`
	ReviewerID     string    `orm:"column(reviewer_id);size(36);null" json:"reviewer_id"`
	RejectReason   string    `orm:"column(reject_reason);size(500);null" json:"reject_reason"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (u *UserCreationRequest) TableName() string {
	return "user_creation_requests"
}
