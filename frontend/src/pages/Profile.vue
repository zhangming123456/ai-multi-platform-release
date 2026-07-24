<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconEdit,
  IconLock,
  IconExport,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { useUserStore } from '@/stores/user'
import type { UserInfo } from '@/types'
import api from '@/utils/api'
import { formatDateTime as formatDate } from '@/utils/time'

const router = useRouter()
const userStore = useUserStore()

interface ProfileInfo {
  id: number
  username: string
  email: string | null
  nickname: string
  role: string
  avatar_url: string | null
  created_at: string
}

interface RoleDef {
  name: string
  display_name: string
  description: string | null
  is_builtin: boolean
}

const user = ref<ProfileInfo | null>(null)
const loading = ref(false)

const roleDefs = ref<RoleDef[]>([])

const roleLabels = computed(() => {
  const map: Record<string, string> = {}
  for (const r of roleDefs.value) {
    map[r.name] = r.display_name
  }
  return map
})

function rLabel(name: string): string {
  return roleLabels.value[name] || name
}

onMounted(async () => {
  if (userStore.userInfo) {
    user.value = { ...userStore.userInfo }
  } else {
    loading.value = true
    try {
      const res = await api.get<ProfileInfo>('/auth/me')
      user.value = res.data
      userStore.userInfo = res.data
    } catch {
      Message.error('加载用户信息失败')
    } finally {
      loading.value = false
    }
  }

  try {
    const rolesRes = await api.get<RoleDef[]>('/roles')
    roleDefs.value = rolesRes.data
  } catch {}
})

const profileEditing = ref(false)
const profileSaving = ref(false)
const profileForm = ref({
  nickname: '',
  avatar_url: '',
})

function startEditProfile() {
  if (!user.value) return
  profileForm.value = {
    nickname: user.value.nickname,
    avatar_url: user.value.avatar_url || '',
  }
  profileEditing.value = true
}

function cancelEditProfile() {
  profileEditing.value = false
}

async function saveProfile() {
  profileSaving.value = true
  try {
    const res = await api.put<ProfileInfo>('/auth/profile', profileForm.value)
    if (user.value) {
      user.value.nickname = res.data.nickname
      user.value.avatar_url = res.data.avatar_url
    }
    userStore.userInfo = res.data as unknown as UserInfo
    Message.success('个人资料已更新')
    profileEditing.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    profileSaving.value = false
  }
}

const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

function openPwdModal() {
  pwdForm.value = { old_password: '', new_password: '', confirm_password: '' }
  pwdVisible.value = true
}

async function changePassword() {
  if (pwdForm.value.new_password !== pwdForm.value.confirm_password) {
    Message.warning('两次输入的新密码不一致')
    return
  }
  if (pwdForm.value.new_password.length < 6) {
    Message.warning('新密码至少6位')
    return
  }
  pwdSaving.value = true
  try {
    await api.put('/auth/password', {
      old_password: pwdForm.value.old_password,
      new_password: pwdForm.value.new_password,
    })
    Message.success('密码修改成功，请重新登录')
    pwdVisible.value = false
    userStore.logout()
    router.push('/login')
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '密码修改失败')
  } finally {
    pwdSaving.value = false
  }
}

