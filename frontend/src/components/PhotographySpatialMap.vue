<template>
  <div class="photography-spatial-map">
    <div ref="containerRef" class="spatial-map-canvas"></div>
    <div v-if="errorMessage" class="spatial-map-error">{{ errorMessage }}</div>
  </div>
</template>

<script setup lang="ts">
/**
 * 空间分布图容器：与 PhotographyMapPanel 同样的「官方 loader + Vue 生命周期」方式，
 * 只是把标记换成网格色块，用于概率图 / 云量质量图的地图模式。
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  ensureAmapJS,
  isValidPhotographyCoordinate,
  type PhotographyMapConfig,
} from '@/utils/amapSearch'

interface PhotographySpatialPaint {
  latitude: number
  longitude: number
  color: string
  label: string
}

interface AmapRectangleInstance {
  setMap(map: AmapMapInstance | null): void
}
interface AmapMapInstance {
  setCenter(center: [number, number]): void
  setZoom(zoom: number): void
  destroy(): void
}
interface AmapMarkerInstance {
  setPosition(position: [number, number]): void
  setMap(map: AmapMapInstance | null): void
}
interface AmapApi {
  Map: new (
    container: HTMLElement,
    options: { zoom: number; center: [number, number] },
  ) => AmapMapInstance
  Marker: new (options: { position: [number, number] }) => AmapMarkerInstance
  Rectangle: new (options: {
    bounds: [[number, number], [number, number]]
    fillColor: string
    fillOpacity: number
    strokeWeight: number
    strokeColor: string
  }) => AmapRectangleInstance
}
interface GoogleRectangleInstance {
  setMap(map: GoogleMapInstance | null): void
}
interface GoogleMarkerInstance {
  setPosition(position: { lat: number; lng: number }): void
  setMap(map: GoogleMapInstance | null): void
}
interface GoogleMapInstance {
  setCenter(center: { lat: number; lng: number }): void
  setZoom(zoom: number): void
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
  Rectangle: new (options: {
    bounds: { north: number; south: number; east: number; west: number }
    fillColor: string
    fillOpacity: number
    strokeWeight: number
    map: GoogleMapInstance
  }) => GoogleRectangleInstance
}

const props = defineProps<{
  config: PhotographyMapConfig
  /** [lng, lat]，与 AMap 一致 */
  center: [number, number]
  zoom: number
  stepLatitude: number
  stepLongitude: number
  cells: PhotographySpatialPaint[]
}>()

const emit = defineEmits<{ ready: []; error: [message: string] }>()

const containerRef = ref<HTMLDivElement | null>(null)
const errorMessage = ref('')
let amap: AmapMapInstance | null = null
let amapMarker: AmapMarkerInstance | null = null
let googleMap: GoogleMapInstance | null = null
let googleMarker: GoogleMarkerInstance | null = null
let overlays: (AmapRectangleInstance | GoogleRectangleInstance)[] = []
let googleScriptPromise: Promise<void> | null = null

function clearOverlays() {
  for (const overlay of overlays) overlay.setMap(null)
  overlays = []
}

/** 高德所有几何对象最终都会构造 LngLat，NaN 会抛 `Invalid Object`，统一在这里拦掉。 */
function hasValidCenter(): boolean {
  return isValidPhotographyCoordinate(props.center[1], props.center[0])
}

function hasValidStep(): boolean {
  return Number.isFinite(props.stepLatitude) && Number.isFinite(props.stepLongitude)
}

function reportMapError(error: unknown) {
  const message = error instanceof Error ? error.message : '地图渲染失败'
  errorMessage.value = message
  emit('error', message)
}

function drawAmapCells(AMap: AmapApi) {
  if (!amap || !hasValidStep()) return
  for (const cell of props.cells) {
    if (!isValidPhotographyCoordinate(cell.latitude, cell.longitude)) continue
    const halfLat = props.stepLatitude / 2
    const halfLng = props.stepLongitude / 2
    const rectangle = new AMap.Rectangle({
      bounds: [
        [cell.longitude - halfLng, cell.latitude - halfLat],
        [cell.longitude + halfLng, cell.latitude + halfLat],
      ],
      fillColor: cell.color,
      fillOpacity: 0.7,
      strokeWeight: 0,
      strokeColor: cell.color,
    })
    rectangle.setMap(amap)
    overlays.push(rectangle)
  }
}

