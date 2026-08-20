import { Canvas, Image as FabricImage, type FabricObject } from 'fabric'
import { AbstractEngine } from '../base/BaseEngine'
import type { BaseLayer, EngineCapabilities } from '../../types'
import { toFabricObject } from './FabricAdapter'
import { bindFabricEvents } from './FabricEvent'

export class FabricEngine extends AbstractEngine {
  private fabricCanvas: Canvas | null = null
  private layers: Map<string, BaseLayer> = new Map()

  constructor(canvas: HTMLCanvasElement, width: number, height: number) {
    super(canvas, { width, height })
  }

  init(): void {
    this.fabricCanvas = new Canvas(this.canvas, {
      width: this.options.width,
      height: this.options.height,
      preserveObjectStacking: true,
    })

    bindFabricEvents(this.fabricCanvas, (id, patch) => {
      const layer = this.layers.get(id)
      if (layer) {
        Object.assign(layer, patch)
        this.notifyLayerChange(id, patch)
      }
    })
  }

  destroy(): void {
    if (this.fabricCanvas) {
      this.fabricCanvas.dispose()
      this.fabricCanvas = null
    }
    this.layers.clear()
  }

  setSize(width: number, height: number): void {
    if (!this.fabricCanvas) return
    this.options.width = width
    this.options.height = height
    this.fabricCanvas.setDimensions({ width, height })
    this.render()
  }

  addLayer(layer: BaseLayer): void {
    if (!this.fabricCanvas) return

    this.layers.set(layer.id, layer)

    const fabricObj = toFabricObject(layer)
    if (fabricObj) {
      ;(fabricObj as FabricObject & { id?: string }).id = layer.id
      this.fabricCanvas.add(fabricObj)
      this.fabricCanvas.requestRenderAll()
    }
  }

  updateLayer(id: string, patch: Partial<BaseLayer>): void {
    if (!this.fabricCanvas) return

    const layer = this.layers.get(id)
    if (!layer) return

    Object.assign(layer, patch)

    const fabricObj = this.fabricCanvas
      .getObjects()
      .find((obj) => (obj as FabricObject & { id?: string }).id === id) as FabricObject | undefined

    if (fabricObj) {
      fabricObj.set({
        left: layer.x,
        top: layer.y,
        width: layer.width,
        height: layer.height,
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
      })
      this.fabricCanvas.requestRenderAll()
    }
  }

  removeLayer(id: string): void {
    if (!this.fabricCanvas) return

    const fabricObj = this.fabricCanvas
      .getObjects()
      .find((obj) => (obj as FabricObject & { id?: string }).id === id)
    if (fabricObj) {
      this.fabricCanvas.remove(fabricObj)
      this.fabricCanvas.requestRenderAll()
    }
    this.layers.delete(id)
  }

  render(): void {
    if (!this.fabricCanvas) return
    this.fabricCanvas.requestRenderAll()
  }

  async exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob> {
    if (!this.fabricCanvas) {
      throw new Error('Fabric canvas not initialized')
    }

    const dataURL = this.fabricCanvas.toDataURL({
      format,
      quality: quality ?? 0.92,
      multiplier: 1,
    })

    const response = await fetch(dataURL)
    return response.blob()
  }

  clear(): void {
    if (!this.fabricCanvas) return
    this.fabricCanvas.remove(...this.fabricCanvas.getObjects())
    this.fabricCanvas.requestRenderAll()
    this.layers.clear()
  }

  getCapabilities(): EngineCapabilities {
    return {
      multiSelect: true,
      group: true,
      lockLayer: true,
      viewportZoom: true,
      svgImport: true,
      freeTransform: true,
      doubleClickEdit: true,
    }
  }
}
