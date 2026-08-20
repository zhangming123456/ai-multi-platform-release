export type EngineType = 'lightweight' | 'powerful'

export const ENGINE_TYPE: EngineType =
  (import.meta.env.VITE_EDITOR_ENGINE as EngineType) || 'powerful'
