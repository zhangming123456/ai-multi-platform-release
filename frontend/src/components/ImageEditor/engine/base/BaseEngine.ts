import type { BaseLayer, EngineCapabilities, EngineOptions } from '../../types'

export interface BaseEngine {
  init(): void
  destroy(): void
  setSize(width: number, height: number): void
  addLayer(layer: BaseLayer): void
  updateLayer(id: string, patch: Partial<BaseLayer>): void
  removeLayer(id: string): void
  render(): void
  exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob>
  clear(): void
  getCapabilities(): EngineCapabilities
  onLayerChange(callback: (id: string, patch: Partial<BaseLayer>) => void): void
}

export abstract class AbstractEngine implements BaseEngine {
  protected canvas: HTMLCanvasElement
  protected ctx: CanvasRenderingContext2D
  protected options: EngineOptions
  protected layerChangeCallback: ((id: string, patch: Partial<BaseLayer>) => void) | null = null

  constructor(canvas: HTMLCanvasElement, options: EngineOptions) {
    this.canvas = canvas
    this.options = options
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      throw new Error('Failed to get 2d context from canvas')
    }
    this.ctx = ctx
  }

  abstract init(): void
  abstract destroy(): void
  abstract setSize(width: number, height: number): void
  abstract addLayer(layer: BaseLayer): void
  abstract updateLayer(id: string, patch: Partial<BaseLayer>): void
  abstract removeLayer(id: string): void
  abstract render(): void
  abstract exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob>
  abstract clear(): void
  abstract getCapabilities(): EngineCapabilities

  onLayerChange(callback: (id: string, patch: Partial<BaseLayer>) => void): void {
    this.layerChangeCallback = callback
  }

  protected notifyLayerChange(id: string, patch: Partial<BaseLayer>): void {
    if (this.layerChangeCallback) {
      this.layerChangeCallback(id, patch)
    }
  }
}
