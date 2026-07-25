<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconLock,
  IconRefresh,
  IconSafe,
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

function isSelf(user: UserListItem): boolean {
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
</script>

<template>
  <div class="page-main">
    <PageHeader title="账号设置" subtitle="管理系统用户账号与 RBAC3 角色分配">
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
            :disabled="isBuiltInAdmin(editingUser)"
            :options="roleOptions"
            @change="onEditRoleIdsChange"
          />
          <template v-if="isBuiltInAdmin(editingUser)" #extra>
            <span class="text-[11px] text-[#ff3b30]">超级管理员角色不可修改</span>
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
  </div>
</template>
