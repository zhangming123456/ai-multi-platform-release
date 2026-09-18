<template>
  <div class="photography-spatial-map" :style="{ height: mapHeight }">
    <div ref="containerRef" class="spatial-map-canvas"></div>
    <div v-if="currentZoom !== null" class="spatial-map-zoom" :title="zoomTitle">
      Zoom {{ currentZoom.toFixed(1) }}
    </div>
    <div v-if="errorMessage" class="spatial-map-error">{{ errorMessage }}</div>
  </div>
</template>

<script setup lang="ts">
/**
 * 空间分布图容器：与 PhotographyMapPanel 同样的「官方 loader + Vue 生命周期」方式，
 * 只是把标记换成网格色块，用于概率图 / 云量质量图的地图模式。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
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
  /** 指标数值（0-100），Loca 用它做热力权重与网格柱高度。 */
  value?: number | null
}

interface AmapRectangleInstance {
  setMap(map: AmapMapInstance | null): void
}
/** AMap.Bounds 实例，Rectangle 只接受这个类型（传数组会在内部 bounds.clone() 处报错）。 */
interface AmapBoundsInstance {
  className: string
}
interface AmapMapInstance {
  setCenter(center: [number, number]): void
  setZoom(zoom: number): void
  getZoom(): number
  on(event: string, handler: () => void): void
  destroy(): void
}
interface AmapMarkerInstance {
  setPosition(position: [number, number]): void
  setMap(map: AmapMapInstance | null): void
}
interface AmapApi {
  Map: new (
    container: HTMLElement,
    options: { zoom: number; center: [number, number]; zooms?: [number, number] },
  ) => AmapMapInstance
  Bounds: new (southWest: [number, number], northEast: [number, number]) => AmapBoundsInstance
  Marker: new (options: { position: [number, number] }) => AmapMarkerInstance
  Rectangle: new (options: {
    bounds: AmapBoundsInstance
    fillColor: string
    fillOpacity: number
    strokeWeight: number
    strokeColor: string
  }) => AmapRectangleInstance
}

