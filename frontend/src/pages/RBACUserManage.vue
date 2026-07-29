<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconLock,
  IconRefresh,
  IconSafe,
  IconSettings,
} from '@arco-design/web-vue/es/icon'
import { Message, Modal } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore } from '@/stores/permission'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/time'
import api from '@/utils/api'

interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface UserListItem {
  id: string
  username: string
  email: string | null
  nickname: string
  role: string
  avatar_url: string | null
  created_at: string
  roles: Role[]
}

const userStore = useUserStore()
const permStore = usePermissionStore()

const loading = ref(false)
const users = ref<UserListItem[]>([])
const roles = ref<Role[]>([])

const canCreate = computed(() => permStore.hasPermission('users:create', 'write'))
const canUpdate = computed(() => permStore.hasPermission('users:update', 'write'))
const canDelete = computed(() => permStore.hasPermission('users:delete', 'write'))
const canChangePassword = computed(() => permStore.hasPermission('users:change_password', 'write'))
const canManageUsers = computed(() => permStore.hasPermission('users:write', 'write'))

const roleColorMap = computed(() => {
  const builtins: Record<string, string> = {
    admin: 'red',
    manager: 'orangered',
    operator: 'blue',
    reviewer: 'green',
  }
  const custom = ['arcoblue', 'purple', 'cyan', 'orange', 'pink', 'gold', 'lime', 'magenta']
  const map: Record<string, string> = {}
  let idx = 0
  for (const r of roles.value) {
    map[r.id] = builtins[r.name] || custom[idx++ % custom.length]
  }
  return map
})

function roleColor(roleId: string): string {
  return roleColorMap.value[roleId] || 'arcoblue'
}

const sortedRoles = computed(() => {
  return [...roles.value].sort((a, b) => {
    if (a.is_super_admin) return -1
    if (b.is_super_admin) return 1
    const order = ['manager', 'operator', 'reviewer']
    const ai = order.indexOf(a.name)
    const bi = order.indexOf(b.name)
    if (ai !== -1 && bi !== -1) return ai - bi
    if (ai !== -1) return -1
    if (bi !== -1) return 1
    return 0
  })
})

const roleOptions = computed(() =>
  sortedRoles.value.map((r) => ({ value: r.id, label: r.display_name })),
)

const columns = [
  { title: '用户名', dataIndex: 'username' },
  { title: '昵称', dataIndex: 'nickname' },
  { title: '邮箱', dataIndex: 'email', slotName: 'email' },
  { title: '角色', dataIndex: 'roles', slotName: 'roles' },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt' },
  { title: '操作', slotName: 'actions', align: 'right' as const },
]

