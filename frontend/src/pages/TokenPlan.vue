<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, onMounted } from 'vue'
import { useTokenPlanStore, providerLabel, type PlanProvider } from '@/stores/tokenPlan'
import {
  IconCheckCircle,
  IconUpCircle,
  IconBarChart,
  IconPlus,
  IconDelete,
  IconRefresh,
  IconDown,
  IconInfoCircleFill,
  IconLink,
  IconEdit,
} from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'

const store = useTokenPlanStore()

interface FormState {
  mode: 'provider' | 'custom'
  provider: PlanProvider
  apiFormat: string
  apiKey: string
  baseUrl: string
  fullUrl: boolean
  model: string
  multimodal: boolean
  modelSeries: string
  displayName: string
  contextInput: number
  contextOutput: number
  toolCallRounds: number
  monthlyQuota: number
}

function blankForm(): FormState {
  return {
    mode: 'provider',
    provider: 'openai',
    apiFormat: 'openai_chat',
    apiKey: '',
    baseUrl: '',
    fullUrl: false,
    model: 'gpt-4o-mini',
    multimodal: false,
    modelSeries: 'default',
    displayName: '',
    contextInput: 128000,
    contextOutput: 4096,
    toolCallRounds: 200,
    monthlyQuota: 1000000,
  }
}

const showModal = ref(false)
const isEdit = ref(false)
const editingId = ref('')
const form = ref<FormState>(blankForm())
const advanced = ref(false)
const fetching = ref(false)
const fetchedModels = ref<string[]>([])

const presetModels: Record<string, string[]> = {
  openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo'],
  deepseek: ['deepseek-chat', 'deepseek-r1-chat', 'deepseek-coder'],
  moonshot: ['moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k'],
  zhipu: ['glm-4-plus', 'glm-4-air', 'glm-4-flash', 'glm-4v-plus'],
  custom: [],
}

const providerDefaultModel: Record<string, string> = {
  openai: 'gpt-4o-mini',
  deepseek: 'deepseek-chat',
  moonshot: 'moonshot-v1-8k',
  zhipu: 'glm-4-plus',
  custom: '',
}

const modelContext: Record<string, { i: number; o: number }> = {
  'gpt-4o': { i: 128000, o: 16384 },
  'gpt-4o-mini': { i: 128000, o: 16384 },
  'gpt-4-turbo': { i: 128000, o: 4096 },
  'gpt-4': { i: 8192, o: 8192 },
  'gpt-3.5-turbo': { i: 16384, o: 4096 },
  'deepseek-chat': { i: 128000, o: 8192 },
  'deepseek-r1-chat': { i: 128000, o: 8192 },
  'deepseek-coder': { i: 128000, o: 8192 },
  'moonshot-v1-8k': { i: 8192, o: 8192 },
  'moonshot-v1-32k': { i: 32768, o: 8192 },
  'moonshot-v1-128k': { i: 128000, o: 8192 },
  'glm-4-plus': { i: 128000, o: 4096 },
  'glm-4-air': { i: 128000, o: 4096 },
  'glm-4-flash': { i: 128000, o: 4096 },
  'glm-4v-plus': { i: 8192, o: 4096 },
}

const providerOptions = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'moonshot', label: 'Moonshot' },
  { value: 'zhipu', label: '智谱 AI' },
]

const apiFormatOptions = [
  { value: 'openai_chat', label: 'OpenAI Chat Completions 格式' },
  { value: 'anthropic', label: 'Anthropic Messages 格式' },
  { value: 'gemini', label: 'Google Gemini 格式' },
]

const modelSeriesOptions = [
  { value: 'default', label: '默认' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'claude', label: 'Claude' },
  { value: 'qwen', label: 'Qwen' },
  { value: 'glm', label: 'GLM' },
]

const bannerText = computed(() => {
  if (form.value.apiFormat === 'openai_chat') {
    return '请填写兼容 OpenAI API 的服务端点地址，不要以斜杠结尾。/chat/completions 将会被补充到你填写的地址末尾。'
  }
  if (form.value.apiFormat === 'anthropic') {
    return '请填写兼容 Anthropic API 的服务端点地址，/v1/messages 将会被补充到地址末尾。'
  }
  return '请填写兼容所选 API 格式的服务端点地址，不要以斜杠结尾。'
})

function getModelContext(id: string) {
  return modelContext[id] || { i: 128000, o: 4096 }
}

function applyContext(id: string) {
  const c = getModelContext(id)
  form.value.contextInput = c.i
  form.value.contextOutput = c.o
}

