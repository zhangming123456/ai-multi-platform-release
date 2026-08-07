package models

import (
	"time"
)

const (
	PlatformWechatMP    = "wechat_mp"
	PlatformXiaohongshu = "xiaohongshu"
	PlatformDouyin      = "douyin"
	PlatformWechatVideo = "wechat_video"

	AccountStatusActive   = "active"
	AccountStatusInactive = "inactive"
	AccountStatusError    = "error"
)

type Account struct {
	ID             string     `orm:"column(id);pk;size(36)" json:"id"`
	UserID         string     `orm:"column(user_id);size(36);index" json:"user_id"`
	Platform       string     `orm:"column(platform);size(20)" json:"platform"`
	Nickname       string     `orm:"column(nickname);size(200)" json:"nickname"`
	AvatarURL      string     `orm:"column(avatar_url);size(500);null" json:"avatar_url"`
	Status         string     `orm:"column(status);size(20);default(active)" json:"status"`
	CookieData     string     `orm:"column(cookie_data);type(text);null" json:"cookie_data"`
	AccessToken    string     `orm:"column(access_token);type(text);null" json:"access_token"`
	TokenExpiresAt *time.Time `orm:"column(token_expires_at);type(datetime);null" json:"token_expires_at"`
	LastCheckAt    *time.Time `orm:"column(last_check_at);type(datetime);null" json:"last_check_at"`
	ErrorMessage   string     `orm:"column(error_message);type(text);null" json:"error_message"`
	CreatedAt      time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (a *Account) TableName() string {
	return "accounts"
}