async function fetchData() {
  loading.value = true
  try {
    const [usersRes, rolesRes] = await Promise.all([
      api.get<UserListItem[]>('/v2/users'),
      api.get<Role[]>('/v2/roles'),
    ])
    users.value = Array.isArray(usersRes.data) ? usersRes.data : []
    roles.value = Array.isArray(rolesRes.data) ? rolesRes.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载用户数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)

function isSelf(user: UserListItem | null | undefined): boolean {
  if (!user) return false
  return userStore.userInfo?.id === user.id
}

function isBuiltInAdmin(user: UserListItem | null | undefined): boolean {
  if (!user) return false
  return user.id === '1' || user.role === 'admin'
}

// Add user
const addVisible = ref(false)
const addSaving = ref(false)
const newUser = ref({
  username: '',
  email: '',
  password: '',
  nickname: '',
  avatar_url: '',
  role: 'operator',
  role_ids: [] as string[],
})

watch(
  () => newUser.value.username,
  (val) => {
    newUser.value.password = val ? `${val}123` : ''
    newUser.value.nickname = val || ''
  },
)

function openAdd() {
  newUser.value = {
    username: '',
    email: '',
    password: '',
    nickname: '',
    avatar_url: '',
    role: 'operator',
    role_ids: [],
  }
  addVisible.value = true
}

async function createUser() {
  if (!newUser.value.username || !newUser.value.password || !newUser.value.nickname) {
    Message.warning('请填写用户名、密码和昵称')
    return
  }
  addSaving.value = true
  try {
    const res = await api.post<UserListItem>('/v2/users', newUser.value)
    users.value.unshift(res.data)
    addVisible.value = false
    Message.success('用户创建成功')
  } catch (e: any) {
    if (e.response?.status === 202) {
      addVisible.value = false
      Message.success(e.response?.data?.detail || '账号创建申请已提交审核')
    } else {
      Message.error(e.response?.data?.detail || '创建失败')
    }
  } finally {
    addSaving.value = false
  }
}

// Edit user
const editVisible = ref(false)
const editSaving = ref(false)
const editingUser = ref<UserListItem | null>(null)
const editForm = ref({
  nickname: '',
  email: '',
  avatar_url: '',
  role_ids: [] as string[],
})

function openEdit(user: UserListItem) {
  editingUser.value = user
  editForm.value = {
    nickname: user.nickname,
    email: user.email || '',
    avatar_url: user.avatar_url || '',
    role_ids: user.roles.map((r) => r.id),
  }
  editVisible.value = true
}

async function saveEdit() {
  if (!editingUser.value) return
  editSaving.value = true
  try {
    const res = await api.put<UserListItem>(`/v2/users/${editingUser.value.id}`, editForm.value)
    const idx = users.value.findIndex((u) => u.id === editingUser.value!.id)
    if (idx !== -1) users.value[idx] = res.data
    Message.success('用户信息已更新')
    editVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    editSaving.value = false
  }
}

// Password
const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdLoading = ref(false)
const pwdUser = ref<UserListItem | null>(null)
const newPassword = ref('')
const oldPassword = ref('')
const isDefaultPwd = ref(false)

async function openPwdChange(user: UserListItem) {
  pwdUser.value = user
  newPassword.value = ''
  oldPassword.value = ''
  isDefaultPwd.value = false
  pwdVisible.value = true
  if (canManageUsers.value) {
    isDefaultPwd.value = true
    return
  }
  pwdLoading.value = true
  try {
    const res = await api.get<{ is_default_password: boolean }>(`/users/${user.id}/password-status`)
    isDefaultPwd.value = res.data.is_default_password
  } catch {
    isDefaultPwd.value = false
  } finally {
    pwdLoading.value = false
  }
}

async function changePassword() {
  if (!pwdUser.value || !newPassword.value) return
  if (!isDefaultPwd.value && !oldPassword.value) {
    Message.warning('请输入旧密码')
    return
  }
  pwdSaving.value = true
  try {
    const body: { new_password: string; old_password?: string } = {
      new_password: newPassword.value,
    }
    if (!isDefaultPwd.value) body.old_password = oldPassword.value
    await api.put(`/v2/users/${pwdUser.value.id}/password`, body)
    Message.success('密码修改成功')
    pwdVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '密码修改失败')
  } finally {
    pwdSaving.value = false
  }
}

function removeUser(user: UserListItem) {
  Modal.warning({
    title: '删除用户',
    content: `确定要删除用户「${user.nickname}」吗？该操作不可逆。`,
    hideCancel: false,
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      try {
        await api.delete(`/v2/users/${user.id}`)
        users.value = users.value.filter((u) => u.id !== user.id)
        Message.success('用户已删除')
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '删除失败')
      }
    },
  })
}

function onAddRoleIdsChange(value: unknown) {
  newUser.value.role_ids = Array.isArray(value) ? value.map(String) : []
}

function onEditRoleIdsChange(value: unknown) {
  editForm.value.role_ids = Array.isArray(value) ? value.map(String) : []
}

// ---- Permission Overrides (用户自定义权限) ----
interface AllPermission {
  id: string
  key: string
  operation: string
  resource: {
    id: string
    key: string
    name: string
    type: string
  }
}

interface PermOverrideEntry {
  permission_key: string
  granted: boolean
}

interface UserEffectivePerm {
  key: string
  read: boolean
  write: boolean
}