function getBaseUrl(provider: string, customUrl?: string): string {
  if (provider === 'openai') return 'https://api.openai.com/v1'
  if (provider === 'deepseek') return 'https://api.deepseek.com/v1'
  if (provider === 'moonshot') return 'https://api.moonshot.cn/v1'
  if (provider === 'zhipu') return 'https://open.bigmodel.cn/api/paas/v4'
  return customUrl || ''
}

async function doFetch(url: string, target: 'provider' | 'custom') {
  if (!form.value.apiKey) {
    Message.warning('请先填写 API 密钥')
    return
  }
  if (!url) {
    Message.warning('请先填写请求地址')
    return
  }
  fetching.value = true
  try {
    const res = await api.post('/models/fetch', {
      base_url: url,
      api_key: form.value.apiKey,
    })
    const models: string[] = res.data.models || []
    fetchedModels.value = models
    if (models.length === 0) {
      Message.warning('未获取到模型列表')
    } else {
      Message.success(`获取到 ${models.length} 个模型`)
      if (target === 'custom' && !form.value.model) {
        form.value.model = models[0]
        applyContext(models[0])
      }
    }
  } catch (e: unknown) {
    const msg =
      e instanceof Error ? e.message : (e as { response?: { data?: { detail?: string } } })?.response?.data?.detail || '未知错误'
    Message.error(`拉取失败：${msg}`)
  } finally {
    fetching.value = false
  }
}

function fetchProviderModels() {
  doFetch(getBaseUrl(form.value.provider), 'provider')
}

function fetchCustomModels() {
  doFetch(form.value.baseUrl, 'custom')
}

function onProviderChange() {
  form.value.model = providerDefaultModel[form.value.provider] || ''
  fetchedModels.value = []
  if (form.value.model) applyContext(form.value.model)
}

function onModelChange(val: unknown) {
  if (typeof val === 'string' && val) applyContext(val)
}

function setMode(m: 'provider' | 'custom') {
  form.value.mode = m
  if (m === 'custom') {
    form.value.provider = 'custom'
  } else if (form.value.provider === 'custom') {
    form.value.provider = 'openai'
    form.value.model = providerDefaultModel.openai
    applyContext(form.value.model)
  }
  if (m === 'custom') {
    form.value.model = ''
  }
  fetchedModels.value = []
}

function openAdd() {
  form.value = blankForm()
  isEdit.value = false
  editingId.value = ''
  advanced.value = false
  fetchedModels.value = []
  showModal.value = true
}

function openEdit(id: string) {
  const p = store.plans.find((x) => x.id === id)
  if (!p) return
  form.value = {
    mode: p.mode,
    provider: p.provider,
    apiFormat: p.apiFormat,
    apiKey: p.apiKey,
    baseUrl: p.baseUrl,
    fullUrl: p.fullUrl,
    model: p.model,
    multimodal: p.multimodal,
    modelSeries: p.modelSeries,
    displayName: p.displayName,
    contextInput: p.contextInput,
    contextOutput: p.contextOutput,
    toolCallRounds: p.toolCallRounds,
    monthlyQuota: p.monthlyQuota,
  }
  isEdit.value = true
  editingId.value = id
  advanced.value = true
  fetchedModels.value = []
  showModal.value = true
}

function validate(): boolean {
  const f = form.value
  if (f.mode === 'provider') {
    if (!f.provider) {
      Message.warning('请选择服务商')
      return false
    }
    if (!f.model) {
      Message.warning('请选择或输入模型')
      return false
    }
  } else {
    if (!f.apiFormat) {
      Message.warning('请选择 API 格式')
      return false
    }
    if (!f.baseUrl.trim()) {
      Message.warning('请填写自定义请求地址')
      return false
    }
    if (!f.model.trim()) {
      Message.warning('请填写模型 ID')
      return false
    }
  }
  if (!f.apiKey.trim()) {
    Message.warning('请填写 API 密钥')
    return false
  }
  return true
}

function save() {
  if (!validate()) return
  const f = form.value
  const name = f.displayName || f.model || providerLabel(f.provider)
  const payload = { ...f, name }
  if (isEdit.value) {
    store.updatePlan(editingId.value, payload)
    Message.success('配置已保存')
  } else {
    const np = store.addPlan({ ...payload, enabled: true })
    if (!store.activePlan) store.setActivePlan(np.id)
    Message.success('模型已添加')
  }
  showModal.value = false
}

function toggleEnabled(id: string, enabled: unknown) {
  store.setPlanEnabled(id, Boolean(enabled))
  if (enabled) store.setActivePlan(id)
}

function removePlan(id: string) {
  store.removePlan(id)
}

