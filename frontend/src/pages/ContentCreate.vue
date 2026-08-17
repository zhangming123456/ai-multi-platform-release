<template>
  <div class="page-main">
    <PageHeader title="创作内容" subtitle="输入主题，一键生成适配各平台风格的内容变体">
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

    <div class="px-4 md:px-6 lg:px-8 flex-1">
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
              <div
                class="w-14 h-14 rounded-[14px] bg-[#5856D6]/10 flex items-center justify-center"
              >
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

        <div class="content-left">
          <div class="log-terminal animate-fade-up">
            <div class="log-terminal__bar">
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
              <div class="flex items-center gap-1">
                <span class="log-terminal__count">{{ logs.length }} 条</span>
                <button class="log-terminal__btn" title="清空日志" @click="clearLogs">
                  <IconDelete :size="13" />
                </button>
              </div>
            </div>

            <div class="log-terminal__body-wrap">
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
          </div>

          <div class="chat-input-section">
            <div class="platform-select-bar">
              <button
                v-for="choice in platformChoices"
                :key="choice.value"
                type="button"
                class="platform-chip"
                :class="{ 'platform-chip--active': selectedPlatforms.includes(choice.value) }"
                @click="togglePlatform(choice.value)"
              >
                <PlatformIcon
                  :platform="
                    choice.value as 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
                  "
                  size="sm"
                />
                <span>{{ choice.label }}</span>
              </button>
            </div>

            <div
              ref="inputCardRef"
              class="chat-input-card"
              :class="{ 'chat-input-card--drag': isDragOver }"
              @dragenter.prevent="onDragEnter"
              @dragover.prevent="onDragOver"
              @dragleave.prevent="onDragLeave"
              @drop.prevent="onDrop"
            >
              <div v-if="uploadedFiles.length > 0" class="uploaded-files">
                <div v-for="(item, idx) in uploadedFiles" :key="idx" class="file-chip">
                  <template v-if="item.kind === 'link'">
                    <img v-if="item.isImage" :src="item.url" class="file-thumb" />
                    <div v-else class="file-icon"><IconFile :size="20" /></div>
                  </template>
                  <template v-else>
                    <img
                      v-if="isImage(item.file)"
                      :src="filePreview(item.file)"
                      class="file-thumb"
                    />
                    <div v-else class="file-icon"><IconFile :size="20" /></div>
                  </template>
                  <div class="file-info">
                    <div class="file-name">{{ itemName(item) }}</div>
                    <div class="file-meta">{{ itemMeta(item) }}</div>
                  </div>
                  <div v-if="item.isImage" class="file-actions">
                    <button
                      type="button"
                      class="file-action-btn"
                      title="查看"
                      @click="openViewer(idx)"
                    >
                      <IconEye :size="13" />
                      <span>查看</span>
                    </button>
                    <button
                      v-if="item.kind === 'file'"
                      type="button"
                      class="file-action-btn"
                      title="编辑"
                      @click="openEditor(idx)"
                    >
                      <IconEdit :size="13" />
                      <span>编辑</span>
                    </button>
                  </div>
                  <button type="button" class="file-remove" @click="removeFile(idx)">
                    <IconClose :size="14" />
                  </button>
                </div>
              </div>

              <a-textarea
                v-model="promptText"
                :auto-size="{ minRows: 3, maxRows: 5 }"
                placeholder="输入创作需求，使用 #标签 添加关键词，例如：写一篇小红书文案 #穿搭 #夏季"
                class="chat-textarea"
                @keydown="handleTextareaKeydown"
                @paste="handleTextareaPaste"
              />

              <div class="chat-toolbar">
                <div class="toolbar-left">
                  <button
                    type="button"
                    class="toolbar-btn"
                    title="上传文件"
                    @click="triggerUpload('all')"
                  >
                    <IconFolderAdd :size="18" />
                  </button>
                  <button
                    type="button"
                    class="toolbar-btn"
                    title="图片"
                    @click="triggerUpload('image')"
                  >
                    <IconImage :size="18" />
                  </button>
                  <button
                    type="button"
                    class="toolbar-btn"
                    title="视频"
                    @click="triggerUpload('video')"
                  >
                    <IconVideoCamera :size="18" />
                  </button>
                  <span class="chat-shortcut-hint">{{ shortcutHint }}</span>
                </div>
                <div class="toolbar-right">
                  <span
                    class="model-select-wrap"
                    :class="{ 'model-select-wrap--compact': modelCompact }"
                  >
                    <a-select
                      :model-value="activeModelKey"
                      :placeholder="modelOptions.length ? '选择模型' : '无可用模型'"
                      size="small"
                      class="chat-model-select"
                      @change="onModelChange"
                    >
                      <a-option v-for="opt in modelOptions" :key="opt.key" :value="opt.key">
                        <span class="provider-opt">
                          <span class="provider-opt__name">
                            <span class="provider-opt__bracket">【</span>{{ opt.planName
                            }}<span class="provider-opt__bracket">】</span>
                          </span>
                          <span class="provider-opt__model">{{ opt.modelId }}</span>
                        </span>
                      </a-option>
                    </a-select>
                    <IconRobot class="model-select-icon" :size="18" />
                  </span>
                  <button
                    type="button"
                    class="send-btn"
                    :disabled="!canGenerate || isGenerating"
                    @click="generate"
                  >
                    <IconArrowUp v-if="!isGenerating" :size="18" />
                    <IconLoading v-else :size="16" spin />
                  </button>
                </div>
              </div>
            </div>

            <input
              ref="fileInputRef"
              type="file"
              :accept="acceptType"
              multiple
              class="hidden-file-input"
              @change="handleNativeFileChange"
            />
          </div>
        </div>
      </div>
    </div>

    <ImageViewerModal
      :visible="viewerVisible"
      :items="viewerItems"
      :index="viewerIndex"
      @close="viewerVisible = false"
      @prev="viewerPrev"
      @next="viewerNext"
    />

    <ImageEditorModal
      :visible="editorVisible"
      :url="editorTarget?.url || ''"
      :name="editorTarget?.name || ''"
      @close="closeEditor"
      @confirm="onEditorConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch, defineAsyncComponent } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconCopy,
  IconSave,
  IconSettings,
  IconCode,
  IconDelete,
  IconImage,
  IconVideoCamera,
  IconFile,
  IconClose,
  IconArrowUp,
  IconLoading,
  IconFolderAdd,
  IconStar,
  IconRobot,
  IconEye,
  IconEdit,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import ImageViewerModal from '@/components/ImageViewerModal.vue'
