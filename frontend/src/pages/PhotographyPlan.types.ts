export interface PhotographyWeatherSource {
  id: string
  name: string
  provider: string
  model: string
  description: string
  available: boolean
  max_days: number
}

export type PhotographySession = 'sunrise' | 'sunset' | 'both'
export type PhotographyAssessmentLevel = 'excellent' | 'good' | 'fair' | 'poor'

export interface PhotographyLocation {
  name: string
  display_name: string
  detail: string
  country: string
  country_code: string
  admin1: string
  city_code: string
  adcode: string
  latitude: number
  longitude: number
  timezone: string
}

export interface PhotographyForecastHour {
  time: string
  temperature: number | null
  weather_code: number | null
  weather_text: string
}

export interface PhotographyForecastDay {
  date: string
  sunrise: string
  sunset: string
  sunrise_score: number
  sunrise_level: PhotographyAssessmentLevel
  sunrise_reasons: string[]
  sunset_score: number
  sunset_level: PhotographyAssessmentLevel
  sunset_reasons: string[]
  cloud_cover: number | null
  precipitation_probability: number | null
  weather_code: number | null
  weather_text: string
  temperature_max: number | null
  temperature_min: number | null
  relative_humidity: number | null
  hours: PhotographyForecastHour[]
}

export interface PhotographyForecast {
  latitude: number
  longitude: number
  timezone: string
  source: string
  source_name: string
  fallback_reason?: string
  days: PhotographyForecastDay[]
}

export interface PhotographyForecastSnapshot {
  days: PhotographyForecastDay[]
}

export interface PhotographyPlan {
  id: string
  user_id: string
  name: string
  date: string
  session: PhotographySession
  location_name: string
  latitude: number
  longitude: number
  timezone: string
  weather_source: string
  weather_source_name: string
  note: string
  forecast_snapshot: PhotographyForecastSnapshot
  last_synced_at: string | null
  sync_error: string
  created_at: string
  updated_at: string
}

export interface PhotographyPlanPayload {
  name: string
  date: string
  session: PhotographySession
  location_name: string
  latitude: number
  longitude: number
  weather_source: string
  note: string
}

export interface PaginatedPhotographyPlans {
  items: PhotographyPlan[]
  total: number
  page: number
  page_size: number
}
