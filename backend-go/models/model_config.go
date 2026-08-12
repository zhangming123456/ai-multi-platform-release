package models

import (
	"time"
)

type ModelConfig struct {
	ID             string    `orm:"column(id);pk;size(64)" json:"id"`
	Name           string    `orm:"column(name);size(100)" json:"name"`
	DisplayName    string    `orm:"column(display_name);size(100)" json:"display_name"`
	Provider       string    `orm:"column(provider);size(20)" json:"provider"`
	Mode           string    `orm:"column(mode);size(20)" json:"mode"`
	APIFormat      string    `orm:"column(api_format);size(50);default(openai_chat)" json:"api_format"`
	APIKey         string    `orm:"column(api_key);size(500);null" json:"api_key"`
	BaseURL        string    `orm:"column(base_url);size(500);null" json:"base_url"`
	FullURL        bool      `orm:"column(full_url);default(false)" json:"full_url"`
	Model          string    `orm:"column(model);size(2000)" json:"model"`
	Multimodal     bool      `orm:"column(multimodal);default(false)" json:"multimodal"`
	ModelSeries    string    `orm:"column(model_series);size(50);default(default)" json:"model_series"`
	ContextInput   int       `orm:"column(context_input);default(128000)" json:"context_input"`
	ContextOutput  int       `orm:"column(context_output);default(4096)" json:"context_output"`
	ToolCallRounds int       `orm:"column(tool_call_rounds);default(200)" json:"tool_call_rounds"`
	Enabled        bool      `orm:"column(enabled);default(false)" json:"enabled"`
	MonthlyQuota   int       `orm:"column(monthly_quota);default(1000000)" json:"monthly_quota"`
	UsedTokens     int       `orm:"column(used_tokens);default(0)" json:"used_tokens"`
	SortOrder      int       `orm:"column(sort_order);default(0)" json:"sort_order"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt      time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (m *ModelConfig) TableName() string {
	return "model_configs"
}