import { useTokenPlanStore, parseModelField } from '@/stores/tokenPlan'
import api from '@/utils/api'

const ImageEditorModal = defineAsyncComponent(() => import('@/components/ImageEditorModal.vue'))

const router = useRouter()
const store = useTokenPlanStore()

onMounted(() => {
  if (!store.loaded) {
    store.loadPlans()
  }
})

type LogLevel = 'info' | 'req' | 'ok' | 'err'

interface LogEntry {
  id: string
  time: string
  level: LogLevel
  message: string
}

const logs = ref<LogEntry[]>([])
const logPanelRef = ref<HTMLElement | null>(null)
const streamingTextRef = ref<HTMLElement | null>(null)
let logSeq = 0

function pushLog(level: LogLevel, message: string) {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  logs.value.push({
    id: String(++logSeq),
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

const promptText = ref('')
const selectedPlatforms = ref<string[]>(['xiaohongshu'])
const isGenerating = ref(false)
const streamingText = ref('')

interface UploadedItemBase {
  name: string
  isImage: boolean
}

interface UploadedFileItem extends UploadedItemBase {
  kind: 'file'
  file: File
}

interface UploadedLinkItem extends UploadedItemBase {
  kind: 'link'
  url: string
}

type UploadedItem = UploadedFileItem | UploadedLinkItem

const uploadedFiles = ref<UploadedItem[]>([])
const isDragOver = ref(false)
let dragDepth = 0

function onDragEnter() {
  dragDepth++
  isDragOver.value = true
}

function onDragOver() {
  isDragOver.value = true
}

function onDragLeave() {
  dragDepth--
  if (dragDepth <= 0) {
    dragDepth = 0
    isDragOver.value = false
  }
}

function onDrop(e: DragEvent) {
  dragDepth = 0
  isDragOver.value = false
  const files = Array.from(e.dataTransfer?.files || [])
  if (files.length > 0) {
    addFiles(files)
  }
}
const streamingPlatform = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)
const acceptType = ref('image/*,video/*')
const inputCardRef = ref<HTMLElement | null>(null)
const modelCompact = ref(false)
const MODEL_COMPACT_THRESHOLD = 440

let cardResizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (inputCardRef.value) {
    cardResizeObserver = new ResizeObserver((entries) => {
      const width = entries[0]?.contentRect.width ?? 0
      modelCompact.value = width < MODEL_COMPACT_THRESHOLD
    })
    cardResizeObserver.observe(inputCardRef.value)
  }
})