function getUsagePercent(p: (typeof store.plans)[0]) {
  return Math.min(100, Math.round((p.usedTokens / p.monthlyQuota) * 100))
}

function fmtK(n: number) {
  if (n >= 1000) return `${Math.round((n / 1000) * 10) / 10}k`
  return String(n)
}

function providerColor(p: PlanProvider) {
  switch (p) {
    case 'openai':
      return '#10A37F'
    case 'deepseek':
      return '#4D6BFE'
    case 'moonshot':
      return '#0EA5E9'
    case 'zhipu':
      return '#14B8A6'
    default:
      return '#636366'
  }
}

function providerLetter(p: PlanProvider) {
  switch (p) {
    case 'openai':
      return 'O'
    case 'deepseek':
      return 'D'
    case 'moonshot':
      return 'M'
    case 'zhipu':
      return 'Z'
    default:
      return 'C'
  }
}

watch(showModal, (v) => {
  document.body.classList.toggle('tp-cabin-open', v)
})

onMounted(() => {
  store.loadPlans()
})

onBeforeUnmount(() => {
  document.body.classList.remove('tp-cabin-open')
})
</script>

<template>
  <div>
    <PageHeader title="模型管理" subtitle="配置 AI 模型服务商与自定义接入，统一管理调用配额">
      <template #actions>
        <a-tag :color="store.activePlan ? 'green' : 'red'" size="small">
          <template #icon>
            <IconCheckCircle v-if="store.activePlan" :size="12" />
            <IconUpCircle v-else :size="12" />
          </template>
          {{ store.activePlan ? '已就绪' : '未配置' }}
        </a-tag>
        <a-button type="primary" size="small" @click="openAdd">
          <template #icon><IconPlus /></template>添加模型
        </a-button>
      </template>
    </PageHeader>

    <a-row :gutter="24" class="mb-6">
      <a-col :xs="24" :lg="16">
        <a-space direction="vertical" :size="14" fill>
          <div
            v-for="plan in store.plans"
            :key="plan.id"
            class="plan-card"
            :class="{ active: store.activePlanId === plan.id && plan.enabled }"
          >
            <span class="plan-accent" :style="{ background: providerColor(plan.provider) }" />
            <div class="plan-top">
              <div class="plan-badge" :style="{ background: providerColor(plan.provider) }">
                {{ providerLetter(plan.provider) }}
              </div>
              <div class="plan-meta">
                <div class="plan-title">
                  {{ plan.displayName || plan.model || '未命名模型' }}
                  <span v-if="store.activePlanId === plan.id && plan.enabled" class="plan-live"
                    >生效中</span
                  >
                </div>
                <div class="plan-sub">
                  <span>{{ providerLabel(plan.provider) }}</span>
                  <span class="dot">·</span>
                  <span>{{ plan.mode === 'custom' ? '自定义接入' : '官方服务商' }}</span>
                  <span v-if="plan.multimodal" class="chip-mm">多模态</span>
                </div>
              </div>
              <div class="plan-acts">
                <a-switch
                  :model-value="plan.enabled"
                  @change="(v: unknown) => toggleEnabled(plan.id, v)"
                />
                <button class="icon-btn" title="编辑" @click="openEdit(plan.id)">
                  <IconEdit />
                </button>
                <button
                  class="icon-btn danger"
                  title="删除"
                  :disabled="store.plans.length <= 1"
                  @click="removePlan(plan.id)"
                >
                  <IconDelete />
                </button>
              </div>
            </div>
            <div class="plan-info">
              <span class="info-chip"><b>模型</b>{{ plan.model || '—' }}</span>
              <span class="info-chip"
                ><b>上下文</b>{{ fmtK(plan.contextInput) }} / {{ fmtK(plan.contextOutput) }}</span
              >
              <span class="info-chip"><b>工具轮次</b>{{ plan.toolCallRounds }}</span>
              <span class="info-chip right">
                <b>配额</b>
                <em :class="{ warn: getUsagePercent(plan) > 80 }">{{ getUsagePercent(plan) }}%</em>
              </span>
            </div>
            <div class="plan-bar">
              <span
                :style="{ width: getUsagePercent(plan) + '%' }"
                :class="{ warn: getUsagePercent(plan) > 80 }"
              />
            </div>
          </div>

          <div v-if="store.plans.length === 0" class="empty-card">
            <div class="empty-glyph">＋</div>
            <p>还没有配置任何模型</p>
            <a-button type="primary" size="small" @click="openAdd">
              <template #icon><IconPlus /></template>添加第一个模型
            </a-button>
          </div>
        </a-space>
      </a-col>

      <a-col :xs="24" :lg="8">
        <a-card :bordered="false" title="用量总览" style="padding: 20px">
          <a-space direction="vertical" :size="16" fill>
            <div v-for="plan in store.plans" :key="plan.id">
              <div class="flex items-center justify-between mb-2">
                <a-typography-text class="text-[13px]">{{
                  plan.displayName || plan.model
                }}</a-typography-text>
                <a-typography-text
                  class="text-[12px]"
                  :type="getUsagePercent(plan) > 80 ? 'danger' : 'secondary'"
                >
                  {{ getUsagePercent(plan) }}%
                </a-typography-text>
              </div>
              <a-progress
                :percent="getUsagePercent(plan)"
                :show-text="false"
                size="small"
                :color="getUsagePercent(plan) > 80 ? '#FF3B30' : '#34C759'"
              />
              <div class="flex justify-between mt-1">
                <a-typography-text type="disabled" class="text-[11px]">
                  {{ plan.usedTokens.toLocaleString() }} / {{ plan.monthlyQuota.toLocaleString() }}
                </a-typography-text>
                <a-typography-text type="secondary" class="text-[11px]">
                  剩余 {{ (plan.monthlyQuota - plan.usedTokens).toLocaleString() }}
                </a-typography-text>
              </div>
            </div>

            <a-divider v-if="store.plans.length > 0" style="margin: 12px 0" />

            <div v-if="store.activePlan" class="bg-[#34C759]/10 rounded-[12px] p-4">
              <div class="flex items-center gap-2 mb-2">
                <IconBarChart :size="16" style="color: #34c759" />
                <a-typography-text bold class="text-[13px]" style="color: #248a3d"
                  >当前生效模型</a-typography-text
                >
              </div>
              <a-typography-text class="text-[12px]" style="color: #34c759">
                {{ store.activePlan.displayName || store.activePlan.model }} ·
                {{ providerLabel(store.activePlan.provider) }}
              </a-typography-text>
              <a-typography-text type="secondary" class="text-[11px] block mt-1">
                本月剩余配额：{{ store.getRemainingQuota().toLocaleString() }} tokens
              </a-typography-text>
            </div>
            <div v-else class="bg-[#FF9500]/10 rounded-[12px] p-4">
              <a-typography-text class="text-[12px]" style="color: #ff9500">
                未启用任何模型，请在卡片右侧打开开关并设为默认。
              </a-typography-text>
            </div>
          </a-space>
        </a-card>
      </a-col>
    </a-row>

    <a-modal
      v-model:visible="showModal"
      modal-class="tp-cabin-modal"
      body-class="tp-cabin-body-modal"
      :width="480"
      align-center
      :mask-closable="false"
      :esc-to-close="true"
      unmount-on-close
    >
      <template #title>
        <span class="tp-title">{{ isEdit ? '编辑模型' : '添加模型' }}</span>
      </template>

      <div class="tp-cabin">
        <div class="tp-seg">
          <span class="tp-seg-thumb" :class="{ right: form.mode === 'custom' }" />
          <button
            type="button"
            class="tp-seg-btn"
            :class="{ active: form.mode === 'provider' }"
            @click="setMode('provider')"
          >
            模型服务商
          </button>
          <button
            type="button"
            class="tp-seg-btn"
            :class="{ active: form.mode === 'custom' }"
            @click="setMode('custom')"
          >
            自定义配置
          </button>
        </div>

        <template v-if="form.mode === 'provider'">
          <div class="tp-row">
            <label class="tp-label"><i class="tp-req">*</i>服务商</label>
            <a-select
              v-model="form.provider"
              :options="providerOptions"
              placeholder="选择模型服务商"
              @change="onProviderChange"
            />
          </div>

          <div class="tp-row">
            <label class="tp-label"><i class="tp-req">*</i>模型</label>
            <a-select
              v-model="form.model"
              placeholder="选择模型"
              allow-search
              allow-create
              @change="onModelChange"
            >
              <a-select-opt-group v-if="presetModels[form.provider].length" label="预设模型">
                <a-option v-for="m in presetModels[form.provider]" :key="m" :value="m">
                  {{ m
                  }}<span class="tp-opt-meta"
                    >{{ getModelContext(m).i.toLocaleString() }} tokens</span
                  >
                </a-option>
              </a-select-opt-group>
              <a-select-opt-group v-if="fetchedModels.length" label="API 拉取">
                <a-option v-for="m in fetchedModels" :key="m" :value="m">
                  {{ m
                  }}<span class="tp-opt-meta"
                    >{{ getModelContext(m).i.toLocaleString() }} tokens</span
                  >
                </a-option>
              </a-select-opt-group>
            </a-select>
            <div class="tp-aux">
              <span class="tp-aux-hint">支持搜索，未列出的模型可直接输入 ID 创建</span>
              <button
                v-if="form.apiKey"
                type="button"
                class="tp-aux-link"
                :disabled="fetching"
                @click="fetchProviderModels"
              >
                <IconRefresh :class="{ 'tp-spin': fetching }" />{{
                  fetching ? '拉取中' : '拉取列表'
                }}
              </button>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="tp-row">
            <label class="tp-label"><i class="tp-req">*</i>API 格式</label>
            <a-select v-model="form.apiFormat" :options="apiFormatOptions" />
          </div>

          <div class="tp-row">
            <div class="tp-labelrow">
              <label class="tp-label"><i class="tp-req">*</i>自定义请求地址</label>
              <span class="tp-inline">
                <IconLink :size="14" />完整 URL
                <button
                  type="button"
                  role="switch"
                  class="tp-toggle"
                  :class="{ on: form.fullUrl }"
                  :aria-checked="form.fullUrl"
                  @click="form.fullUrl = !form.fullUrl"
                >
                  <span class="tp-knob" />
                </button>
              </span>
            </div>
            <input
              v-model="form.baseUrl"
              class="tp-field"
              placeholder="e.g. https://api.openai.com/v1"
            />
            <div class="tp-banner">
              <IconInfoCircleFill class="tp-banner-i" />
              <p>{{ bannerText }}</p>
            </div>
          </div>

          <div class="tp-row">
            <div class="tp-labelrow">
              <label class="tp-label"><i class="tp-req">*</i>模型 ID</label>
              <span class="tp-inline">
                多模态
                <button
                  type="button"
                  role="switch"
                  class="tp-toggle"
                  :class="{ on: form.multimodal }"
                  :aria-checked="form.multimodal"
                  @click="form.multimodal = !form.multimodal"
                >
                  <span class="tp-knob" />
                </button>
              </span>
            </div>
            <a-select
              v-model="form.model"
              placeholder="直接输入模型 ID，或拉取列表后选择"
              allow-search
              allow-create
              @change="onModelChange"
            >
              <a-select-opt-group v-if="fetchedModels.length" label="API 拉取">
                <a-option v-for="m in fetchedModels" :key="m" :value="m">
                  {{ m
                  }}<span class="tp-opt-meta"
                    >{{ getModelContext(m).i.toLocaleString() }} tokens</span
                  >
                </a-option>
              </a-select-opt-group>
            </a-select>
            <div class="tp-aux">
              <span class="tp-aux-hint">支持直接输入任意模型 ID，也可通过 API 密钥拉取后选择</span>
              <button
                v-if="form.apiKey && form.baseUrl"
                type="button"
                class="tp-aux-link"
                :disabled="fetching"
                @click="fetchCustomModels"
              >
                <IconRefresh :class="{ 'tp-spin': fetching }" />{{
                  fetching ? '拉取中' : '拉取列表'
                }}
              </button>
            </div>
          </div>
        </template>

        <div class="tp-row">
          <label class="tp-label"><i class="tp-req">*</i>API 密钥</label>
          <input
            v-model="form.apiKey"
            class="tp-field"
            type="password"
            placeholder="输入 API 密钥"
          />
        </div>

        <button type="button" class="tp-collapse-hd" @click="advanced = !advanced">
          <IconDown class="tp-chev" :class="{ open: advanced }" />高级配置
        </button>

        <div class="tp-collapse-bd" :class="{ open: advanced }">
          <div class="tp-collapse-inner">
            <div v-if="form.mode === 'custom'" class="tp-row">
              <label class="tp-label">模型系列</label>
              <p class="tp-hint">针对特定模型系列优化了 Prompt 和超参，未选择时使用默认配置。</p>
              <a-select v-model="form.modelSeries" :options="modelSeriesOptions" />
            </div>

            <div class="tp-row">
              <label class="tp-label">模型展示名称</label>
              <p class="tp-hint">在模型列表中展示的名称，未设置时默认显示 Model ID。</p>
              <div class="tp-countwrap">
                <input
                  v-model="form.displayName"
                  class="tp-field tp-count-input"
                  maxlength="32"
                  placeholder="请输入模型展示名称"
                />
                <span class="tp-count">{{ (form.displayName || '').length }}/32</span>
              </div>
            </div>

            <div class="tp-row">
              <label class="tp-label">上下文窗口</label>
              <div class="tp-ctx">
                <span class="tp-ctx-k">输入</span>
                <input v-model.number="form.contextInput" type="number" class="tp-field tp-ctx-f" />
                <span class="tp-ctx-k">输出</span>
                <input
                  v-model.number="form.contextOutput"
                  type="number"
                  class="tp-field tp-ctx-f"
                />
              </div>
            </div>

            <div class="tp-row">
              <label class="tp-label">工具调用轮次</label>
              <input v-model.number="form.toolCallRounds" type="number" class="tp-field" />
            </div>

            <div class="tp-row last">
              <label class="tp-label">月度配额</label>
              <p class="tp-hint">每月可用 token 上限，用于用量统计与告警。</p>
              <input v-model.number="form.monthlyQuota" type="number" class="tp-field" />
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <button type="button" class="tp-submit" @click="save">
          {{ isEdit ? '保存配置' : '添加模型' }}
        </button>
      </template>
    </a-modal>
  </div>
