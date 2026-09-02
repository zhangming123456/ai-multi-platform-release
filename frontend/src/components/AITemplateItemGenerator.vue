<template>
  <a-drawer
    :visible="visible"
    title="AI 智能添加检查项"
    :width="660"
    :drawer-style="{
      maxWidth: '100%',
    }"
    :footer="false"
    placement="right"
    unmount-on-close
    @cancel="onClose"
    @before-open="onOpen"
  >
    <div class="space-y-4">
      <a-form ref="formRef" :model="formState" layout="vertical" class="space-y-3">
        <a-form-item
          label="AI 模型"
          field="model"
          :rules="modelRules"
          :validate-trigger="['change', 'blur']"
        >
          <ModelSelect
            v-model="formState.model"
            :require-vision="formState.photoUrls.length > 0"
            size="small"
            @change="onModelChange"
          />
        </a-form-item>
        <a-form-item label="图片 / 描述" field="content" :rules="contentRules">
          <AttachmentInputArea
            v-model="formState.description"
            v-model:file-list="formState.photoUrls"
            :upload="fileToDataUrl"
            upload-mode="auto"
            :max-count="10"
            :max-length="500"
            :min-rows="3"
            :max-rows="6"
            placeholder="例如：需要检查门店门头招牌是否完好、灯箱是否正常亮起，以及地面是否干净无积水…"
            @change="onContentChange"
          />
          <template #extra>
            <span
              >图片将直接以 base64 作为 AI
              识别输入（无需上传服务器），确认添加后作为标准图保存</span
            >
          </template>
        </a-form-item>
        <a-form-item>
          <a-button
            type="primary"
            long
            :loading="generating"
            :disabled="generatedItems.length > 0"
            @click="handleGenerate"
          >
            <template #icon><icon-robot :size="15" /></template>
            {{ generating ? '识别中…' : '开始识别' }}
          </a-button>
        </a-form-item>
      </a-form>

      <div
        v-if="generating || aiLogs.length > 0"
        class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-4 space-y-2"
      >
        <div class="text-[13px] font-medium text-[#1D1D1F]">识别进度</div>
        <div v-if="aiLogs.length" class="max-h-28 overflow-y-auto space-y-1 pr-1">
          <div
            v-for="(log, i) in aiLogs"
            :key="'l-' + i"
            class="flex items-start gap-1.5 text-[12px] leading-5"
          >
            <span
              class="shrink-0 w-1.5 h-1.5 rounded-full mt-[7px]"
              :style="{ backgroundColor: logColor(log.level) }"
            />
            <span class="text-[#3A3A3C] break-all">{{ log.message }}</span>
          </div>
        </div>
        <div
          v-if="streamingText"
          class="text-[12px] leading-5 text-[#3A3A3C] whitespace-pre-wrap break-all max-h-32 overflow-y-auto border-t border-[#F0F0F2] pt-2"
        >
          {{ streamingText }}
        </div>
      </div>

      <template v-if="generatedItems.length > 0">
        <div class="text-[14px] font-semibold text-[#1D1D1F]">
          识别结果（{{ generatedItems.length }} 项，可编辑后确认添加）
        </div>
        <div
          v-if="aiTemplateName || aiTemplateDescription"
          class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-4 space-y-2"
        >
          <div class="text-[12px] text-[#86868b]">
            AI 建议的模板信息（未填写的模板名称与描述将自动回填）
          </div>
          <div>
            <div class="text-[12px] text-[#86868b] mb-0.5">模板名称</div>
            <a-input v-model="aiTemplateName" size="small" placeholder="模板名称" :maxlength="20" />
          </div>
          <div>
            <div class="text-[12px] text-[#86868b] mb-0.5">模板描述</div>
            <a-textarea
              v-model="aiTemplateDescription"
              size="small"
              placeholder="模板描述"
              :auto-size="{ minRows: 1, maxRows: 3 }"
              :maxlength="50"
              show-word-limit
            />
          </div>
        </div>
        <div
          v-for="(item, idx) in generatedItems"
          :key="'g-' + idx"
          class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-4 space-y-2"
        >
          <div class="grid grid-cols-[1fr_110px] gap-2">
            <div>
              <div class="text-[12px] text-[#86868b] mb-0.5">分类</div>
              <a-input v-model="item.category" size="small" placeholder="分类名称" />
            </div>
            <div>
              <div class="text-[12px] text-[#86868b] mb-0.5">满分</div>
              <a-select v-model="item.max_score" size="small">
                <a-option v-for="s in [1, 2, 3, 4, 5]" :key="s" :value="s">{{ s }} 分</a-option>
              </a-select>
            </div>
          </div>
          <div>
            <div class="text-[12px] text-[#86868b] mb-0.5">检查项标题</div>
            <a-input v-model="item.title" size="small" placeholder="检查项标题" :maxlength="20" />
          </div>
          <div>
            <div class="text-[12px] text-[#86868b] mb-0.5">检查标准</div>
            <a-textarea
              v-model="item.standard"
              size="small"
              placeholder="检查标准"
              :auto-size="{ minRows: 2, maxRows: 4 }"
              :maxlength="120"
            />
          </div>
          <div v-if="item.standard_images && item.standard_images.length > 0">
            <div class="text-[12px] text-[#86868b] mb-1">标准图</div>
            <div class="flex flex-wrap gap-2">
              <div
                v-for="(img, i2) in item.standard_images"
                :key="'im-' + i2"
                class="relative w-[56px] h-[56px] rounded-lg overflow-hidden border border-[#E5E5EA] bg-[#F5F5F7]"
              >
                <img :src="img" class="w-full h-full object-cover" alt="" />
                <button
                  class="absolute top-0.5 right-0.5 w-4 h-4 rounded-full bg-black/50 text-white text-[10px] leading-none flex items-center justify-center hover:bg-black/70"
                  @click="removeStandardImage(idx, i2)"
                >
                  ×
                </button>
              </div>
            </div>
          </div>
        </div>
        <a-button type="primary" long :loading="confirming" @click="handleConfirm">
          确认添加 {{ generatedItems.length }} 个检查项
        </a-button>
      </template>
    </div>
  </a-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, onBeforeUnmount } from 'vue'
