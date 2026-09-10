export interface ImageEditorModalProps {
  visible: boolean
  src: string
}

export type ImageEditorModalEmits = {
  (e: 'close'): void
  (e: 'confirm', blob: Blob): void
}
