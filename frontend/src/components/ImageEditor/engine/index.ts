import type { BaseEngine } from './base/BaseEngine.types'
import { ENGINE_TYPE } from '../config/engine.config'
import type { EngineType } from '../config/engine.config.types'

export async function createEngine(
  canvasEl: HTMLCanvasElement,
  width: number,
  height: number,
  engineType: EngineType = ENGINE_TYPE,
): Promise<BaseEngine> {
  if (engineType === 'powerful') {
    const { FabricEngine } = await import('./powerful/FabricEngine')
    const engine = new FabricEngine(canvasEl, width, height)
    engine.init()
    return engine
  }

  const { CanvasEngine } = await import('./lightweight/CanvasEngine')
  const engine = new CanvasEngine(canvasEl, width, height)
  engine.init()
  return engine
}

export type { BaseEngine } from './base/BaseEngine.types'
export { AbstractEngine } from './base/BaseEngine'
