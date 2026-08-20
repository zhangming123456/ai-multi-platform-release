<template>
  <div
    class="aia-root"
    :class="{
      'aia-root--dragging': dragging,
      'aia-root--disabled': disabled,
      'aia-root--borderless': !bordered,
      'aia-root--compact': compact,
      'aia-root--dark': theme === 'dark',
    }"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <slot name="title" />

    <div v-if="showImages" class="aia-preview">
      <div v-if="displayItems.length > 0" class="aia-cards-wrap">
        <div ref="cardsRef" class="aia-cards" @scroll="updateScrollState">
          <div
            v-for="(item, index) in displayItems"
            :key="item.key"
            class="aia-card"
            :class="{
              'aia-card--pending': item.pending,
              'aia-card--uploading': isUploading(item.key),
            }"
            @click="openViewer(index)"
          >
            <div class="aia-card__thumb">
              <a-image
                v-if="item.type === 'image'"
                :src="item.pending ? previewUrl(item) : item.url"
                :alt="item.name"
                fit="cover"
                :width="thumbSize"
                :height="thumbSize"
                :preview-props="{ srcList: imagePreviewSrcs, index: imageIndex(item.key) }"
              />
              <video
                v-else-if="item.type === 'video'"
                :src="item.pending ? previewUrl(item) : item.url"
                muted
                playsinline
                preload="metadata"
              />
              <div v-else class="aia-card__file-icon">
                <IconFile :size="22" />
              </div>
              <div v-if="item.pending || isUploading(item.key)" class="aia-card__spinner">
                <a-spin :size="16" />
              </div>
            </div>
            <div class="aia-card__info">
              <span class="aia-card__name" :title="item.name">{{ item.name }}</span>
              <span class="aia-card__meta">{{ cardMeta(item) }}</span>
            </div>
            <div v-if="!disabled" class="aia-card__actions">
              <a-tooltip v-if="item.type === 'image' && (item.pending || !!upload)" content="编辑">
                <button type="button" class="aia-card__action" @click.stop="openEditor(index)">
                  <IconEdit :size="12" />
                </button>
              </a-tooltip>
            </div>
            <a-tooltip v-if="!disabled" content="删除">
              <button type="button" class="aia-card__delete" @click.stop="removeItem(index)">
                <IconClose :size="12" />
              </button>
            </a-tooltip>
          </div>
        </div>
        <button
          v-if="canScrollLeft"
          type="button"
          class="aia-cards__arrow aia-cards__arrow--left"
          @click="scrollCards(-1)"
        >
          <IconLeft :size="14" />
        </button>
        <button
          v-if="canScrollRight"
          type="button"
          class="aia-cards__arrow aia-cards__arrow--right"
          @click="scrollCards(1)"
        >
          <IconRight :size="14" />
        </button>
      </div>
      <slot v-else name="empty" />
    </div>

    <a-textarea
      v-if="showTextarea"
      :model-value="text"
      :auto-size="{ minRows, maxRows }"
      :max-length="maxLength"
      :show-word-limit="maxLength > 0 && showWordLimit"
      :disabled="disabled"
      :placeholder="placeholder || DEFAULT_PLACEHOLDER"
      class="aia-textarea"
      @input="handleTextInput"
      @paste="handleTextPaste"
      @keydown="handleTextKeydown"
    />

    <div class="aia-toolbar">
      <div class="aia-toolbar-left">
        <a-tooltip>
          <template #content>
            <p style="max-width: 200px">
              <span>{{ uploadBtnTooltip }}</span>
              <span v-if="maxCount > 0"> ({{ displayItems.length }}/{{ maxCount }}) </span>
              <span v-if="hintText">, {{ hintText }}</span>
            </p>
          </template>
          <span class="aia-toolbar-btn-wrap">
            <button
              type="button"
              class="aia-toolbar-btn"
              :disabled="uploadBtnDisabled"
              @click="pickFiles"
            >
              <component :is="toolbarIcon" :size="16" />
            </button>
          </span>
        </a-tooltip>
        <slot name="toolbar-left" />
      </div>
      <div class="aia-toolbar-right">
        <slot name="toolbar-right" />
      </div>
    </div>

    <input ref="fileInputRef" type="file" :accept="accept" multiple hidden @change="onFileChange" />
    <ImageEditorModal
      :visible="editorVisible"
      :url="editorUrl"
      :name="editorName"
      @close="editorVisible = false"
      @confirm="handleEditorConfirm"
    />
    <a-modal
      v-model:visible="videoVisible"
      :footer="false"
      :width="680"
      :unmount-on-close="true"
      class="aia-video-modal"
    >
      <video v-if="videoVisible" :src="videoUrl" controls autoplay class="aia-video-player" />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import {
  ref,
  computed,
  reactive,
  nextTick,
  defineAsyncComponent,
  onMounted,
  onBeforeUnmount,
  watch,
} from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconPlus,
  IconImage,
  IconVideoCamera,
  IconFile,
  IconEdit,
  IconClose,
  IconLeft,
  IconRight,
} from '@arco-design/web-vue/es/icon'
import { compressImage, validateImageFile } from '@/composables/useFileUpload'
import {
  typeFromUrl,
  extractUrlsFromText,
  cleanUrlsFromText,
  urlFileName,
} from '@/composables/useUrlExtractor'

