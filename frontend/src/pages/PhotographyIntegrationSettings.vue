<template>
  <div class="page-main photography-settings-page">
    <PageHeader
      title="第三方服务配置"
      subtitle="配置地图、天气、潮汐、极光与日历订阅服务；密钥由后端 AES-GCM 加密保存"
    >
      <template #actions>
        <a-button size="mini" :loading="loading" @click="loadConfig">
          <template #icon><IconRefresh :size="13" /></template>
          刷新配置
        </a-button>
      </template>
    </PageHeader>

    <div class="flex-1 px-4 pb-8 md:px-6 lg:px-8">
      <a-alert type="info" show-icon class="mb-4">
        浏览器地图 Key 会作为运行时公开 Key 下发，请在高德/Google 控制台配置域名或 IP
        限制。服务端天气、潮汐和 NOAA 密钥不会返回前端，也不会写入日志。
      </a-alert>

      <a-spin :loading="loading" class="w-full">
        <a-row :gutter="[16, 16]">
          <a-col :xs="24">
            <a-card :bordered="false" class="h-full shadow-sm">
              <template #title>地图服务</template>
              <div class="provider-blocks">
                <div class="provider-block">
                  <div class="provider-block-head">
                    <div>
                      <div class="provider-block-title">高德地图</div>
                      <div class="provider-block-desc">
                        中国大陆（含港澳台）默认使用，不自动降级。需要分别申请「Web端(JS
                        API)」与「Web服务」两种 Key。
                      </div>
                    </div>
                  </div>
                  <a-form :model="form" layout="vertical">
                    <a-form-item>
                      <template #label>
                        高德 Web Key（Web端 JS API）
                        <a
                          class="key-guide"
                          href="https://console.amap.com/dev/key/app"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.amap_web_key"
                        :placeholder="placeholder(config?.amap_web_key_masked)"
                        allow-clear
                      />
                      <template #extra>
                        <span class="text-[12px] text-[#86868B]">
                          用于浏览器地图显示。高德要求区分平台，这里必须填「Web端(JS API)」类型的
                          Key。
                        </span>
                      </template>
                    </a-form-item>
                    <a-checkbox v-model="form.clear_amap_web_key">清除高德 Web Key</a-checkbox>
                    <a-form-item>
                      <template #label>
                        高德 Web 服务 Key（服务端）
                        <a
                          class="key-guide"
                          href="https://console.amap.com/dev/key/app"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.amap_web_service_key"
                        :placeholder="placeholder(config?.amap_web_service_key_masked)"
                        allow-clear
                      />
                      <template #extra>
                        <span class="text-[12px] text-[#86868B]">
                          用于服务端地址搜索（地理编码）。必须填「Web服务」类型的 Key；若误填 Web端
                          Key，高德会返回 USERKEY_PLAT_NOMATCH。未配置时地址搜索将不可用。
                        </span>
                      </template>
                    </a-form-item>
                    <a-checkbox v-model="form.clear_amap_web_service_key">
                      清除高德 Web 服务 Key
                    </a-checkbox>
                    <a-form-item>
                      <template #label>
                        高德安全密钥（Web端 JS API）
                        <a
                          class="key-guide"
                          href="https://lbs.amap.com/api/javascript-api-v2/guide/abc/prepare"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.amap_security_key"
                        :placeholder="placeholder(config?.amap_security_key_masked)"
                        allow-clear
                      />
                    </a-form-item>
                    <a-checkbox v-model="form.clear_amap_security_key">清除高德安全密钥</a-checkbox>
                  </a-form>
                </div>

                <div class="provider-block">
                  <div class="provider-block-head">
                    <div>
                      <div class="provider-block-title">Google 地图</div>
                      <div class="provider-block-desc">
                        非中国大陆地区默认使用。浏览器端 Key 请在 Google Cloud 配置来源（HTTP
                        referrer）限制。
                      </div>
                    </div>
                  </div>
                  <a-form :model="form" layout="vertical">
                    <a-form-item>
                      <template #label>
                        Google Maps Browser Key
                        <a
                          class="key-guide"
                          href="https://console.cloud.google.com/google/maps-apis/credentials"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.google_maps_key"
                        :placeholder="placeholder(config?.google_maps_key_masked)"
                        allow-clear
                      />
                    </a-form-item>
                    <a-checkbox v-model="form.clear_google_maps_key">
                      清除 Google Maps Key
                    </a-checkbox>
                  </a-form>
                </div>
              </div>
              <div class="mt-5 flex justify-end">
                <a-button
                  v-perm="'photography_tool:config:update'"
                  type="primary"
                  :loading="savingSections.map"
                  @click="saveConfig('map')"
                >
                  保存地图配置
                </a-button>
              </div>
            </a-card>
          </a-col>

          <a-col :xs="24">
            <a-card :bordered="false" class="h-full shadow-sm">
              <template #title>天气服务</template>
              <template #extra>
                <router-link
                  v-perm="'photography_weather_config:read'"
                  class="key-guide"
                  to="/settings/photography-weather"
                >
                  按模型 / 数据源配置
                </router-link>
              </template>
              <div class="provider-blocks">
                <div class="provider-block">
                  <div class="provider-block-head">
                    <div>
                      <div class="provider-block-title">Open-Meteo</div>
                      <div class="provider-block-desc">
                        日出日落、云量、降水概率与空间分布的核心数据来源；不配置 Key 也可以使用。
                      </div>
                      <div class="provider-block-status">
                        {{ providerHint('weather_providers', 'open-meteo') }}
                      </div>
                    </div>
                    <a-switch v-model="form.open_meteo_enabled" />
                  </div>
                  <a-form :model="form" layout="vertical">
                    <a-form-item label="Open-Meteo API Host">
                      <a-input
                        v-model="form.open_meteo_host"
                        placeholder="https://api.open-meteo.com/v1/forecast"
                      />
                    </a-form-item>
                    <a-form-item>
                      <template #label>
                        Open-Meteo API Key（可选）
                        <a
                          class="key-guide"
                          href="https://open-meteo.com/en/pricing"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.open_meteo_key"
                        :placeholder="placeholder(config?.open_meteo_key_masked)"
                        allow-clear
                      />
                    </a-form-item>
                    <a-checkbox v-model="form.clear_open_meteo_key">清除 Open-Meteo Key</a-checkbox>
                  </a-form>
                </div>

                <div class="provider-block">
                  <div class="provider-block-head">
                    <div>
                      <div class="provider-block-title">和风天气</div>
                      <div class="provider-block-desc">
                        支持 API Key 与 JWT Token 两种认证方式，逐日与逐小时数据最多未来 10 天。
                      </div>
                      <div class="provider-block-status">
                        {{ providerHint('weather_providers', 'qweather') }}
                      </div>
                    </div>
                    <a-switch v-model="form.qweather_enabled" />
                  </div>
                  <a-form :model="form" layout="vertical">
                    <a-form-item label="和风天气认证方式">
                      <a-radio-group v-model="form.qweather_credential_type" type="button">
                        <a-radio value="api_key">API Key</a-radio>
                        <a-radio value="token">JWT Token</a-radio>
                      </a-radio-group>
                    </a-form-item>
                    <a-form-item>
                      <template #label>
                        和风天气 API Key / Token
                        <a
                          class="key-guide"
                          href="https://console.qweather.com/project"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.qweather_key"
                        :placeholder="placeholder(config?.qweather_key_masked)"
                        allow-clear
                      />
                    </a-form-item>
                    <a-checkbox v-model="form.clear_qweather_key">
                      清除和风天气 Key / Token
                    </a-checkbox>
                    <a-form-item label="和风天气 API Host">
                      <a-input
                        v-model="form.qweather_host"
                        placeholder="https://devapi.qweather.com"
                      />
                    </a-form-item>
                  </a-form>
                </div>
              </div>
              <div class="mt-5 flex justify-end">
                <a-button
                  v-perm="'photography_tool:config:update'"
                  type="primary"
                  :loading="savingSections.weather"
                  @click="saveConfig('weather')"
                >
                  保存天气配置
                </a-button>
              </div>
            </a-card>
          </a-col>

          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" class="h-full shadow-sm">
              <template #title>潮汐服务</template>
              <template #extra>
                <a-space :size="8">
                  <span class="text-xs text-[#86868B]">启用潮汐服务</span>
                  <a-switch v-model="form.tide_enabled" size="small" />
                </a-space>
              </template>
              <div class="provider-blocks">
                <div class="provider-block">
                  <div class="provider-block-head">
                    <div>
                      <div class="provider-block-title">NOAA CO-OPS</div>
                      <div class="provider-block-desc">美国海域官方站点，无需 Key。</div>
                    </div>
                  </div>
                </div>

                <div class="provider-block">
                  <div class="provider-block-head">
                    <div>
                      <div class="provider-block-title">Stormglass（全球潮汐）</div>
                      <div class="provider-block-desc">覆盖美国以外海域，需要 API Key。</div>
                      <div class="provider-block-status">
                        {{ providerHint('tide_providers', 'stormglass') }}
                      </div>
                    </div>
                  </div>
                  <a-form :model="form" layout="vertical">
                    <a-form-item>
                      <template #label>
                        全球潮汐服务 Key（Stormglass 等）
                        <a
                          class="key-guide"
                          href="https://dashboard.stormglass.io/"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.tide_key"
                        :placeholder="placeholder(config?.tide_key_masked)"
                        allow-clear
                      />
                    </a-form-item>
                    <a-checkbox v-model="form.clear_tide_key">清除潮汐服务 Key</a-checkbox>
                    <a-form-item label="全球潮汐 API Host">
                      <a-input v-model="form.tide_host" placeholder="可选，由后端适配器使用" />
                    </a-form-item>
                  </a-form>
                </div>
              </div>
              <div class="mt-5 flex justify-end">
                <a-button
                  v-perm="'photography_tool:config:update'"
                  type="primary"
                  :loading="savingSections.tide"
                  @click="saveConfig('tide')"
                >
                  保存潮汐配置
                </a-button>
              </div>
            </a-card>
          </a-col>

          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" class="h-full shadow-sm">
              <template #title>极光服务</template>
              <div class="provider-blocks">
                <div class="provider-block">
                  <div class="provider-block-head">
                    <div>
                      <div class="provider-block-title">NOAA SWPC</div>
                      <div class="provider-block-desc">
                        公开 Kp 指数预报，Key 可选，仅用于提高配额。
                      </div>
                      <div class="provider-block-status">
                        {{ providerHint('aurora_providers', 'noaa-swpc') }}
                      </div>
                    </div>
                    <a-switch v-model="form.noaa_enabled" />
                  </div>
                  <a-form :model="form" layout="vertical">
                    <a-form-item>
                      <template #label>
                        NOAA API Key（可选）
                        <a
                          class="key-guide"
                          href="https://www.ncei.noaa.gov/cdo-web/token"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          如何获取
                        </a>
                      </template>
                      <a-input-password
                        v-model="form.noaa_key"
                        :placeholder="placeholder(config?.noaa_key_masked)"
                        allow-clear
                      />
                    </a-form-item>
                    <a-checkbox v-model="form.clear_noaa_key">清除 NOAA Key</a-checkbox>
                    <a-form-item label="NOAA Host">
                      <a-input
                        v-model="form.noaa_host"
                        placeholder="可选，默认 services.swpc.noaa.gov"
                      />
                    </a-form-item>
                  </a-form>
                </div>
              </div>
              <div class="mt-5 flex justify-end">
                <a-button
                  v-perm="'photography_tool:config:update'"
                  type="primary"
                  :loading="savingSections.aurora"
                  @click="saveConfig('aurora')"
                >
                  保存极光配置
                </a-button>
              </div>
            </a-card>
          </a-col>

          <a-col v-if="canReadHolidaySource" :xs="24">
            <a-card :bordered="false" class="shadow-sm">
              <template #title>日历订阅源</template>
              <template #extra>
                <a
                  class="key-guide"
                  href="https://support.apple.com/zh-cn/guide/calendar/icl1022/mac"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  如何获取订阅地址
                </a>
              </template>
              <HolidaySourceManager />
            </a-card>
          </a-col>
        </a-row>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconRefresh } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import HolidaySourceManager from '@/components/HolidaySourceManager.vue'
