import type { BaseLayer } from '../../../types'

export function renderSticker(
  ctx: CanvasRenderingContext2D,
  layer: BaseLayer,
  imageCache: Map<string, HTMLImageElement>,
): void {
  if (!layer.sticker) return

  const img = imageCache.get(layer.sticker.url)
  if (!img) return

  ctx.save()
  ctx.translate(layer.x + layer.width / 2, layer.y + layer.height / 2)
  ctx.rotate((layer.rotate * Math.PI) / 180)
  ctx.scale(layer.scaleX, layer.scaleY)
  ctx.drawImage(img, -layer.width / 2, -layer.height / 2, layer.width, layer.height)
  ctx.restore()
}
