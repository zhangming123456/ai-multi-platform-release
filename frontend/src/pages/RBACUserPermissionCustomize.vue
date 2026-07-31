<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message, Modal } from '@arco-design/web-vue'
import { IconLeft, IconSafe, IconRefresh } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore } from '@/stores/permission'
import api from '@/utils/api'
import { isAdminTypePermission } from '@/utils/rbac'

interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface ResourceRef {
  id: string
  key: string
  name: string
  description?: string | null
}

interface Permission {
  id: string
  key: string
  operation: string
  is_active: boolean
  resource: ResourceRef
}

function _isPageKey(key: string): boolean {
  const parts = key.split(':')
  return parts.length === 2 && (parts[1] === 'read' || parts[1] === 'write')
}

interface PermissionOverrideEntry {
  permission_key: string
  granted: boolean
}

interface ModuleItemDef {
  permId: string
  resourceId: string
  title: string
  subtitle: string
  readKey?: string
  writeKeys: string[]
}

interface ModuleDef {
  key: string
  label: string
  items: ModuleItemDef[]
}

interface UserInfo {
  id: string
  username: string
  email: string | null
  nickname: string
  role: string
  avatar_url: string | null
  roles: Role[]
}

const route = useRoute()
const router = useRouter()
const permStore = usePermissionStore()
const userId = computed(() => route.params.id as string)

const loading = ref(false)
const saving = ref(false)
const resetting = ref(false)
const targetUser = ref<UserInfo | null>(null)
const permissions = ref<Permission[]>([])
const effectivePermissions = ref<Record<string, string>>({})
const permissionOverrides = ref<PermissionOverrideEntry[]>([])
const selectedKeys = ref<Set<string>>(new Set())

const isBuiltInAdmin = computed(
  () => targetUser.value?.id === '1' || targetUser.value?.role === 'admin',
)

const canWrite = computed(() => permStore.hasPermission('users:custom_permissions:write'))

const readOnly = computed(() => isBuiltInAdmin.value || !canWrite.value)

const isNonAdminUser = computed(() => {
  const roles = targetUser.value?.roles
  if (!roles || roles.length === 0) return false
  return roles.every((r) => r.role_type === 'other')
})

function filterModuleItems(items: ModuleItemDef[]): ModuleItemDef[] {
  if (!isNonAdminUser.value) return items
  return items.filter((item) => {
    const checkKey = item.readKey || item.writeKeys[0]
    if (!checkKey) return false
    return !isAdminTypePermission(checkKey)
  })
}

const activePermissions = computed(() => {
  return permissions.value.filter((p) => p.is_active)
})

// 所有可勾选的权限 key 集合（来自 available_permissions，即角色权限闭包）
const allAvailableKeys = computed(() => {
  const keys = new Set<string>()
  for (const p of activePermissions.value) {
    keys.add(p.key)
  }
  return keys
})

// 交集模型：保存所有可勾选 key 的状态
// 勾选 = granted=true，未勾选 = granted=false
const currentOverrides = computed(() => {
  const overrides: PermissionOverrideEntry[] = []
  for (const key of allAvailableKeys.value) {
    overrides.push({
      permission_key: key,
      granted: selectedKeys.value.has(key),
    })
  }
  return overrides
})

// 初始化选中状态：
// - 有覆盖：从 permissionOverrides 中 granted=true 的 keys 初始化
// - 无覆盖：从 effectivePermissions 初始化（此时 effective = 角色权限）
function initSelectedKeys() {
  const keys = new Set<string>()
  if (permissionOverrides.value.length > 0) {
    for (const o of permissionOverrides.value) {
      if (o.granted) {
        keys.add(o.permission_key)
      }
    }
  } else {
    for (const key of Object.keys(effectivePermissions.value)) {
      keys.add(key)
    }
  }
  selectedKeys.value = keys
}

const pageItemDefs = computed<ModuleItemDef[]>(() => {
  return activePermissions.value
    .filter((p) => _isPageKey(p.key))
    .map((p) => ({
      permId: p.id,
      resourceId: p.resource.id,
      title: p.resource.name,
      subtitle: '页面访问',
      readKey: p.key,
      writeKeys: [] as string[],
    }))
})

