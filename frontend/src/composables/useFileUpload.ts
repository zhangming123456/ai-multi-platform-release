import api from '@/utils/api'

export const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp']
export const MAX_IMAGE_SIZE_MB = 10

export function isImageFile(file: File): boolean {
  return file.type.startsWith('image/')
}

export function validateImageFile(file: File): string | null {
  if (!file.type.startsWith('image/')) return '仅支持图片文件'
  if (!ALLOWED_IMAGE_TYPES.includes(file.type)) return '仅支持 jpg / jpeg / png / webp 图片格式'
  if (file.size > MAX_IMAGE_SIZE_MB * 1024 * 1024) return `图片大小不能超过 ${MAX_IMAGE_SIZE_MB}MB`
  return null
}

export function fileExt(file: File): string {
  const parts = file.name.split('.')
  return parts.length > 1 ? parts.pop()!.toUpperCase() : 'FILE'
}

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

export function filePreviewUrl(file: File): string {
  return URL.createObjectURL(file)
}

export async function compressImage(file: File, maxSize = 1600, quality = 0.8): Promise<File> {
  if (!file.type.startsWith('image/')) return file
  return new Promise((resolve) => {
    const img = new Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      URL.revokeObjectURL(url)
      let { width, height } = img
      if (width <= maxSize && height <= maxSize) {
        resolve(file)
        return
      }
      if (width > height) {
        height = Math.round((height * maxSize) / width)
        width = maxSize
      } else {
        width = Math.round((width * maxSize) / height)
        height = maxSize
      }
      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        resolve(file)
        return
      }
      ctx.drawImage(img, 0, 0, width, height)
      canvas.toBlob(
        (blob) => {
          if (!blob) {
            resolve(file)
            return
          }
          const compressed = new File([blob], file.name, { type: 'image/jpeg' })
          resolve(compressed)
        },
        'image/jpeg',
        quality,
      )
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      resolve(file)
    }
    img.src = url
  })
}

export async function uploadImageFile(
  file: File,
  onProgress?: (percent: number) => void,
): Promise<string> {
  const formData = new FormData()
  formData.append('file', file, file.name)
  const res = await api.post('/uploads', formData, {
    onUploadProgress: (e) => {
      if (onProgress && e.total && e.total > 0) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    },
  })
  return res.data?.url || ''
}

export async function fileToBase64(file: File): Promise<{ data: string; mime_type: string }> {
  const processed = file.type.startsWith('image/') ? await compressImage(file) : file
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      const base64 = result.split(',')[1] || ''
      resolve({ data: base64, mime_type: processed.type || 'application/octet-stream' })
    }
    reader.onerror = reject
    reader.readAsDataURL(processed)
  })
}
