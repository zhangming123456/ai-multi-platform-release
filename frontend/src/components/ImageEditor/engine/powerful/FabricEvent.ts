import { Canvas, type FabricObject } from 'fabric'
import type { BaseLayer } from '../../types'
import { fromFabricObject } from './FabricAdapter'

export function bindFabricEvents(
  canvas: Canvas,
  onChange: (id: string, patch: Partial<BaseLayer>) => void,
  onSelectionChange?: (ids: string[]) => void,
  getWorldSize?: () => { width: number; height: number },
): void {
  const clampToBounds = (obj: FabricObject): void => {
    const size = getWorldSize?.()
    if (!size) return
    if ((obj as { dataLayerType?: string }).dataLayerType === 'draw') return
    const w = obj.getScaledWidth()
    const h = obj.getScaledHeight()
    const halfW = w / 2
    const halfH = h / 2
    const left = clamp(obj.left ?? 0, halfW, Math.max(halfW, size.width - halfW))
    const top = clamp(obj.top ?? 0, halfH, Math.max(halfH, size.height - halfH))
    if (obj.left !== left || obj.top !== top) {
      obj.set({ left, top })
    }
  }

  canvas.on('object:modified', (e) => {
    const obj = e.target as FabricObject & { id?: string }
    if (obj && obj.id) {
      onChange(obj.id, fromFabricObject(obj))
    }
  })

  canvas.on('object:moving', (e) => {
    const obj = e.target as FabricObject & { id?: string }
    if (obj && obj.id) {
      clampToBounds(obj)
      onChange(obj.id, fromFabricObject(obj))
    }
  })

  canvas.on('object:scaling', (e) => {
    const obj = e.target as FabricObject & { id?: string }
    if (obj && obj.id) {
      onChange(obj.id, fromFabricObject(obj))
    }
  })

  canvas.on('object:rotating', (e) => {
    const obj = e.target as FabricObject & { id?: string }
    if (obj && obj.id) {
      onChange(obj.id, fromFabricObject(obj))
    }
  })

  canvas.on('selection:created', (e) => {
    if (onSelectionChange) {
      const selected = e.selected || []
      const ids = selected
        .map((obj) => (obj as FabricObject & { id?: string }).id)
        .filter((id): id is string => id !== undefined)
      onSelectionChange(ids)
    }
  })

  canvas.on('selection:updated', (e) => {
    if (onSelectionChange) {
      const selected = e.selected || []
      const ids = selected
        .map((obj) => (obj as FabricObject & { id?: string }).id)
        .filter((id): id is string => id !== undefined)
      onSelectionChange(ids)
    }
  })

  canvas.on('selection:cleared', () => {
    if (onSelectionChange) {
      onSelectionChange([])
    }
  })
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}
