<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconPlus,
  IconDelete,
  IconLock,
  IconUser,
  IconUserGroup,
  IconEye,
  IconEdit,
  IconClockCircle,
  IconCheck,
  IconClose,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore, type RBACRole, type UserRoleAssignment } from '@/stores/permission'
import api from '@/utils/api'
import { useUserStore } from '@/stores/user'

const permStore = usePermissionStore()
const userStore = useUserStore()

interface UserItem {
  id: string
  username: string
  email: string | null
  nickname: string
  role: string
  is_active: boolean
  created_at: string
  updated_at: string
}

const loading = ref(false)
const users = ref<UserItem[]>([])
const keyword = ref('')
const roleFilter = ref('all')
const selectedUser = ref<UserItem | null>(null)

const userRoles = ref<UserRoleAssignment[]>([])
const userRolesLoading = ref(false)
const userPermissions = ref<Record<string, { read: boolean; write: boolean }>>({})
const userPermLoading = ref(false)

const addVisible = ref(false)
const addSaving = ref(false)
const newUser = ref({
  username: '',
  email: '',
  password: '',
  nickname: '',
  role: 'operator',
  role_ids: [] as string[],
})

const assignRoleVisible = ref(false)
const availableRoles = ref<RBACRole[]>([])
const selectedRoleId = ref('')

const editVisible = ref(false)
const editSaving = ref(false)
const editForm = ref({
  id: '',
  username: '',
  email: '',
  nickname: '',
  role: '',
})

const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdForm = ref({
  userId: '',
  username: '',
  newPassword: '',
})

const activeTab = ref<'users' | 'reviews'>('users')

interface CreationRequest {
  id: string
  requester_id: string
  requester_name: string
  username: string
  email: string | null
  nickname: string
  role: string
  status: string
  reviewer_id: string | null
  reviewer_name: string | null
  reject_reason: string | null
  created_at: string
}

const pendingReviews = ref<CreationRequest[]>([])
const reviewLoading = ref(false)
const reviewRejectVisible = ref(false)
const reviewRejectId = ref('')
const reviewRejectReason = ref('')
const reviewRejectSaving = ref(false)

const ROLE_COLORS: Record<string, string> = {
  super_admin: 'red',
  admin: 'orangered',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
  auditor: 'cyan',
}

function roleColor(roleName: string): string {
  return ROLE_COLORS[roleName] || 'arcoblue'
}

const filteredUsers = computed(() => {
  let result = users.value
  if (roleFilter.value !== 'all') {
    result = result.filter(u => u.role === roleFilter.value)
  }
  if (keyword.value) {
    const kw = keyword.value.toLowerCase()
    result = result.filter(
      u => u.username.toLowerCase().includes(kw) ||
           u.nickname.toLowerCase().includes(kw) ||
           (u.email && u.email.toLowerCase().includes(kw))
    )
  }
  return result
})

