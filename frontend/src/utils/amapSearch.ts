/**
 * 高德 JS API 2.0 共用入口：用官方 loader 按需加载 SDK，并用 AutoComplete 做地址/城市搜索。
 * 地图与搜索复用同一个 SDK 实例，Key 由后端 /photography-tools/map-config 下发。
 * 接入方式对齐官方「JS API 结合 Vue 使用」文档：
 * https://lbs.amap.com/api/javascript-api-v2/guide/abc/amap-vue
 */
import AMapLoader from '@amap/amap-jsapi-loader'

export interface PhotographyMapConfig {
  provider: 'amap' | 'google'
  configured: boolean
  browser_key: string
  security_key: string
  message: string
}

export interface AmapSearchTip {
  name: string
  display_name: string
  detail: string
  district: string
  city_code: string
  adcode: string
  latitude: number
  longitude: number
}

/** 与页面 PhotographyLocation 类型一致，避免两处重复映射。 */
export interface AmapLocationResult {
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
  map_provider: 'amap'
}

export interface AmapAddressResult {
  name: string
  display_name: string
  detail: string
  district: string
  city_code: string
  adcode: string
}

interface AmapLngLat {
  lng: number
  lat: number
}

interface AmapAutoCompleteTip {
  name?: unknown
  district?: unknown
  adcode?: unknown
  citycode?: unknown
  address?: unknown
  location?: AmapLngLat | null
}

interface AmapAutoCompleteInstance {
  search(
    keyword: string,
    callback: (status: string, result: { tips?: AmapAutoCompleteTip[] }) => void,
  ): void
}

interface AmapGeocoderInstance {
  getAddress(
    location: [number, number],
    callback: (
      status: string,
      result: {
        regeocode?: {
          formattedAddress?: unknown
          addressComponent?: Record<string, unknown>
        }
      },
    ) => void,
  ): void
}

interface AmapSdk {
  AutoComplete?: new (options: { city?: string }) => AmapAutoCompleteInstance
  Geocoder?: new (options: { extensions?: string }) => AmapGeocoderInstance
  convertFrom?: (
    location: [number, number],
    coordsys: string,
    callback: (status: string, result: { locations?: AmapLngLat[] }) => void,
  ) => void
}

export interface AmapLocatedPosition {
  latitude: number
  longitude: number
  accuracy: number | null
}

let amapSdkPromise: Promise<void> | null = null

export const AMAP_JS_VERSION = '2.0'
export const AMAP_JS_PLUGINS = ['AMap.AutoComplete', 'AMap.Geocoder']

/** 经纬度必须落在合法范围内，否则高德构造 LngLat 会抛 `Invalid Object`。 */
export function isValidPhotographyCoordinate(latitude: number, longitude: number): boolean {
  return (
    Number.isFinite(latitude) &&
    Number.isFinite(longitude) &&
    Math.abs(latitude) <= 90 &&
    Math.abs(longitude) <= 180
  )
}

function amapSdk(): AmapSdk | undefined {
  return (window as unknown as { AMap?: AmapSdk }).AMap
}

/** 官方要求安全密钥必须在 JS API 首次加载前挂到 window 上。 */
export function applyAmapSecurityConfig(securityKey: string) {
  if (!securityKey) return
  ;(window as unknown as { _AMapSecurityConfig?: { securityJsCode: string } })._AMapSecurityConfig =
    { securityJsCode: securityKey }
}

// 高德的 address / adcode / citycode 在不同结果里可能是字符串或数组，统一取文本。
function amapText(value: unknown): string {
  if (typeof value === 'string') return value.trim()
  if (Array.isArray(value)) {
    return value.map(amapText).filter(Boolean).join(' ')
  }
  if (value && typeof value === 'object') {
    return Object.values(value).map(amapText).filter(Boolean).join(' ')
  }
  return ''
}

export function ensureAmapJS(config: PhotographyMapConfig | null | undefined): Promise<void> {
  if (amapSdk()?.AutoComplete) return Promise.resolve()
  if (!config?.configured || config.provider !== 'amap' || !config.browser_key) {
    return Promise.reject(
      new Error('未配置高德 Web端(JS API) Key，请在「系统管理 → 气象数据源 → 地图服务」中配置'),
    )
  }
  if (!amapSdkPromise) {
    applyAmapSecurityConfig(config.security_key)
    amapSdkPromise = AMapLoader.load({
      key: config.browser_key,
      version: AMAP_JS_VERSION,
      plugins: AMAP_JS_PLUGINS,
    })
      .then(() => {
        if (!amapSdk()?.AutoComplete) throw new Error('高德地图 SDK 未正确初始化')
      })
      .catch((error: Error) => {
        amapSdkPromise = null
        throw error
      })
  }
  return amapSdkPromise
}

