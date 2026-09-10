import { h } from 'vue'
import type { TriggerProps } from '@arco-design/web-vue'

type VNode = Parameters<typeof h>[0] | number

export type Option = {
  key: string
  type?: 'divider' | 'item' | 'loading' | 'empty'
  value?: string | number
  label?: VNode
  icon?: VNode
  iconSize?: string | number
  disabled?: boolean
  readonly?: boolean
  visible?: boolean
  popupVisible?: boolean
  popupVisibleValue?: boolean | string | number
  showSearch?: boolean
  searchPlaceholder?: string
  keyword?: string
  request: {
    valueKey?: string
    labelKey?: string
    loading?: boolean
    empty?: boolean
    page?: number
    pageSize?: number
    total?: number
    keyword: DropdownMenuOptions
    data: DropdownMenuOptions
  }
  triggerProps?: Partial<TriggerProps>
  onClick?: (
    option: Option,
    index: number,
    instance?: any,
  ) => void | boolean | Promise<void | boolean>
  onOptionClick?: (
    option: Option,
    index: number,
    instance?: any,
  ) => void | boolean | Promise<void | boolean>
  parent?: Option
  multiple?: boolean
  isCheck?: (option: Option) => boolean
  children?: DropdownMenuOptions | DropdownMenuChildren
}

export type DropdownMenuChildren = (
  keyword?: string,
  option?: Partial<Option>,
  index?: number,
) =>
  | DropdownMenuOptions
  | Promise<
      Partial<{
        data: DropdownMenuOptions
        page: number
        pageSize: number
        total: number
        valueKey?: string
        labelKey?: string
        response?: any
      }>
    >

export type DropdownMenuOption = Option
export type DropdownMenuOptions = Array<Partial<Option> | Partial<Option>[]>

export interface DropdownMenuProps {
  disabled?: boolean
  visible?: boolean
  iconSize?: string | number
  loadingText?: string
  emptyText?: string
  options: DropdownMenuOptions
}

export type DropdownMenuEmits = {
  (e: 'update:visible', value: boolean): void
  (e: 'option-click', option: Option, index: number, instance?: any): void
}

export type MenuItem = Option & {
  index: number
  childrenNodes?: MenuItem[]
  keywordNodes?: MenuItem[]
  dataNodes?: MenuItem[]
}

export type RequestParams = {
  valueKey?: string
  labelKey?: string
  [k: string]: any
}
