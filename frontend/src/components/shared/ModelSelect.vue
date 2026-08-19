<template>
  <a-select
    v-model="selectedKey"
    placeholder="选择模型"
    class="w-full"
    :size="size"
    allow-clear
    :trigger-props="{
      popupStyle: {
        width: '300px',
      },
    }"
    @change="handleChange"
  >
    <a-option
      v-for="opt in options"
      :key="opt.key"
      :value="opt.key"
      :disabled="requireVision && !opt.hasVision"
    >
      <span class="provider-opt">
        <span>{{ opt.planName }} · {{ opt.modelId }}</span>
        <span v-if="requireVision && !opt.hasVision" class="text-[#FF3B30] text-[12px]">
          （不支持视觉）
        </span>
      </span>
    </a-option>
  </a-select>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, unref } from 'vue'
import { useTokenPlanStore, parseModelField } from '@/stores/tokenPlan'

export interface ModelSelectValue {
  planId: string
  modelId: string
  hasVision: boolean
}

interface ModelOption {
  key: string
  planName: string
  modelId: string
  hasVision: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue?: string
    autoSelect?: boolean
    requireVision?: boolean
    size?: 'mini' | 'small' | 'medium' | 'large'
  }>(),
  { modelValue: '', autoSelect: true, requireVision: true, size: 'medium' },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: ModelSelectValue): void
}>()

const tokenPlanStore = useTokenPlanStore()

const options = computed<ModelOption[]>(() => {
  const list: ModelOption[] = []
  for (const p of tokenPlanStore.enabledPlans) {
    const planName = p.displayName || p.name
    for (const m of parseModelField(p.model)) {
      if (!m.id) continue
      list.push({
        key: `${p.id}:${m.id}`,
        planName,
        modelId: m.id,
        hasVision: m.types.includes('vision'),
      })
    }
  }
  return list
})

const selectedKey = ref(props.modelValue || '')

watch(
  () => props.modelValue,
  (val) => {
    selectedKey.value = val
    handlePropsChange()
  },
)

function handlePropsChange() {
  const list = unref(options)
  if (!props.autoSelect) return
  if (props.requireVision) {
    if (list.some((o) => o.hasVision && o.key === unref(selectedKey))) return
    const first = list.find((o) => o.hasVision)
    if (first) select(first)
  } else if (list.length > 0) {
    if (list.some((o) => o.key === unref(selectedKey))) return
    select(list[0])
  }
}

watch(options, handlePropsChange, { immediate: true })
watch(() => props.requireVision, handlePropsChange, { immediate: true })

function parseKey(key: string): { planId: string; modelId: string } {
  const idx = key.indexOf(':')
  if (idx <= 0) return { planId: '', modelId: '' }
  return { planId: key.slice(0, idx), modelId: key.slice(idx + 1) }
}

function select(opt: ModelOption) {
  selectedKey.value = opt.key
  emit('update:modelValue', opt.key)
  const { planId, modelId } = parseKey(opt.key)
  emit('change', { planId, modelId, hasVision: opt.hasVision })
}

function handleChange(val: unknown) {
  if (typeof val !== 'string') return
  if (!val) {
    selectedKey.value = ''
    emit('update:modelValue', '')
    emit('change', { planId: '', modelId: '', hasVision: false })
    return
  }
  const opt = options.value.find((o) => o.key === val)
  if (opt) {
    select(opt)
  } else {
    const { planId, modelId } = parseKey(val)
    emit('update:modelValue', val)
    emit('change', { planId, modelId, hasVision: false })
  }
}

onMounted(() => {
  tokenPlanStore.loadPlans()
})
</script>
<style scoped lang="scss">
:global(.arco-select-dropdown:has(.provider-opt)) {
  min-width: 300px;
  width: max-content !important;
  max-width: min(560px, 92vw);
  //background: #2c2c2e;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.5),
    0 2px 8px rgba(0, 0, 0, 0.3);
  border-radius: 12px;
}
</style>
