<template>
  <div class="page-main">
    <PageHeader
      :title="isEdit ? '编辑素材' : '新建素材'"
      subtitle="配置可复用的检查项素材，包含标题、检查标准、标准图与评分"
    />

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <div class="max-w-3xl">
        <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5 mb-4">
          <a-form :model="form" layout="vertical">
            <a-form-item label="分类">
              <a-input
                v-model="form.category"
                placeholder="如：形象 / 卫生（可选）"
                :maxlength="50"
                show-word-limit
              />
            </a-form-item>

            <a-form-item label="标题" required>
              <a-input
                v-model="form.title"
                placeholder="如：门头形象"
                :maxlength="100"
                show-word-limit
              />
            </a-form-item>

            <a-form-item label="检查标准与标准图">
              <AttachmentInputArea
                ref="standardAreaRef"
                v-model="standardImages"
                v-model:text="form.standard"
                :upload="uploadImageFile"
                :max-count="MAX_STANDARD_IMAGES"
                :max-length="500"
                :min-rows="3"
                :max-rows="6"
                placeholder="填写该项的检查标准..."
              />
              <template #extra>
                <div class="text-[11px] text-[#86868b] mt-1.5">
                  单张不超过 10MB，最多 {{ MAX_STANDARD_IMAGES }} 张 ({{ standardImages.length }}/{{
                    MAX_STANDARD_IMAGES
                  }}), 支持拖拽 / 粘贴图片，或粘贴图片链接自动识别为标准图
                </div>
              </template>
            </a-form-item>

            <a-form-item label="评分方式" required>
              <a-radio-group v-model="form.score_type" type="button" size="small">
                <a-radio value="score">分值评分</a-radio>
                <a-radio value="pass_fail">选项评分</a-radio>
              </a-radio-group>
              <div class="text-[11px] text-[#86868b] mt-1.5">
                分值评分按满分计算；选项评分不参与分数计算，仅记录所选序号。
              </div>
            </a-form-item>

            <a-form-item :label="form.score_type === 'score' ? '评分选项' : '选项设置'">
              <div class="w-full rounded-lg border border-[#E5E5EA] p-2.5">
                <div
                  v-for="(opt, optIndex) in form.score_options"
                  :key="optIndex"
                  class="flex items-center gap-1.5 mb-1.5"
                >
                  <span class="text-[11px] text-[#86868b] w-7 shrink-0">{{
                    form.score_type === 'pass_fail' ? '序号' : '分值'
                  }}</span>
                  <a-select
                    v-if="form.score_type === 'score'"
                    :model-value="Number(opt.score)"
                    size="mini"
                    class="!w-24"
                    placeholder="选择分值"
                    @change="(val: any) => onOptionScoreSelect(optIndex, val)"
                  >
                    <a-option
                      v-for="v in ALLOWED_SCORE_VALUES"
                      :key="v"
                      :value="v"
                      :disabled="
                        form.score_options.some((o, i) => i !== optIndex && Number(o.score) === v)
                      "
                      >{{ v }}</a-option
                    >
                  </a-select>
                  <span
                    v-else
                    class="inline-flex items-center justify-center rounded-md bg-[#F5F5F7] text-[#1D1D1F] text-[13px] font-semibold min-w-[60px] h-[28px] px-2 select-none tabular-nums"
                    >{{ optIndex + 1 }}</span
                  >
                  <span class="text-[11px] text-[#86868b] w-7 shrink-0">{{
                    form.score_type === 'pass_fail' ? '选项' : '描述'
                  }}</span>
                  <a-input
                    v-model="opt.label"
                    :placeholder="form.score_type === 'pass_fail' ? '选项名（必填）' : '如：0分'"
                    size="mini"
                    class="flex-1"
                  />
                  <a-button
                    type="text"
                    size="mini"
                    status="danger"
                    :disabled="form.score_options.length <= 1"
                    @click="removeOption(optIndex)"
                  >
                    <template #icon><IconDelete :size="12" /></template>
                  </a-button>
                </div>
                <a-button type="outline" size="mini" :disabled="!canAddOption" @click="addOption">
                  <template #icon><IconPlus :size="10" /></template>
                  添加选项
                </a-button>
                <div class="text-[11px] text-[#FF9500] mt-1.5 leading-relaxed">
                  {{
                    form.score_type === 'pass_fail'
                      ? `选项评分不参与分数计算，仅记录所选序号；选项名必填且不重复。`
                      : `分值仅允许 ${ALLOWED_SCORE_VALUES.join('、')}，必填不重复；必须包含最小值 ${MIN_SCORE} 与最大值 ${MAX_SCORE}。`
                  }}
                </div>
              </div>
            </a-form-item>
          </a-form>
        </div>
      </div>
    </div>

    <div
      class="sticky bottom-0 z-30 border-t border-[#E5E5EA] bg-white/90 backdrop-blur-xl px-4 md:px-6 lg:px-8 py-3 flex items-center justify-end gap-3"
    >
      <a-button size="small" @click="goBack">取消</a-button>
      <a-button type="primary" size="small" :loading="saving" @click="handleSave">保存</a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconDelete } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import { uploadImageFile } from '@/composables/useFileUpload'
import type { InspectionMaterial, ScoreOption } from '@/types'
import api from '@/utils/api'

