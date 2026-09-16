import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface CurrentLocation {
  latitude: number
  longitude: number
  accuracy: number | null
  timestamp: number
}

export type LocationStatus = 'idle' | 'locating' | 'success' | 'error'

export const useLocationStore = defineStore('location', () => {
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

  function locate(force = false): Promise<CurrentLocation | null> {
    if (!force && current.value) return Promise.resolve(current.value)
    if (pending) return pending
    if (!navigator.geolocation) {
      status.value = 'error'
      error.value = '当前浏览器不支持定位'
      return Promise.resolve(null)
    }

    status.value = 'locating'
    error.value = ''
    pending = new Promise((resolve) => {
      navigator.geolocation.getCurrentPosition(
        (position) => {
          current.value = {
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
            accuracy: Number.isFinite(position.coords.accuracy) ? position.coords.accuracy : null,
            timestamp: position.timestamp,
          }
          status.value = 'success'
          pending = null
          resolve(current.value)
        },
        (positionError) => {
          status.value = 'error'
          error.value = locationErrorMessage(positionError)
          pending = null
          resolve(null)
        },
        { enableHighAccuracy: true, timeout: 10000, maximumAge: 300000 },
      )
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
