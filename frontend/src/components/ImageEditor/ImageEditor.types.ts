export const EditorMode = {
  Select: 'select',
  Crop: 'crop',
  Draw: 'draw',
  Text: 'text',
  Sticker: 'sticker',
  Filter: 'filter',
} as const

export type EditorMode = (typeof EditorMode)[keyof typeof EditorMode]

export type LayerType = 'image' | 'text' | 'draw' | 'sticker'

export interface BaseLayer {
  id: string
  type: LayerType
  visible: boolean
  zIndex: number
  locked: boolean
  x: number
  y: number
  width: number
  height: number
  rotate: number
  scaleX: number
  scaleY: number
  image?: {
    src: string
    naturalWidth: number
    naturalHeight: number
  }
  text?: {
    content: string
    fontSize: number
    color: string
    bold: boolean
  }
  draw?: {
    points: number[][]
    strokeColor: string
    strokeWidth: number
    isEraser?: boolean
  }
  sticker?: {
    url: string
  }
}

export interface EditorState {
  canvasWidth: number
  canvasHeight: number
  mode: EditorMode
  activeLayerId: string | null
  layers: BaseLayer[]
  history: HistoryState
  viewport: Viewport
}

export interface Viewport {
  scale: number
  offsetX: number
  offsetY: number
}

export interface HistoryState {
  undoStack: BaseLayer[][]
  redoStack: BaseLayer[][]
  max: number
}

export interface EngineCapabilities {
  multiSelect: boolean
  group: boolean
  lockLayer: boolean
  viewportZoom: boolean
  svgImport: boolean
  freeTransform: boolean
  doubleClickEdit: boolean
}

export interface EngineOptions {
  width: number
  height: number
}

export interface ExportOptions {
  format: 'png' | 'jpeg'
  quality?: number
}
