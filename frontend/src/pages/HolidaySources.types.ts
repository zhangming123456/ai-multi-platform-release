export type HolidayRefreshInterval = 'manual' | 'hourly' | 'daily' | 'weekly' | 'monthly' | 'yearly'

export type HolidaySourceStatus = 'pending' | 'success' | 'failed'

export interface HolidaySourceItem {
  id: string
  name: string
  color: string
  url: string
  enabled: boolean
  refresh_interval: HolidayRefreshInterval
  last_refreshed_at: string | null
  last_attempt_at: string | null
  last_status: HolidaySourceStatus
  last_error: string
  event_count: number
  created_at: string
  updated_at: string
}

export interface HolidayRefreshOption {
  value: HolidayRefreshInterval
  label: string
}

export interface HolidaySourceListResponse {
  items: HolidaySourceItem[]
  interval_options: HolidayRefreshOption[]
}

export interface HolidaySourceFormValue {
  name: string
  color: string
  url: string
  enabled: boolean
  refresh_interval: HolidayRefreshInterval
}

export interface HolidaySourceSavePayload {
  name: string
  color: string
  url: string
  enabled: boolean
  refresh_interval: HolidayRefreshInterval
}
