export type PhotographyEstimateLevel = 'excellent' | 'good' | 'fair' | 'poor' | 'unavailable'
export type PhotographyConfidence = 'high' | 'medium' | 'low'

export interface PhotographyLocationResult {
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
  elevation: number | null
  map_provider: 'amap' | 'google'
}

export interface PhotographyTimeRange {
  start: string
  end: string
}
export interface PhotographyEstimate {
  available: boolean
  score: number | null
  probability: number | null
  level: PhotographyEstimateLevel
  reasons: string[]
  risks: string[]
  confidence: PhotographyConfidence
}
export interface PhotographyForecastHour {
  time: string
  temperature: number | null
  weather_code: number | null
  weather_text: string
}
export interface PhotographyWeatherToday {
  available: boolean
  temperature: number | null
  feels_like: number | null
  temperature_max: number | null
  temperature_min: number | null
  relative_humidity: number | null
  wind_speed: number | null
  wind_direction: number | null
  precipitation_probability: number | null
  precipitation: number | null
  visibility: number | null
  cloud_cover: number | null
  weather_code: number | null
  weather_description: string
  risk: string
}
export interface PhotographyTideEvent {
  time: string
  type: string
  height: number
}
export interface PhotographyTideSummary {
  available: boolean
  source: string
  station: string
  distance_km: number | null
  current_level: number | null
  next_high: PhotographyTideEvent | null
  next_low: PhotographyTideEvent | null
  events: PhotographyTideEvent[]
  curve: PhotographyTideEvent[]
  message: string
}
export interface PhotographyAuroraSummary {
  available: boolean
  kp: number | null
  magnetic_latitude: number | null
  visible_start: string
  visible_end: string
  score: number | null
  level: PhotographyEstimateLevel
  reasons: string[]
  risks: string[]
  confidence: PhotographyConfidence
  message: string
}
export interface PhotographyDay {
  date: string
  sunrise: string
  sunset: string
  golden_hours: { morning: PhotographyTimeRange | null; evening: PhotographyTimeRange | null }
  blue_hours: { morning: PhotographyTimeRange | null; evening: PhotographyTimeRange | null }
  sunrise_assessment: PhotographyEstimate
  sunset_assessment: PhotographyEstimate
  cloud_sea: PhotographyEstimate
  stargazing_index: number
  cloud_cover_quality: number | null
  cloud_cover: number | null
  precipitation_probability: number | null
  weather_code: number | null
  confidence: PhotographyConfidence
}
export interface PhotographyChartSeries {
  name: string
  values: number[]
  color: string
}
export interface PhotographyOverview {
  location: {
    name: string
    latitude: number
    longitude: number
    timezone: string
    country_code: string
    elevation: number | null
    map_provider: 'amap' | 'google'
  }
  today: {
    weather: PhotographyWeatherToday
    hours: PhotographyForecastHour[]
    rainbow: PhotographyEstimate
    frost_rime: PhotographyEstimate
    tide: PhotographyTideSummary
    aurora: PhotographyAuroraSummary
  }
  days: PhotographyDay[]
  charts: { probability: PhotographyChartSeries[]; cloud_quality: PhotographyChartSeries[] }
  sources: { weather: string; tide: string; aurora: string }
  warnings: string[]
}
export interface PhotographyMapConfig {
  provider: 'amap' | 'google'
  configured: boolean
  browser_key: string
  security_key: string
  message: string
}
/** 空间分布图（概率图 / 云量质量图的地图模式）单个网格。 */
export interface PhotographySpatialCell {
  latitude: number
  longitude: number
  cloud_cover: number | null
  precipitation_probability: number | null
  probability: number | null
  probability_level: PhotographyEstimateLevel | ''
  cloud_quality: number | null
  cloud_quality_level: PhotographyEstimateLevel | ''
}
export interface PhotographySpatialGrid {
  date: string
  period: 'sunrise' | 'sunset'
  rows: number
  cols: number
  step_latitude: number
  step_longitude: number
  latitude: number
  longitude: number
  timezone: string
  source: string
  source_name: string
  message: string
  cells: PhotographySpatialCell[]
}
export interface PhotographyWeatherSource {
  id: string
  name: string
  provider: string
  model: string
  description: string
  available: boolean
  max_days: number
}

type NullableArray<T> = T[] | null | undefined
export type PhotographyEstimatePayload = Partial<PhotographyEstimate> | null | undefined
export type PhotographyDayPayload = Partial<
  Omit<PhotographyDay, 'sunrise_assessment' | 'sunset_assessment' | 'cloud_sea'>
> & {
  sunrise_assessment?: PhotographyEstimatePayload
  sunset_assessment?: PhotographyEstimatePayload
  cloud_sea?: PhotographyEstimatePayload
}
export type PhotographyTidePayload = Partial<Omit<PhotographyTideSummary, 'events' | 'curve'>> & {
  events?: NullableArray<PhotographyTideEvent>
  curve?: NullableArray<PhotographyTideEvent>
}
export type PhotographyAuroraPayload = Partial<PhotographyAuroraSummary> | null | undefined
export interface PhotographyOverviewPayload {
  location?: Partial<PhotographyOverview['location']> | null
  today?: {
    weather?: Partial<PhotographyWeatherToday> | null
    hours?: NullableArray<PhotographyForecastHour>
    rainbow?: PhotographyEstimatePayload
    frost_rime?: PhotographyEstimatePayload
    tide?: PhotographyTidePayload | null
    aurora?: PhotographyAuroraPayload
  } | null
  days?: NullableArray<PhotographyDayPayload>
  charts?: {
    probability?: NullableArray<PhotographyChartSeries>
    cloud_quality?: NullableArray<PhotographyChartSeries>
  } | null
  sources?: Partial<PhotographyOverview['sources']> | null
  warnings?: NullableArray<string>
}