const columns = [
  { title: '用户名', dataIndex: 'username', width: 140 },
  { title: '昵称', dataIndex: 'nickname', width: 120 },
  { title: '邮箱', dataIndex: 'email', slotName: 'email', width: 200 },
  { title: '主角色', dataIndex: 'role', slotName: 'role', width: 120 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt', width: 180 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 200 },
]

const pendingUsernames = computed(() => new Set(pendingReviews.value.map(r => r.username)))

function getUserStatus(user: UserItem): { label: string; color: string } {
  if (!user.is_active) return { label: '已禁用', color: 'gray' }
  if (pendingUsernames.value.has(user.username)) return { label: '待审核', color: 'orangered' }
  return { label: '正常', color: 'green' }
}

async function loadUsers() {
  loading.value = true
  try {
    const [usersRes] = await Promise.all([
      api.get<UserItem[]>('/v2/users'),
      permStore.fetchRoles(),
    ])
    users.value = Array.isArray(usersRes.data) ? usersRes.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载用户列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadUsers()
  if (canReviewCreations.value) {
    loadPendingReviews()
  }
})

async function selectUser(user: UserItem) {
  selectedUser.value = user
  await Promise.all([
    loadUserRoles(user.id),
    loadUserPermissions(user.id),
  ])
}

async function loadUserRoles(userId: string) {
  userRolesLoading.value = true
  try {
    userRoles.value = await permStore.fetchUserRoles(userId)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载用户角色失败')
  } finally {
    userRolesLoading.value = false
  }
}

async function loadUserPermissions(userId: string) {
  userPermLoading.value = true
  try {
    const res = await api.get<{ user_id: string; permissions: Record<string, { read: boolean; write: boolean }> }>(
      `/v2/users/${userId}/permissions`
    )
    userPermissions.value = res.data.permissions
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载用户权限失败')
  } finally {
    userPermLoading.value = false
  }
}

function validatePasswordFrontend(password: string): { valid: boolean; message: string } {
  if (!password) return { valid: true, message: '' }
  if (password.length < 6) return { valid: false, message: '密码长度至少6位' }
  if (!/^[a-zA-Z]/.test(password)) return { valid: false, message: '密码首字符必须为字母' }
  if (!/^[a-zA-Z][a-zA-Z0-9._@$]*$/.test(password)) return { valid: false, message: '密码只能包含字母、数字和 . _ @ $ 字符' }
  return { valid: true, message: '' }
}

function defaultPasswordHint(username: string): string {
  return username ? `${username}123` : '用户名 + 123'
}

async function createUser() {
  if (!newUser.value.username || !newUser.value.nickname) return
  const password = newUser.value.password || defaultPasswordHint(newUser.value.username)
  const validation = validatePasswordFrontend(password)
  if (!validation.valid) {
    Message.warning(validation.message)
    return
  }
  addSaving.value = true
  try {
    const res = await api.post('/v2/users', {
      username: newUser.value.username,
      password: newUser.value.password || undefined,
      nickname: newUser.value.nickname,
      email: newUser.value.email || undefined,
      role: newUser.value.role,
      role_ids: newUser.value.role_ids,
    })
    if (res.status === 202) {
      Message.success('账号创建申请已提交审核，请等待管理员审批')
    } else {
      Message.success('用户创建成功')
    }
    addVisible.value = false
    newUser.value = { username: '', email: '', password: '', nickname: '', role: 'operator', role_ids: [] }
    await loadUsers()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '创建失败')
  } finally {
    addSaving.value = false
  }
}

function openAssignRole(user: UserItem) {
  selectedUser.value = user
  selectedRoleId.value = ''
  availableRoles.value = permStore.roles.filter(
    r => !userRoles.value.some(ur => ur.role_id === r.id)
  )
  assignRoleVisible.value = true
}

async function assignRole() {
  if (!selectedRoleId.value || !selectedUser.value) return
  try {
    await permStore.assignRoleToUser(selectedUser.value.id, selectedRoleId.value)
    Message.success('角色分配成功')
    assignRoleVisible.value = false
    await loadUserRoles(selectedUser.value.id)
    await loadUserPermissions(selectedUser.value.id)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '分配失败')
  }
}

async function revokeRole(roleId: string) {
  if (!selectedUser.value) return
  try {
    await permStore.revokeRoleFromUser(selectedUser.value.id, roleId)
    Message.success('角色已撤销')
    await loadUserRoles(selectedUser.value.id)
    await loadUserPermissions(selectedUser.value.id)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '撤销失败')
  }
}

function openEdit(user: UserItem) {
  editForm.value = {
    id: user.id,
    username: user.username,
    email: user.email || '',
    nickname: user.nickname,
    role: user.role,
  }
  editVisible.value = true
}

async function saveEdit() {
  if (!editForm.value.id || !editForm.value.nickname) return
  editSaving.value = true
  try {
    const res = await api.put<UserItem>(`/v2/users/${editForm.value.id}`, {
      nickname: editForm.value.nickname,
      email: editForm.value.email || undefined,
    })
    const idx = users.value.findIndex((u) => u.id === editForm.value.id)
    if (idx !== -1) {
      users.value[idx] = res.data
    }
    if (selectedUser.value?.id === editForm.value.id) {
      selectedUser.value = res.data
    }
    Message.success('用户已更新')
    editVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    editSaving.value = false
  }
}

function openPwdChange(user: UserItem) {
  pwdForm.value = {
    userId: user.id,
    username: user.username,
    newPassword: '',
  }
  pwdVisible.value = true
}

