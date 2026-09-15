<template>
  <div class="page-main photography-weather-page">
    <PageHeader title="天气" subtitle="按地点查询今日气象与未来三天摄影指数">
      <template #actions>
        <a-button size="mini" :loading="loading" @click="runQuery"
          ><template #icon><IconRefresh :size="13" /></template>刷新</a-button
        >
        <a-button type="primary" size="mini" :loading="locating" @click="useCurrentLocation"
          ><template #icon><IconLocation :size="13" /></template>使用当前位置</a-button
        >
      </template>
    </PageHeader>
    <div class="flex-1 px-4 pb-8 md:px-6 lg:px-8">
      <a-alert type="info" show-icon class="mb-4"
        >彩虹、雾凇、极光、朝霞、晚霞和云海均为天气条件估算，不代表自然现象一定发生。天气数据由后端统一请求。</a-alert
      >
      <a-alert
        v-if="overview?.warnings?.some((warning) => warning.includes('已切换至'))"
        type="warning"
        show-icon
        class="mb-4"
      >
        {{ overview?.warnings?.find((warning) => warning.includes('已切换至')) }}
      </a-alert>
      <a-card :bordered="false" class="mb-4 shadow-sm">
        <template #title
          ><div class="flex items-center gap-2">
            <IconSearch class="text-[#007AFF]" />地点与数据源
          </div></template
        >
        <template #extra
          ><span class="text-xs text-[#86868B]">地图：{{ mapProviderLabel }}</span></template
        >
        <a-row :gutter="16">
          <a-col :xs="24" :lg="10">
            <a-form-item label="搜索地址或城市">
              <a-trigger :popup-visible="searchVisible && searchResults.length > 0">
                <template #content
                  ><div v-if="searchVisible && searchResults.length" class="search-popover">
                    <button
                      v-for="item in searchResults"
                      :key="item.display_name + item.latitude"
                      type="button"
                      class="search-result"
                      @click="selectLocation(item)"
                    >
                      <IconLocation class="mt-0.5 shrink-0 text-[#007AFF]" /><span class="min-w-0"
                        ><span class="block truncate text-[13px]">{{
                          item.display_name || item.name
                        }}</span
                        ><span class="block text-[11px] text-[#86868B]"
                          >{{ item.latitude.toFixed(4) }}, {{ item.longitude.toFixed(4) }} ·
                          {{ item.timezone || '自动时区' }}</span
                        ></span
                      >
                    </button>
                  </div></template
                >
                <a-input
                  v-model="searchText"
                  allow-clear
                  placeholder="搜索城市或地址，例如：杭州西湖"
                  @focus="searchVisible = searchResults.length > 0"
                  @clear="clearSearch"
                  ><template #suffix><IconSearch class="text-[#86868B]" /></template
                ></a-input>
              </a-trigger>
            </a-form-item>
            <div class="mb-3 flex flex-wrap items-center gap-2 text-xs text-[#5D5D63]">
              <a-tag color="arcoblue">{{ selectedLocationName || '自定义坐标' }}</a-tag
              ><span v-if="timezone">{{ timezone }}</span
              ><span v-if="countryCode">{{ countryCode }}</span
              ><span v-if="overview && overview.location.elevation !== null"
                >海拔 {{ overview.location.elevation.toFixed(0) }} m</span
              >
            </div>
            <a-row :gutter="8"
              ><a-col :span="12"
                ><a-form-item label="纬度"
                  ><a-input-number
                    v-model="latitude"
                    :min="-90"
                    :max="90"
                    :precision="6"
                    class="w-full"
                    @change="handleCoordinateInput" /></a-form-item></a-col
              ><a-col :span="12"
                ><a-form-item label="经度"
                  ><a-input-number
                    v-model="longitude"
                    :min="-180"
                    :max="180"
                    :precision="6"
                    class="w-full"
                    @change="handleCoordinateInput" /></a-form-item></a-col
            ></a-row>
            <div class="flex gap-2">
              <a-button type="primary" :loading="loading || locating" @click="runQuery">{{
                hasCoordinates() ? '查询' : '查询当前位置'
              }}</a-button
              ><a-button @click="resetLocation">清空地点</a-button>
            </div>
          </a-col>
          <a-col :xs="24" :lg="8"
            ><a-form-item label="天气数据源"
              ><a-select v-model="weatherSource" :loading="sourcesLoading"
                ><a-option
                  v-for="source in weatherSources"
                  :key="source.id"
                  :value="source.id"
                  :disabled="!source.available"
                  >{{ source.name }}<span v-if="!source.available">（未配置）</span></a-option
                ></a-select
              ></a-form-item
            ><a-form-item label="潮汐数据源"
              ><a-select v-model="tideSource"
                ><a-option value="noaa-coops">NOAA CO-OPS（美国海域）</a-option
                ><a-option value="stormglass">Stormglass（全球，需 Key）</a-option
                ><a-option value="none">不查询潮汐</a-option></a-select
              ></a-form-item
            >
            <div class="rounded-xl bg-[#F5F5F7] p-3 text-xs leading-relaxed text-[#5D5D63]">
              {{
                mapConfig?.message ||
                '默认使用高德地图；明确选择其他国家或地区时使用 Google 地图。地图 Key 未配置时仍可使用搜索和坐标输入。'
              }}
            </div></a-col
          >
          <a-col :xs="24" :lg="6">
            <div class="coordinate-map">
              <div
                ref="mapCanvas"
                class="map-canvas"
                :class="{ 'map-canvas-hidden': !mapReady }"
              ></div>
              <div v-if="!mapReady" class="map-fallback" @click="pickMapCoordinate">
                <div class="map-grid"></div>
                <div class="map-marker" :style="markerStyle"><IconLocation :size="22" /></div>
                <div class="map-label"><IconLocation :size="13" /> 点击地图区域选点</div>
              </div>
              <div v-else class="map-label map-sdk-label">
                <IconLocation :size="13" /> 点击地图选点
              </div>
            </div>
            <div v-if="mapError" class="mt-2 text-xs leading-relaxed text-[#B45309]">
              {{ mapError }}
            </div>
          </a-col>
        </a-row>
      </a-card>
      <a-spin :loading="loading" class="w-full">
        <template v-if="overview">
          <div
            class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-xl bg-white px-4 py-3 shadow-sm"
          >
            <div>
              <div class="text-[15px] font-semibold">{{ overview.location.name }}</div>
              <div class="mt-1 text-xs text-[#86868B]">
                {{ overview.location.latitude.toFixed(5) }},
                {{ overview.location.longitude.toFixed(5) }} · {{ overview.location.timezone }}
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-2 text-xs text-[#5D5D63]">
              <a-tag color="arcoblue">{{
                overview.location.map_provider === 'amap' ? '高德地图' : 'Google 地图'
              }}</a-tag
              ><span>天气：{{ overview.sources.weather }}</span
              ><span>潮汐：{{ overview.sources.tide }}</span
              ><span>{{ updatedAt }}</span>
            </div>
          </div>
          <a-alert
            v-for="warning in overview.warnings"
            :key="warning"
            type="warning"
            show-icon
            class="mb-2"
            >{{ warning }}</a-alert
          >
          <a-row :gutter="[16, 16]" class="mb-4">
            <a-col :xs="24" :lg="10"
              ><a-card :bordered="false" class="h-full shadow-sm"
                ><template #title>今日天气</template
                ><template #extra
                  ><a-tag color="green">{{
                    weather.weather_description || '天气'
                  }}</a-tag></template
                >
                <div class="flex items-end gap-3">
                  <div class="text-5xl font-bold">{{ formatNumber(weather.temperature) }}°</div>
                  <div class="pb-1 text-xs text-[#86868B]">
                    体感 {{ formatNumber(weather.feels_like) }}°<br />最高
                    {{ formatNumber(weather.temperature_max) }}° · 最低
                    {{ formatNumber(weather.temperature_min) }}°
                  </div>
                </div>
                <div class="mt-4 grid grid-cols-2 gap-2 text-xs">
                  <div class="metric-box">
                    湿度 <b>{{ formatPercent(weather.relative_humidity) }}</b>
                  </div>
                  <div class="metric-box">
                    云量 <b>{{ formatPercent(weather.cloud_cover) }}</b>
                  </div>
                  <div class="metric-box">
                    降水概率 <b>{{ formatPercent(weather.precipitation_probability) }}</b>
                  </div>
                  <div class="metric-box">
                    能见度 <b>{{ formatVisibility(weather.visibility) }}</b>
                  </div>
                  <div class="metric-box">
                    风速 <b>{{ formatNumber(weather.wind_speed) }} km/h</b>
                  </div>
                  <div class="metric-box">
                    风向 <b>{{ formatNumber(weather.wind_direction) }}°</b>
                  </div>
                </div>
                <div class="mt-4 rounded-xl bg-[#FFF8E8] px-3 py-2 text-xs text-[#8A6300]">
                  风险提示：{{ weather.risk }}
                </div></a-card
              ></a-col
            >
            <a-col :xs="24" :lg="14"
              ><div class="grid h-full grid-cols-1 gap-4 sm:grid-cols-2">
                <div v-for="item in todayPhenomena" :key="item.key" class="info-panel">
                  <div class="mb-2 flex items-center justify-between">
                    <span class="font-semibold">{{ item.icon }} {{ item.title }}</span
                    ><a-tag :color="scoreColor(item.estimate.level)"
                      >{{ item.estimate.probability ?? '—' }} ·
                      {{ levelLabel(item.estimate.level) }}</a-tag
                    >
                  </div>
                  <ul class="space-y-1 text-xs text-[#5D5D63]">
                    <li v-for="reason in item.estimate.reasons" :key="reason">· {{ reason }}</li>
                    <li v-for="risk in item.estimate.risks" :key="risk" class="text-[#B45309]">
                      · {{ risk }}
                    </li>
                  </ul>
                  <div class="mt-2 text-[11px] text-[#86868B]">
                    置信度：{{ item.estimate.confidence }}
                  </div>
                </div>
              </div></a-col
            >
          </a-row>
          <a-card :bordered="false" class="mb-4 shadow-sm">
            <template #title>
              <div class="flex items-center gap-2">
                <span>24 小时趋势</span>
                <span class="text-xs font-normal text-[#86868B]">当地时间 · 温度</span>
              </div>
            </template>
            <template #extra>
              <span class="text-xs text-[#86868B]">{{ todayHours.length }} 个时段</span>
            </template>
            <template v-if="todayHours.length">
              <div class="overflow-x-auto pb-2">
                <div class="flex min-w-[720px] gap-1">
                  <div
                    v-for="hour in todayHours"
                    :key="hour.time"
                    class="flex w-8 shrink-0 flex-col items-center gap-1 text-[10px] text-[#86868B]"
                    :title="hour.weather_text || weatherLabel(hour.weather_code)"
                  >
                    <span>{{ formatHour(hour.time) }}</span>
                    <span class="text-[18px] leading-none">{{
                      weatherIcon(hour.weather_code)
                    }}</span>
                    <span class="font-medium text-[#1D1D1F]">{{
                      formatTemperature(hour.temperature)
                    }}</span>
                  </div>
                </div>
              </div>
              <div class="mt-2 overflow-x-auto">
                <div ref="hourlyChartRef" class="h-64 w-full min-w-[720px]"></div>
              </div>
            </template>
            <a-empty v-else description="暂无 24 小时天气数据，请刷新天气" />
          </a-card>
          <a-card :bordered="false" class="mb-4 shadow-sm"
            ><template #title>今日潮汐与极光</template
            ><a-row :gutter="16"
              ><a-col :xs="24" :lg="12"
                ><div class="info-panel">
                  <div class="mb-3 flex items-center justify-between">
                    <b>潮汐</b
                    ><a-tag :color="overview.today.tide.available ? 'green' : 'orange'">{{
                      overview.today.tide.available ? '已获取' : '暂无数据'
                    }}</a-tag>
                  </div>
                  <div v-if="overview.today.tide.available" class="grid grid-cols-2 gap-3 text-xs">
                    <div>
                      当前潮位 <b>{{ formatNumber(overview.today.tide.current_level) }} m</b>
                    </div>
                    <div>
                      站点 <b>{{ overview.today.tide.station || '附近站点' }}</b>
                    </div>
                    <div>
                      下一高潮 <b>{{ formatEvent(overview.today.tide.next_high) }}</b>
                    </div>
                    <div>
                      下一低潮 <b>{{ formatEvent(overview.today.tide.next_low) }}</b>
                    </div>
                  </div>
                  <div v-else class="text-xs text-[#86868B]">
                    {{ overview.today.tide.message || '附近没有可用潮汐站点。' }}
                  </div>
                  <div v-if="tideCurve.length" class="mt-4 tide-line">
                    <span
                      v-for="(point, index) in tideCurve.slice(0, 12)"
                      :key="index"
                      :style="{ height: tideHeight(point.height) + '%' }"
                    ></span>
                  </div></div></a-col
              ><a-col :xs="24" :lg="12"
                ><div class="info-panel">
                  <div class="mb-3 flex items-center justify-between">
                    <b>极光</b
                    ><a-tag
                      :color="
                        overview.today.aurora.available
                          ? scoreColor(overview.today.aurora.level)
                          : 'orange'
                      "
                      >{{
                        overview.today.aurora.available
                          ? (overview.today.aurora.score ?? 0) + ' 分'
                          : '暂无数据'
                      }}</a-tag
                    >
                  </div>
                  <div
                    v-if="overview.today.aurora.available"
                    class="grid grid-cols-2 gap-3 text-xs"
                  >
                    <div>
                      Kp 指数 <b>{{ formatNumber(overview.today.aurora.kp) }}</b>
                    </div>
                    <div>
                      估算磁纬度 <b>{{ formatNumber(overview.today.aurora.magnetic_latitude) }}°</b>
                    </div>
                    <div class="col-span-2">
                      建议 <b>{{ levelLabel(overview.today.aurora.level) }}</b>
                    </div>
                  </div>
                  <ul class="mt-3 space-y-1 text-xs text-[#5D5D63]">
                    <li v-for="reason in overview.today.aurora.reasons" :key="reason">
                      · {{ reason }}
                    </li>
                    <li
                      v-for="risk in overview.today.aurora.risks"
                      :key="risk"
                      class="text-[#B45309]"
                    >
                      · {{ risk }}
                    </li>
                  </ul>
                  <div class="mt-3 text-xs text-[#86868B]">
                    {{ overview.today.aurora.message || 'NOAA SWPC 空间天气数据' }}
                  </div>
                </div></a-col
              ></a-row
            ></a-card
          >
          <a-card :bordered="false" class="mb-4 shadow-sm"
            ><template #title>未来三天摄影条件</template
            ><template #extra
              ><span class="text-xs text-[#86868B]">概率为天气条件估算</span></template
            >
            <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
              <div v-for="(day, index) in overview.days" :key="day.date" class="day-card">
                <div class="mb-3 flex items-center justify-between">
                  <b>{{ dayLabel(day.date, index) }}</b
                  ><span class="text-xs text-[#86868B]">{{ day.date }}</span>
                </div>
                <div class="mb-3 text-xs text-[#5D5D63]">
                  日出 {{ formatTime(day.sunrise) }} · 日落 {{ formatTime(day.sunset) }}
                </div>
                <div class="score-row">
                  <span>朝霞</span
                  ><a-tag :color="scoreColor(day.sunrise_assessment.level)"
                    >{{ day.sunrise_assessment.probability ?? '—' }} ·
                    {{ levelLabel(day.sunrise_assessment.level) }}</a-tag
                  >
                </div>
                <div class="score-row">
                  <span>晚霞</span
                  ><a-tag :color="scoreColor(day.sunset_assessment.level)"
                    >{{ day.sunset_assessment.probability ?? '—' }} ·
                    {{ levelLabel(day.sunset_assessment.level) }}</a-tag
                  >
                </div>
                <div class="score-row">
                  <span>云海</span
                  ><a-tag :color="scoreColor(day.cloud_sea.level)"
                    >{{ day.cloud_sea.probability ?? '—' }} ·
                    {{ levelLabel(day.cloud_sea.level) }}</a-tag
                  >
                </div>
                <div class="mt-3 grid grid-cols-2 gap-2 text-[11px] text-[#5D5D63]">
                  <span>金色 {{ formatRange(day.golden_hours.morning) }}</span
                  ><span>蓝色 {{ formatRange(day.blue_hours.evening) }}</span
                  ><span>云量质量 {{ day.cloud_cover_quality ?? '—' }}</span
                  ><span>观星 {{ day.stargazing_index }}</span>
                </div>
              </div>
            </div></a-card
          >
          <a-row :gutter="16" class="mb-4"
            ><a-col :xs="24" :lg="12"
              ><a-card :bordered="false" class="shadow-sm"
                ><template #title>概率图</template>
                <div class="chart-bars">
                  <div
                    v-for="series in overview.charts.probability"
                    :key="series.name"
                    class="chart-series"
                  >
                    <span class="chart-name" :style="{ color: series.color }">{{
                      series.name
                    }}</span>
                    <div class="chart-columns">
                      <div
                        v-for="(value, index) in series.values"
                        :key="index"
                        class="chart-column"
                      >
                        <span
                          class="chart-bar"
                          :style="{ height: value + '%', background: series.color }"
                        ></span
                        ><small>{{ value }}</small>
                      </div>
                    </div>
                  </div>
                </div></a-card
              ></a-col
            ><a-col :xs="24" :lg="12"
              ><a-card :bordered="false" class="shadow-sm"
                ><template #title>云量质量图</template>
                <div class="chart-bars">
                  <div
                    v-for="series in overview.charts.cloud_quality"
                    :key="series.name"
                    class="chart-series"
                  >
                    <span class="chart-name" :style="{ color: series.color }">{{
                      series.name
                    }}</span>
                    <div class="chart-columns">
                      <div
                        v-for="(value, index) in series.values"
                        :key="index"
                        class="chart-column"
                      >
                        <span
                          class="chart-bar"
                          :style="{ height: value + '%', background: series.color }"
                        ></span
                        ><small>{{ value }}</small>
                      </div>
                    </div>
                  </div>
                </div></a-card
              ></a-col
            ></a-row
          >
          <a-card :bordered="false" class="shadow-sm"
            ><template #title>估算说明</template>
            <div
              class="grid grid-cols-1 gap-3 text-xs leading-relaxed text-[#5D5D63] md:grid-cols-2"
            >
              <div>
                评分综合云量、降水概率、天气码、湿度、露点差、能见度和风速；金色/蓝色时刻使用太阳事件的近似窗口。
              </div>
              <div>
                观星指数未接入精细光污染和月亮模型，极光使用 NOAA SWPC Kp
                与当地黑夜条件估算。数据来源：{{ overview.sources.weather }}、{{
                  overview.sources.tide
                }}、{{ overview.sources.aurora }}。
              </div>
            </div></a-card
          >
        </template>
        <a-empty v-else description="使用当前位置或搜索一个地点开始查询" class="my-16" />
      </a-spin>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { Message } from '@arco-design/web-vue'
import { IconLocation, IconRefresh, IconSearch } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api, { WEATHER_API_TIMEOUT, getApiErrorDetail } from '@/utils/api'
import { countryCodeForRegion } from '@/utils/time'
import { normalizePhotographyOverview } from './PhotographyWeatherOverview.types'
import { useRegionStore } from '@/stores/region'
import type {
  PhotographyForecastHour,
  PhotographyLocationResult,
  PhotographyMapConfig,
  PhotographyOverview,
  PhotographyWeatherSource,
} from './PhotographyWeatherOverview.types'

interface AMapLngLat {
  getLng(): number
  getLat(): number
}
interface AMapClickEvent {
  lnglat: AMapLngLat
}
interface AMapInstance {
  on(event: string, handler: (event: AMapClickEvent) => void): void
  setCenter(center: [number, number]): void
  setZoom(zoom: number): void
  destroy?(): void
}
interface AMapMarkerInstance {
  setPosition(position: [number, number]): void
  setMap(map: AMapInstance | null): void
}
interface AMapApi {
  Map: new (
    container: HTMLElement,
    options: { zoom: number; center: [number, number] },
  ) => AMapInstance
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
    AMap?: AMapApi
    google?: { maps?: GoogleMapsApi }
    _AMapSecurityConfig?: { securityJsCode: string }
  }
}

