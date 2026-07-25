<script setup lang="ts">
import { computed } from 'vue'
import {
  IconNav,
  IconCode,
  IconCheck,
  IconLock,
} from '@arco-design/web-vue/es/icon'

interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface Resource {
  id: string
  key: string
  name: string
  description: string | null
  type: 'page' | 'action'
  parent_id: string | null
  is_active: boolean
  children: Resource[]
}

interface Permission {
  id: string
  key: string
  operation: string
  is_active: boolean
  resource: {
    id: string
    key: string
    name: string
    type: string
  }
}

interface RolePermission {
  id: string
  key: string
  operation: string
  resource_id: string
  resource_key: string
  resource_name: string
  grant_type: 'direct' | 'inherited'
}

const props = defineProps<{
  resource: Resource
  depth: number
  selectedRole: Role | null
  rolePermissionMap: Map<string, RolePermission>
  permissionMap: Map<string, Permission>
  permissions: Permission[]
}>()

const emit = defineEmits<{
  (e: 'togglePermission', key: string, mode: 'read' | 'write'): void
  (e: 'toggleResourceRead', resource: Resource): void
  (e: 'toggleResourceWrite', resource: Resource): void
}>()

const READ_OPS = new Set(['read', 'execute'])
const WRITE_OPS = new Set(['create', 'update', 'delete', 'approve', 'reject', 'execute'])

const resourcePerms = computed(() => {
  return props.permissions.filter(
    (p) => p.resource.id === props.resource.id && p.is_active,
  )
})

const readPerm = computed(() => resourcePerms.value.find((p) => p.operation === 'read'))
const writePerm = computed(() => resourcePerms.value.find((p) => p.operation === 'write'))

function hasRead(key: string): boolean {
  const rp = props.rolePermissionMap.get(key)
  if (!rp) return false
  return READ_OPS.has(rp.operation)
}

function hasWrite(key: string): boolean {
  const rp = props.rolePermissionMap.get(key)
  if (!rp) return false
  return WRITE_OPS.has(rp.operation)
}

function isInherited(key: string): boolean {
  return props.rolePermissionMap.get(key)?.grant_type === 'inherited'
}

function isReadResource(): boolean {
  const key = `${props.resource.key}:read`
  return hasRead(key)
}

function isWriteResource(): boolean {
  return resourcePerms.value.some((p) => hasWrite(p.key))
}

function isInheritedResource(): boolean {
  const key = `${props.resource.key}:read`
  return isInherited(key)
}

function allChildrenIds(resource: Resource): string[] {
  const ids: string[] = [resource.id]
  for (const child of resource.children) {
    ids.push(...allChildrenIds(child))
  }
  return ids
}

function descendantPermissions(): Permission[] {
  const ids = allChildrenIds(props.resource)
  return props.permissions.filter(
    (p) => ids.includes(p.resource.id) && p.is_active,
  )
}

function allSelected(mode: 'read' | 'write'): boolean {
  const relevant = descendantPermissions().filter((p) =>
    mode === 'read' ? READ_OPS.has(p.operation) : WRITE_OPS.has(p.operation),
  )
  if (relevant.length === 0) return false
  return relevant.every((p) => (mode === 'read' ? hasRead(p.key) : hasWrite(p.key)))
}

function partialSelected(mode: 'read' | 'write'): boolean {
  const relevant = descendantPermissions().filter((p) =>
    mode === 'read' ? READ_OPS.has(p.operation) : WRITE_OPS.has(p.operation),
  )
  const selectedCount = relevant.filter((p) =>
    mode === 'read' ? hasRead(p.key) : hasWrite(p.key),
  ).length
  return selectedCount > 0 && selectedCount < relevant.length
}

function toggleAllRead() {
  if (props.selectedRole?.is_super_admin) return
  const nextVal = !allSelected('read')
  const perms = descendantPermissions().filter((p) => READ_OPS.has(p.operation))
  for (const p of perms) {
    const currently = hasRead(p.key)
    if (nextVal && !currently) emit('togglePermission', p.key, 'read')
    if (!nextVal && currently) emit('togglePermission', p.key, 'read')
  }
}

function toggleAllWrite() {
  if (props.selectedRole?.is_super_admin) return
  const nextVal = !allSelected('write')
  const perms = descendantPermissions().filter((p) => WRITE_OPS.has(p.operation))
  for (const p of perms) {
    const currently = hasWrite(p.key)
    if (nextVal && !currently) emit('togglePermission', p.key, 'write')
    if (!nextVal && currently) emit('togglePermission', p.key, 'write')
  }
}

function operationLabel(operation: string): string {
  const map: Record<string, string> = {
    read: '读取',
    write: '写入',
    create: '创建',
    update: '编辑',
    delete: '删除',
    approve: '通过',
    reject: '驳回',
    execute: '执行',
  }
  return map[operation] || operation
}

function resourceIcon(type: string) {
  return type === 'page' ? IconNav : IconCode
}

function cardClass(): string {
  if (props.selectedRole?.is_super_admin) {
    return 'border-[#30d158]/20 bg-[#30d158]/[0.04]'
  }
  if (isInheritedResource()) {
    return 'border-[#34c759]/20 bg-[#34c759]/[0.04]'
  }
  if (isReadResource() || isWriteResource()) {
    return 'border-[#007aff]/20 bg-[#007aff]/[0.04]'
  }
  return 'border-black/[0.06] bg-white'
}
</script>

