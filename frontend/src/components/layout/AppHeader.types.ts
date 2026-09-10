export interface AppHeaderProps {
  collapsed: boolean
}

export type AppHeaderEmits = {
  (e: 'toggleSidebar'): void
}

export interface BreadcrumbItem {
  label: string
  path: string
}

export type SelectChangeValue = string | number | boolean | Record<string, any> | undefined
