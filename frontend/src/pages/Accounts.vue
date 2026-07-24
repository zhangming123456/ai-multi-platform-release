<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconLock,
  IconSettings,
} from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'
import { useUserStore } from '@/stores/user'
import { formatDateTime as formatDate } from '@/utils/time'

interface UserInfo {
  id: number
  username: string
  email: string | null
  nickname: string
  role: string
  avatar_url: string | null
  created_at: string
}

interface PermissionDef {
  key: string
  name: string
  group: string
  type: string
}

interface UserPermDetail {
  user_id: number
  role: string
  role_permissions: string[]
  custom_permissions: string[] | null
  effective_permissions: string[]
}

interface RoleDef {
  name: string
  display_name: string
  description: string | null
  is_builtin: boolean
}

const userStore = useUserStore()
const loading = ref(false)
const users = ref<UserInfo[]>([])
const roleFilter = ref('all')

const canManageUserPerm = computed(() => {
  return userStore.userInfo?.permissions?.includes('user_perm_manage') ?? false
})

const canCreateUser = computed(() => {
  return userStore.userInfo?.permissions?.includes('user:create') ?? false
})

const canUpdateUser = computed(() => {
  return userStore.userInfo?.permissions?.includes('user:update') ?? false
})

const canDeleteUser = computed(() => {
  return userStore.userInfo?.permissions?.includes('user:delete') ?? false
})

const canChangePassword = computed(() => {
  return userStore.userInfo?.permissions?.includes('user:change_password') ?? false
})

const isAdmin = computed(() => {
  return userStore.userInfo?.role === 'admin'
})

function isSelf(record: UserInfo): boolean {
  return userStore.userInfo?.id === record.id
}

const filteredUsers = computed(() => {
  if (roleFilter.value === 'all') return users.value
  return users.value.filter((u) => u.role === roleFilter.value)
})

const columns = [
  { title: '用户名', dataIndex: 'username' },
  { title: '昵称', dataIndex: 'nickname' },
  { title: '邮箱', dataIndex: 'email', slotName: 'email' },
  { title: '角色', dataIndex: 'role', slotName: 'role' },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt' },
  { title: '操作', slotName: 'actions', align: 'right' as const },
]

const ROLE_LABELS: Record<string, string> = {
  admin: '管理员',
  operator: '运营者',
  reviewer: '审核员',
}

const ROLE_COLORS: Record<string, string> = {
  admin: 'red',
  operator: 'blue',
  reviewer: 'green',
}

const BUILTIN_COLORS: Record<string, string> = {
  admin: 'red',
  operator: 'blue',
  reviewer: 'green',
}

const CUSTOM_COLORS = ['arcoblue', 'purple', 'cyan', 'orange', 'pink', 'gold', 'lime', 'magenta']

const roleDefs = ref<RoleDef[]>([])

const dynamicRoleLabels = computed(() => {
  const map: Record<string, string> = {}
  for (const r of roleDefs.value) {
    map[r.name] = r.display_name
  }
  return map
})

const dynamicRoleColors = computed(() => {
  const map: Record<string, string> = {}
  let customIdx = 0
  for (const r of roleDefs.value) {
    if (BUILTIN_COLORS[r.name]) {
      map[r.name] = BUILTIN_COLORS[r.name]
    } else {
      map[r.name] = CUSTOM_COLORS[customIdx % CUSTOM_COLORS.length]
      customIdx++
    }
  }
  return map
})

function roleColor(name: string): string {
  return dynamicRoleColors.value[name] || 'arcoblue'
}

function roleLabel(name: string): string {
  return dynamicRoleLabels.value[name] || name
}

const customRoles = computed(() => roleDefs.value.filter((r) => !r.is_builtin))

onMounted(async () => {
  loading.value = true
  try {
    const [usersRes, rolesRes] = await Promise.all([
      api.get<UserInfo[]>('/users/'),
      api.get<RoleDef[]>('/roles'),
    ])
    users.value = Array.isArray(usersRes.data) ? usersRes.data : []
    roleDefs.value = Array.isArray(rolesRes.data) ? rolesRes.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载用户列表失败')
  } finally {
    loading.value = false
  }
})

