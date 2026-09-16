<template>
  <div ref="containerRef" class="photography-map-panel"></div>
</template>

<script setup lang="ts">
/**
 * 地图容器组件：按高德官方「JS API 结合 Vue 使用」教程，把 JS API 的加载、创建与销毁
 * 收敛到 Vue 组件生命周期里（官方 @amap/amap-jsapi-loader + onMounted/onBeforeUnmount）。
 * https://lbs.amap.com/api/javascript-api-v2/guide/abc/amap-vue
 * Key / 安全密钥由后端下发，组件只负责渲染，不感知业务查询。
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  ensureAmapJS,
  isValidPhotographyCoordinate,
  type PhotographyMapConfig,
} from '@/utils/amapSearch'

interface AMapLngLat {
  getLng(): number
  getLat(): number
}
interface AMapClickEvent {
  lnglat: AMapLngLat
}
interface AMapMapInstance {
  on(event: string, handler: (event: AMapClickEvent) => void): void
  setCenter(center: [number, number]): void
  setZoom(zoom: number): void
  destroy(): void
}
interface AMapMarkerInstance {
  setPosition(position: [number, number]): void
  setMap(map: AMapMapInstance | null): void
}
interface AMapApi {
  Map: new (
    container: HTMLElement,
    options: { zoom: number; center: [number, number] },
  ) => AMapMapInstance
  Marker: new (options: { position: [number, number] }) => AMapMarkerInstance
}
interface GoogleLatLng {
  lat(): number
  lng(): number
}
interface GoogleMapClickEvent {
  latLng?: GoogleLatLng
}
interface GoogleMapInstance {
  setCenter(center: { lat: number; lng: number }): void
  setZoom(zoom: number): void
  addListener(event: string, handler: (event: GoogleMapClickEvent) => void): void
}
interface GoogleMarkerInstance {
  setPosition(position: { lat: number; lng: number }): void
  setMap(map: GoogleMapInstance | null): void
}
interface GoogleMapsApi {
  Map: new (
    container: HTMLElement,
    options: {
      center: { lat: number; lng: number }
      zoom: number
      mapTypeControl: boolean
      streetViewControl: boolean
      fullscreenControl: boolean
    },
  ) => GoogleMapInstance
  Marker: new (options: {
    position: { lat: number; lng: number }
    map: GoogleMapInstance
  }) => GoogleMarkerInstance
}
declare global {
  interface Window {
    google?: { maps?: GoogleMapsApi }
  }
}

const props = withDefaults(
  defineProps<{
    /** 地图服务配置，高德分支内部走官方 loader 加载 JS API 2.0。 */
    config: PhotographyMapConfig
    /** 地图中心与点标记位置，[经度, 纬度]。 */
    center: [number, number]
    zoom?: number
  }>(),
  { zoom: 14 },
)
const emit = defineEmits<{
  (event: 'ready'): void
  (event: 'pick', latitude: number, longitude: number): void
  (event: 'error', message: string): void
}>()

const containerRef = ref<HTMLElement | null>(null)
let amap: AMapMapInstance | null = null
let amapMarker: AMapMarkerInstance | null = null
let googleMap: GoogleMapInstance | null = null
let googleMarker: GoogleMarkerInstance | null = null
let googleScriptPromise: Promise<void> | null = null

function dispose() {
  amapMarker?.setMap(null)
  amap?.destroy()
  googleMarker?.setMap(null)
  amap = null
  amapMarker = null
  googleMap = null
  googleMarker = null
}

function hasValidCenter(): boolean {
  return isValidPhotographyCoordinate(props.center[1], props.center[0])
}

// 坐标变化只同步视图，不重建地图实例。
function syncPosition() {
  // 高德 Marker / Map 内部构造 LngLat，NaN 会抛 `Invalid Object`，直接跳过。
  if (!hasValidCenter()) return
  try {
    if (amap && amapMarker) {
      amap.setCenter(props.center)
      amap.setZoom(props.zoom)
      amapMarker.setPosition(props.center)
      return
    }
    if (googleMap && googleMarker) {
      const position = { lat: props.center[1], lng: props.center[0] }
      googleMap.setCenter(position)
      googleMap.setZoom(props.zoom)
      googleMarker.setPosition(position)
    }
  } catch (error) {
    emit('error', error instanceof Error ? error.message : '地图更新失败')
  }
}

function loadGoogleSdk(browserKey: string) {
  if (window.google?.maps) return Promise.resolve()
  if (!googleScriptPromise) {
    googleScriptPromise = new Promise<void>((resolve, reject) => {
      const script = document.createElement('script')
      script.async = true
      script.src = 'https://maps.googleapis.com/maps/api/js?key=' + encodeURIComponent(browserKey)
      script.onload = () =>
        window.google?.maps ? resolve() : reject(new Error('Google 地图 SDK 未正确初始化'))
      script.onerror = () => reject(new Error('Google 地图 SDK 加载失败'))
      document.head.appendChild(script)
    }).catch((error: Error) => {
      googleScriptPromise = null
      throw error
    })
  }
  return googleScriptPromise
}

async function initMap() {
  const container = containerRef.value
  if (!container) return
  try {
    if (!hasValidCenter()) {
      emit('error', '当前经纬度无效，无法加载地图')
      return
    }
    if (props.config.provider === 'amap') {
      await ensureAmapJS(props.config)
      const AMap = (window as unknown as { AMap?: AMapApi }).AMap
      if (!AMap) throw new Error('高德地图 SDK 不可用')
      amap = new AMap.Map(container, { zoom: props.zoom, center: props.center })
      amapMarker = new AMap.Marker({ position: props.center })
      amapMarker.setMap(amap)
      amap.on('click', (event) => emit('pick', event.lnglat.getLat(), event.lnglat.getLng()))
    } else {
      await loadGoogleSdk(props.config.browser_key)
      const maps = window.google?.maps
      if (!maps) throw new Error('Google 地图 SDK 不可用')
      const position = { lat: props.center[1], lng: props.center[0] }
      googleMap = new maps.Map(container, {
        center: position,
        zoom: props.zoom,
        mapTypeControl: false,
        streetViewControl: false,
        fullscreenControl: false,
      })
      googleMarker = new maps.Marker({ position, map: googleMap })
      googleMap.addListener('click', (event) => {
        const lat = event.latLng?.lat()
        const lng = event.latLng?.lng()
        if (lat !== undefined && lng !== undefined) emit('pick', lat, lng)
      })
    }
    emit('ready')
  } catch (error) {
    dispose()
    emit('error', error instanceof Error ? error.message : '地图加载失败')
  }
}

onMounted(() => void initMap())
onBeforeUnmount(dispose)
watch([() => props.center[0], () => props.center[1], () => props.zoom], syncPosition)
watch(
  () => [props.config.provider, props.config.browser_key],
  () => {
    dispose()
    void initMap()
  },
)
</script>

<style scoped>
.photography-map-panel {
  width: 100%;
  height: 100%;
}
</style>