const mapScriptPromises: Partial<Record<'amap' | 'google', Promise<void>>> = {}
const { selectedTz } = useRegionStore()
const overview = ref<PhotographyOverview | null>(null)
const weatherSources = ref<PhotographyWeatherSource[]>([])
const weatherSource = ref('open-meteo-best-match')
const tideSource = ref('noaa-coops')
const latitude = ref<number>()
const longitude = ref<number>()
const searchText = ref('')
const searchResults = ref<PhotographyLocationResult[]>([])
const searchVisible = ref(false)
const selectedLocationName = ref('')
const countryCode = ref('')
const locationCountryCodeLocked = ref(false)
const timezone = ref('')
const mapConfig = ref<PhotographyMapConfig | null>(null)
const mapCanvas = ref<HTMLElement | null>(null)
const loading = ref(false)
const locating = ref(false)
const sourcesLoading = ref(false)
const mapReady = ref(false)
const mapError = ref('')
const updatedAt = ref('—')
let searchTimer: number | undefined
let amapInstance: AMapInstance | null = null
let amapMarker: AMapMarkerInstance | null = null
let googleMapInstance: GoogleMapInstance | null = null
let googleMarker: GoogleMarkerInstance | null = null
let activeMapProvider: 'amap' | 'google' | null = null
const hourlyChartRef = ref<HTMLDivElement | null>(null)
let hourlyChart: echarts.ECharts | null = null

