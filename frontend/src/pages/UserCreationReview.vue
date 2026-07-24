<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'
import { useUserStore } from '@/stores/user'
import { formatDateTime as formatDate } from '@/utils/time'

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

interface RoleDef {
  name: string
  display_name: string
  description: string | null
  is_builtin: boolean
  role_type: string
}

const userStore = useUserStore()
const loading = ref(false)
const requests = ref<CreationRequest[]>([])
const roleDefs = ref<RoleDef[]>([])

const dynamicRoleLabels = computed(() => {
  const map: Record<string, string> = {}
  for (const r of roleDefs.value) {
    map[r.name] = r.display_name
  }
  return map
})

function roleLabel(name: string): string {
  return dynamicRoleLabels.value[name] || name
}

const rejectVisible = ref(false)
const rejectRequestId = ref('')
const rejectReason = ref('')
const rejectSaving = ref(false)

const columns = [
  { title: '申请人', dataIndex: 'requester_name' },
  { title: '申请账号', dataIndex: 'username' },
  { title: '昵称', dataIndex: 'nickname' },
  { title: '邮箱', dataIndex: 'email', slotName: 'email' },
  { title: '分配角色', dataIndex: 'role', slotName: 'role' },
  { title: '申请时间', dataIndex: 'created_at', slotName: 'createdAt' },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 180 },
]

async function fetchRequests() {
  loading.value = true
  try {
    const res = await api.get<CreationRequest[]>('/user-creation-reviews/')
    requests.value = Array.isArray(res.data) ? res.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载审核列表失败')
    requests.value = []
  } finally {
    loading.value = false
  }
}

async function approveRequest(id: string) {
  try {
    await api.post(`/user-creation-reviews/${id}/approve`)
    Message.success('账号创建申请已通过，用户已创建')
    await fetchRequests()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '审批失败')
  }
}

function openReject(id: string) {
  rejectRequestId.value = id
  rejectReason.value = ''
  rejectVisible.value = true
}

async function confirmReject() {
  if (!rejectReason.value.trim()) {
    Message.warning('请输入驳回原因')
    return
  }
  rejectSaving.value = true
  try {
    await api.post(`/user-creation-reviews/${rejectRequestId.value}/reject`, {
      reason: rejectReason.value,
    })
    Message.success('已驳回该申请')
    rejectVisible.value = false
    await fetchRequests()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '驳回失败')
  } finally {
    rejectSaving.value = false
  }
}

onMounted(async () => {
  try {
    const rolesRes = await api.get<RoleDef[]>('/roles')
    roleDefs.value = Array.isArray(rolesRes.data) ? rolesRes.data : []
  } catch {}
  await fetchRequests()
})
</script>

<template>
  <div class="user-creation-review-page">
    <PageHeader title="用户创建审核" subtitle="审核非管理员提交的账号创建申请" />

    <a-spin :loading="loading" tip="加载中...">
      <a-table
        :columns="columns"
        :data="requests"
        :bordered="false"
        :hoverable="true"
        :pagination="false"
      >
        <template #email="{ record }">
          <span class="text-[13px] text-[#86868b]">{{ record.email || '--' }}</span>
        </template>
        <template #role="{ record }">
          <a-tag size="small" color="arcoblue">
            {{ roleLabel(record.role) }}
          </a-tag>
        </template>
        <template #createdAt="{ record }">
          <span class="text-[13px] text-[#86868b]">{{ formatDate(record.created_at) }}</span>
        </template>
        <template #actions="{ record }">
          <a-space :size="6">
            <a-button type="primary" size="small" @click="approveRequest(record.id)">
              通过
            </a-button>
            <a-button status="danger" size="small" @click="openReject(record.id)">
              驳回
            </a-button>
          </a-space>
        </template>
        <template #empty>
          <a-empty description="暂无待审核的账号创建申请" />
        </template>
      </a-table>
    </a-spin>

    <a-modal
      v-model:visible="rejectVisible"
      title="驳回申请"
      :width="440"
      :ok-loading="rejectSaving"
      @ok="confirmReject"
      ok-text="确认驳回"
    >
      <a-form layout="vertical">
        <a-form-item label="驳回原因" required>
          <a-textarea
            v-model="rejectReason"
            placeholder="请输入驳回原因"
            :max-length="500"
            show-word-limit
            :auto-size="{ minRows: 3, maxRows: 6 }"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped lang="scss">
.user-creation-review-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
