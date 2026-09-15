export type PhotographyIntegrationSection = 'map' | 'weather' | 'tide' | 'aurora'

export interface PhotographyIntegrationProvider {
  id: string
  name: string
  api_key_masked: string
  api_host: string
  configured: boolean
  enabled: boolean
  requires_key: boolean
}

export interface PhotographyIntegrationConfig {
  amap_web_key_masked: string
  amap_security_key_masked: string
  google_maps_key_masked: string
  open_meteo_key_masked: string
  qweather_key_masked: string
  qweather_credential_type: 'api_key' | 'token'
  noaa_key_masked: string
  tide_key_masked: string
  open_meteo_host: string
  qweather_host: string
  noaa_host: string
  tide_host: string
  open_meteo_enabled: boolean
  qweather_enabled: boolean
  noaa_enabled: boolean
  tide_enabled: boolean
  weather_providers: PhotographyIntegrationProvider[]
  tide_providers: PhotographyIntegrationProvider[]
  aurora_providers: PhotographyIntegrationProvider[]
}

export interface PhotographyIntegrationForm {
  amap_web_key: string
  amap_security_key: string
  google_maps_key: string
  open_meteo_key: string
  qweather_key: string
  qweather_credential_type: 'api_key' | 'token'
  noaa_key: string
  tide_key: string
  open_meteo_host: string
  qweather_host: string
  noaa_host: string
  tide_host: string
  open_meteo_enabled: boolean
  qweather_enabled: boolean
  noaa_enabled: boolean
  tide_enabled: boolean
  clear_amap_web_key: boolean
  clear_amap_security_key: boolean
  clear_google_maps_key: boolean
  clear_open_meteo_key: boolean
  clear_qweather_key: boolean
  clear_noaa_key: boolean
  clear_tide_key: boolean
}

export interface PhotographyIntegrationPayload {
  section?: PhotographyIntegrationSection
  amap_web_key?: string
  amap_security_key?: string
  google_maps_key?: string
  open_meteo_key?: string
  qweather_key?: string
  qweather_credential_type?: 'api_key' | 'token'
  noaa_key?: string
  tide_key?: string
  open_meteo_host?: string
  qweather_host?: string
  noaa_host?: string
  tide_host?: string
  open_meteo_enabled?: boolean
  qweather_enabled?: boolean
  noaa_enabled?: boolean
  tide_enabled?: boolean
  clear_amap_web_key?: boolean
  clear_amap_security_key?: boolean
  clear_google_maps_key?: boolean
  clear_open_meteo_key?: boolean
  clear_qweather_key?: boolean
  clear_noaa_key?: boolean
  clear_tide_key?: boolean
}
