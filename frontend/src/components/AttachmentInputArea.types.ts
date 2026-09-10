import type { Option } from '@/components/DropdownMenu/DropdownMenu.types'

export type AttachmentFileType = 'image' | 'video' | 'file'

export interface PendingItem {
  key: string
  file: File
  type: AttachmentFileType
  name: string
  size: number
}

export interface DisplayItem {
  key: string
  type: AttachmentFileType
  url?: string
  file?: File
  pending: boolean
  name: string
  size: number
}

export interface AttachmentInputAreaProps {
  modelValue?: string
  getText?: () => string
  fileList?: string[]
  upload?: (file: File, onProgress?: (percent: number) => void) => Promise<string>
  uploadMode?: 'auto' | 'manual'
  fileTypes?: AttachmentFileType[]
  maxCount?: number
  disabled?: boolean
  bordered?: boolean
  compact?: boolean
  placeholder?: string
  hint?: string
  maxLength?: number
  showWordLimit?: boolean
  minRows?: number
  maxRows?: number
  showTextarea?: boolean
  showImages?: boolean
  extractUrls?: boolean
  accept?: string
  enterBehavior?: 'newline' | 'send'
  theme?: 'light' | 'dark'
  materialPicker?: boolean
  toolbarOptions?: any[]
}

export type AttachmentInputAreaEmits = {
  (e: 'update:modelValue', value: string): void
  (e: 'update:fileList', value: string[]): void
  (e: 'enter'): void
  (e: 'change', payload: { total: number; pending: number }): void
  (e: 'option-click', option?: Option, index?: number): void
}
