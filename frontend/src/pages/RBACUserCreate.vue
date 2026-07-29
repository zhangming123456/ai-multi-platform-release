<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { IconLeft, IconPlus } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore } from '@/stores/permission'
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
  nickname: string
  role: string
  roles: Role[]
}

const router = useRouter()
const permStore = usePermissionStore()

const saving = ref(false)
const canManageUsers = computed(() => permStore.hasPermission('users:create:write'))

const form = ref({
  username: '',
  email: '',
  password: '',
  nickname: '',
  avatar_url: '',
  role: 'operator' as string,
  role_ids: [] as string[],
})

const roles = ref<Role[]>([])

watch(
  () => form.value.username,
  (val) => {
    form.value.password = val ? `${val}123` : ''
    form.value.nickname = val || ''
  },
)

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

async function loadRoles() {
  try {
    const res = await api.get<Role[]>('/v2/roles')
    roles.value = Array.isArray(res.data) ? res.data : []
  } catch {
    // ignore
  }
}

loadRoles()

async function submit() {
  if (!form.value.username || !form.value.password || !form.value.nickname) {
    Message.warning('请填写用户名、密码和昵称')
    return
  }
  saving.value = true
  try {
    await api.post<UserListItem>('/v2/users', form.value)
    Message.success('用户创建成功')
    router.push({ name: 'RBACUserManage' })
  } catch (e: any) {
    if (e.response?.status === 202) {
      Message.success(e.response?.data?.detail || '账号创建申请已提交审核')
      router.push({ name: 'RBACUserManage' })
    } else {
      Message.error(e.response?.data?.detail || '创建失败')
    }
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

<template>
  <div class="page-main">
    <PageHeader title="创建用户" subtitle="添加新的系统用户账号">
      <template #actions>
        <a-button @click="goBack">
          <template #icon><IconLeft /></template>
          返回列表
        </a-button>
      </template>
    </PageHeader>

    <div class="max-w-[560px]">
      <a-card :bordered="false" class="!rounded-xl">
        <a-form :model="form" layout="vertical" class="!max-w-[480px]">
          <a-form-item label="用户名" required>
            <a-input v-model="form.username" placeholder="登录用户名，仅支持字母数字下划线" />
          </a-form-item>
          <a-form-item label="邮箱（选填）">
            <a-input v-model="form.email" type="text" placeholder="绑定邮箱后可邮箱登录" />
          </a-form-item>
          <a-form-item label="密码" required>
            <a-input-password v-model="form.password" placeholder="至少6位密码" />
            <template #extra>
              <span class="text-[11px] text-[#86868b]">默认 ${username}123，首字符须为字母，支持大小写字母、数字及 . _ @ $</span>
            </template>
          </a-form-item>
          <a-form-item label="昵称" required>
            <a-input v-model="form.nickname" placeholder="用户昵称" />
            <template #extra>
              <span class="text-[11px] text-[#86868b]">默认与用户名相同</span>
            </template>
          </a-form-item>
          <a-form-item label="头像链接（选填）">
            <a-input v-model="form.avatar_url" placeholder="https://example.com/avatar.png" />
          </a-form-item>
          <a-form-item label="默认角色">
            <a-select v-model="form.role">
              <a-option v-if="canManageUsers" value="manager">管理员</a-option>
              <a-option value="operator">运营者</a-option>
              <a-option v-if="canManageUsers" value="reviewer">审核员</a-option>
            </a-select>
          </a-form-item>
          <a-form-item v-if="canManageUsers" label="RBAC3 角色分配">
            <a-select
              :model-value="form.role_ids"
              placeholder="选择要分配的 RBAC3 角色"
              multiple
              :options="roleOptions"
              @change="onRoleIdsChange"
            />
            <template #extra>
              <span class="text-[11px] text-[#86868b]">不选择时将按默认角色自动同步</span>
            </template>
          </a-form-item>
          <a-form-item>
            <a-space>
              <a-button type="primary" :loading="saving" @click="submit">
                <template #icon><IconPlus /></template>
                {{ canManageUsers ? '创建用户' : '提交审核' }}
              </a-button>
              <a-button @click="goBack">取消</a-button>
            </a-space>
          </a-form-item>
        </a-form>
      </a-card>
    </div>
  </div>
</template>
