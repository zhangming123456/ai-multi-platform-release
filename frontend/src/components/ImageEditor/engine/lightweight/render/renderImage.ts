import type { BaseLayer } from '../../../types'

export function renderImage(
  ctx: CanvasRenderingContext2D,
  layer: BaseLayer,
  imageCache: Map<string, HTMLImageElement>,
): void {
  if (!layer.image) return

  const img = imageCache.get(layer.image.src)
  if (!img) return

  ctx.save()
  ctx.translate(layer.x + layer.width / 2, layer.y + layer.height / 2)
  ctx.rotate((layer.rotate * Math.PI) / 180)
  ctx.scale(layer.scaleX, layer.scaleY)
  ctx.drawImage(img, -layer.width / 2, -layer.height / 2, layer.width, layer.height)
  ctx.restore()
}

export async function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = src
  })
}