function buildActionGroupMap(
  filterKeys?: string[],
): Map<string, { readPerm?: Permission; writePerm?: Permission }> {
  const groups = new Map<string, { readPerm?: Permission; writePerm?: Permission }>()
  for (const perm of activePermissions.value) {
    if (_isPageKey(perm.key)) continue
    const rk = perm.resource.key.split(':')[0]
    if (filterKeys && !filterKeys.includes(rk)) continue
    const parts = perm.key.split(':')
    const baseKey = `${parts[0]}:${parts[1]}`
    if (!groups.has(baseKey)) {
      groups.set(baseKey, {})
    }
    const group = groups.get(baseKey)!
    if (parts.length >= 3 && parts[2] === 'write') {
      group.writePerm = perm
    } else {
      group.readPerm = perm
    }
  }
  return groups
}

function groupMapToItems(
  groups: Map<string, { readPerm?: Permission; writePerm?: Permission }>,
): ModuleItemDef[] {
  const items: ModuleItemDef[] = []
  for (const [, group] of groups) {
    const writePerm = group.writePerm
    const readPerm = group.readPerm
    const displayPerm = writePerm || readPerm!
    items.push({
      permId: displayPerm.id,
      resourceId: displayPerm.resource.id,
      title: displayPerm.resource.name,
      subtitle: '操作权限',
      readKey: readPerm?.key,
      writeKeys: writePerm ? [writePerm.key] : [],
    })
  }
  return items
}

function actionItemsForKeys(resourceKeys: string[]): ModuleItemDef[] {
  return groupMapToItems(buildActionGroupMap(resourceKeys))
}

const modules = computed<ModuleDef[]>(() => {
  const systemItems = actionItemsForKeys([
    'users',
    'permissions',
    'roles',
    'constraints',
    'model_config',
  ])

  const contentItems = actionItemsForKeys(['content', 'templates', 'publish'])

  const reviewItems = actionItemsForKeys(['review', 'db_change'])

  const basicItems = actionItemsForKeys(['account', 'db', 'db_history'])

  const coveredKeys = new Set<string>()
  for (const items of [pageItemDefs.value, systemItems, contentItems, reviewItems, basicItems]) {
    for (const item of items) {
      if (item.readKey) coveredKeys.add(item.readKey)
      for (const k of item.writeKeys) coveredKeys.add(k)
    }
  }

  const otherGroups = buildActionGroupMap()
  const otherItems = groupMapToItems(otherGroups).filter((item) => {
    const k = item.readKey || item.writeKeys[0]
    return !coveredKeys.has(k)
  })

  const filteredPage = filterModuleItems(pageItemDefs.value)
  const filteredSystem = filterModuleItems(systemItems)
  const filteredContent = filterModuleItems(contentItems)
  const filteredReview = filterModuleItems(reviewItems)
  const filteredBasic = filterModuleItems(basicItems)
  const filteredOther = filterModuleItems(otherItems)

  const result: ModuleDef[] = [{ key: 'page', label: '页面权限', items: filteredPage }]
  if (filteredSystem.length > 0)
    result.push({ key: 'system', label: '系统设置', items: filteredSystem })
  if (filteredContent.length > 0)
    result.push({ key: 'content', label: '内容管理', items: filteredContent })
  if (filteredReview.length > 0)
    result.push({ key: 'review', label: '审核管理', items: filteredReview })
  if (filteredBasic.length > 0)
    result.push({ key: 'basic', label: '基础操作', items: filteredBasic })
  if (filteredOther.length > 0)
    result.push({ key: 'other', label: '其他权限', items: filteredOther })
  return result
})

function getModuleReadKeys(module: ModuleDef): string[] {
  const keys: string[] = []
  for (const item of module.items) {
    if (item.readKey) keys.push(item.readKey)
  }
  return keys
}

function getModuleWriteKeys(module: ModuleDef): string[] {
  const keys: string[] = []
  for (const item of module.items) {
    for (const k of item.writeKeys) keys.push(k)
  }
  return keys
}

function moduleReadChecked(module: ModuleDef): boolean {
  const keys = getModuleReadKeys(module)
  if (keys.length === 0) return false
  return keys.every((k) => selectedKeys.value.has(k))
}

function moduleReadIndeterminate(module: ModuleDef): boolean {
  const keys = getModuleReadKeys(module)
  const checkedCount = keys.filter((k) => selectedKeys.value.has(k)).length
  return checkedCount > 0 && checkedCount < keys.length
}