import { Message } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconRobot } from '@arco-design/web-vue/es/icon'
import ModelSelect from '@/components/shared/ModelSelect.vue'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import { uploadImageFile } from '@/composables/useFileUpload'
import { getApiErrorDetail } from '@/utils/api'

export interface AIGeneratedItem {
  category: string
  title: string
  standard: string
  standard_images: string[]
  score_type: 'score' | 'pass_fail'
  max_score: number
  score_options: { score: number; label: string }[]
}

const props = defineProps<{
  visible: boolean
  existingCategories: string[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'confirm', result: { name: string; description: string; items: AIGeneratedItem[] }): void
}>()

const formRef = ref<FormInstance>()
const formState = reactive({
  model: '',
  description: '',
  photoUrls: [] as string[],
})

const modelRules = [{ required: true, message: '请选择 AI 模型' }]

const contentRules = [
  {
    validator: (_value: unknown, callback: (error?: string) => void) => {
      if (!formState.description.trim() && formState.photoUrls.length === 0) {
        callback('请上传现场图片或填写检查项描述（至少填一项）')
      } else {
        callback()
      }
    },
  },
]

const generatedItems = ref<AIGeneratedItem[]>([])
const generating = ref(false)
const confirming = ref(false)
const aiTemplateName = ref('')
const aiTemplateDescription = ref('')
const aiLogs = ref<{ level: string; message: string }[]>([])
const streamingText = ref('')
let streamAbort: AbortController | null = null

function pushLog(level: string, message: string) {
  aiLogs.value.push({ level, message })
}

function logColor(level: string): string {
  switch (level) {
    case 'err':
      return '#E5484D'
    case 'warn':
      return '#F5A524'
    case 'ok':
      return '#30A46C'
    case 'req':
      return '#86868B'
    default:
      return '#007AFF'
  }
}

function onOpen() {
  formState.model = ''
  formState.description = ''
  formState.photoUrls = []
  generatedItems.value = []
  aiTemplateName.value = ''
  aiTemplateDescription.value = ''
  aiLogs.value = []
  streamingText.value = ''
}

function onClose() {
  if (generating.value) return
  emit('update:visible', false)
}

function onModelChange() {
  void formRef.value?.validateField('model').catch(() => {})
}

function onContentChange() {
  void formRef.value?.validateField('content').catch(() => {})
}

function removeStandardImage(itemIdx: number, imgIdx: number) {
  const item = generatedItems.value[itemIdx]
  if (item) item.standard_images.splice(imgIdx, 1)
}

