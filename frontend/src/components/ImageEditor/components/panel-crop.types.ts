import type { Viewport } from '../ImageEditor.types'
import type { CropRect } from '../utils/image.types'

export interface PanelCropProps {
  canvasWidth: number
  canvasHeight: number
  canvasEl: HTMLCanvasElement
  viewport: Viewport
}

export type PanelCropEmits = {
  (e: 'apply', rect: CropRect): void
  (e: 'cancel'): void
  (e: 'panImage', dx: number, dy: number): void
  (e: 'zoomImage', factor: number): void
}