function moduleWriteChecked(module: ModuleDef): boolean {
  const keys = getModuleWriteKeys(module)
  if (keys.length === 0) return false
  return keys.every((k) => selectedKeys.value.has(k))
}

function moduleWriteIndeterminate(module: ModuleDef): boolean {
  const keys = getModuleWriteKeys(module)
  const checkedCount = keys.filter((k) => selectedKeys.value.has(k)).length
  return checkedCount > 0 && checkedCount < keys.length
}

function toggleModuleAllRead(module: ModuleDef, checked: boolean) {
  if (readOnly.value) return
  const newKeys = new Set(selectedKeys.value)
  for (const key of getModuleReadKeys(module)) {
    if (checked) {
      newKeys.add(key)
    } else {
      newKeys.delete(key)
    }
  }
  selectedKeys.value = newKeys
}

function toggleModuleAllWrite(module: ModuleDef, checked: boolean) {
  if (readOnly.value) return
  const newKeys = new Set(selectedKeys.value)
  for (const key of getModuleWriteKeys(module)) {
    if (checked) {
      newKeys.add(key)
    } else {
      newKeys.delete(key)
    }
  }
  selectedKeys.value = newKeys
}

function toggleReadKey(key: string) {
  if (readOnly.value) return
  const newKeys = new Set(selectedKeys.value)
  if (newKeys.has(key)) {
    newKeys.delete(key)
  } else {
    newKeys.add(key)
  }
  selectedKeys.value = newKeys
}

function toggleWriteKey(key: string) {
  if (readOnly.value) return
  const newKeys = new Set(selectedKeys.value)
  if (newKeys.has(key)) {
    newKeys.delete(key)
  } else {
    newKeys.add(key)
  }
  selectedKeys.value = newKeys
}

async function fetchUser() {
  const res = await api.get<UserInfo>(`/v2/users/${userId.value}`)
  targetUser.value = res.data
}

interface UserPermissionsData {
  effective_permissions: Record<string, string>
  available_permissions: Permission[]
}

async function fetchAllPermissions() {
  const res = await api.get<UserPermissionsData>(`/v2/users/${userId.value}/permissions`)
  effectivePermissions.value = res.data.effective_permissions || {}
  permissions.value = Array.isArray(res.data.available_permissions)
    ? res.data.available_permissions
    : []
  initSelectedKeys()
}

async function fetchOverrides() {
  const res = await api.get<PermissionOverrideEntry[]>(
    `/v2/users/${userId.value}/permission-overrides`,
  )
  permissionOverrides.value = Array.isArray(res.data) ? res.data : []
}

async function loadAll() {
  loading.value = true
  try {
    await Promise.all([fetchUser(), fetchAllPermissions(), fetchOverrides()])
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载数据失败')
  } finally {
    loading.value = false
  }
}

async function saveOverrides() {
  if (!userId.value || !canWrite.value) return
  saving.value = true
  try {
    const overrides = currentOverrides.value
    await api.put(`/v2/users/${userId.value}/permission-overrides`, { overrides })
    Message.success('自定义权限保存成功')
    await Promise.all([fetchAllPermissions(), fetchOverrides()])
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '保存失败')
  } finally {
    saving.value = false
  }
}

function resetOverrides() {
  if (!userId.value || !canWrite.value || readOnly.value) return
  Modal.warning({
    title: '重置自定义权限',
    content: '确定要移除该用户的所有自定义权限覆盖吗？移除后将恢复为角色默认权限。',
    okText: '确认重置',
    cancelText: '取消',
    hideCancel: false,
    onOk: async () => {
      resetting.value = true
      try {
        await api.delete(`/v2/users/${userId.value}/permission-overrides`)
        Message.success('已重置为角色默认权限')
        await Promise.all([fetchAllPermissions(), fetchOverrides()])
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '重置失败')
      } finally {
        resetting.value = false
      }
    },
  })
}

function goBack() {
  router.push({ name: 'RBACUserManage' })
}

onMounted(loadAll)
</script>

