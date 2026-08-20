import {
  Image as FabricImage,
  Text as FabricText,
  Path as FabricPath,
  Group as FabricGroup,
  type FabricObject,
} from 'fabric'
import type { BaseLayer } from '../../types'

export function toFabricObject(layer: BaseLayer): FabricObject | null {
  switch (layer.type) {
    case 'image':
      if (!layer.image?.src) return null
      return new FabricImage({
        left: layer.x,
        top: layer.y,
        width: layer.width,
        height: layer.height,
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
        src: layer.image.src,
      })
    case 'text':
      if (!layer.text) return null
      return new FabricText(layer.text.content, {
        left: layer.x,
        top: layer.y,
        fontSize: layer.text.fontSize,
        fill: layer.text.color,
        fontWeight: layer.text.bold ? 'bold' : 'normal',
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
      })
    case 'draw':
      if (!layer.draw || layer.draw.points.length < 2) return null
      const pathData = createPathData(layer.draw.points)
      return new FabricPath(pathData, {
        left: layer.x,
        top: layer.y,
        stroke: layer.draw.strokeColor,
        strokeWidth: layer.draw.strokeWidth,
        fill: undefined,
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
      })
    case 'sticker':
      if (!layer.sticker?.url) return null
      return new FabricImage({
        left: layer.x,
        top: layer.y,
        width: layer.width,
        height: layer.height,
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
        src: layer.sticker.url,
      })
    default:
      return null
  }
}

export function fromFabricObject(obj: FabricObject): Partial<BaseLayer> {
  return {
    x: obj.left || 0,
    y: obj.top || 0,
    width: obj.width || 0,
    height: obj.height || 0,
    rotate: obj.angle || 0,
    scaleX: obj.scaleX || 1,
    scaleY: obj.scaleY || 1,
  }
}

function createPathData(points: number[][]): string {
  if (points.length < 2) return ''

  let path = `M ${points[0][0]} ${points[0][1]}`

  for (let i = 1; i < points.length - 1; i++) {
    const xc = (points[i][0] + points[i + 1][0]) / 2
    const yc = (points[i][1] + points[i + 1][1]) / 2
    path += ` Q ${points[i][0]} ${points[i][1]} ${xc} ${yc}`
  }

  const last = points[points.length - 1]
  path += ` L ${last[0]} ${last[1]}`

  return path
}
