import type { CropRect, FilterType } from './image.types'

export function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    if (/^https?:\/\//i.test(src)) {
      img.crossOrigin = 'anonymous'
    }
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = src
  })
}

export function applyFilterToImageData(
  imageData: ImageData,
  type: FilterType,
  blockSize = 8,
): void {
  const data = imageData.data
  const { width, height } = imageData

  switch (type) {
    case 'grayscale':
      for (let i = 0; i < data.length; i += 4) {
        const gray = Math.round(data[i] * 0.299 + data[i + 1] * 0.587 + data[i + 2] * 0.114)
        data[i] = gray
        data[i + 1] = gray
        data[i + 2] = gray
      }
      break
    case 'sepia':
      for (let i = 0; i < data.length; i += 4) {
        const r = data[i]
        const g = data[i + 1]
        const b = data[i + 2]
        data[i] = Math.min(255, r * 0.393 + g * 0.769 + b * 0.189)
        data[i + 1] = Math.min(255, r * 0.349 + g * 0.686 + b * 0.168)
        data[i + 2] = Math.min(255, r * 0.272 + g * 0.534 + b * 0.131)
      }
      break
    case 'invert':
      for (let i = 0; i < data.length; i += 4) {
        data[i] = 255 - data[i]
        data[i + 1] = 255 - data[i + 1]
        data[i + 2] = 255 - data[i + 2]
      }
      break
    case 'pixelate': {
      for (let by = 0; by < height; by += blockSize) {
        for (let bx = 0; bx < width; bx += blockSize) {
          let r = 0
          let g = 0
          let b = 0
          let count = 0
          const maxY = Math.min(by + blockSize, height)
          const maxX = Math.min(bx + blockSize, width)
          for (let y = by; y < maxY; y++) {
            for (let x = bx; x < maxX; x++) {
              const idx = (y * width + x) * 4
              r += data[idx]
              g += data[idx + 1]
              b += data[idx + 2]
              count++
            }
          }
          r = Math.round(r / count)
          g = Math.round(g / count)
          b = Math.round(b / count)
          for (let y = by; y < maxY; y++) {
            for (let x = bx; x < maxX; x++) {
              const idx = (y * width + x) * 4
              data[idx] = r
              data[idx + 1] = g
              data[idx + 2] = b
            }
          }
        }
      }
      break
    }
    case 'emboss': {
      const src = new Uint8ClampedArray(data)
      for (let y = 1; y < height - 1; y++) {
        for (let x = 1; x < width - 1; x++) {
          const idx = (y * width + x) * 4
          const topLeft = (y - 1) * width + (x - 1)
          const bottomRight = (y + 1) * width + (x + 1)
          for (let c = 0; c < 3; c++) {
            const val = src[topLeft * 4 + c] - src[bottomRight * 4 + c] + 128
            data[idx + c] = Math.min(255, Math.max(0, val))
          }
        }
      }
      break
    }
    case 'none':
    default:
      break
  }
}

export async function filterImage(
  src: string,
  type: FilterType,
  maxWidth = 2048,
  blockSize = 8,
): Promise<string> {
  if (type === 'none') return src

  const img = await loadImage(src)
  const scale = Math.min(1, maxWidth / img.naturalWidth)
  const width = Math.max(1, Math.round(img.naturalWidth * scale))
  const height = Math.max(1, Math.round(img.naturalHeight * scale))

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Failed to get 2d context')

  ctx.drawImage(img, 0, 0, width, height)
  const imageData = ctx.getImageData(0, 0, width, height)
  applyFilterToImageData(imageData, type, blockSize)
  ctx.putImageData(imageData, 0, 0)

  return canvas.toDataURL('image/jpeg', 0.92)
}

export async function cropImage(src: string, rect: CropRect): Promise<string> {
  const img = await loadImage(src)

  const x = clamp(Math.round(rect.x), 0, img.naturalWidth)
  const y = clamp(Math.round(rect.y), 0, img.naturalHeight)
  const width = clamp(Math.round(rect.width), 1, img.naturalWidth - x)
  const height = clamp(Math.round(rect.height), 1, img.naturalHeight - y)

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Failed to get 2d context')

  ctx.drawImage(img, x, y, width, height, 0, 0, width, height)
  return canvas.toDataURL('image/jpeg', 0.92)
}

export async function rotateImage(
  src: string,
  times = 1,
): Promise<{ src: string; width: number; height: number }> {
  const img = await loadImage(src)
  const degree = ((times % 4) + 4) % 4
  if (degree === 0) {
    return { src, width: img.naturalWidth, height: img.naturalHeight }
  }

  const isPerpendicular = degree % 2 === 1
  const width = isPerpendicular ? img.naturalHeight : img.naturalWidth
  const height = isPerpendicular ? img.naturalWidth : img.naturalHeight

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Failed to get 2d context')

  ctx.translate(width / 2, height / 2)
  ctx.rotate((degree * 90 * Math.PI) / 180)
  ctx.drawImage(img, -img.naturalWidth / 2, -img.naturalHeight / 2)

  return {
    src: canvas.toDataURL('image/jpeg', 0.92),
    width,
    height,
  }
}

export function createEmojiSticker(emoji: string, size = 128): string {
  const canvas = document.createElement('canvas')
  canvas.width = size
  canvas.height = size
  const ctx = canvas.getContext('2d')
  if (!ctx) return ''

  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.font = `${Math.floor(size * 0.8)}px sans-serif`
  ctx.fillText(emoji, size / 2, size / 2)

  return canvas.toDataURL('image/png')
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}
