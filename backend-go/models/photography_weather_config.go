package models

import "time"

type PhotographyWeatherConfig struct {
	ID             string    `orm:"column(id);pk;size(50)" json:"id"`
	CredentialType string    `orm:"column(credential_type);size(20);default(api_key)" json:"credential_type"`
	APIKey         string    `orm:"column(api_key);size(500);null" json:"-"`
	APIHost        string    `orm:"column(api_host);size(500);null" json:"api_host"`
	Enabled        bool      `orm:"column(enabled);default(true)" json:"enabled"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (c *PhotographyWeatherConfig) TableName() string {
	return "photography_weather_configs"
}