const permOverrideVisible = ref(false)
const permOverrideSaving = ref(false)
const permOverrideUser = ref<UserListItem | null>(null)
const allPermissions = ref<AllPermission[]>([])
const userRoleBasedPerms = ref<Record<string, UserEffectivePerm>>({})

// Current override state: permission_key -> { granted: bool | null }
// null = use role default, true = force grant, false = force deny
const overrideState = ref<Record<string, boolean | null>>({})

async function openPermOverride(user: UserListItem) {
  permOverrideUser.value = user
  permOverrideVisible.value = true

  // Fetch all permissions, user effective role-based permissions, and current overrides
  try {
    const [allPermsRes, userPermsRes, overridesRes] = await Promise.all([
      api.get<AllPermission[]>('/v2/permissions'),
      api.get<Record<string, UserEffectivePerm>>(`/v2/users/${user.id}/permissions`),
      api.get<PermOverrideEntry[]>(`/v2/users/${user.id}/permission-overrides`),
    ])
    allPermissions.value = Array.isArray(allPermsRes.data) ? allPermsRes.data : []
    userRoleBasedPerms.value = userPermsRes.data || {}

    const overrides = Array.isArray(overridesRes.data) ? overridesRes.data : []
    const state: Record<string, boolean | null> = {}
    for (const p of allPermissions.value) {
      const override = overrides.find((o) => o.permission_key === p.key)
      state[p.key] = override ? override.granted : null
    }
    overrideState.value = state
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载权限数据失败')
    permOverrideVisible.value = false
  }
}

async function savePermOverride() {
  if (!permOverrideUser.value) return
  permOverrideSaving.value = true
  try {
    const overrides: PermOverrideEntry[] = []
    for (const [key, granted] of Object.entries(overrideState.value)) {
      if (granted !== null) {
        overrides.push({ permission_key: key, granted })
      }
    }
    await api.put(`/v2/users/${permOverrideUser.value.id}/permission-overrides`, {
      overrides,
    })
    Message.success('用户权限已更新')
    permOverrideVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '保存权限失败')
  } finally {
    permOverrideSaving.value = false
  }
}

// Group permissions by resource key prefix
const permGroups = computed(() => {
  const groups: Record<string, { label: string; perms: AllPermission[] }> = {}
  const groupLabels: Record<string, string> = {
    dashboard: '仪表盘',
    platforms: '平台管理',
    content: '内容管理',
    publish: '发布管理',
    templates: '模板管理',
    review: '内容审核',
    sql_review: 'SQL审核',
    token_plan: 'Token方案',
    api_docs: 'API文档',
    db: '数据库',
    users: '用户管理',
    roles: '角色管理',
    permissions: '权限管理',
    constraints: '约束管理',
  }

  for (const p of allPermissions.value) {
    const prefix = p.key.split(':')[0]
    if (!groups[prefix]) {
      groups[prefix] = { label: groupLabels[prefix] || prefix, perms: [] }
    }
    groups[prefix].perms.push(p)
  }

  return Object.values(groups)
})

function hasRolePerm(key: string): string {
  const p = userRoleBasedPerms.value[key]
  if (!p) return '无'
  if (p.read && p.write) return '读写'
  if (p.write) return '写入'
  if (p.read) return '只读'
  return '无'
}
</script>