function normalizeEstimate(value: PhotographyEstimatePayload): PhotographyEstimate {
  return {
    available: value?.available ?? false,
    score: value?.score ?? null,
    probability: value?.probability ?? null,
    level: value?.level ?? 'unavailable',
    reasons: Array.isArray(value?.reasons) ? value.reasons : [],
    risks: Array.isArray(value?.risks) ? value.risks : [],
    confidence: value?.confidence ?? 'low',
  }
}

function normalizeTimeRange(
  value: PhotographyTimeRange | null | undefined,
): PhotographyTimeRange | null {
  if (!value?.start || !value.end) return null
  return { start: value.start, end: value.end }
}

function normalizeTide(value: PhotographyTidePayload | null | undefined): PhotographyTideSummary {
  return {
    available: value?.available ?? false,
    source: value?.source ?? '',
    station: value?.station ?? '',
    distance_km: value?.distance_km ?? null,
    current_level: value?.current_level ?? null,
    next_high: value?.next_high ?? null,
    next_low: value?.next_low ?? null,
    events: Array.isArray(value?.events) ? value.events : [],
    curve: Array.isArray(value?.curve) ? value.curve : [],
    message: value?.message ?? '',
  }
}

function normalizeAurora(value: PhotographyAuroraPayload): PhotographyAuroraSummary {
  return {
    available: value?.available ?? false,
    kp: value?.kp ?? null,
    magnetic_latitude: value?.magnetic_latitude ?? null,
    visible_start: value?.visible_start ?? '',
    visible_end: value?.visible_end ?? '',
    score: value?.score ?? null,
    level: value?.level ?? 'unavailable',
    reasons: Array.isArray(value?.reasons) ? value.reasons : [],
    risks: Array.isArray(value?.risks) ? value.risks : [],
    confidence: value?.confidence ?? 'low',
    message: value?.message ?? '',
  }
}

function normalizeDay(value: PhotographyDayPayload): PhotographyDay {
  return {
    date: value.date ?? '',
    sunrise: value.sunrise ?? '',
    sunset: value.sunset ?? '',
    golden_hours: {
      morning: normalizeTimeRange(value.golden_hours?.morning),
      evening: normalizeTimeRange(value.golden_hours?.evening),
    },
    blue_hours: {
      morning: normalizeTimeRange(value.blue_hours?.morning),
      evening: normalizeTimeRange(value.blue_hours?.evening),
    },
    sunrise_assessment: normalizeEstimate(value.sunrise_assessment),
    sunset_assessment: normalizeEstimate(value.sunset_assessment),
    cloud_sea: normalizeEstimate(value.cloud_sea),
    stargazing_index: value.stargazing_index ?? 0,
    cloud_cover_quality: value.cloud_cover_quality ?? null,
    cloud_cover: value.cloud_cover ?? null,
    precipitation_probability: value.precipitation_probability ?? null,
    weather_code: value.weather_code ?? null,
    confidence: value.confidence ?? 'low',
  }
}

export function normalizePhotographyOverview(
  payload: PhotographyOverviewPayload | null | undefined,
): PhotographyOverview {
  const location = payload?.location
  const today = payload?.today
  const weather = today?.weather
  return {
    location: {
      name: location?.name ?? '',
      latitude: location?.latitude ?? 0,
      longitude: location?.longitude ?? 0,
      timezone: location?.timezone ?? '',
      country_code: location?.country_code ?? '',
      elevation: location?.elevation ?? null,
      map_provider: location?.map_provider === 'google' ? 'google' : 'amap',
    },
    today: {
      weather: {
        available: weather?.available ?? false,
        temperature: weather?.temperature ?? null,
        feels_like: weather?.feels_like ?? null,
        temperature_max: weather?.temperature_max ?? null,
        temperature_min: weather?.temperature_min ?? null,
        relative_humidity: weather?.relative_humidity ?? null,
        wind_speed: weather?.wind_speed ?? null,
        wind_direction: weather?.wind_direction ?? null,
        precipitation_probability: weather?.precipitation_probability ?? null,
        precipitation: weather?.precipitation ?? null,
        visibility: weather?.visibility ?? null,
        cloud_cover: weather?.cloud_cover ?? null,
        weather_code: weather?.weather_code ?? null,
        weather_description: weather?.weather_description ?? '',
        risk: weather?.risk ?? '暂无数据',
      },
      hours: Array.isArray(today?.hours) ? today.hours : [],
      rainbow: normalizeEstimate(today?.rainbow),
      frost_rime: normalizeEstimate(today?.frost_rime),
      tide: normalizeTide(today?.tide),
      aurora: normalizeAurora(today?.aurora),
    },
    days: Array.isArray(payload?.days) ? payload.days.map(normalizeDay) : [],
    charts: {
      probability: Array.isArray(payload?.charts?.probability) ? payload.charts.probability : [],
      cloud_quality: Array.isArray(payload?.charts?.cloud_quality)
        ? payload.charts.cloud_quality
        : [],
    },
    sources: {
      weather: payload?.sources?.weather ?? '',
      tide: payload?.sources?.tide ?? '',
      aurora: payload?.sources?.aurora ?? '',
    },
    warnings: Array.isArray(payload?.warnings) ? payload.warnings : [],
  }
}
