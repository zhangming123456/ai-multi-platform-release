import { AbstractEngine } from '../base/BaseEngine'
import type { BaseLayer, EngineCapabilities, Viewport } from '../../types'
import { EditorMode } from '../../types'
import { LayerManager } from './LayerManager'
import { renderImage, loadImage } from './render/renderImage'
import { renderText } from './render/renderText'
import { renderDraw } from './render/renderDraw'
import { renderSticker } from './render/renderSticker'

const MIN_SCALE = 0.1
const MAX_SCALE = 8
const ZOOM_STEP = 1.5

interface DragState {
  layerId: string | null
  startPointerX: number
  startPointerY: number
  startLayerX: number
  startLayerY: number
}

export class CanvasEngine extends AbstractEngine {
  private layerManager: LayerManager
  private imageCache: Map<string, HTMLImageElement> = new Map()
  private viewport: Viewport = { scale: 1, offsetX: 0, offsetY: 0 }
  private selectedLayerId: string | null = null
  private dragState: DragState | null = null
  private panStart: { x: number; y: number } | null = null
  private mode: EditorMode = EditorMode.Select
  private drawColor = '#007aff'
  private drawWidth = 4
  private drawing = false
  private drawingPoints: number[][] = []
  private drawCompleteCallback:
    ((points: number[][], color: string, width: number) => void) | null = null

  constructor(canvas: HTMLCanvasElement, width: number, height: number) {
    super(canvas, { width, height })
    this.layerManager = new LayerManager()
    this.bindEvents()
  }

  init(): void {
    this.canvas.width = this.options.width
    this.canvas.height = this.options.height
    this.render()
  }

  destroy(): void {
    this.unbindEvents()
    this.layerManager.clear()
    this.imageCache.clear()
  }

  setSize(width: number, height: number): void {
    this.options.width = width
    this.options.height = height
    this.render()
  }

  resizeBuffer(width: number, height: number): void {
    this.canvas.width = Math.max(1, Math.floor(width))
    this.canvas.height = Math.max(1, Math.floor(height))
    this.render()
  }

  fitToViewport(): void {
    const worldW = this.options.width
    const worldH = this.options.height
    if (worldW <= 0 || worldH <= 0) return
    const scale = clamp(
      Math.min(this.canvas.width / worldW, this.canvas.height / worldH),
      MIN_SCALE,
      MAX_SCALE,
    )
    this.viewport = {
      scale,
      offsetX: (this.canvas.width - worldW * scale) / 2,
      offsetY: (this.canvas.height - worldH * scale) / 2,
    }
    this.render()
  }

  addLayer(layer: BaseLayer): void {
    this.layerManager.addLayer(layer)

    if (layer.type === 'image' && layer.image?.src) {
      this.track(this.loadImageToCache(layer.image.src))
    } else if (layer.type === 'sticker' && layer.sticker?.url) {
      this.track(this.loadImageToCache(layer.sticker.url))
    }

    this.render()
  }

  updateLayer(id: string, patch: Partial<BaseLayer>): void {
    this.layerManager.updateLayer(id, patch)

    const layer = this.layerManager.getLayer(id)
    const src = layer?.image?.src ?? layer?.sticker?.url
    if (src) {
      this.track(this.loadImageToCache(src))
    }

    this.render()
  }

  removeLayer(id: string): void {
    this.layerManager.removeLayer(id)
    if (this.selectedLayerId === id) {
      this.selectedLayerId = null
    }
    this.render()
  }

  render(): void {
    const ctx = this.ctx
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.clearRect(0, 0, this.canvas.width, this.canvas.height)

    ctx.setTransform(
      this.viewport.scale,
      0,
      0,
      this.viewport.scale,
      this.viewport.offsetX,
      this.viewport.offsetY,
    )

    this.renderLayers(ctx)

    if (this.selectedLayerId) {
      const layer = this.layerManager.getLayer(this.selectedLayerId)
      if (layer && layer.visible) {
        this.drawSelection(ctx, layer)
      }
    }

    if (this.mode === EditorMode.Draw && this.drawingPoints.length >= 2) {
      this.drawPreviewPath(ctx)
    }

    ctx.setTransform(1, 0, 0, 1, 0, 0)
  }