const weather = computed(
  () =>
    overview.value?.today.weather ?? {
      available: false,
      temperature: null,
      feels_like: null,
      temperature_max: null,
      temperature_min: null,
      relative_humidity: null,
      wind_speed: null,
      wind_direction: null,
      precipitation_probability: null,
      precipitation: null,
      visibility: null,
      cloud_cover: null,
      weather_code: null,
      weather_description: '',
      risk: '暂无数据',
    },
)
const todayHours = computed<PhotographyForecastHour[]>(() => overview.value?.today.hours ?? [])
const tideCurve = computed(() => overview.value?.today.tide.curve ?? [])
const defaultCountryCode = computed(() => countryCodeForRegion(selectedTz.value))
const mapCountryCode = computed(() =>
  locationCountryCodeLocked.value ? countryCode.value : defaultCountryCode.value,
)
const mapProviderLabel = computed(() =>
  (mapConfig.value?.provider || overview.value?.location.map_provider) === 'google'
    ? 'Google 地图'
    : '高德地图',
)
const todayPhenomena = computed(() =>
  overview.value
    ? [
        { key: 'rainbow', title: '彩虹（今）', icon: '🌈', estimate: overview.value.today.rainbow },
        {
          key: 'frost',
          title: '雾凇（今）',
          icon: '❄️',
          estimate: overview.value.today.frost_rime,
        },
      ]
    : [],
)
const markerStyle = computed(() => {
  const lat = latitude.value || 0
  const lon = longitude.value || 0
  return {
    left: Math.min(94, Math.max(6, ((lon + 180) / 360) * 100)) + '%',
    top: Math.min(90, Math.max(10, ((90 - lat) / 180) * 100)) + '%',
  }
})

