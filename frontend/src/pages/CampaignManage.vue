<template>
  <div class="page-main">
    <PageHeader title="活动管理" subtitle="维护创作内容关联的活动背景，AI 创作时自动注入活动信息">
      <template #actions>
        <a-button
          v-perm="'campaign:create:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="goCreate"
        >
          <template #icon><IconPlus :size="13" /></template>
          新增活动
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4 flex-wrap">
        <a-input-search
          v-model="keyword"
          placeholder="搜索活动名称"
          allow-clear
          class="!w-full sm:!w-64"
          @search="onSearch"
          @clear="onSearch"
        />
        <a-select
          v-model="statusFilter"
          placeholder="全部状态"
          allow-clear
          class="!w-full sm:!w-36"
          @change="onSearch"
        >
          <a-option value="active">进行中</a-option>
          <a-option value="archived">已归档</a-option>
        </a-select>
        <a-select
          v-model="platformFilter"
          placeholder="全部平台"
          allow-clear
          class="!w-full sm:!w-44"
          @change="onSearch"
        >
          <a-option v-for="p in platformChoices" :key="p.value" :value="p.value">
            {{ p.label }}
          </a-option>
        </a-select>
      </div>

      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <a-table
          :columns="columns"
          :data="campaigns"
          :bordered="false"
          :hoverable="true"
          :pagination="false"
        >
          <template #name="{ record }">
            <div class="flex flex-col gap-0.5 min-w-0">
              <span class="font-medium text-[13px] text-[#1D1D1F] truncate">{{ record.name }}</span>
              <span
                v-if="record.description"
                class="text-[12px] text-[#86868b] truncate max-w-[320px]"
              >
                {{ record.description }}
              </span>
            </div>
          </template>
          <template #status="{ record }">
            <a-tag :color="record.status === 'active' ? 'green' : 'gray'" size="small">
              {{ record.status === 'active' ? '进行中' : '已归档' }}
            </a-tag>
          </template>
          <template #platforms="{ record }">
            <div class="flex items-center gap-2">
              <template v-for="p in parsePlatforms(record.platforms)" :key="p">
                <a-tooltip :content="platformLabel(p)">
                  <span class="inline-flex">
                    <PlatformIcon :platform="p" />
                  </span>
                </a-tooltip>
              </template>
              <span v-if="!parsePlatforms(record.platforms).length" class="text-[#c9cdd4]">--</span>
            </div>
          </template>
          <template #media="{ record }">
            <a-avatar-group v-if="parseMedia(record.media_urls).length" :size="28">
              <a-avatar
                v-for="(m, i) in parseMedia(record.media_urls).slice(0, 3)"
                :key="i"
                shape="square"
                :image-url="m"
                :style="{ cursor: 'pointer' }"
                @click="previewImage(m)"
              />
            </a-avatar-group>
            <span v-else class="text-[#c9cdd4]">--</span>
          </template>
          <template #location="{ record }">
            <span class="text-[13px] text-[#86868b]">{{ record.location || '--' }}</span>
          </template>
          <template #createdAt="{ record }">
            <span class="text-[13px] text-[#86868b]">{{ formatDateTime(record.created_at) }}</span>
          </template>
          <template #actions="{ record }">
            <a-space :size="2">
              <a-tooltip content="查看详情">
                <a-button type="text" size="small" @click="goDetail(record)">
                  <template #icon><IconEye /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip content="编辑">
                <a-button
                  v-perm="'campaign:update:write'"
                  type="text"
                  size="small"
                  @click="goEdit(record)"
                >
                  <template #icon><IconEdit /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip :content="record.status === 'active' ? '归档' : '启用'">
                <a-button
                  v-perm="'campaign:update:write'"
                  type="text"
                  size="small"
                  @click="handleToggleStatus(record)"
                >
                  <template #icon>
                    <IconStop v-if="record.status === 'active'" />
                    <IconPlayArrow v-else />
                  </template>
                </a-button>
              </a-tooltip>
              <a-popconfirm
                content="确定要删除该活动吗？删除后归档不可恢复。"
                @ok="handleDelete(record)"
              >
                <a-tooltip content="删除">
                  <a-button
                    v-perm="'campaign:delete:write'"
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
            <a-empty description="暂无活动，点击右上角「新增活动」创建" />
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
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconEye,
  IconStop,
  IconPlayArrow,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import type { PlatformIconType } from '@/components/shared/PlatformIcon.ts'
