<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconRefresh, IconDelete, IconCheck } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import SegmentedControl from '@/components/shared/SegmentedControl.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import Modal from '@/components/shared/Modal.vue'
import api from '@/utils/api'
import { formatRelativeTime } from '@/utils/time'

type Platform = 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
type AccountStatus = 'active' | 'inactive' | 'error'

interface Account {
  id: number
  user_id: number
  platform: Platform
  nickname: string
  avatar_url: string | null
  status: AccountStatus
  cookie_data: string | null
  access_token: string | null
  token_expires_at: string | null
  last_check_at: string | null
  error_message: string | null
  created_at: string
  updated_at: string
}

const platformFilter = ref('all')
const showAddModal = ref(false)
const loading = ref(false)
const submitting = ref(false)
const checkingId = ref<number | null>(null)
const deletingId = ref<number | null>(null)

const platformOptions = [
  { key: 'all', label: '全部' },
  { key: 'wechat_mp', label: '公众号' },
  { key: 'xiaohongshu', label: '小红书' },
  { key: 'douyin', label: '抖音' },
  { key: 'wechat_video', label: '视频号' },
]

const accounts = ref<Account[]>([])

const filteredAccounts = computed(() => {
  if (platformFilter.value === 'all') return accounts.value
  return accounts.value.filter((a) => a.platform === platformFilter.value)
})

const newAccount = ref({
  platform: 'wechat_mp' as Platform,
  nickname: '',
  cookie: '',
})

const platformChoices = [
  { value: 'wechat_mp', label: '微信公众号' },
  { value: 'xiaohongshu', label: '小红书' },
  { value: 'douyin', label: '抖音' },
  { value: 'wechat_video', label: '微信视频号' },
]

async function loadAccounts() {
  loading.value = true
  try {
    const { data } = await api.get<Account[]>('/accounts/')
    accounts.value = Array.isArray(data) ? data : []
  } catch (e) {
    Message.error('加载账号列表失败')
  } finally {
    loading.value = false
  }
}

async function addAccount() {
  if (!newAccount.value.nickname) return
  submitting.value = true
  try {
    const payload: Record<string, string> = {
      platform: newAccount.value.platform,
      nickname: newAccount.value.nickname,
    }
    if (newAccount.value.cookie) {
      payload.cookie_data = newAccount.value.cookie
    }
    const { data } = await api.post<Account>('/accounts/', payload)
    accounts.value.push(data)
    showAddModal.value = false
    newAccount.value = { platform: 'wechat_mp', nickname: '', cookie: '' }
    Message.success('账号添加成功')
  } catch (e) {
    Message.error('添加账号失败')
  } finally {
    submitting.value = false
  }
}

async function deleteAccount(id: number) {
  deletingId.value = id
  try {
    await api.delete(`/accounts/${id}`)
    accounts.value = accounts.value.filter((a) => a.id !== id)
    Message.success('账号已移除')
  } catch (e) {
    Message.error('移除账号失败')
  } finally {
    deletingId.value = null
  }
}

async function checkStatus(id: number) {
  checkingId.value = id
  try {
    const { data } = await api.post<{ id: number; status: AccountStatus; last_check_at: string; error_message: string | null }>(`/accounts/${id}/check`)
    const idx = accounts.value.findIndex((a) => a.id === id)
    if (idx !== -1) {
      accounts.value[idx] = {
        ...accounts.value[idx],
        status: data.status,
        last_check_at: data.last_check_at,
        error_message: data.error_message,
      }
    }
    Message.success('状态已刷新')
  } catch (e) {
    Message.error('状态检查失败')
  } finally {
    checkingId.value = null
  }
}

onMounted(() => {
  loadAccounts()
})
</script>

<template>
  <div>
    <PageHeader title="平台管理" subtitle="统一管理您在各平台的账号矩阵">
      <template #actions>
        <a-button type="primary" @click="showAddModal = true">
          <template #icon>
            <IconPlus />
          </template>
          添加账号
        </a-button>
      </template>
    </PageHeader>

    <div class="mb-5">
      <SegmentedControl v-model="platformFilter" :options="platformOptions" />
    </div>

    <a-spin :loading="loading" tip="加载中..." style="display: block; width: 100%">
      <a-empty v-if="!loading && filteredAccounts.length === 0" description="暂无账号" />
      <a-row v-else :gutter="[16, 20]">
        <a-col v-for="account in filteredAccounts" :key="account.id" :xs="24" :md="12" :lg="8">
          <a-card :bordered="false" hoverable style="padding: 20px">
            <a-space :size="12" align="start" fill>
              <PlatformIcon :platform="account.platform" size="lg" />
              <div style="flex: 1; min-width: 0">
                <div style="display: flex; align-items: center; justify-content: space-between">
                  <a-typography-text bold style="font-size: 15px">{{
                    account.nickname
                  }}</a-typography-text>
                  <StatusBadge :status="account.status" />
                </div>
                <a-typography-text type="secondary" style="font-size: 12px"
                  >粉丝 --</a-typography-text
                >
              </div>
            </a-space>
            <a-divider style="margin: 12px 0" />
            <div style="display: flex; align-items: center; justify-content: space-between">
              <a-typography-text type="disabled" style="font-size: 11px"
                >最近检查 {{ formatRelativeTime(account.last_check_at) }}</a-typography-text
              >
              <a-space :size="4">
                <a-button
                  type="text"
                  size="mini"
                  title="刷新状态"
                  :loading="checkingId === account.id"
                  @click="checkStatus(account.id)"
                >
                  <template #icon>
                    <IconRefresh />
                  </template>
                </a-button>
                <a-button
                  type="text"
                  size="mini"
                  status="danger"
                  title="移除账号"
                  :loading="deletingId === account.id"
                  @click="deleteAccount(account.id)"
                >
                  <template #icon>
                    <IconDelete />
                  </template>
                </a-button>
              </a-space>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </a-spin>

    <Modal v-model:visible="showAddModal" title="添加平台账号" width="480px">
      <a-form :model="newAccount" layout="vertical">
        <a-form-item label="选择平台">
          <a-radio-group v-model="newAccount.platform" type="button">
            <a-radio v-for="choice in platformChoices" :key="choice.value" :value="choice.value">
              <a-space :size="6" align="center">
                <PlatformIcon :platform="choice.value as 'wechat_mp'" size="sm" />
                {{ choice.label }}
              </a-space>
            </a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="账号昵称">
          <a-input v-model="newAccount.nickname" placeholder="请输入账号昵称" />
        </a-form-item>
        <a-form-item label="Cookie / 授权信息">
          <a-textarea
            v-model="newAccount.cookie"
            placeholder="粘贴该平台的登录 Cookie 或授权 Token"
            :auto-size="{ minRows: 3, maxRows: 5 }"
          />
          <template #extra> 我们将使用独立浏览器指纹与环境隔离，保障账号安全 </template>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showAddModal = false">取消</a-button>
          <a-button
            type="primary"
            :disabled="!newAccount.nickname"
            :loading="submitting"
            @click="addAccount"
          >
            <template #icon>
              <IconCheck />
            </template>
            验证并添加
          </a-button>
        </a-space>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
@media (max-width: 248px) {
  :deep(.arco-card-header) {
    padding: 8px 10px !important;
  }
  :deep(.arco-card-body) {
    padding: 10px !important;
  }
  :deep(.arco-card-header-title) {
    font-size: 13px !important;
  }
}
</style>
