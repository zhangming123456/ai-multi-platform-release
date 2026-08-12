<template>
  <div class="page-main">
    <PageHeader
      title="图片素材"
      subtitle="统一管理图片素材，支持标签分类，可被巡店标准图等模块引用"
    >
      <template #actions>
        <a-upload
          v-perm="'material:create:write'"
          :auto-upload="false"
          :show-file-list="false"
          accept="image/*"
          multiple
          @change="handleBatchUpload"
        >
          <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto">
            <template #icon><IconUpload :size="13" /></template>
            上传图片
          </a-button>
        </a-upload>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <!-- 分类筛选 + 搜索 -->
      <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
        <a-input-search
          v-model="keyword"
          placeholder="搜索名称 / 分类"
          allow-clear
          class="!w-full sm:!w-72"
          @search="onSearch"
          @clear="onSearch"
        />
        <a-select
          v-model="categoryFilter"
          placeholder="全部分类"
          allow-clear
          class="!w-full sm:!w-44"
          @change="onSearch"
        >
          <a-option v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</a-option>
        </a-select>
      </div>

      <!-- 分类标签快捷筛选 -->
      <div v-if="categories.length > 0" class="flex flex-wrap items-center gap-1.5 mb-4">
        <a-tag
          checkable
          :checked="!categoryFilter"
          color="arcoblue"
          size="small"
          @check="onTagCheck('')"
          >全部</a-tag
        >
        <a-tag
          v-for="cat in categories"
          :key="cat"
          checkable
          :checked="categoryFilter === cat"
          color="arcoblue"
          size="small"
          @check="onTagCheck(cat)"
          >{{ cat }}</a-tag
        >
      </div>

      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <div
          v-if="materials.length > 0"
          class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3"
        >
          <div
            v-for="item in materials"
            :key="item.id"
            class="group relative rounded-xl border border-[#E5E5EA] bg-white overflow-hidden hover:border-[#165DFF] hover:shadow-md transition-all"
          >
            <div
              class="aspect-square bg-[#F5F5F7] flex items-center justify-center overflow-hidden"
            >
              <a-image
                :src="item.url"
                :preview-src="item.url"
                fit="cover"
                class="!w-full !h-full"
                :preview-props="{ actionsLayout: [] }"
              />
            </div>
            <div class="p-2.5">
              <div class="text-[13px] font-medium text-[#1D1D1F] truncate" :title="item.name">
                {{ item.name }}
              </div>
              <div class="flex items-center justify-between mt-1">
                <a-tag v-if="item.category" size="small" color="arcoblue" class="!m-0">{{
                  item.category
                }}</a-tag>
                <span v-else class="text-[11px] text-[#C7C7CC]">未分类</span>
                <span class="text-[11px] text-[#86868b]">{{
                  formatDateTime(item.created_at)
                }}</span>
              </div>
            </div>
            <!-- 悬浮操作 -->
            <div
              class="absolute top-2 right-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <a-tooltip content="编辑">
                <a-button
                  v-perm="'material:update:write'"
                  type="text"
                  size="mini"
                  class="!bg-white/90 !backdrop-blur"
                  @click.stop="openEdit(item)"
                >
                  <template #icon><IconEdit :size="13" /></template>
                </a-button>
              </a-tooltip>
              <a-popconfirm content="确定删除该素材？" @ok="handleDelete(item)">
                <a-tooltip content="删除">
                  <a-button
                    v-perm="'material:delete:write'"
                    type="text"
                    size="mini"
                    status="danger"
                    class="!bg-white/90 !backdrop-blur"
                    @click.stop
                  >
                    <template #icon><IconDelete :size="13" /></template>
                  </a-button>
                </a-tooltip>
              </a-popconfirm>
            </div>
          </div>
        </div>
        <a-empty v-else description="暂无图片素材" class="py-16" />
      </a-spin>

      <div class="flex justify-end mt-4">
        <a-pagination
          :total="total"
          :current="page"
          :page-size="pageSize"
          show-total
          show-page-size
          :page-size-options="[12, 24, 48]"
          @change="onPageChange"
          @page-size-change="onPageSizeChange"
        />
      </div>
    </div>

    <!-- 编辑弹窗 -->
    <a-modal
      v-model:visible="editVisible"
      :title="editing.id ? '编辑素材' : '素材信息'"
      :width="420"
      :mask-closable="false"
      @ok="handleSave"
      @cancel="editVisible = false"
    >
      <a-form :model="editing" layout="vertical">
        <a-form-item label="预览">
          <div
            class="w-full h-32 rounded-lg bg-[#F5F5F7] flex items-center justify-center overflow-hidden"
          >
            <a-image v-if="editing.url" :src="editing.url" fit="contain" class="!w-full !h-full" />
            <span v-else class="text-[12px] text-[#C7C7CC]">无图片</span>
          </div>
        </a-form-item>
        <a-form-item label="名称" required>
          <a-input v-model="editing.name" placeholder="图片名称" :maxlength="100" show-word-limit />
        </a-form-item>
        <a-form-item label="分类标签">
          <a-select
            v-model="editing.category"
            placeholder="选择或输入分类标签"
            allow-create
            allow-search
            :options="categoryOptions"
          />
          <div class="text-[11px] text-[#86868b] mt-1">如「巡店标准图」「商品图」「环境图」等</div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconUpload, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import type { Paginated, Material } from '@/types'
