/**
 * 高德 JS API 2.0 共用入口：按需加载 SDK，并用 AutoComplete 做地址/城市搜索。
 * 地图与搜索复用同一个 SDK 实例，Key 由后端 /photography-tools/map-config 下发。
 */
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
}

let amapSdkPromise: Promise<void> | null = null

function amapSdk(): AmapSdk | undefined {
  return (window as unknown as { AMap?: AmapSdk }).AMap
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
    if (config.security_key) {
      ;(
        window as unknown as { _AMapSecurityConfig?: { securityJsCode: string } }
      )._AMapSecurityConfig = { securityJsCode: config.security_key }
    }
    amapSdkPromise = new Promise<void>((resolve, reject) => {
      const script = document.createElement('script')
      script.async = true
      script.src =
        'https://webapi.amap.com/maps?v=2.0&plugin=AMap.AutoComplete,AMap.Geocoder&key=' +
        encodeURIComponent(config.browser_key)
      script.onload = () =>
        amapSdk()?.AutoComplete ? resolve() : reject(new Error('高德地图 SDK 未正确初始化'))
      script.onerror = () => reject(new Error('高德地图 SDK 加载失败'))
      document.head.appendChild(script)
    }).catch((error: Error) => {
      amapSdkPromise = null
      throw error
    })
  }
  return amapSdkPromise
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
  if (!Geocoder) return null
  const regeocode = await new Promise<{
    formattedAddress?: unknown
    addressComponent?: Record<string, unknown>
  } | null>((resolve) => {
    const geocoder = new Geocoder({ extensions: 'base' })
    geocoder.getAddress([longitude, latitude], (status, result) => {
      resolve(status === 'complete' ? result?.regeocode || null : null)
    })
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