</template>

<style>
.arco-modal.tp-cabin-modal {
  display: flex;
  flex-direction: column;
  max-height: 88vh;
  padding: 0;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(28px) saturate(180%);
  -webkit-backdrop-filter: blur(28px) saturate(180%);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 20px;
  box-shadow:
    0 32px 80px -20px rgba(0, 0, 0, 0.25),
    0 0 0 0.5px rgba(0, 0, 0, 0.03);
  overflow: hidden;
}
.arco-modal.tp-cabin-modal .arco-modal-header {
  flex: none;
  padding: 20px 24px 16px;
  background: transparent;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}
.arco-modal.tp-cabin-modal .arco-modal-title {
  color: #1d1d1f;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 0.2px;
}
.arco-modal.tp-cabin-modal .arco-modal-content {
  padding: 0;
  background: transparent;
}
.arco-modal.tp-cabin-modal .arco-modal-body {
  flex: 1 1 auto;
  min-height: 0;
  padding: 0 !important;
  overflow-y: auto;
}
.arco-modal.tp-cabin-modal .arco-modal-footer {
  flex: none;
  padding: 16px 24px 20px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0) 0%, rgba(255, 255, 255, 0.96) 38%);
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}
.arco-modal.tp-cabin-modal .arco-modal-close-btn {
  color: #86868b;
  border-radius: 9px;
  transition:
    background 0.18s,
    color 0.18s;
}
.arco-modal.tp-cabin-modal .arco-modal-close-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #1d1d1f;
}

