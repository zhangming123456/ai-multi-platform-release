<template>
  <div class="page-main">
    <PageHeader
      title="检查表模板"
      subtitle="配置巡店检查表模板与检查项，支持检查标准、评分方式与必填项设置"
    >
      <template #actions>
        <a-button
          v-perm="'inspection:template:create:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="goCreate"
        >
          <template #icon><IconPlus :size="13" /></template>
          新建模板
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
        <a-input-search
          v-model="keyword"
          placeholder="搜索模板名称 / 描述"
          allow-clear
          class="!w-full sm:!w-72"
          @search="onSearch"
          @clear="onSearch"
        />
      </div>

      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <a-table
          :columns="columns"
          :data="templates"
          :bordered="false"
          :hoverable="true"
          :pagination="false"
        >
          <template #name="{ record }">
            <div class="flex items-center gap-2">
              <span class="font-medium text-[13px] text-[#1D1D1F]">{{ record.name }}</span>
              <a-tag v-if="record.is_active" size="small" color="green" class="!m-0">启用中</a-tag>
              <a-tag v-else size="small" color="gray" class="!m-0">已停用</a-tag>
            </div>
          </template>
          <template #description="{ record }">
            <span class="text-[13px] text-[#86868b]">{{ record.description || '--' }}</span>
          </template>
          <template #itemCount="{ record }">
            <a-tag size="small" color="arcoblue" class="!m-0">{{ record.item_count }} 项</a-tag>
          </template>
          <template #createdAt="{ record }">
            <span class="text-[13px] text-[#86868b]">{{ formatDateTime(record.created_at) }}</span>
          </template>
          <template #actions="{ record }">
            <a-space :size="2">
              <a-tooltip content="编辑">
                <a-button
                  v-perm="'inspection:template:update:write'"
                  type="text"
                  size="small"
                  @click="goEdit(record)"
                >
                  <template #icon><IconEdit /></template>
                </a-button>
              </a-tooltip>
              <a-popconfirm content="确定要删除该模板吗？" @ok="handleDelete(record)">
                <a-tooltip content="删除">
                  <a-button
                    v-perm="'inspection:template:delete:write'"
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
            <a-empty description="暂无模板" />
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
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import type { Paginated, InspectionTemplate } from '@/types'
import api from '@/utils/api'

const router = useRouter()
const loading = ref(false)
const templates = ref<InspectionTemplate[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')

const columns = [
  { title: '模板名称', dataIndex: 'name', slotName: 'name', width: 260 },
  { title: '描述', dataIndex: 'description', slotName: 'description' },
  { title: '检查项', dataIndex: 'item_count', slotName: 'itemCount', width: 90 },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt', width: 150 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 110 },
]

async function fetchTemplates() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (keyword.value) params.keyword = keyword.value
    const res = await api.get<Paginated<InspectionTemplate>>('/inspection-templates/', { params })
    templates.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载模板失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchTemplates()
}

function onPageChange(p: number) {
  page.value = p
  fetchTemplates()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchTemplates()
}

function goCreate() {
  router.push('/inspection/templates/create')
}

function goEdit(template: InspectionTemplate) {
  router.push(`/inspection/templates/${template.id}/edit`)
}

async function handleDelete(template: InspectionTemplate) {
  try {
    await api.delete(`/inspection-templates/${template.id}`)
    Message.success('删除成功')
    if (templates.value.length === 1 && page.value > 1) page.value -= 1
    await fetchTemplates()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '删除失败')
  }
}

onMounted(fetchTemplates)
</script>
