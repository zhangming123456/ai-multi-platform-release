import { ref, toRaw, type Ref } from 'vue'
import type { BaseLayer } from '../ImageEditor.types'

function cloneLayers(layers: BaseLayer[]): BaseLayer[] {
  return JSON.parse(JSON.stringify(toRaw(layers)))
}

export function useHistory(max = 50) {
  const undoStack = ref<BaseLayer[][]>([])
  const redoStack = ref<BaseLayer[][]>([])

  function pushSnapshot(layers: BaseLayer[]) {
    const snap = cloneLayers(layers)
    undoStack.value.push(snap)
    if (undoStack.value.length > max) undoStack.value.shift()
    redoStack.value = []
  }

  function undo(layers: Ref<BaseLayer[]>) {
    const snap = undoStack.value.pop()
    if (!snap) return false
    redoStack.value.push(cloneLayers(layers.value))
    layers.value = snap
    return true
  }

  function redo(layers: Ref<BaseLayer[]>) {
    const snap = redoStack.value.pop()
    if (!snap) return false
    undoStack.value.push(cloneLayers(layers.value))
    layers.value = snap
    return true
  }

  function canUndo() {
    return undoStack.value.length > 0
  }

  function canRedo() {
    return redoStack.value.length > 0
  }

  function clear() {
    undoStack.value = []
    redoStack.value = []
  }

  return {
    undoStack,
    redoStack,
    pushSnapshot,
    undo,
    redo,
    canUndo,
    canRedo,
    clear,
  }
}