.tp-cabin .arco-select {
  width: 100%;
}
.tp-cabin .arco-select-view {
  height: 42px;
  background: #f5f5f7;
  border: 1px solid rgba(0, 0, 0, 0.07);
  border-radius: 10px;
  transition:
    border-color 0.18s,
    box-shadow 0.18s,
    background 0.18s;
}
.tp-cabin .arco-select-view:hover {
  border-color: rgba(0, 0, 0, 0.14);
}
.tp-cabin .arco-select-view.arco-select-view-focus {
  background: #fff;
  border-color: #007aff;
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.16);
}
.tp-cabin .arco-select-view-value,
.tp-cabin .arco-select-view-input {
  color: #1d1d1f;
}
.tp-cabin .arco-select-view-input::placeholder,
.tp-cabin .arco-select-placeholder {
  color: #aeaeb2;
}
.tp-cabin .arco-select-view-suffix .arco-icon {
  color: #86868b;
}

body.tp-cabin-open .arco-select-dropdown {
  background: rgba(255, 255, 255, 0.98);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  padding: 6px;
  box-shadow: 0 20px 50px -16px rgba(0, 0, 0, 0.22);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}
body.tp-cabin-open .arco-select-option {
  color: #1d1d1f;
  border-radius: 8px;
}
body.tp-cabin-open .arco-select-option-hover,
body.tp-cabin-open .arco-select-option:hover {
  background: #f5f5f7;
}
body.tp-cabin-open .arco-select-option-selected {
  background: rgba(0, 122, 255, 0.1);
  color: #007aff;
}
body.tp-cabin-open .arco-select-group-title {
  color: #aeaeb2;
  font-size: 11px;
  letter-spacing: 0.6px;
  text-transform: uppercase;
}
body.tp-cabin-open .tp-opt-meta {
  color: #aeaeb2;
}
</style>

