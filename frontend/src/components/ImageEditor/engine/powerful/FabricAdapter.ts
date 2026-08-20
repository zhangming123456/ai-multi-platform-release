import {
  Image as FabricImage,
  Text as FabricText,
  Path as FabricPath,
  type FabricObject,
} from 'fabric'
import type { BaseLayer } from '../../types'

type TaggedObject = FabricObject & { dataLayerType?: BaseLayer['type']; dataSrc?: string }

function tag(obj: FabricObject, layer: BaseLayer): void {
  const tagged = obj as TaggedObject
  tagged.dataLayerType = layer.type
  if (layer.type === 'image' && layer.image) {
    tagged.dataSrc = layer.image.src
  } else if (layer.type === 'sticker' && layer.sticker) {
    tagged.dataSrc = layer.sticker.url
  }
}

export async function toFabricObject(layer: BaseLayer): Promise<FabricObject | null> {
  switch (layer.type) {
    case 'image': {
      if (!layer.image?.src) return null
      const image = await FabricImage.fromURL(layer.image.src)
      tag(image, layer)
      image.set({
        left: layer.x + (layer.width * Math.abs(layer.scaleX)) / 2,
        top: layer.y + (layer.height * Math.abs(layer.scaleY)) / 2,
        originX: 'center',
        originY: 'center',
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
      })
      return image
    }
    case 'text': {
      if (!layer.text) return null
      const text = new FabricText(layer.text.content, {
        fontSize: layer.text.fontSize,
        fill: layer.text.color,
        fontWeight: layer.text.bold ? 'bold' : 'normal',
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
      })
      tag(text, layer)
      text.set({
        left: layer.x + (text.width * Math.abs(layer.scaleX)) / 2,
        top: layer.y + (text.height * Math.abs(layer.scaleY)) / 2,
        originX: 'center',
        originY: 'center',
      })
      return text
    }
    case 'draw': {
      if (!layer.draw || layer.draw.points.length < 2) return null
      const pathData = createPathData(layer.draw.points)
      const path = new FabricPath(pathData, {
        left: layer.x,
        top: layer.y,
        stroke: layer.draw.strokeColor,
        strokeWidth: layer.draw.strokeWidth,
        fill: undefined,
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
        pathOffset: { x: 0, y: 0 },
      })
      tag(path, layer)
      return path
    }
    case 'sticker': {
      if (!layer.sticker?.url) return null
      const stickerImage = await FabricImage.fromURL(layer.sticker.url)
      tag(stickerImage, layer)
      stickerImage.set({
        left: layer.x + (layer.width * Math.abs(layer.scaleX)) / 2,
        top: layer.y + (layer.height * Math.abs(layer.scaleY)) / 2,
        originX: 'center',
        originY: 'center',
        angle: layer.rotate,
        scaleX: layer.scaleX,
        scaleY: layer.scaleY,
      })
      return stickerImage
    }
    default:
      return null
  }
}

export function fromFabricObject(obj: FabricObject): Partial<BaseLayer> {
  const type = (obj as TaggedObject).dataLayerType

  if (type === 'draw') {
    return {
      rotate: obj.angle || 0,
      scaleX: obj.scaleX || 1,
      scaleY: obj.scaleY || 1,
    }
  }

  const halfW = ((obj.width || 0) * Math.abs(obj.scaleX || 1)) / 2
  const halfH = ((obj.height || 0) * Math.abs(obj.scaleY || 1)) / 2
  return {
    x: (obj.left || 0) - halfW,
    y: (obj.top || 0) - halfH,
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
