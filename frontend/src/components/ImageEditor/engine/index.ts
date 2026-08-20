import type { BaseEngine } from './base/BaseEngine'
import { ENGINE_TYPE } from '../config/engine.config'

export async function createEngine(
  canvasEl: HTMLCanvasElement,
  width: number,
  height: number,
): Promise<BaseEngine> {
  if (ENGINE_TYPE === 'powerful') {
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

export type { BaseEngine } from './base/BaseEngine'
export { AbstractEngine } from './base/BaseEngine'
export { CanvasEngine } from './lightweight/CanvasEngine'