function formatNumber(value: number | null | undefined) {
  return value === null || value === undefined || Number.isNaN(value)
    ? '—'
    : Number(value).toFixed(1)
}
function formatPercent(value: number | null | undefined) {
  return value === null || value === undefined ? '—' : Number(value).toFixed(0) + '%'
}
function formatVisibility(value: number | null | undefined) {
  return value === null || value === undefined ? '—' : (value / 1000).toFixed(1) + ' km'
}
function formatTime(value: string) {
  return value ? (value.includes('T') ? value.split('T')[1].slice(0, 5) : value.slice(0, 5)) : '—'
}
function formatHour(value: string) {
  return formatTime(value)
}
function formatTemperature(value: number | null | undefined) {
  return value === null || value === undefined ? '—' : Math.round(value) + '°'
}
function weatherIcon(code: number | null | undefined) {
  if (code === null || code === undefined) return '—'
  if (code === 0) return '☀️'
  if (code === 1 || code === 2) return '🌤️'
  if (code === 3) return '☁️'
  if (code === 45 || code === 48) return '🌫️'
  if (code >= 51 && code <= 67) return '🌧️'
  if (code >= 71 && code <= 77) return '🌨️'
  if (code >= 80 && code <= 86) return '🌦️'
  if (code >= 95) return '⛈️'
  return '🌥️'
}
function weatherLabel(code: number | null | undefined) {
  if (code === null || code === undefined) return '暂无数据'
  if (code === 0) return '晴朗'
  if (code === 1 || code === 2) return '少云'
  if (code === 3) return '阴天'
  if (code === 45 || code === 48) return '雾'
  if (code >= 51 && code <= 67) return '降雨'
  if (code >= 71 && code <= 77) return '降雪'
  if (code >= 80 && code <= 86) return '阵雨或阵雪'
  if (code >= 95) return '雷暴'
  return '多云'
}
function formatRange(value: { start: string; end: string } | null) {
  return value ? value.start + '-' + value.end : '无'
}
function formatEvent(value: { time: string; height: number } | null) {
  return value ? formatTime(value.time) + ' (' + value.height.toFixed(2) + 'm)' : '—'
}
function dayLabel(_date: string, index: number) {
  return ['今天', '明天', '后天'][index] || _date
}
function levelLabel(level: string) {
  return (
    (
      { excellent: '优', good: '良', fair: '一般', poor: '不建议', unavailable: '暂无' } as Record<
        string,
        string
      >
    )[level] || '暂无'
  )
}
function scoreColor(level: string) {
  return (
    (
      {
        excellent: 'green',
        good: 'arcoblue',
        fair: 'orange',
        poor: 'red',
        unavailable: 'gray',
      } as Record<string, string>
    )[level] || 'gray'
  )
}
function tideHeight(value: number) {
  return Math.min(100, Math.max(10, 50 + value * 20))
}