<style scoped>
.tp-title {
  display: inline-block;
}

.tp-cabin {
  padding: 22px 24px 6px;
}

.tp-seg {
  position: relative;
  display: flex;
  padding: 3px;
  margin-bottom: 22px;
  background: rgba(0, 0, 0, 0.055);
  border-radius: 11px;
}
.tp-seg-thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: calc(50% - 3px);
  height: calc(100% - 6px);
  background: #fff;
  border-radius: 9px;
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.1),
    0 0 0 0.5px rgba(0, 0, 0, 0.04);
  transition: transform 0.32s cubic-bezier(0.34, 1.4, 0.5, 1);
}
.tp-seg-thumb.right {
  transform: translateX(100%);
}
.tp-seg-btn {
  position: relative;
  z-index: 1;
  flex: 1;
  height: 32px;
  border: none;
  background: transparent;
  color: #86868b;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.2s;
}
.tp-seg-btn.active {
  color: #1d1d1f;
}

.tp-row {
  margin-bottom: 18px;
}
.tp-row.last {
  margin-bottom: 4px;
}

.tp-label {
  display: block;
  margin-bottom: 9px;
  color: #1d1d1f;
  font-size: 13.5px;
  font-weight: 600;
  letter-spacing: 0.1px;
}
.tp-req {
  color: #ff3b30;
  margin-right: 5px;
  font-style: normal;
}
.tp-hint {
  margin: -2px 0 10px;
  color: #86868b;
  font-size: 12.5px;
  line-height: 1.5;
}

