<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconStar,
  IconCopy,
  IconSave,
  IconSettings,
  IconCode,
  IconDelete,
  IconDown,
  IconUp,
  IconEdit,
  IconMindMapping,
  IconEye,
  IconImage,
  IconVideoCamera,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import {
  useTokenPlanStore,
  MODEL_TYPE_LABELS,
  MODEL_TYPE_COLORS,
  MODEL_TYPE_ICONS,
  type ModelType,
} from '@/stores/tokenPlan'
import api from '@/utils/api'

const router = useRouter()
const store = useTokenPlanStore()

onMounted(() => {
  if (!store.loaded) {
    store.loadPlans()
  }
})

type LogLevel = 'info' | 'req' | 'ok' | 'err'

interface LogEntry {
  id: number
  time: string
  level: LogLevel
  message: string
}

const logs = ref<LogEntry[]>([])
const logExpanded = ref(true)
const logPanelRef = ref<HTMLElement | null>(null)
const streamingTextRef = ref<HTMLElement | null>(null)
let logSeq = 0

function pushLog(level: LogLevel, message: string) {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  logs.value.push({
    id: ++logSeq,
    time: `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}.${String(
      now.getMilliseconds(),
    ).padStart(3, '0')}`,
    level,
    message,
  })
  if (logs.value.length > 200) {
    logs.value.splice(0, logs.value.length - 200)
  }
}

function clearLogs() {
  logs.value = []
}

watch(
  () => logs.value.length,
  async () => {
    await nextTick()
    if (logPanelRef.value) {
      logPanelRef.value.scrollTop = logPanelRef.value.scrollHeight
    }
  },
)

function platformLabel(value: string) {
  return platformChoices.find((p) => p.value === value)?.label || value
}

function modelTypeLabel(type: string): string {
  return MODEL_TYPE_LABELS[type as ModelType] || '文本'
}

function modelTypeColor(type: string): string {
  return MODEL_TYPE_COLORS[type as ModelType] || '#007AFF'
}

const MODEL_ICON_MAP: Record<string, typeof IconEdit> = {
  IconEdit,
  IconMindMapping,
  IconEye,
  IconImage,
  IconVideoCamera,
}

function modelTypeIcon(type: string) {
  const key = MODEL_TYPE_ICONS[type as ModelType] || 'IconEdit'
  return MODEL_ICON_MAP[key] || IconEdit
}

const fileUploadTip = computed(() => {
  const entry = store.selectedModelEntry
  if (!entry) return ''
  const types = entry.types
  if (types.includes('image') && types.includes('video')) {
    return '支持上传图片和视频文件（最多 10 个）'
  }
  if (types.includes('image')) {
    return '支持上传图片文件（最多 10 个）'
  }
  if (types.includes('video')) {
    return '支持上传视频文件（最多 10 个）'
  }
  return '支持上传图片/视频文件（最多 10 个）'
})

function levelText(level: LogLevel) {
  switch (level) {
    case 'req':
      return 'REQ '
    case 'ok':
      return 'OK  '
    case 'err':
      return 'ERR '
    default:
      return 'INFO'
  }
}

const topic = ref('')
const keywords = ref('')
const selectedPlatforms = ref<string[]>(['xiaohongshu'])
const isGenerating = ref(false)
const streamingText = ref('')
const uploadedFiles = ref<any[]>([])
const streamingPlatform = ref('')

function getNativeFile(file: any): File | null {
  return file?.file || file?.originFile || (file instanceof File ? file : null)
}

function handleFileChange(fileList: any[]) {
  uploadedFiles.value = fileList.filter((item) => {
    const file = getNativeFile(item)
    if (!file) return false
    if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) {
      Message.warning(`不支持的文件类型：${file.name}，仅支持图片和视频`)
      return false
    }
    if (file.type.startsWith('video/') && file.size > 50 * 1024 * 1024) {
      Message.warning(`视频文件过大：${file.name}（最大 50MB）`)
      return false
    }
    return true
  })
}

