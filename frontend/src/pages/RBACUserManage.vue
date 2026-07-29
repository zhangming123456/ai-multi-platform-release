<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconLock,
  IconEye,
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

const router = useRouter()
const userStore = useUserStore()
const permStore = usePermissionStore()

const loading = ref(false)
const users = ref<UserListItem[]>([])
const roles = ref<Role[]>([])

const canCreate = computed(() => permStore.hasPermission('users:create:write'))
const canUpdate = computed(() => permStore.hasPermission('users:update:write'))
const canDelete = computed(() => permStore.hasPermission('users:delete:write'))
const canChangePassword = computed(() => permStore.hasPermission('users:change_password:write'))
const canCustomizePermissions = computed(() => permStore.hasPermission('users:custom_permissions:write'))

function canViewPermissions(user: UserListItem): boolean {
  return canCustomizePermissions.value || isSelf(user)
}

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

const columns = [
  { title: '用户名', dataIndex: 'username' },
  { title: '昵称', dataIndex: 'nickname' },
  { title: '邮箱', dataIndex: 'email', slotName: 'email' },
  { title: '角色', dataIndex: 'roles', slotName: 'roles' },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt' },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 200 },
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

function isBuiltInAdmin(user: UserListItem): boolean {
  return user.id === '1' || user.role === 'admin'
}

function goCreate() {
  router.push({ name: 'RBACUserCreate' })
}

function goEdit(user: UserListItem) {
  router.push({ name: 'RBACUserEdit', params: { id: user.id } })
}

function goPassword(user: UserListItem) {
  router.push({ name: 'RBACUserPassword', params: { id: user.id } })
}

function goPermissions(user: UserListItem) {
  router.push({ name: 'RBACUserPermissionCustomize', params: { id: user.id } })
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
</script>

<template>
  <div class="page-main">
    <PageHeader title="用户管理" subtitle="管理系统用户账号与 RBAC3 角色分配">
      <template #actions>
        <a-button v-if="canCreate" type="primary" @click="goCreate">
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
          <a-space :size="2">
            <a-tooltip v-if="canViewPermissions(record)" content="自定义权限">
              <a-button
                type="text"
                size="small"
                @click="goPermissions(record)"
              >
                <template #icon><IconEye /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip v-if="canChangePassword" content="修改密码">
              <a-button
                type="text"
                size="small"
                @click="goPassword(record)"
              >
                <template #icon><IconLock /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip v-if="canUpdate" content="编辑用户">
              <a-button
                type="text"
                size="small"
                @click="goEdit(record)"
              >
                <template #icon><IconEdit /></template>
              </a-button>
            </a-tooltip>
            <a-popconfirm
              v-if="canDelete && !isBuiltInAdmin(record) && !isSelf(record)"
              content="确定要删除该用户吗？"
              @ok="removeUser(record)"
            >
              <a-tooltip content="删除">
                <a-button type="text" status="danger" size="small">
                  <template #icon><IconDelete /></template>
                </a-button>
              </a-tooltip>
            </a-popconfirm>
          </a-space>
        </template>
        <template #empty>
          <a-empty description="暂无用户" />
        </template>
      </a-table>
    </a-spin>
  </div>
</template>
