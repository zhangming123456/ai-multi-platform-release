<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { IconLeft, IconLock, IconSafe } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore } from '@/stores/permission'
import api from '@/utils/api'

interface UserDetail {
  id: string
  username: string
  email: string | null
  nickname: string
  role: string
  avatar_url: string | null
  created_at: string
}

const router = useRouter()
const route = useRoute()
const permStore = usePermissionStore()

const userId = route.params.id as string

const loading = ref(true)
const saving = ref(false)
const user = ref<UserDetail | null>(null)
const newPassword = ref('')
const oldPassword = ref('')
const isDefaultPwd = ref(false)
const canManageUsers = computed(() => permStore.hasPermission('users:change_password:write'))

async function fetchUser() {
  loading.value = true
  try {
    const [userRes, pwdRes] = await Promise.all([
      api.get<UserDetail>(`/v2/users/${userId}`),
      api.get<{ is_default_password: boolean }>(`/v2/users/${userId}/password-status`),
    ])
    user.value = userRes.data
    isDefaultPwd.value = canManageUsers.value || pwdRes.data.is_default_password
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载用户信息失败')
    router.push({ name: 'RBACUserManage' })
  } finally {
    loading.value = false
  }
}

onMounted(fetchUser)

async function changePassword() {
  if (!newPassword.value) {
    Message.warning('请输入新密码')
    return
  }
  if (!isDefaultPwd.value && !oldPassword.value) {
    Message.warning('请输入旧密码')
    return
  }
  saving.value = true
  try {
    const body: { new_password: string; old_password?: string } = {
      new_password: newPassword.value,
    }
    if (!isDefaultPwd.value) body.old_password = oldPassword.value
    await api.put(`/v2/users/${userId}/password`, body)
    Message.success('密码修改成功')
    router.push({ name: 'RBACUserManage' })
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '密码修改失败')
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push({ name: 'RBACUserManage' })
}
</script>

<template>
  <div class="page-main">
    <PageHeader :title="`修改密码 - ${user?.nickname || '...'}`" subtitle="重置或修改用户的登录密码">
      <template #actions>
        <a-button @click="goBack">
          <template #icon><IconLeft /></template>
          返回列表
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" class="w-full">
      <div class="max-w-[480px]">
        <a-card :bordered="false" class="!rounded-xl">
          <a-form layout="vertical" class="!max-w-[400px]">
            <a-form-item label="用户">
              <a-input :model-value="`${user?.nickname || '...'}  (${user?.username || '...'})`" disabled />
            </a-form-item>

            <div v-if="isDefaultPwd" class="mb-4">
              <div
                class="flex items-center gap-2 px-4 py-3 rounded-lg bg-[#30d158]/[0.06] border border-[#30d158]/[0.15]"
              >
                <IconSafe :size="16" class="text-[#30d158] shrink-0" />
                <span class="text-[13px] text-[#30d158] font-medium">当前为默认密码或管理员重置，可直接设置新密码</span>
              </div>
            </div>

            <a-form-item v-else label="旧密码" required>
              <a-input-password v-model="oldPassword" placeholder="请输入当前密码" />
            </a-form-item>

            <a-form-item label="新密码" required>
              <a-input-password v-model="newPassword" placeholder="输入新密码" />
              <template #extra>
                <span class="text-[11px] text-[#86868b]">首字符须为字母，支持大小写字母、数字及 . _ @ $</span>
              </template>
            </a-form-item>

            <a-form-item>
              <a-space>
                <a-button type="primary" :loading="saving" @click="changePassword">
                  <template #icon><IconLock /></template>
                  确认修改
                </a-button>
                <a-button @click="goBack">取消</a-button>
              </a-space>
            </a-form-item>
          </a-form>
        </a-card>
      </div>
    </a-spin>
  </div>
</template>