async function changePassword() {
  if (!pwdForm.value.userId || !pwdForm.value.newPassword) return
  const validation = validatePasswordFrontend(pwdForm.value.newPassword)
  if (!validation.valid) {
    Message.warning(validation.message)
    return
  }
  pwdSaving.value = true
  try {
    await api.put(`/v2/users/${pwdForm.value.userId}/password`, {
      new_password: pwdForm.value.newPassword,
    })
    Message.success('密码修改成功')
    pwdVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '密码修改失败')
  } finally {
    pwdSaving.value = false
  }
}

async function removeUser(user: UserItem) {
  try {
    await api.delete(`/v2/users/${user.id}`)
    users.value = users.value.filter((u) => u.id !== user.id)
    if (selectedUser.value?.id === user.id) {
      selectedUser.value = null
    }
    Message.success('用户已删除')
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '删除失败')
  }
}

async function loadPendingReviews() {
  reviewLoading.value = true
  try {
    const res = await api.get<CreationRequest[]>('/user-creation-reviews/')
    pendingReviews.value = Array.isArray(res.data) ? res.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载审核列表失败')
    pendingReviews.value = []
  } finally {
    reviewLoading.value = false
  }
}

async function approveReview(id: string) {
  try {
    await api.post(`/user-creation-reviews/${id}/approve`)
    Message.success('账号创建申请已通过，用户已创建')
    await loadPendingReviews()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '审批失败')
  }
}

function openRejectReview(id: string) {
  reviewRejectId.value = id
  reviewRejectReason.value = ''
  reviewRejectVisible.value = true
}

async function confirmRejectReview() {
  if (!reviewRejectReason.value.trim()) {
    Message.warning('请输入驳回原因')
    return
  }
  reviewRejectSaving.value = true
  try {
    await api.post(`/user-creation-reviews/${reviewRejectId.value}/reject`, {
      reason: reviewRejectReason.value,
    })
    Message.success('已驳回该申请')
    reviewRejectVisible.value = false
    await loadPendingReviews()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '驳回失败')
  } finally {
    reviewRejectSaving.value = false
  }
}

function onTabChange(key: string | number) {
  if (key === 'reviews' && pendingReviews.value.length === 0) {
    loadPendingReviews()
  }
}

const permissionGroups = computed(() => {
  const groups: Record<string, Array<{ key: string; read: boolean; write: boolean }>> = {}
  for (const [key, access] of Object.entries(userPermissions.value)) {
    const resourceKey = key.split(':')[0]
    if (!groups[resourceKey]) groups[resourceKey] = []
    groups[resourceKey].push({ key, ...access })
  }
  return groups
})

const canCreateUser = computed(() => permStore.hasPermission('users:create', 'write'))
const canUpdateUser = computed(() => permStore.hasPermission('users:update', 'write'))
const canDeleteUser = computed(() => permStore.hasPermission('users:delete', 'write'))
const canChangePassword = computed(() => permStore.hasPermission('users:change_password', 'write'))
const canAssignRole = computed(() => permStore.hasPermission('users:assign_role', 'write'))
const canReviewCreations = computed(() => permStore.hasPermission('user_creation_review:read', 'read'))
</script>

