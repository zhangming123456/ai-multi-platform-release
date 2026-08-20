import { Canvas, Point, Path as FabricPath, PencilBrush, type FabricObject } from 'fabric'
import { AbstractEngine } from '../base/BaseEngine'
import type { BaseLayer, EditorMode, EngineCapabilities, Viewport } from '../../types'
import { toFabricObject } from './FabricAdapter'
import { bindFabricEvents } from './FabricEvent'

const MIN_SCALE = 0.1
const MAX_SCALE = 8
const ZOOM_STEP = 1.5

export class FabricEngine extends AbstractEngine {
  private fabricCanvas: Canvas | null = null
  private layers: Map<string, BaseLayer> = new Map()
  private drawColor = '#007aff'
  private drawWidth = 4
  private drawCompleteCallback:
    ((points: number[][], color: string, width: number) => void) | null = null
  private generation = 0

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

    const brush = new PencilBrush(this.fabricCanvas)
    brush.color = this.drawColor
    brush.width = this.drawWidth
    this.fabricCanvas.freeDrawingBrush = brush

    this.fabricCanvas.on('path:created', (e) => {
      const path = e.path as FabricPath
      if (!path || !this.drawCompleteCallback) return
      const points = this.extractPathPoints(path)
      this.fabricCanvas?.remove(path)
      this.fabricCanvas?.requestRenderAll()
      if (points.length >= 2) {
        this.drawCompleteCallback(points, this.drawColor, this.drawWidth)
      }
    })

