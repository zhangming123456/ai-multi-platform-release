import type { EngineType } from './engine.config.types'

export type { EngineType }

export const ENGINE_TYPE: EngineType =
  (import.meta.env.VITE_EDITOR_ENGINE as EngineType) || 'powerful'