import api, { getApiErrorDetail } from '@/utils/api'
import { usePermissionStore } from '@/stores/permission'
import type {
  PhotographyIntegrationConfig,
  PhotographyIntegrationForm,
  PhotographyIntegrationPayload,
  PhotographyIntegrationSection,
} from './PhotographyIntegrationSettings.types'

const config = ref<PhotographyIntegrationConfig | null>(null)
const loading = ref(false)
const permStore = usePermissionStore()
// 日历订阅源有自己的权限位，无权限时不渲染该卡片。
const canReadHolidaySource = computed(() => permStore.hasPermission('holiday_source:read'))

type ProviderGroup = 'weather_providers' | 'tide_providers' | 'aurora_providers'

// 小区域头部展示凭据状态：已配置脱敏值 / 未配置 / 无需 Key。
function providerHint(group: ProviderGroup, id: string): string {
  const provider = (config.value?.[group] || []).find((item) => item.id === id)
  if (!provider) return ''
  if (provider.api_key_masked) return `已配置 ${provider.api_key_masked}`
  return provider.requires_key ? '未配置' : '无需 Key'
}

const savingSections = reactive<Record<PhotographyIntegrationSection, boolean>>({
  map: false,
  weather: false,
  tide: false,
  aurora: false,
})

const sectionNames: Record<PhotographyIntegrationSection, string> = {
  map: '地图',
  weather: '天气',
  tide: '潮汐',
  aurora: '极光',
}

