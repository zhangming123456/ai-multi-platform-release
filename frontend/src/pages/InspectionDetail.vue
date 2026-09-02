<template>
  <div class="page-main">
    <PageHeader
      title="巡店详情"
      :subtitle="
        inspection ? inspection.store_name + ' · ' + inspection.inspector_name : '巡店检查记录详情'
      "
    >
      <template #actions>
        <a-button class="!mr-2" @click="goBack">返回</a-button>
        <a-button
          v-if="inspection && inspection.status === 'draft'"
          v-perm="'inspection:update:write'"
          type="primary"
          @click="goEdit"
        >
          编辑
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <template v-if="inspection">
          <div class="grid grid-cols-1 xl:grid-cols-3 gap-4">
            <div class="xl:col-span-2 space-y-4">
              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="flex items-center justify-between mb-4">
                  <div class="text-[15px] font-semibold text-[#1D1D1F]">基本信息</div>
                  <div class="flex items-center gap-2">
                    <a-tag :color="inspectionStatusColor(inspection.status)" size="small">
                      {{ inspectionStatusText(inspection.status) }}
                    </a-tag>
                    <a-tag
                      v-if="inspection.status !== 'draft'"
                      :color="inspection.passed ? 'green' : 'red'"
                      size="small"
                    >
                      {{ inspection.passed ? '合格' : '不合格' }}
                    </a-tag>
                  </div>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-3">
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">门店</span>
                    <span class="text-[13px] font-medium text-[#1D1D1F]">
                      {{ inspection.store_name || '--' }}
                      <a-tag v-if="inspection.store_code" size="small" color="gray" class="!ml-1">
                        {{ inspection.store_code }}
                      </a-tag>
                    </span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">检查表模板</span>
                    <span class="text-[13px] text-[#1D1D1F]">
                      {{ inspection.template_name || '默认检查项' }}
                    </span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">地址</span>
                    <span class="text-[13px] text-[#1D1D1F]">{{
                      inspection.store_address || '--'
                    }}</span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">检查人</span>
                    <span class="flex items-center gap-1.5 text-[13px] text-[#1D1D1F]">
                      {{ inspection.inspector_name || '--' }}
                      <a-tag v-if="inspection.ai_generated" size="small" color="green" class="!m-0">
                        <IconRobot :size="11" /> AI 巡店生成
                      </a-tag>
                    </span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">检查时间</span>
                    <span class="text-[13px] text-[#1D1D1F]">{{
                      formatDateTime(inspection.checked_at)
                    }}</span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">主题</span>
                    <span class="text-[13px] text-[#1D1D1F]">{{ inspection.title || '--' }}</span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">创建时间</span>
                    <span class="text-[13px] text-[#1D1D1F]">{{
                      formatDateTime(inspection.created_at)
                    }}</span>
                  </div>
                </div>
              </div>

              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="text-[15px] font-semibold text-[#1D1D1F] mb-4">检查项评分</div>
                <a-table
                  :columns="scoreColumns"
                  :data="inspection.scores"
                  :bordered="false"
                  :pagination="false"
                  size="small"
                >
                  <template #item="{ record }">
                    <span class="text-[13px] font-medium text-[#1D1D1F]">{{
                      record.item_name
                    }}</span>
                    <a-tag v-if="record.ai_generated" size="small" color="green" class="!ml-1 !m-0">
                      <IconRobot :size="11" /> AI
                    </a-tag>
                    <a-tag
                      v-if="record.score_type === 'pass_fail'"
                      size="small"
                      color="purple"
                      class="!ml-2 !m-0"
                    >
                      选项评分
                    </a-tag>
                    <a-tag v-else size="small" color="arcoblue" class="!ml-2 !m-0">分值评分</a-tag>
                  </template>
                  <template #standard="{ record }">
                    <div class="flex items-start gap-2">
                      <a-image
                        v-if="record.standard_images?.length"
                        :src="record.standard_images?.[0]"
                        :preview-src="record.standard_images?.[0]"
                        :width="48"
                        :height="48"
                        fit="cover"
                        class="!rounded-md overflow-hidden shrink-0 !w-12 !h-12"
                      />
                      <span
                        v-if="record.standard"
                        class="text-[12px] text-[#86868b] leading-relaxed"
                      >
                        {{ record.standard }}
                      </span>
                      <span v-else class="text-[12px] text-[#C7C7CC]">--</span>
                    </div>
                  </template>
                  <template #score="{ record }">
                    <template v-if="record.score_type === 'pass_fail'">
                      <span class="font-semibold text-[13px] text-[#1D1D1F]">
                        {{ scoreOptionLabel(record) }}
                      </span>
                    </template>
                    <template v-else>
                      <span
                        class="font-semibold text-[13px]"
                        :class="
                          record.score >= record.max_score * 0.8
                            ? 'text-[#34C759]'
                            : record.score >= record.max_score * 0.6
                              ? 'text-[#FF9500]'
                              : 'text-[#FF3B30]'
                        "
                      >
                        {{ record.score }}
                      </span>
                      <span class="text-[#86868b] text-[12px]"> / {{ record.max_score }}</span>
                    </template>
                  </template>
                  <template #comment="{ record }">
                    <span v-if="record.show_remark === false" class="text-[12px] text-[#C7C7CC]"
                      >--</span
                    >
                    <span v-else class="text-[13px] text-[#1D1D1F]">{{
                      record.comment || '--'
                    }}</span>
                  </template>
                  <template #photos="{ record }">
                    <div
                      v-if="
                        record.show_photo !== false && record.photos && record.photos.length > 0
                      "
                      class="flex items-center gap-1"
                    >
                      <a-image
                        v-for="(photo, index) in record.photos"
                        :key="photo"
                        :src="photo"
                        :preview-src="photo"
                        :width="40"
                        :height="40"
                        fit="cover"
                        class="!rounded-md overflow-hidden cursor-pointer"
                        @click="previewVisible = `${record.item_id}-${index}`"
                      />
                    </div>
                    <span v-else class="text-[12px] text-[#C7C7CC]">--</span>
                  </template>
                </a-table>
              </div>

              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="flex items-center justify-between mb-2">
                  <div class="text-[15px] font-semibold text-[#1D1D1F]">问题与备注</div>
                  <a-tag v-if="inspection.ai_generated" size="small" color="green" class="!m-0">
                    <IconRobot :size="11" /> AI 生成
                  </a-tag>
                </div>
                <div class="text-[13px] text-[#1D1D1F] leading-relaxed whitespace-pre-wrap">
                  {{ inspection.issues || '暂无记录' }}
                </div>
              </div>

              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="flex items-center justify-between mb-2">
                  <div class="text-[15px] font-semibold text-[#1D1D1F]">AI 整改建议</div>
                  <a-tag v-if="inspection.ai_generated" size="small" color="green" class="!m-0">
                    <IconRobot :size="11" /> AI 生成
                  </a-tag>
                </div>
                <div class="text-[13px] text-[#1D1D1F] leading-relaxed whitespace-pre-wrap">
                  {{ inspection.suggestion || '暂无建议' }}
                </div>
              </div>
            </div>

            <div class="space-y-4">
              <div
                v-if="relatedTask"
                class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5 cursor-pointer hover:shadow-sm transition-shadow"
                @click="goTaskDetail"
              >
                <div class="flex items-center justify-between mb-3">
                  <div class="text-[15px] font-semibold text-[#1D1D1F]">整改任务</div>
                  <a-tag :color="taskStatusColor(relatedTask.status)" size="small">{{
                    taskStatusText(relatedTask.status)
                  }}</a-tag>
                </div>
                <div class="flex items-end gap-2 mb-3">
                  <span
                    class="text-[32px] font-bold leading-none tabular"
                    :class="
                      relatedTask.fixed_count >= relatedTask.item_count
                        ? 'text-[#34C759]'
                        : 'text-[#FF9500]'
                    "
                  >
                    {{ relatedTask.fixed_count }}
                  </span>
                  <span class="text-[#86868b] text-[13px] mb-1"
                    >/ {{ relatedTask.item_count }} 项已修复</span
                  >
                </div>
                <a-progress
                  :percent="taskProgressPercent"
                  :color="relatedTask.fixed_count >= relatedTask.item_count ? '#34C759' : '#FF9500'"
                  :stroke-width="8"
                  size="small"
                />
                <div class="mt-3 flex items-center justify-between text-[13px]">
                  <span class="text-[#86868b]"
                    >负责人：{{ relatedTask.responsible_name || '--' }}</span
                  >
                  <span class="text-[#007AFF]">查看详情 →</span>
                </div>
              </div>

              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="text-[15px] font-semibold text-[#1D1D1F] mb-3">检查结果</div>
                <div class="flex items-end gap-2 mb-4">
                  <span
                    class="text-[40px] font-bold leading-none tabular"
                    :class="inspection.passed ? 'text-[#34C759]' : 'text-[#FF3B30]'"
                  >
                    {{ inspection.total_score.toFixed(1) }}
                  </span>
                  <span class="text-[#86868b] text-[13px] mb-1">/ {{ maxTotal }}</span>
                </div>
                <a-progress
                  :percent="progressPercent"
                  :color="inspection.passed ? '#34C759' : '#FF3B30'"
                  :stroke-width="8"
                  size="small"
                />
              </div>

              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="text-[15px] font-semibold text-[#1D1D1F] mb-3">
                  巡店照片（{{ inspection.photos.length }}）
                </div>
                <div v-if="inspection.photos.length > 0" class="grid grid-cols-2 gap-2">
                  <a-image
                    v-for="(photo, index) in inspection.photos"
                    :key="photo"
                    :src="photo"
                    :preview-src="photo"
                    :preview-visible="previewVisible === `top-${index}`"
                    :width="'100%'"
                    :height="'100px'"
                    fit="cover"
                    class="!rounded-lg overflow-hidden cursor-pointer"
                    @click="previewVisible = `top-${index}`"
                  />
                </div>
                <div v-else class="text-[13px] text-[#86868b]">暂无照片</div>
              </div>
            </div>
          </div>
        </template>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconRobot } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import type { Inspection, InspectionScore, InspectionTask } from '@/types'