/**
 * WGS84（浏览器定位）→ GCJ-02（高德地图打点使用）。
 *
 * 不用高德 `AMap.Geolocation` 插件：它在 HTML5 定位回调里直接
 * `new LngLat(coords.longitude, coords.latitude)` 且不做数值校验，浏览器/系统返回异常坐标时
 * 会在插件内部抛 `Invalid Object: LngLat(lng, NaN)`，回调也随之不再触发（页面一直停在“定位中”）。
 * 这里由调用方自己取浏览器坐标并校验，只把坐标转换交给高德官方接口。
 * 转换失败（未配置地图服务、Key 无权限、网络异常）返回 null，调用方沿用原始坐标。
 */
export async function convertToGcj02ByAmapJS(
  latitude: number,
  longitude: number,
): Promise<AmapLocatedPosition | null> {
  const convertFrom = amapSdk()?.convertFrom
  if (typeof convertFrom !== 'function') return null
  if (!isValidPhotographyCoordinate(latitude, longitude)) return null
  return new Promise<AmapLocatedPosition | null>((resolve) => {
    let settled = false
    const finish = (value: AmapLocatedPosition | null) => {
      if (settled) return
      settled = true
      resolve(value)
    }
    // 高德接口异常时可能既不回调也不报错，超时兜底避免整个定位流程被卡住。
    const timer = window.setTimeout(() => finish(null), 3000)
    try {
      convertFrom([longitude, latitude], 'gps', (status, result) => {
        window.clearTimeout(timer)
        const first = status === 'complete' ? result?.locations?.[0] : undefined
        const convertedLatitude = Number(first?.lat)
        const convertedLongitude = Number(first?.lng)
        finish(
          isValidPhotographyCoordinate(convertedLatitude, convertedLongitude)
            ? { latitude: convertedLatitude, longitude: convertedLongitude, accuracy: null }
            : null,
        )
      })
    } catch {
      window.clearTimeout(timer)
      finish(null)
    }
  })
}

export async function searchLocationsByAmapJS(keyword: string): Promise<AmapSearchTip[]> {
  const AutoComplete = amapSdk()?.AutoComplete
  if (!AutoComplete) return []
  const tips = await new Promise<AmapAutoCompleteTip[]>((resolve) => {
    const autoComplete = new AutoComplete({})
    autoComplete.search(keyword, (status, result) => {
      resolve(status === 'complete' ? result?.tips || [] : [])
    })
  })

  const locations: AmapSearchTip[] = []
  for (const tip of tips) {
    const location = tip.location
    if (!location || !Number.isFinite(location.lat) || !Number.isFinite(location.lng)) continue
    const name = amapText(tip.name)
    const district = amapText(tip.district)
    locations.push({
      name,
      display_name: [name, district].filter(Boolean).join(' · '),
      detail: amapText(tip.address) || district || name,
      district,
      city_code: amapText(tip.citycode),
      adcode: amapText(tip.adcode),
      latitude: Number(location.lat),
      longitude: Number(location.lng),
    })
  }
  return locations
}

export function amapTipToLocation(tip: AmapSearchTip, timezone = ''): AmapLocationResult {
  return {
    name: tip.name,
    display_name: tip.display_name,
    detail: tip.detail,
    country: '',
    country_code: '',
    admin1: tip.district,
    city_code: tip.city_code,
    adcode: tip.adcode,
    latitude: tip.latitude,
    longitude: tip.longitude,
    timezone,
    elevation: null,
    map_provider: 'amap',
  }
}

/** 经纬度反向地理编码：用于当前定位、地图选点、手填坐标后补全地址信息。 */
export async function reverseGeocodeByAmapJS(
  latitude: number,
  longitude: number,
): Promise<AmapAddressResult | null> {
  const Geocoder = amapSdk()?.Geocoder
  // Geocoder 内部会构造 LngLat，非法坐标会直接抛 `Invalid Object`，这里先拦掉。
  if (!Geocoder || !isValidPhotographyCoordinate(latitude, longitude)) return null
  const regeocode = await new Promise<{
    formattedAddress?: unknown
    addressComponent?: Record<string, unknown>
  } | null>((resolve) => {
    try {
      const geocoder = new Geocoder({ extensions: 'base' })
      geocoder.getAddress([longitude, latitude], (status, result) => {
        resolve(status === 'complete' ? result?.regeocode || null : null)
      })
    } catch {
      resolve(null)
    }
  })
  if (!regeocode) return null
  const component = regeocode.addressComponent || {}
  const formatted = amapText(regeocode.formattedAddress)
  const district = ['province', 'city', 'district']
    .map((key) => amapText(component[key]))
    .filter(Boolean)
    .join('')
  const detail = formatted || district
  if (!detail) return null
  return {
    name: detail,
    display_name: detail,
    detail,
    district: district || detail,
    city_code: amapText(component.citycode),
    adcode: amapText(component.adcode),
  }
}
