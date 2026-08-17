export const FILE_URL_PATTERN = /https?:\/\/[^\s<>"'（）()，。；！？、]+/gi
export const IMAGE_URL_PATTERN = /\.(png|jpe?g|gif|webp|avif|svg|bmp|ico)(\?.*)?$/i
export const VIDEO_URL_PATTERN = /\.(mp4|mov|avi|webm|mkv|flv|wmv|m4v|ogv)(\?.*)?$/i
export const FILE_URL_EXT_PATTERN = /\.(pdf|docx?|xlsx?|pptx?|txt|csv|zip|rar|7z)(\?.*)?$/i

export type UrlFileType = 'image' | 'video' | 'file'

export function typeFromUrl(url: string): UrlFileType | null {
  if (IMAGE_URL_PATTERN.test(url)) return 'image'
  if (VIDEO_URL_PATTERN.test(url)) return 'video'
  if (FILE_URL_EXT_PATTERN.test(url)) return 'file'
  return null
}

export function extractImageUrlsFromText(text: string): string[] {
  return Array.from(new Set(text.match(FILE_URL_PATTERN) || []))
    .map((u) => u.replace(/[.,;:!?，。；：！？、]+$/, ''))
    .filter((u) => IMAGE_URL_PATTERN.test(u))
}

export function extractUrlsFromText(text: string, types: UrlFileType[]): string[] {
  return Array.from(new Set(text.match(FILE_URL_PATTERN) || []))
    .map((u) => u.replace(/[.,;:!?，。；：！？、]+$/, ''))
    .filter((u) => {
      const t = typeFromUrl(u)
      return t !== null && types.includes(t)
    })
}

export function cleanUrlsFromText(text: string, types: UrlFileType[] = ['image']): string {
  return text
    .replace(FILE_URL_PATTERN, (match) => {
      const u = match.replace(/[.,;:!?，。；：！？、]+$/, '')
      const t = typeFromUrl(u)
      return t !== null && types.includes(t) ? '' : match
    })
    .replace(/ +/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
}

export function urlFileName(url: string): string {
  const path = url.split(/[?#]/)[0]
  let name = path.split('/').pop() || url
  try {
    name = decodeURIComponent(name)
  } catch {
    /* keep original */
  }
  return name
}
