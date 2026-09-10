export interface SegmentedControlOption {
  key: string
  label: string
}

export interface SegmentedControlProps {
  options: SegmentedControlOption[]
  modelValue: string
}

export type SegmentedControlEmits = {
  'update:modelValue': [value: string]
}