function compressImage(file: File, maxWidth = 1920, quality = 0.8): Promise<File> {
  return new Promise((resolve, reject) => {
    if (!file.type.startsWith('image/')) {
      resolve(file)
      return
    }
    const reader = new FileReader()
    reader.onload = (e) => {
      const img = new Image()
      img.onload = () => {
        let { width, height } = img
        if (width > maxWidth) {
          height = (height * maxWidth) / width
          width = maxWidth
        }
        const canvas = document.createElement('canvas')
        canvas.width = width
        canvas.height = height
        const ctx = canvas.getContext('2d')
        if (!ctx) {
          resolve(file)
          return
        }
        ctx.drawImage(img, 0, 0, width, height)
        canvas.toBlob(
          (blob) => {
            if (blob) {
              const compressed = new File([blob], file.name, {
                type: file.type || 'image/jpeg',
                lastModified: Date.now(),
              })
              resolve(compressed)
            } else {
              resolve(file)
            }
          },
          file.type || 'image/jpeg',
          quality,
        )
      }
      img.onerror = () => resolve(file)
      img.src = e.target?.result as string
    }
    reader.onerror = () => resolve(file)
    reader.readAsDataURL(file)
  })
}

async function fileToBase64(file: File): Promise<{ data: string; mime_type: string }> {
  const processed = file.type.startsWith('image/') ? await compressImage(file) : file
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      const base64 = result.split(',')[1] || ''
      resolve({ data: base64, mime_type: processed.type || 'application/octet-stream' })
    }
    reader.onerror = reject
    reader.readAsDataURL(processed)
  })
}
const isSaving = ref(false)
const hasGenerated = ref(false)
const activePreview = ref('')

watch(streamingText, async () => {
  await nextTick()
  if (streamingTextRef.value) {
    streamingTextRef.value.scrollTop = streamingTextRef.value.scrollHeight
  }
})

const platformChoices = [
  { value: 'wechat_mp', label: '公众号' },
  { value: 'xiaohongshu', label: '小红书' },
  { value: 'douyin', label: '抖音' },
  { value: 'wechat_video', label: '视频号' },
]

const generatedVariants = ref<Record<string, { title: string; body: string; hashtags: string[] }>>(
  {},
)

const previewPlatforms = computed(() =>
  platformChoices.filter((p) => selectedPlatforms.value.includes(p.value)),
)

