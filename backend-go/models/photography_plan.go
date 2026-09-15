package models

import "time"

const (
	PhotographyPlanSessionSunrise = "sunrise"
	PhotographyPlanSessionSunset  = "sunset"
	PhotographyPlanSessionBoth    = "both"
)

type PhotographyPlan struct {
	ID               string     `orm:"column(id);pk;size(36)" json:"id"`
	UserID           string     `orm:"column(user_id);size(36);index" json:"user_id"`
	Name             string     `orm:"column(name);size(200)" json:"name"`
	Date             string     `orm:"column(date);size(10);index" json:"date"`
	Session          string     `orm:"column(session);size(20)" json:"session"`
	LocationName     string     `orm:"column(location_name);size(200)" json:"location_name"`
	Latitude         float64    `orm:"column(latitude)" json:"latitude"`
	Longitude        float64    `orm:"column(longitude)" json:"longitude"`
	Timezone         string     `orm:"column(timezone);size(100)" json:"timezone"`
	WeatherSource    string     `orm:"column(weather_source);size(50);default(open-meteo-best-match)" json:"weather_source"`
	Note             string     `orm:"column(note);type(text);null" json:"note"`
	ForecastSnapshot string     `orm:"column(forecast_snapshot);type(text);null" json:"-"`
	LastSyncedAt     *time.Time `orm:"column(last_synced_at);type(datetime);null" json:"last_synced_at"`
	SyncError        string     `orm:"column(sync_error);type(text);null" json:"sync_error"`
	CreatedAt        time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt        time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (p *PhotographyPlan) TableName() string {
	return "photography_plans"
}