const form = reactive<PhotographyIntegrationForm>({
  amap_web_key: '',
  amap_web_service_key: '',
  amap_security_key: '',
  google_maps_key: '',
  open_meteo_key: '',
  qweather_key: '',
  qweather_credential_type: 'api_key',
  noaa_key: '',
  tide_key: '',
  open_meteo_host: '',
  qweather_host: '',
  noaa_host: '',
  tide_host: '',
  open_meteo_enabled: true,
  qweather_enabled: false,
  noaa_enabled: true,
  tide_enabled: true,
  clear_amap_web_key: false,
  clear_amap_web_service_key: false,
  clear_amap_security_key: false,
  clear_google_maps_key: false,
  clear_open_meteo_key: false,
  clear_qweather_key: false,
  clear_noaa_key: false,
  clear_tide_key: false,
})

function placeholder(value?: string) {
  return value ? '已配置 ' + value + '，留空表示不修改' : '请输入密钥'
}

function resetSecretFields() {
  form.amap_web_key = ''
  form.amap_web_service_key = ''
  form.amap_security_key = ''
  form.google_maps_key = ''
  form.open_meteo_key = ''
  form.qweather_key = ''
  form.noaa_key = ''
  form.tide_key = ''
  form.clear_amap_web_key = false
  form.clear_amap_web_service_key = false
  form.clear_amap_security_key = false
  form.clear_google_maps_key = false
  form.clear_open_meteo_key = false
  form.clear_qweather_key = false
  form.clear_noaa_key = false
  form.clear_tide_key = false
}