async function generate() {
  if (!store.loaded) {
    await store.loadPlans()
  }
  if (!store.activePlan) {
    pushLog('err', '未检测到可用的模型配置，已跳转至模型管理页')
    router.push('/settings/token-plan')
    return
  }
  if (!topic.value) return

  isGenerating.value = true
  hasGenerated.value = false
  streamingText.value = ''
  streamingPlatform.value = ''

  const startedAt = performance.now()
  const plan = store.activePlan
  const modelId = store.selectedModelId || store.activeModelList[0]?.id || ''
  pushLog(
    'info',
    `开始生成任务 · 主题「${topic.value}」 · 平台 ${selectedPlatforms.value
      .map(platformLabel)
      .join(' / ')}`,
  )
  pushLog('info', `使用模型配置 ${plan.name}（${modelId}）`)

  const keywordsArray = keywords.value
    ? keywords.value
        .split(',')
        .map((k) => k.trim())
        .filter(Boolean)
    : undefined

  let filesPayload: { data: string; mime_type: string }[] | undefined
  if (uploadedFiles.value.length > 0) {
    const nativeFiles = uploadedFiles.value
      .map((f) => getNativeFile(f))
      .filter((f): f is File => f !== null)
    if (nativeFiles.length > 0) {
      pushLog('info', `正在编码 ${nativeFiles.length} 个上传文件…`)
      filesPayload = await Promise.all(nativeFiles.map((f) => fileToBase64(f)))
      pushLog('ok', `文件编码完成，共 ${filesPayload.length} 个`)
    }
  }

  try {
    const response = await fetch('/api/contents/ai-generate-stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({
        topic: topic.value,
        platforms: selectedPlatforms.value,
        plan_id: plan.id,
        model_id: modelId,
        ...(keywordsArray && keywordsArray.length > 0 ? { keywords: keywordsArray } : {}),
        ...(filesPayload && filesPayload.length > 0 ? { files: filesPayload } : {}),
      }),
    })

    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        router.push('/login')
        return
      }
      const errorData = await response.json().catch(() => ({}))
      const detail = errorData.detail
      let errorMsg = 'AI 生成失败，请重试'
      if (typeof detail === 'object' && detail?.message) {
        errorMsg = detail.message
        if (Array.isArray(detail.available_plans) && detail.available_plans.length > 0) {
          const planNames = detail.available_plans
            .map(
              (p: { display_name?: string; name: string; provider: string; model: string }) =>
                `${p.display_name || p.name}（${p.provider}/${p.model}）`,
            )
            .join('、')
          Message.warning(`可前往设置切换至：${planNames}`)
        }
      } else if (typeof detail === 'string') {
        errorMsg = detail
      }
      pushLog('err', `HTTP ${response.status} · ${errorMsg}`)
      Message.error(errorMsg)
      return
    }

    pushLog('req', `POST /api/contents/ai-generate-stream → SSE 连接已建立`)

    const reader = response.body!.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    const variants: Record<string, { title: string; body: string; hashtags: string[] }> = {}

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

        const payload = JSON.parse(dataStr)

        switch (eventType) {
          case 'log':
            pushLog(payload.level || 'info', payload.message || '')
            break
          case 'chunk':
            if (streamingPlatform.value !== payload.platform) {
              streamingPlatform.value = payload.platform
              streamingText.value = ''
            }
            streamingText.value += payload.text || ''
            break
          case 'done':
            if (payload.variant) {
              variants[payload.platform] = {
                title: payload.variant.title || '',
                body: payload.variant.body || '',
                hashtags: Array.isArray(payload.variant.hashtags) ? payload.variant.hashtags : [],
              }
            }
            pushLog('ok', `${platformLabel(payload.platform)} 生成完成`)
            streamingText.value = ''
            streamingPlatform.value = ''
            break
          case 'error':
            pushLog('err', payload.message || '生成失败')
            streamingText.value = ''
            streamingPlatform.value = ''
            if (payload.available_plans?.length > 0) {
              const planNames = payload.available_plans
                .map(
                  (p: { display_name?: string; name: string; provider: string; model: string }) =>
                    `${p.display_name || p.name}（${p.provider}/${p.model}）`,
                )
                .join('、')
              Message.warning(`可前往设置切换至：${planNames}`)
            }
            Message.error(payload.message || 'AI 生成失败')
            break
          case 'complete':
            pushLog('ok', '全部生成完成')
            break
        }
      }
    }

    if (Object.keys(variants).length > 0) {
      generatedVariants.value = variants
      activePreview.value =
        selectedPlatforms.value.find((p) => variants[p]) || selectedPlatforms.value[0]
      hasGenerated.value = true
      const total = Math.round(performance.now() - startedAt)
      pushLog(
        'ok',
        `生成完成 · ${Object.keys(variants).length} 个平台就绪 · 总耗时 ${(total / 1000).toFixed(1)}s`,
      )
    } else {
      pushLog('err', '所有平台均未返回有效变体，本次生成结束')
      Message.error('未获取到生成结果，请重试')
    }
  } catch (error: unknown) {
    const err = error as { message?: string }
    pushLog('err', err.message || '网络异常')
    Message.error(err.message || 'AI 生成失败，请重试')
  } finally {
    isGenerating.value = false
    streamingText.value = ''
    streamingPlatform.value = ''
  }
}

async function saveContent() {
  const variant = generatedVariants.value[activePreview.value]
  if (!variant) return

  isSaving.value = true
  try {
    await api.post('/contents/', {
      title: variant.title,
      body: variant.body,
      platform: activePreview.value,
      status: 'draft',
    })
    Message.success('内容已保存为草稿，即将跳转到内容工坊')
    setTimeout(() => {
      router.push('/content')
    }, 800)
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: unknown } }; message?: string }
    const detail = err.response?.data?.detail
    if (detail && typeof detail === 'object') {
      const detailObj = detail as { message?: string }
      Message.error(detailObj.message || '保存失败，请重试')
    } else if (typeof detail === 'string') {
      Message.error(detail)
    } else {
      Message.error(err.message || '保存失败，请重试')
    }
  } finally {
    isSaving.value = false
  }
}

