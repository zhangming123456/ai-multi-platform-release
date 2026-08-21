import type {
  BaseLayer,
  EditorMode,
  EngineCapabilities,
  EngineOptions,
  Viewport,
} from '../../types'

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

export abstract class AbstractEngine implements BaseEngine {
  protected canvas: HTMLCanvasElement
  protected ctx: CanvasRenderingContext2D
  protected options: EngineOptions
  protected layerChangeCallback: ((id: string, patch: Partial<BaseLayer>) => void) | null = null
  protected pendingOps = new Set<Promise<unknown>>()

  constructor(canvas: HTMLCanvasElement, options: EngineOptions) {
    this.canvas = canvas
    this.options = options
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      throw new Error('Failed to get 2d context from canvas')
    }
    this.ctx = ctx
  }

  protected track<T>(promise: Promise<T>): Promise<T> {
    this.pendingOps.add(promise)
    const remove = (): void => {
      this.pendingOps.delete(promise)
    }
    promise.then(remove, remove)
    return promise
  }

  async whenReady(): Promise<void> {
    while (this.pendingOps.size > 0) {
      await Promise.allSettled([...this.pendingOps])
    }
  }

  abstract init(): void
  abstract destroy(): void
  abstract setSize(width: number, height: number): void
  abstract addLayer(layer: BaseLayer): void | Promise<void>
  abstract updateLayer(id: string, patch: Partial<BaseLayer>): void
  abstract removeLayer(id: string): void
  abstract render(): void
  abstract exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob>
  abstract clear(): void
  abstract getCapabilities(): EngineCapabilities

  onLayerChange(callback: (id: string, patch: Partial<BaseLayer>) => void): void {
    this.layerChangeCallback = callback
  }

  setMode(_mode: EditorMode): void {}

  setDrawStyle(_color: string, _width: number): void {}

  onDrawComplete(_callback: (points: number[][], color: string, width: number) => void): void {}

  onSelectionChange(_callback: (ids: string[]) => void): void {}

  onDoubleClickEdit(_callback: (id: string) => void): void {}

  protected notifyLayerChange(id: string, patch: Partial<BaseLayer>): void {
    if (this.layerChangeCallback) {
      this.layerChangeCallback(id, patch)
    }
  }
}
