import type { BaseLayer } from '../../../types'

export function renderText(ctx: CanvasRenderingContext2D, layer: BaseLayer): void {
  if (!layer.text) return

  ctx.save()
  ctx.translate(layer.x + layer.width / 2, layer.y + layer.height / 2)
  ctx.rotate((layer.rotate * Math.PI) / 180)
  ctx.scale(layer.scaleX, layer.scaleY)

  ctx.fillStyle = layer.text.color
  ctx.font = `${layer.text.bold ? 'bold ' : ''}${layer.text.fontSize}px sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText(layer.text.content, 0, 0)

  ctx.restore()
}
