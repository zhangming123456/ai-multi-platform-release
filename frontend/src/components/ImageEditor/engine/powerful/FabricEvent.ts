import { Canvas, type FabricObject } from 'fabric'
import type { BaseLayer } from '../../types'
import { fromFabricObject } from './FabricAdapter'

export function bindFabricEvents(
  canvas: Canvas,
  onChange: (id: string, patch: Partial<BaseLayer>) => void,
  onSelectionChange?: (ids: string[]) => void,
): void {
  canvas.on('object:modified', (e) => {
    const obj = e.target as FabricObject & { id?: string }
    if (obj && obj.id) {
      onChange(obj.id, fromFabricObject(obj))
    }
  })

  canvas.on('object:moving', (e) => {
    const obj = e.target as FabricObject & { id?: string }
    if (obj && obj.id) {
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
