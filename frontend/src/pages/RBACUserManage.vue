<template>
  <div class="page-main">
    <PageHeader title="用户管理" subtitle="管理系统用户账号与 RBAC3 角色分配">
      <template #actions>
        <a-button
          v-perm="'users:create:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="goCreate"
        >
          <template #icon><IconPlus :size="13" /></template>
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
            <span v-if="record.roles.length === 0" class="text-[13px] text-[#86868b]"
              >未分配角色</span
            >
          </div>
        </template>
        <template #createdAt="{ record }">
          <span class="text-[13px] text-[#86868b]">{{ formatDateTime(record.created_at) }}</span>
        </template>
        <template #actions="{ record }">
          <a-space :size="2">
            <a-tooltip content="自定义权限">
              <a-button
                v-perm="{
                  key: 'users:custom_permissions:read || isSelf(user_id)',
                  ctx: { user_id: record.id },
                }"
                type="text"
                @click="goPermissions(record)"
              >
                <template #icon><IconEye /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="修改密码">
              <a-button
                v-perm="{
                  key: 'users:change_password:read || isSelf(user_id)',
                  ctx: { user_id: record.id },
                }"
                type="text"
                size="small"
                @click="goPassword(record)"
              >
                <template #icon><IconLock /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="编辑用户">
              <a-button
                v-perm="{ key: 'users:update:read||isSelf(user_id)', ctx: { user_id: record.id } }"
                type="text"
                size="small"
                @click="goEdit(record)"
              >
                <template #icon><IconEdit /></template>
              </a-button>
            </a-tooltip>
            <a-popconfirm content="确定要删除该用户吗？" @ok="removeUser(record)">
              <a-tooltip content="删除">
                <a-button
                  v-perm="{
                    key: 'users:delete:write & !isSelf(user_id) & !isBuiltInAdmin(user_id)',
                    ctx: { user_id: record.id },
                  }"
                  type="text"
                  status="danger"
                  size="small"
                >
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

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { IconPlus, IconEdit, IconDelete, IconLock, IconEye } from '@arco-design/web-vue/es/icon'
import { Message, Modal } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
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

const loading = ref(false)
const users = ref<UserListItem[]>([])
const roles = ref<Role[]>([])

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