    this.canvas.addEventListener('wheel', this.handleWheel, { passive: false })
  }

  destroy(): void {
    this.canvas.removeEventListener('wheel', this.handleWheel)
    if (this.fabricCanvas) {
      this.fabricCanvas.dispose()
      this.fabricCanvas = null
    }
    this.layers.clear()
  }

  setSize(width: number, height: number): void {
    this.options.width = width
    this.options.height = height
    this.render()
  }

  resizeBuffer(width: number, height: number): void {
    if (!this.fabricCanvas) return
    this.fabricCanvas.setDimensions({
      width: Math.max(1, Math.floor(width)),
      height: Math.max(1, Math.floor(height)),
    })
  }

  fitToViewport(): void {
    if (!this.fabricCanvas) return
    const worldW = this.options.width
    const worldH = this.options.height
    if (worldW <= 0 || worldH <= 0) return
    const bufferWidth = this.fabricCanvas.getWidth()
    const bufferHeight = this.fabricCanvas.getHeight()
    const scale = clamp(Math.min(bufferWidth / worldW, bufferHeight / worldH), MIN_SCALE, MAX_SCALE)
    this.fabricCanvas.setViewportTransform([
      scale,
      0,
      0,
      scale,
      (bufferWidth - worldW * scale) / 2,
      (bufferHeight - worldH * scale) / 2,
    ])
  }

  setViewport(viewport: Partial<Viewport>): void {
    if (!this.fabricCanvas) return
    const current = this.getViewport()
    const scale = clamp(viewport.scale ?? current.scale, MIN_SCALE, MAX_SCALE)
    const offsetX = viewport.offsetX ?? current.offsetX
    const offsetY = viewport.offsetY ?? current.offsetY
    this.fabricCanvas.setViewportTransform([scale, 0, 0, scale, offsetX, offsetY])
  }

  getViewport(): Viewport {
    const vt = this.fabricCanvas?.viewportTransform
    if (!vt) return { scale: 1, offsetX: 0, offsetY: 0 }
    return { scale: vt[0], offsetX: vt[4], offsetY: vt[5] }
  }

  zoomAt(screenX: number, screenY: number, factor: number): void {
    if (!this.fabricCanvas) return
    const nextScale = clamp(this.fabricCanvas.getZoom() * factor, MIN_SCALE, MAX_SCALE)
    this.fabricCanvas.zoomToPoint(new Point(screenX, screenY), nextScale)
  }

  async addLayer(layer: BaseLayer): Promise<void> {
    if (!this.fabricCanvas) return

    this.layers.set(layer.id, layer)

    const gen = this.generation
    const fabricObj = await this.track(toFabricObject(layer))
    if (this.generation !== gen || !fabricObj) return
    ;(fabricObj as FabricObject & { id?: string }).id = layer.id
    this.addObjectWithZIndex(layer, fabricObj)
    this.normalizePathPosition(fabricObj)
    this.fabricCanvas.requestRenderAll()
  }

  async updateLayer(id: string, patch: Partial<BaseLayer>): Promise<void> {
    if (!this.fabricCanvas) return

    const layer = this.layers.get(id)
    if (!layer) return

    Object.assign(layer, patch)

    const fabricObj = this.findObject(id)
    const currentSrc =
      layer.type === 'image'
        ? layer.image?.src
        : layer.type === 'sticker'
          ? layer.sticker?.url
          : undefined

    if (
      fabricObj &&
      currentSrc !== undefined &&
      (fabricObj as FabricObject & { dataSrc?: string }).dataSrc !== currentSrc
    ) {
      this.fabricCanvas.remove(fabricObj)
      this.fabricCanvas.requestRenderAll()
      const newObj = await this.track(toFabricObject(layer))
      if (newObj) {
        ;(newObj as FabricObject & { id?: string }).id = layer.id
        this.addObjectWithZIndex(layer, newObj)
        this.normalizePathPosition(newObj)
        this.fabricCanvas.requestRenderAll()
      }
      return
    }

    if (fabricObj) {
      if (layer.type === 'draw') {
        const path = fabricObj as FabricPath
        fabricObj.set({
          left: path.pathOffset.x,
          top: path.pathOffset.y,
          angle: layer.rotate,
          scaleX: layer.scaleX,
          scaleY: layer.scaleY,
        })
      } else {
        fabricObj.set({
          left: layer.x + ((fabricObj.width || 0) * Math.abs(layer.scaleX || 1)) / 2,
          top: layer.y + ((fabricObj.height || 0) * Math.abs(layer.scaleY || 1)) / 2,
          angle: layer.rotate,
          scaleX: layer.scaleX,
          scaleY: layer.scaleY,
        })
      }
      this.fabricCanvas.requestRenderAll()
    }
  }

  removeLayer(id: string): void {
    if (!this.fabricCanvas) return

    const fabricObj = this.findObject(id)
    if (fabricObj) {
      this.fabricCanvas.remove(fabricObj)
      this.fabricCanvas.requestRenderAll()
    }
    this.layers.delete(id)
  }

  private findObject(id: string): (FabricObject & { id?: string }) | undefined {
    if (!this.fabricCanvas) return undefined
    return this.fabricCanvas
      .getObjects()
      .find((obj) => (obj as FabricObject & { id?: string }).id === id) as
      (FabricObject & { id?: string }) | undefined
  }

  private normalizePathPosition(obj: FabricObject): void {
    if (obj.type !== 'path') return
    const path = obj as FabricPath
    path.set({ left: path.pathOffset.x, top: path.pathOffset.y })
  }

  private addObjectWithZIndex(layer: BaseLayer, obj: FabricObject): void {
    if (!this.fabricCanvas) return
    const objects = this.fabricCanvas.getObjects()
    let index = objects.length
    for (let i = 0; i < objects.length; i++) {
      const existingId = (objects[i] as FabricObject & { id?: string }).id
      const existingLayer = existingId ? this.layers.get(existingId) : undefined
      const existingZ = existingLayer ? existingLayer.zIndex : 0
      if (existingZ >= layer.zIndex) {
        index = i
        break
      }
    }
    if (index < objects.length) {
      this.fabricCanvas.insertAt(index, obj)
    } else {
      this.fabricCanvas.add(obj)
    }
  }

  render(): void {
    if (!this.fabricCanvas) return
    this.fabricCanvas.requestRenderAll()
  }

  async exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob> {
    if (!this.fabricCanvas) {
      throw new Error('Fabric canvas not initialized')
    }

    await this.whenReady()

    const savedViewport = this.fabricCanvas.viewportTransform
    const savedWidth = this.fabricCanvas.getWidth()
    const savedHeight = this.fabricCanvas.getHeight()

    this.fabricCanvas.setViewportTransform([1, 0, 0, 1, 0, 0])
    this.fabricCanvas.setDimensions({
      width: Math.max(1, Math.round(this.options.width)),
      height: Math.max(1, Math.round(this.options.height)),
    })

    try {
      const blob = await this.fabricCanvas.toBlob({
        format,
        quality: quality ?? 0.92,
        multiplier: 1,
      })
      if (!blob) {
        throw new Error('Failed to export fabric canvas to blob')
      }
      return blob
    } finally {
      this.fabricCanvas.setDimensions({ width: savedWidth, height: savedHeight })
      this.fabricCanvas.setViewportTransform(savedViewport)
      this.fabricCanvas.requestRenderAll()
    }
  }

  clear(): void {
    if (!this.fabricCanvas) return
    this.generation++
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

  setMode(mode: EditorMode): void {
    if (!this.fabricCanvas) return
    const isDraw = mode === 'draw'
    this.fabricCanvas.isDrawingMode = isDraw
    this.fabricCanvas.selection = !isDraw
    this.fabricCanvas.defaultCursor = isDraw ? 'crosshair' : 'default'
    this.fabricCanvas.requestRenderAll()
  }

  setDrawStyle(color: string, width: number): void {
    this.drawColor = color
    this.drawWidth = width
    if (this.fabricCanvas?.freeDrawingBrush) {
      this.fabricCanvas.freeDrawingBrush.color = color
      this.fabricCanvas.freeDrawingBrush.width = width
    }
  }

  onDrawComplete(callback: (points: number[][], color: string, width: number) => void): void {
    this.drawCompleteCallback = callback
  }

  private extractPathPoints(path: FabricPath): number[][] {
    const offsetX = path.left - path.pathOffset.x
    const offsetY = path.top - path.pathOffset.y
    const points: number[][] = []
    for (const cmd of path.path) {
      const type = cmd[0] as string
      const x = cmd[cmd.length - 2] as number
      const y = cmd[cmd.length - 1] as number
      if (type === 'M' || type === 'L') {
        points.push([offsetX + x, offsetY + y])
      } else if (type === 'Q' || type === 'T') {
        points.push([offsetX + x, offsetY + y])
      } else if (type === 'C' || type === 'S') {
        points.push([offsetX + x, offsetY + y])
      }
    }
    return points
  }

  private handleWheel = (e: WheelEvent): void => {
    e.preventDefault()
    if (!this.fabricCanvas) return
    const rect = this.canvas.getBoundingClientRect()
    const screenX = e.clientX - rect.left
    const screenY = e.clientY - rect.top
    const factor = e.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP
    this.zoomAt(screenX, screenY, factor)
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}