function syncForm(value: PhotographyIntegrationConfig) {
  config.value = value
  form.open_meteo_host = value.open_meteo_host || ''
  form.qweather_host = value.qweather_host || ''
  form.qweather_credential_type = value.qweather_credential_type || 'api_key'
  form.noaa_host = value.noaa_host || ''
  form.tide_host = value.tide_host || ''
  form.open_meteo_enabled = value.open_meteo_enabled
  form.qweather_enabled = value.qweather_enabled
  form.noaa_enabled = value.noaa_enabled
  form.tide_enabled = value.tide_enabled
  resetSecretFields()
}

function syncSavedSection(
  section: PhotographyIntegrationSection,
  value: PhotographyIntegrationConfig,
) {
  config.value = value
  if (section === 'map') {
    form.amap_web_key = ''
    form.amap_web_service_key = ''
    form.amap_security_key = ''
    form.google_maps_key = ''
    form.clear_amap_web_key = false
    form.clear_amap_web_service_key = false
    form.clear_amap_security_key = false
    form.clear_google_maps_key = false
  } else if (section === 'weather') {
    form.open_meteo_host = value.open_meteo_host || ''
    form.qweather_host = value.qweather_host || ''
    form.qweather_credential_type = value.qweather_credential_type || 'api_key'
    form.open_meteo_enabled = value.open_meteo_enabled
    form.qweather_enabled = value.qweather_enabled
    form.open_meteo_key = ''
    form.qweather_key = ''
    form.clear_open_meteo_key = false
    form.clear_qweather_key = false
  } else if (section === 'tide') {
    form.tide_host = value.tide_host || ''
    form.tide_enabled = value.tide_enabled
    form.tide_key = ''
    form.clear_tide_key = false
  } else {
    form.noaa_host = value.noaa_host || ''
    form.noaa_enabled = value.noaa_enabled
    form.noaa_key = ''
    form.clear_noaa_key = false
  }
}

