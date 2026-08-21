import { ref, computed, type Ref } from 'vue'
import type { BaseEngine } from '../engine/base/BaseEngine'
import type { BaseLayer, EditorState, EngineCapabilities, ExportOptions } from '../types'
import { EditorMode } from '../types'
import { useHistory } from './useHistory'
import { cropImage, filterImage, rotateImage, type CropRect, type FilterType } from '../utils/image'

export interface TextLayerOptions {
  content: string
  fontSize?: number
  color?: string
  bold?: boolean
}

export function useEditor() {
  const engine = ref<BaseEngine | null>(null)
  const state = ref<EditorState>({
    canvasWidth: 800,
    canvasHeight: 600,
    mode: EditorMode.Select,
    activeLayerId: null,
    layers: [],
    history: {
      undoStack: [],
      redoStack: [],
      max: 50,
    },
    viewport: {
      scale: 1,
      offsetX: 0,
      offsetY: 0,
    },
  })

  const layers = ref<BaseLayer[]>([]) as Ref<BaseLayer[]>
  const originalImage = ref<{ src: string; width: number; height: number } | null>(null)
  let filterBaseSrc: string | null = null
  const { pushSnapshot, undo, redo, canUndo, canRedo, clear: clearHistory } = useHistory()

  const capabilities = computed<EngineCapabilities>(() => {
    return (
      engine.value?.getCapabilities() || {
        multiSelect: false,
        group: false,
        lockLayer: false,
        viewportZoom: false,
        svgImport: false,
        freeTransform: false,
        doubleClickEdit: false,
      }
    )
  })

  const activeLayer = computed<BaseLayer | null>(() => {
    if (!state.value.activeLayerId) return null
    return layers.value.find((l) => l.id === state.value.activeLayerId) || null
  })

  function setEngine(eng: BaseEngine) {
    engine.value = eng
    eng.onLayerChange((id, patch) => {
      const layer = layers.value.find((l) => l.id === id)
      if (layer) {
        pushSnapshot(layers.value)
        Object.assign(layer, patch)
        eng.updateLayer(id, patch)
      }
    })
    eng.onSelectionChange?.((ids) => {
      state.value.activeLayerId = ids.length === 1 ? ids[0] : null
    })
  }

  function setMode(mode: EditorMode) {
    state.value.mode = mode
    const eng = engine.value
    if (!eng) return
    eng.setMode?.(mode)
    if (mode === EditorMode.Crop) {
      eng.fitToViewport?.()
      const viewport = eng.getViewport?.()
      if (viewport) {
        state.value.viewport = { ...viewport }
      }
    }
  }

  function setOriginalImage(src: string, width: number, height: number) {
    originalImage.value = { src, width, height }
  }

  function addLayer(layer: BaseLayer) {
    if (!engine.value) return
    pushSnapshot(layers.value)
    layers.value.push(layer)
    void engine.value.addLayer(layer)
  }

  function updateLayer(id: string, patch: Partial<BaseLayer>) {
    if (!engine.value) return
    pushSnapshot(layers.value)
    const layer = layers.value.find((l) => l.id === id)
    if (layer) {
      Object.assign(layer, patch)
      engine.value.updateLayer(id, patch)
    }
  }

  function removeLayer(id: string) {
    if (!engine.value) return
    pushSnapshot(layers.value)
    layers.value = layers.value.filter((l) => l.id !== id)
    engine.value.removeLayer(id)
    if (state.value.activeLayerId === id) {
      state.value.activeLayerId = null
    }
  }

  function selectLayer(id: string | null) {
    state.value.activeLayerId = id
  }

  async function syncLayersToEngine() {
    if (!engine.value) return
    engine.value.clear()
    for (const layer of layers.value) {
      await engine.value.addLayer(layer)
    }
  }

  function undoAction() {
    if (undo(layers)) {
      void syncLayersToEngine()
    }
  }

  function redoAction() {
    if (redo(layers)) {
      void syncLayersToEngine()
    }
  }

  async function exportToBlob(options: ExportOptions): Promise<Blob> {
    if (!engine.value) {
      throw new Error('Engine not initialized')
    }
    return engine.value.exportToBlob(options.format, options.quality)
  }

  function clearAll() {
    if (!engine.value) return
    pushSnapshot(layers.value)
    filterBaseSrc = null
    layers.value = []
    engine.value.clear()
    state.value.activeLayerId = null
  }

  function setCanvasSize(width: number, height: number) {
    state.value.canvasWidth = width
    state.value.canvasHeight = height
    if (engine.value) {
      engine.value.setSize(width, height)
      engine.value.fitToViewport?.()
    }
  }

  async function rotate90() {
    if (!engine.value) return
    const bg = getBaseImageLayer()
    if (!bg) return
    const result = await rotateImage(bg.image!.src)
    if (result.src === bg.image!.src) return

    pushSnapshot(layers.value)
    filterBaseSrc = null
    bg.image = {
      src: result.src,
      naturalWidth: result.width,
      naturalHeight: result.height,
    }
    bg.width = result.width
    bg.height = result.height
    bg.x = 0
    bg.y = 0
    bg.scaleX = 1
    bg.scaleY = 1
    state.value.canvasWidth = result.width
    state.value.canvasHeight = result.height
    engine.value.setSize(result.width, result.height)
    engine.value.fitToViewport?.()
    engine.value.updateLayer(bg.id, {
      x: 0,
      y: 0,
      width: result.width,
      height: result.height,
      scaleX: 1,
      scaleY: 1,
      image: bg.image,
    })
  }

  async function applyCrop(rect: CropRect) {
    if (!engine.value) return
    const bg = getBaseImageLayer()
    if (!bg) return
    const sx = (rect.x - bg.x) / bg.scaleX
    const sy = (rect.y - bg.y) / bg.scaleY
    const sw = rect.width / bg.scaleX
    const sh = rect.height / bg.scaleY
    const src = await cropImage(filterBaseSrc || bg.image!.src, {
      x: sx,
      y: sy,
      width: sw,
      height: sh,
    })
    const width = Math.max(1, Math.round(rect.width))
    const height = Math.max(1, Math.round(rect.height))

    pushSnapshot(layers.value)
    filterBaseSrc = null
    bg.image = {
      src,
      naturalWidth: width,
      naturalHeight: height,
    }
    bg.x = 0
    bg.y = 0
    bg.width = width
    bg.height = height
    bg.scaleX = 1
    bg.scaleY = 1
    state.value.canvasWidth = width
    state.value.canvasHeight = height
    engine.value.setSize(width, height)
    engine.value.fitToViewport?.()
    engine.value.updateLayer(bg.id, {
      x: 0,
      y: 0,
      width,
      height,
      scaleX: 1,
      scaleY: 1,
      image: bg.image,
    })
  }

  async function applyFilter(type: FilterType) {
    if (!engine.value) return
    const bg = getBaseImageLayer()
    if (!bg) return
    const baseSrc = filterBaseSrc || bg.image!.src
    const src = await filterImage(baseSrc, type)

    if (type === 'none') {
      if (bg.image!.src === baseSrc) return
      pushSnapshot(layers.value)
      bg.image = { ...bg.image!, src: baseSrc }
      filterBaseSrc = null
    } else {
      if (src === bg.image!.src) return
      pushSnapshot(layers.value)
      bg.image = { ...bg.image!, src }
      filterBaseSrc = filterBaseSrc || baseSrc
    }
    engine.value.updateLayer(bg.id, { image: bg.image })
  }

  function addTextLayer(options: TextLayerOptions) {
    if (!engine.value || !options.content) return
    const layer: BaseLayer = {
      id: `text-${Date.now()}`,
      type: 'text',
      visible: true,
      zIndex: layers.value.length + 1,
      locked: false,
      x: state.value.canvasWidth / 2,
      y: state.value.canvasHeight / 2,
      width: 200,
      height: 50,
      rotate: 0,
      scaleX: 1,
      scaleY: 1,
      text: {
        content: options.content,
        fontSize: options.fontSize ?? 32,
        color: options.color ?? '#1d1d1f',
        bold: options.bold ?? false,
      },
    }
    addLayer(layer)
  }

  function addStickerLayer(url: string) {
    if (!engine.value || !url) return
    const size = 120
    const layer: BaseLayer = {
      id: `sticker-${Date.now()}`,
      type: 'sticker',
      visible: true,
      zIndex: layers.value.length + 1,
      locked: false,
      x: state.value.canvasWidth / 2 - size / 2,
      y: state.value.canvasHeight / 2 - size / 2,
      width: size,
      height: size,
      rotate: 0,
      scaleX: 1,
      scaleY: 1,
      sticker: { url },
    }
    addLayer(layer)
  }

  function resetEditor() {
    if (!engine.value) return
    const original = originalImage.value
    if (!original) return

    pushSnapshot(layers.value)
    filterBaseSrc = null
    layers.value = [
      {
        id: 'bg-0',
        type: 'image',
        visible: true,
        zIndex: 0,
        locked: true,
        x: 0,
        y: 0,
        width: original.width,
        height: original.height,
        rotate: 0,
        scaleX: 1,
        scaleY: 1,
        image: {
          src: original.src,
          naturalWidth: original.width,
          naturalHeight: original.height,
        },
      },
    ]
    state.value.canvasWidth = original.width
    state.value.canvasHeight = original.height
    state.value.activeLayerId = null
    state.value.mode = EditorMode.Select
    engine.value.setSize(original.width, original.height)
    engine.value.fitToViewport?.()
    void syncLayersToEngine()
  }

  function getBaseImageLayer(): BaseLayer | undefined {
    return layers.value.find((l) => l.id === 'bg-0' && l.type === 'image')
  }

  function moveBaseImage(dx: number, dy: number): void {
    if (!engine.value) return
    const bg = getBaseImageLayer()
    if (!bg) return
    const scaledW = bg.width * Math.abs(bg.scaleX)
    const scaledH = bg.height * Math.abs(bg.scaleY)
    const minX = Math.min(0, state.value.canvasWidth - scaledW)
    const minY = Math.min(0, state.value.canvasHeight - scaledH)
    const nx = clamp(bg.x + dx, minX, 0)
    const ny = clamp(bg.y + dy, minY, 0)
    if (nx === bg.x && ny === bg.y) return
    bg.x = nx
    bg.y = ny
    engine.value.updateLayer(bg.id, { x: nx, y: ny })
  }

  function scaleBaseImage(factor: number): void {
    if (!engine.value) return
    const bg = getBaseImageLayer()
    if (!bg) return
    const next = clamp(bg.scaleX * factor, 1, 4)
    if (next === bg.scaleX) return
    const canvasW = state.value.canvasWidth
    const canvasH = state.value.canvasHeight
    const centerX = bg.x + (bg.width * bg.scaleX) / 2
    const centerY = bg.y + (bg.height * bg.scaleY) / 2
    bg.scaleX = next
    bg.scaleY = next
    const scaledW = bg.width * next
    const scaledH = bg.height * next
    bg.x = clamp(centerX - scaledW / 2, Math.min(0, canvasW - scaledW), 0)
    bg.y = clamp(centerY - scaledH / 2, Math.min(0, canvasH - scaledH), 0)
    engine.value.updateLayer(bg.id, { x: bg.x, y: bg.y, scaleX: next, scaleY: next })
  }

  return {
    engine,
    state,
    layers,
    capabilities,
    activeLayer,
    setEngine,
    setMode,
    setOriginalImage,
    addLayer,
    updateLayer,
    removeLayer,
    selectLayer,
    undoAction,
    redoAction,
    canUndo,
    canRedo,
    exportToBlob,
    clearAll,
    setCanvasSize,
    rotate90,
    applyCrop,
    applyFilter,
    addTextLayer,
    addStickerLayer,
    resetEditor,
    moveBaseImage,
    scaleBaseImage,
    clearHistory,
    syncLayersToEngine,
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}