<template>
  <div class="page-main">
    <PageHeader
      :title="`${targetUser?.nickname || '用户'} 的自定义权限`"
      :subtitle="
        canWrite
          ? '为该用户单独配置权限覆盖，将影响其最终有效权限'
          : '查看用户的有效权限（只读模式）'
      "
    >
      <template #actions>
        <a-space>
          <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="goBack">
            <template #icon><IconLeft :size="13" /></template>
            返回用户列表
          </a-button>
          <a-button
            v-perm="{
              key: 'users:custom_permissions:write & !isBuiltInAdmin(user_id)',
              ctx: { user_id: targetUser?.id },
            }"
            type="text"
            size="mini"
            class="!text-[#007AFF] !px-0 !h-auto"
            :loading="resetting"
            :disabled="readOnly"
            @click="resetOverrides"
          >
            <template #icon><IconRefresh :size="13" /></template>
            重置权限
          </a-button>
          <a-button
            v-perm="{
              key: 'users:custom_permissions:write & !isBuiltInAdmin(user_id)',
              ctx: { user_id: targetUser?.id },
            }"
            type="text"
            size="mini"
            class="!text-[#007AFF] !px-0 !h-auto"
            :loading="saving"
            @click="saveOverrides"
          >
            保存权限
          </a-button>
        </a-space>
      </template>
    </PageHeader>

    <a-spin :loading="loading" tip="加载中..." class="w-full">
      <div
        v-if="isBuiltInAdmin"
        class="p-4 rounded-2xl bg-[#ff9500]/[0.06] border border-[#ff9500]/[0.15] flex items-start gap-3 mb-5"
      >
        <IconSafe :size="18" class="text-[#ff9500] mt-0.5 shrink-0" />
        <div>
          <p class="text-[14px] font-medium text-[#1D1D1F] m-0">超级管理员</p>
          <p class="text-[12px] text-[#86868B] m-0 mt-1">
            超级管理员拥有所有权限，不可单独配置自定义权限覆盖。
          </p>
        </div>
      </div>

      <div
        v-else-if="!canWrite"
        class="p-4 rounded-2xl bg-[#007AFF]/[0.06] border border-[#007AFF]/[0.15] flex items-start gap-3 mb-5"
      >
        <IconSafe :size="18" class="text-[#007AFF] mt-0.5 shrink-0" />
        <div>
          <p class="text-[14px] font-medium text-[#1D1D1F] m-0">查看模式</p>
          <p class="text-[12px] text-[#86868B] m-0 mt-1">
            你没有写入权限，只能查看该用户的有效权限，无法修改覆盖项。
          </p>
        </div>
      </div>

      <div class="space-y-4">
        <div
          v-for="module in modules"
          :key="module.key"
          class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5"
        >
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">{{ module.label }}</h3>
              <span class="text-[12px] text-[#86868B] font-medium">{{ module.items.length }}</span>
            </div>
            <div class="flex items-center gap-3">
              <a-checkbox
                v-if="module.items.some((i) => i.readKey)"
                :model-value="moduleReadChecked(module)"
                :indeterminate="moduleReadIndeterminate(module)"
                :disabled="readOnly"
                @change="toggleModuleAllRead(module, $event)"
              >
                读
              </a-checkbox>
              <a-checkbox
                v-if="module.items.some((i) => i.writeKeys.length > 0)"
                :model-value="moduleWriteChecked(module)"
                :indeterminate="moduleWriteIndeterminate(module)"
                :disabled="readOnly"
                @change="toggleModuleAllWrite(module, $event)"
              >
                写
              </a-checkbox>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div
              v-for="item in module.items"
              :key="item.permId"
              class="flex items-center justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
            >
              <div class="min-w-0 mr-3">
                <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">{{ item.title }}</p>
                <p class="text-[11px] text-[#86868B] m-0 truncate">{{ item.subtitle }}</p>
              </div>
              <div class="flex flex-col items-end gap-1 shrink-0">
                <a-checkbox
                  v-if="item.readKey"
                  :model-value="selectedKeys.has(item.readKey)"
                  :disabled="readOnly"
                  @change="toggleReadKey(item.readKey!)"
                >
                  读
                </a-checkbox>
                <a-checkbox
                  v-if="item.writeKeys.length > 0"
                  :model-value="item.writeKeys.some((k) => selectedKeys.has(k))"
                  :disabled="readOnly"
                  @change="toggleWriteKey(item.writeKeys[0])"
                >
                  写
                </a-checkbox>
              </div>
            </div>
          </div>
        </div>
      </div>
    </a-spin>
  </div>
</template>