.tp-labelrow {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 9px;
}
.tp-labelrow .tp-label {
  margin-bottom: 0;
}
.tp-inline {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #86868b;
  font-size: 13px;
}

.tp-field {
  width: 100%;
  height: 42px;
  padding: 0 13px;
  background: #f5f5f7;
  border: 1px solid rgba(0, 0, 0, 0.07);
  border-radius: 10px;
  color: #1d1d1f;
  font-size: 14px;
  outline: none;
  transition:
    border-color 0.18s,
    box-shadow 0.18s,
    background 0.18s;
}
.tp-field::placeholder {
  color: #aeaeb2;
}
.tp-field:hover {
  border-color: rgba(0, 0, 0, 0.14);
}
.tp-field:focus {
  background: #fff;
  border-color: #007aff;
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.16);
}
.tp-field[type='number'] {
  -moz-appearance: textfield;
}
.tp-field[type='number']::-webkit-outer-spin-button,
.tp-field[type='number']::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.tp-toggle {
  position: relative;
  width: 40px;
  height: 24px;
  padding: 0;
  border: none;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.16);
  cursor: pointer;
  transition: background 0.24s;
}
.tp-toggle.on {
  background: #34c759;
}
.tp-knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #fff;
  box-shadow:
    0 2px 5px rgba(0, 0, 0, 0.22),
    0 0 1px rgba(0, 0, 0, 0.12);
  transition: transform 0.26s cubic-bezier(0.34, 1.5, 0.5, 1);
}
.tp-toggle.on .tp-knob {
  transform: translateX(16px);
}

.tp-banner {
  display: flex;
  gap: 10px;
  margin-top: 12px;
  padding: 12px 14px;
  background: rgba(0, 122, 255, 0.07);
  border: 1px solid rgba(0, 122, 255, 0.2);
  border-radius: 11px;
}
.tp-banner-i {
  flex: none;
  margin-top: 1px;
  color: #007aff;
  font-size: 16px;
}
.tp-banner p {
  margin: 0;
  color: #4b6f9e;
  font-size: 12.5px;
  line-height: 1.55;
}

.tp-aux {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 8px;
}
.tp-aux-hint {
  color: #aeaeb2;
  font-size: 12px;
}
.tp-aux-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0;
  border: none;
  background: transparent;
  color: #007aff;
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.18s;
}
.tp-aux-link:hover:not(:disabled) {
  opacity: 0.72;
}
.tp-aux-link:disabled {
  color: #aeaeb2;
  cursor: default;
}

.tp-collapse-hd {
  display: flex;
  align-items: center;
  gap: 7px;
  width: 100%;
  padding: 6px 0 4px;
  margin: 4px 0 2px;
  border: none;
  background: transparent;
  color: #1d1d1f;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
}
.tp-chev {
  color: #86868b;
  transition: transform 0.28s;
}
.tp-chev.open {
  transform: rotate(180deg);
}
.tp-collapse-bd {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.32s cubic-bezier(0.4, 0, 0.2, 1);
}
.tp-collapse-bd.open {
  grid-template-rows: 1fr;
}
.tp-collapse-inner {
  overflow: hidden;
  min-height: 0;
  padding-top: 14px;
}

.tp-countwrap {
  position: relative;
}
.tp-count-input {
  padding-right: 56px;
}
.tp-count {
  position: absolute;
  top: 50%;
  right: 14px;
  transform: translateY(-50%);
  color: #aeaeb2;
  font-size: 12px;
  pointer-events: none;
}