const addVisible = ref(false)
const addSaving = ref(false)
const newUser = ref({
  username: '',
  email: '',
  password: '',
  nickname: '',
  role: 'operator' as string,
})

const addCustomPermEnabled = ref(false)
const addCustomPerms = ref<string[]>([])
const addRolePerms = ref<PermissionDef[]>([])
const addRolePermLoading = ref(false)

async function onAddRoleChange(role: string) {
  newUser.value.role = role
  addCustomPermEnabled.value = false
  addCustomPerms.value = []
  if (role === 'admin') return
  addRolePermLoading.value = true
  try {
    const res = await api.get<{ permissions: PermissionDef[]; role_permissions: Record<string, string[]> }>('/permissions/all')
    const roleKeys = res.data.role_permissions[role] || []
    addRolePerms.value = res.data.permissions.filter((p) => roleKeys.includes(p.key))
  } catch {
    addRolePerms.value = []
  } finally {
    addRolePermLoading.value = false
  }
}

async function addUser() {
  if (!newUser.value.username || !newUser.value.password || !newUser.value.nickname) return
  addSaving.value = true
  try {
    const res = await api.post<UserInfo>('/users/', newUser.value)
    const created = res.data
    if (addCustomPermEnabled.value && addCustomPerms.value.length > 0 && created.role !== 'admin') {
      try {
        await api.put(`/permissions/user/${created.id}`, { permissions: addCustomPerms.value })
      } catch {
        Message.warning('用户已创建，但自定义权限设置失败')
      }
    }
    users.value.unshift(created)
    addVisible.value = false
    newUser.value = { username: '', email: '', password: '', nickname: '', role: 'operator' }
    addCustomPermEnabled.value = false
    addCustomPerms.value = []
    Message.success('用户添加成功')
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '添加失败')
  } finally {
    addSaving.value = false
  }
}

const editVisible = ref(false)
const editSaving = ref(false)
const editingUser = ref<UserInfo | null>(null)
const editForm = ref({
  username: '',
  email: '',
  nickname: '',
  role: '',
  avatar_url: '',
})

function openEdit(user: UserInfo) {
  editingUser.value = user
  editForm.value = {
    username: user.username,
    email: user.email || '',
    nickname: user.nickname,
    role: user.role,
    avatar_url: user.avatar_url || '',
  }
  editVisible.value = true
}

async function saveEdit() {
  if (!editingUser.value) return
  editSaving.value = true
  try {
    const res = await api.put<UserInfo>(`/users/${editingUser.value.id}`, editForm.value)
    const idx = users.value.findIndex((u) => u.id === editingUser.value!.id)
    if (idx !== -1) {
      users.value[idx] = res.data
    }
    Message.success('用户已更新')
    editVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    editSaving.value = false
  }
}

const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdUser = ref<UserInfo | null>(null)
const newPassword = ref('')

function openPwdChange(user: UserInfo) {
  pwdUser.value = user
  newPassword.value = ''
  pwdVisible.value = true
}

async function changePassword() {
  if (!pwdUser.value || !newPassword.value) return
  pwdSaving.value = true
  try {
    await api.put(`/users/${pwdUser.value.id}/password`, { new_password: newPassword.value })
    Message.success('密码修改成功')
    pwdVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '密码修改失败')
  } finally {
    pwdSaving.value = false
  }
}

async function removeUser(user: UserInfo) {
  try {
    await api.delete(`/users/${user.id}`)
    users.value = users.value.filter((u) => u.id !== user.id)
    Message.success('用户已删除')
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '删除失败')
  }
}

const permVisible = ref(false)
const permSaving = ref(false)
const permLoading = ref(false)
const permUser = ref<UserInfo | null>(null)
const permDetail = ref<UserPermDetail | null>(null)
const permAllDefs = ref<PermissionDef[]>([])
const selectedPerms = ref<string[]>([])