<template>
  <div>
    <PageHeader title="用户管理 (RBAC3)" subtitle="管理系统用户及其角色分配，支持多角色和精细化权限">
      <template #actions>
        <a-button type="primary" @click="addVisible = true" :disabled="!canCreateUser">
          <template #icon><IconPlus /></template>
          创建用户
        </a-button>
      </template>
    </PageHeader>

    <a-tabs v-model:active-key="activeTab" type="rounded" class="rbac-user-tabs" @change="onTabChange">
      <a-tab-pane key="users" title="用户列表">
        <a-spin :loading="loading" class="w-full">
          <div class="flex flex-col lg:flex-row gap-5">
            <div class="flex-1 min-w-0">
              <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-4 mb-4">
                <div class="flex flex-wrap items-center gap-4">
                  <a-input
                    v-model="keyword"
                    placeholder="搜索用户名、昵称或邮箱"
                    allow-clear
                    style="width: 280px"
                  >
                    <template #prefix>
                      <IconUser :size="16" class="text-[#86868B]" />
                    </template>
                  </a-input>
                  <a-select v-model="roleFilter" style="width: 160px">
                    <a-option value="all">全部角色</a-option>
                    <a-option v-for="role in permStore.roles" :key="role.id" :value="role.name">
                      {{ role.display_name }}
                    </a-option>
                  </a-select>
                </div>
              </div>

              <a-table
                :data="filteredUsers"
                :columns="columns"
                :pagination="{ pageSize: 10 }"
                bordered
              >
                <template #email="{ record }">
                  <span class="text-[#636366]">{{ record.email || '-' }}</span>
                </template>
                <template #role="{ record }">
                  <a-tag :color="roleColor(record.role)" size="small">
                    {{ record.role }}
                  </a-tag>
                </template>
                <template #status="{ record }">
                  <a-tag :color="getUserStatus(record).color" size="small">
                    {{ getUserStatus(record).label }}
                  </a-tag>
                </template>
                <template #createdAt="{ record }">
                  <span class="text-[#636366] text-[13px]">{{ record.created_at }}</span>
                </template>
                <template #actions="{ record }">
                  <a-space size="mini">
                    <a-button type="text" size="small" @click="selectUser(record)">
                      <template #icon><IconEye :size="14" /></template>
                      详情
                    </a-button>
                    <a-button v-if="canAssignRole" type="text" size="small" @click="openAssignRole(record)" title="分配角色">
                      <template #icon><IconUserGroup :size="14" /></template>
                      角色
                    </a-button>
                    <a-button v-if="canUpdateUser" type="text" size="small" @click="openEdit(record)">
                      <template #icon><IconEdit :size="14" /></template>
                      编辑
                    </a-button>
                    <a-button v-if="canChangePassword && record.role !== 'admin'" type="text" size="small" @click="openPwdChange(record)">
                      <template #icon><IconLock :size="14" /></template>
                      密码
                    </a-button>
                    <a-popconfirm
                      v-if="canDeleteUser && record.role !== 'admin' && record.id !== userStore.userInfo?.id"
                      content="确定要删除该用户吗？"
                      @ok="removeUser(record)"
                    >
                      <a-button type="text" size="small" status="danger">
                        <template #icon><IconDelete :size="14" /></template>
                        删除
                      </a-button>
                    </a-popconfirm>
                  </a-space>
                </template>
              </a-table>
            </div>

            <div class="lg:w-[380px] shrink-0">
              <a-spin :loading="userRolesLoading || userPermLoading">
                <div v-if="selectedUser" class="space-y-4">
                  <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden">
                    <div class="px-5 py-4 border-b border-black/[0.04] bg-black/[0.01]">
                      <div class="flex items-center gap-3">
                        <div class="w-11 h-11 rounded-xl bg-gradient-to-br from-[#007aff] to-[#5856d6] flex items-center justify-center text-white font-semibold text-[15px]">
                          {{ selectedUser.nickname.charAt(0) }}
                        </div>
                        <div class="min-w-0">
                          <p class="text-[14px] font-semibold text-[#1D1D1F] m-0 truncate">{{ selectedUser.nickname }}</p>
                          <p class="text-[12px] text-[#86868B] m-0 font-mono truncate">@{{ selectedUser.username }}</p>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden">
                    <div class="flex items-center justify-between px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]">
                      <div class="flex items-center gap-2">
                        <IconUserGroup :size="16" class="text-[#007aff]" />
                        <span class="text-[14px] font-semibold text-[#1D1D1F]">已分配角色</span>
                        <a-tag size="small" color="arcoblue" class="!m-0">{{ userRoles.length }}</a-tag>
                      </div>
                      <a-button
                        v-if="canAssignRole"
                        type="outline"
                        size="small"
                        @click="openAssignRole(selectedUser)"
                      >
                        <template #icon><IconPlus /></template>
                        分配
                      </a-button>
                    </div>
                    <div class="p-4">
                      <div v-if="userRoles.length === 0" class="text-center py-6 text-[#86868B] text-[13px]">
                        暂无分配的角色
                      </div>
                      <div v-else class="space-y-2">
                        <div
                          v-for="ur in userRoles"
                          :key="ur.id"
                          class="flex items-center justify-between px-3 py-2.5 rounded-xl bg-black/[0.02] border border-black/[0.04]"
                        >
                          <div class="flex items-center gap-2">
                            <a-tag :color="roleColor(ur.role_name)" size="small" class="!m-0">
                              {{ ur.role_display_name }}
                            </a-tag>
                          </div>
                          <a-button
                            v-if="canAssignRole && ur.role_name !== 'super_admin'"
                            type="text"
                            size="mini"
                            status="danger"
                            @click="revokeRole(ur.role_id)"
                          >
                            <template #icon><IconDelete :size="12" /></template>
                          </a-button>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden">
                    <div class="flex items-center gap-2 px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]">
                      <IconLock :size="16" class="text-[#ff9500]" />
                      <span class="text-[14px] font-semibold text-[#1D1D1F]">有效权限</span>
                      <a-tag size="small" color="orangered" class="!m-0">
                        {{ Object.keys(userPermissions).length }}
                      </a-tag>
                    </div>
                    <div class="p-4 max-h-[400px] overflow-y-auto">
                      <div v-if="Object.keys(userPermissions).length === 0" class="text-center py-6 text-[#86868B] text-[13px]">
                        暂无权限
                      </div>
                      <div v-else class="space-y-3">
                        <div v-for="(perms, group) in permissionGroups" :key="group">
                          <p class="text-[12px] text-[#86868B] font-medium mb-2 px-1">{{ group }}</p>
                          <div class="flex flex-wrap gap-1.5">
                            <span
                              v-for="p in perms"
                              :key="p.key"
                              :class="[
                                'text-[11px] px-2 py-1 rounded-md',
                                p.read || p.write
                                  ? 'bg-[#30d158]/[0.08] text-[#30d158] border border-[#30d158]/[0.15]'
                                  : 'bg-black/[0.03] text-[#86868B]'
                              ]"
                            >
                              {{ p.key.split(':').pop() }}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div v-else class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-8 text-center">
                  <IconUser :size="32" class="text-[#c7c7cc] mx-auto mb-3" />
                  <p class="text-[13px] text-[#86868B]">选择用户查看详情</p>
                </div>
              </a-spin>
            </div>
          </div>
        </a-spin>
      </a-tab-pane>

      <a-tab-pane v-if="canReviewCreations" key="reviews" title="创建审核">
        <a-spin :loading="reviewLoading" class="w-full">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden">
            <div class="px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01] flex items-center justify-between">
              <div class="flex items-center gap-2">
                <IconClockCircle :size="16" class="text-[#ff9500]" />
                <span class="text-[14px] font-semibold text-[#1D1D1F]">待审核账号创建申请</span>
                <a-tag size="small" color="orangered" class="!m-0">{{ pendingReviews.length }}</a-tag>
              </div>
            </div>
            <a-table
              :data="pendingReviews"
              :columns="[
                { title: '申请人', dataIndex: 'requester_name', width: 120 },
                { title: '申请账号', dataIndex: 'username', width: 140 },
                { title: '昵称', dataIndex: 'nickname', width: 120 },
                { title: '邮箱', dataIndex: 'email', slotName: 'email', width: 180 },
                { title: '分配角色', dataIndex: 'role', slotName: 'role', width: 120 },
                { title: '申请时间', dataIndex: 'created_at', slotName: 'createdAt', width: 180 },
                { title: '操作', slotName: 'actions', align: 'right' as const, width: 180 },
              ]"
              :pagination="{ pageSize: 10 }"
              bordered
            >
              <template #email="{ record }">
                <span class="text-[#636366]">{{ record.email || '-' }}</span>
              </template>
              <template #role="{ record }">
                <a-tag :color="roleColor(record.role)" size="small">
                  {{ record.role }}
                </a-tag>
              </template>
              <template #createdAt="{ record }">
                <span class="text-[#636366] text-[13px]">{{ record.created_at }}</span>
              </template>
              <template #actions="{ record }">
                <a-space size="mini">
                  <a-button type="primary" size="small" @click="approveReview(record.id)">
                    <template #icon><IconCheck :size="12" /></template>
                    通过
                  </a-button>
                  <a-button status="danger" size="small" @click="openRejectReview(record.id)">
                    <template #icon><IconClose :size="12" /></template>
                    驳回
                  </a-button>
                </a-space>
              </template>
              <template #empty>
                <a-empty description="暂无待审核的账号创建申请" />
              </template>
            </a-table>
          </div>
        </a-spin>
      </a-tab-pane>
    </a-tabs>

    <a-modal
      v-model:visible="addVisible"
      title="创建用户"
      :width="520"
      @ok="createUser"
      :ok-loading="addSaving"
      ok-text="创建"
    >
      <a-form :model="newUser" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="用户名" required>
              <a-input v-model="newUser.username" placeholder="登录用户名" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="昵称" required>
              <a-input v-model="newUser.nickname" placeholder="显示名称" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="密码" required>
              <a-input v-model="newUser.password" type="password" placeholder="登录密码" />
              <template #extra>
                <span class="text-[11px] text-[#86868b]">
                  留空则默认密码为 <code class="font-mono text-[#007aff]">{{ defaultPasswordHint(newUser.username) }}</code>
                </span>
              </template>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="邮箱">
              <a-input v-model="newUser.email" placeholder="邮箱地址" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="主角色">
          <a-select v-model="newUser.role">
            <a-option v-for="role in permStore.roles" :key="role.id" :value="role.name">
              {{ role.display_name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="额外角色（RBAC3 多角色）">
          <a-select v-model="newUser.role_ids" multiple placeholder="选择额外角色">
            <a-option
              v-for="role in permStore.roles.filter(r => r.name !== newUser.role)"
              :key="role.id"
              :value="role.id"
            >
              {{ role.display_name }}
            </a-option>
          </a-select>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              用户将拥有所有分配角色的权限并集，支持角色继承
            </span>
          </template>
        </a-form-item>
        <div class="bg-[#007aff]/[0.04] border border-[#007aff]/[0.1] rounded-xl p-3 text-[12px] text-[#636366]">
          <p class="m-0 mb-1 font-medium text-[#1D1D1F]">密码策略</p>
          <p class="m-0">首字符必须为字母，支持字母、数字及 <code class="font-mono">. _ @ $</code> 字符，长度至少 6 位。</p>
        </div>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="assignRoleVisible"
      title="分配角色"
      :width="420"
      @ok="assignRole"
      ok-text="分配"
    >
      <a-form :model="{}" layout="vertical">
        <a-form-item label="选择角色">
          <a-select v-model="selectedRoleId" placeholder="选择要分配的角色">
            <a-option v-for="role in availableRoles" :key="role.id" :value="role.id">
              {{ role.display_name }} ({{ role.name }})
            </a-option>
          </a-select>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              用户将获得该角色及其所有父角色的权限
            </span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="editVisible"
      title="编辑用户"
      :width="520"
      @ok="saveEdit"
      :ok-loading="editSaving"
      ok-text="保存"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="用户名">
          <a-input v-model="editForm.username" disabled />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="昵称" required>
              <a-input v-model="editForm.nickname" placeholder="显示名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="邮箱">
              <a-input v-model="editForm.email" placeholder="邮箱地址" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="主角色">
          <a-input v-model="editForm.role" disabled />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="reviewRejectVisible"
      title="驳回账号创建申请"
      :width="460"
      @ok="confirmRejectReview"
      :ok-loading="reviewRejectSaving"
      ok-text="驳回"
      :ok-button-props="{ status: 'danger' }"
    >
      <a-form :model="{}" layout="vertical">
        <a-form-item label="驳回原因" required>
          <a-textarea
            v-model="reviewRejectReason"
            placeholder="请输入驳回原因，申请人将看到此说明"
            :auto-size="{ minRows: 3, maxRows: 5 }"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="pwdVisible"
      :title="`修改密码 - ${pwdForm.username}`"
      :width="420"
      @ok="changePassword"
      :ok-loading="pwdSaving"
      ok-text="保存"
    >
      <a-form :model="pwdForm" layout="vertical">
        <a-form-item label="新密码" required>
          <a-input v-model="pwdForm.newPassword" type="password" placeholder="输入新密码" />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              首字符必须为字母，支持字母、数字及 . _ @ $ 字符，长度至少 6 位
            </span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

  </div>
</template>
