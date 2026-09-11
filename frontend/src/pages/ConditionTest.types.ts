import type { ConditionGroup } from '@/components/ConditionBuilder/ConditionBuilder.types'

export type { ConditionRuleState } from '@/components/ConditionBuilder/ConditionBuilder.types'

export interface ConditionPreset {
  key: string
  label: string
  group: ConditionGroup
}

export interface ConditionDataPreset {
  key: string
  label: string
  data: Record<string, unknown>
}

export interface ConditionSchemaField {
  name: string
  type: string
  desc: string
}