.tp-ctx {
  display: flex;
  align-items: center;
  gap: 10px;
}
.tp-ctx-k {
  flex: none;
  color: #86868b;
  font-size: 13px;
}
.tp-ctx-f {
  flex: 1;
  min-width: 0;
}

.tp-submit {
  width: 100%;
  height: 44px;
  border: none;
  border-radius: 11px;
  background: linear-gradient(180deg, #2b8eff 0%, #007aff 100%);
  color: #fff;
  font-size: 15px;
  font-weight: 650;
  letter-spacing: 0.3px;
  cursor: pointer;
  box-shadow:
    0 6px 16px -6px rgba(0, 122, 255, 0.55),
    inset 0 1px 0 rgba(255, 255, 255, 0.25);
  transition:
    transform 0.12s,
    box-shadow 0.2s,
    filter 0.2s;
}
.tp-submit:hover {
  filter: brightness(1.06);
  box-shadow:
    0 10px 22px -8px rgba(0, 122, 255, 0.6),
    inset 0 1px 0 rgba(255, 255, 255, 0.25);
}
.tp-submit:active {
  transform: scale(0.99);
}

.tp-opt-meta {
  margin-left: 8px;
  font-size: 12px;
}

.tp-spin {
  animation: tp-rotate 0.8s linear infinite;
}
@keyframes tp-rotate {
  to {
    transform: rotate(360deg);
  }
}

.plan-card {
  position: relative;
  padding: 18px 20px 16px 22px;
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 16px;
  overflow: hidden;
  transition:
    transform 0.22s cubic-bezier(0.25, 0.1, 0.25, 1),
    box-shadow 0.22s;
}
.plan-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 16px 36px -18px rgba(0, 0, 0, 0.22);
}
.plan-card.active {
  border-color: rgba(0, 122, 255, 0.35);
  box-shadow: 0 12px 30px -18px rgba(0, 122, 255, 0.4);
}
.plan-accent {
  position: absolute;
  top: 0;
  left: 0;
  width: 4px;
  height: 100%;
  opacity: 0.85;
}
.plan-top {
  display: flex;
  align-items: center;
  gap: 14px;
}
.plan-badge {
  flex: none;
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  font-size: 16px;
}
.plan-meta {
  flex: 1;
  min-width: 0;
}
.plan-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1d1d1f;
  font-size: 15px;
  font-weight: 650;
}
.plan-live {
  padding: 1px 8px;
  border-radius: 999px;
  background: rgba(52, 199, 89, 0.14);
  color: #248a3d;
  font-size: 11px;
  font-weight: 600;
}
.plan-sub {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 3px;
  color: #86868b;
  font-size: 12.5px;
}
.plan-sub .dot {
  opacity: 0.5;
}
.chip-mm {
  padding: 1px 7px;
  border-radius: 999px;
  background: rgba(88, 86, 214, 0.12);
  color: #5856d6;
  font-size: 11px;
  font-weight: 600;
}
.plan-acts {
  display: flex;
  align-items: center;
  gap: 8px;
}
.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: #86868b;
  cursor: pointer;
  transition:
    background 0.18s,
    color 0.18s;
}
.icon-btn:hover:not(:disabled) {
  background: rgba(0, 0, 0, 0.05);
  color: #1d1d1f;
}
.icon-btn.danger:hover:not(:disabled) {
  background: rgba(255, 59, 48, 0.1);
  color: #ff3b30;
}
.icon-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.plan-info {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
}
.info-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #f5f5f7;
  border-radius: 8px;
  color: #1d1d1f;
  font-size: 12px;
}
.info-chip b {
  color: #86868b;
  font-weight: 500;
}
.info-chip em {
  font-style: normal;
  font-weight: 600;
}
.info-chip em.warn {
  color: #ff3b30;
}
.info-chip.right {
  margin-left: auto;
}
.plan-bar {
  height: 4px;
  margin-top: 12px;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 999px;
  overflow: hidden;
}
.plan-bar span {
  display: block;
  height: 100%;
  background: #34c759;
  border-radius: 999px;
  transition: width 0.6s cubic-bezier(0.25, 0.1, 0.25, 1);
}
.plan-bar span.warn {
  background: #ff3b30;
}

.empty-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 20px;
  background: #fff;
  border: 1px dashed rgba(0, 0, 0, 0.12);
  border-radius: 16px;
  text-align: center;
}
.empty-glyph {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: rgba(0, 122, 255, 0.1);
  color: #007aff;
  font-size: 26px;
  line-height: 48px;
}
.empty-card p {
  margin: 0;
  color: #86868b;
  font-size: 14px;
}
</style>