onUnmounted(() => {
  cardResizeObserver?.disconnect()
  cardResizeObserver = null
})

const parsedPrompt = computed(() => {
  const text = promptText.value
  const keywords: string[] = []
  const topic = text
    .replace(/#[\p{L}\p{N}_\u4e00-\u9fa5]+/gu, (match) => {
      keywords.push(match.slice(1))
      return ''
    })
    .trim()
  return { topic, keywords }
})

const canGenerate = computed(
  () =>
    !!parsedPrompt.value.topic ||
    parsedPrompt.value.keywords.length > 0 ||
    uploadedFiles.value.length > 0,
)

function onModelChange(val: unknown) {
  if (typeof val !== 'string') return
  const idx = val.indexOf(':')
  if (idx <= 0) return
  store.selectModel(val.slice(0, idx), val.slice(idx + 1))
}

interface ModelOption {
  key: string
  planName: string
  modelId: string
}

const modelOptions = computed<ModelOption[]>(() => {
  const options: ModelOption[] = []
  for (const p of store.enabledPlans) {
    const planName = p.displayName || p.name
    for (const m of parseModelField(p.model)) {
      if (m.id) options.push({ key: `${p.id}:${m.id}`, planName, modelId: m.id })
    }
  }
  return options
})

const activeModelKey = computed(() => {
  if (!store.activePlan) return ''
  const modelId = store.selectedModelId || store.activeModelList[0]?.id || ''
  return modelId ? `${store.activePlanId}:${modelId}` : ''
})

function togglePlatform(value: string) {
  const idx = selectedPlatforms.value.indexOf(value)
  if (idx > -1) {
    if (selectedPlatforms.value.length > 1) {
      selectedPlatforms.value.splice(idx, 1)
    }
  } else {
    selectedPlatforms.value.push(value)
  }
}

function triggerUpload(type: 'all' | 'image' | 'video') {
  acceptType.value = type === 'image' ? 'image/*' : type === 'video' ? 'video/*' : 'image/*,video/*'
  nextTick(() => {
    fileInputRef.value?.click()
  })
}

function handleNativeFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  addFiles(files)
  input.value = ''
}

function addFiles(files: File[]) {
  const valid: UploadedItem[] = []
  for (const file of files) {
    if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) {
      Message.warning(`不支持的文件类型：${file.name}，仅支持图片和视频`)
      continue
    }
    if (file.type.startsWith('video/') && file.size > 50 * 1024 * 1024) {
      Message.warning(`视频文件过大：${file.name}（最大 50MB）`)
      continue
    }
    valid.push({
      kind: 'file',
      file,
      name: file.name,
      isImage: file.type.startsWith('image/'),
    })
  }
  uploadedFiles.value.push(...valid)
  if (uploadedFiles.value.length > 10) {
    uploadedFiles.value = uploadedFiles.value.slice(0, 10)
    Message.warning('最多上传 10 个文件')
  }
}

function removeFile(idx: number) {
  uploadedFiles.value.splice(idx, 1)
}

const viewerVisible = ref(false)
const viewerIndex = ref(0)
const viewerItems = ref<{ url: string; name: string }[]>([])

function openViewer(idx: number) {
  const target = uploadedFiles.value[idx]
  if (!target || !target.isImage) return
  const imageList = uploadedFiles.value.filter((i) => i.isImage)
  viewerItems.value = imageList.map((i) => ({
    url: i.kind === 'file' ? filePreview(i.file) : i.url,
    name: i.name,
  }))
  viewerIndex.value = imageList.indexOf(target)
  viewerVisible.value = true
}

function viewerPrev() {
  if (viewerIndex.value > 0) viewerIndex.value--
}

function viewerNext() {
  if (viewerIndex.value < viewerItems.value.length - 1) viewerIndex.value++
}

const editorVisible = ref(false)
const editorTarget = ref<{ index: number; url: string; name: string } | null>(null)