import api from '@/utils/api'

const loading = ref(false)
const materials = ref<Material[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const keyword = ref('')
const categoryFilter = ref('')
const categories = ref<string[]>([])

const editVisible = ref(false)
const editing = ref<{ id: string; name: string; url: string; category: string }>({
  id: '',
  name: '',
  url: '',
  category: '',
})

const categoryOptions = computed(() => {
  const presets = ['巡店标准图', '商品图', '环境图', '门店形象', '其他']
  const all = new Set<string>(presets)
  categories.value.forEach((c) => all.add(c))
  return Array.from(all).map((c) => ({ label: c, value: c }))
})

async function fetchMaterials() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      type: 'image',
      page: page.value,
      page_size: pageSize.value,
    }
    if (keyword.value) params.keyword = keyword.value
    if (categoryFilter.value) params.category = categoryFilter.value
    const res = await api.get<Paginated<Material>>('/materials/', { params })
    materials.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载素材失败')
  } finally {
    loading.value = false
  }
}

async function fetchCategories() {
  try {
    const res = await api.get<string[]>('/materials/categories', { params: { type: 'image' } })
    categories.value = res.data || []
  } catch {
    categories.value = []
  }
}

function onSearch() {
  page.value = 1
  fetchMaterials()
}

function onTagCheck(cat: string) {
  categoryFilter.value = cat
  onSearch()
}

function onPageChange(p: number) {
  page.value = p
  fetchMaterials()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchMaterials()
}

async function handleBatchUpload(fileList: any[]) {
  const files = (fileList || []).filter((f) => f.file).map((f) => f.file as File)
  if (files.length === 0) return
  let successCount = 0
  for (const file of files) {
    try {
      const formData = new FormData()
      formData.append('file', file, file.name)
      const upRes = await api.post('/uploads', formData)
      const url = upRes.data?.url || ''
      if (!url) continue
      await api.post('/materials/', {
        name: file.name.replace(/\.[^.]+$/, ''),
        url,
        type: 'image',
        category: categoryFilter.value || '',
      })
      successCount += 1
    } catch {
      // 忽略单个失败
    }
  }
  if (successCount > 0) {
    Message.success(`成功上传 ${successCount} 张图片`)
    await fetchMaterials()
    await fetchCategories()
  } else {
    Message.error('上传失败')
  }
}

function openEdit(item: Material) {
  editing.value = {
    id: item.id,
    name: item.name,
    url: item.url,
    category: item.category || '',
  }
  editVisible.value = true
}

async function handleSave() {
  if (!editing.value.name.trim()) {
    Message.warning('请填写名称')
    return
  }
  try {
    await api.put(`/materials/${editing.value.id}`, {
      name: editing.value.name.trim(),
      url: editing.value.url,
      type: 'image',
      category: editing.value.category || '',
    })
    Message.success('已更新')
    editVisible.value = false
    await fetchMaterials()
    await fetchCategories()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '保存失败')
  }
}

async function handleDelete(item: Material) {
  try {
    await api.delete(`/materials/${item.id}`)
    Message.success('删除成功')
    if (materials.value.length === 1 && page.value > 1) page.value -= 1
    await fetchMaterials()
    await fetchCategories()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '删除失败')
  }
}

onMounted(() => {
  fetchMaterials()
  fetchCategories()
})
</script>