  private renderLayers(ctx: CanvasRenderingContext2D): void {
    const layers = this.layerManager.getAllLayers()
    for (const layer of layers) {
      if (!layer.visible) continue

      switch (layer.type) {
        case 'image':
          renderImage(ctx, layer, this.imageCache)
          break
        case 'text':
          renderText(ctx, layer)
          break
        case 'draw':
          renderDraw(ctx, layer)
          break
        case 'sticker':
          renderSticker(ctx, layer, this.imageCache)
          break
      }
    }
  }

  async exportToBlob(format: 'png' | 'jpeg', quality?: number): Promise<Blob> {
    await this.whenReady()

    const exportCanvas = document.createElement('canvas')
    exportCanvas.width = Math.max(1, Math.round(this.options.width))
    exportCanvas.height = Math.max(1, Math.round(this.options.height))
    const exportCtx = exportCanvas.getContext('2d')
    if (!exportCtx) {
      throw new Error('Failed to create export canvas')
    }
    this.renderLayers(exportCtx)

    return new Promise((resolve, reject) => {
      exportCanvas.toBlob(
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
    this.selectedLayerId = null
    this.ctx.setTransform(1, 0, 0, 1, 0, 0)
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height)
  }

  getCapabilities(): EngineCapabilities {
    return {
      multiSelect: false,
      group: false,
      lockLayer: false,
      viewportZoom: true,
      svgImport: false,
      freeTransform: false,
      doubleClickEdit: false,
    }
  }

  setViewport(viewport: Partial<Viewport>): void {
    this.viewport = {
      ...this.viewport,
      ...viewport,
      scale: clamp(viewport.scale ?? this.viewport.scale, MIN_SCALE, MAX_SCALE),
    }
    this.render()
  }

  getViewport(): Viewport {
    return { ...this.viewport }
  }

  zoomAt(screenX: number, screenY: number, factor: number): void {
    const nextScale = clamp(this.viewport.scale * factor, MIN_SCALE, MAX_SCALE)
    const k = nextScale / this.viewport.scale
    this.viewport.offsetX = screenX - (screenX - this.viewport.offsetX) * k
    this.viewport.offsetY = screenY - (screenY - this.viewport.offsetY) * k
    this.viewport.scale = nextScale
    this.render()
  }

  selectLayer(id: string | null): void {
    this.selectedLayerId = id
    this.render()
  }

  getSelectedLayerId(): string | null {
    return this.selectedLayerId
  }

  setMode(mode: EditorMode): void {
    this.mode = mode
    this.drawing = false
    this.drawingPoints = []
    this.canvas.style.cursor =
      mode === EditorMode.Draw ? 'crosshair' : this.selectedLayerId ? 'move' : 'default'
    this.render()
  }

  setDrawStyle(color: string, width: number): void {
    this.drawColor = color
    this.drawWidth = width
  }

  onDrawComplete(callback: (points: number[][], color: string, width: number) => void): void {
    this.drawCompleteCallback = callback
  }

  private bindEvents(): void {
    this.canvas.addEventListener('pointerdown', this.handlePointerDown)
    this.canvas.addEventListener('pointermove', this.handlePointerMove)
    this.canvas.addEventListener('pointerup', this.handlePointerUp)
    this.canvas.addEventListener('pointercancel', this.handlePointerUp)
    this.canvas.addEventListener('wheel', this.handleWheel, { passive: false })
    this.canvas.style.touchAction = 'none'
  }

  private unbindEvents(): void {
    this.canvas.removeEventListener('pointerdown', this.handlePointerDown)
    this.canvas.removeEventListener('pointermove', this.handlePointerMove)
    this.canvas.removeEventListener('pointerup', this.handlePointerUp)
    this.canvas.removeEventListener('pointercancel', this.handlePointerUp)
    this.canvas.removeEventListener('wheel', this.handleWheel)
    this.canvas.style.touchAction = ''
  }

  private handlePointerDown = (e: PointerEvent): void => {
    if (e.button !== 0) return

    if (this.mode === EditorMode.Draw) {
      const point = this.screenToWorld(e)
      this.drawing = true
      this.drawingPoints = [[point.x, point.y]]
      this.canvas.setPointerCapture(e.pointerId)
      this.render()
      return
    }

    const point = this.screenToWorld(e)

    const hitLayer = this.hitTest(point.x, point.y)
    if (hitLayer) {
      this.selectedLayerId = hitLayer.id
      this.dragState = {
        layerId: hitLayer.id,
        startPointerX: point.x,
        startPointerY: point.y,
        startLayerX: hitLayer.x,
        startLayerY: hitLayer.y,
      }
      this.canvas.style.cursor = 'grabbing'
    } else {
      this.selectedLayerId = null
      this.panStart = { x: e.clientX, y: e.clientY }
      this.canvas.style.cursor = 'grabbing'
    }

    this.canvas.setPointerCapture(e.pointerId)
    this.render()
  }

  private handlePointerMove = (e: PointerEvent): void => {
    if (this.mode === EditorMode.Draw && this.drawing) {
      const point = this.screenToWorld(e)
      this.drawingPoints.push([point.x, point.y])
      this.render()
      return
    }

    if (this.dragState && this.dragState.layerId) {
      const point = this.screenToWorld(e)
      const layer = this.layerManager.getLayer(this.dragState.layerId)
      if (layer) {
        const dx = point.x - this.dragState.startPointerX
        const dy = point.y - this.dragState.startPointerY
        layer.x = this.dragState.startLayerX + dx
        layer.y = this.dragState.startLayerY + dy
        this.notifyLayerChange(layer.id, { x: layer.x, y: layer.y })
        this.render()
      }
      return
    }

    if (this.panStart) {
      this.viewport.offsetX += e.clientX - this.panStart.x
      this.viewport.offsetY += e.clientY - this.panStart.y
      this.panStart = { x: e.clientX, y: e.clientY }
      this.render()
    }
  }

  private handlePointerUp = (): void => {
    if (this.mode === EditorMode.Draw && this.drawing) {
      this.drawing = false
      if (this.drawingPoints.length >= 2 && this.drawCompleteCallback) {
        this.drawCompleteCallback(this.drawingPoints, this.drawColor, this.drawWidth)
      }
      this.drawingPoints = []
      this.render()
      return
    }

    this.dragState = null
    this.panStart = null
    this.canvas.style.cursor = this.selectedLayerId ? 'move' : 'default'
  }

  private handleWheel = (e: WheelEvent): void => {
    e.preventDefault()
    const rect = this.canvas.getBoundingClientRect()
    const screenX = e.clientX - rect.left
    const screenY = e.clientY - rect.top
    const factor = e.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP
    this.zoomAt(screenX, screenY, factor)
  }

  private screenToWorld(e: PointerEvent): { x: number; y: number } {
    const rect = this.canvas.getBoundingClientRect()
    const sx = e.clientX - rect.left
    const sy = e.clientY - rect.top
    return {
      x: (sx - this.viewport.offsetX) / this.viewport.scale,
      y: (sy - this.viewport.offsetY) / this.viewport.scale,
    }
  }

  private hitTest(worldX: number, worldY: number): BaseLayer | null {
    const layers = this.layerManager.getAllLayers()
    for (let i = layers.length - 1; i >= 0; i--) {
      const layer = layers[i]
      if (!layer.visible || layer.locked) continue

      const w = layer.width * Math.abs(layer.scaleX)
      const h = layer.height * Math.abs(layer.scaleY)
      if (
        worldX >= layer.x &&
        worldX <= layer.x + w &&
        worldY >= layer.y &&
        worldY <= layer.y + h
      ) {
        return layer
      }
    }
    return null
  }

  private drawSelection(ctx: CanvasRenderingContext2D, layer: BaseLayer): void {
    const w = layer.width * Math.abs(layer.scaleX)
    const h = layer.height * Math.abs(layer.scaleY)

    ctx.save()
    ctx.strokeStyle = '#007aff'
    ctx.lineWidth = 1.5 / this.viewport.scale
    ctx.setLineDash([6 / this.viewport.scale, 4 / this.viewport.scale])
    ctx.strokeRect(layer.x, layer.y, w, h)
    ctx.setLineDash([])
    ctx.restore()
  }

  private drawPreviewPath(ctx: CanvasRenderingContext2D): void {
    if (this.drawingPoints.length < 2) return

    ctx.save()
    ctx.strokeStyle = this.drawColor
    ctx.lineWidth = this.drawWidth / this.viewport.scale
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'
    ctx.beginPath()
    ctx.moveTo(this.drawingPoints[0][0], this.drawingPoints[0][1])
    for (let i = 1; i < this.drawingPoints.length; i++) {
      ctx.lineTo(this.drawingPoints[i][0], this.drawingPoints[i][1])
    }
    ctx.stroke()
    ctx.restore()
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

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}
