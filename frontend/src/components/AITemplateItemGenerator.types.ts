export interface AIGeneratedItem {
  category: string
  title: string
  standard: string
  standard_images: string[]
  score_type: 'score' | 'pass_fail'
  max_score: number
  score_options: { score: number; label: string }[]
}

export interface AITemplateItemGeneratorProps {
  visible: boolean
  existingCategories: string[]
}

export interface AITemplateItemGeneratorConfirmResult {
  name: string
  description: string
  items: AIGeneratedItem[]
}

export type AITemplateItemGeneratorEmits = {
  (e: 'update:visible', value: boolean): void
  (e: 'confirm', result: AITemplateItemGeneratorConfirmResult): void
}