function handleLogout() {
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="profile-page">
    <PageHeader title="个人资料" subtitle="管理您的账户信息与安全设置" />

    <a-spin :loading="loading" tip="加载中...">
      <div v-if="user" class="profile-content">
        <a-card :bordered="false" class="profile-info-card">
          <template #title>
            <span class="text-[16px] font-semibold">基本信息</span>
          </template>
          <template #extra>
            <a-button
              v-if="!profileEditing"
              type="text"
              size="small"
              @click="startEditProfile"
            >
              <template #icon><IconEdit /></template>
              编辑
            </a-button>
          </template>

          <template v-if="!profileEditing">
            <div class="flex items-center gap-5 mb-6">
              <a-avatar
                v-if="user.avatar_url"
                :size="64"
                :image-url="user.avatar_url"
                class="shrink-0"
              />
              <a-avatar
                v-else
                :size="64"
                class="shrink-0"
                :style="{ background: 'linear-gradient(135deg, #30d158 0%, #007aff 100%)' }"
              >
                {{ user.nickname.charAt(0).toUpperCase() }}
              </a-avatar>
              <div>
                <p class="text-[18px] font-bold text-[#1d1d1f]">{{ user.nickname }}</p>
                <p class="text-[13px] text-[#86868b] mt-0.5">
                  {{ rLabel(user.role) }}
                </p>
              </div>
            </div>
            <a-descriptions :column="1" bordered size="small">
              <a-descriptions-item label="用户名">{{ user.username }}</a-descriptions-item>
              <a-descriptions-item label="邮箱">{{ user.email || '未绑定' }}</a-descriptions-item>
              <a-descriptions-item label="昵称">{{ user.nickname }}</a-descriptions-item>
              <a-descriptions-item label="角色">
                {{ rLabel(user.role) }}
              </a-descriptions-item>
              <a-descriptions-item label="注册时间">
                {{ formatDate(user.created_at) }}
              </a-descriptions-item>
            </a-descriptions>
          </template>

          <template v-else>
            <a-form :model="profileForm" layout="vertical">
              <a-form-item label="昵称">
                <a-input v-model="profileForm.nickname" placeholder="请输入昵称" />
              </a-form-item>
              <a-form-item label="头像链接">
                <a-input v-model="profileForm.avatar_url" placeholder="https://example.com/avatar.png" />
              </a-form-item>
            </a-form>
            <div class="flex justify-end gap-2 mt-2">
              <a-button @click="cancelEditProfile">取消</a-button>
              <a-button type="primary" :loading="profileSaving" @click="saveProfile">
                保存修改
              </a-button>
            </div>
          </template>
        </a-card>

        <a-card :bordered="false" title="安全设置" class="profile-security-card">
          <div class="security-item">
            <div class="flex items-center gap-3">
              <div class="security-item__icon">
                <IconLock :size="18" />
              </div>
              <div class="flex-1">
                <p class="text-[14px] font-medium text-[#1d1d1f]">登录密码</p>
                <p class="text-[12px] text-[#86868b] mt-0.5">定期更换密码可以保护账户安全</p>
              </div>
            </div>
            <a-button type="outline" size="small" @click="openPwdModal">修改密码</a-button>
          </div>
        </a-card>

        <a-card :bordered="false" title="账户操作" class="profile-danger-card">
          <div class="security-item">
            <div class="flex items-center gap-3">
              <div class="security-item__icon security-item__icon--danger">
                <IconExport :size="18" />
              </div>
              <div class="flex-1">
                <p class="text-[14px] font-medium text-[#1d1d1f]">退出登录</p>
                <p class="text-[12px] text-[#86868b] mt-0.5">退出当前账户，返回到登录页面</p>
              </div>
            </div>
            <a-button type="outline" status="danger" size="small" @click="handleLogout">
              退出登录
            </a-button>
          </div>
        </a-card>
      </div>
    </a-spin>

    <a-modal
      v-model:visible="pwdVisible"
      title="修改密码"
      :width="420"
      @ok="changePassword"
      :ok-loading="pwdSaving"
      ok-text="确认修改"
    >
      <a-form :model="pwdForm" layout="vertical">
        <a-form-item label="旧密码">
          <a-input-password v-model="pwdForm.old_password" placeholder="请输入旧密码" />
        </a-form-item>
        <a-form-item label="新密码">
          <a-input-password v-model="pwdForm.new_password" placeholder="至少6位" />
        </a-form-item>
        <a-form-item label="确认新密码">
          <a-input-password v-model="pwdForm.confirm_password" placeholder="再次输入新密码" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped lang="scss">
.profile-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 680px;
}

.profile-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.profile-info-card {
  :deep(.arco-card-body) {
    padding: 24px;
  }
}

.profile-security-card {
  :deep(.arco-card-body) {
    padding: 16px 24px;
  }
}

.profile-danger-card {
  :deep(.arco-card-body) {
    padding: 16px 24px;
  }
}

.security-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.security-item__icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: #f0f0f5;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #636366;
  flex-shrink: 0;

  &--danger {
    background: rgba(255, 59, 48, 0.08);
    color: #ff3b30;
  }
}

@media (max-width: 248px) {
  .profile-info-card :deep(.arco-card-body),
  .profile-security-card :deep(.arco-card-body),
  .profile-danger-card :deep(.arco-card-body) {
    padding: 10px 12px !important;
  }
  .security-item {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
