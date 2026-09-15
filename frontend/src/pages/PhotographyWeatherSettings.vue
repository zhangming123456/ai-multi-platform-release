<template>
  <div class="page-main photography-settings-page">
    <PageHeader title="天气数据源 / 模型" subtitle="集中管理摄影工具天气查询所使用的第三方服务凭据">
      <template #actions>
        <a-button size="mini" :loading="loading" @click="loadConfigs">
          <template #icon><IconRefresh :size="13" /></template>
          刷新配置
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1 pb-8">
      <a-alert type="info" show-icon class="mb-4">
        API Key 仅提交到后端保存，页面只显示脱敏值。Open-Meteo 不配置 Key
        也可以使用；和风天气需要配置 API Key 或 JWT Token。
      </a-alert>

      <a-spin :loading="loading" class="w-full">
        <a-row :gutter="[16, 16]">
          <a-col v-for="item in configs" :key="item.source" :xs="24" :lg="12">
            <a-card :bordered="false" class="h-full shadow-sm">
              <template #title>
                <div class="flex items-center gap-3">
                  <div class="source-icon"><IconCloud /></div>
                  <div>
                    <div class="text-[15px] font-semibold text-[#1D1D1F]">{{ item.name }}</div>
                    <div class="text-[11px] text-[#86868B]">{{ item.provider }}</div>
                  </div>
                </div>
              </template>
              <template #extra>
                <a-tag
                  :color="item.requires_key && !item.configured ? 'orange' : 'green'"
                  size="small"
                >
                  {{ item.requires_key && !item.configured ? '未配置' : '可用' }}
                </a-tag>
              </template>

              <div class="mb-4 text-[12px] leading-relaxed text-[#5D5D63]">
                {{ item.description }}
              </div>
              <a-form v-if="forms[item.source]" :model="forms[item.source]" layout="vertical">
                <a-form-item v-if="item.requires_key" label="认证方式">
                  <a-radio-group v-model="forms[item.source].credential_type" type="button">
                    <a-radio value="api_key">API Key</a-radio>
                    <a-radio value="token">JWT Token</a-radio>
                  </a-radio-group>
                </a-form-item>
                <a-form-item
                  :label="item.requires_key ? 'Key / Token' : 'Open-Meteo API Key（可选）'"
                >
                  <a-input-password
                    v-model="forms[item.source].api_key"
                    :placeholder="
                      item.api_key_masked
                        ? '已配置 ' + item.api_key_masked + '，留空表示不修改'
                        : '请输入凭据'
                    "
                    allow-clear
                  />
                  <div v-if="item.api_key_masked" class="mt-1 text-[11px] text-[#86868B]">
                    当前值：{{ item.api_key_masked }}
                  </div>
                </a-form-item>
                <a-form-item v-if="item.supports_host" label="API Host">
                  <a-input
                    v-model="forms[item.source].api_host"
                    placeholder="例如：https://devapi.qweather.com"
                  />
                </a-form-item>
                <a-checkbox v-if="item.api_key_masked" v-model="forms[item.source].clear_api_key">
                  清除已保存的凭据
                </a-checkbox>
                <div class="mt-5 flex justify-end">
                  <a-button
                    v-perm="'photography_weather_config:update:write'"
                    type="primary"
                    size="small"
                    :loading="savingSource === item.source"
                    @click="saveConfig(item)"
                  >
                    保存配置
                  </a-button>
                </div>
              </a-form>
            </a-card>
          </a-col>
        </a-row>
        <a-empty
          v-if="!loading && configs.length === 0"
          description="暂无天气数据源配置"
          class="py-12"
        />
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconCloud, IconRefresh } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api, { getApiErrorDetail } from '@/utils/api'
import type {
  PhotographyWeatherConfig,
  PhotographyWeatherConfigPayload,
  PhotographyWeatherCredentialType,
} from './PhotographyWeatherSettings.types'

type PhotographyWeatherConfigForm = PhotographyWeatherConfigPayload

const configs = ref<PhotographyWeatherConfig[]>([])
const forms = reactive<Record<string, PhotographyWeatherConfigForm>>({})
const loading = ref(false)
const savingSource = ref<string | null>(null)

function ensureForm(item: PhotographyWeatherConfig) {
  if (!forms[item.source]) {
    forms[item.source] = {
      credential_type: item.credential_type || 'api_key',
      api_key: '',
      api_host: item.api_host || '',
      clear_api_key: false,
    }
  } else {
    forms[item.source].credential_type = item.credential_type || 'api_key'
    forms[item.source].api_host = item.api_host || ''
    forms[item.source].api_key = ''
    forms[item.source].clear_api_key = false
  }
}

async function loadConfigs() {
  loading.value = true
  try {
    const response = await api.get<PhotographyWeatherConfig[]>('/photography-weather-configs')
    configs.value = response.data || []
    configs.value.forEach(ensureForm)
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '加载天气配置失败')
  } finally {
    loading.value = false
  }
}

async function saveConfig(item: PhotographyWeatherConfig) {
  const form = forms[item.source]
  if (!form) return
  savingSource.value = item.source
  try {
    const payload: PhotographyWeatherConfigPayload = {
      credential_type: form.credential_type as PhotographyWeatherCredentialType,
      api_key: form.api_key.trim(),
      api_host: form.api_host.trim(),
      clear_api_key: form.clear_api_key,
    }
    const response = await api.put<PhotographyWeatherConfig>(
      '/photography-weather-configs/' + item.source,
      payload,
    )
    const index = configs.value.findIndex((config) => config.source === item.source)
    if (index !== -1) configs.value[index] = response.data
    ensureForm(response.data)
    Message.success(item.name + '配置已保存')
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '保存天气配置失败')
  } finally {
    savingSource.value = null
  }
}

onMounted(loadConfigs)
</script>

<style scoped lang="scss">
.photography-settings-page :deep(.arco-card-header) {
  padding-bottom: 14px;
}

.source-icon {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  color: #007aff;
  background: rgba(0, 122, 255, 0.1);
}
</style>