function buildPayload(section: PhotographyIntegrationSection): PhotographyIntegrationPayload {
  if (section === 'map') {
    return {
      section,
      amap_web_key: form.amap_web_key,
      amap_web_service_key: form.amap_web_service_key,
      amap_security_key: form.amap_security_key,
      google_maps_key: form.google_maps_key,
      clear_amap_web_key: form.clear_amap_web_key,
      clear_amap_web_service_key: form.clear_amap_web_service_key,
      clear_amap_security_key: form.clear_amap_security_key,
      clear_google_maps_key: form.clear_google_maps_key,
    }
  }
  if (section === 'weather') {
    return {
      section,
      open_meteo_key: form.open_meteo_key,
      qweather_key: form.qweather_key,
      qweather_credential_type: form.qweather_credential_type,
      open_meteo_host: form.open_meteo_host,
      qweather_host: form.qweather_host,
      open_meteo_enabled: form.open_meteo_enabled,
      qweather_enabled: form.qweather_enabled,
      clear_open_meteo_key: form.clear_open_meteo_key,
      clear_qweather_key: form.clear_qweather_key,
    }
  }
  if (section === 'tide') {
    return {
      section,
      tide_key: form.tide_key,
      tide_host: form.tide_host,
      tide_enabled: form.tide_enabled,
      clear_tide_key: form.clear_tide_key,
    }
  }
  return {
    section,
    noaa_key: form.noaa_key,
    noaa_host: form.noaa_host,
    noaa_enabled: form.noaa_enabled,
    clear_noaa_key: form.clear_noaa_key,
  }
}

async function loadConfig() {
  loading.value = true
  try {
    const response = await api.get<PhotographyIntegrationConfig>('/system/integrations/photography')
    syncForm(response.data)
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '加载摄影工具集成配置失败')
  } finally {
    loading.value = false
  }
}

async function saveConfig(section: PhotographyIntegrationSection) {
  savingSections[section] = true
  try {
    const response = await api.put<PhotographyIntegrationConfig>(
      '/system/integrations/photography',
      buildPayload(section),
    )
    syncSavedSection(section, response.data)
    Message.success(sectionNames[section] + '配置已保存')
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '保存' + sectionNames[section] + '配置失败')
  } finally {
    savingSections[section] = false
  }
}

onMounted(loadConfig)
</script>

<style scoped lang="scss">
.key-guide {
  margin-left: 6px;
  font-size: 12px;
  font-weight: 400;
  color: #007aff;
  text-decoration: none;
}

.key-guide:hover {
  color: #0071e3;
  text-decoration: underline;
}

.provider-blocks {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 14px;
}

.provider-block {
  border: 1px solid #e5e5ea;
  border-radius: 12px;
  background: #fafafa;
  padding: 14px 16px;
}

.provider-block-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.provider-block-head > div:first-child {
  min-width: 0;
}

.provider-block-title {
  font-size: 14px;
  font-weight: 600;
  color: #1d1d1f;
}

.provider-block-desc {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.6;
  color: #86868b;
}

.provider-block-status {
  margin-top: 6px;
  font-size: 12px;
  color: #007aff;
}
</style>