import { formatDateTime } from '@/utils/time'
import type { Paginated, Campaign } from '@/types'
import api from '@/utils/api'

const router = useRouter()

const loading = ref(false)
const campaigns = ref<Campaign[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const statusFilter = ref<string | undefined>(undefined)
const platformFilter = ref<string | undefined>(undefined)

const platformChoices: { value: PlatformIconType; label: string }[] = [
  { value: 'wechat_mp', label: '公众号' },
  { value: 'xiaohongshu', label: '小红书' },
  { value: 'douyin', label: '抖音' },
  { value: 'wechat_video', label: '视频号' },
  { value: 'wechat_moments', label: '朋友圈' },
  { value: 'weibo', label: '微博' },
]

const columns = [
  { title: '活动', dataIndex: 'name', slotName: 'name', width: 260 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '宣发图', dataIndex: 'media_urls', slotName: 'media', width: 120 },
  { title: '面向平台', dataIndex: 'platforms', slotName: 'platforms', width: 130 },
  { title: '活动地点', dataIndex: 'location', slotName: 'location' },
  { title: '创建时间', dataIndex: 'created_at', slotName: 'createdAt', width: 150 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 160 },
]

function platformLabel(value: string) {
  return platformChoices.find((p) => p.value === value)?.label || value
}

function parsePlatforms(raw: string | string[] | null | undefined): PlatformIconType[] {
  if (!raw) return []
  const arr = Array.isArray(raw) ? raw : parseRawList(raw)
  return arr.filter((x): x is PlatformIconType => platformChoices.some((p) => p.value === x))
}

function parseRawList(raw: string): string[] {
  try {
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? (arr as string[]) : []
  } catch {
    return []
  }
}

function parseMedia(raw: string | string[] | null | undefined): string[] {
  if (!raw) return []
  return Array.isArray(raw) ? raw : parseRawList(raw)
}

async function fetchCampaigns() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (keyword.value) params.keyword = keyword.value
    if (statusFilter.value) params.status = statusFilter.value
    if (platformFilter.value) params.platform = platformFilter.value
    const res = await api.get<Paginated<Campaign>>('/campaigns/', { params })
    campaigns.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载活动失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchCampaigns()
}

function onPageChange(p: number) {
  page.value = p
  fetchCampaigns()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchCampaigns()
}

function goCreate() {
  router.push({ name: 'CampaignCreate' })
}

function goDetail(campaign: Campaign) {
  router.push({ name: 'CampaignDetail', params: { id: campaign.id } })
}

function goEdit(campaign: Campaign) {
  router.push({ name: 'CampaignEdit', params: { id: campaign.id } })
}

async function handleToggleStatus(campaign: Campaign) {
  const next = campaign.status === 'active' ? 'archived' : 'active'
  try {
    await api.put(`/campaigns/${campaign.id}`, { status: next })
    Message.success(next === 'active' ? '活动已启用' : '活动已归档，将不再出现在创作候选活动中')
    await fetchCampaigns()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '操作失败')
  }
}

async function handleDelete(campaign: Campaign) {
  try {
    await api.delete(`/campaigns/${campaign.id}`)
    Message.success('删除成功')
    if (campaigns.value.length === 1 && page.value > 1) page.value -= 1
    await fetchCampaigns()
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '删除失败')
  }
}

function previewImage(url: string) {
  window.open(url, '_blank')
}

onMounted(() => {
  fetchCampaigns()
})
</script>
