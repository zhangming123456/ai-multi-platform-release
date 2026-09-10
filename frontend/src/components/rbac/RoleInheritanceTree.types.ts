export interface RoleRef {
  id: string
  name: string
  display_name: string
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

export interface InheritanceNode {
  role: RoleRef
  direct_permissions: Record<string, string>
  level: number
}

export interface InheritanceData {
  role: RoleRef
  direct_permissions: Record<string, string>
  ancestor_chain: InheritanceNode[]
  descendant_tree: InheritanceNode[]
}

export interface PermissionGroupKey {
  key: string
  name: string
}

export interface PermissionGroup {
  resource: string
  keys: PermissionGroupKey[]
}

export interface RoleInheritanceTreeProps {
  roleId: string
  roleName: string
  roleDisplayName: string
}

export type RoleInheritanceTreeEmits = {
  (e: 'close'): void
}
