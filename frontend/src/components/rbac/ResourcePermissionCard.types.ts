export interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

export interface Resource {
  id: string
  key: string
  name: string
  description: string | null
  parent_id: string | null
  is_active: boolean
  children: Resource[]
}

export interface Permission {
  id: string
  key: string
  operation: string
  is_active: boolean
  resource: {
    id: string
    key: string
    name: string
  }
}

export interface RolePermission {
  id: string
  key: string
  operation: string
  resource_id: string
  resource_key: string
  resource_name: string
  grant_type: 'direct' | 'inherited'
}

export type PermMode = 'read' | 'write'

export interface ResourcePermissionCardProps {
  resource: Resource
  depth: number
  selectedRole: Role | null
  rolePermissionMap: Map<string, RolePermission>
  permissionMap: Map<string, Permission>
  permissions: Permission[]
}

export type ResourcePermissionCardEmits = {
  (e: 'togglePermission', key: string, mode: PermMode): void
  (e: 'toggleResourceRead', resource: Resource): void
  (e: 'toggleResourceWrite', resource: Resource): void
}