async function handleGenerate() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  generating.value = true
  aiLogs.value = []
  streamingText.value = ''
  pushLog('req', '开始 AI 智能添加检查项…')
  pushLog('info', `使用模型配置 ${formState.model}`)

  streamAbort = new AbortController()
  try {
    const sep = formState.model.indexOf(':')
    const planId = sep > 0 ? formState.model.slice(0, sep) : formState.model
    const mid = sep > 0 ? formState.model.slice(sep + 1) : ''
    const photos = await Promise.all(formState.photoUrls.map(urlToBase64))
    const response = await fetch('/api/inspection-templates/ai-generate-items-stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({
        plan_id: planId,
        model_id: mid,
        description: formState.description.trim(),
        photos,
        existing_categories: props.existingCategories,
      }),
      signal: streamAbort.signal,
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      const detail = errorData.detail || 'AI 识别失败，请重试'
      pushLog('err', `HTTP ${response.status} · ${detail}`)
      Message.error(String(detail))
      return
    }

    pushLog('req', 'POST /api/inspection-templates/ai-generate-items-stream → SSE 连接已建立')

    const reader = response.body!.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const events = buffer.split('\n\n')
      buffer = events.pop() || ''
      for (const eventStr of events) {
        if (!eventStr.trim()) continue
        const lines = eventStr.split('\n')
        let eventType = ''
        let dataStr = ''
        for (const line of lines) {
          if (line.startsWith('event:')) eventType = line.slice(6).trim()
          else if (line.startsWith('data:')) dataStr = line.slice(5).trim()
        }
        if (!eventType || !dataStr) continue
        let payload: {
          level?: string
          message?: string
          text?: string
          items?: AIGeneratedItem[]
          template_name?: string
          template_description?: string
        }
        try {
          payload = JSON.parse(dataStr)
        } catch {
          continue
        }
        switch (eventType) {
          case 'log':
            pushLog(payload.level || 'info', payload.message || '')
            break
          case 'chunk':
            streamingText.value += payload.text || ''
            break
          case 'result': {
            const items: AIGeneratedItem[] = Array.isArray(payload.items) ? payload.items : []
            if (items.length > 0) {
              for (const it of items) {
                it.standard_images = it.standard_images || []
                it.max_score = it.max_score || 5
                if (it.score_type !== 'pass_fail') it.score_type = 'score'
                it.score_options = normalizeScoreOptions(it.max_score)
              }
              generatedItems.value = items
              if (typeof payload.template_name === 'string')
                aiTemplateName.value = payload.template_name.trim()
              if (typeof payload.template_description === 'string')
                aiTemplateDescription.value = payload.template_description.trim()
              pushLog('ok', `识别出 ${items.length} 个检查项`)
              Message.success(`识别出 ${items.length} 个检查项`)
            } else {
              pushLog('err', 'AI 未识别出检查项')
            }
            break
          }
          case 'error':
            pushLog('err', payload.message || 'AI 识别失败')
            Message.error(payload.message || 'AI 识别失败')
            break
        }
      }
    }
    if (generatedItems.value.length === 0) {
      const hasErr = aiLogs.value.some((l) => l.level === 'err')
      if (!hasErr) {
        pushLog('err', '未收到识别结果，请重试')
        Message.error('未收到识别结果，请重试')
      }
    }
  } catch (e) {
    const errMessage = e instanceof Error ? e.message : ''
    if (!(e instanceof Error && e.name === 'AbortError')) {
      pushLog('err', errMessage || 'AI 识别失败')
      Message.error(errMessage || 'AI 识别失败')
    }
  } finally {
    generating.value = false
    streamAbort = null
  }
}

function fileToDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

async function urlToBase64(url: string): Promise<{ data: string; mime_type: string }> {
  if (url.startsWith('data:')) {
    const commaIdx = url.indexOf(',')
    const meta = url.slice(5, commaIdx)
    const data = url.slice(commaIdx + 1)
    const mime_type = meta.split(';')[0] || 'image/jpeg'
    return { data, mime_type }
  }
  const resp = await fetch(url)
  const blob = await resp.blob()
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      resolve({ data: result.split(',')[1] || '', mime_type: blob.type || 'image/jpeg' })
    }
    reader.onerror = reject
    reader.readAsDataURL(blob)
  })
}

function normalizeScoreOptions(maxScore: number): { score: number; label: string }[] {
  const max = Math.min(5, Math.max(1, Number(maxScore) || 5))
  return [
    { score: 0, label: '0分' },
    { score: max, label: `${max}分` },
  ]
}

async function resolveDataUrl(url: string): Promise<string> {
  if (!url.startsWith('data:')) return url
  try {
    const resp = await fetch(url)
    const blob = await resp.blob()
    const file = new File([blob], `std-${Date.now()}.jpg`, { type: blob.type || 'image/jpeg' })
    return await uploadImageFile(file)
  } catch {
    return ''
  }
}

async function handleConfirm() {
  confirming.value = true
  try {
    const items: AIGeneratedItem[] = []
    for (const it of generatedItems.value) {
      const imgs = await Promise.all((it.standard_images || []).map(resolveDataUrl))
      items.push({ ...it, standard_images: imgs.filter(Boolean) })
    }
    emit('confirm', {
      name: aiTemplateName.value.trim(),
      description: aiTemplateDescription.value.trim(),
      items,
    })
    emit('update:visible', false)
  } catch (e) {
    const message = e instanceof Error ? e.message : ''
    Message.error(getApiErrorDetail(e) || message || '标准图上传失败')
  } finally {
    confirming.value = false
  }
}

onBeforeUnmount(() => {
  streamAbort?.abort()
})
</script>
