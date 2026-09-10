import type { BaseLayer } from '../ImageEditor.types'

export interface PanelTextProps {
  editLayer?: BaseLayer | null
}

export interface TextLayerPayload {
  content: string
  fontSize: number
  color: string
  bold: boolean
}

export type PanelTextEmits = {
  (e: 'add', options: TextLayerPayload): void
  (e: 'update', id: string, options: TextLayerPayload): void
}
