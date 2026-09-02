<template>
  <div class="page-main">
    <PageHeader
      title="整改任务详情"
      :subtitle="task ? `${task.store_name} · ${task.title}` : '巡店整改任务详情'"
    >
      <template #actions>
        <a-button @click="goBack">返回</a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <template v-if="task">
          <div class="grid grid-cols-1 xl:grid-cols-3 gap-4">
            <div class="xl:col-span-2 space-y-4">
              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="flex items-center justify-between mb-4">
                  <div class="text-[15px] font-semibold text-[#1D1D1F]">基本信息</div>
                  <a-tag :color="statusColor(task.status)" size="small">{{
                    statusText(task.status)
                  }}</a-tag>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-3">
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">门店</span>
                    <span class="text-[13px] font-medium text-[#1D1D1F]">{{
                      task.store_name || '--'
                    }}</span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">巡店人</span>
                    <span class="text-[13px] text-[#1D1D1F]">{{
                      task.inspector_name || '--'
                    }}</span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">负责人</span>
                    <span class="text-[13px] text-[#1D1D1F]">{{
                      task.responsible_name || '--'
                    }}</span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">联系方式</span>
                    <span class="text-[13px] text-[#1D1D1F]">{{ task.phone || '--' }}</span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">截止时间</span>
                    <span
                      class="text-[13px]"
                      :class="task.overdue ? 'text-[#FF3B30]' : 'text-[#1D1D1F]'"
                    >
                      {{ task.deadline ? formatDateTime(task.deadline) : '--' }}
                    </span>
                  </div>
                  <div class="flex gap-3">
                    <span class="text-[13px] text-[#86868b] w-16 shrink-0">整改进度</span>
                    <span class="text-[13px] text-[#1D1D1F]"
                      >{{ task.fixed_count }}/{{ task.item_count }}</span
                    >
                  </div>
                </div>
              </div>

              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="flex items-center justify-between mb-4">
                  <div class="text-[15px] font-semibold text-[#1D1D1F]">问题项清单</div>
                  <div
                    v-if="canSubmitRectify || canRecheck || canManualConfirm"
                    class="flex items-center gap-2"
                  >
                    <a-button
                      v-if="canSubmitRectify"
                      type="primary"
                      size="small"
                      :loading="submitting"
                      @click="openSubmitModal"
                    >
                      提交整改
                    </a-button>
                    <a-button
                      v-if="canRecheck"
                      type="primary"
                      size="small"
                      :loading="rechecking"
                      @click="handleRecheck"
                    >
                      AI 复核
                    </a-button>
                    <a-button
                      v-if="canManualConfirm"
                      size="small"
                      :loading="confirming"
                      @click="openConfirmModal"
                    >
                      人工确认
                    </a-button>
                  </div>
                </div>

                <div v-for="item in items" :key="item.id" class="mb-4 last:mb-0">
                  <div
                    class="rounded-xl border p-4 transition-colors"
                    :class="[
                      selectedItemIds.includes(item.id)
                        ? 'border-[#007AFF] bg-[#007AFF]/5'
                        : 'border-[#E5E5EA] bg-white/50',
                    ]"
                    @click="toggleSelect(item)"
                  >
                    <div class="flex items-start justify-between gap-3">
                      <div class="flex-1">
                        <div class="flex items-center gap-2 mb-1">
                          <span class="font-medium text-[14px] text-[#1D1D1F]">{{
                            item.item_name
                          }}</span>
                          <a-tag :color="itemStatusColor(item.status)" size="small">{{
                            itemStatusText(item.status)
                          }}</a-tag>
                          <a-tag
                            v-if="item.score_type === 'pass_fail'"
                            size="small"
                            color="purple"
                            class="!m-0"
                            >选项评分</a-tag
                          >
                          <a-tag v-else size="small" color="arcoblue" class="!m-0">分值评分</a-tag>
                        </div>
                        <div v-if="item.standard" class="text-[12px] text-[#86868b] mb-2">
                          {{ item.standard }}
                        </div>
                        <div v-if="item.ai_suggestion" class="text-[12px] text-[#007AFF] mb-2">
                          整改建议：{{ item.ai_suggestion }}
                        </div>
                        <div class="text-[13px] text-[#1D1D1F]">
                          原问题：{{ item.comment || '--' }}
                          <span class="text-[#86868b] ml-2">({{ scoreLabel(item) }})</span>
                        </div>
                      </div>
                      <div class="text-right shrink-0">
                        <div class="text-[13px] text-[#86868b]">
                          复核 {{ item.recheck_count }}/2 次
                        </div>
                      </div>
                    </div>

                    <div class="mt-3">
                      <div class="text-[12px] text-[#86868b] mb-1">原照片</div>
                      <div v-if="item.original_photos.length > 0" class="flex items-center gap-2">
                        <a-image
                          v-for="(photo, idx) in item.original_photos"
                          :key="idx"
                          :src="photo"
                          :preview-src="photo"
                          :width="48"
                          :height="48"
                          fit="cover"
                          class="!rounded-md overflow-hidden cursor-pointer"
                        />
                      </div>
                      <span v-else class="text-[12px] text-[#C7C7CC]">--</span>
                    </div>

                    <div
                      v-if="item.rectify_photos.length > 0 || item.rectify_comment"
                      class="mt-3 rounded-lg bg-[#F5F5F7] p-3"
                    >
                      <div class="text-[12px] text-[#86868b] mb-1">整改照片 / 说明</div>
                      <div
                        v-if="item.rectify_photos.length > 0"
                        class="flex items-center gap-2 mb-2"
                      >
                        <a-image
                          v-for="(photo, idx) in item.rectify_photos"
                          :key="idx"
                          :src="photo"
                          :preview-src="photo"
                          :width="48"
                          :height="48"
                          fit="cover"
                          class="!rounded-md overflow-hidden cursor-pointer"
                        />
                      </div>
                      <div v-if="item.rectify_comment" class="text-[13px] text-[#1D1D1F]">
                        {{ item.rectify_comment }}
                      </div>
                    </div>

                    <div
                      v-if="item.ai_result"
                      class="mt-3 rounded-lg p-3"
                      :class="item.ai_result.fixed ? 'bg-[#34C759]/10' : 'bg-[#FF3B30]/10'"
                    >
                      <div
                        class="text-[12px] font-medium mb-1"
                        :class="item.ai_result.fixed ? 'text-[#34C759]' : 'text-[#FF3B30]'"
                      >
                        {{ item.ai_result.fixed ? 'AI 判定已修复' : 'AI 判定未修复' }}
                        <span v-if="item.score_type !== 'pass_fail'" class="ml-2"
                          >修复后评分：{{ item.ai_result.score }}</span
                        >
                      </div>
                      <div class="text-[13px] text-[#1D1D1F]">{{ item.ai_result.reason }}</div>
                    </div>
                  </div>
                </div>
              </div>

              <div
                v-if="logs.length > 0"
                class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5"
              >
                <div class="text-[15px] font-semibold text-[#1D1D1F] mb-4">操作日志</div>
                <a-timeline>
                  <a-timeline-item
                    v-for="log in logs"
                    :key="log.id"
                    :label="formatDateTime(log.created_at)"
                  >
                    <div class="text-[13px] text-[#1D1D1F]">{{ log.content }}</div>
                    <div v-if="log.operator_name" class="text-[12px] text-[#86868b]">
                      操作人：{{ log.operator_name }}
                    </div>
                  </a-timeline-item>
                </a-timeline>
              </div>
            </div>

            <div class="space-y-4">
              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="text-[15px] font-semibold text-[#1D1D1F] mb-3">AI 复核模型</div>
                <ModelSelect @change="onModelChange" />
                <div class="text-[12px] text-[#86868b] mt-2">AI 复核需选择支持视觉理解的模型。</div>
              </div>

              <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
                <div class="text-[15px] font-semibold text-[#1D1D1F] mb-3">状态说明</div>
                <div class="space-y-2 text-[12px] text-[#86868b]">
                  <div>
                    <span class="inline-block w-2 h-2 rounded-full bg-[#FF9500] mr-2"></span
                    >待整改：等待负责人提交整改照片
                  </div>
                  <div>
                    <span class="inline-block w-2 h-2 rounded-full bg-[#007AFF] mr-2"></span
                    >复核中：已提交，等待 AI 复核
                  </div>
                  <div>
                    <span class="inline-block w-2 h-2 rounded-full bg-[#FF2D55] mr-2"></span
                    >打回整改：AI 判定未修复，需重新整改
                  </div>
                  <div>
                    <span class="inline-block w-2 h-2 rounded-full bg-[#AF52DE] mr-2"></span
                    >转人工：复核达 2 次上限，需管理员或巡店人判定
                  </div>
                  <div>
                    <span class="inline-block w-2 h-2 rounded-full bg-[#34C759] mr-2"></span
                    >已闭环：AI 复核全部通过或人工确认到位
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </a-spin>
    </div>

    <a-modal
      v-model:visible="submitVisible"
      title="提交整改"
      :width="600"
      @ok="handleSubmitRectify"
      @cancel="submitVisible = false"
    >
      <div class="space-y-4 mt-2">
        <div class="text-[13px] text-[#86868b]">已选择 {{ selectedItems.length }} 个问题项</div>
        <div
          v-for="item in selectedItems"
          :key="item.id"
          class="rounded-lg border border-[#E5E5EA] p-3"
        >
          <div class="font-medium text-[14px] text-[#1D1D1F] mb-2">{{ item.item_name }}</div>
          <AttachmentInputArea
            :ref="(el) => setRectifyAreaRef(item, el)"
            v-model="rectifyComments[item.id]"
            v-model:file-list="rectifyPhotos[item.id]"
            :upload="uploadImageFile"
            :max-count="MAX_RECTIFY_PHOTOS"
            :min-rows="2"
            :max-rows="4"
            placeholder="整改说明（选填）...&#10;支持拖拽 / 粘贴图片，或粘贴图片链接自动识别"
          />
        </div>
      </div>
    </a-modal>

    <a-modal
      v-model:visible="confirmVisible"
      title="人工确认"
      :width="600"
      @ok="handleManualConfirm"
      @cancel="confirmVisible = false"
    >
      <div class="space-y-4 mt-2">
        <div class="text-[13px] text-[#86868b]">
          已选择 {{ selectedItems.length }} 个转人工问题项
        </div>
        <div
          v-for="item in selectedItems"
          :key="item.id"
          class="rounded-lg border border-[#E5E5EA] p-3"
        >
          <div class="font-medium text-[14px] text-[#1D1D1F] mb-2">{{ item.item_name }}</div>
          <a-radio-group v-model="confirmResults[item.id]" type="button">
            <a-radio :value="true">整改到位</a-radio>
            <a-radio :value="false">整改未到位</a-radio>
          </a-radio-group>
          <a-textarea
            v-model="confirmComments[item.id]"
            placeholder="确认备注（选填）"
            :auto-size="{ minRows: 2, maxRows: 4 }"
            class="mt-2 !rounded-lg"
          />
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import ModelSelect, { type ModelSelectValue } from '@/components/shared/ModelSelect.vue'
import { uploadImageFile } from '@/composables/useFileUpload'
import { formatDateTime } from '@/utils/time'
import type { InspectionTask, InspectionTaskItem, InspectionTaskLog } from '@/types'
import api, { getApiErrorDetail } from '@/utils/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const task = ref<InspectionTask | null>(null)
const items = ref<InspectionTaskItem[]>([])
const logs = ref<InspectionTaskLog[]>([])
const selectedItemIds = ref<string[]>([])
const submitting = ref(false)
const rechecking = ref(false)
const confirming = ref(false)
const submitVisible = ref(false)
const confirmVisible = ref(false)
const MAX_RECTIFY_PHOTOS = 5
const rectifyPhotos = ref<Record<string, string[]>>({})
const rectifyComments = ref<Record<string, string>>({})

