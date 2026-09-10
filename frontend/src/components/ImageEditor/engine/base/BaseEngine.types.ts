import type { BaseLayer, EditorMode, EngineCapabilities, Viewport } from '../../ImageEditor.types'

export interface BaseEngine {
  init(): void
  destroy(): void
  setSize(width: number, height: number): void
  addLayer(layer: BaseLayer): void | Promise<void>
  updateLayer(id: string, patch: Partial<BaseLayer>): void
  removeLayer(id: string): void
  render(): void
  exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob>
  clear(): void
  whenReady(): Promise<void>
  getCapabilities(): EngineCapabilities
  onLayerChange(callback: (id: string, patch: Partial<BaseLayer>) => void): void
  setMode?(mode: EditorMode): void
  setDrawStyle?(color: string, width: number): void
  onDrawComplete?(callback: (points: number[][], color: string, width: number) => void): void
  setViewport?(viewport: Partial<Viewport>): void
  getViewport?(): Viewport
  zoomAt?(screenX: number, screenY: number, factor: number): void
  fitToViewport?(): void
  resizeBuffer?(width: number, height: number): void
  onSelectionChange?(callback: (ids: string[]) => void): void
  onDoubleClickEdit?(callback: (id: string) => void): void
}