async function copyContent() {
  const variant = generatedVariants.value[activePreview.value]
  if (!variant) return
  const text = `${variant.title}\n\n${variant.body}\n\n${variant.hashtags.join(' ')}`
  try {
    await navigator.clipboard.writeText(text)
    Message.success('已复制到剪贴板')
  } catch {
    Message.error('复制失败，请手动选择文本复制')
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="AI 内容创建" subtitle="输入主题，一键生成适配各平台风格的内容变体">
      <template #actions>
        <a-tag
          :color="store.activePlan ? 'green' : 'orange'"
          size="small"
          @click="router.push('/settings/token-plan')"
          style="cursor: pointer"
        >
          <template #icon>
            <IconSettings :size="12" />
          </template>
          {{
            store.activePlan
              ? `${store.activePlan.name} - 剩余 ${store.getRemainingQuota().toLocaleString()} tokens`
              : '未配置 Token'
          }}
        </a-tag>
      </template>
    </PageHeader>

    <div class="content-layout">
      <a-card :bordered="false" title="生成预览" class="content-create-card content-preview-card">
        <template v-if="hasGenerated" #extra>
          <a-tabs
            :active-key="activePreview"
            @change="(key: string | number) => (activePreview = String(key))"
            type="rounded"
            size="mini"
          >
            <a-tab-pane v-for="p in previewPlatforms" :key="p.value" :title="p.label" />
          </a-tabs>
        </template>

        <a-empty v-if="!hasGenerated && !isGenerating">
          <template #image>
            <div class="w-14 h-14 rounded-[14px] bg-[#5856D6]/10 flex items-center justify-center">
              <IconStar :size="26" :style="{ color: '#5856D6' }" />
            </div>
          </template>
          <span class="text-[14px] font-medium">准备好开始创作了</span>
          <template #description>
            <span class="text-[12px]">填写下方主题与平台，点击生成按钮</span>
          </template>
        </a-empty>

        <div v-else-if="isGenerating && streamingText" class="streaming-view">
          <div class="streaming-view__header">
            <PlatformIcon
              :platform="
                streamingPlatform as 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
              "
              size="sm"
            />
            <span class="text-[13px] font-medium">{{ platformLabel(streamingPlatform) }}</span>
            <span class="streaming-view__badge">AI 生成中</span>
          </div>
          <div ref="streamingTextRef" class="streaming-view__text">
            {{ streamingText }}<span class="streaming-cursor"></span>
          </div>
        </div>

        <a-spin v-else-if="isGenerating" :loading="true" class="w-full py-10">
          <template #icon><IconStar :size="30" :style="{ color: '#007AFF' }" spin /></template>
          <div class="text-center">
            <p class="text-[13px] text-[#86868B]">
              正在为 {{ selectedPlatforms.length }} 个平台生成适配文案…
            </p>
          </div>
        </a-spin>

        <div v-else-if="generatedVariants[activePreview]" class="space-y-4">
          <div>
            <a-typography-text
              type="secondary"
              class="text-[11px] font-semibold uppercase tracking-[0.06em] block mb-1.5"
              >标题</a-typography-text
            >
            <a-input
              :model-value="generatedVariants[activePreview].title"
              read-only
              class="font-semibold"
            />
          </div>
          <div>
            <a-typography-text
              type="secondary"
              class="text-[11px] font-semibold uppercase tracking-[0.06em] block mb-1.5"
              >正文</a-typography-text
            >
            <a-textarea
              :model-value="generatedVariants[activePreview].body"
              read-only
              :auto-size="{ minRows: 4, maxRows: 10 }"
            />
          </div>
          <div>
            <a-typography-text
              type="secondary"
              class="text-[11px] font-semibold uppercase tracking-[0.06em] block mb-2"
              >推荐话题标签</a-typography-text
            >
            <a-space :size="6" wrap>
              <a-tag
                v-for="tag in generatedVariants[activePreview].hashtags"
                :key="tag"
                color="arcoblue"
              >
                {{ tag }}
              </a-tag>
            </a-space>
          </div>
          <div class="flex items-center justify-end gap-2 pt-1">
            <a-button @click="copyContent">
              <template #icon><IconCopy /></template>
              复制文案
            </a-button>
            <a-button type="primary" :loading="isSaving" @click="saveContent">
              <template #icon><IconSave /></template>
              保存为内容
            </a-button>
          </div>
        </div>
      </a-card>

      <div class="content-bottom">
        <div class="content-bottom__left">
          <div class="log-terminal animate-fade-up">
            <div
              class="log-terminal__bar"
              :class="{ 'log-terminal__bar--collapsed': !logExpanded }"
              @click="logExpanded = !logExpanded"
            >
              <div class="flex items-center gap-2">
                <span class="log-dot log-dot--r"></span>
                <span class="log-dot log-dot--y"></span>
                <span class="log-dot log-dot--g"></span>
                <IconCode :size="14" class="ml-2" style="color: #8e8e93" />
                <span class="log-terminal__title">API 调用日志</span>
                <span v-if="isGenerating" class="log-live">
                  <span class="log-live__pulse"></span>
                  实时监听中
                </span>
                <span v-else-if="logs.length > 0" class="log-idle">空闲</span>
              </div>
              <div class="flex items-center gap-1" @click.stop>
                <span class="log-terminal__count">{{ logs.length }} 条</span>
                <button class="log-terminal__btn" title="清空日志" @click="clearLogs">
                  <IconDelete :size="13" />
                </button>
                <button
                  class="log-terminal__btn"
                  :title="logExpanded ? '收起' : '展开'"
                  @click="logExpanded = !logExpanded"
                >
                  <IconUp v-if="logExpanded" :size="13" />
                  <IconDown v-else :size="13" />
                </button>
              </div>
            </div>

            <transition name="log-collapse">
              <div v-show="logExpanded" class="log-terminal__body-wrap">
                <div ref="logPanelRef" class="log-terminal__body">
                  <div v-if="logs.length === 0" class="log-terminal__empty">
                    <span class="log-terminal__prompt">➜</span>
                    暂无调用记录，点击「AI 生成内容」后这里将实时输出接口日志
                  </div>
                  <div
                    v-for="entry in logs"
                    :key="entry.id"
                    class="log-line"
                    :class="`log-line--${entry.level}`"
                  >
                    <span class="log-line__time">{{ entry.time }}</span>
                    <span class="log-line__level">{{ levelText(entry.level) }}</span>
                    <span class="log-line__msg">{{ entry.message }}</span>
                  </div>
                  <div v-if="isGenerating" class="log-line log-line--cursor">
                    <span class="log-terminal__prompt">➜</span>
                    <span class="log-cursor"></span>
                  </div>
                </div>
              </div>
            </transition>
          </div>
        </div>

        <div class="content-bottom__right">
          <a-card :bordered="false" title="创作输入" class="content-create-card">
            <a-form :model="{}" layout="vertical">
              <a-form-item label="内容主题">
                <a-input v-model="topic" placeholder="例如：春季护肤、职场成长、美食探店" />
              </a-form-item>
              <a-form-item label="关键词（可选）">
                <a-input v-model="keywords" placeholder="用逗号分隔，例如：保湿,防晒,敏感肌" />
              </a-form-item>
              <a-form-item label="目标平台">
                <a-checkbox-group v-model="selectedPlatforms">
                  <a-row :gutter="[8, 8]">
                    <a-col :span="12" v-for="choice in platformChoices" :key="choice.value">
                      <a-checkbox :value="choice.value">
                        <a-space :size="6" align="center">
                          <PlatformIcon
                            :platform="
                              choice.value as
                                'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
                            "
                            size="sm"
                          />
                          {{ choice.label }}
                        </a-space>
                      </a-checkbox>
                    </a-col>
                  </a-row>
                </a-checkbox-group>
              </a-form-item>
              <a-form-item v-if="store.activeModelList.length > 1" label="模型 ID">
                <a-select
                  v-model="store.selectedModelId"
                  placeholder="选择要使用的模型"
                  allow-search
                >
                  <a-option v-for="m in store.activeModelList" :key="m.id" :value="m.id">
                    <span class="model-opt">
                      <span class="model-opt__icons">
                        <component
                          v-for="t in m.types || ['text']"
                          :key="t"
                          :is="modelTypeIcon(t)"
                          :size="12"
                          :style="{ color: modelTypeColor(t) }"
                        />
                      </span>
                      {{ m.id }}
                    </span>
                  </a-option>
                </a-select>
              </a-form-item>
              <a-form-item v-if="store.selectedModelSupportsFiles" label="上传文件">
                <a-upload
                  :file-list="uploadedFiles"
                  @change="handleFileChange"
                  :auto-upload="false"
                  multiple
                  :limit="10"
                  draggable
                  :tip="fileUploadTip"
                  accept="image/*,video/*"
                />
              </a-form-item>
              <a-form-item>
                <a-button
                  type="primary"
                  long
                  :loading="isGenerating"
                  :disabled="!topic || isGenerating"
                  @click="generate"
                >
                  <template #icon><IconStar /></template>
                  {{ isGenerating ? 'AI 正在创作…' : 'AI 生成内容' }}
                </a-button>
              </a-form-item>
            </a-form>
            <a-typography-text type="secondary" class="text-[11px] block text-center">
              AI 将根据各平台的推荐算法与用户偏好调整文案风格
            </a-typography-text>
          </a-card>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.model-opt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.model-opt__icons {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.content-layout {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.content-preview-card {
  min-height: 200px;
}

.content-preview-card .arco-card-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.content-bottom {
  display: flex;
  gap: 12px;
  align-items: stretch;
  max-height: 750px;
  > div {
    flex: 1;
  }
}

.content-bottom__left {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.content-bottom__right {
  display: flex;
  flex-direction: column;
}

.content-bottom__right .content-create-card {
  flex: 1;
}

.content-bottom__left .log-terminal {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.content-bottom__left .log-terminal__body-wrap {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.content-bottom__left .log-terminal__body {
  max-height: none;
  height: 100%;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.streaming-view {
  padding: 4px 0;
}

.streaming-view__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.streaming-view__badge {
  font-size: 10px;
  color: #007aff;
  background: rgba(0, 122, 255, 0.1);
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
  letter-spacing: 0.04em;
}

.streaming-view__text {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14px;
  line-height: 1.85;
  color: #1d1d1f;
  max-height: 420px;
  overflow-y: auto;
  padding: 16px 18px;
  background: #f5f5f7;
  border-radius: 10px;
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.streaming-cursor {
  display: inline-block;
  width: 2px;
  height: 16px;
  background: #007aff;
  animation: cursor-blink 0.9s steps(1) infinite;
  vertical-align: text-bottom;
  margin-left: 1px;
  border-radius: 1px;
}

.log-terminal {
  border-radius: 12px;
  overflow: hidden;
  background: #1d1d1f;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.28),
    0 2px 8px rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.log-terminal__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: linear-gradient(180deg, #2c2c2e 0%, #252527 100%);
  cursor: pointer;
  user-select: none;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.log-terminal__bar--collapsed {
  border-bottom-color: transparent;
}

.log-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}
.log-dot--r {
  background: #ff5f57;
}
.log-dot--y {
  background: #febc2e;
}
.log-dot--g {
  background: #28c840;
}

.log-terminal__title {
  font-size: 12px;
  font-weight: 600;
  color: #d1d1d6;
  letter-spacing: 0.04em;
}

.log-live {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
  color: #34c759;
  margin-left: 8px;
  font-weight: 500;
}

.log-live__pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #34c759;
  animation: log-pulse 1.2s ease-in-out infinite;
}

@keyframes log-pulse {
  0%,
  100% {
    opacity: 1;
    box-shadow: 0 0 0 0 rgba(52, 199, 89, 0.5);
  }
  50% {
    opacity: 0.6;
    box-shadow: 0 0 0 4px rgba(52, 199, 89, 0);
  }
}

.log-idle {
  font-size: 10px;
  color: #636366;
  margin-left: 8px;
}

.log-terminal__count {
  font-size: 11px;
  color: #636366;
  font-variant-numeric: tabular-nums;
  margin-right: 6px;
}

.log-terminal__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: #8e8e93;
  cursor: pointer;
  transition:
    background 0.15s ease,
    color 0.15s ease;
}

.log-terminal__btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}

.log-terminal__body-wrap {
  overflow: hidden;
}

.log-terminal__body {
  max-height: 240px;
  overflow-y: auto;
  padding: 12px 14px;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, 'Cascadia Code', 'Roboto Mono', monospace;
  font-size: 12px;
  line-height: 1.7;
  background:
    radial-gradient(ellipse 60% 40% at 80% 0%, rgba(0, 122, 255, 0.05), transparent), #1d1d1f;
}

.content-bottom__left .log-terminal__body {
  max-height: none;
}

.log-terminal__empty {
  color: #48484a;
  font-size: 12px;
}

.log-terminal__prompt {
  color: #34c759;
  margin-right: 6px;
}

.log-line {
  display: flex;
  align-items: baseline;
  gap: 10px;
  animation: log-in 0.25s cubic-bezier(0.25, 0.1, 0.25, 1) both;
  padding: 1px 0;
}

@keyframes log-in {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.log-line__time {
  color: #48484a;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.log-line__level {
  flex-shrink: 0;
  font-weight: 700;
  font-size: 10px;
  letter-spacing: 0.08em;
  padding: 1px 6px;
  border-radius: 4px;
  white-space: pre;
}

.log-line--info .log-line__level {
  color: #8e8e93;
  background: rgba(142, 142, 147, 0.12);
}
.log-line--info .log-line__msg {
  color: #aeaeb2;
}

.log-line--req .log-line__level {
  color: #ff9f0a;
  background: rgba(255, 159, 10, 0.12);
}
.log-line--req .log-line__msg {
  color: #ffd60a;
}

.log-line--ok .log-line__level {
  color: #30d158;
  background: rgba(48, 209, 88, 0.12);
}
.log-line--ok .log-line__msg {
  color: #6ee7a0;
}

.log-line--err .log-line__level {
  color: #ff453a;
  background: rgba(255, 69, 58, 0.14);
}
.log-line--err .log-line__msg {
  color: #ff6961;
}

.log-line__msg {
  word-break: break-all;
}

.log-cursor {
  display: inline-block;
  width: 7px;
  height: 14px;
  background: #34c759;
  animation: cursor-blink 0.9s steps(1) infinite;
  vertical-align: middle;
}

@keyframes cursor-blink {
  50% {
    opacity: 0;
  }
}

.log-collapse-enter-active,
.log-collapse-leave-active {
  transition: opacity 0.22s ease;
  opacity: 1;
}

.log-collapse-enter-from,
.log-collapse-leave-to {
  opacity: 0;
}

@media (max-width: 768px) {
  .content-create-card .arco-card-body {
    padding: 14px 16px !important;
  }
  .content-create-card .arco-card-header {
    padding: 10px 16px !important;
  }
  .content-create-card .arco-card-header-title {
    font-size: 15px !important;
  }
  .content-bottom {
    grid-template-columns: 1fr;
  }
  .content-bottom__left .log-terminal__body {
    max-height: 200px;
  }
  .streaming-view__text {
    max-height: 320px;
    padding: 12px 14px;
    font-size: 13.5px;
    line-height: 1.75;
  }
  .log-terminal__body {
    font-size: 11px;
    padding: 10px 12px;
  }
  .log-terminal__bar {
    padding: 8px 12px;
  }
  .log-terminal__title {
    font-size: 11px;
  }
  .log-line {
    gap: 6px;
  }
  .log-line__time {
    font-size: 10.5px;
  }
  .log-line__level {
    font-size: 9px;
    padding: 1px 4px;
  }
}

@media (max-width: 480px) {
  .content-create-card .arco-card-body {
    padding: 12px 14px !important;
  }
  .streaming-view__text {
    max-height: 260px;
    padding: 10px 12px;
    font-size: 13px;
  }
  .content-bottom__left .log-terminal__body {
    max-height: 160px;
  }
  .streaming-view__header {
    margin-bottom: 10px;
  }
  .streaming-view__badge {
    font-size: 9.5px;
    padding: 2px 6px;
  }
}

@media (max-width: 248px) {
  .content-create-card .arco-card-body {
    padding: 10px 12px !important;
  }
  .content-create-card .arco-card-header {
    padding: 8px 12px !important;
  }
  .content-create-card .arco-card-header-title {
    font-size: 13px !important;
  }
  .platform-select-grid {
    grid-template-columns: 1fr;
    gap: 6px;
  }
  .platform-select-item {
    padding: 6px 8px;
  }
  .platform-select-item__name {
    font-size: 12px;
  }
  .streaming-view__text {
    max-height: 180px;
    padding: 8px 10px;
    font-size: 12px;
  }
  .streaming-view__header {
    margin-bottom: 8px;
    font-size: 12px;
  }
  .streaming-view__badge {
    font-size: 9px;
    padding: 1px 5px;
  }
  .content-bottom__left .log-terminal__body {
    max-height: 120px;
    font-size: 11px;
    padding: 8px 10px;
  }
  .log-terminal__toolbar {
    font-size: 11px;
    padding: 6px 10px;
  }
  .variant-preview__tabs {
    flex-wrap: wrap;
  }
  .variant-preview__tab {
    font-size: 11px;
    padding: 4px 8px;
  }
}
</style>