const rectifyAreaRefs = new Map<string, { flushPending: () => Promise<boolean> }>()
function setRectifyAreaRef(item: { id: string }, el: unknown) {
  if (el) {
    rectifyAreaRefs.set(item.id, el as { flushPending: () => Promise<boolean> })
  } else {
    rectifyAreaRefs.delete(item.id)
  }
}

const confirmResults = ref<Record<string, boolean>>({})
const confirmComments = ref<Record<string, string>>({})

const selectedPlanId = ref('')
const selectedModelId = ref('')
const selectedHasVision = ref(false)

const pendingItems = computed(() =>
  items.value.filter((it) => it.status === 'pending' || it.status === 'not_fixed'),
)
const submittedItems = computed(() => items.value.filter((it) => it.status === 'submitted'))
const manualItems = computed(() => items.value.filter((it) => it.status === 'manual'))

const canSubmitRectify = computed(
  () =>
    task.value &&
    (task.value.status === 'pending' || task.value.status === 'rectifying') &&
    pendingItems.value.length > 0,
)
const canRecheck = computed(
  () => task.value && task.value.status === 'rechecking' && submittedItems.value.length > 0,
)
const canManualConfirm = computed(
  () => task.value && task.value.status === 'manual_review' && manualItems.value.length > 0,
)