<template>
  <div class="page-main">
    <PageHeader title="用户管理" subtitle="管理系统用户账号与 RBAC3 角色分配">
      <template #actions>
        <a-button v-if="canCreate" type="primary" @click="openAdd">
          <template #icon><IconPlus /></template>
          添加账号
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" tip="加载中..." class="w-full">
      <a-table
        :columns="columns"
        :data="users"
        :bordered="false"
        :hoverable="true"
        :pagination="false"
      >
        <template #email="{ record }">
          <span class="text-[13px] text-[#86868b]">{{ record.email || '--' }}</span>
        </template>
        <template #roles="{ record }">
          <div class="flex flex-wrap gap-1.5">
            <a-tag
              v-for="role in record.roles"
              :key="role.id"
              :color="roleColor(role.id)"
              size="small"
              class="!m-0"
            >
              {{ role.display_name }}
            </a-tag>
            <span v-if="record.roles.length === 0" class="text-[13px] text-[#86868b]">未分配角色</span>
          </div>
        </template>
        <template #createdAt="{ record }">
          <span class="text-[13px] text-[#86868b]">{{ formatDateTime(record.created_at) }}</span>
        </template>
        <template #actions="{ record }">
          <a-space :size="4">
            <a-button
              v-if="canUpdate && canManageUsers && !isBuiltInAdmin(record)"
              type="text"
              size="small"
              title="自定义权限"
              @click="openPermOverride(record)"
            >
              <template #icon><IconSettings /></template>
            </a-button>
            <a-button
              v-if="canChangePassword && (canManageUsers || isSelf(record))"
              type="text"
              size="small"
              title="修改密码"
              @click="openPwdChange(record)"
            >
              <template #icon><IconLock /></template>
            </a-button>
            <a-button
              v-if="canUpdate && (canManageUsers || isSelf(record))"
              type="text"
              size="small"
              title="编辑"
              @click="openEdit(record)"
            >
              <template #icon><IconEdit /></template>
            </a-button>
            <a-button
              v-if="canUpdate && canManageUsers && !isBuiltInAdmin(record)"
              type="text"
              size="small"
              title="重置角色"
              @click="openEdit(record)"
            >
              <template #icon><IconRefresh /></template>
            </a-button>
            <a-popconfirm
              v-if="canDelete && !isBuiltInAdmin(record) && !isSelf(record)"
              content="确定要删除该用户吗？"
              @ok="removeUser(record)"
            >
              <a-button type="text" status="danger" size="small" title="删除">
                <template #icon><IconDelete /></template>
              </a-button>
            </a-popconfirm>
          </a-space>
        </template>
        <template #empty>
          <a-empty description="暂无用户" />
        </template>
      </a-table>
    </a-spin>

    <a-modal
      v-model:visible="addVisible"
      :title="canCreate ? '添加账号' : '提交账号创建申请'"
      :width="520"
      :ok-loading="addSaving"
      :ok-text="canCreate ? '添加' : '提交审核'"
      @ok="createUser"
    >
      <a-form :model="newUser" layout="vertical">
        <a-form-item label="用户名" required>
          <a-input v-model="newUser.username" placeholder="登录用户名" />
        </a-form-item>
        <a-form-item label="邮箱（选填）">
          <a-input v-model="newUser.email" type="text" placeholder="绑定邮箱后可邮箱登录" />
        </a-form-item>
        <a-form-item label="密码" required>
          <a-input-password v-model="newUser.password" placeholder="至少6位密码" />
        </a-form-item>
        <a-form-item label="昵称" required>
          <a-input v-model="newUser.nickname" placeholder="用户昵称" />
        </a-form-item>
        <a-form-item label="头像链接（选填）">
          <a-input v-model="newUser.avatar_url" placeholder="https://example.com/avatar.png" />
        </a-form-item>
        <a-form-item label="默认角色">
          <a-select v-model="newUser.role">
            <a-option v-if="canManageUsers" value="manager">管理员</a-option>
            <a-option value="operator">运营者</a-option>
            <a-option v-if="canManageUsers" value="reviewer">审核员</a-option>
          </a-select>
        </a-form-item>
        <a-form-item v-if="canManageUsers" label="RBAC3 角色分配">
          <a-select
            :model-value="newUser.role_ids"
            placeholder="选择要分配的 RBAC3 角色"
            multiple
            :options="roleOptions"
            @change="onAddRoleIdsChange"
          />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">不选择时将按默认角色自动同步</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="editVisible"
      title="编辑用户"
      :width="520"
      :ok-loading="editSaving"
      ok-text="保存"
      @ok="saveEdit"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="昵称" required>
          <a-input v-model="editForm.nickname" placeholder="用户昵称" />
        </a-form-item>
        <a-form-item label="邮箱（选填）">
          <a-input v-model="editForm.email" type="text" placeholder="user@example.com" />
        </a-form-item>
        <a-form-item label="头像链接（选填）">
          <a-input v-model="editForm.avatar_url" placeholder="https://example.com/avatar.png" />
        </a-form-item>
        <a-form-item label="RBAC3 角色分配">
          <a-select
            :model-value="editForm.role_ids"
            placeholder="选择角色"
            multiple
            :disabled="isBuiltInAdmin(editingUser) || (!canManageUsers && isSelf(editingUser))"
            :options="roleOptions"
            @change="onEditRoleIdsChange"
          />
          <template v-if="isBuiltInAdmin(editingUser)" #extra>
            <span class="text-[11px] text-[#ff3b30]">超级管理员角色不可修改</span>
          </template>
          <template v-else-if="!canManageUsers && isSelf(editingUser)" #extra>
            <span class="text-[11px] text-[#86868b]">仅管理员可修改角色分配，当前为查看模式</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="pwdVisible"
      title="修改密码"
      :width="420"
      :ok-loading="pwdSaving"
      ok-text="确认修改"
      @ok="changePassword"
    >
      <a-spin :loading="pwdLoading" class="w-full">
        <a-form :model="{}" layout="vertical">
          <a-form-item v-if="!isDefaultPwd" label="旧密码" required>
            <a-input-password v-model="oldPassword" placeholder="请输入当前密码" />
          </a-form-item>
          <a-form-item v-else>
            <div
              class="flex items-center gap-2 px-3 py-2 rounded-lg bg-[#30d158]/[0.06] border border-[#30d158]/[0.15] -mb-3"
            >
              <IconSafe :size="14" class="text-[#30d158]" />
              <span class="text-[12px] text-[#30d158] font-medium">当前为默认密码或管理员重置，可直接设置新密码</span>
            </div>
          </a-form-item>
          <a-form-item>
            <template #label>新密码 - {{ pwdUser?.nickname }}</template>
            <a-input-password v-model="newPassword" placeholder="输入新密码" />
          </a-form-item>
          <div class="text-[11px] text-[#86868b] -mt-3 mb-2 leading-relaxed">
            首字符须为字母，支持大小写字母、数字及 . _ @ $
          </div>
        </a-form>
      </a-spin>
    </a-modal>

    <!-- 用户自定义权限 Modal -->
    <a-modal
      v-model:visible="permOverrideVisible"
      :title="`自定义权限 - ${permOverrideUser?.nickname || ''}`"
      :width="640"
      :ok-loading="permOverrideSaving"
      ok-text="保存"
      @ok="savePermOverride"
    >
      <div class="text-[13px] text-[#86868b] mb-4">
        开启开关则强制授予该权限，关闭则强制禁止该权限。
        保持默认状态（不开启不关闭）则继续使用角色权限。
        角色默认权限：<span class="text-[#1D1D1F] font-medium">{{ permOverrideUser?.roles?.map(r => r.display_name).join('、') || '无' }}</span>
      </div>

      <div class="max-h-[50vh] overflow-y-auto pr-2 space-y-4">
        <div v-for="group in permGroups" :key="group.label" class="border border-[#E5E5EA] rounded-lg p-3">
          <div class="text-[13px] font-semibold text-[#1D1D1F] mb-2">{{ group.label }}</div>
          <div class="space-y-1">
            <div
              v-for="perm in group.perms"
              :key="perm.key"
              class="flex items-center justify-between py-1.5 px-2 rounded-md hover:bg-[#F5F5F7] transition-colors"
            >
              <div class="flex items-center gap-2 min-w-0">
                <span class="text-[13px] text-[#1D1D1F] truncate">{{ perm.resource.name }}{{ perm.operation !== 'read' ? ` (${perm.operation})` : '' }}</span>
              </div>
              <div class="flex items-center gap-3 shrink-0">
                <span class="text-[11px] text-[#86868b] w-8 text-right">{{ hasRolePerm(perm.key) }}</span>
                <a-switch
                  size="small"
                  :model-value="overrideState[perm.key]"
                  :disabled="permOverrideSaving"
                  @change="(val: boolean) => overrideState[perm.key] = val"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>