import api, { getApiErrorDetail } from '@/utils/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const inspection = ref<Inspection | null>(null)
const relatedTask = ref<InspectionTask | null>(null)
const previewVisible = ref<string | null>(null)

const scoringScores = computed(() =>
  (inspection.value?.scores || []).filter((s) => s.score_type !== 'pass_fail'),
)
const maxTotal = computed(() => scoringScores.value.reduce((sum, s) => sum + s.max_score, 0))

const progressPercent = computed(() => {
  if (!inspection.value || maxTotal.value === 0) return 0
  return Math.min(1, inspection.value.total_score / maxTotal.value)
})

const scoreColumns = computed(() => {
  const cols = [
    { title: '检查项', dataIndex: 'item', slotName: 'item', width: 150 },
    { title: '检查标准', dataIndex: 'standard', slotName: 'standard' },
    { title: '得分', dataIndex: 'score', slotName: 'score', width: 110 },
  ]
  const scores = inspection.value?.scores || []
  if (scores.some((s) => s.show_remark !== false)) {
    cols.push({ title: '问题描述', dataIndex: 'comment', slotName: 'comment', width: 160 })
  }
  if (scores.some((s) => s.show_photo !== false)) {
    cols.push({ title: '巡店图片', dataIndex: 'photos', slotName: 'photos', width: 120 })
  }
  return cols
})

