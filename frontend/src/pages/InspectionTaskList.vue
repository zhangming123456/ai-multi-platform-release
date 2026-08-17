<template>
  <div class="page-main">
    <PageHeader title="整改任务" subtitle="查看由巡店生成的整改任务与复核进度">
      <template #actions>
        <a-button
          v-perm="'inspection:create:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="goInspection"
        >
          <template #icon><IconCheck :size="13" /></template>
          去巡店
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
        <a-select
          v-model="storeFilter"
          placeholder="全部门店"
          allow-clear
          :options="storeOptions"
          class="!w-full sm:!w-56"
          @change="onSearch"
        />
        <a-select
          v-model="statusFilter"
          placeholder="全部状态"
          allow-clear
          class="!w-full sm:!w-40"
          @change="onSearch"
        >
          <a-option value="pending">待整改</a-option>
          <a-option value="rechecking">复核中</a-option>
          <a-option value="rectifying">打回整改</a-option>
          <a-option value="manual_review">转人工</a-option>
          <a-option value="rectified">已闭环</a-option>
          <a-option value="confirmed">人工闭环</a-option>
          <a-option value="rejected">未通过</a-option>
        </a-select>
      </div>

      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <a-table
          :columns="columns"
          :data="tasks"
          :bordered="false"
          :hoverable="true"
          :pagination="false"
        >
          <template #store="{ record }">
            <div class="flex items-center gap-2">
              <span class="font-medium text-[13px] text-[#1D1D1F]">{{ record.store_name }}</span>
            </div>
          </template>
          <template #status="{ record }">
            <a-tag :color="statusColor(record.status)" size="small">{{
              statusText(record.status)
            }}</a-tag>
          </template>
          <template #progress="{ record }">
            <span class="text-[13px] text-[#1D1D1F]"
              >{{ record.fixed_count }}/{{ record.item_count }}</span
            >
          </template>
          <template #deadline="{ record }">
            <span class="text-[13px]" :class="record.overdue ? 'text-[#FF3B30]' : 'text-[#86868b]'">
              {{ record.deadline ? formatDateTime(record.deadline) : '--' }}
            </span>
          </template>
          <template #actions="{ record }">
            <a-button type="text" size="small" @click="goDetail(record)">查看</a-button>
          </template>
          <template #empty>
            <a-empty description="暂无整改任务" />
          </template>
        </a-table>
      </a-spin>

      <div class="flex justify-end mt-4">
        <a-pagination
          :total="total"
          :current="page"
          :page-size="pageSize"
          show-total
          show-page-size
          :page-size-options="[10, 20, 50]"
          @change="onPageChange"
          @page-size-change="onPageSizeChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconCheck } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import type { InspectionTask, Paginated, Store } from '@/types'
import api from '@/utils/api'

const router = useRouter()

const loading = ref(false)
const tasks = ref<InspectionTask[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const storeFilter = ref<string | undefined>(undefined)
const statusFilter = ref<string | undefined>(undefined)
const storeOptions = ref<{ label: string; value: string }[]>([])

const columns = [
  { title: '门店', dataIndex: 'store', slotName: 'store', width: 180 },
  { title: '任务标题', dataIndex: 'title', width: 220 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 110 },
  { title: '负责人', dataIndex: 'responsible_name', width: 110 },
  { title: '整改进度', dataIndex: 'progress', slotName: 'progress', width: 100 },
  { title: '截止时间', dataIndex: 'deadline', slotName: 'deadline', width: 150 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 90 },
]

function statusColor(status: string): string {
  switch (status) {
    case 'pending':
      return 'orange'
    case 'rechecking':
      return 'arcoblue'
    case 'rectifying':
      return 'magenta'
    case 'manual_review':
      return 'purple'
    case 'rectified':
    case 'confirmed':
      return 'green'
    case 'rejected':
      return 'red'
    default:
      return 'gray'
  }
}

function statusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待整改',
    rechecking: '复核中',
    rectifying: '打回整改',
    manual_review: '转人工',
    rectified: '已闭环',
    confirmed: '人工闭环',
    rejected: '未通过',
  }
  return map[status] || status
}

async function fetchStores() {
  try {
    const res = await api.get<Store[]>('/stores/all')
    storeOptions.value = (res.data || []).map((s) => ({ label: s.name, value: s.id }))
  } catch {
    storeOptions.value = []
  }
}

async function fetchTasks() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: pageSize.value }
    if (storeFilter.value) params.store_id = storeFilter.value
    if (statusFilter.value) params.status = statusFilter.value
    const res = await api.get<Paginated<InspectionTask>>('/inspection-tasks/', { params })
    tasks.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载整改任务失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchTasks()
}

function onPageChange(p: number) {
  page.value = p
  fetchTasks()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchTasks()
}

function goDetail(record: InspectionTask) {
  router.push({ name: 'InspectionTaskDetail', params: { id: record.id } })
}

function goInspection() {
  router.push({ name: 'InspectionList' })
}

onMounted(() => {
  fetchStores()
  fetchTasks()
})
</script>