const groupedRolePerms = computed(() => {
  if (!permDetail.value) return {}
  const groups: Record<string, PermissionDef[]> = {}
  for (const def of permAllDefs.value) {
    if (permDetail.value.role_permissions.includes(def.key)) {
      if (!groups[def.group]) groups[def.group] = []
      groups[def.group].push(def)
    }
  }
  return groups
})

async function openUserPerm(user: UserInfo) {
  permUser.value = user
  permVisible.value = true
  permLoading.value = true
  permDetail.value = null
  try {
    const [detailRes, allRes] = await Promise.all([
      api.get<UserPermDetail>(`/permissions/user/${user.id}`),
      api.get<{ permissions: PermissionDef[] }>('/permissions/all'),
    ])
    permDetail.value = detailRes.data
    permAllDefs.value = allRes.data.permissions
    selectedPerms.value = [...detailRes.data.effective_permissions]
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载权限信息失败')
  } finally {
    permLoading.value = false
  }
}

function togglePerm(key: string) {
  const idx = selectedPerms.value.indexOf(key)
  if (idx >= 0) {
    selectedPerms.value.splice(idx, 1)
  } else {
    selectedPerms.value.push(key)
  }
}

async function saveUserPerm() {
  if (!permUser.value) return
  permSaving.value = true
  try {
    await api.put(`/permissions/user/${permUser.value.id}`, { permissions: selectedPerms.value })
    Message.success('权限配置已保存')
    permVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '保存失败')
  } finally {
    permSaving.value = false
  }
}

async function resetUserPerm() {
  if (!permUser.value) return
  permSaving.value = true
  try {
    const res = await api.delete<UserPermDetail>(`/permissions/user/${permUser.value.id}/custom`)
    selectedPerms.value = [...res.data.effective_permissions]
    if (permDetail.value) {
      permDetail.value.custom_permissions = null
      permDetail.value.effective_permissions = res.data.effective_permissions
    }
    Message.success('已重置为角色默认权限')
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '重置失败')
  } finally {
    permSaving.value = false
  }
}

const addGroupedPerms = computed(() => {
  const groups: Record<string, PermissionDef[]> = {}
  for (const def of addRolePerms.value) {
    if (!groups[def.group]) groups[def.group] = []
    groups[def.group].push(def)
  }
  return groups
})
</script>

