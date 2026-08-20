import type { BaseLayer } from '../../../types'

export function renderDraw(ctx: CanvasRenderingContext2D, layer: BaseLayer): void {
  if (!layer.draw || layer.draw.points.length < 2) return

  ctx.save()
  ctx.translate(layer.x + layer.width / 2, layer.y + layer.height / 2)
  ctx.rotate((layer.rotate * Math.PI) / 180)
  ctx.scale(layer.scaleX, layer.scaleY)

  ctx.strokeStyle = layer.draw.strokeColor
  ctx.lineWidth = layer.draw.strokeWidth
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  if (layer.draw.isEraser) {
    ctx.globalCompositeOperation = 'destination-out'
  }

  ctx.beginPath()
  const points = layer.draw.points
  ctx.moveTo(points[0][0] - layer.width / 2, points[0][1] - layer.height / 2)

  for (let i = 1; i < points.length - 1; i++) {
    const xc = (points[i][0] + points[i + 1][0]) / 2 - layer.width / 2
    const yc = (points[i][1] + points[i + 1][1]) / 2 - layer.height / 2
    ctx.quadraticCurveTo(points[i][0] - layer.width / 2, points[i][1] - layer.height / 2, xc, yc)
  }

  const last = points[points.length - 1]
  ctx.lineTo(last[0] - layer.width / 2, last[1] - layer.height / 2)
  ctx.stroke()

  ctx.restore()
}
