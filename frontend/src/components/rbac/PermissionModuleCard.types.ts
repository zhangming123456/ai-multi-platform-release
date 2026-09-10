export interface ModuleItem {
  id: string
  title: string
  subtitle: string
  readKey?: string
  writeKeys: string[]
}

export interface PermissionModuleCardProps {
  title: string
  count: number
  items: ModuleItem[]
  effectiveKeys: Set<string>
  inheritedKeys: Set<string>
  readonly?: boolean
}

export type PermissionModuleCardEmits = {
  (e: 'toggle-read', key: string): void
  (e: 'toggle-write', keys: string[]): void
  (e: 'select-all-read'): void
  (e: 'deselect-all-read', keys: string[]): void
  (e: 'select-all-write'): void
  (e: 'deselect-all-write', keys: string[]): void
}
