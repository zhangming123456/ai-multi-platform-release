<template>
  <div class="page-main">
    <PageHeader title="素材管理" subtitle="管理巡店检查项素材与标准图素材" />

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-tabs v-model:active-key="activeTab" type="rounded" @change="onTabChange">
        <!-- 检查项 -->
        <a-tab-pane key="items" title="检查项">
          <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4 mt-2">
            <a-input-search
              v-model="itemKeyword"
              placeholder="搜索标题 / 分类 / 检查标准"
              allow-clear
              class="!w-full sm:!w-72"
              @search="onItemSearch"
              @clear="onItemSearch"
            />
            <div class="flex-1" />
            <a-button
              v-perm="'inspection:material:create:write'"
              type="primary"
              size="small"
              @click="goCreateItem"
            >
              <template #icon><IconPlus :size="13" /></template>
              新建检查项
            </a-button>
          </div>

          <a-spin :loading="itemLoading" tip="加载中..." class="w-full">
            <a-table
              :columns="itemColumns"
              :data="materials"
              :bordered="false"
              :hoverable="true"
              :pagination="false"
            >
              <template #title="{ record }">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-[13px] text-[#1D1D1F]">{{ record.title }}</span>
                  <a-tag v-if="record.category" size="small" color="arcoblue" class="!m-0">{{
                    record.category
                  }}</a-tag>
                </div>
              </template>
              <template #standard="{ record }">
                <span
                  class="text-[13px] text-[#86868b] truncate max-w-[240px] inline-block"
                  :title="record.standard || ''"
                  >{{ record.standard || '--' }}</span
                >
              </template>
              <template #standardImage="{ record }">
                <a-image
                  v-if="record.standard_images?.length"
                  :src="record.standard_images?.[0]"
                  :preview-src="record.standard_images?.[0]"
                  :width="48"
                  :height="48"
                  fit="cover"
                  class="!rounded-lg overflow-hidden cursor-pointer"
                />
                <span v-else class="text-[12px] text-[#C7C7CC]">--</span>
              </template>
              <template #score="{ record }">
                <span class="text-[13px] text-[#1D1D1F]">
                  {{ record.score_type === 'pass_fail' ? '选项评分' : `${record.max_score} 分` }}
                </span>
              </template>
              <template #createdAt="{ record }">
                <span class="text-[13px] text-[#86868b]">{{
                  formatDateTime(record.created_at)
                }}</span>
              </template>
              <template #actions="{ record }">
                <a-space :size="2">
                  <a-tooltip content="编辑">
                    <a-button
                      v-perm="'inspection:material:update:write'"
                      type="text"
                      size="small"
                      @click="goEditItem(record)"
                    >
                      <template #icon><IconEdit /></template>
                    </a-button>
                  </a-tooltip>
                  <a-popconfirm content="确定要删除该检查项素材吗？" @ok="handleDeleteItem(record)">
                    <a-tooltip content="删除">
                      <a-button
                        v-perm="'inspection:material:delete:write'"
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
                <a-empty description="暂无检查项素材" />
              </template>
            </a-table>
          </a-spin>

          <div class="flex justify-end mt-4">
            <a-pagination
              :total="itemTotal"
              :current="itemPage"
              :page-size="itemPageSize"
              show-total
              show-page-size
              :page-size-options="[10, 20, 50]"
              @change="onItemPageChange"
              @page-size-change="onItemPageSizeChange"
            />
          </div>
        </a-tab-pane>

        <!-- 标准图素材 -->
        <a-tab-pane key="images" title="标准图素材">
          <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4 mt-2">
            <a-input-search
              v-model="imgKeyword"
              placeholder="搜索图片名称"
              allow-clear
              class="!w-full sm:!w-72"
              @search="onImgSearch"
              @clear="onImgSearch"
            />
            <div class="flex-1" />
            <a-upload
              v-perm="'material:create:write'"
              :auto-upload="false"
              :show-file-list="false"
              accept="image/*"
              multiple
              @change="handleImgBatchUpload"
            >
              <a-button type="primary" size="small">
                <template #icon><IconUpload :size="13" /></template>
                上传标准图
              </a-button>
            </a-upload>
          </div>

          <a-spin :loading="imgLoading" tip="加载中..." class="w-full">
            <div
              v-if="images.length > 0"
              class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3"
            >
              <div
                v-for="item in images"
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
                    <a-tag size="small" color="arcoblue" class="!m-0">巡店标准图</a-tag>
                    <span class="text-[11px] text-[#86868b]">{{
                      formatDateTime(item.created_at)
                    }}</span>
                  </div>
                </div>
                <div
                  class="absolute top-2 right-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <a-popconfirm content="确定删除该标准图素材？" @ok="handleDeleteImg(item)">
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
            <a-empty v-else description="暂无标准图素材" class="py-16" />
          </a-spin>

          <div class="flex justify-end mt-4">
            <a-pagination
              :total="imgTotal"
              :current="imgPage"
              :page-size="imgPageSize"
              show-total
              show-page-size
              :page-size-options="[12, 24, 48]"
              @change="onImgPageChange"
              @page-size-change="onImgPageSizeChange"
            />
          </div>
        </a-tab-pane>
      </a-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete, IconUpload } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import type { Paginated, InspectionMaterial, Material } from '@/types'
