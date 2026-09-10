export interface ModelSelectValue {
  planId: string
  modelId: string
  hasVision: boolean
}

export interface ModelOption {
  key: string
  planName: string
  modelId: string
  hasVision: boolean
}

export interface ModelSelectProps {
  modelValue?: string
  autoSelect?: boolean
  requireVision?: boolean
  size?: 'mini' | 'small' | 'medium' | 'large'
}

export type ModelSelectEmits = {
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: ModelSelectValue): void
}
