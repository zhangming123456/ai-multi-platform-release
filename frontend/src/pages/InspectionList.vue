<template>
  <div class="page-main">
    <PageHeader title="巡店检查" subtitle="发起门店巡店检查，记录检查项评分与问题">
      <template #actions>
        <a-button
          v-perm="'inspection:create:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="goCreate"
        >
          <template #icon><IconPlus :size="13" /></template>
          发起巡店
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
          <a-option value="draft">草稿</a-option>
          <a-option value="pending">待整改</a-option>
          <a-option value="rectifying">整改中</a-option>
          <a-option value="closed">已闭环</a-option>
        </a-select>
      </div>

      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <a-table
          :columns="columns"
          :data="inspections"
          :bordered="false"
          :hoverable="true"
          :pagination="false"
        >
          <template #store="{ record }">
            <div class="flex items-center gap-2">
              <span class="font-medium text-[13px] text-[#1D1D1F]">{{
                record.store_name || '--'
              }}</span>
              <a-tag v-if="record.store_code" size="small" color="gray" class="!m-0">{{
                record.store_code
              }}</a-tag>
            </div>
          </template>
          <template #status="{ record }">
            <a-tag :color="statusColor(record.status)" size="small">
              {{ statusText(record.status) }}
            </a-tag>
          </template>
          <template #result="{ record }">
            <template v-if="record.status !== 'draft'">
              <span
                class="font-semibold text-[14px] tabular"
                :class="record.passed ? 'text-[#34C759]' : 'text-[#FF3B30]'"
              >
                {{ record.total_score }}
              </span>
              <a-tag :color="record.passed ? 'green' : 'red'" size="small" class="!ml-2">
                {{ record.passed ? '合格' : '不合格' }}
              </a-tag>
            </template>
            <span v-else class="text-[#86868b] text-[13px]">--</span>
          </template>
          <template #inspector="{ record }">
            <div class="flex items-center gap-1.5">
              <span class="text-[13px] text-[#1D1D1F]">{{ record.inspector_name || '--' }}</span>
              <a-tag v-if="record.ai_generated" size="small" color="green" class="!m-0">
                <IconRobot :size="11" /> AI
              </a-tag>
            </div>
          </template>
          <template #template="{ record }">
            <a-tag v-if="record.template_name" size="small" color="arcoblue" class="!m-0">{{
              record.template_name
            }}</a-tag>
            <span v-else class="text-[13px] text-[#C7C7CC]">默认检查项</span>
          </template>
          <template #checkedAt="{ record }">
            <span class="text-[13px] text-[#86868b]">{{ formatDateTime(record.checked_at) }}</span>
          </template>
          <template #actions="{ record }">
            <a-space :size="2" @click.stop>
              <a-tooltip content="查看详情">
                <a-button type="text" size="small" @click="goDetail(record)">
                  <template #icon><IconEye /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip v-if="record.status === 'draft'" content="编辑">
                <a-button
                  v-perm="'inspection:update:write'"
                  type="text"
                  size="small"
                  @click="goEdit(record)"
                >
                  <template #icon><IconEdit /></template>
                </a-button>
              </a-tooltip>
              <a-popconfirm content="确定要删除该巡店记录吗？" @ok="handleDelete(record)">
                <a-tooltip content="删除">
                  <a-button
                    v-perm="'inspection:delete:write'"
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
            <a-empty description="暂无巡店记录" />
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
import { IconPlus, IconEdit, IconDelete, IconEye, IconRobot } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import type { Paginated, Store } from '@/types'
import api, { getApiErrorDetail } from '@/utils/api'

interface InspectionListItem {
  id: string
  title: string
  status: 'draft' | 'pending' | 'rectifying' | 'closed'
  store_id: string
  store_name: string
  store_code: string
  template_id: string | null
  template_name: string | null
  inspector_id: string
  inspector_name: string
  total_score: number
  passed: boolean
  ai_generated: boolean
  checked_at: string
  created_at: string
}

const router = useRouter()

const loading = ref(false)
const inspections = ref<InspectionListItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const storeFilter = ref<string | undefined>(undefined)
const statusFilter = ref<string | undefined>(undefined)
const storeOptions = ref<{ label: string; value: string }[]>([])

const columns = [
  { title: '门店', dataIndex: 'store', slotName: 'store', width: 220 },
  { title: '检查表模板', dataIndex: 'template', slotName: 'template', width: 140 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '检查结果', dataIndex: 'result', slotName: 'result', width: 130 },
  { title: '检查人', dataIndex: 'inspector', slotName: 'inspector', width: 110 },
  { title: '检查时间', dataIndex: 'checked_at', slotName: 'checkedAt', width: 160 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 130 },
]

async function fetchStores() {
  try {
    const res = await api.get<Store[]>('/stores/all')
    storeOptions.value = (res.data || []).map((s) => ({ label: s.name, value: s.id }))
  } catch {
    storeOptions.value = []
  }
}

async function fetchInspections() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (storeFilter.value) params.store_id = storeFilter.value
    if (statusFilter.value) params.status = statusFilter.value
    const res = await api.get<Paginated<InspectionListItem>>('/inspections/', { params })
    inspections.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载巡店记录失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchInspections()
}

function onPageChange(p: number) {
  page.value = p
  fetchInspections()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchInspections()
}

function statusText(status: string) {
  switch (status) {
    case 'draft':
      return '草稿'
    case 'pending':
      return '待整改'
    case 'rectifying':
      return '整改中'
    case 'closed':
      return '已闭环'
    default:
      return status
  }
}

function statusColor(status: string) {
  switch (status) {
    case 'draft':
      return 'gray'
    case 'pending':
      return 'orange'
    case 'rectifying':
      return 'arcoblue'
    case 'closed':
      return 'green'
    default:
      return 'gray'
  }
}

function goCreate() {
  router.push({ name: 'InspectionCreate' })
}

function goDetail(record: InspectionListItem) {
  router.push({ name: 'InspectionDetail', params: { id: record.id } })
}

function goEdit(record: InspectionListItem) {
  router.push({ name: 'InspectionEdit', params: { id: record.id } })
}

async function handleDelete(record: InspectionListItem) {
  try {
    await api.delete(`/inspections/${record.id}`)
    Message.success('删除成功')
    if (inspections.value.length === 1 && page.value > 1) page.value -= 1
    await fetchInspections()
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '删除失败')
  }
}

onMounted(() => {
  fetchStores()
  fetchInspections()
})
</script>
