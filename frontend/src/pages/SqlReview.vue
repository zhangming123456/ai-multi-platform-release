<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconCheckCircle,
  IconCloseCircle,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'
import { formatDateTime } from '@/utils/time'

interface SqlChangeItem {
  id: string
  requester_id: string
  requester_name: string | null
  change_type: string
  sql_text: string
  description: string | null
  status: string
  approvals: number
  required_approvals: number
  approved_by: string[]
  reject_reason: string | null
  execute_message: string | null
  created_at: string
}

const loading = ref(false)
const changes = ref<SqlChangeItem[]>([])
const statusFilter = ref('pending')

const fetchChanges = async () => {
  loading.value = true
  try {
    const res = await api.get('/db-changes/', {
      params: statusFilter.value ? { status: statusFilter.value } : {},
    })
    changes.value = res.data.items
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '获取审核列表失败')
  } finally {
    loading.value = false
  }
}

const handleApprove = (item: SqlChangeItem) => {
  Modal.confirm({
    title: '确认审核通过',
    content: `确定要通过这条 SQL 变更吗？\n\n${item.sql_text}\n\n当前进度：${item.approvals}/${item.required_approvals}，通过后${
      item.approvals + 1 >= item.required_approvals ? '将自动执行' : '还需其他审核员通过'
    }。`,
    hideCancel: false,
    okText: '审核通过',
    onOk: async () => {
      try {
        const res = await api.post(`/db-changes/${item.id}/approve`)
        const newStatus = res.data.status
        if (newStatus === 'executed') {
          Message.success('审核通过，SQL 已自动执行')
        } else if (newStatus === 'execute_failed') {
          Message.warning('审核通过，但 SQL 执行失败')
        } else {
          Message.success(`审核通过（${res.data.approvals}/${res.data.required_approvals}）`)
        }
        await fetchChanges()
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '审核失败')
      }
    },
  })
}

const handleReject = (item: SqlChangeItem) => {
  let reason = ''
  Modal.warning({
    title: '驳回 SQL 变更',
    content: '确定要驳回这条 SQL 变更请求吗？',
    hideCancel: false,
    okText: '确认驳回',
    onOk: async () => {
      try {
        await api.post(`/db-changes/${item.id}/reject`, { reason })
        Message.success('已驳回')
        await fetchChanges()
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '驳回失败')
      }
    },
  })
}

const getTypeLabel = (type: string) => {
  switch (type) {
    case 'delete':
      return '删除'
    case 'update':
      return '修改'
    case 'row_delete':
      return '行删除'
    default:
      return type
  }
}

const getTypeColor = (type: string) => {
  switch (type) {
    case 'delete':
    case 'row_delete':
      return 'red'
    case 'update':
      return 'orange'
    default:
      return 'gray'
  }
}

const getStatusLabel = (status: string) => {
  switch (status) {
    case 'pending':
      return '待审核'
    case 'approved':
      return '部分通过'
    case 'rejected':
      return '已驳回'
    case 'executed':
      return '已执行'
    case 'execute_failed':
      return '执行失败'
    default:
      return status
  }
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'pending':
      return 'orange'
    case 'approved':
      return 'arcoblue'
    case 'rejected':
      return 'red'
    case 'executed':
      return 'green'
    case 'execute_failed':
      return 'red'
    default:
      return 'gray'
  }
}

onMounted(() => {
  fetchChanges()
})
</script>

<template>
  <div class="sql-review-page">
    <PageHeader title="SQL 变更审核" subtitle="审核数据库删除/修改操作，至少 2 人通过后自动执行">
      <template #actions>
        <a-select v-model="statusFilter" style="width: 140px" @change="fetchChanges">
          <a-option value="pending">待审核</a-option>
          <a-option value="executed">已执行</a-option>
          <a-option value="rejected">已驳回</a-option>
          <a-option value="execute_failed">执行失败</a-option>
        </a-select>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <a-spin :loading="loading">
        <a-table :data="changes" :pagination="false" :bordered="false" row-key="id">
          <template #columns>
            <a-table-column title="类型" :width="90">
              <template #cell="{ record }">
                <a-tag size="small" :color="getTypeColor(record.change_type)">
                  {{ getTypeLabel(record.change_type) }}
                </a-tag>
              </template>
            </a-table-column>

            <a-table-column title="SQL 语句" :width="360">
              <template #cell="{ record }">
                <code class="sql-text">{{ record.sql_text }}</code>
                <div v-if="record.description" class="sql-desc">{{ record.description }}</div>
              </template>
            </a-table-column>

            <a-table-column title="提交人" data-index="requester_name" :width="100" />

            <a-table-column title="审核进度" :width="110">
              <template #cell="{ record }">
                <a-tag size="small" color="arcoblue">
                  {{ record.approvals }}/{{ record.required_approvals }}
                </a-tag>
              </template>
            </a-table-column>

            <a-table-column title="状态" :width="100">
              <template #cell="{ record }">
                <a-tag size="small" :color="getStatusColor(record.status)">
                  {{ getStatusLabel(record.status) }}
                </a-tag>
              </template>
            </a-table-column>

            <a-table-column title="提交时间" :width="150">
              <template #cell="{ record }">
                {{ formatDateTime(record.created_at) }}
              </template>
            </a-table-column>

            <a-table-column title="操作" :width="150" fixed="right">
              <template #cell="{ record }">
                <a-space v-if="record.status === 'pending'">
                  <a-button
                    type="primary"
                    size="mini"
                    @click="handleApprove(record)"
                  >
                    <template #icon><IconCheckCircle /></template>
                    通过
                  </a-button>
                  <a-button
                    status="danger"
                    size="mini"
                    @click="handleReject(record)"
                  >
                    <template #icon><IconCloseCircle /></template>
                    驳回
                  </a-button>
                </a-space>
                <span v-else class="text-[12px] text-[#aeaeb2]">
                  {{ record.execute_message || record.reject_reason || '—' }}
                </span>
              </template>
            </a-table-column>
          </template>

          <template #empty>
            <a-empty description="暂无待审核的 SQL 变更" />
          </template>
        </a-table>
      </a-spin>
    </a-card>
  </div>
</template>

<style scoped lang="scss">
.sql-review-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sql-text {
  display: block;
  font-size: 12px;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, monospace;
  color: var(--apple-text-primary);
  background: var(--apple-bg);
  padding: 6px 10px;
  border-radius: 6px;
  word-break: break-all;
  white-space: pre-wrap;
}

.sql-desc {
  margin-top: 4px;
  font-size: 11px;
  color: var(--apple-text-tertiary);
}
</style>