import api from '@/utils/api'

const router = useRouter()
const activeTab = ref<'items' | 'images'>('items')

// 检查项
const itemLoading = ref(false)
const materials = ref<InspectionMaterial[]>([])
const itemTotal = ref(0)
const itemPage = ref(1)
const itemPageSize = ref(10)
const itemKeyword = ref('')

const itemColumns = [
  { title: '素材标题', dataIndex: 'title', slotName: 'title', width: 260 },
  { title: '检查标准', dataIndex: 'standard', slotName: 'standard' },
  { title: '标准图', dataIndex: 'standard_images', slotName: 'standardImage', width: 90 },
  { title: '评分', dataIndex: 'score', slotName: 'score', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt', width: 150 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 110 },
]

async function fetchItems() {
  itemLoading.value = true
  try {
    const params: Record<string, any> = {
      page: itemPage.value,
      page_size: itemPageSize.value,
    }
    if (itemKeyword.value) params.keyword = itemKeyword.value
    const res = await api.get<Paginated<InspectionMaterial>>('/inspection-materials/', { params })
    materials.value = res.data.items || []
    itemTotal.value = res.data.total || 0
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载检查项素材失败')
  } finally {
    itemLoading.value = false
  }
}

function onItemSearch() {
  itemPage.value = 1
  fetchItems()
}

function onItemPageChange(p: number) {
  itemPage.value = p
  fetchItems()
}

function onItemPageSizeChange(size: number) {
  itemPageSize.value = size
  itemPage.value = 1
  fetchItems()
}

function goCreateItem() {
  router.push('/inspection/materials/create')
}

function goEditItem(material: InspectionMaterial) {
  router.push(`/inspection/materials/${material.id}/edit`)
}

async function handleDeleteItem(material: InspectionMaterial) {
  try {
    await api.delete(`/inspection-materials/${material.id}`)
    Message.success('删除成功')
    if (materials.value.length === 1 && itemPage.value > 1) itemPage.value -= 1
    await fetchItems()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '删除失败')
  }
}

// 标准图素材（取自图片素材中分类为「巡店标准图」的图片）
const STANDARD_IMAGE_CATEGORY = '巡店标准图'
const imgLoading = ref(false)
const images = ref<Material[]>([])
const imgTotal = ref(0)
const imgPage = ref(1)
const imgPageSize = ref(12)
const imgKeyword = ref('')

async function fetchImages() {
  imgLoading.value = true
  try {
    const params: Record<string, any> = {
      type: 'image',
      category: STANDARD_IMAGE_CATEGORY,
      page: imgPage.value,
      page_size: imgPageSize.value,
    }
    if (imgKeyword.value) params.keyword = imgKeyword.value
    const res = await api.get<Paginated<Material>>('/materials/', { params })
    images.value = res.data.items || []
    imgTotal.value = res.data.total || 0
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载标准图素材失败')
  } finally {
    imgLoading.value = false
  }
}

function onImgSearch() {
  imgPage.value = 1
  fetchImages()
}

function onImgPageChange(p: number) {
  imgPage.value = p
  fetchImages()
}

function onImgPageSizeChange(size: number) {
  imgPageSize.value = size
  imgPage.value = 1
  fetchImages()
}

async function handleImgBatchUpload(fileList: any[]) {
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
        category: STANDARD_IMAGE_CATEGORY,
      })
      successCount += 1
    } catch {
      // 忽略单个失败
    }
  }
  if (successCount > 0) {
    Message.success(`成功上传 ${successCount} 张标准图`)
    await fetchImages()
  } else {
    Message.error('上传失败')
  }
}

async function handleDeleteImg(item: Material) {
  try {
    await api.delete(`/materials/${item.id}`)
    Message.success('删除成功')
    if (images.value.length === 1 && imgPage.value > 1) imgPage.value -= 1
    await fetchImages()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '删除失败')
  }
}

function onTabChange(key: string) {
  if (key === 'images' && images.value.length === 0) {
    fetchImages()
  } else if (key === 'items' && materials.value.length === 0) {
    fetchItems()
  }
}

onMounted(() => {
  fetchItems()
})
</script>
