<template>
  <div class="page-main">
    <PageHeader
      :title="campaign ? campaign.name : '活动详情'"
      subtitle="查看活动背景信息，AI 创作时将自动注入该活动内容"
    >
      <template #actions>
        <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="goBack">
          <template #icon><IconLeft :size="13" /></template>
          返回列表
        </a-button>
        <a-button v-perm="'campaign:update:write'" type="primary" size="small" @click="goEdit">
          <template #icon><IconEdit :size="13" /></template>
          编辑活动
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-spin :loading="loading" class="w-full">
        <a-card :bordered="false" class="!rounded-xl max-w-[720px]">
          <div class="flex flex-col gap-5">
            <div class="flex items-center gap-2">
              <span class="text-[18px] font-semibold text-[#1D1D1F]">{{ campaign?.name }}</span>
              <a-tag :color="campaign?.status === 'active' ? 'green' : 'gray'" size="small">
                {{ campaign?.status === 'active' ? '进行中' : '已归档' }}
              </a-tag>
            </div>
            <div class="text-[12px] text-[#86868b]">
              创建于 {{ campaign ? formatDateTime(campaign.created_at) : '' }}
              <span v-if="campaign?.updated_at" class="mx-1">·</span>
              更新于 {{ campaign?.updated_at ? formatDateTime(campaign.updated_at) : '' }}
            </div>

            <div
              v-if="campaign?.description"
              class="text-[13px] text-[#4e5969] leading-relaxed whitespace-pre-wrap"
            >
              {{ campaign.description }}
            </div>

            <div v-if="parseMedia(campaign?.media_urls).length">
              <div class="text-[12px] text-[#86909c] mb-2">
                宣发图（{{ parseMedia(campaign?.media_urls).length }}）
              </div>
              <div class="grid grid-cols-3 gap-2">
                <a-image
                  v-for="(m, i) in parseMedia(campaign?.media_urls)"
                  :key="i"
                  :src="m"
                  :preview-props="{ src: m }"
                  width="100%"
                  height="90"
                  fit="cover"
                  class="rounded-[8px] overflow-hidden"
                />
              </div>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div>
                <div class="text-[12px] text-[#86909c]">活动地点</div>
                <div class="text-[13px] text-[#1D1D1F] mt-0.5">
                  {{ campaign?.location || '--' }}
                </div>
              </div>
              <div>
                <div class="text-[12px] text-[#86909c]">面向平台</div>
                <div class="flex items-center gap-2 mt-1">
                  <template v-for="p in parsePlatforms(campaign?.platforms)" :key="p">
                    <a-tooltip :content="platformLabel(p)">
                      <span class="inline-flex"><PlatformIcon :platform="p" /></span>
                    </a-tooltip>
                  </template>
                  <span
                    v-if="!parsePlatforms(campaign?.platforms).length"
                    class="text-[13px] text-[#c9cdd4]"
                    >--</span
                  >
                </div>
              </div>
            </div>
          </div>
        </a-card>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconLeft, IconEdit } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import type { PlatformIconType } from '@/components/shared/PlatformIcon.ts'
import { formatDateTime } from '@/utils/time'
import type { Campaign } from '@/types'
import api from '@/utils/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const campaign = ref<Campaign | null>(null)

const platformChoices: { value: PlatformIconType; label: string }[] = [
  { value: 'wechat_mp', label: '公众号' },
  { value: 'xiaohongshu', label: '小红书' },
  { value: 'douyin', label: '抖音' },
  { value: 'wechat_video', label: '视频号' },
]

function platformLabel(value: string) {
  return platformChoices.find((p) => p.value === value)?.label || value
}

function parseRawList(raw: string): string[] {
  try {
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? (arr as string[]) : []
  } catch {
    return []
  }
}

function parsePlatforms(raw: string | string[] | null | undefined): PlatformIconType[] {
  if (!raw) return []
  const arr = Array.isArray(raw) ? raw : parseRawList(raw)
  return arr.filter((x): x is PlatformIconType => platformChoices.some((p) => p.value === x))
}

function parseMedia(raw: string | string[] | null | undefined): string[] {
  if (!raw) return []
  return Array.isArray(raw) ? raw : parseRawList(raw)
}

async function fetchCampaign() {
  loading.value = true
  try {
    const res = await api.get<Campaign>(`/campaigns/${route.params.id}`)
    campaign.value = res.data
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载活动失败')
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push({ name: 'CampaignManage' })
}

function goEdit() {
  router.push({ name: 'CampaignEdit', params: { id: route.params.id } })
}

onMounted(() => {
  fetchCampaign()
})
</script>
