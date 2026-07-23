<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { IconPlus, IconEye, IconCopy } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import api from '@/utils/api'
import PageHeader from '@/components/layout/PageHeader.vue'
import SegmentedControl from '@/components/shared/SegmentedControl.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'

interface Template {
  id: number
  name: string
  platform: string
  thumbnail_url: string | null
  config: unknown
  created_at: string
  updated_at: string
}

const activeTab = ref('all')
const loading = ref(false)

const tabs = [
  { key: 'all', label: '全部' },
  { key: 'wechat_mp', label: '微信公众号' },
  { key: 'xiaohongshu', label: '小红书' },
  { key: 'douyin', label: '抖音' },
  { key: 'wechat_video', label: '视频号' },
]

const platformNames: Record<string, string> = {
  wechat_mp: '微信公众号',
  xiaohongshu: '小红书',
  douyin: '抖音',
  wechat_video: '视频号',
}

const platformGradients: Record<string, string> = {
  wechat_mp: 'linear-gradient(135deg, #2DC100 0%, #07C160 100%)',
  xiaohongshu: 'linear-gradient(135deg, #FF5A6E 0%, #E6002D 100%)',
  douyin: 'linear-gradient(135deg, #3A465C 0%, #161823 100%)',
  wechat_video: 'linear-gradient(135deg, #FA9D3B 0%, #FA5151 100%)',
}

const templates = ref<Template[]>([])

const filteredTemplates = computed(() => {
  if (activeTab.value === 'all') return templates.value
  return templates.value.filter((t) => t.platform === activeTab.value)
})

function getGradient(platform: string): string {
  return (
    platformGradients[platform] || 'linear-gradient(135deg, #007AFF 0%, #0055D4 100%)'
  )
}

function getDescription(tpl: Template): string {
  if (typeof tpl.config === 'string' && tpl.config) return tpl.config
  return ''
}

async function loadTemplates() {
  loading.value = true
  try {
    const res = await api.get('/templates/')
    templates.value = res.data
  } catch {
    Message.error('加载模板失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadTemplates()
})
</script>

<template>
  <div>
    <PageHeader title="模板中心" subtitle="使用模板快速创建优质内容">
      <template #actions>
        <a-button type="primary">
          <template #icon><IconPlus /></template>
          创建模板
        </a-button>
      </template>
    </PageHeader>

    <div class="mb-5">
      <SegmentedControl v-model="activeTab" :options="tabs" />
    </div>

    <a-spin :loading="loading" style="width: 100%; display: block">
      <a-empty v-if="!loading && filteredTemplates.length === 0" description="暂无模板" />
      <a-row v-else :gutter="[16, 20]">
        <a-col
          v-for="(tpl, index) in filteredTemplates"
          :key="tpl.id"
          :xs="24"
          :sm="24"
          :md="12"
          :lg="8"
          :xl="6"
        >
          <a-card
            hoverable
            :bordered="false"
            style="padding: 16px"
            :style="{ animationDelay: `${index * 50}ms`, padding: '16px' }"
          >
            <template #cover>
              <div
                class="h-32 flex items-center justify-center rounded-[12px] mb-3"
                :style="{ background: getGradient(tpl.platform) }"
              >
                <span class="text-white/80 text-4xl font-bold opacity-30">{{
                  tpl.name.charAt(0)
                }}</span>
              </div>
            </template>
            <a-space :size="8" align="center" class="mb-2">
              <PlatformIcon :platform="tpl.platform as 'wechat_mp'" size="sm" />
              <span class="text-[11px] text-secondary">{{ platformNames[tpl.platform] }}</span>
              <a-tag size="small" color="gray">通用</a-tag>
            </a-space>
            <a-typography-text bold class="text-[13px] mb-1 block">{{ tpl.name }}</a-typography-text>
            <a-typography-text
              type="secondary"
              class="text-[12px] mb-3 block"
              :ellipsis="{ rows: 2 }"
              >{{ getDescription(tpl) }}</a-typography-text
            >
            <template #actions>
              <span class="text-[11px] text-tertiary tabular">{{ platformNames[tpl.platform] }}</span>
              <a-button type="text" size="mini" title="预览">
                <template #icon><IconEye /></template>
              </a-button>
              <a-button type="text" size="mini" title="复制">
                <template #icon><IconCopy /></template>
              </a-button>
            </template>
          </a-card>
        </a-col>
      </a-row>
    </a-spin>
  </div>
</template>
