<template>
  <div class="page-main">
    <PageHeader title="门店管理" subtitle="维护巡店对象门店档案，支持新增、编辑、删除">
      <template #actions>
        <a-button
          v-perm="'stores:create:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="openCreate"
        >
          <template #icon><IconPlus :size="13" /></template>
          新增门店
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
        <a-input-search
          v-model="keyword"
          placeholder="搜索门店名称 / 编号 / 地址"
          allow-clear
          class="!w-full sm:!w-72"
          @search="onSearch"
          @clear="onSearch"
        />
        <a-select
          v-model="statusFilter"
          placeholder="全部状态"
          allow-clear
          class="!w-full sm:!w-40"
          @change="onSearch"
        >
          <a-option value="active">营业中</a-option>
          <a-option value="inactive">已停业</a-option>
        </a-select>
      </div>

      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <a-table
          :columns="columns"
          :data="stores"
          :bordered="false"
          :hoverable="true"
          :pagination="false"
        >
          <template #name="{ record }">
            <div class="flex items-center gap-2">
              <span class="font-medium text-[13px] text-[#1D1D1F]">{{ record.name }}</span>
              <a-tag v-if="record.code" size="small" color="gray" class="!m-0">{{
                record.code
              }}</a-tag>
            </div>
          </template>
          <template #status="{ record }">
            <a-tag :color="record.status === 'active' ? 'green' : 'gray'" size="small">
              {{ record.status === 'active' ? '营业中' : '已停业' }}
            </a-tag>
          </template>
          <template #contact="{ record }">
            <div class="text-[13px] text-[#1D1D1F]">
              <div>{{ record.contact || '--' }}</div>
              <div class="text-[#86868b] text-[12px]">{{ record.phone || '--' }}</div>
            </div>
          </template>
          <template #address="{ record }">
            <span class="text-[13px] text-[#86868b]">{{ record.address || '--' }}</span>
          </template>
          <template #createdAt="{ record }">
            <span class="text-[13px] text-[#86868b]">{{ formatDateTime(record.created_at) }}</span>
          </template>
          <template #actions="{ record }">
            <a-space :size="2">
              <a-tooltip content="编辑">
                <a-button
                  v-perm="'stores:update:write'"
                  type="text"
                  size="small"
                  @click="openEdit(record)"
                >
                  <template #icon><IconEdit /></template>
                </a-button>
              </a-tooltip>
              <a-popconfirm content="确定要删除该门店吗？" @ok="handleDelete(record)">
                <a-tooltip content="删除">
                  <a-button v-perm="'stores:delete:write'" type="text" status="danger" size="small">
                    <template #icon><IconDelete /></template>
                  </a-button>
                </a-tooltip>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <a-empty description="暂无门店" />
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

    <a-modal
      v-model:visible="modalVisible"
      :title="editingId ? '编辑门店' : '新增门店'"
      :width="520"
      @ok="handleSave"
      @cancel="modalVisible = false"
    >
      <a-form :model="form" layout="vertical" class="mt-2">
        <a-form-item label="门店名称" required>
          <a-input v-model="form.name" placeholder="如：北京朝阳旗舰店" maxlength="100" />
        </a-form-item>
        <a-form-item label="门店编号">
          <a-input v-model="form.code" placeholder="如：BJ-001" maxlength="50" />
        </a-form-item>
        <a-form-item label="地址">
          <a-input v-model="form.address" placeholder="门店详细地址" maxlength="300" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="联系人">
              <a-input v-model="form.contact" placeholder="联系人姓名" maxlength="50" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="联系电话">
              <a-input v-model="form.phone" placeholder="联系电话" maxlength="30" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="状态">
          <a-radio-group v-model="form.status" type="button">
            <a-radio value="active">营业中</a-radio>
            <a-radio value="inactive">已停业</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import type { Paginated, Store } from '@/types'
import api from '@/utils/api'

const loading = ref(false)
const stores = ref<Store[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const statusFilter = ref<string | undefined>(undefined)

const modalVisible = ref(false)
const editingId = ref<string | null>(null)

const form = ref({
  name: '',
  code: '',
  address: '',
  contact: '',
  phone: '',
  status: 'active' as 'active' | 'inactive',
})

const columns = [
  { title: '门店', dataIndex: 'name', slotName: 'name', width: 220 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '联系人', dataIndex: 'contact', slotName: 'contact', width: 160 },
  { title: '地址', dataIndex: 'address', slotName: 'address' },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt', width: 150 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 110 },
]

async function fetchStores() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (keyword.value) params.keyword = keyword.value
    if (statusFilter.value) params.status = statusFilter.value
    const res = await api.get<Paginated<Store>>('/stores/', { params })
    stores.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载门店失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchStores()
}

function onPageChange(p: number) {
  page.value = p
  fetchStores()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchStores()
}

function openCreate() {
  editingId.value = null
  form.value = { name: '', code: '', address: '', contact: '', phone: '', status: 'active' }
  modalVisible.value = true
}

function openEdit(store: Store) {
  editingId.value = store.id
  form.value = {
    name: store.name,
    code: store.code || '',
    address: store.address || '',
    contact: store.contact || '',
    phone: store.phone || '',
    status: store.status,
  }
  modalVisible.value = true
}

async function handleSave() {
  if (!form.value.name.trim()) {
    Message.warning('请填写门店名称')
    return
  }
  try {
    if (editingId.value) {
      await api.put(`/stores/${editingId.value}`, { ...form.value })
      Message.success('更新成功')
    } else {
      await api.post('/stores/', { ...form.value })
      Message.success('创建成功')
    }
    modalVisible.value = false
    await fetchStores()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '保存失败')
  }
}

async function handleDelete(store: Store) {
  try {
    await api.delete(`/stores/${store.id}`)
    Message.success('删除成功')
    if (stores.value.length === 1 && page.value > 1) page.value -= 1
    await fetchStores()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '删除失败')
  }
}

onMounted(fetchStores)
</script>