const ImageEditorModal = defineAsyncComponent(() => import('@/components/ImageEditorModal.vue'))

type AttachmentFileType = 'image' | 'video' | 'file'

interface PendingItem {
  key: string
  file: File
  type: AttachmentFileType
  name: string
  size: number
}

interface DisplayItem {
  key: string
  type: AttachmentFileType
  url?: string
  file?: File
  pending: boolean
  name: string
  size: number
}

const DEFAULT_PLACEHOLDER = '填写内容...支持拖拽 / 粘贴文件，或粘贴文件链接自动识别'

const ACCEPT_BY_TYPE: Record<AttachmentFileType, string> = {
  image: 'image/jpeg,image/png,image/webp',
  video: 'video/*',
  file: 'application/pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.csv,.zip,.rar,.7z',
}

const MAX_SIZE_BY_TYPE: Record<AttachmentFileType, number> = {
  image: 10 * 1024 * 1024,
  video: 50 * 1024 * 1024,
  file: 20 * 1024 * 1024,
}

const props = withDefaults(
  defineProps<{
    modelValue: string
    fileList?: string[]
    upload?: (file: File, onProgress?: (percent: number) => void) => Promise<string>
    uploadMode?: 'auto' | 'manual'
    fileTypes?: AttachmentFileType[]
    maxCount?: number
    disabled?: boolean
    bordered?: boolean
    compact?: boolean
    placeholder?: string
    hint?: string
    maxLength?: number
    showWordLimit?: boolean
    minRows?: number
    maxRows?: number
    showTextarea?: boolean
    showImages?: boolean
    extractUrls?: boolean
    accept?: string
    enterBehavior?: 'newline' | 'send'
    theme?: 'light' | 'dark'
  }>(),
  {
    fileList: undefined,
    uploadMode: 'manual',
    fileTypes: () => ['image'],
    maxCount: 0,
    disabled: false,
    bordered: true,
    compact: false,
    placeholder: DEFAULT_PLACEHOLDER,
    hint: '',
    maxLength: 0,
    showWordLimit: true,
    minRows: 2,
    maxRows: 5,
    showTextarea: true,
    showImages: true,
    extractUrls: true,
    enterBehavior: 'newline',
    theme: 'light',
    upload: undefined,
    accept: undefined,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'update:fileList', value: string[]): void
  (e: 'enter'): void
  (e: 'change', payload: { total: number; pending: number }): void
}>()

const fileInputRef = ref<HTMLInputElement>()
const uploading = ref(false)
const dragging = ref(false)
let dragDepth = 0

const editorVisible = ref(false)
const editorIndex = ref(0)
const editorUrl = ref('')
const editorName = ref('')
const videoVisible = ref(false)
const videoUrl = ref('')

const pendingItems = ref<PendingItem[]>([])
const objectUrlMap = new Map<File, string>()
const uploadingKeys = ref<Set<string>>(new Set())
const urlSizeMap = new Map<string, number>()
const progressMap = reactive(new Map<string, number>())

const cardsRef = ref<HTMLDivElement>()
const canScrollLeft = ref(false)
const canScrollRight = ref(false)

const fileList = computed<string[]>({
  get: () => props.fileList ?? [],
  set: (value) => emit('update:fileList', value),
})

const text = computed(() => props.modelValue ?? '')

function dataUrlType(url: string): AttachmentFileType | null {
  if (url.startsWith('data:image/')) return 'image'
  if (url.startsWith('data:video/')) return 'video'
  return null
}