const selectedItems = computed(() =>
  items.value.filter((it) => selectedItemIds.value.includes(it.id)),
)

function statusColor(status: string): string {
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

function statusText(status: string): string {
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

function itemStatusColor(status: string): string {
  switch (status) {
    case 'pending':
      return 'orange'
    case 'submitted':
      return 'arcoblue'
    case 'fixed':
    case 'confirmed':
      return 'green'
    case 'not_fixed':
      return 'magenta'
    case 'manual':
      return 'purple'
    case 'rejected':
      return 'red'
    default:
      return 'gray'
  }
}

function itemStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待整改',
    submitted: '已提交',
    fixed: '已修复',
    not_fixed: '未修复',
    manual: '转人工',
    confirmed: '已确认',
    rejected: '未通过',
  }
  return map[status] || status
}

function scoreLabel(item: InspectionTaskItem): string {
  if (item.score_type === 'pass_fail') {
    return '选项评分'
  }
  return `${item.score} / ${item.max_score}`
}

function toggleSelect(item: InspectionTaskItem) {
  if (!isSelectable(item.status)) return
  const idx = selectedItemIds.value.indexOf(item.id)
  if (idx >= 0) {
    selectedItemIds.value.splice(idx, 1)
  } else {
    selectedItemIds.value.push(item.id)
  }
}

