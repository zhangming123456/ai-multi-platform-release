<template>
  <div class="page-main">
    <PageHeader
      :title="`编辑用户 - ${user?.nickname || '...'}`"
      :subtitle="canWrite ? '修改用户基本信息和角色分配' : '查看用户基本信息（只读模式）'"
    >
      <template #actions>
        <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="goBack">
          <template #icon><IconLeft :size="13" /></template>
          返回列表
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-spin :loading="loading" class="w-full">
        <div class="max-w-[560px]">
          <a-card :bordered="false" class="!rounded-xl">
            <a-form :model="form" layout="vertical" class="!max-w-[480px]">
              <a-form-item label="用户名">
                <a-input :model-value="user?.username" disabled />
                <template #extra>
                  <span class="text-[11px] text-[#86868b]">用户名不可修改</span>
                </template>
              </a-form-item>
              <a-form-item label="昵称" required>
                <a-input v-model="form.nickname" placeholder="用户昵称" :disabled="formDisabled" />
              </a-form-item>
              <a-form-item label="邮箱（选填）">
                <a-input
                  v-model="form.email"
                  type="text"
                  placeholder="user@example.com"
                  :disabled="formDisabled"
                />
              </a-form-item>
              <a-form-item label="头像链接（选填）">
                <a-input
                  v-model="form.avatar_url"
                  placeholder="https://example.com/avatar.png"
                  :disabled="formDisabled"
                />
              </a-form-item>
              <a-form-item label="RBAC3 角色分配">
                <a-select
                  :model-value="form.role_ids"
                  placeholder="选择角色"
                  multiple
                  value-key="id"
                  :disabled="formDisabled || user?.id === '1' || !canManageUsers"
                  :options="roleOptions"
                  @change="onRoleIdsChange"
                >
                  <a-option
                    v-for="item of roleOptions"
                    :key="item.id"
                    :value="item.id"
                    :label="item.display_name"
                  />
                </a-select>
                <template v-if="formHint" #extra>
                  <span
                    class="text-[11px]"
                    :class="user?.id === '1' ? 'text-[#ff3b30]' : 'text-[#86868b]'"
                    >{{ formHint }}</span
                  >
                </template>
              </a-form-item>
            </a-form>
          </a-card>
        </div>
      </a-spin>
    </div>
    <div
      v-if="canWrite"
      class="sticky bottom-0 z-30 border-t border-[#E5E5EA] bg-white/90 backdrop-blur-xl px-4 md:px-6 lg:px-8 py-3 flex items-center justify-between"
    >
      <a-button @click="goBack">取消</a-button>
      <a-button type="primary" :loading="saving" @click="saveEdit">
        <template #icon><IconCheck /></template>
        保存
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, unref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { IconLeft, IconCheck } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore } from '@/stores/permission'
import { useUserStore } from '@/stores/user'
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

interface UserDetail {
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
const route = useRoute()
const permStore = usePermissionStore()
const userStore = useUserStore()

const userId = route.params.id as string

const loading = ref(true)
const saving = ref(false)
const user = ref<UserDetail | null>(null)
const roles = ref<Role[]>([])
const canManageUsers = computed(() => permStore.hasPermission('users:update:write&isAdmin()'))
const canWrite = computed(() =>
  permStore.hasPermission('users:update:write || isSelf(user_id)', { user_id: userId }),
)

const form = ref({
  nickname: '',
  email: '',
  avatar_url: '',
  role_ids: [] as string[],
})

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

const roleOptions = computed(() => {
  if (user.value?.id === '1') {
    return unref(sortedRoles)
  }
  return sortedRoles.value.filter((r) => !r.is_super_admin)
})

const formDisabled = computed(() => {
  if (!user.value) return true
  if (user.value.id === '1') return false
  return !canWrite.value
})

const formHint = computed(() => {
  if (!user.value) return ''
  if (user.value.id === '1') return '超级管理员不可编辑'
  if (!canWrite.value) return '当前为只读查看模式'
  return ''
})

async function fetchUser() {
  loading.value = true
  try {
    const [userRes, rolesRes] = await Promise.all([
      api.get<UserDetail>(`/v2/users/${userId}`),
      api.get<Role[]>('/v2/roles'),
    ])
    user.value = userRes.data
    roles.value = Array.isArray(rolesRes.data) ? rolesRes.data : []
    form.value = {
      nickname: user.value.nickname,
      email: user.value.email || '',
      avatar_url: user.value.avatar_url || '',
      role_ids: user.value.roles.map((r) => r.id),
    }
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载用户信息失败')
    router.push({ name: 'RBACUserManage' })
  } finally {
    loading.value = false
  }
}

onMounted(fetchUser)

async function saveEdit() {
  if (!canWrite.value) return
  saving.value = true
  try {
    const body: any = {
      nickname: form.value.nickname,
      email: form.value.email || null,
      avatar_url: form.value.avatar_url || null,
    }
    if (canManageUsers.value && user.value?.id !== '1') {
      body.role_ids = form.value.role_ids
    }
    await api.put(`/v2/users/${userId}`, body)
    if (user.value?.id === userStore.userInfo?.id) {
      userStore.fetchUserInfo().catch(() => {})
      permStore.loadPermissions().catch(() => {})
    }
    Message.success('用户信息已更新')
    router.push({ name: 'RBACUserManage' })
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    saving.value = false
  }
}

function onRoleIdsChange(value: unknown) {
  form.value.role_ids = Array.isArray(value) ? value.map(String) : []
}

function goBack() {
  router.push({ name: 'RBACUserManage' })
}
</script>
