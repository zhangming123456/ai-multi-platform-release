import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import api from '@/utils/api'
import { countryCodeForRegion } from '@/utils/time'
import { useRegionStore } from '@/stores/region'
import {
  convertToGcj02ByAmapJS,
  ensureAmapJS,
  isValidPhotographyCoordinate,
  type PhotographyMapConfig,
} from '@/utils/amapSearch'

export interface CurrentLocation {
  latitude: number
  longitude: number
  accuracy: number | null
  timestamp: number
}

export type LocationStatus = 'idle' | 'locating' | 'success' | 'error'

export const useLocationStore = defineStore('location', () => {
  const regionStore = useRegionStore()
  const current = ref<CurrentLocation | null>(null)
  const status = ref<LocationStatus>('idle')
  const error = ref('')
  let pending: Promise<CurrentLocation | null> | null = null

  const latitude = computed(() => current.value?.latitude ?? null)
  const longitude = computed(() => current.value?.longitude ?? null)
  const hasLocation = computed(() => current.value !== null)
  const isLocating = computed(() => status.value === 'locating')

  function locationErrorMessage(positionError: GeolocationPositionError) {
    if (positionError.code === positionError.PERMISSION_DENIED) {
      return '请允许浏览器访问当前位置'
    }
    if (positionError.code === positionError.POSITION_UNAVAILABLE) {
      return '暂时无法获取当前位置，请稍后重试'
    }
    if (positionError.code === positionError.TIMEOUT) {
      return '定位超时，请重试'
    }
    return positionError.message || '获取当前位置失败，请重试'
  }

  /**
   * 当前位置：浏览器 HTML5 定位（高德插件内部走的也是这条链路，但它不校验坐标，
   * 异常坐标会在插件内部抛 `Invalid Object: LngLat(lng, NaN)` 且回调不再触发）。
   */
  function browserLocateOnce(
    enableHighAccuracy: boolean,
  ): Promise<CurrentLocation | null | 'invalid'> {
    return new Promise((resolve) => {
      navigator.geolocation.getCurrentPosition(
        (position) => {
          const { latitude, longitude, accuracy } = position.coords
          if (!isValidPhotographyCoordinate(latitude, longitude)) {
            resolve('invalid')
            return
          }
          resolve({
            latitude,
            longitude,
            accuracy: Number.isFinite(accuracy) ? accuracy : null,
            timestamp: position.timestamp,
          })
        },
        (positionError) => {
          error.value = locationErrorMessage(positionError)
          resolve(null)
        },
        { enableHighAccuracy, timeout: 10000, maximumAge: 0 },
      )
    })
  }

  async function browserLocate(): Promise<CurrentLocation | null> {
    if (!navigator.geolocation) {
      error.value = '当前浏览器不支持定位'
      return null
    }
    const accurate = await browserLocateOnce(true)
    if (accurate && accurate !== 'invalid') return accurate
    // 系统高精度定位可能返回 NaN 纬度，降级到网络定位再试一次。
    const fallback = await browserLocateOnce(false)
    if (fallback && fallback !== 'invalid') return fallback
    if (fallback === 'invalid') {
      error.value = '当前位置坐标无效，请检查系统定位服务后重试'
    }
    return null
  }

  /**
   * 大陆及港澳台：浏览器拿到的是 WGS84，直接在高德地图上打点会偏移约 500 米，
   * 用高德地图服务转成 GCJ-02。地图服务不可用或转换失败时沿用原始坐标，不影响定位。
   */
  async function convertByMapService(located: CurrentLocation): Promise<CurrentLocation> {
    try {
      const response = await api.get<PhotographyMapConfig>('/photography-tools/map-config', {
        params: { country_code: 'CN' },
      })
      const config = response.data
      if (!config?.configured || config.provider !== 'amap') return located
      await ensureAmapJS(config)
      const converted = await convertToGcj02ByAmapJS(located.latitude, located.longitude)
      if (!converted) return located
      return { ...located, latitude: converted.latitude, longitude: converted.longitude }
    } catch {
      // 地图服务不可用（未配置 Key、SDK 加载失败）时保留浏览器原始坐标。
      return located
    }
  }

  async function resolveCurrentLocation(): Promise<CurrentLocation | null> {
    const located = await browserLocate()
    if (!located) return null
    if (countryCodeForRegion(regionStore.selectedTz.value) !== 'CN') return located
    return convertByMapService(located)
  }

  function locate(force = false): Promise<CurrentLocation | null> {
    if (!force && current.value) return Promise.resolve(current.value)
    if (pending) return pending
    status.value = 'locating'
    error.value = ''
    pending = resolveCurrentLocation()
      .then((location) => {
        if (!location) {
          status.value = 'error'
          if (!error.value) error.value = '获取当前位置失败，请重试'
          return null
        }
        current.value = location
        status.value = 'success'
        return location
      })
      .catch((locateError: Error) => {
        status.value = 'error'
        error.value = locateError.message || '获取当前位置失败，请重试'
        return null
      })
      .finally(() => {
        pending = null
      })
    return pending
  }

  function clear() {
    current.value = null
    status.value = 'idle'
    error.value = ''
    pending = null
  }

  return {
    current,
    status,
    error,
    latitude,
    longitude,
    hasLocation,
    isLocating,
    locate,
    clear,
  }
})