const ALLOWED_SCORE_VALUES = [0, 0.5, 1, 2, 3, 4, 5]
const MIN_SCORE = 0
const MAX_SCORE = 5
const MAX_STANDARD_IMAGES = 5

const DEFAULT_SCORE_OPTIONS: ScoreOption[] = [
  { score: 0, label: '0分' },
  { score: 5, label: '5分' },
]

const DEFAULT_PASS_FAIL_OPTIONS: ScoreOption[] = [
  { score: 1, label: '合格' },
  { score: 2, label: '不合格' },
]

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => !!route.params.id)
const saving = ref(false)
const loading = ref(false)

const form = ref({
  category: '',
  title: '',
  standard: '',
  standard_image: '',
  score_type: 'score' as 'score' | 'pass_fail',
  max_score: 5,
  score_options: cloneOptions(DEFAULT_SCORE_OPTIONS),
})

const standardImages = ref<string[]>([])
const standardAreaRef = ref<{ flushPending: () => Promise<boolean> }>()

watch(
  () => form.value.score_type,
  (newType, oldType) => {
    if (newType === oldType) return
    form.value.score_options = cloneOptions(
      newType === 'pass_fail' ? DEFAULT_PASS_FAIL_OPTIONS : DEFAULT_SCORE_OPTIONS,
    )
  },
)

const canAddOption = computed(() => {
  if (form.value.score_type === 'pass_fail') return true
  return form.value.score_options.length < ALLOWED_SCORE_VALUES.length
})

function cloneOptions(options: ScoreOption[]): ScoreOption[] {
  return options.map((o) => ({ score: o.score, label: o.label }))
}

function onOptionScoreSelect(index: number, val: any) {
  form.value.score_options[index].score = Number(val)
}

function addOption() {
  if (form.value.score_type === 'pass_fail') {
    const nextIndex = form.value.score_options.length + 1
    form.value.score_options.push({ score: nextIndex, label: '' })
    return
  }
  const used = new Set(form.value.score_options.map((o) => Number(o.score)))
  for (const v of ALLOWED_SCORE_VALUES) {
    if (!used.has(v)) {
      form.value.score_options.push({ score: v, label: `${v}分` })
      break
    }
  }
}

function removeOption(index: number) {
  if (form.value.score_options.length <= 1) return
  form.value.score_options.splice(index, 1)
}

function validateForm(): string | null {
  if (!form.value.title.trim()) return '请填写素材标题'
  if (form.value.score_options.length === 0) return '评分选项不能为空'
  if (form.value.score_type === 'score') {
    const used = new Set<number>()
    let hasMin = false
    let hasMax = false
    for (const opt of form.value.score_options) {
      const score = Number(opt.score)
      if (!ALLOWED_SCORE_VALUES.includes(score)) return '分值仅允许 0、0.5、1、2、3、4、5'
      if (used.has(score)) return '选项分值不能重复'
      used.add(score)
      if (score === MIN_SCORE) hasMin = true
      if (score === MAX_SCORE) hasMax = true
    }
    if (!hasMin || !hasMax) return '评分选项必须包含最小值 0 与最大值 5'
  } else {
    const used = new Set<string>()
    for (const opt of form.value.score_options) {
      const label = opt.label.trim()
      if (!label) return '选项名不能为空'
      if (used.has(label)) return '选项名不能重复'
      used.add(label)
    }
  }
  return null
}

async function handleSave() {
  const error = validateForm()
  if (error) {
    Message.warning(error)
    return
  }
  if (standardAreaRef.value && !(await standardAreaRef.value.flushPending())) {
    return
  }
  saving.value = true
  try {
    const payload = {
      category: form.value.category,
      title: form.value.title.trim(),
      standard: form.value.standard,
      standard_image: standardImages.value[0] || '',
      standard_images: standardImages.value,
      score_type: form.value.score_type,
      max_score:
        form.value.score_type === 'pass_fail'
          ? form.value.score_options.length
          : form.value.max_score,
      score_options: form.value.score_options.map((o) => ({
        score: Number(o.score) || 0,
        label: o.label.trim(),
      })),
    }
    if (isEdit.value) {
      await api.put(`/inspection-materials/${route.params.id}`, payload)
      Message.success('素材已更新')
    } else {
      await api.post('/inspection-materials/', payload)
      Message.success('素材已创建')
    }
    router.push('/inspection/materials')
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '保存失败')
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push('/inspection/materials')
}

async function fetchDetail() {
  if (!isEdit.value) return
  loading.value = true
  try {
    const res = await api.get<InspectionMaterial>(`/inspection-materials/${route.params.id}`)
    const data = res.data
    form.value = {
      category: data.category || '',
      title: data.title,
      standard: data.standard || '',
      standard_image: data.standard_image || '',
      score_type: data.score_type,
      max_score: data.max_score,
      score_options:
        data.score_options && data.score_options.length > 0
          ? data.score_options.map((o) => ({ score: o.score, label: o.label }))
          : cloneOptions(
              data.score_type === 'pass_fail' ? DEFAULT_PASS_FAIL_OPTIONS : DEFAULT_SCORE_OPTIONS,
            ),
    }
    const images =
      data.standard_images && data.standard_images.length > 0
        ? data.standard_images
        : data.standard_image
          ? [data.standard_image]
          : []
    standardImages.value = images
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载素材失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchDetail)
</script>