const displayItems = computed<DisplayItem[]>(() => {
  const result: DisplayItem[] = []
  fileList.value.forEach((url, i) => {
    result.push({
      key: `u-${i}`,
      type: dataUrlType(url) || typeFromUrl(url) || 'file',
      url,
      pending: false,
      name: url.startsWith('data:') ? '本地图片' : urlFileName(url),
      size: urlSizeMap.get(url) || 0,
    })
  })
  pendingItems.value.forEach((p) => {
    result.push({
      key: p.key,
      type: p.type,
      file: p.file,
      pending: true,
      name: p.name,
      size: p.size,
    })
  })
  return result
})

const remaining = computed(() =>
  props.maxCount > 0 ? Math.max(0, props.maxCount - displayItems.value.length) : Infinity,
)

const imageItems = computed<DisplayItem[]>(() =>
  displayItems.value.filter((i) => i.type === 'image'),
)

const imagePreviewSrcs = computed<string[]>(() =>
  imageItems.value.map((i) => (i.pending ? previewUrl(i) : i.url || '')),
)

function imageIndex(key: string): number {
  return imageItems.value.findIndex((i) => i.key === key)
}

const thumbSize = computed(() => (props.compact ? 38 : 44))

const toolbarIcon = computed(() => {
  if (props.fileTypes.length === 1 && props.fileTypes[0] === 'image') return IconImage
  if (props.fileTypes.length === 1 && props.fileTypes[0] === 'video') return IconVideoCamera
  return IconPlus
})

const uploadBtnTooltip = computed(() => {
  if (props.uploadMode === 'auto' && !props.upload) return '未配置上传'
  if (props.fileTypes.length === 1 && props.fileTypes[0] === 'image') return '上传图片'
  if (props.fileTypes.length === 1 && props.fileTypes[0] === 'video') return '上传视频'
  return '上传文件'
})

const uploadBtnDisabled = computed(
  () =>
    props.disabled ||
    (props.uploadMode === 'auto' && !props.upload) ||
    remaining.value <= 0 ||
    uploading.value,
)

const fileTypeLabel = computed(() => {
  const t = props.fileTypes
  if (t.length === 1 && t[0] === 'image') return '图片'
  if (t.length === 1 && t[0] === 'video') return '视频'
  return '文件'
})

const hintText = computed(() => {
  if (props.hint) return props.hint
  if (props.fileTypes.length === 1 && props.fileTypes[0] === 'image') {
    return '支持拖拽 / 粘贴图片，链接自动识别'
  }
  if (props.fileTypes.length === 1 && props.fileTypes[0] === 'video') {
    return '支持拖拽 / 粘贴视频，链接自动识别'
  }
  return '支持拖拽 / 粘贴文件，链接自动识别'
})

const accept = computed(() => {
  if (props.accept) return props.accept
  const parts = props.fileTypes.map((t) => ACCEPT_BY_TYPE[t]).filter(Boolean)
  return parts.length > 0 ? parts.join(',') : '*/*'
})

function emitChange() {
  emit('change', { total: displayItems.value.length, pending: pendingItems.value.length })
}

