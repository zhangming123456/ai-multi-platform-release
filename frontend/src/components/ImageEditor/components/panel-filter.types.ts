import type { FilterType } from '../utils/image.types'

export type PanelFilterEmits = {
  (e: 'apply', type: FilterType): void
}