<template>
  <div
    class="rounded-2xl border overflow-hidden transition-all duration-200"
    :class="cardClass()"
    :style="{ marginLeft: depth > 0 ? '24px' : '0' }"
  >
    <div
      class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]"
    >
      <div class="flex items-center gap-2.5 min-w-0">
        <component :is="resourceIcon(resource.type)" :size="16" class="text-[#86868b] shrink-0" />
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-[14px] font-semibold text-[#1D1D1F]">{{ resource.name }}</span>
            <span class="text-[11px] text-[#86868B]">{{ resource.key }}</span>
          </div>
          <p v-if="resource.description" class="text-[11px] text-[#86868B] m-0 mt-0.5 truncate">
            {{ resource.description }}
          </p>
        </div>
      </div>

      <div class="flex items-center gap-4 shrink-0">
        <div
          v-if="selectedRole?.is_super_admin"
          class="flex items-center gap-1.5 text-[12px] text-[#30d158] font-medium"
        >
          <IconCheck :size="12" />
          已拥有
        </div>
        <div
          v-else-if="isInheritedResource()"
          class="flex items-center gap-1.5 text-[12px] text-[#34c759] font-medium"
        >
          <IconLock :size="12" />
          继承自父角色
        </div>

        <template v-if="!selectedRole?.is_super_admin">
          <div class="flex items-center gap-3">
            <a-checkbox
              v-if="readPerm"
              :model-value="isReadResource()"
              :disabled="isInherited(readPerm.key)"
              @change="$emit('toggleResourceRead', resource)"
            >
              <span class="text-[12px]">可访问</span>
            </a-checkbox>
            <a-checkbox
              v-if="writePerm"
              :model-value="isWriteResource()"
              :disabled="isInherited(writePerm.key)"
              @change="$emit('toggleResourceWrite', resource)"
            >
              <span class="text-[12px]">可编辑</span>
            </a-checkbox>
          </div>

          <div v-if="resource.children.length > 0" class="flex items-center gap-3 pl-3 border-l border-black/[0.06]">
            <a-checkbox
              :model-value="allSelected('read')"
              :indeterminate="partialSelected('read')"
              @change="toggleAllRead"
            >
              <span class="text-[12px]">全选读</span>
            </a-checkbox>
            <a-checkbox
              :model-value="allSelected('write')"
              :indeterminate="partialSelected('write')"
              @change="toggleAllWrite"
            >
              <span class="text-[12px]">全选写</span>
            </a-checkbox>
          </div>
        </template>
      </div>
    </div>

    <div class="p-4">
      <div
        v-if="resourcePerms.length > 0"
        class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2"
      >
        <div
          v-for="perm in resourcePerms"
          :key="perm.id"
          class="flex items-center justify-between px-3.5 py-2.5 rounded-xl border transition-all duration-200"
          :class="
            selectedRole?.is_super_admin
              ? 'border-[#30d158]/15 bg-[#30d158]/[0.03]'
              : isInherited(perm.key)
                ? 'border-[#34c759]/15 bg-[#34c759]/[0.03]'
                : hasRead(perm.key) || hasWrite(perm.key)
                  ? 'border-[#007aff]/15 bg-[#007aff]/[0.03]'
                  : 'border-black/[0.05] bg-white'
          "
        >
          <div class="flex items-center gap-2 min-w-0">
            <span class="text-[12px] font-medium text-[#1D1D1F]">
              {{ operationLabel(perm.operation) }}
            </span>
            <span class="text-[10px] text-[#86868B] truncate">{{ perm.key }}</span>
          </div>

          <div class="flex items-center gap-3 shrink-0 ml-3">
            <div
              v-if="selectedRole?.is_super_admin"
              class="w-4 h-4 rounded-full bg-[#30d158] flex items-center justify-center"
            >
              <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                <path
                  d="M1 4L3.5 6.5L9 1"
                  stroke="white"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
            <template v-else>
              <div
                v-if="READ_OPS.has(perm.operation)"
                class="flex items-center gap-1 cursor-pointer"
                @click.stop="!isInherited(perm.key) && $emit('togglePermission', perm.key, 'read')"
              >
                <a-switch
                  :model-value="hasRead(perm.key)"
                  :disabled="isInherited(perm.key)"
                  size="small"
                />
                <span class="text-[11px] text-[#86868B]">读</span>
              </div>
              <div
                v-if="WRITE_OPS.has(perm.operation)"
                class="flex items-center gap-1 cursor-pointer"
                @click.stop="!isInherited(perm.key) && $emit('togglePermission', perm.key, 'write')"
              >
                <a-switch
                  :model-value="hasWrite(perm.key)"
                  :disabled="isInherited(perm.key)"
                  size="small"
                />
                <span class="text-[11px] text-[#86868B]">写</span>
              </div>
            </template>
          </div>
        </div>
      </div>

      <div v-else class="text-[12px] text-[#86868B] py-2">
        该资源暂无配置权限项
      </div>
    </div>

    <div v-if="resource.children.length > 0" class="pb-2">
      <ResourcePermissionCard
        v-for="child in resource.children"
        :key="child.id"
        :resource="child"
        :depth="depth + 1"
        :selected-role="selectedRole"
        :role-permission-map="rolePermissionMap"
        :permission-map="permissionMap"
        :permissions="permissions"
        @toggle-permission="(k: string, m: 'read' | 'write') => $emit('togglePermission', k, m)"
        @toggle-resource-read="(r: Resource) => $emit('toggleResourceRead', r)"
        @toggle-resource-write="(r: Resource) => $emit('toggleResourceWrite', r)"
      />
    </div>
  </div>
</template>
