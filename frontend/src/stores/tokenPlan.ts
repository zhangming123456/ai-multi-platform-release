import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import api from '@/utils/api'

export type PlanProvider = 'openai' | 'deepseek' | 'moonshot' | 'zhipu' | 'custom'
export type PlanMode = 'provider' | 'custom'

export interface TokenPlan {
  id: string
  name: string
  displayName: string
  provider: PlanProvider
  mode: PlanMode
  apiFormat: string
  apiKey: string
  baseUrl: string
  fullUrl: boolean
  model: string
  multimodal: boolean
  modelSeries: string
  contextInput: number
  contextOutput: number
  toolCallRounds: number
  enabled: boolean
  monthlyQuota: number
  usedTokens: number
}

export function providerLabel(provider: PlanProvider): string {
  switch (provider) {
    case 'openai':
      return 'OpenAI'
    case 'deepseek':
      return 'DeepSeek'
    case 'moonshot':
      return 'Moonshot'
    case 'zhipu':
      return '智谱 AI'
    default:
      return '自定义'
  }
}

function snakeToCamel(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const key of Object.keys(obj)) {
    const camelKey = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase())
    result[camelKey] = obj[key]
  }
  return result
}

function camelToSnake(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const key of Object.keys(obj)) {
    const snakeKey = key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`)
    result[snakeKey] = obj[key]
  }
  return result
}

export const useTokenPlanStore = defineStore('tokenPlan', () => {
  const plans = ref<TokenPlan[]>([])
  const activePlanId = ref('')
  const loaded = ref(false)

  const activePlan = computed(() => {
    return plans.value.find((p) => p.id === activePlanId.value && p.enabled) || null
  })

  const enabledPlans = computed(() => plans.value.filter((p) => p.enabled))

  let nextId = 1
  function generateId() {
    return `plan-${Date.now()}-${nextId++}`
  }

  async function loadPlans() {
    try {
      const res = await api.get('/model-configs')
      const list: TokenPlan[] = (res.data.data || []).map((item: Record<string, unknown>) => {
        const mapped = snakeToCamel(item) as unknown as TokenPlan
        return mapped
      })
      plans.value = list
      if (list.length > 0 && !activePlan.value) {
        activePlanId.value = enabledPlans.value[0]?.id || list[0].id
      }
      loaded.value = true
    } catch (e) {
      loaded.value = true
      Message.error('加载模型配置失败')
      console.error('loadPlans error:', e)
    }
  }

  async function addPlan(data?: Partial<TokenPlan>) {
    const provider = data?.provider || 'openai'
    const model =
      data?.model ||
      (provider === 'openai' ? 'gpt-4o-mini' : provider === 'deepseek' ? 'deepseek-chat' : '')
    const displayName = data?.displayName || model
    const plan: TokenPlan = {
      id: data?.id || generateId(),
      name: data?.name || displayName || providerLabel(provider),
      displayName,
      provider,
      mode: data?.mode || (provider === 'custom' ? 'custom' : 'provider'),
      apiFormat: data?.apiFormat || 'openai_chat',
      apiKey: data?.apiKey || '',
      baseUrl: data?.baseUrl || '',
      fullUrl: data?.fullUrl ?? false,
      model,
      multimodal: data?.multimodal ?? false,
      modelSeries: data?.modelSeries || 'default',
      contextInput: data?.contextInput || 128000,
      contextOutput: data?.contextOutput || 4096,
      toolCallRounds: data?.toolCallRounds ?? 200,
      enabled: data?.enabled ?? false,
      monthlyQuota: data?.monthlyQuota || 1000000,
      usedTokens: data?.usedTokens || 0,
    }
    try {
      await api.post('/model-configs', camelToSnake(plan as unknown as Record<string, unknown>))
    } catch (e) {
      Message.error('保存模型配置失败')
      console.error('addPlan error:', e)
      throw e
    }
    plans.value.push(plan)
    return plan
  }

  async function removePlan(id: string) {
    try {
      await api.delete(`/model-configs/${id}`)
    } catch (e) {
      Message.error('删除模型配置失败')
      console.error('removePlan error:', e)
      throw e
    }
    const idx = plans.value.findIndex((p) => p.id === id)
    if (idx !== -1) {
      plans.value.splice(idx, 1)
      if (activePlanId.value === id) {
        activePlanId.value = enabledPlans.value[0]?.id || ''
      }
    }
  }

  async function setPlanEnabled(id: string, enabled: boolean) {
    const plan = plans.value.find((p) => p.id === id)
    if (plan) {
      plan.enabled = enabled
      try {
        await api.put(`/model-configs/${id}`, { enabled })
      } catch (e) {
        plan.enabled = !enabled
        Message.error('更新模型配置状态失败')
        console.error('setPlanEnabled error:', e)
        throw e
      }
      if (enabled && !activePlan.value) {
        activePlanId.value = id
      } else if (!enabled && activePlanId.value === id) {
        activePlanId.value = enabledPlans.value[0]?.id || ''
      }
    }
  }

  async function updatePlan(id: string, updates: Partial<TokenPlan>) {
    const plan = plans.value.find((p) => p.id === id)
    if (plan) {
      const oldValues = { ...plan }
      Object.assign(plan, updates)
      try {
        await api.put(`/model-configs/${id}`, camelToSnake(updates as unknown as Record<string, unknown>))
      } catch (e) {
        Object.assign(plan, oldValues)
        Message.error('更新模型配置失败')
        console.error('updatePlan error:', e)
        throw e
      }
    }
  }

  function setActivePlan(id: string) {
    const plan = plans.value.find((p) => p.id === id)
    if (plan && plan.enabled) {
      activePlanId.value = id
    }
  }

  function incrementUsage(tokens: number) {
    if (activePlan.value) {
      activePlan.value.usedTokens += tokens
    }
  }

  function getRemainingQuota() {
    if (!activePlan.value) return 0
    return activePlan.value.monthlyQuota - activePlan.value.usedTokens
  }

  return {
    plans,
    activePlanId,
    activePlan,
    enabledPlans,
    loaded,
    loadPlans,
    addPlan,
    removePlan,
    setPlanEnabled,
    updatePlan,
    setActivePlan,
    incrementUsage,
    getRemainingQuota,
  }
})
