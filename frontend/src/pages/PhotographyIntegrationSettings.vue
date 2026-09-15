<template>
  <div class="page-main photography-settings-page">
    <PageHeader
      title="摄影工具集成"
      subtitle="配置地图、天气、潮汐和极光服务；密钥由后端 AES-GCM 加密保存"
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
        浏览器地图 Key 会作为运行时公开 Key 下发，请在高德/Google 控制台配置域名或 IP 限制。服务端天气、潮汐和 NOAA 密钥不会返回前端，也不会写入日志。
      </a-alert>

      <a-spin :loading="loading" class="w-full">
        <a-row :gutter="[16, 16]">
          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" class="h-full shadow-sm">
              <template #title>地图服务</template>
              <a-form :model="form" layout="vertical">
                <a-form-item label="高德 Web Key">
                  <a-input-password v-model="form.amap_web_key" :placeholder="placeholder(config?.amap_web_key_masked)" allow-clear />
                </a-form-item>
                <a-checkbox v-model="form.clear_amap_web_key">清除高德 Web Key</a-checkbox>
                <a-form-item label="高德安全密钥">
                  <a-input-password v-model="form.amap_security_key" :placeholder="placeholder(config?.amap_security_key_masked)" allow-clear />
                </a-form-item>
                <a-checkbox v-model="form.clear_amap_security_key">清除高德安全密钥</a-checkbox>
                <a-form-item label="Google Maps Browser Key">
                  <a-input-password v-model="form.google_maps_key" :placeholder="placeholder(config?.google_maps_key_masked)" allow-clear />
                </a-form-item>
                <a-checkbox v-model="form.clear_google_maps_key">清除 Google Maps Key</a-checkbox>
              </a-form>
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

          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" class="h-full shadow-sm">
              <template #title>天气服务</template>
              <a-form :model="form" layout="vertical">
                <div v-for="provider in config?.weather_providers || []" :key="provider.id" class="provider-row">
                  <div>
                    <div class="font-medium">{{ provider.name }}</div>
                    <div class="text-xs text-[#86868B]">
                      {{ provider.configured ? provider.api_key_masked || '无需 Key' : '未配置' }}
                    </div>
                  </div>
                  <a-switch v-if="provider.id === 'open-meteo'" v-model="form.open_meteo_enabled" />
                  <a-switch v-else v-model="form.qweather_enabled" />
                </div>
                <a-form-item label="Open-Meteo API Host">
                  <a-input v-model="form.open_meteo_host" placeholder="https://api.open-meteo.com/v1/forecast" />
                </a-form-item>
                <a-form-item label="Open-Meteo API Key（可选）">
                  <a-input-password v-model="form.open_meteo_key" :placeholder="placeholder(config?.open_meteo_key_masked)" allow-clear />
                </a-form-item>
                <a-checkbox v-model="form.clear_open_meteo_key">清除 Open-Meteo Key</a-checkbox>
                <a-form-item label="和风天气认证方式">
                  <a-radio-group v-model="form.qweather_credential_type" type="button">
                    <a-radio value="api_key">API Key</a-radio>
                    <a-radio value="token">JWT Token</a-radio>
                  </a-radio-group>
                </a-form-item>
                <a-form-item label="和风天气 API Key / Token">
                  <a-input-password v-model="form.qweather_key" :placeholder="placeholder(config?.qweather_key_masked)" allow-clear />
                </a-form-item>
                <a-checkbox v-model="form.clear_qweather_key">清除和风天气 Key / Token</a-checkbox>
                <a-form-item label="和风天气 API Host">
                  <a-input v-model="form.qweather_host" placeholder="https://devapi.qweather.com" />
                </a-form-item>
              </a-form>
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
              <a-form :model="form" layout="vertical">
                <div class="provider-row">
                  <div>
                    <div class="font-medium">NOAA CO-OPS</div>
                    <div class="text-xs text-[#86868B]">美国海域公共站点，无需 Key</div>
                  </div>
                  <a-switch v-model="form.tide_enabled" />
                </div>
                <a-form-item label="全球潮汐服务 Key（Stormglass 等）">
                  <a-input-password v-model="form.tide_key" :placeholder="placeholder(config?.tide_key_masked)" allow-clear />
                </a-form-item>
                <a-checkbox v-model="form.clear_tide_key">清除潮汐服务 Key</a-checkbox>
                <a-form-item label="全球潮汐 API Host">
                  <a-input v-model="form.tide_host" placeholder="可选，由后端适配器使用" />
                </a-form-item>
              </a-form>
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
              <a-form :model="form" layout="vertical">
                <div class="provider-row">
                  <div>
                    <div class="font-medium">NOAA SWPC</div>
                    <div class="text-xs text-[#86868B]">公开 Kp 指数；Key 可选</div>
                  </div>
                  <a-switch v-model="form.noaa_enabled" />
                </div>
                <a-form-item label="NOAA API Key（可选）">
                  <a-input-password v-model="form.noaa_key" :placeholder="placeholder(config?.noaa_key_masked)" allow-clear />
                </a-form-item>
                <a-checkbox v-model="form.clear_noaa_key">清除 NOAA Key</a-checkbox>
                <a-form-item label="NOAA Host">
                  <a-input v-model="form.noaa_host" placeholder="可选，默认 services.swpc.noaa.gov" />
                </a-form-item>
              </a-form>
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
        </a-row>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconRefresh } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api, { getApiErrorDetail } from '@/utils/api'
import type {
  PhotographyIntegrationConfig,
  PhotographyIntegrationForm,
  PhotographyIntegrationPayload,
  PhotographyIntegrationSection,
} from './PhotographyIntegrationSettings.types'

const config = ref<PhotographyIntegrationConfig | null>(null)
const loading = ref(false)
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
  form.amap_security_key = ''
  form.google_maps_key = ''
  form.open_meteo_key = ''
  form.qweather_key = ''
  form.noaa_key = ''
  form.tide_key = ''
  form.clear_amap_web_key = false
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

function syncSavedSection(section: PhotographyIntegrationSection, value: PhotographyIntegrationConfig) {
  config.value = value
  if (section === 'map') {
    form.amap_web_key = ''
    form.amap_security_key = ''
    form.google_maps_key = ''
    form.clear_amap_web_key = false
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
      amap_security_key: form.amap_security_key,
      google_maps_key: form.google_maps_key,
      clear_amap_web_key: form.clear_amap_web_key,
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
.provider-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
  border-radius: 12px;
  background: #f5f5f7;
  padding: 11px 13px;
}
</style>