function drawGoogleCells(maps: GoogleMapsApi) {
  if (!googleMap || !hasValidStep()) return
  for (const cell of props.cells) {
    if (!isValidPhotographyCoordinate(cell.latitude, cell.longitude)) continue
    const rectangle = new maps.Rectangle({
      bounds: {
        north: cell.latitude + props.stepLatitude / 2,
        south: cell.latitude - props.stepLatitude / 2,
        east: cell.longitude + props.stepLongitude / 2,
        west: cell.longitude - props.stepLongitude / 2,
      },
      fillColor: cell.color,
      fillOpacity: 0.7,
      strokeWeight: 0,
      map: googleMap,
    })
    overlays.push(rectangle)
  }
}

function redraw() {
  try {
    clearOverlays()
    if (props.config.provider === 'amap') {
      const AMap = (window as unknown as { AMap?: AmapApi }).AMap
      if (AMap) drawAmapCells(AMap)
      return
    }
    const maps = (window as unknown as { google?: { maps?: GoogleMapsApi } }).google?.maps
    if (maps) drawGoogleCells(maps)
  } catch (error) {
    reportMapError(error)
  }
}

function loadGoogleSdk(browserKey: string) {
  const existing = (window as unknown as { google?: { maps?: GoogleMapsApi } }).google?.maps
  if (existing) return Promise.resolve()
  if (!googleScriptPromise) {
    googleScriptPromise = new Promise<void>((resolve, reject) => {
      const script = document.createElement('script')
      script.async = true
      script.src = 'https://maps.googleapis.com/maps/api/js?key=' + encodeURIComponent(browserKey)
      script.onload = () => resolve()
      script.onerror = () => reject(new Error('Google 地图 SDK 加载失败'))
      document.head.appendChild(script)
    }).catch((error: Error) => {
      googleScriptPromise = null
      throw error
    })
  }
  return googleScriptPromise
}

function disposeMaps() {
  clearOverlays()
  amapMarker?.setMap(null)
  amap?.destroy()
  googleMarker?.setMap(null)
  amap = null
  amapMarker = null
  googleMap = null
  googleMarker = null
}

async function initMap() {
  const container = containerRef.value
  if (!container) return
  errorMessage.value = ''
  try {
    disposeMaps()
    if (!hasValidCenter()) {
      errorMessage.value = '当前经纬度无效，无法加载地图'
      emit('error', errorMessage.value)
      return
    }
    if (props.config.provider === 'amap') {
      await ensureAmapJS(props.config)
      const AMap = (window as unknown as { AMap?: AmapApi }).AMap
      if (!AMap) throw new Error('高德地图 SDK 不可用')
      amap = new AMap.Map(container, { zoom: props.zoom, center: props.center })
      amapMarker = new AMap.Marker({ position: props.center })
      amapMarker.setMap(amap)
      drawAmapCells(AMap)
    } else {
      await loadGoogleSdk(props.config.browser_key)
      const maps = (window as unknown as { google?: { maps?: GoogleMapsApi } }).google?.maps
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
      drawGoogleCells(maps)
    }
    emit('ready')
  } catch (error) {
    reportMapError(error)
  }
}

function syncPosition() {
  if (!hasValidCenter()) return
  try {
    if (amap) {
      amap.setCenter(props.center)
      amap.setZoom(props.zoom)
      amapMarker?.setPosition(props.center)
    }
    if (googleMap) {
      const position = { lat: props.center[1], lng: props.center[0] }
      googleMap.setCenter(position)
      googleMap.setZoom(props.zoom)
      googleMarker?.setPosition(position)
    }
  } catch (error) {
    reportMapError(error)
  }
  redraw()
}

onMounted(() => void initMap())
onBeforeUnmount(disposeMaps)
watch(() => [props.center[0], props.center[1]], syncPosition)
watch(() => props.cells, redraw, { deep: true })
watch(
  () => [props.config.provider, props.config.browser_key],
  () => void initMap(),
)
</script>

<style scoped>
.photography-spatial-map {
  position: relative;
  height: 320px;
  overflow: hidden;
  border-radius: 12px;
  background: #f5f5f7;
}
.spatial-map-canvas {
  position: absolute;
  inset: 0;
}
.spatial-map-error {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  font-size: 12px;
  color: #b45309;
  text-align: center;
  background: #fffbeb;
}
</style>
