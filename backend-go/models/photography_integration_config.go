package models

import "time"

// PhotographyIntegrationConfig 保存摄影工具的第三方集成配置。
// 所有 *_encrypted 字段只保存 AES-GCM 密文，明文密钥不会写入数据库。
type PhotographyIntegrationConfig struct {
	ID                          string    `orm:"column(id);pk;size(50)" json:"id"`
	AMapWebKeyEncrypted         string    `orm:"column(amap_web_key_encrypted);type(text);null" json:"-"`
	AMapWebServiceKeyEncrypted  string    `orm:"column(amap_web_service_key_encrypted);type(text);null" json:"-"`
	AMapSecurityKeyEncrypted    string    `orm:"column(amap_security_key_encrypted);type(text);null" json:"-"`
	GoogleMapsKeyEncrypted      string    `orm:"column(google_maps_key_encrypted);type(text);null" json:"-"`
	OpenMeteoKeyEncrypted       string    `orm:"column(open_meteo_key_encrypted);type(text);null" json:"-"`
	QWeatherKeyEncrypted        string    `orm:"column(qweather_key_encrypted);type(text);null" json:"-"`
	QWeatherCredentialType      string    `orm:"column(qweather_credential_type);size(20);default(api_key)" json:"-"`
	ShenzhenWeatherKeyEncrypted string    `orm:"column(shenzhen_weather_key_encrypted);type(text);null" json:"-"`
	NOAAKeyEncrypted            string    `orm:"column(noaa_key_encrypted);type(text);null" json:"-"`
	TideKeyEncrypted            string    `orm:"column(tide_key_encrypted);type(text);null" json:"-"`
	OpenMeteoHost               string    `orm:"column(open_meteo_host);size(500);null" json:"open_meteo_host"`
	QWeatherHost                string    `orm:"column(qweather_host);size(500);null" json:"qweather_host"`
	CMAHost                     string    `orm:"column(cma_host);size(500);null" json:"cma_host"`
	ShenzhenWeatherHost         string    `orm:"column(shenzhen_weather_host);size(500);null" json:"shenzhen_weather_host"`
	NOAAHost                    string    `orm:"column(noaa_host);size(500);null" json:"noaa_host"`
	TideHost                    string    `orm:"column(tide_host);size(500);null" json:"tide_host"`
	OpenMeteoEnabled            bool      `orm:"column(open_meteo_enabled);default(true)" json:"open_meteo_enabled"`
	QWeatherEnabled             bool      `orm:"column(qweather_enabled);default(false)" json:"qweather_enabled"`
	CMAEnabled                  bool      `orm:"column(cma_enabled);default(true)" json:"cma_enabled"`
	ShenzhenWeatherEnabled      bool      `orm:"column(shenzhen_weather_enabled);default(true)" json:"shenzhen_weather_enabled"`
	NOAAEnabled                 bool      `orm:"column(noaa_enabled);default(true)" json:"noaa_enabled"`
	TideEnabled                 bool      `orm:"column(tide_enabled);default(true)" json:"tide_enabled"`
	UpdatedBy                   string    `orm:"column(updated_by);size(36);null" json:"updated_by"`
	CreatedAt                   time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt                   time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (c *PhotographyIntegrationConfig) TableName() string {
	return "photography_integration_configs"
}