function openEditor(idx: number) {
  const item = uploadedFiles.value[idx]
  if (!item || item.kind !== 'file' || !item.isImage) return
  editorTarget.value = {
    index: idx,
    url: filePreview(item.file),
    name: item.name,
  }
  editorVisible.value = true
}

function closeEditor() {
  editorVisible.value = false
  editorTarget.value = null
}

function onEditorConfirm(result: { blob: Blob; name: string }) {
  const t = editorTarget.value
  if (t) {
    const item = uploadedFiles.value[t.index]
    if (item && item.kind === 'file') {
      item.file = new File([result.blob], result.name, { type: 'image/jpeg' })
      item.name = result.name
    }
  }
  closeEditor()
}

const FILE_URL_PATTERN = /https?:\/\/[^\s<>"'（）()，。；！？、]+/gi
const IMAGE_URL_PATTERN = /\.(png|jpe?g|gif|webp|avif|svg|bmp|ico)(\?.*)?$/i
const FILE_URL_EXT_PATTERN =
  /\.(png|jpe?g|gif|webp|avif|svg|bmp|ico|mp4|webm|mov|m4v|avi|mkv|flv|wmv|pdf|zip|rar|7z|tar|gz|mp3|wav|flac|aac|ogg|docx?|xlsx?|pptx?|txt)(\?.*)?$/i

function extractFileLinks() {
  const text = promptText.value
  const urls = Array.from(new Set(text.match(FILE_URL_PATTERN) || []))
    .map((u) => u.replace(/[.,;:!?，。；：！？、]+$/, ''))
    .filter((u) => FILE_URL_EXT_PATTERN.test(u))
  if (urls.length === 0) return
  const existing = new Set(uploadedFiles.value.filter((i) => i.kind === 'link').map((i) => i.url))
  for (const url of urls) {
    if (existing.has(url)) continue
    const path = url.split(/[?#]/)[0]
    let name = path.split('/').pop() || url
    try {
      name = decodeURIComponent(name)
    } catch {
      /* 保留原始名称 */
    }
    uploadedFiles.value.push({
      kind: 'link',
      url,
      name,
      isImage: IMAGE_URL_PATTERN.test(url),
    })
    existing.add(url)
  }
  if (uploadedFiles.value.length > 10) {
    uploadedFiles.value = uploadedFiles.value.slice(0, 10)
  }
  const cleaned = text.replace(FILE_URL_PATTERN, (match) => {
    const url = match.replace(/[.,;:!?，。；：！？、]+$/, '')
    return FILE_URL_EXT_PATTERN.test(url) ? '' : match
  })
  promptText.value = cleaned.replace(/ {2,}/g, ' ').replace(/[ \t]+(?=[，。；：！？、])/g, '')
}

watch(
  promptText,
  () => {
    extractFileLinks()
  },
  { immediate: true },
)

const isMac = /Mac|iPhone|iPad|iPod/i.test(navigator.userAgent)
const shortcutHint = computed(() => (isMac ? '⌘+Enter 换行' : 'Ctrl+Enter 换行'))

function handleTextareaKeydown(e: KeyboardEvent) {
  if (e.key !== 'Enter') return
  if (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) {
    e.preventDefault()
    const target = e.target as HTMLTextAreaElement
    const start = target.selectionStart
    const end = target.selectionEnd
    target.value = target.value.slice(0, start) + '\n' + target.value.slice(end)
    target.selectionStart = target.selectionEnd = start + 1
    target.dispatchEvent(new Event('input', { bubbles: true }))
    requestAnimationFrame(() => scrollCaretIntoView(target))
    return
  }
  e.preventDefault()
  generate()
}

function scrollCaretIntoView(target: HTMLTextAreaElement) {
  const pos = target.selectionStart
  if (pos == null) return
  const cs = getComputedStyle(target)
  const mirror = document.createElement('div')
  mirror.style.position = 'absolute'
  mirror.style.left = '-9999px'
  mirror.style.top = '0'
  mirror.style.visibility = 'hidden'
  mirror.style.pointerEvents = 'none'
  mirror.style.whiteSpace = 'pre-wrap'
  mirror.style.overflowWrap = 'break-word'
  mirror.style.boxSizing = cs.boxSizing
  mirror.style.width = cs.width
  mirror.style.fontFamily = cs.fontFamily
  mirror.style.fontSize = cs.fontSize
  mirror.style.fontWeight = cs.fontWeight
  mirror.style.fontStyle = cs.fontStyle
  mirror.style.lineHeight = cs.lineHeight
  mirror.style.letterSpacing = cs.letterSpacing
  mirror.style.wordSpacing = cs.wordSpacing
  mirror.style.paddingTop = cs.paddingTop
  mirror.style.paddingRight = cs.paddingRight
  mirror.style.paddingBottom = cs.paddingBottom
  mirror.style.paddingLeft = cs.paddingLeft
  mirror.style.borderTopWidth = cs.borderTopWidth
  mirror.style.borderRightWidth = cs.borderRightWidth
  mirror.style.borderBottomWidth = cs.borderBottomWidth
  mirror.style.borderLeftWidth = cs.borderLeftWidth
  const before = document.createElement('span')
  before.textContent = target.value.slice(0, pos)
  const after = document.createElement('span')
  after.textContent = target.value.slice(pos) || ' '
  mirror.appendChild(before)
  mirror.appendChild(after)
  document.body.appendChild(mirror)
  const caretTop = after.getBoundingClientRect().top - mirror.getBoundingClientRect().top
  document.body.removeChild(mirror)

  const lineHeight = parseFloat(cs.lineHeight) || 20
  const visibleTop = target.scrollTop
  const visibleBottom = visibleTop + target.clientHeight
  if (caretTop < visibleTop + lineHeight) {
    target.scrollTop = Math.max(0, caretTop - lineHeight)
  } else if (caretTop + lineHeight > visibleBottom) {
    target.scrollTop = caretTop + lineHeight - target.clientHeight
  }
}

function handleTextareaPaste(e: ClipboardEvent) {
  const files = Array.from(e.clipboardData?.files || [])
  if (files.length > 0) {
    e.preventDefault()
    addFiles(files)
  }
}

function itemName(item: UploadedItem): string {
  return item.name
}

function itemMeta(item: UploadedItem): string {
  if (item.kind === 'link') return item.isImage ? '图片链接' : '文件链接'
  return `${fileExt(item.file!)} · ${fileSize(item.file!)}`
}

function fileExt(file: File): string {
  const parts = file.name.split('.')
  return parts.length > 1 ? parts.pop()!.toUpperCase() : 'FILE'
}

function fileSize(file: File): string {
  if (file.size < 1024) return file.size + ' B'
  if (file.size < 1024 * 1024) return (file.size / 1024).toFixed(1) + ' KB'
  return (file.size / (1024 * 1024)).toFixed(1) + ' MB'
}

function filePreview(file: File): string {
  return URL.createObjectURL(file)
}

function isImage(file: File): boolean {
  return file.type.startsWith('image/')
}

function compressImage(file: File, maxWidth = 1920, quality = 0.8): Promise<File> {
  return new Promise((resolve, _reject) => {
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

async function itemToBase64(
  item: UploadedItem,
): Promise<{ data: string; mime_type: string } | null> {
  try {
    if (item.kind === 'file' && item.file) {
      return await fileToBase64(item.file)
    }
    if (item.kind === 'link' && item.url) {
      const res = await fetch(item.url)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const blob = await res.blob()
      const file = new File([blob], item.name || 'file', {
        type: blob.type || 'application/octet-stream',
      })
      return await fileToBase64(file)
    }
  } catch (err) {
    const e = err as { message?: string }
    pushLog('err', `链接素材获取失败：${item.name}（${e.message || '网络错误'}）`)
  }
  return null
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
  if (!canGenerate.value) {
    pushLog('err', '内容主题、关键词、上传文件至少填写一项')
    Message.error('内容主题、关键词、上传文件至少填写一项')
    return
  }
  if (uploadedFiles.value.length > 0 && !store.selectedModelSupportsFiles) {
    pushLog('err', '当前模型不支持文件上传，请切换到支持视觉/图片/视频的模型')
    Message.error('当前模型不支持文件上传，请切换到支持视觉/图片/视频的模型')
    return
  }

  isGenerating.value = true
  hasGenerated.value = false
  streamingText.value = ''
  streamingPlatform.value = ''

  const startedAt = performance.now()
  const plan = store.activePlan
  const modelId = store.selectedModelId || store.activeModelList[0]?.id || ''
  const { topic, keywords } = parsedPrompt.value
  const inputDesc = topic
    ? `主题「${topic}」`
    : keywords.length > 0
      ? `关键词「${keywords.join('、')}」`
      : `上传素材 ${uploadedFiles.value.length} 个`
  pushLog(
    'info',
    `开始生成任务 · ${inputDesc} · 平台 ${selectedPlatforms.value.map(platformLabel).join(' / ')}`,
  )
  pushLog('info', `使用模型配置 ${plan.name}（${modelId}）`)

  const keywordsArray = keywords.length > 0 ? keywords : undefined

  let filesPayload: { data: string; mime_type: string }[] | undefined
  if (uploadedFiles.value.length > 0) {
    pushLog('info', `正在编码 ${uploadedFiles.value.length} 个素材文件…`)
    const results = await Promise.all(uploadedFiles.value.map(itemToBase64))
    filesPayload = results.filter((r): r is { data: string; mime_type: string } => r !== null)
    if (filesPayload.length > 0) {
      pushLog('ok', `素材编码完成，共 ${filesPayload.length} 个`)
    } else {
      pushLog('err', '所有素材均无法编码，本次生成将不带文件')
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
        topic: topic,
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
.provider-opt {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  white-space: nowrap;
  max-width: 100%;
}
.provider-opt__name {
  font-weight: 500;
}
.provider-opt__bracket {
  color: #007aff;
  font-weight: 600;
}
.provider-opt__model {
  font-size: 12px;
  color: #86909c;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.content-layout {
  display: flex;
  flex-direction: row;
  gap: 12px;
  align-items: stretch;
  height: 760px;
}

.content-left {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  order: 1;
  background: #1d1d1f;
  border-radius: 16px;
  padding: 12px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.28),
    0 2px 8px rgba(0, 0, 0, 0.2);
}

.content-left .log-terminal {
  flex: none;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.content-left .log-terminal__body-wrap {
  flex: none;
  min-height: 0;
  overflow: hidden;
}

.content-left .log-terminal__body {
  max-height: 160px;
  height: 160px;
  min-height: 0;
  overflow-y: auto;
}

.content-left .chat-input-section {
  flex-shrink: 0;
}

.content-preview-card {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  order: 2;
}

.content-preview-card :deep(.arco-card-body) {
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
  user-select: none;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
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
  .content-layout {
    flex-direction: column;
    height: auto;
  }
  .content-left .log-terminal__body {
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
  .content-left .log-terminal__body {
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
  .content-left .log-terminal__body {
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

.chat-input-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: auto;
}

.platform-select-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.platform-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 18px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: #2c2c2e;
  color: #d1d1d6;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.platform-chip:hover {
  background: #3a3a3c;
  color: #ffffff;
}

.platform-chip--active {
  background: #007aff;
  color: #ffffff;
  border-color: #007aff;
}

.chat-input-card {
  background: #2c2c2e;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  padding: 14px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.2);
  transition:
    border-color 0.15s ease,
    box-shadow 0.15s ease;
}

.chat-input-card--drag {
  border-color: rgba(0, 122, 255, 0.65);
  box-shadow:
    0 0 0 3px rgba(0, 122, 255, 0.18),
    0 2px 12px rgba(0, 0, 0, 0.2);
}

.chat-textarea {
  border-radius: 10px;
  border: 1px solid transparent;
  background: #2c2c2e;
  &:hover {
    background: #2c2c2e;
    border-color: transparent;
  }
  &:focus-within {
    background: #2c2c2e;
    border-color: transparent;
    box-shadow: none;
  }
  :deep(.arco-textarea) {
    border: none;
    background: transparent;
    padding: 4px 0;
    font-size: 15px;
    line-height: 1.65;
    resize: none;
    box-shadow: none;
    color: #f2f2f7;
    &::-webkit-scrollbar {
      width: 6px;
      height: 6px;
    }
    &::-webkit-scrollbar-track {
      background: transparent;
    }
    &::-webkit-scrollbar-thumb {
      background: rgba(255, 255, 255, 0.25);
      border-radius: 3px;
    }
    &::-webkit-scrollbar-thumb:hover {
      background: rgba(255, 255, 255, 0.4);
    }
    &::placeholder {
      color: #8e8e93;
    }
  }
  :deep(.arco-textarea:focus) {
    box-shadow: none;
  }
}

.uploaded-files {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 10px;
}

.file-chip {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  background: #3a3a3c;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  max-width: 240px;
}

.file-thumb {
  width: 36px;
  height: 36px;
  object-fit: cover;
  border-radius: 6px;
  flex-shrink: 0;
}

.file-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #48484a;
  border-radius: 6px;
  color: #4098ff;
  flex-shrink: 0;
}

.file-info {
  min-width: 0;
  flex: 1;
}

.file-name {
  font-size: 13px;
  font-weight: 500;
  color: #f2f2f7;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-meta {
  font-size: 11px;
  color: #8e8e93;
  margin-top: 2px;
}

.file-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  color: #aeaeb2;
  cursor: pointer;
  flex-shrink: 0;
}

.file-remove:hover {
  background: rgba(255, 255, 255, 0.18);
  color: #ffffff;
}

.file-actions {
  position: absolute;
  left: 4px;
  right: 30px;
  bottom: 4px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.15s;
}

.file-chip:hover .file-actions {
  opacity: 1;
}

.file-action-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  padding: 3px 6px;
  border: none;
  border-radius: 5px;
  background: rgba(0, 0, 0, 0.62);
  color: #ffffff;
  font-size: 11px;
  line-height: 1.4;
  cursor: pointer;
  backdrop-filter: blur(4px);
  transition: background 0.15s;
}

.file-action-btn:hover {
  background: rgba(0, 122, 255, 0.85);
}

.chat-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.toolbar-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #aeaeb2;
  cursor: pointer;
  transition: all 0.15s ease;
}

.toolbar-btn:hover {
  background: #3a3a3c;
  color: #ffffff;
}

.chat-shortcut-hint {
  margin-left: 8px;
  flex-shrink: 0;
  font-size: 11px;
  color: #7a7a80;
  white-space: nowrap;
  user-select: none;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.toolbar-right {
  :deep(.chat-model-select) {
    flex: 0 1 auto;
    width: auto;
    min-width: 0;
    max-width: 240px;
    border-radius: 14px;
    background: #2c2c2e;
    border-color: transparent;
    color: #f2f2f7;
    &:hover,
    &:focus-within,
    &.arco-select-view--focus {
      background: #3a3a3c;
      border-color: rgba(0, 122, 255, 0.45);
    }
    .arco-select-view-value {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      min-width: 0;
    }
    .provider-opt__model {
      color: #aeaeb2;
    }
  }
}

.model-select-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  flex: 0 1 auto;
  min-width: 0;
}

.model-select-icon {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  color: #aeaeb2;
  pointer-events: none;
  display: none;
}

.model-select-wrap--compact .model-select-icon {
  display: block;
}

.model-select-wrap--compact :deep(.chat-model-select) {
  width: 32px;
  min-width: 32px;
  max-width: 32px;
  flex: none;
  padding: 0;
  height: 28px;
  justify-content: center;
}

.model-select-wrap--compact :deep(.chat-model-select .arco-select-view-value),
.model-select-wrap--compact :deep(.chat-model-select .arco-select-view-suffix) {
  display: none;
}

:global(.arco-select-dropdown:has(.provider-opt)) {
  min-width: 340px;
  width: max-content !important;
  max-width: min(560px, 92vw);
  background: #2c2c2e;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.5),
    0 2px 8px rgba(0, 0, 0, 0.3);
  border-radius: 12px;
}

:global(.arco-select-dropdown:has(.provider-opt) .arco-select-option) {
  background: #2c2c2e;
  color: #f2f2f7;
}

:global(.arco-select-dropdown:has(.provider-opt) .arco-select-option:hover) {
  background: #3a3a3c;
  color: #ffffff;
}

:global(.arco-select-dropdown:has(.provider-opt) .arco-select-option-selected),
:global(.arco-select-dropdown:has(.provider-opt) .arco-select-option-active) {
  background: #3a3a3c;
  color: #ffffff;
}

:global(.arco-select-dropdown:has(.provider-opt) .provider-opt__bracket) {
  color: #4098ff;
}

:global(.arco-select-dropdown:has(.provider-opt) .provider-opt__model) {
  color: #aeaeb2;
}

.send-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 50%;
  background: #007aff;
  color: #ffffff;
  cursor: pointer;
  transition: all 0.15s ease;
}

.send-btn:hover:not(:disabled) {
  background: #0062cc;
}

.send-btn:disabled {
  background: #48484a;
  color: #8e8e93;
  cursor: not-allowed;
}

.hidden-file-input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
</style>