/** Loca 数据可视化的最小接口子集，字段按官方 @amap/amap-loca-types 定义。 */
interface LocaGeoJSONSourceInstance {
  destroy?: () => void
}
interface LocaLayerInstance {
  setSource(source: LocaGeoJSONSourceInstance): void
  setStyle(style: Record<string, unknown>): void
  destroy(): void
}
interface LocaContainerInstance {
  add(layer: LocaLayerInstance): void
  destroy(): void
  animate: { start(): void; stop(): void }
}
interface LocaApi {
  Container: new (opts: { map: AmapMapInstance }) => LocaContainerInstance
  GeoJSONSource: new (opts: { data: unknown }) => LocaGeoJSONSourceInstance
  HeatMapLayer: new (opts: {
    zIndex?: number
    opacity?: number
    zooms?: [number, number]
  }) => LocaLayerInstance
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
  getZoom(): number
  addListener(event: string, handler: () => void): void
}
interface GoogleMapsApi {
  Map: new (
    container: HTMLElement,
    options: {
      center: { lat: number; lng: number }
      zoom: number
      minZoom?: number
      maxZoom?: number
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
  /** probability：概率图配色；cloud_quality：云量质量图配色。 */
  mode?: 'probability' | 'cloud_quality'
  height?: number | string
}>()

const emit = defineEmits<{ ready: []; error: [message: string] }>()

const containerRef = ref<HTMLDivElement | null>(null)
/** 地图缩放边界：最远只能缩到 8 级，再放大最多 18 级。 */
const minZoom = 8
const maxZoom = 18
const errorMessage = ref('')
/** 当前地图缩放级别，跟手缩放实时更新，用来核对网格步长与视域是否匹配。 */
const currentZoom = ref<number | null>(null)
const zoomTitle = computed(() => `当前缩放级别 ${currentZoom.value ?? '—'}`)
const mapHeight = computed(() =>
  typeof props.height === 'number' ? `${props.height}px` : props.height || '320px',
)
let amap: AmapMapInstance | null = null
let amapMarker: AmapMarkerInstance | null = null
let googleMap: GoogleMapInstance | null = null
let googleMarker: GoogleMarkerInstance | null = null
let overlays: (AmapRectangleInstance | GoogleRectangleInstance)[] = []
let googleScriptPromise: Promise<void> | null = null
let loca: LocaContainerInstance | null = null
let locaLayer: LocaLayerInstance | null = null
let locaSource: LocaGeoJSONSourceInstance | null = null
let locaScriptPromise: Promise<void> | null = null

function clearOverlays() {
  for (const overlay of overlays) overlay.setMap(null)
  overlays = []
}

function locaApi(): LocaApi | undefined {
  return (window as unknown as { Loca?: LocaApi }).Loca
}

/** Loca 是独立脚本（不走 AMapLoader），首次进入地图模式时按当前 Web Key 懒加载。 */
function loadLocaScript(browserKey: string): Promise<void> {
  if (locaApi()?.Container) return Promise.resolve()
  if (!locaScriptPromise) {
    locaScriptPromise = new Promise<void>((resolve, reject) => {
      const script = document.createElement('script')
      script.async = true
      script.src = `https://webapi.amap.com/loca?key=${encodeURIComponent(browserKey)}&v=2.0`
      script.onload = () => resolve()
      script.onerror = () => reject(new Error('高德 Loca 可视化 SDK 加载失败，已回退为网格色块'))
      document.head.appendChild(script)
    }).catch((error: Error) => {
      locaScriptPromise = null
      throw error
    })
  }
  return locaScriptPromise
}

/** 网格步长换算成米，Loca 用真实地理半径，缩放时格子会跟着变大变小。 */
function spatialGridRadiusMeters(): number {
  const latitudeMeters = Math.abs(props.stepLatitude) * 111320
  const longitudeMeters =
    Math.abs(props.stepLongitude) * 111320 * Math.cos((props.center[1] * Math.PI) / 180)
  return Math.max(40, Math.min(latitudeMeters, Math.abs(longitudeMeters)) / 2)
}

function spatialGeoJSONFeatures() {
  return props.cells
    .filter(
      (cell) =>
        isValidPhotographyCoordinate(cell.latitude, cell.longitude) &&
        typeof cell.value === 'number' &&
        Number.isFinite(cell.value),
    )
    .map((cell) => ({
      type: 'Feature' as const,
      properties: {
        value: typeof cell.value === 'number' && Number.isFinite(cell.value) ? cell.value : 0,
        color: cell.color,
        label: cell.label,
      },
      geometry: { type: 'Point' as const, coordinates: [cell.longitude, cell.latitude] },
    }))
}

interface LocaRenderFeature {
  properties?: Record<string, unknown>
}

function locaFeatureValue(_index: number, feature: LocaRenderFeature): number {
  const raw = feature?.properties?.value
  return typeof raw === 'number' && Number.isFinite(raw) ? raw : 0
}

/** 两张图都用 Loca 热力图表达空间强弱，仅色带不同。 */
function renderLocaLayer(Loca: LocaApi): boolean {
  if (!loca || !hasValidStep()) return false
  locaLayer?.destroy()
  locaSource?.destroy?.()
  locaLayer = null
  locaSource = null
  const features = spatialGeoJSONFeatures()
  if (!features.length) return false
  const source = new Loca.GeoJSONSource({ data: { type: 'FeatureCollection', features } })
  locaSource = source
  const radius = spatialGridRadiusMeters()

  const layer = new Loca.HeatMapLayer({ zIndex: 20, opacity: 0.9, zooms: [minZoom, maxZoom] })
  layer.setSource(source)
  layer.setStyle({
    radius,
    unit: 'meter',
    value: locaFeatureValue,
    gradient:
      props.mode === 'cloud_quality'
        ? { 0: '#F3E8FF', 0.25: '#DDD6FE', 0.5: '#A78BFA', 0.75: '#7C3AED', 1: '#4C1D95' }
        : { 0: '#EF4444', 0.35: '#F59E0B', 0.55: '#3B82F6', 0.8: '#22C55E', 1: '#16A34A' },
    opacity: props.mode === 'cloud_quality' ? [0.25, 0.8] : [0.35, 0.85],
    height: 0,
    min: 0,
    max: 100,
  })
  loca.add(layer)
  locaLayer = layer
  loca.animate.start()
  return true
}

function disposeLoca() {
  locaLayer?.destroy()
  locaSource?.destroy?.()
  loca?.animate.stop()
  loca?.destroy()
  locaLayer = null
  locaSource = null
  loca = null
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
      bounds: new AMap.Bounds(
        [cell.longitude - halfLng, cell.latitude - halfLat],
        [cell.longitude + halfLng, cell.latitude + halfLat],
      ),
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
      if (!AMap) return
      const Loca = locaApi()
      if (loca && Loca) {
        try {
          if (renderLocaLayer(Loca)) return
        } catch {
          // Loca 实例异常时回退色块，避免影响地图。
          disposeLoca()
        }
      }
      drawAmapCells(AMap)
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
  disposeLoca()
  amapMarker?.setMap(null)
  amap?.destroy()
  googleMarker?.setMap(null)
  currentZoom.value = null
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
      amap = new AMap.Map(container, {
        zoom: props.zoom,
        center: props.center,
        zooms: [minZoom, maxZoom],
      })
      amapMarker = new AMap.Marker({ position: props.center })
      amapMarker.setMap(amap)
      let locaReady = false
      try {
        await loadLocaScript(props.config.browser_key)
        const Loca = locaApi()
        if (Loca?.Container) {
          loca = new Loca.Container({ map: amap })
          locaReady = renderLocaLayer(Loca)
        }
      } catch {
        // Loca 加载失败不阻断地图：下面回退为 AMap.Rectangle 网格色块。
        disposeLoca()
      }
      if (!locaReady) drawAmapCells(AMap)
    } else {
      await loadGoogleSdk(props.config.browser_key)
      const maps = (window as unknown as { google?: { maps?: GoogleMapsApi } }).google?.maps
      if (!maps) throw new Error('Google 地图 SDK 不可用')
      const position = { lat: props.center[1], lng: props.center[0] }
      googleMap = new maps.Map(container, {
        center: position,
        zoom: props.zoom,
        minZoom,
        maxZoom,
        mapTypeControl: false,
        streetViewControl: false,
        fullscreenControl: false,
      })
      googleMarker = new maps.Marker({ position, map: googleMap })
      drawGoogleCells(maps)
    }
    bindZoomEvents()
    try {
      currentZoom.value = amap?.getZoom() ?? googleMap?.getZoom() ?? props.zoom
    } catch {
      currentZoom.value = props.zoom
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
      currentZoom.value = amap.getZoom()
    }
    if (googleMap) {
      const position = { lat: props.center[1], lng: props.center[0] }
      googleMap.setCenter(position)
      googleMap.setZoom(props.zoom)
      googleMarker?.setPosition(position)
      currentZoom.value = googleMap.getZoom()
    }
  } catch (error) {
    reportMapError(error)
  }
  redraw()
}