function formatSize(size: number): string {
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)}MB`
  if (size >= 1024) return `${(size / 1024).toFixed(0)}KB`
  return `${size}B`
}

function genKey(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function isUploading(key: string): boolean {
  return uploadingKeys.value.has(key)
}

function fileTypeLabelText(type: AttachmentFileType): string {
  if (type === 'image') return '图片'
  if (type === 'video') return '视频'
  return '文件'
}

function extName(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return ext
}

function cardMeta(item: DisplayItem): string {
  if (isUploading(item.key)) {
    const p = progressMap.get(item.key)
    return p != null ? `上传中... ${p}%` : '上传中...'
  }
  if (item.pending) return '待上传'
  const label = extName(item.name).toUpperCase() || fileTypeLabelText(item.type)
  const size = item.size > 0 ? formatSize(item.size) : ''
  return size ? `${label} · ${size}` : label
}

function updateScrollState() {
  const el = cardsRef.value
  if (!el) return
  canScrollLeft.value = el.scrollLeft > 2
  canScrollRight.value = el.scrollLeft + el.clientWidth < el.scrollWidth - 2
}

function scrollCards(dir: number) {
  cardsRef.value?.scrollBy({ left: dir * 240, behavior: 'smooth' })
}

function typeFromFile(file: File): AttachmentFileType | null {
  const allowed = new Set(props.fileTypes)
  if (file.type.startsWith('image/') && allowed.has('image')) return 'image'
  if (file.type.startsWith('video/') && allowed.has('video')) return 'video'
  if (allowed.has('file')) return 'file'
  return null
}

function validateFileByType(file: File, type: AttachmentFileType): string | null {
  if (type === 'image') return validateImageFile(file)
  const max = MAX_SIZE_BY_TYPE[type]
  if (file.size > max) {
    const label = type === 'video' ? '视频' : '文件'
    return `${label}大小不能超过 ${Math.round(max / 1024 / 1024)}MB`
  }
  return null
}

function previewUrl(item: DisplayItem): string {
  if (!item.pending || !item.file) return item.url || ''
  let url = objectUrlMap.get(item.file)
  if (!url) {
    url = URL.createObjectURL(item.file)
    objectUrlMap.set(item.file, url)
  }
  return url
}

function pickFiles() {
  if (props.disabled) return
  if (props.uploadMode === 'auto' && !props.upload) {
    Message.warning('未配置上传')
    return
  }
  if (remaining.value <= 0) return
  fileInputRef.value?.click()
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  void addFiles(files)
}

async function addFiles(files: File[]) {
  const accepted: { file: File; type: AttachmentFileType }[] = []
  for (const f of files) {
    const t = typeFromFile(f)
    if (t) accepted.push({ file: f, type: t })
  }
  if (accepted.length === 0) {
    Message.warning('请选择允许上传的文件类型')
    return
  }
  if (props.uploadMode === 'auto' && !props.upload) {
    Message.warning('未配置上传')
    return
  }
  if (props.uploadMode === 'auto') uploading.value = true
  try {
    for (const { file, type } of accepted) {
      if (props.maxCount > 0 && displayItems.value.length >= props.maxCount) {
        Message.warning(`最多上传 ${props.maxCount} 个文件`)
        break
      }
      const error = validateFileByType(file, type)
      if (error) {
        Message.error(error)
        continue
      }
      const itemKey = genKey()
      if (props.uploadMode === 'auto' && props.upload) {
        uploadingKeys.value.add(itemKey)
        try {
          const processed = type === 'image' ? await compressImage(file) : file
          const url = await props.upload(processed, (p) => progressMap.set(itemKey, p))
          if (url) {
            urlSizeMap.set(url, processed.size)
            fileList.value = [...fileList.value, url]
          }
        } finally {
          uploadingKeys.value.delete(itemKey)
          progressMap.delete(itemKey)
        }
      } else {
        pendingItems.value = [
          ...pendingItems.value,
          { key: itemKey, file, type, name: file.name, size: file.size },
        ]
      }
    }
  } catch {
    Message.error('上传失败，请重试')
  } finally {
    if (props.uploadMode === 'auto') uploading.value = false
    emitChange()
  }
}

async function flushPending(): Promise<boolean> {
  if (pendingItems.value.length === 0) return true
  if (!props.upload) {
    Message.warning('未配置上传')
    return false
  }
  uploading.value = true
  try {
    for (const item of pendingItems.value) {
      uploadingKeys.value.add(item.key)
      try {
        const processed = item.type === 'image' ? await compressImage(item.file) : item.file
        const url = await props.upload(processed, (p) => progressMap.set(item.key, p))
        if (!url) {
          Message.error(`「${item.name}」上传失败`)
          return false
        }
        urlSizeMap.set(url, processed.size)
        fileList.value = [...fileList.value, url]
      } finally {
        uploadingKeys.value.delete(item.key)
        progressMap.delete(item.key)
      }
    }
    pendingItems.value = []
    return true
  } catch {
    Message.error('上传失败，请重试')
    return false
  } finally {
    uploading.value = false
    emitChange()
  }
}

function getPendingFiles(): PendingItem[] {
  return pendingItems.value.map((p) => ({ ...p }))
}

function removeItem(index: number) {
  const item = displayItems.value[index]
  if (!item) return
  if (item.pending) {
    const url = item.file ? objectUrlMap.get(item.file) : undefined
    if (url) {
      URL.revokeObjectURL(url)
      if (item.file) objectUrlMap.delete(item.file)
    }
    uploadingKeys.value.delete(item.key)
    progressMap.delete(item.key)
    pendingItems.value = pendingItems.value.filter((p) => p.key !== item.key)
  } else {
    if (item.url) urlSizeMap.delete(item.url)
    fileList.value = fileList.value.filter((u) => u !== item.url)
  }
  emitChange()
}

function openViewer(index: number) {
  const item = displayItems.value[index]
  if (!item) return
  if (item.type === 'video') {
    videoUrl.value = item.pending ? previewUrl(item) : item.url || ''
    videoVisible.value = true
  } else if (item.type === 'file') {
    if (item.url) window.open(item.url, '_blank')
  }
}

function openEditor(index: number) {
  const item = displayItems.value[index]
  if (!item || item.type !== 'image') return
  if (!item.pending && !props.upload) return
  editorIndex.value = index
  editorUrl.value = item.pending ? previewUrl(item) : item.url || ''
  editorName.value = item.name
  editorVisible.value = true
}

async function handleEditorConfirm(result: { blob: Blob; name: string }) {
  const item = displayItems.value[editorIndex.value]
  if (!item || item.type !== 'image') return
  const file = new File([result.blob], result.name, { type: result.blob.type || 'image/jpeg' })
  if (item.pending && item.file) {
    const i = pendingItems.value.findIndex((p) => p.file === item.file)
    if (i >= 0) {
      const next = [...pendingItems.value]
      next[i] = { ...next[i], file, name: file.name, size: file.size }
      pendingItems.value = next
    }
    Message.success('图片编辑已保存')
    emitChange()
    return
  }
  if (!props.upload) return
  uploading.value = true
  uploadingKeys.value.add(item.key)
  try {
    const url = await props.upload(file, (p) => progressMap.set(item.key, p))
    if (url) {
      const i = fileList.value.indexOf(item.url || '')
      if (i >= 0) {
        const next = [...fileList.value]
        next[i] = url
        fileList.value = next
      }
      urlSizeMap.set(url, file.size)
      Message.success('图片编辑已保存')
    }
  } finally {
    uploadingKeys.value.delete(item.key)
    progressMap.delete(item.key)
    uploading.value = false
    emitChange()
  }
}

function handleDragEnter(e: DragEvent) {
  if (props.disabled) return
  if (!e.dataTransfer?.types.includes('Files')) return
  dragDepth += 1
  dragging.value = true
}

function handleDragOver(e: DragEvent) {
  if (!e.dataTransfer?.types.includes('Files')) return
  e.preventDefault()
}

function handleDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragging.value = false
}

function handleDrop(e: DragEvent) {
  dragDepth = 0
  dragging.value = false
  if (!e.dataTransfer?.types.includes('Files')) return
  e.preventDefault()
  void addFiles(Array.from(e.dataTransfer.files))
}

function handlePaste(e: ClipboardEvent): boolean {
  if (props.disabled) return false
  const files = Array.from(e.clipboardData?.files || []).filter((f) => typeFromFile(f) !== null)
  if (files.length === 0) return false
  e.preventDefault()
  void addFiles(files)
  return true
}

function handleTextPaste(e: ClipboardEvent) {
  if (handlePaste(e)) return
}

function handleTextKeydown(e: KeyboardEvent) {
  if (e.key !== 'Enter') return
  if (props.enterBehavior === 'send') {
    if (e.ctrlKey || e.metaKey) return
    if (e.shiftKey || e.altKey) return
    e.preventDefault()
    emit('enter')
    return
  }
  if (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) return
  const target = e.target as HTMLTextAreaElement
  const before = target.value.slice(0, target.selectionStart)
  if (before.endsWith('\n\n')) {
    e.preventDefault()
  }
}

function handleTextInput(raw: unknown) {
  let value = ''
  if (typeof raw === 'string') {
    value = raw
  } else if (raw && typeof raw === 'object' && 'value' in raw) {
    value = String((raw as { value: unknown }).value ?? '')
  }
  emit('update:modelValue', value)
  if (props.extractUrls) {
    nextTick(() => processText(value))
  }
}

function processText(raw: string) {
  if (!props.extractUrls) return
  const urls = extractUrlsFromText(raw, props.fileTypes)
  const existing = new Set(fileList.value)
  const added: string[] = []
  for (const url of urls) {
    if (existing.has(url)) continue
    if (props.maxCount > 0 && displayItems.value.length >= props.maxCount) break
    fileList.value = [...fileList.value, url]
    existing.add(url)
    added.push(url)
  }
  if (added.length === 0) return
  const cleaned = cleanUrlsFromText(raw, props.fileTypes)
  if (cleaned !== raw) emit('update:modelValue', cleaned)
  Message.success(`已识别 ${added.length} 个${fileTypeLabel.value}链接并添加`)
  emitChange()
}

onMounted(() => {
  window.addEventListener('resize', updateScrollState)
  nextTick(updateScrollState)
})

watch(
  () => displayItems.value.length,
  () => nextTick(updateScrollState),
)

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateScrollState)
  objectUrlMap.forEach((u) => URL.revokeObjectURL(u))
  objectUrlMap.clear()
})

defineExpose({ handlePaste, pickFiles, addFiles, flushPending, getPendingFiles })
</script>

<style scoped lang="scss">
:deep(.arco-input).arco-input-wrapper:not(.arco-input-disabled),
:deep(.aia-textarea).arco-textarea-wrapper:not(.arco-textarea-disabled) {
  background-color: transparent !important;
  border-color: transparent !important;
}

.aia-root {
  width: 100%;
  border: 1px solid #e5e5ea;
  border-radius: 12px;
  background: #f5f5f7;
  padding: 10px 12px;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    background 0.2s ease;
}
.aia-root--dragging {
  border-color: rgb(var(--primary-6));
  background: rgb(var(--primary-1));
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.12);
}
.aia-root--disabled {
  opacity: 0.6;
}
.aia-root--borderless {
  border: none;
  background: transparent;
  padding: 0;
}
.aia-preview {
  min-height: 0;
}
.aia-cards-wrap {
  position: relative;
  margin-bottom: 10px;
}
.aia-cards {
  display: flex;
  flex-wrap: nowrap;
  gap: 12px;
  overflow-x: auto;
  padding-bottom: 2px;
  scrollbar-width: thin;
}
.aia-cards__arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 2;
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.95);
  color: #4e5969;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
  transition:
    background 0.15s ease,
    color 0.15s ease;
}
.aia-cards__arrow:hover {
  background: #fff;
  color: rgb(var(--primary-6));
}
.aia-cards__arrow--left {
  left: -2px;
}
.aia-cards__arrow--right {
  right: -2px;
}
.aia-card {
  position: relative;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
  width: 200px;
  height: 60px;
  border-radius: 8px;
  padding: 6px 8px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  background: #fff;
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    box-shadow 0.15s ease;
}
.aia-card:hover {
  background: #fafafa;
  border-color: rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}
.aia-card--pending,
.aia-card--uploading {
  border-color: rgba(0, 0, 0, 0.08);
}
.aia-card--pending .aia-card__meta,
.aia-card--uploading .aia-card__meta {
  color: rgb(var(--primary-6));
}
.aia-card__thumb {
  position: relative;
  flex: 0 0 auto;
  width: 44px;
  height: 44px;
  border-radius: 6px;
  overflow: hidden;
  background: #f2f2f7;
  display: flex;
  align-items: center;
  justify-content: center;
}
.aia-card__thumb img,
.aia-card__thumb video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.aia-card__file-icon {
  color: #86909c;
}
.aia-card__spinner {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
}
.aia-card__info {
  flex: 1;
  min-width: 0;
  padding-right: 24px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
}
.aia-card__name {
  font-size: 12px;
  font-weight: 500;
  color: #1d2129;
  line-height: 1.4;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.aia-card__meta {
  font-size: 10px;
  color: #86909c;
  line-height: 1.3;
  margin-top: 2px;
}
.aia-card__actions {
  position: absolute;
  top: 4px;
  right: 28px;
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.15s ease;
}
.aia-card:hover .aia-card__actions {
  opacity: 1;
}
.aia-card__delete {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.95);
  color: #4e5969;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
  transition:
    background 0.15s ease,
    color 0.15s ease;
}
.aia-card__delete:hover {
  background: #fff;
  color: #f53f3f;
}
.aia-card__action {
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.95);
  color: #4e5969;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
  transition:
    background 0.15s ease,
    color 0.15s ease;
}
.aia-card__action:hover {
  background: #fff;
  color: rgb(var(--primary-6));
}
.aia-textarea {
  background-color: transparent;
  border: none;
  box-shadow: none;
}
.aia-textarea :deep(.arco-textarea) {
  border: none;
  background: transparent;
  padding: 4px 0;
  font-size: 14px;
  line-height: 1.65;
  resize: none;
  box-shadow: none;
}
.aia-textarea :deep(.arco-textarea)::placeholder {
  color: #a9aeb8;
}
.aia-textarea :deep(.arco-textarea:focus) {
  box-shadow: none;
  background: transparent;
}
.aia-textarea :deep(.arco-textarea-word-limit) {
  margin: 0;
  padding: 0;
  color: #86909c;
}
.aia-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #e5e5ea;
}
.aia-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.aia-toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.aia-toolbar-btn-wrap {
  display: inline-flex;
}
.aia-toolbar-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #86909c;
  cursor: pointer;
  transition: all 0.15s ease;
}
.aia-toolbar-btn:hover:not(:disabled) {
  background: #e8f1ff;
  color: rgb(var(--primary-6));
}
.aia-toolbar-btn:disabled {
  color: #c9cdd4;
  cursor: not-allowed;
}
.aia-hint {
  font-size: 11px;
  color: #a9aeb8;
  white-space: nowrap;
  user-select: none;
}
.aia-count {
  font-size: 12px;
  color: #86909c;
}
.aia-status {
  font-size: 12px;
  color: #86909c;
}
.aia-video-modal :deep(.arco-modal-content) {
  padding: 0;
  background: #000;
}
.aia-video-modal :deep(.arco-modal-body) {
  padding: 0;
}
.aia-video-player {
  width: 100%;
  max-height: 70vh;
  display: block;
}
.aia-root--compact .aia-card {
  width: 178px;
  height: 52px;
  padding: 5px 6px;
  gap: 6px;
}
.aia-root--compact .aia-card__thumb {
  width: 38px;
  height: 38px;
}
.aia-root--compact .aia-card__name {
  font-size: 11px;
}
.aia-root--compact .aia-card__meta {
  font-size: 9px;
}
.aia-root--dark {
  background: #2c2c2e;
  border-color: rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  padding: 14px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.2);
}
.aia-root--dark.aia-root--dragging {
  border-color: rgba(0, 122, 255, 0.65);
  background: rgba(0, 122, 255, 0.08);
  box-shadow:
    0 0 0 3px rgba(0, 122, 255, 0.18),
    0 2px 12px rgba(0, 0, 0, 0.2);
}
.aia-root--dark .aia-card {
  border-color: rgba(255, 255, 255, 0.1);
  background: #3a3a3c;
}
.aia-root--dark .aia-card:hover {
  background: #434345;
  border-color: rgba(255, 255, 255, 0.16);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25);
}
.aia-root--dark .aia-card--pending,
.aia-root--dark .aia-card--uploading {
  border-color: rgba(255, 255, 255, 0.12);
}
.aia-root--dark .aia-card--pending .aia-card__meta,
.aia-root--dark .aia-card--uploading .aia-card__meta {
  color: rgb(var(--primary-6));
}
.aia-root--dark .aia-card__thumb {
  background: #2c2c2e;
}
.aia-root--dark .aia-cards__arrow {
  background: rgba(60, 60, 62, 0.95);
  color: #aeaeb2;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
}
.aia-root--dark .aia-cards__arrow:hover {
  background: #48484a;
  color: #ffffff;
}
.aia-root--dark .aia-card__name {
  color: #f2f2f7;
}
.aia-root--dark .aia-card__meta {
  color: #8e8e93;
}
.aia-root--dark .aia-card__file-icon {
  color: #aeaeb2;
}
.aia-root--dark .aia-textarea :deep(.arco-textarea) {
  color: #f2f2f7;
}
.aia-root--dark .aia-textarea :deep(.arco-textarea)::placeholder {
  color: #8e8e93;
}
.aia-root--dark .aia-textarea :deep(.arco-textarea-word-limit) {
  color: #7a7a80;
}
.aia-root--dark .aia-toolbar {
  border-top-color: rgba(255, 255, 255, 0.08);
}
.aia-root--dark .aia-toolbar-btn {
  color: #aeaeb2;
}
.aia-root--dark .aia-toolbar-btn:hover:not(:disabled) {
  background: #3a3a3c;
  color: #ffffff;
}
.aia-root--dark .aia-toolbar-btn:disabled {
  color: #5f5f63;
}
.aia-root--dark .aia-hint,
.aia-root--dark .aia-count {
  color: #7a7a80;
}
</style>