function scoreOptionLabel(record: InspectionScore): string {
  if (Array.isArray(record.score_options) && record.score_options.length > 0) {
    const opt = record.score_options.find((o) => Number(o.score) === Number(record.score))
    if (opt) return opt.label
    return record.score_options[0].label
  }
  if (record.score_type === 'pass_fail') {
    return '—'
  }
  return String(record.score)
}

function goBack() {
  router.push({ name: 'InspectionList' })
}

function goEdit() {
  router.push({ name: 'InspectionEdit', params: { id: inspection.value?.id } })
}

function inspectionStatusColor(status: string): string {
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

function inspectionStatusText(status: string): string {
  const map: Record<string, string> = {
    draft: '草稿',
    pending: '待整改',
    rectifying: '整改中',
    closed: '已闭环',
  }
  return map[status] || status
}

function taskStatusColor(status: string): string {
  switch (status) {
    case 'pending':
      return 'orange'
    case 'rechecking':
      return 'arcoblue'
    case 'rectifying':
      return 'magenta'
    case 'manual_review':
      return 'purple'
    case 'rectified':
    case 'confirmed':
      return 'green'
    case 'rejected':
      return 'red'
    default:
      return 'gray'
  }
}

function taskStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待整改',
    rechecking: '复核中',
    rectifying: '打回整改',
    manual_review: '转人工',
    rectified: '已闭环',
    confirmed: '人工闭环',
    rejected: '未通过',
  }
  return map[status] || status
}

const taskProgressPercent = computed(() => {
  if (!relatedTask.value || relatedTask.value.item_count === 0) return 0
  return relatedTask.value.fixed_count / relatedTask.value.item_count
})

function goTaskDetail() {
  if (!relatedTask.value) return
  router.push({ name: 'InspectionTaskDetail', params: { id: relatedTask.value.id } })
}

async function fetchRelatedTask() {
  if (!inspection.value?.id) return
  try {
    const res = await api.get<{ items: InspectionTask[] }>('/inspection-tasks/', {
      params: { inspection_id: inspection.value.id },
    })
    const list = res.data.items || []
    if (list.length > 0) relatedTask.value = list[0]
  } catch {
    relatedTask.value = null
  }
}

async function fetchDetail() {
  loading.value = true
  try {
    const res = await api.get<Inspection>(`/inspections/${route.params.id}`)
    inspection.value = res.data
    await fetchRelatedTask()
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载巡店详情失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchDetail)
</script>
