import { AbstractEngine } from '../base/BaseEngine'
import type { BaseLayer, EngineCapabilities } from '../../types'
import { LayerManager } from './LayerManager'
import { renderImage, loadImage } from './render/renderImage'
import { renderText } from './render/renderText'
import { renderDraw } from './render/renderDraw'
import { renderSticker } from './render/renderSticker'

export class CanvasEngine extends AbstractEngine {
  private layerManager: LayerManager
  private imageCache: Map<string, HTMLImageElement> = new Map()

  constructor(canvas: HTMLCanvasElement, width: number, height: number) {
    super(canvas, { width, height })
    this.layerManager = new LayerManager()
  }

  init(): void {
    this.canvas.width = this.options.width
    this.canvas.height = this.options.height
    this.render()
  }

  destroy(): void {
    this.layerManager.clear()
    this.imageCache.clear()
  }

  setSize(width: number, height: number): void {
    this.options.width = width
    this.options.height = height
    this.canvas.width = width
    this.canvas.height = height
    this.render()
  }

  addLayer(layer: BaseLayer): void {
    this.layerManager.addLayer(layer)

    if (layer.type === 'image' && layer.image?.src) {
      this.loadImageToCache(layer.image.src)
    } else if (layer.type === 'sticker' && layer.sticker?.url) {
      this.loadImageToCache(layer.sticker.url)
    }

    this.render()
  }

  updateLayer(id: string, patch: Partial<BaseLayer>): void {
    this.layerManager.updateLayer(id, patch)
    this.render()
  }

  removeLayer(id: string): void {
    this.layerManager.removeLayer(id)
    this.render()
  }

  render(): void {
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height)

    const layers = this.layerManager.getAllLayers()
    for (const layer of layers) {
      if (!layer.visible) continue

      switch (layer.type) {
        case 'image':
          renderImage(this.ctx, layer, this.imageCache)
          break
        case 'text':
          renderText(this.ctx, layer)
          break
        case 'draw':
          renderDraw(this.ctx, layer)
          break
        case 'sticker':
          renderSticker(this.ctx, layer, this.imageCache)
          break
      }
    }
  }

  async exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob> {
    return new Promise((resolve, reject) => {
      this.canvas.toBlob(
        (blob) => {
          if (blob) {
            resolve(blob)
          } else {
            reject(new Error('Failed to export canvas to blob'))
          }
        },
        `image/${format}`,
        quality,
      )
    })
  }

  clear(): void {
    this.layerManager.clear()
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height)
  }

  getCapabilities(): EngineCapabilities {
    return {
      multiSelect: false,
      group: false,
      lockLayer: false,
      viewportZoom: false,
      svgImport: false,
      freeTransform: false,
      doubleClickEdit: false,
    }
  }

  private async loadImageToCache(src: string): Promise<void> {
    if (this.imageCache.has(src)) return

    try {
      const img = await loadImage(src)
      this.imageCache.set(src, img)
      this.render()
    } catch (error) {
      console.error('Failed to load image:', src, error)
    }
  }
}
