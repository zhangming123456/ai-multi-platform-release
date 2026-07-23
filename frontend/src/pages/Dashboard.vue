<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Component } from 'vue'
import { useRouter } from 'vue-router'
import {
  IconUserGroup,
  IconSend,
  IconClockCircle,
  IconStar,
  IconRight,
} from '@arco-design/web-vue/es/icon'
import api from '@/utils/api'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatCard from '@/components/shared/StatCard.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'

const router = useRouter()

type Platform = 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
type PublishStatus =
  | 'active'
  | 'inactive'
  | 'error'
  | 'pending'
  | 'publishing'
  | 'published'
  | 'failed'
  | 'draft'
  | 'review'
  | 'approved'
  | 'ready'

interface PlatformStat {
  platform: Platform
  name: string
  accounts: number
  active: number
  articles: number
}

interface RecentPublish {
  id: number
  title: string
  platform: Platform
  account: string
  status: PublishStatus
  time: string
}

interface DashboardStats {
  total_accounts: number
  today_published: number
  pending_tasks: number
  ai_generated_count: number
  platform_stats: PlatformStat[]
  recent_publishes: RecentPublish[]
}

interface StatItem {
  title: string
  value: number
  trend: string
  icon: Component
  color: string
}

const loading = ref(true)
const stats = ref<StatItem[]>([])
const platformStatus = ref<PlatformStat[]>([])
const recentPublishes = ref<RecentPublish[]>([])

const isEmpty = computed(
  () =>
    !loading.value &&
    stats.value.every((s) => s.value === 0) &&
    platformStatus.value.length === 0 &&
    recentPublishes.value.length === 0,
)

function healthPercent(active: number, total: number) {
  if (total === 0) return 0
  return Math.round((active / total) * 100)
}

async function loadDashboard() {
  loading.value = true
  try {
    const { data } = await api.get<DashboardStats>('/dashboard/stats')
    stats.value = [
      { title: '总账号数', value: data.total_accounts, trend: '+0%', icon: IconUserGroup, color: '#007AFF' },
      { title: '今日发布', value: data.today_published, trend: '+0%', icon: IconSend, color: '#34C759' },
      { title: '待处理任务', value: data.pending_tasks, trend: '+0%', icon: IconClockCircle, color: '#FF9500' },
      { title: 'AI 生成次数', value: data.ai_generated_count, trend: '+0%', icon: IconStar, color: '#5856D6' },
    ]
    platformStatus.value = data.platform_stats || []
    recentPublishes.value = data.recent_publishes || []
  } catch (e) {
    stats.value = []
    platformStatus.value = []
    recentPublishes.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>

<template>
  <div>
    <PageHeader title="数据概览" subtitle="今天是发布矩阵的好日子，这是您的实时数据概览">
      <template #actions>
        <a-tag color="green" size="small">
          <template #icon>
            <span
              style="
                display: inline-block;
                width: 6px;
                height: 6px;
                border-radius: 50%;
                background: #34c759;
                margin-right: 4px;
                animation: pulse 2s infinite;
              "
            ></span>
          </template>
          {{ platformStatus.length }} 个平台运行中
        </a-tag>
      </template>
    </PageHeader>

    <a-spin :loading="loading" tip="加载中..." style="width: 100%; min-height: 240px">
      <div v-if="loading" style="min-height: 240px"></div>
      <a-empty v-else-if="isEmpty" description="暂无数据" style="margin: 80px 0" />
      <template v-else>
        <a-row :gutter="16" class="mb-6">
          <a-col v-for="(stat, index) in stats" :key="stat.title" :xs="24" :md="12" :lg="6">
            <StatCard
              :title="stat.title"
              :value="stat.value"
              :trend="stat.trend"
              :icon="stat.icon"
              :color="stat.color"
              :delay="index * 70"
            />
          </a-col>
        </a-row>

        <a-row :gutter="24" class="mb-6">
          <a-col :xs="24" :lg="10">
            <a-card :bordered="false" title="账号健康度" style="padding: 20px">
              <template #extra>
                <a-button type="text" size="small" @click="router.push('/accounts')">
                  全部账号
                  <template #icon>
                    <IconRight />
                  </template>
                </a-button>
              </template>
              <a-empty v-if="platformStatus.length === 0" description="暂无平台数据" />
              <a-space v-else direction="vertical" :size="16" fill>
                <div
                  v-for="item in platformStatus"
                  :key="item.platform"
                  style="display: flex; align-items: center; gap: 12px"
                >
                  <PlatformIcon :platform="item.platform" size="md" />
                  <div style="flex: 1; min-width: 0">
                    <div
                      style="
                        display: flex;
                        align-items: center;
                        justify-content: space-between;
                        margin-bottom: 4px;
                      "
                    >
                      <span style="font-size: 13px; font-weight: 500; color: var(--color-text-1)">{{
                        item.name
                      }}</span>
                      <span style="font-size: 12px; color: var(--color-text-3)"
                        >{{ item.active }}/{{ item.accounts }} 在线</span
                      >
                    </div>
                    <a-progress
                      :percent="healthPercent(item.active, item.accounts) / 100"
                      :color="healthPercent(item.active, item.accounts) === 100 ? '#34C759' : '#FF9500'"
                      :show-text="false"
                      size="small"
                    />
                  </div>
                </div>
              </a-space>
            </a-card>
          </a-col>

          <a-col :xs="24" :lg="14">
            <a-card :bordered="false" title="最近发布" style="padding: 20px">
              <template #extra>
                <a-button type="text" size="small" @click="router.push('/publish')">
                  发布管理
                  <template #icon>
                    <IconRight />
                  </template>
                </a-button>
              </template>
              <a-empty v-if="recentPublishes.length === 0" description="暂无发布记录" />
              <a-list v-else :bordered="false" :split="true">
                <a-list-item v-for="item in recentPublishes" :key="item.id">
                  <a-list-item-meta>
                    <template #avatar>
                      <PlatformIcon :platform="item.platform" size="sm" />
                    </template>
                    <template #title>
                      <span style="font-size: 13px; font-weight: 500; color: var(--color-text-1)">{{
                        item.title
                      }}</span>
                    </template>
                    <template #description>
                      <span style="font-size: 11px; color: var(--color-text-3)">{{
                        item.account
                      }}</span>
                    </template>
                  </a-list-item-meta>
                  <template #actions>
                    <a-space :size="8" align="center">
                      <StatusBadge :status="item.status" />
                      <span
                        style="
                          font-size: 11px;
                          color: var(--color-text-4);
                          width: 40px;
                          text-align: right;
                          flex-shrink: 0;
                        "
                        >{{ item.time }}</span
                      >
                    </a-space>
                  </template>
                </a-list-item>
              </a-list>
            </a-card>
          </a-col>
        </a-row>
      </template>
    </a-spin>
  </div>
</template>

<style scoped>
@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
