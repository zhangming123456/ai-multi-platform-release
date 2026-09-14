import type {
  ConditionEvaluation,
  ConditionGroup,
} from '@/components/ConditionBuilder/ConditionBuilder.types'

export type {
  ConditionDocTabKey,
  ConditionOperatorDoc,
  ConditionRuleState,
  ConditionSchemaField,
} from '@/components/ConditionBuilder/ConditionBuilder.types'

export interface ConditionPreset {
  key: string
  label: string
  group: ConditionGroup
  data?: Record<string, unknown>[]
}

export interface ConditionDataPreset {
  key: string
  label: string
  data: Record<string, unknown>
}

export interface ConditionDataResult {
  key: string
  label: string
  evaluation: ConditionEvaluation
}