async function loadSources() {
  sourcesLoading.value = true
  try {
    const response = await api.get<PhotographyWeatherSource[]>('/photography-plans/weather-sources')
    weatherSources.value = response.data || []
  } catch (error) {
    Message.warning(getApiErrorDetail(error) || '天气数据源列表加载失败')
  } finally {
    sourcesLoading.value = false
  }
}
function currentMapCenter() {
  return { lat: latitude.value || 0, lng: longitude.value || 0 }
}
function hasCoordinates() {
  return latitude.value !== undefined && longitude.value !== undefined
}
function disposeMap() {
  if (amapMarker) amapMarker.setMap(null)
  amapInstance?.destroy?.()
  googleMarker?.setMap(null)
  amapInstance = null
  amapMarker = null
  googleMapInstance = null
  googleMarker = null
  activeMapProvider = null
  mapReady.value = false
  if (mapCanvas.value) mapCanvas.value.innerHTML = ''
}
function applyMapCoordinate(lat: number, lon: number) {
  latitude.value = Number(lat.toFixed(6))
  longitude.value = Number(lon.toFixed(6))
  selectedLocationName.value = '地图选点'
  countryCode.value = ''
  locationCountryCodeLocked.value = false
  timezone.value = ''
  syncMapPosition()
  void runQuery()
}
function syncMapPosition() {
  if (!mapReady.value) return
  const center = currentMapCenter()
  if (activeMapProvider === 'amap' && amapInstance && amapMarker) {
    amapInstance.setCenter([center.lng, center.lat])
    amapInstance.setZoom(14)
    amapMarker.setPosition([center.lng, center.lat])
  } else if (activeMapProvider === 'google' && googleMapInstance && googleMarker) {
    const position = { lat: center.lat, lng: center.lng }
    googleMapInstance.setCenter(position)
    googleMapInstance.setZoom(14)
    googleMarker.setPosition(position)
  }
}
async function loadMapSdk(config: PhotographyMapConfig) {
  if (!config.configured) {
    disposeMap()
    return
  }
  const provider = config.provider
  if (!mapScriptPromises[provider]) {
    mapScriptPromises[provider] = new Promise<void>((resolve, reject) => {
      if (provider === 'amap' && window.AMap) {
        resolve()
        return
      }
      if (provider === 'google' && window.google?.maps) {
        resolve()
        return
      }
      if (provider === 'amap' && config.security_key)
        window._AMapSecurityConfig = { securityJsCode: config.security_key }
      const script = document.createElement('script')
      script.async = true
      script.dataset.photographyMap = provider
      script.src =
        provider === 'amap'
          ? 'https://webapi.amap.com/maps?v=2.0&key=' + encodeURIComponent(config.browser_key)
          : 'https://maps.googleapis.com/maps/api/js?key=' + encodeURIComponent(config.browser_key)
      script.onload = () => {
        if ((provider === 'amap' && window.AMap) || (provider === 'google' && window.google?.maps))
          resolve()
        else reject(new Error((provider === 'amap' ? '高德' : 'Google') + ' 地图 SDK 未正确初始化'))
      }
      script.onerror = () =>
        reject(new Error((provider === 'amap' ? '高德' : 'Google') + ' 地图 SDK 加载失败'))
      document.head.appendChild(script)
    })
  }
  await mapScriptPromises[provider]
}
async function initializeMap(config: PhotographyMapConfig) {
  await nextTick()
  if (!mapCanvas.value || !config.configured) return
  if (activeMapProvider === config.provider && mapReady.value) {
    syncMapPosition()
    return
  }
  disposeMap()
  const center = currentMapCenter()
  if (config.provider === 'amap') {
    const AMap = window.AMap
    if (!AMap) throw new Error('高德地图 SDK 不可用')
    amapInstance = new AMap.Map(mapCanvas.value, { zoom: 14, center: [center.lng, center.lat] })
    amapMarker = new AMap.Marker({ position: [center.lng, center.lat] })
    amapMarker.setMap(amapInstance)
    amapInstance.on('click', (event) =>
      applyMapCoordinate(event.lnglat.getLat(), event.lnglat.getLng()),
    )
  } else {
    const maps = window.google?.maps
    if (!maps) throw new Error('Google 地图 SDK 不可用')
    googleMapInstance = new maps.Map(mapCanvas.value, {
      center,
      zoom: 14,
      mapTypeControl: false,
      streetViewControl: false,
      fullscreenControl: false,
    })
    googleMarker = new maps.Marker({ position: center, map: googleMapInstance })
    googleMapInstance.addListener('click', (event) => {
      const lat = event.latLng?.lat()
      const lon = event.latLng?.lng()
      if (lat !== undefined && lon !== undefined) applyMapCoordinate(lat, lon)
    })
  }
  activeMapProvider = config.provider
  mapReady.value = true
}
async function loadMapConfig() {
  try {
    const response = await api.get<PhotographyMapConfig>('/photography-tools/map-config', {
      params: { country_code: mapCountryCode.value },
    })
    mapConfig.value = response.data || null
    mapError.value = ''
    if (!mapConfig.value?.configured) {
      disposeMap()
      return
    }
    await loadMapSdk(mapConfig.value)
    await initializeMap(mapConfig.value)
  } catch (error) {
    disposeMap()
    mapError.value = getApiErrorDetail(error) || '地图服务不可用，仍可使用地址搜索或经纬度输入'
  }
}
async function runQuery() {
  if (!hasCoordinates()) {
    if (!locating.value) await useCurrentLocation()
    return
  }
  loading.value = true
  void loadMapConfig()
  try {
    const response = await api.get<PhotographyOverview>('/photography-tools/overview', {
      params: {
        latitude: latitude.value,
        longitude: longitude.value,
        weather_source: weatherSource.value,
        tide_source: tideSource.value,
        location_name: selectedLocationName.value,
        country_code: mapCountryCode.value,
      },
      timeout: WEATHER_API_TIMEOUT,
    })
    overview.value = normalizePhotographyOverview(response.data)
    timezone.value = overview.value.location.timezone
    countryCode.value = overview.value.location.country_code
    updatedAt.value = new Date().toLocaleString('zh-CN', { hour12: false })
    await renderHourlyChart()
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '天气服务暂不可用，已保留上一次结果')
  } finally {
    loading.value = false
  }
}
function getCurrentPosition(): Promise<GeolocationPosition> {
  return new Promise((resolve, reject) => {
    navigator.geolocation.getCurrentPosition(resolve, reject, {
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 300000,
    })
  })
}
async function useCurrentLocation() {
  if (!navigator.geolocation) {
    Message.error('当前浏览器不支持定位，请搜索地址或填写经纬度')
    return
  }
  if (locating.value) return
  locating.value = true
  try {
    const position = await getCurrentPosition()
    latitude.value = Number(position.coords.latitude.toFixed(6))
    longitude.value = Number(position.coords.longitude.toFixed(6))
    selectedLocationName.value = '当前位置'
    countryCode.value = ''
    locationCountryCodeLocked.value = false
    timezone.value = ''
    searchText.value = ''
    syncMapPosition()
    await runQuery()
  } catch (error) {
    const positionError = error as GeolocationPositionError
    const message =
      positionError?.code === 1
        ? '无法获取当前位置，请允许浏览器定位权限'
        : positionError?.message || '暂时无法获取当前位置，请稍后重试'
    Message.error(message)
  } finally {
    locating.value = false
  }
}
function handleCoordinateInput() {
  selectedLocationName.value = '自定义坐标'
  countryCode.value = ''
  locationCountryCodeLocked.value = false
  timezone.value = ''
  syncMapPosition()
}
function clearSearch() {
  searchVisible.value = false
  searchResults.value = []
}
function resetLocation() {
  disposeMap()
  latitude.value = undefined
  longitude.value = undefined
  selectedLocationName.value = ''
  countryCode.value = ''
  locationCountryCodeLocked.value = false
  timezone.value = ''
  overview.value = null
  mapConfig.value = null
  mapError.value = ''
  clearSearch()
}
function selectLocation(item: PhotographyLocationResult) {
  latitude.value = item.latitude
  longitude.value = item.longitude
  selectedLocationName.value = item.display_name || item.name
  countryCode.value = item.country_code || ''
  locationCountryCodeLocked.value = Boolean(item.country_code)
  timezone.value = item.timezone || ''
  searchText.value = item.display_name || item.name
  searchResults.value = []
  searchVisible.value = false
  syncMapPosition()
  void runQuery()
}
function pickMapCoordinate(event: MouseEvent) {
  if (!(event.currentTarget instanceof HTMLElement)) return
  const rect = event.currentTarget.getBoundingClientRect()
  const x = Math.min(1, Math.max(0, (event.clientX - rect.left) / rect.width))
  const y = Math.min(1, Math.max(0, (event.clientY - rect.top) / rect.height))
  applyMapCoordinate(90 - y * 180, -180 + x * 360)
}
function searchLocations() {
  if (searchTimer) window.clearTimeout(searchTimer)
  if (searchText.value.trim().length < 2) {
    searchResults.value = []
    return
  }
  searchTimer = window.setTimeout(async () => {
    try {
      const response = await api.get<PhotographyLocationResult[]>('/photography-tools/geocode', {
        params: { query: searchText.value.trim(), country_code: mapCountryCode.value },
      })
      searchResults.value = response.data || []
      searchVisible.value = searchResults.value.length > 0
    } catch (error) {
      Message.warning(getApiErrorDetail(error) || '地址搜索失败')
    }
  }, 350)
}
async function renderHourlyChart() {
  await nextTick()
  const hours = todayHours.value
  if (!hourlyChartRef.value || hours.length === 0) {
    hourlyChart?.dispose()
    hourlyChart = null
    return
  }
  if (!hourlyChart) hourlyChart = echarts.init(hourlyChartRef.value)
  hourlyChart.setOption(
    {
      animation: false,
      grid: { left: 12, right: 18, top: 20, bottom: 24, containLabel: true },
      tooltip: {
        trigger: 'axis',
        formatter: (params: unknown) => {
          const point = (Array.isArray(params) ? params[0] : params) as {
            dataIndex?: number
            value?: number | null
          }
          const hour = hours[point.dataIndex ?? 0]
          return `${formatHour(hour.time)}<br/>${weatherIcon(hour.weather_code)} ${
            hour.weather_text || weatherLabel(hour.weather_code)
          }<br/>温度：${formatTemperature(point.value)}`
        },
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: hours.map((hour) => formatHour(hour.time)),
        axisLabel: { interval: 2, color: '#86868B' },
        axisLine: { lineStyle: { color: '#D8D8DE' } },
      },
      yAxis: {
        type: 'value',
        name: '°C',
        nameTextStyle: { color: '#86868B' },
        axisLabel: { color: '#86868B', formatter: '{value}°' },
        splitLine: { lineStyle: { color: '#EDEDF0' } },
      },
      series: [
        {
          type: 'line',
          smooth: true,
          connectNulls: true,
          symbol: 'circle',
          symbolSize: 7,
          data: hours.map((hour) => hour.temperature),
          lineStyle: { width: 3, color: '#007AFF' },
          itemStyle: { color: '#007AFF', borderColor: '#FFFFFF', borderWidth: 2 },
          areaStyle: { color: 'rgba(0, 122, 255, 0.10)' },
        },
      ],
    },
    true,
  )
}
function handleHourlyChartResize() {
  hourlyChart?.resize()
}
watch(searchText, () => searchLocations())
watch([latitude, longitude], () => syncMapPosition())
watch(selectedTz, () => {
  if (!locationCountryCodeLocked.value && hasCoordinates()) void runQuery()
})
onMounted(() => {
  void loadSources()
  window.addEventListener('resize', handleHourlyChartResize)
})
onBeforeUnmount(() => {
  if (searchTimer) window.clearTimeout(searchTimer)
  window.removeEventListener('resize', handleHourlyChartResize)
  hourlyChart?.dispose()
  hourlyChart = null
  disposeMap()
})
</script>
<style scoped lang="scss">
.search-popover {
  width: min(420px, calc(100vw - 48px));
  border: 1px solid #e5e5ea;
  border-radius: 10px;
  background: #fff;
  padding: 4px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.12);
}
.search-result {
  display: flex;
  width: 100%;
  gap: 8px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  padding: 9px 10px;
  text-align: left;
  cursor: pointer;
}
.search-result:hover {
  background: #f5f7fa;
}
.coordinate-map {
  position: relative;
  min-height: 220px;
  overflow: hidden;
  border-radius: 16px;
  background: linear-gradient(145deg, #dbeafe, #dcfce7 48%, #fef3c7);
  cursor: crosshair;
}
.map-canvas,
.map-fallback {
  position: absolute;
  inset: 0;
}
.map-canvas-hidden {
  visibility: hidden;
}
.map-fallback {
  cursor: crosshair;
}
.map-sdk-label {
  pointer-events: none;
}
.map-grid {
  position: absolute;
  inset: 0;
  opacity: 0.45;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.65) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.65) 1px, transparent 1px);
  background-size: 28px 28px;
}
.map-marker {
  position: absolute;
  z-index: 2;
  transform: translate(-50%, -100%);
  color: #007aff;
  filter: drop-shadow(0 3px 5px rgba(0, 0, 0, 0.2));
}
.map-label {
  position: absolute;
  bottom: 8px;
  left: 8px;
  z-index: 3;
  display: flex;
  align-items: center;
  gap: 4px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.82);
  padding: 4px 7px;
  font-size: 11px;
  color: #5d5d63;
}
.metric-box {
  border-radius: 10px;
  background: #f5f5f7;
  padding: 8px;
  color: #86868b;
}
.metric-box b {
  display: block;
  margin-top: 2px;
  color: #1d1d1f;
  font-size: 13px;
}
.info-panel {
  height: 100%;
  border-radius: 16px;
  background: #f8fafc;
  padding: 16px;
}
.tide-line {
  display: flex;
  height: 60px;
  align-items: end;
  gap: 3px;
  border-bottom: 1px solid #dbeafe;
}
.tide-line span {
  flex: 1;
  min-width: 3px;
  border-radius: 4px 4px 0 0;
  background: linear-gradient(180deg, #60a5fa, #bfdbfe);
}
.day-card {
  border: 1px solid #eef0f3;
  border-radius: 14px;
  padding: 14px;
}
.score-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 12px;
  color: #5d5d63;
}
.chart-bars {
  display: flex;
  flex-direction: column;
  gap: 15px;
  min-height: 190px;
  justify-content: center;
}
.chart-series {
  display: grid;
  grid-template-columns: 110px 1fr;
  align-items: end;
  gap: 10px;
}
.chart-name {
  font-size: 11px;
}
.chart-columns {
  display: flex;
  align-items: end;
  gap: 12px;
  height: 135px;
  border-bottom: 1px solid #e5e7eb;
}
.chart-column {
  display: flex;
  height: 100%;
  min-width: 26px;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: end;
  gap: 2px;
  font-size: 10px;
  color: #86868b;
}
.chart-bar {
  display: block;
  width: 18px;
  min-height: 2px;
  border-radius: 5px 5px 0 0;
  opacity: 0.8;
}
</style>