<template>
  <div class="accounts-page">
    <PageHeader title="账号管理" subtitle="管理系统用户账号与角色分配">
      <template #actions>
        <a-button v-if="canCreateUser" type="primary" @click="addVisible = true">
          <template #icon><IconPlus /></template>
          添加账号
        </a-button>
      </template>
    </PageHeader>

    <div class="mb-5">
      <a-space :size="10">
        <a-radio-group v-model="roleFilter" type="button" size="small">
          <a-radio value="all">全部</a-radio>
          <a-radio v-for="rd in roleDefs" :key="rd.name" :value="rd.name">
            {{ rd.display_name }}
          </a-radio>
        </a-radio-group>
      </a-space>
    </div>

    <a-spin :loading="loading" tip="加载中...">
      <a-table
        :columns="columns"
        :data="filteredUsers"
        :bordered="false"
        :hoverable="true"
        :pagination="false"
      >
        <template #email="{ record }">
          <span class="text-[13px] text-[#86868b]">{{ record.email || '--' }}</span>
        </template>
        <template #role="{ record }">
          <a-tag :color="roleColor(record.role)" size="small">
            {{ roleLabel(record.role) }}
          </a-tag>
        </template>
        <template #createdAt="{ record }">
          <span class="text-[13px] text-[#86868b]">{{ formatDate(record.created_at) }}</span>
        </template>
        <template #actions="{ record }">
          <a-space :size="4">
            <a-button
              v-if="canManageUserPerm && record.role !== 'admin'"
              type="text"
              size="small"
              title="权限配置"
              @click="openUserPerm(record)"
            >
              <template #icon><IconSettings /></template>
            </a-button>
            <a-button v-if="(isAdmin || isSelf(record)) && canChangePassword" type="text" size="small" title="修改密码" @click="openPwdChange(record)">
              <template #icon><IconLock /></template>
            </a-button>
            <a-button v-if="(isAdmin || isSelf(record)) && canUpdateUser" type="text" size="small" title="编辑" @click="openEdit(record)">
              <template #icon><IconEdit /></template>
            </a-button>
            <a-popconfirm
              v-if="canDeleteUser && record.role !== 'admin'"
              content="确定要删除该用户吗？"
              @ok="removeUser(record)"
            >
              <a-button
                type="text"
                status="danger"
                size="small"
                title="删除"
              >
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
      title="添加账号"
      :width="520"
      @ok="addUser"
      :ok-loading="addSaving"
      ok-text="添加"
    >
      <a-form :model="newUser" layout="vertical">
        <a-form-item label="用户名" required>
          <a-input v-model="newUser.username" placeholder="登录用户名" />
        </a-form-item>
        <a-form-item label="邮箱（选填）">
          <a-input v-model="newUser.email" type="email" placeholder="绑定邮箱后可邮箱登录" />
        </a-form-item>
        <a-form-item label="密码" required>
          <a-input-password v-model="newUser.password" placeholder="至少6位密码" />
        </a-form-item>
        <a-form-item label="昵称" required>
          <a-input v-model="newUser.nickname" placeholder="用户昵称" />
        </a-form-item>
        <a-form-item label="角色">
          <a-select :model-value="newUser.role" @change="onAddRoleChange">
            <a-option v-if="isAdmin" value="admin">管理员 - 全部权限</a-option>
            <a-option value="operator">运营者</a-option>
            <a-option v-if="isAdmin" value="reviewer">审核员</a-option>
            <a-option v-if="isAdmin" v-for="rd in customRoles" :key="rd.name" :value="rd.name">
              {{ rd.display_name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item v-if="newUser.role !== 'admin' && isAdmin && canManageUserPerm">
          <template #label>
            <div class="flex items-center gap-2">
              <span>自定义权限</span>
              <a-switch v-model="addCustomPermEnabled" size="small" />
            </div>
          </template>
          <template v-if="addCustomPermEnabled">
            <a-spin :loading="addRolePermLoading" class="w-full">
              <div class="w-full space-y-3 mt-1">
                <div v-for="(perms, group) in addGroupedPerms" :key="group">
                  <p class="text-[12px] text-[#86868b] font-medium mb-1.5">{{ group }}</p>
                  <div class="flex flex-wrap gap-1.5">
                    <a-checkbox
                      v-for="perm in perms"
                      :key="perm.key"
                      :model-value="addCustomPerms.includes(perm.key)"
                      @change="(val: any) => { if (val) addCustomPerms.push(perm.key); else addCustomPerms = addCustomPerms.filter(k => k !== perm.key) }"
                    >
                      {{ perm.name }}
                    </a-checkbox>
                  </div>
                </div>
                <p v-if="addRolePerms.length === 0 && !addRolePermLoading" class="text-[12px] text-[#86868b]">
                  该角色暂无可用权限
                </p>
              </div>
            </a-spin>
          </template>
          <template v-if="!addCustomPermEnabled" #extra>
            <span class="text-[11px] text-[#86868b]">开启后可在角色权限范围内自定义该用户的权限</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="editVisible"
      title="编辑用户"
      :width="440"
      @ok="saveEdit"
      :ok-loading="editSaving"
      ok-text="保存"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="用户名">
          <a-input v-model="editForm.username" placeholder="登录用户名" />
        </a-form-item>
        <a-form-item label="邮箱（选填）">
          <a-input v-model="editForm.email" placeholder="user@example.com" />
        </a-form-item>
        <a-form-item label="昵称">
          <a-input v-model="editForm.nickname" placeholder="用户昵称" />
        </a-form-item>
        <a-form-item label="头像链接">
          <a-input v-model="editForm.avatar_url" placeholder="https://example.com/avatar.png" />
        </a-form-item>
        <a-form-item label="角色">
          <a-select v-model="editForm.role" :disabled="editingUser?.role === 'admin' || !isAdmin">
            <a-option v-for="rd in roleDefs" :key="rd.name" :value="rd.name">
              {{ rd.display_name }}
            </a-option>
          </a-select>
          <template v-if="editingUser?.role === 'admin'" #extra>
            <span class="text-[11px] text-[#ff3b30]">超级管理员角色不可修改</span>
          </template>
          <template v-else-if="!isAdmin" #extra>
            <span class="text-[11px] text-[#86868b]">无权限切换角色</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="pwdVisible"
      title="修改密码"
      :width="400"
      @ok="changePassword"
      :ok-loading="pwdSaving"
      ok-text="确认修改"
    >
      <a-form layout="vertical">
        <a-form-item>
          <template #label>
            新密码 - {{ pwdUser?.nickname }}
          </template>
          <a-input-password v-model="newPassword" placeholder="请输入新密码（至少6位）" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="permVisible"
      title="权限配置"
      :width="560"
      :ok-loading="permSaving"
      @ok="saveUserPerm"
      ok-text="保存权限"
    >
      <template #title>
        <div class="flex items-center gap-2">
          <span>权限配置</span>
          <a-tag v-if="permUser" :color="roleColor(permUser.role)" size="small">
            {{ roleLabel(permUser.role) }}
          </a-tag>
          <span class="text-[13px] text-[#86868b] font-normal">{{ permUser?.nickname }}</span>
        </div>
      </template>

      <a-spin :loading="permLoading" class="w-full">
        <div v-if="permDetail" class="space-y-4">
          <div
            v-if="permDetail.custom_permissions"
            class="flex items-center justify-between px-3 py-2 rounded-lg bg-[#ff9500]/[0.06] border border-[#ff9500]/[0.15]"
          >
            <span class="text-[12px] text-[#ff9500] font-medium">已自定义权限（非角色默认）</span>
            <a-button type="text" size="mini" status="warning" @click="resetUserPerm">
              重置为默认
            </a-button>
          </div>
          <div
            v-else
            class="px-3 py-2 rounded-lg bg-[#30d158]/[0.06] border border-[#30d158]/[0.15]"
          >
            <span class="text-[12px] text-[#30d158] font-medium">当前使用角色默认权限</span>
          </div>

          <div v-for="(perms, group) in groupedRolePerms" :key="group">
            <p class="text-[13px] font-semibold text-[#1D1D1F] mb-2">{{ group }}</p>
            <div class="grid grid-cols-2 gap-1.5">
              <div
                v-for="perm in perms"
                :key="perm.key"
                :class="[
                  'flex items-center gap-2.5 px-3 py-2.5 rounded-lg border cursor-pointer transition-all duration-200',
                  selectedPerms.includes(perm.key)
                    ? 'border-[#007aff]/20 bg-[#007aff]/[0.04]'
                    : 'border-black/[0.06] hover:border-black/[0.12]',
                ]"
                @click="togglePerm(perm.key)"
              >
                <a-checkbox
                  :model-value="selectedPerms.includes(perm.key)"
                  class="pointer-events-none"
                />
                <div>
                  <p class="text-[13px] text-[#1D1D1F] m-0 leading-tight">{{ perm.name }}</p>
                  <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                    {{ perm.type === 'page' ? '页面' : '操作' }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <p class="text-[11px] text-[#86868b] text-center pt-2 border-t border-black/[0.04]">
            仅可在「{{ roleLabel(permDetail.role) }}」角色已分配的权限范围内自定义
          </p>
        </div>
      </a-spin>
    </a-modal>
  </div>
</template>

<style scoped lang="scss">
.accounts-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

@media (max-width: 248px) {
  :deep(.arco-table-th),
  :deep(.arco-table-td) {
    padding: 6px 8px !important;
    font-size: 11px !important;
  }
  :deep(.arco-table-cell) {
    font-size: 11px !important;
  }
}
</style>
