export interface ModalProps {
  visible: boolean
  title: string
  width?: string | number
}

export type ModalEmits = {
  'update:visible': [value: boolean]
  close: []
}
