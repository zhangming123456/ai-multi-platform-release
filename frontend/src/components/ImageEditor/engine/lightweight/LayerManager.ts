import type { BaseLayer } from '../../ImageEditor.types'

export class LayerManager {
  private layers: Map<string, BaseLayer> = new Map()

  addLayer(layer: BaseLayer): void {
    this.layers.set(layer.id, layer)
  }

  updateLayer(id: string, patch: Partial<BaseLayer>): void {
    const layer = this.layers.get(id)
    if (!layer) return
    Object.assign(layer, patch)
  }

  removeLayer(id: string): void {
    this.layers.delete(id)
  }

  getLayer(id: string): BaseLayer | undefined {
    return this.layers.get(id)
  }

  getAllLayers(): BaseLayer[] {
    return Array.from(this.layers.values()).sort((a, b) => a.zIndex - b.zIndex)
  }

  clear(): void {
    this.layers.clear()
  }

  getLayerCount(): number {
    return this.layers.size
  }
}
