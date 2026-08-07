package models

import (
	"time"
)

const (
	UserRoleAdmin    = "admin"
	UserRoleManager  = "manager"
	UserRoleOperator = "operator"
	UserRoleReviewer = "reviewer"
)

type User struct {
	ID             string    `orm:"column(id);pk;size(36)" json:"id"`
	Username       string    `orm:"column(username);unique;size(100)" json:"username"`
	Email          string    `orm:"column(email);unique;size(255);null" json:"email"`
	HashedPassword string    `orm:"column(hashed_password);size(255)" json:"-"`
	Nickname       string    `orm:"column(nickname);size(100)" json:"nickname"`
	Role           string    `orm:"column(role);size(50);default(operator)" json:"role"`
	AvatarURL      string    `orm:"column(avatar_url);size(500);null" json:"avatar_url"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (u *User) TableName() string {
	return "users"
}

func (u *User) GetUserRoles() []string {
	switch u.Role {
	case UserRoleAdmin:
		return []string{"admin"}
	case UserRoleManager:
		return []string{"manager"}
	case UserRoleOperator:
		return []string{"operator"}
	case UserRoleReviewer:
		return []string{"reviewer"}
	default:
		return []string{u.Role}
	}
}