// 高德 / Google 的缩放事件名不同，统一在回调里读当前级别。
function bindZoomEvents() {
  if (amap) {
    const syncZoom = () => {
      try {
        currentZoom.value = amap?.getZoom() ?? null
      } catch {
        /* 地图销毁瞬间读不到级别，忽略 */
      }
    }
    for (const event of ['zoomchange', 'zoomend', 'moveend']) amap.on(event, syncZoom)
  }
  if (googleMap) {
    const syncZoom = () => {
      try {
        currentZoom.value = googleMap?.getZoom() ?? null
      } catch {
        /* 同上 */
      }
    }
    for (const event of ['zoom_changed', 'bounds_changed', 'idle'])
      googleMap.addListener(event, syncZoom)
  }
}

onMounted(() => void initMap())
onBeforeUnmount(disposeMaps)
// 网格范围变化会带来新的缩放级别，这里一起监听，避免地图停留在大范围或只剩一格。
watch(() => [props.center[0], props.center[1], props.zoom], syncPosition)
watch(() => props.cells, redraw, { deep: true })
watch(() => props.mode, redraw)
watch(
  () => [props.config.provider, props.config.browser_key],
  () => void initMap(),
)
</script>

<style scoped>
.photography-spatial-map {
  position: relative;
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
.spatial-map-zoom {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 8px;
  font-size: 11px;
  line-height: 18px;
  color: #1f2937;
  pointer-events: none;
  background: rgba(255, 255, 255, 0.82);
  border-radius: 10px;
}
</style>