function isSelectable(status: string): boolean {
  return ['pending', 'not_fixed', 'submitted', 'manual'].includes(status)
}

function onModelChange(val: ModelSelectValue) {
  selectedPlanId.value = val.planId
  selectedModelId.value = val.modelId
  selectedHasVision.value = val.hasVision
}

function openSubmitModal() {
  if (selectedItems.value.length === 0) {
    Message.warning('请先选择要提交整改的问题项')
    return
  }
  rectifyPhotos.value = {}
  rectifyComments.value = {}
  for (const item of selectedItems.value) {
    rectifyPhotos.value[item.id] = []
    rectifyComments.value[item.id] = ''
  }
  submitVisible.value = true
}

async function handleSubmitRectify() {
  for (const areaRef of rectifyAreaRefs.values()) {
    if (!(await areaRef.flushPending())) return
  }
  const payloadItems = []
  for (const item of selectedItems.value) {
    const photos = rectifyPhotos.value[item.id] || []
    if (photos.length === 0) {
      Message.warning(`「${item.item_name}」请上传整改照片`)
      return
    }
    payloadItems.push({
      item_id: item.id,
      photos,
      comment: rectifyComments.value[item.id] || '',
    })
  }
  submitting.value = true
  try {
    await api.post(`/inspection-tasks/${task.value?.id}/submit-rectify`, { items: payloadItems })
    Message.success('整改已提交')
    submitVisible.value = false
    selectedItemIds.value = []
    await fetchDetail()
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '提交失败')
  } finally {
    submitting.value = false
  }
}

async function handleRecheck() {
  if (!selectedPlanId.value || !selectedModelId.value) {
    Message.warning('请选择 AI 复核模型')
    return
  }
  if (!selectedHasVision.value) {
    Message.warning('请选择支持视觉理解的模型')
    return
  }
  rechecking.value = true
  try {
    const res = await api.post(`/inspection-tasks/${task.value?.id}/recheck`, {
      items: submittedItems.value.map((it) => ({ item_id: it.id })),
      plan_id: selectedPlanId.value,
      model_id: selectedModelId.value,
    })
    Message.success(`复核完成：${statusText(res.data.status)}`)
    await fetchDetail()
  } catch (e) {
    const detail = getApiErrorDetail(e)
    if (detail) {
      Message.error(detail)
    } else {
      Message.error('复核失败')
    }
  } finally {
    rechecking.value = false
  }
}

function openConfirmModal() {
  if (selectedItems.value.length === 0) {
    Message.warning('请先选择要人工确认的问题项')
    return
  }
  confirmResults.value = {}
  confirmComments.value = {}
  for (const item of selectedItems.value) {
    confirmResults.value[item.id] = true
    confirmComments.value[item.id] = ''
  }
  confirmVisible.value = true
}

async function handleManualConfirm() {
  const payloadItems = selectedItems.value.map((it) => ({
    item_id: it.id,
    confirmed: confirmResults.value[it.id] ?? true,
    comment: confirmComments.value[it.id] || '',
  }))
  confirming.value = true
  try {
    await api.post(`/inspection-tasks/${task.value?.id}/manual-confirm`, { items: payloadItems })
    Message.success('人工确认完成')
    confirmVisible.value = false
    selectedItemIds.value = []
    await fetchDetail()
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '确认失败')
  } finally {
    confirming.value = false
  }
}

async function fetchDetail() {
  loading.value = true
  try {
    const res = await api.get<{
      task: InspectionTask
      items: InspectionTaskItem[]
      logs: InspectionTaskLog[]
    }>(`/inspection-tasks/${route.params.id}`)
    task.value = res.data.task
    items.value = res.data.items || []
    logs.value = res.data.logs || []
    selectedItemIds.value = []
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载整改任务详情失败')
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push({ name: 'InspectionTaskList' })
}

onMounted(() => {
  fetchDetail()
})
</script>
