import { ref, computed, type Ref } from 'vue'
import type { BaseEngine } from '../engine/base/BaseEngine'
import type { BaseLayer, EditorState, EngineCapabilities, ExportOptions } from '../types'
import { EditorMode } from '../types'
import { useHistory } from './useHistory'

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
  }

  function setMode(mode: EditorMode) {
    state.value.mode = mode
  }

  function addLayer(layer: BaseLayer) {
    if (!engine.value) return
    pushSnapshot(layers.value)
    layers.value.push(layer)
    engine.value.addLayer(layer)
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

  function undoAction() {
    if (undo(layers)) {
      syncLayersToEngine()
    }
  }

  function redoAction() {
    if (redo(layers)) {
      syncLayersToEngine()
    }
  }

  function syncLayersToEngine() {
    if (!engine.value) return
    engine.value.clear()
    for (const layer of layers.value) {
      engine.value.addLayer(layer)
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
    layers.value = []
    engine.value.clear()
    state.value.activeLayerId = null
  }

  function setCanvasSize(width: number, height: number) {
    state.value.canvasWidth = width
    state.value.canvasHeight = height
    if (engine.value) {
      engine.value.setSize(width, height)
    }
  }

  return {
    engine,
    state,
    layers,
    capabilities,
    activeLayer,
    setEngine,
    setMode,
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
    clearHistory,
  }
}
