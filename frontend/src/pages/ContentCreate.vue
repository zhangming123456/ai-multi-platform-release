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

          <template v-else-if="isGenerating">
            <div v-if="streamingText" class="streaming-view">
              <div class="streaming-view__header">
                <PlatformIcon
                  v-if="streamingPlatform"
                  :platform="streamingPlatformIcon"
                  size="sm"
                />
                <span v-if="streamingPlatform" class="text-[13px] font-medium">{{
                  platformLabel(streamingPlatform)
                }}</span>
                <span class="streaming-view__badge">AI 生成中</span>
              </div>
              <div ref="streamingTextRef" class="streaming-view__text">
                {{ streamingText }}<span class="streaming-cursor"></span>
              </div>
            </div>
            <a-spin v-else :loading="true" class="w-full py-10">
              <template #icon><IconStar :size="30" :style="{ color: '#007AFF' }" spin /></template>
              <div class="text-center">
                <p class="text-[13px] text-[#86868B]">
                  正在为 {{ selectedPlatforms.length }} 个平台生成适配文案…
                </p>
              </div>
            </a-spin>
          </template>

          <template v-else-if="hasGenerated">
            <div class="preview-result">
              <div class="preview-tier">
                <a-tabs
                  :active-key="activePreview"
                  type="rounded"
                  size="mini"
                  @change="onPlatformTabChange"
                >
                  <a-tab-pane
                    v-for="p in previewPlatforms"
                    :key="p.value"
                    :title="p.label as string"
                  />
                </a-tabs>
              </div>
              <div v-if="availableForms.length" class="preview-tier">
                <a-tabs
                  :active-key="activeForm"
                  type="rounded"
                  size="mini"
                  @change="onFormTabChange"
                >
                  <a-tab-pane
                    v-for="f in availableForms"
                    :key="f.value"
                    :title="f.label as string"
                  />
                </a-tabs>
              </div>
              <div v-if="currentVariants.length" class="space-y-4 pt-2">
                <div v-if="currentVariants.length > 1" class="version-preview-bar">
                  <a-radio-group v-model="activeVersionIndex" size="small" type="button">
                    <a-radio v-for="(_, vi) in currentVariants" :key="vi" :value="vi">
                      版本 {{ vi + 1 }}
                    </a-radio>
                  </a-radio-group>
                </div>
                <div v-if="currentVariant">
                  <a-typography-text
                    type="secondary"
                    class="text-[11px] font-semibold uppercase tracking-[0.06em] block mb-1.5"
                    >标题</a-typography-text
                  >
                  <a-input :model-value="currentVariant.title" read-only class="font-semibold" />
                </div>
                <div v-if="currentVariant">
                  <a-typography-text
                    type="secondary"
                    class="text-[11px] font-semibold uppercase tracking-[0.06em] block mb-1.5"
                  >
                    正文（底部含推荐话题标签）
                  </a-typography-text>
                  <a-textarea
                    :model-value="bodyWithTags(currentVariant)"
                    read-only
                    :auto-size="true"
                  />
                </div>
              </div>
              <a-empty
                v-if="availableForms.length === 0"
                class="pt-6"
                description="当前平台与内容形式暂无生成结果，请切换查看"
              />
            </div>
          </template>

          <template #extra v-if="currentVariants.length && currentVariant">
            <div class="flex items-center justify-end gap-2 pt-1">
              <a-button size="mini" @click="copyContent">
                <template #icon><IconCopy /></template>
                复制文案
              </a-button>
              <a-button size="mini" type="primary" :loading="isSaving" @click="saveContent">
                <template #icon><IconSave /></template>
                保存为内容
              </a-button>
            </div>
          </template>
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
            <div class="event-block">
              <div v-if="selectedCampaign" class="campaign-summary">
                <a-avatar
                  v-if="campaignMedia(selectedCampaign).length"
                  shape="square"
                  :size="44"
                  :image-url="campaignMedia(selectedCampaign)[0]"
                  class="campaign-summary__media"
                />
                <div v-else class="campaign-summary__media campaign-summary__media--empty">
                  <IconGift :size="20" />
                </div>
                <div class="campaign-summary__info">
                  <div class="campaign-summary__name">
                    {{ selectedCampaign.name }}
                    <a-tag size="small" color="blue" class="campaign-summary__tag">活动库</a-tag>
                  </div>
                  <div class="campaign-summary__meta">
                    <span v-if="selectedCampaign.location" class="campaign-summary__loc">{{
                      selectedCampaign.location
                    }}</span>
                    <template v-for="p in campaignPlatforms(selectedCampaign)" :key="p">
                      <PlatformIcon :platform="p" size="sm" />
                    </template>
                  </div>
                  <div class="campaign-summary__desc">
                    {{ selectedCampaign.description || '暂无描述' }}
                  </div>
                </div>
                <div class="campaign-summary__ops">
                  <a-button size="mini" @click="openCampaignPicker">更换</a-button>
                  <a-button size="mini" status="danger" @click="clearSelectedCampaign">
                    清除
                  </a-button>
                </div>
              </div>
              <div v-else-if="eventForm.name.trim()" class="campaign-summary event-summary">
                <a-avatar
                  v-if="eventForm.imageUrls.length"
                  shape="square"
                  :size="44"
                  :image-url="eventForm.imageUrls[0]"
                  class="campaign-summary__media"
                />
                <div v-else class="campaign-summary__media campaign-summary__media--empty">
                  <IconGift :size="20" />
                </div>
                <div class="campaign-summary__info">
                  <div class="campaign-summary__name">
                    {{ eventForm.name }}
                    <a-tag size="small" color="arcoblue" class="campaign-summary__tag"
                      >临时活动</a-tag
                    >
                  </div>
                  <div class="campaign-summary__meta">
                    <span v-if="eventForm.location.trim()" class="campaign-summary__loc">{{
                      eventForm.location
                    }}</span>
                    <a-tag
                      v-if="saveCampaignStore"
                      size="small"
                      color="green"
                      class="campaign-summary__tag"
                    >
                      保存到活动库
                    </a-tag>
                  </div>
                  <div class="campaign-summary__desc">
                    {{ eventForm.description || '暂无描述' }}
                  </div>
                </div>
                <div class="campaign-summary__ops">
                  <a-button size="mini" @click="openTempEventPicker">编辑</a-button>
                  <a-button size="mini" status="danger" @click="clearTempEvent">清除</a-button>
                </div>
              </div>
            </div>

            <div class="combo-hint" :class="{ 'combo-hint--over': comboExceeded }">
              {{ comboSnippet }}
            </div>

            <div v-if="hasFiles" class="compress-bar">
              <span class="compress-bar__label">图片压缩</span>
              <a-select
                v-model="compressMaxWidth"
                size="small"
                class="compress-bar__select"
                @change="onCompressChange"
              >
                <a-option :value="1280">最大宽度 1280</a-option>
                <a-option :value="1920">最大宽度 1920</a-option>
                <a-option :value="2560">最大宽度 2560</a-option>
              </a-select>
              <a-select
                v-model="compressQuality"
                size="small"
                class="compress-bar__select"
                @change="onCompressChange"
              >
                <a-option :value="0.6">质量 60%</a-option>
                <a-option :value="0.8">质量 80%</a-option>
                <a-option :value="0.9">质量 90%</a-option>
              </a-select>
              <span class="compress-bar__hint">压缩后单张 ≤ 500KB</span>
            </div>

            <a-form ref="createFormRef" layout="vertical" :model="createForm" class="create-form">
              <a-form-item field="content" :rules="contentRules" class="!mb-0">
                <AttachmentInputArea
                  ref="createAreaRef"
                  v-model="createForm.promptText"
                  v-model:file-list="createForm.fileUrls"
                  theme="dark"
                  :file-types="['image', 'video']"
                  :max-count="MAX_UPLOAD_FILES"
                  :min-rows="3"
                  :max-rows="5"
                  :hint="inputHint"
                  :placeholder="promptPlaceholder"
                  enter-behavior="send"
                  :material-picker="true"
                  :toolbar-options="toolbarOptions"
                  @enter="generate"
                  @change="onAreaChange"
                >
                  <template #toolbar-right>
                    <span
                      class="model-select-wrap"
                      :class="{ 'model-select-wrap--compact': modelCompact }"
                    >
                      <a-select
                        v-model="createForm.model"
                        :placeholder="modelOptions.length ? '选择模型' : '无可用模型'"
                        size="small"
                        class="chat-model-select"
                        @change="onModelChange"
                      >
                        <a-option v-for="opt in modelOptions" :key="opt.key" :value="opt.key">
                          <span class="provider-opt">
                            <span class="provider-opt__name">
                              <span class="provider-opt__bracket">【</span>
                              <span>{{ opt.planName }}</span>
                              <span class="provider-opt__bracket">】</span>
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
                      :disabled="isGenerating"
                      @click="generate"
                    >
                      <IconArrowUp v-if="!isGenerating" :size="18" />
                      <IconLoading v-else :size="16" spin />
                    </button>
                  </template>
                </AttachmentInputArea>
              </a-form-item>
              <a-form-item field="model" :rules="modelRules" style="display: none" />
            </a-form>
          </div>
        </div>
      </div>
    </div>

    <a-modal
      v-model:visible="campaignPickerVisible"
      title="选择活动"
      :width="760"
      :footer="false"
      :unmount-on-close="true"
    >
      <a-tabs v-model:active-key="campaignPickerTab" class="activity-modal-tabs">
        <a-tab-pane key="library" title="从活动库选择">
          <div class="campaign-picker">
            <div class="campaign-picker__filters">
              <a-input-search
                v-model="campaignKeyword"
                placeholder="搜索活动名称 / 描述"
                allow-clear
                class="campaign-picker__search"
                @search="onCampaignSearch"
                @press-enter="onCampaignSearch"
                @clear="onCampaignSearch"
              />
            </div>

            <a-spin :loading="campaignLoading" class="w-full flex-1">
              <div v-if="campaignList.length" class="campaign-picker__list">
                <div
                  v-for="c in campaignList"
                  :key="c.id"
                  class="campaign-row"
                  :class="{ 'campaign-row--selected': campaignPickedId === c.id }"
                  @click="campaignPickedId = c.id"
                >
                  <span class="campaign-row__radio">
                    <span v-if="campaignPickedId === c.id" class="campaign-row__radio-dot"></span>
                  </span>
                  <a-avatar
                    v-if="campaignMedia(c).length"
                    shape="square"
                    :size="44"
                    :image-url="campaignMedia(c)[0]"
                    class="campaign-row__media"
                  />
                  <div v-else class="campaign-row__media campaign-row__media--empty">
                    <IconGift :size="18" />
                  </div>
                  <div class="campaign-row__info">
                    <div class="campaign-row__head">
                      <span class="campaign-row__name">{{ c.name }}</span>
                      <span v-if="c.location" class="campaign-row__loc">{{ c.location }}</span>
                      <a-tag
                        :color="c.status === 'active' ? 'green' : 'gray'"
                        size="small"
                        class="campaign-row__status"
                      >
                        {{ c.status === 'active' ? '进行中' : '已归档' }}
                      </a-tag>
                    </div>
                    <div class="campaign-row__platforms">
                      <template v-for="p in campaignPlatforms(c)" :key="p">
                        <PlatformIcon :platform="p" size="sm" />
                      </template>
                      <span v-if="!campaignPlatforms(c).length" class="text-[#c9cdd4]"
                        >未设置平台</span
                      >
                    </div>
                    <div class="campaign-row__desc">{{ c.description || '暂无描述' }}</div>
                  </div>
                  <span class="campaign-row__time">{{ formatDateTime(c.created_at) }}</span>
                </div>
              </div>
              <div v-else-if="!campaignLoading" class="campaign-picker__empty">
                <a-empty description="未找到匹配活动" />
                <a-button size="small" @click="onEmptyCreateTemp">
                  <template #icon><IconPlus /></template>
                  临时录入
                </a-button>
              </div>
            </a-spin>

            <div class="campaign-picker__footer">
              <a-pagination
                v-if="campaignTotal > 0"
                :total="campaignTotal"
                :current="campaignPage"
                :page-size="campaignPageSize"
                size="small"
                show-total
                @change="onCampaignPageChange"
              />
              <div class="campaign-picker__footer-actions">
                <a-button @click="campaignPickerVisible = false">取消</a-button>
                <a-button type="primary" @click="confirmCampaignPicker">确定</a-button>
              </div>
            </div>
          </div>
        </a-tab-pane>
        <a-tab-pane key="temp" title="临时录入">
          <div class="event-entry">
            <div class="event-entry__field">
              <span class="event-entry__label">
                活动名称
                <span class="event-entry__required">*</span>
              </span>
              <a-input
                v-model="tempDraft.name"
                placeholder="请输入活动名称"
                maxlength="60"
                show-word-limit
              />
            </div>
            <div class="event-entry__field">
              <span class="event-entry__label">
                活动描述
                <span class="event-entry__required">*</span>
              </span>
              <AttachmentInputArea
                v-model="tempDraft.description"
                v-model:file-list="tempDraft.imageUrls"
                :upload="dataUrlUpload"
                placeholder="描述活动背景、宣发重点，AI 将结合这些信息创作文案"
                upload-mode="auto"
                :file-types="['image']"
                :max-count="9"
                hint="宣传图（选填），最多 9 张"
              />
            </div>
            <div class="event-entry__field">
              <span class="event-entry__label">活动地点</span>
              <a-input
                v-model="tempDraft.location"
                placeholder="线下地点或线上形式，选填"
                maxlength="120"
              />
            </div>
            <a-checkbox v-model="tempSaveStore" class="event-entry__check">
              生成前将活动保存到活动库
            </a-checkbox>
          </div>
          <div class="event-entry__actions">
            <a-button @click="resetTempEventForm">重置</a-button>
            <div class="event-entry__actions-right">
              <a-button @click="campaignPickerVisible = false">取消</a-button>
              <a-button type="primary" @click="confirmEventForm">确认</a-button>
            </div>
          </div>
        </a-tab-pane>
      </a-tabs>
    </a-modal>

    <a-modal
      v-model:visible="materialPickerVisible"
      title="从素材库选择"
      :width="720"
      :unmount-on-close="true"
      @close="resetMaterialPicker"
    >
      <div class="material-picker">
        <div class="material-picker__filters">
          <a-input-search
            v-model="materialKeyword"
            placeholder="搜索素材名称"
            allow-clear
            class="material-picker__search"
            @search="onMaterialSearch"
            @clear="onMaterialSearch"
          />
        </div>
        <div v-if="materialLoading" class="material-picker__empty">加载中...</div>
        <div v-else-if="materialList.length === 0" class="material-picker__empty">暂无素材</div>
        <div v-else class="material-picker__grid">
          <div
            v-for="m in materialList"
            :key="m.id"
            class="material-picker__card"
            :class="{ 'material-picker__card--active': materialSelected.has(m.id) }"
            @click="toggleMaterial(m)"
          >
            <img
              v-if="m.type === 'image'"
              :src="m.url"
              :alt="m.name"
              loading="lazy"
              class="material-picker__img"
            />
            <div v-else class="material-picker__fileicon">
              <IconFile :size="24" />
            </div>
            <div class="material-picker__meta">
              <span class="material-picker__name">{{ m.name }}</span>
            </div>
            <span v-if="materialSelected.has(m.id)" class="material-picker__check">
              <IconCheck :size="14" />
            </span>
          </div>
        </div>
        <div v-if="materialTotal > materialList.length" class="material-picker__more">
          <a-button size="small" type="text" :loading="materialLoading" @click="loadMoreMaterials">
            加载更多
          </a-button>
        </div>
      </div>
      <template #footer>
        <a-button @click="materialPickerVisible = false">取消</a-button>
        <a-button
          type="primary"
          :disabled="materialSelected.size === 0"
          @click="confirmPickMaterials"
        >
          添加（{{ materialSelected.size }}）
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted, nextTick, watch, unref, h } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { type DropdownMenuOption, type DropdownMenuOptions } from '@/components/DropdownMenu/types'

import {
  IconCopy,
  IconSave,
  IconSettings,
  IconCode,
  IconDelete,
  IconArrowUp,
  IconLoading,
  IconStar,
  IconRobot,
  IconGift,
  IconPlus,
  IconFile,
  IconCheck,
  IconFolder,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import { useTokenPlanStore, parseModelField } from '@/stores/tokenPlan'
import { urlFileName } from '@/composables/useUrlExtractor'
import api, { getApiErrorDetail } from '@/utils/api'
import { formatDateTime } from '@/utils/time'
import type { Campaign, Material, Paginated } from '@/types'
import type { PlatformIconType } from '@/components/shared/PlatformIcon.ts'
import { isString } from 'lodash-es'

const router = useRouter()
const store = useTokenPlanStore()

type LogLevel = 'info' | 'req' | 'ok' | 'err'

interface LogEntry {
  id: string
  time: string
  level: LogLevel
  message: string
}

interface VariantItem {
  title: string
  body: string
  hashtags: string[]
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
  return unref(platformChoices).find((p) => p.value === value)?.label || value
}

function contentFormLabel(value: string) {
  return contentFormChoices.find((f) => f.value === value)?.label || value
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

const selectedPlatforms = ref<string[]>(['xiaohongshu'])
const selectedContentForms = ref<string[]>(['post'])

const creationMode = computed<'free' | 'event'>(() => {
  return selectedCampaign.value ? 'event' : 'free'
})

const eventForm = reactive({
  name: '',
  description: '',
  location: '',
  imageUrls: [] as string[],
})
const saveCampaignStore = ref(false)
const selectedCampaign = ref<Campaign | null>(null)

function selectCampaign(campaign: Campaign | DropdownMenuOption) {
  selectedCampaign.value = campaign as Campaign
  const valid = (unref(selectedCampaign)?.platforms || []).filter((p) =>
    unref(platformChoices).some((pc) => pc.value === p),
  )
  if (valid.length > 0) {
    selectedPlatforms.value = valid as PlatformIconType[]
  }
  eventForm.name = ''
  eventForm.description = ''
  eventForm.location = ''
  eventForm.imageUrls = []
  saveCampaignStore.value = false
  pushLog('ok', `已关联活动「${unref(selectedCampaign)?.name}」`)
}

function clearSelectedCampaign() {
  if (selectedCampaign.value) {
    pushLog('info', `已取消活动关联「${selectedCampaign.value.name}」`)
  }
  selectedCampaign.value = null
}

function campaignMedia(c: Campaign): string[] {
  return (c.media_urls || []).filter(Boolean)
}

function campaignPlatforms(c: Campaign): PlatformIconType[] {
  return (c.platforms || []).filter((p): p is PlatformIconType =>
    unref(platformChoices).some((pc) => pc.value === p),
  )
}

const campaignPickerVisible = ref(false)
const campaignPickerTab = ref<'library' | 'temp'>('library')
const tempEventImageNote = ref('')
const tempDraft = reactive({
  name: '',
  description: '',
  location: '',
  imageUrls: [] as string[],
})
const tempSaveStore = ref(false)
const campaignLoading = ref(false)
const campaignList = ref<Campaign[]>([])
const campaignTotal = ref(0)
const campaignPage = ref(1)
const campaignPageSize = ref(10)
const campaignKeyword = ref('')
const campaignPlatformFilter = ref<string | undefined>(undefined)
const campaignStatusFilter = ref('active')
const campaignLocationFilter = ref('')
const campaignDateRange = ref<[string, string] | undefined>(undefined)
const campaignPickedId = ref('')

const platformChoices = computed<Partial<DropdownMenuOption>[]>(() => {
  return [
    { value: 'wechat_mp', label: '公众号' },
    { value: 'xiaohongshu', label: '小红书' },
    { value: 'douyin', label: '抖音' },
    { value: 'wechat_video', label: '视频号' },
    { value: 'wechat_moments', label: '朋友圈' },
    { value: 'weibo', label: '微博' },
  ].map((item) => {
    ;(item as any).icon = h(PlatformIcon, {
      platform: item.value as any,
      size: 16,
    })
    return item
  })
})

const contentFormChoices: Partial<DropdownMenuOption>[] = [
  { value: 'post', label: '图文笔记/推文' },
  { value: 'script', label: '短视频脚本' },
  { value: 'snippet', label: '短文案/标题' },
  { value: 'live_script', label: '直播脚本' },
  { value: 'product_detail', label: '商品详情页文案' },
]

const versionFormChoices = computed(() => {
  return Array.from({ length: 3 }).map((value, index) => {
    value = String(index + 1)
    return {
      value,
      label: value,
    }
  })
})

const toolbarOptions = computed(() => {
  const options: DropdownMenuOptions = [
    {
      value: 'pick-material',
      label: '从素材库选择',
      icon: IconFolder,
      onClick: openMaterialPicker,
    },
    [
      {
        label: '形式',
        icon: IconFolder,
        // showSearch: true,
        multiple: true,
        isCheck: (option) => (unref(selectedContentForms) as any[]).includes(option.value ?? null),
        onOptionClick: (option) => {
          const values = unref(selectedContentForms)
          if (values.some((value) => option.value === value)) {
            selectedContentForms.value = values.filter((value) => value !== option.value)
          } else {
            unref(selectedContentForms).push(option.value as string)
          }
        },
        children: (keyword) => {
          if (keyword) {
            return unref(contentFormChoices).filter((item) => {
              return isString(item.label) && item.label.includes(keyword)
            })
          }
          return unref(contentFormChoices)
        },
      },
      {
        label: '平台',
        icon: IconFolder as never,
        // showSearch: true,
        multiple: true,
        isCheck: (option) => (unref(selectedPlatforms) as any[]).includes(option.value ?? null),
        onOptionClick: (option) => {
          const values = unref(selectedPlatforms)
          if (values.some((value) => option.value === value)) {
            selectedPlatforms.value = values.filter((value) => value !== option.value)
          } else {
            unref(selectedPlatforms).push(option.value as string)
          }
        },
        children: (keyword?: string) => {
          if (keyword) {
            return unref(platformChoices).filter((item) => {
              return isString(item.label) && item.label.includes(keyword)
            })
          }
          return unref(platformChoices)
        },
      },
      {
        label: '选择活动',
        icon: IconFolder as never,
        showSearch: true,
        isCheck: (option) => unref(selectedCampaign)?.id === option?.value,
        onOptionClick: selectCampaign,
        children: async (keyword, option) => {
          try {
            const result = await fetchCampaignOptions({
              keyword: keyword ?? '',
              page: option!.request!.page,
            })
            return {
              valueKey: 'id',
              labelKey: 'name',
              data: (result?.items ?? []) as never,
              total: result?.total ?? 0,
              page: result?.page ?? 0,
              pageSize: result?.page_size ?? 0,
              response: result,
            }
          } catch (error) {
            return {}
          }
        },
      },
      {
        label: '选择版本',
        icon: IconFolder as never,
        isCheck: (option) => unref(versionNum) === option.value,
        onOptionClick: (option) => {
          versionNum.value = option.value as string
        },
        children: unref(versionFormChoices) as any,
        triggerProps: {
          contentStyle: {},
        },
      },
    ],
  ]
  return options
})

async function fetchCampaignOptions(params?: Record<string, unknown>) {
  campaignLoading.value = true
  try {
    if (!params) {
      params = {
        page: campaignPage.value,
        page_size: campaignPageSize.value,
        status: campaignStatusFilter.value,
      }
      if (campaignKeyword.value.trim()) {
        params.keyword = campaignKeyword.value.trim()
      }
      if (campaignPlatformFilter.value) {
        params.platform = campaignPlatformFilter.value
      }
      if (campaignLocationFilter.value.trim()) {
        params.location = campaignLocationFilter.value.trim()
      }
      if (campaignDateRange.value) {
        params.start_date = campaignDateRange.value[0]
        params.end_date = campaignDateRange.value[1]
      }
    }
    const res = await api.get<Paginated<Campaign>>('/campaigns/options', { params })
    campaignList.value = res.data.items || []
    campaignTotal.value = res.data.total || 0
    return res.data
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '活动库加载失败')
  } finally {
    campaignLoading.value = false
  }
}

function openCampaignPicker() {
  campaignKeyword.value = ''
  campaignPlatformFilter.value = undefined
  campaignStatusFilter.value = 'active'
  campaignLocationFilter.value = ''
  campaignDateRange.value = undefined
  campaignPage.value = 1
  campaignPickedId.value = selectedCampaign.value?.id || ''
  campaignPickerTab.value = 'library'
  campaignPickerVisible.value = true
  void fetchCampaignOptions()
}

function onCampaignSearch() {
  campaignPage.value = 1
  void fetchCampaignOptions()
}

function onCampaignPageChange(page: number) {
  campaignPage.value = page
  void fetchCampaignOptions()
}

function confirmCampaignPicker() {
  const campaign = campaignList.value.find((c) => c.id === campaignPickedId.value)
  if (!campaign) {
    Message.warning('请先选择一条活动')
    return
  }
  if (campaign.status !== 'active') {
    Message.warning('已归档活动不可用于创作，请选择进行中活动')
    return
  }
  selectCampaign(campaign)
  campaignPickerVisible.value = false
}

function openTempEventPicker() {
  tempDraft.name = eventForm.name
  tempDraft.description = eventForm.description
  tempDraft.location = eventForm.location
  tempDraft.imageUrls = [...eventForm.imageUrls]
  tempSaveStore.value = saveCampaignStore.value
  tempEventImageNote.value = ''
  campaignPickerTab.value = 'temp'
  campaignPickerVisible.value = true
}

function confirmEventForm() {
  if (!tempDraft.name.trim() || !tempDraft.description.trim()) {
    Message.warning('请完整填写活动名称与活动描述')
    return
  }
  selectedCampaign.value = null
  eventForm.name = tempDraft.name.trim()
  eventForm.description = tempDraft.description.trim()
  eventForm.location = tempDraft.location.trim()
  eventForm.imageUrls = [...tempDraft.imageUrls]
  saveCampaignStore.value = tempSaveStore.value
  campaignPickerVisible.value = false
  pushLog('ok', `已确认临时活动「${eventForm.name}」`)
}

function resetTempEventForm() {
  tempDraft.name = ''
  tempDraft.description = ''
  tempDraft.location = ''
  tempDraft.imageUrls = []
  tempSaveStore.value = false
  tempEventImageNote.value = ''
}

function clearTempEvent() {
  if (eventForm.name.trim()) {
    pushLog('info', `已清除临时活动「${eventForm.name}」`)
  }
  eventForm.name = ''
  eventForm.description = ''
  eventForm.location = ''
  eventForm.imageUrls = []
  saveCampaignStore.value = false
}

const materialPickerVisible = ref(false)
const materialList = ref<Material[]>([])
const materialKeyword = ref('')
const materialLoading = ref(false)
const materialTotal = ref(0)
const materialPage = ref(1)
const materialPageSize = 18
const materialSelected = reactive(new Set<string>())
const materialMap = ref(new Map<string, Material>())

async function loadMaterials(reset = false) {
  if (reset) {
    materialList.value = []
    materialPage.value = 1
  }
  materialLoading.value = true
  try {
    const params: Record<string, unknown> = {
      type: 'image',
      page: materialPage.value,
      page_size: materialPageSize,
    }
    if (materialKeyword.value) params.keyword = materialKeyword.value
    const res = await api.get<Paginated<Material>>('/materials/', { params })
    const items = res.data.items || []
    materialList.value = reset ? items : [...materialList.value, ...items]
    materialTotal.value = res.data.total || 0
    items.forEach((m) => materialMap.value.set(m.id, m))
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载素材失败')
  } finally {
    materialLoading.value = false
  }
}

function openMaterialPicker() {
  materialPickerVisible.value = true
  if (materialList.value.length === 0) void loadMaterials(true)
}

function onMaterialSearch() {
  void loadMaterials(true)
}

function loadMoreMaterials() {
  if (materialLoading.value) return
  materialPage.value += 1
  void loadMaterials()
}

function toggleMaterial(m: Material) {
  if (materialSelected.has(m.id)) {
    materialSelected.delete(m.id)
  } else {
    materialSelected.add(m.id)
  }
}

function resetMaterialPicker() {
  materialSelected.clear()
  materialKeyword.value = ''
}

function confirmPickMaterials() {
  const urls = createForm.fileUrls
  const existing = new Set(urls)
  const remaining = MAX_UPLOAD_FILES - urls.length
  let added = 0
  materialSelected.forEach((id) => {
    const m = materialMap.value.get(id)
    if (!m || added >= remaining) return
    if (existing.has(m.url)) return
    urls.push(m.url)
    existing.add(m.url)
    added += 1
  })
  materialPickerVisible.value = false
  resetMaterialPicker()
  if (added > 0) {
    Message.success(`已添加 ${added} 个素材`)
    hasFiles.value = urls.length > 0
    void createFormRef.value?.validateField('content').catch(() => {})
  } else {
    Message.info('已达到上传数量上限或素材已存在')
  }
}

function onEmptyCreateTemp() {
  openTempEventPicker()
}

const isGenerating = ref(false)
const streamingText = ref('')
const streamingPlatform = ref('')
const streamingPlatformIcon = computed(() => streamingPlatform.value as PlatformIconType)
const MAX_UPLOAD_FILES = 10
const hasFiles = ref(false)

const createFormRef = ref<FormInstance>()
const createForm = reactive({
  model: '',
  promptText: '',
  fileUrls: [] as string[],
})

const modelRules = [{ required: true, message: '请选择 AI 模型' }]

const contentRules = [
  {
    validator: (_value: unknown, callback: (error?: string) => void) => {
      const hasKeyword = !!parsedPrompt.value.topic || parsedPrompt.value.keywords.length > 0
      const hasFile = hasFiles.value
      if (!hasKeyword && !hasFile) {
        callback('请输入关键词或上传图片（至少填一项）')
      } else {
        callback()
      }
    },
  },
]

interface PendingFileItem {
  file: File
  type: string
  name: string
  size: number
}

const createAreaRef = ref<{
  getPendingFiles: () => PendingFileItem[]
  pickFiles: () => void
  $el?: HTMLElement
}>()

function onAreaChange(payload: { total: number; pending: number }) {
  hasFiles.value = payload.total > 0
  void createFormRef.value?.validateField('content').catch(() => {})
}

const modelCompact = ref(false)
const MODEL_COMPACT_THRESHOLD = 440

let cardResizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (!store.loaded) {
    void store.loadPlans()
  }
  const el = createAreaRef.value?.$el
  if (el) {
    cardResizeObserver = new ResizeObserver((entries) => {
      const width = entries[0]?.contentRect.width ?? 0
      modelCompact.value = width < MODEL_COMPACT_THRESHOLD
    })
    cardResizeObserver.observe(el)
  }
})

onUnmounted(() => {
  cardResizeObserver?.disconnect()
  cardResizeObserver = null
})

const parsedPrompt = computed(() => {
  const text = createForm.promptText
  const keywords: string[] = []
  const topic = text
    .replace(/#[\p{L}\p{N}_\u4e00-\u9fa5]+/gu, (match) => {
      keywords.push(match.slice(1))
      return ''
    })
    .trim()
  return { topic, keywords }
})

function onModelChange(val: unknown) {
  if (typeof val !== 'string') return
  const idx = val.indexOf(':')
  if (idx <= 0) return
  store.selectModel(val.slice(0, idx), val.slice(idx + 1))
  void createFormRef.value?.validateField('model').catch(() => {})
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

watch(
  activeModelKey,
  (key) => {
    if (key && !createForm.model) {
      createForm.model = key
    }
  },
  { immediate: true },
)

watch(
  () => createForm.promptText,
  () => {
    void createFormRef.value?.validateField('content').catch(() => {})
  },
)

const inputHint = computed(() => (creationMode.value === 'event' ? '补充卖点/切入角度' : ''))
const promptPlaceholder = computed(() =>
  creationMode.value === 'event'
    ? '补充卖点/切入角度，例如：主打 #性价比 #亲子'
    : '输入创作需求，使用 #标签 添加关键词，例如：写一篇小红书文案 #穿搭 #夏季',
)

const compressMaxWidth = ref(1920)
const compressQuality = ref(0.8)

function onCompressChange() {
  pushLog(
    'info',
    `图片压缩参数已更新：最大宽度 ${compressMaxWidth.value}px · 质量 ${Math.round(compressQuality.value * 100)}%`,
  )
}

function compressImage(file: File): Promise<File> {
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
        const maxWidth = compressMaxWidth.value
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
          compressQuality.value,
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

async function fetchUrlToBase64(url: string): Promise<{ data: string; mime_type: string } | null> {
  try {
    const res = await fetch(url)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const blob = await res.blob()
    const file = new File([blob], urlFileName(url), {
      type: blob.type || 'application/octet-stream',
    })
    return await fileToBase64(file)
  } catch (err) {
    const e = err as { message?: string }
    pushLog('err', `链接素材获取失败：${url}（${e.message || '网络错误'}）`)
  }
  return null
}

async function buildFilesPayload(): Promise<{ data: string; mime_type: string }[] | undefined> {
  const pendingFiles = createAreaRef.value?.getPendingFiles() ?? []
  const urls = createForm.fileUrls
  const total = pendingFiles.length + urls.length
  if (total === 0) return undefined
  pushLog('info', `正在编码 ${total} 个素材文件…`)
  const tasks: Promise<{ data: string; mime_type: string } | null>[] = []
  for (const p of pendingFiles) {
    tasks.push(fileToBase64(p.file))
  }
  for (const url of urls) {
    tasks.push(fetchUrlToBase64(url))
  }
  const results = await Promise.all(tasks)
  const payload = results.filter((r): r is { data: string; mime_type: string } => r !== null)
  if (payload.length > 0) {
    pushLog('ok', `素材编码完成，共 ${payload.length} 个`)
  } else {
    pushLog('err', '所有素材均无法编码，本次生成将不带文件')
  }
  return payload
}

function dataUrlMime(dataUrl: string): string {
  const matched = /^data:([^;,]+)/.exec(dataUrl)
  return matched ? matched[1] : 'application/octet-stream'
}

function dataUrlBase(dataUrl: string): string {
  const comma = dataUrl.indexOf(',')
  return comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl
}

const eventMediaPayload = computed<{ data: string; mime_type: string }[]>(() =>
  eventForm.imageUrls
    .filter((u) => u.startsWith('data:'))
    .map((u) => ({ data: dataUrlBase(u), mime_type: dataUrlMime(u) })),
)

async function dataUrlUpload(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(new Error('文件读取失败'))
    reader.readAsDataURL(file)
  })
}

async function uploadEventImages(): Promise<string[]> {
  const out: string[] = []
  for (const u of eventForm.imageUrls) {
    if (u.startsWith('data:')) {
      try {
        const blob = await (await fetch(u)).blob()
        const file = new File([blob], 'event-media.png', { type: blob.type || 'image/png' })
        const fd = new FormData()
        fd.append('file', file, file.name)
        const res = await api.post<{ url: string }>('/uploads', fd)
        const url = res.data?.url || ''
        if (url) out.push(url)
      } catch {
        Message.error('宣传图上传失败，请重试')
        return out
      }
    } else {
      out.push(u)
    }
  }
  return out
}

async function createCampaignFromEventForm(): Promise<Campaign | null> {
  const mediaUrls = await uploadEventImages()
  const payload = {
    name: eventForm.name.trim(),
    description: eventForm.description,
    media_urls: mediaUrls,
    location: eventForm.location.trim(),
    platforms: [...selectedPlatforms.value],
  }
  try {
    const res = await api.post<Campaign>('/campaigns/', payload)
    return res.data
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '活动入库失败，请重试')
    return null
  }
}

const isSaving = ref(false)
const hasGenerated = ref(false)
const activePreview = ref('')
const activeForm = ref('post')
const activeVersionIndex = ref(0)

watch(streamingText, async () => {
  await nextTick()
  if (streamingTextRef.value) {
    streamingTextRef.value.scrollTop = streamingTextRef.value.scrollHeight
  }
})

const comboPlatformCount = computed(() => selectedPlatforms.value.length)
const comboFormCount = computed(() => selectedContentForms.value.length)
const comboVersionCount = computed(() => Number(versionNum.value) || 1)
const comboTotal = computed(
  () => comboPlatformCount.value * comboFormCount.value * comboVersionCount.value,
)
const comboTasks = computed(() => comboPlatformCount.value * comboFormCount.value)
const comboExceeded = computed(() => comboTotal.value > 6)
const comboSnippet = computed(
  () => `将生成 ${comboTotal.value} 篇（约 ${Math.ceil(comboTasks.value / 3)} 批）`,
)

const generatedVariants = ref<Record<string, VariantItem[]>>({})
const versionNum = ref('2')

const previewPlatforms = computed(() =>
  unref(platformChoices).filter((p) => selectedPlatforms.value.includes(p.value as string)),
)

const availableForms = computed(() =>
  contentFormChoices.filter((c) => {
    const list = generatedVariants.value[`${activePreview.value}::${c.value}`]
    return !!list && list.some(Boolean)
  }),
)

const currentVariants = computed(
  () => generatedVariants.value[`${activePreview.value}::${activeForm.value}`] || [],
)
const currentVariant = computed(() => currentVariants.value[activeVersionIndex.value])

function resetGenerationState() {
  hasGenerated.value = false
  generatedVariants.value = {}
  streamingText.value = ''
  streamingPlatform.value = ''
  activeVersionIndex.value = 0
  activePreview.value = selectedPlatforms.value[0] || ''
  activeForm.value = selectedContentForms.value[0] || 'post'
}

function onFormTabChange(key: string | number) {
  activeForm.value = String(key)
  activeVersionIndex.value = 0
}

function onPlatformTabChange(key: string | number) {
  activePreview.value = String(key)
  activeVersionIndex.value = 0
}

watch([activePreview, activeForm], () => {
  activeVersionIndex.value = 0
})

watch(currentVariants, (list) => {
  if (activeVersionIndex.value >= list.length) {
    activeVersionIndex.value = 0
  }
})

watch(
  () => [...selectedPlatforms.value],
  (list) => {
    if (list.length > 0 && !list.includes(activePreview.value)) {
      activePreview.value = list[0]
    }
  },
)

function hashtagsText(tags: string[]): string {
  return tags
    .map((t) => t.trim().replace(/^#+/, ''))
    .filter(Boolean)
    .map((t) => `#${t}`)
    .join(' ')
}

function bodyWithTags(variant: VariantItem): string {
  const tags = hashtagsText(variant.hashtags)
  return tags ? `${variant.body}\n${tags}` : variant.body
}

async function generate() {
  if (!store.loaded) {
    await store.loadPlans()
  }
  if (!store.activePlan) {
    pushLog('err', '未检测到可用的模型配置，已跳转至模型管理页')
    router.push('/settings/token-plan')
    return
  }
  try {
    await createFormRef.value?.validate()
  } catch (e) {
    const errors = e as Record<string, { message: string }[]>
    const first = Object.values(errors || {})[0]?.[0]?.message
    if (first) Message.error(first)
    return
  }
  if (selectedContentForms.value.length === 0) {
    pushLog('err', '请至少选择一种内容形式')
    Message.error('请至少选择一种内容形式')
    return
  }
  if (comboExceeded.value) {
    pushLog('err', '组合数量超出上限（平台×形式×版本需 ≤ 6），已取消生成')
    Message.error('组合数量超出上限（平台×形式×版本需 ≤ 6），请减少平台/形式/版本')
    return
  }
  if (hasFiles.value && !store.selectedModelSupportsFiles) {
    pushLog('err', '当前模型不支持文件上传，请切换到支持视觉/图片/视频的模型')
    Message.error('当前模型不支持文件上传，请切换到支持视觉/图片/视频的模型')
    return
  }

  const useEventMode = creationMode.value === 'event'
  if (useEventMode && !selectedCampaign.value) {
    if (!eventForm.name.trim() || !eventForm.description.trim()) {
      openTempEventPicker()
      pushLog('err', '活动创作需关联已有活动或完整填写临时活动信息')
      Message.error('请选择已有活动，或完整填写临时活动（名称与描述为必填）')
      return
    }
    if (saveCampaignStore.value) {
      pushLog('info', '正在将临时活动保存到活动库…')
      const created = await createCampaignFromEventForm()
      if (created) {
        selectCampaign(created)
        Message.success('活动已保存到活动库并关联')
      } else {
        saveCampaignStore.value = false
        pushLog('err', '活动入库失败，已取消勾选，将按临时活动继续生成')
        Message.error('活动入库失败，将按临时活动继续生成')
      }
    }
  }

  isGenerating.value = true
  resetGenerationState()
  activePreview.value = selectedPlatforms.value[0] || ''
  activeForm.value = selectedContentForms.value[0] || 'post'
  streamingPlatform.value = selectedPlatforms.value[0] || ''

  const startedAt = performance.now()
  const plan = store.activePlan
  const modelId = store.selectedModelId || store.activeModelList[0]?.id || ''
  const { topic, keywords } = parsedPrompt.value
  const inputDesc = topic
    ? `主题「${topic}」`
    : keywords.length > 0
      ? `关键词「${keywords.join('、')}」`
      : '上传素材'
  pushLog(
    'info',
    `开始生成任务 · ${inputDesc} · 平台 ${selectedPlatforms.value
      .map(platformLabel)
      .join(' / ')} · 形式 ${selectedContentForms.value
      .map(contentFormLabel)
      .join(' / ')} · ${versionNum.value} 版`,
  )
  pushLog('info', `使用模型配置 ${plan.name}（${modelId}）`)

  const keywordsArray = keywords.length > 0 ? keywords : undefined

  let filesPayload: { data: string; mime_type: string }[] | undefined
  if (hasFiles.value) {
    filesPayload = await buildFilesPayload()
  }

  const payload: Record<string, unknown> = {
    topic: topic,
    platforms: [...selectedPlatforms.value],
    plan_id: plan.id,
    model_id: modelId,
    generate_version: Number(versionNum.value),
    content_forms: [...selectedContentForms.value],
    ...(keywordsArray && keywordsArray.length > 0 ? { keywords: keywordsArray } : {}),
    ...(filesPayload && filesPayload.length > 0 ? { files: filesPayload } : {}),
  }
  if (useEventMode) {
    if (selectedCampaign.value) {
      payload.campaign_id = selectedCampaign.value.id
    } else {
      payload.event_context = {
        name: eventForm.name.trim(),
        description: eventForm.description,
        location: eventForm.location.trim(),
        media_files: eventMediaPayload.value,
      }
    }
  }

  try {
    const response = await fetch('/api/contents/ai-generate-stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(payload),
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
    const variants: Record<string, VariantItem[]> = {}

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

        const payloadEvent = JSON.parse(dataStr)

        switch (eventType) {
          case 'log':
            pushLog(payloadEvent.level || 'info', payloadEvent.message || '')
            break
          case 'chunk':
            if (payloadEvent.platform && streamingPlatform.value !== payloadEvent.platform) {
              streamingPlatform.value = payloadEvent.platform
              streamingText.value = ''
            } else if (!streamingPlatform.value) {
              streamingPlatform.value = activePreview.value || selectedPlatforms.value[0] || ''
            }
            streamingText.value += payloadEvent.text || ''
            break
          case 'done': {
            const form =
              typeof payloadEvent.content_form === 'string' && payloadEvent.content_form
                ? payloadEvent.content_form
                : 'post'
            if (payloadEvent.variant) {
              const v: VariantItem = {
                title: payloadEvent.variant.title || '',
                body: payloadEvent.variant.body || '',
                hashtags: Array.isArray(payloadEvent.variant.hashtags)
                  ? payloadEvent.variant.hashtags
                  : [],
              }
              const key = `${payloadEvent.platform}::${form}`
              if (!variants[key]) variants[key] = []
              const idx =
                typeof payloadEvent.variant_index === 'number'
                  ? payloadEvent.variant_index
                  : variants[key].length
              variants[key][idx] = v
            }
            pushLog(
              'ok',
              `${platformLabel(payloadEvent.platform)} · ${contentFormLabel(form)} 生成完成`,
            )
            streamingText.value = ''
            streamingPlatform.value = ''
            break
          }
          case 'error':
            pushLog('err', payloadEvent.message || '生成失败')
            streamingText.value = ''
            streamingPlatform.value = ''
            if (payloadEvent.available_plans?.length > 0) {
              const planNames = payloadEvent.available_plans
                .map(
                  (p: { display_name?: string; name: string; provider: string; model: string }) =>
                    `${p.display_name || p.name}（${p.provider}/${p.model}）`,
                )
                .join('、')
              Message.warning(`可前往设置切换至：${planNames}`)
            }
            Message.error(payloadEvent.message || 'AI 生成失败')
            break
          case 'complete':
            break
        }
      }
    }

    const totalCount = Object.values(variants).reduce(
      (sum, list) => sum + list.filter(Boolean).length,
      0,
    )

    if (totalCount > 0) {
      generatedVariants.value = variants
      hasGenerated.value = true
      const resultPlatform = previewPlatforms.value.find((p) =>
        Object.keys(variants).some((k) => k.startsWith(`${p.value}::`)),
      )
      activePreview.value = resultPlatform?.value || selectedPlatforms.value[0] || ''
      const resultForms = contentFormChoices.filter((c) => {
        const list = variants[`${activePreview.value}::${c.value}`]
        return !!list && list.some(Boolean)
      })
      activeForm.value = resultForms[0]?.value || selectedContentForms.value[0] || 'post'
      activeVersionIndex.value = 0
      const total = Math.round(performance.now() - startedAt)
      pushLog('ok', `生成完成 · ${totalCount} 篇 · 总耗时 ${(total / 1000).toFixed(1)}s`)
      Message.success(`生成完成 · 共 ${totalCount} 篇`)
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
  const variant = currentVariant.value
  if (!variant) return

  isSaving.value = true
  try {
    const payload: Record<string, unknown> = {
      title: variant.title,
      body: bodyWithTags(variant),
      platform: activePreview.value,
      status: 'draft',
    }
    if (creationMode.value === 'event') {
      if (selectedCampaign.value) {
        payload.campaign_id = selectedCampaign.value.id
      } else if (eventForm.name.trim()) {
        payload.event_meta = {
          name: eventForm.name.trim(),
          description: eventForm.description,
          location: eventForm.location.trim(),
        }
      }
    }
    await api.post('/contents/', payload)
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
  const variant = currentVariant.value
  if (!variant) return
  const text = `${variant.title}\n\n${bodyWithTags(variant)}`
  try {
    await navigator.clipboard.writeText(text)
    Message.success('已复制到剪贴板')
  } catch {
    Message.error('复制失败，请手动选择文本复制')
  }
}
</script>

<style scoped lang="scss">
.creation-mode-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 2px 0 12px;
}

.creation-mode-bar__label {
  font-size: 13px;
  font-weight: 500;
  color: #6e6e73;
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
  min-width: 200px;
  max-width: 375px;
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
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.content-left .log-terminal__body-wrap {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.content-left .log-terminal__body {
  height: 100%;
  min-height: 0;
  overflow-y: auto;
}

.content-left .chat-input-section {
  flex-shrink: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: thin;
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
  display: flex;
  flex-direction: column;
}

.preview-result {
  padding-bottom: 20px;
}

.preview-tier {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.preview-tier:last-of-type {
  margin-bottom: 0;
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
  max-height: 100%;
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

.chat-input-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: auto;
}

.event-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.create-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: 8px 10px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.toolbar-seg {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.toolbar-seg--grow {
  flex: 1 1 auto;
  min-width: 180px;
}

.toolbar-label {
  flex: 0 0 auto;
  font-size: 11px;
  color: #8e8e93;
  white-space: nowrap;
}

.toolbar-divider {
  width: 1px;
  height: 16px;
  flex: 0 0 auto;
  background: rgba(255, 255, 255, 0.12);
}

.toolbar-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 220px;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: #2c2c2e;
  color: #d1d1d6;
  font-size: 12px;
  line-height: 18px;
  cursor: pointer;
  transition:
    background 0.15s ease,
    color 0.15s ease,
    border-color 0.15s ease;
}

.toolbar-chip:hover {
  background: #3a3a3c;
  color: #f2f2f7;
  border-color: rgba(0, 122, 255, 0.45);
}

.toolbar-chip--accent {
  color: #7cb8ff;
  border-color: rgba(0, 122, 255, 0.5);
}

.toolbar-chip--active {
  background: rgba(0, 122, 255, 0.18);
  border-color: rgba(0, 122, 255, 0.55);
  color: #7cb8ff;
}

.toolbar-chip__text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.toolbar-chip__count {
  min-width: 16px;
  height: 16px;
  padding: 0 5px;
  border-radius: 999px;
  background: #007aff;
  color: #fff;
  font-size: 11px;
  line-height: 16px;
  text-align: center;
}

.toolbar-platform-select {
  width: 200px;
}

.toolbar-platform-select :deep(.arco-select-view) {
  background: #2c2c2e;
  border-color: rgba(255, 255, 255, 0.14);
}

.toolbar-platform-select :deep(.arco-select-view-value),
.toolbar-platform-select :deep(.arco-select-view-placeholder) {
  color: #d1d1d6;
}

.toolbar-platform-select :deep(.arco-select-view:hover),
.toolbar-platform-select :deep(.arco-select-view.arco-select-view--focus) {
  border-color: rgba(0, 122, 255, 0.45);
}

.toolbar-platform-select :deep(.arco-select-view .arco-tag) {
  background: #3a3a3c;
  border-color: rgba(255, 255, 255, 0.12);
  color: #f2f2f7;
}

.toolbar-platform-select :deep(.arco-select-view .arco-tag .arco-icon-close) {
  color: #aeaeb2;
}

.campaign-summary__tag {
  margin-left: 4px;
}

.activity-modal-tabs :deep(.arco-tabs-content) {
  padding-top: 14px;
}

.preview-tier :deep(.arco-tabs-content) {
  display: none;
}

.event-entry {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.event-entry__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.event-entry__label {
  font-size: 13px;
  font-weight: 500;
  color: #4e5969;
}

.event-entry__required {
  margin-left: 2px;
  color: #f53f3f;
  font-style: normal;
}

.event-entry__check :deep(.arco-checkbox-label) {
  font-size: 13px;
  color: #4e5969;
}

.event-entry__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border-2);
}

.event-entry__actions-right {
  display: inline-flex;
  gap: 8px;
}

.campaign-summary {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.campaign-summary__media {
  flex: 0 0 auto;
  border-radius: 8px;
  overflow: hidden;
}

.campaign-summary__media--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: #2c2c2e;
  color: #8e8e93;
}

.campaign-summary__info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.campaign-summary__name {
  font-size: 13px;
  font-weight: 600;
  color: #f2f2f7;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.campaign-summary__meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.campaign-summary__loc {
  font-size: 11px;
  color: #8e8e93;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 160px;
}

.campaign-summary__desc {
  font-size: 11px;
  line-height: 1.5;
  color: #aeaeb2;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-all;
}

.campaign-summary__ops {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.event-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.event-form__title {
  font-size: 12px;
  font-weight: 600;
  color: #d1d1d6;
}

.event-form :deep(.arco-input-wrapper),
.event-form :deep(.arco-textarea-wrapper) {
  background: #2c2c2e !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
}

.event-form :deep(.arco-input-wrapper:hover),
.event-form :deep(.arco-textarea-wrapper:hover) {
  border-color: rgba(0, 122, 255, 0.45) !important;
}

.event-form :deep(.arco-input),
.event-form :deep(.arco-textarea) {
  color: #f2f2f7;
  background: transparent !important;
}

.event-form :deep(.arco-input::placeholder),
.event-form :deep(.arco-textarea::placeholder) {
  color: #8e8e93;
}

.event-form__upload {
  margin-top: 2px;
}

.event-form__check :deep(.arco-checkbox-label) {
  color: #d1d1d6;
}

.platform-tagbar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.platform-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 6px 3px 4px;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: #2c2c2e;
  color: #d1d1d6;
  font-size: 12px;
  cursor: pointer;
  transition:
    background 0.15s ease,
    color 0.15s ease,
    border-color 0.15s ease;
}

.platform-pill:hover {
  background: #3a3a3c;
  color: #ffffff;
}

.platform-pill__del {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.12);
  color: #aeaeb2;
  cursor: pointer;
  transition: all 0.15s ease;
}

.platform-pill__del:hover {
  background: #ff453a;
  color: #ffffff;
}

.platform-pill--empty {
  padding: 5px 12px;
  color: #8e8e93;
  border-style: dashed;
}

.content-form-bar {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.content-form-bar__label {
  font-size: 12px;
  color: #86909c;
  white-space: nowrap;
  flex-shrink: 0;
  line-height: 24px;
}

.content-form-bar :deep(.arco-checkbox-group) {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
}

.content-form-bar :deep(.arco-checkbox-label) {
  color: #d1d1d6;
}

.combo-hint {
  font-size: 11px;
  color: #7c7c84;
  padding-left: 1px;
}

.combo-hint--over {
  color: #ff6b5e;
  font-weight: 500;
}

.version-select-bar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.version-select-bar__label {
  font-size: 12px;
  color: #86909c;
  white-space: nowrap;
  flex-shrink: 0;
}

.version-select-bar__hint {
  font-size: 12px;
  color: #7c7c84;
}

.version-preview-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.compress-bar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.compress-bar__label {
  font-size: 12px;
  color: #86909c;
  white-space: nowrap;
  flex-shrink: 0;
}

.compress-bar__select {
  width: 150px;
  flex-shrink: 0;
}

.compress-bar__hint {
  font-size: 12px;
  color: #7c7c84;
  white-space: nowrap;
}

.platform-toolbar-select {
  flex: 0 1 auto;
  width: 132px;
  min-width: 0;
}

.platform-toolbar-select :deep(.arco-select-view) {
  background: #2c2c2e;
  border-color: rgba(255, 255, 255, 0.08);
  color: #f2f2f7;
  min-height: 28px;
}

.platform-toolbar-select :deep(.arco-select-view:hover),
.platform-toolbar-select :deep(.arco-select-view.arco-select-view--focus) {
  background: #3a3a3c;
  border-color: rgba(0, 122, 255, 0.45);
}

.platform-toolbar-select :deep(.arco-select-view-value) {
  color: #f2f2f7;
}

.platform-toolbar-select :deep(.arco-tag) {
  background: #48484a;
  border-color: transparent;
  color: #f2f2f7;
}

.platform-toolbar-select :deep(.arco-select-view-placeholder) {
  color: #8e8e93;
}

.platform-toolbar-select :deep(.arco-select-view-icon),
.platform-toolbar-select :deep(.arco-select-view .arco-icon-close) {
  color: #aeaeb2;
}

.platform-opt {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.model-select-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  flex: 0 1 auto;
  min-width: 0;
}

.model-select-wrap :deep(.chat-model-select) {
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

:global(.platform-dropdown) {
  background: #2c2c2e;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.5),
    0 2px 8px rgba(0, 0, 0, 0.3);
  border-radius: 12px;
}

:global(.platform-dropdown .arco-select-option) {
  background: #2c2c2e;
  color: #f2f2f7;
}

:global(.platform-dropdown .arco-select-option:hover) {
  background: #3a3a3c;
  color: #ffffff;
}

:global(.platform-dropdown .arco-select-option-selected),
:global(.platform-dropdown .arco-select-option-active) {
  background: #3a3a3c;
  color: #ffffff;
}

:global(.platform-dropdown .platform-opt .arco-avatar) {
  width: 18px;
  height: 18px;
  font-size: 8px;
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

.campaign-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 420px;
}

.campaign-picker__filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.campaign-picker__search {
  width: 240px;
}

.campaign-picker__platform {
  width: 140px;
}

.campaign-picker__status {
  width: 120px;
}

.campaign-picker__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 320px;
  overflow-y: auto;
  padding-right: 2px;
}

.campaign-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid #e5e6eb;
  border-radius: 10px;
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    background 0.15s ease,
    box-shadow 0.15s ease;
}

.campaign-row:hover {
  background: #f7f8fa;
  border-color: rgb(var(--primary-6));
}

.campaign-row--selected {
  border-color: rgb(var(--primary-6));
  background: rgb(var(--primary-1));
  box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.1);
}

.campaign-row__radio {
  flex: 0 0 auto;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1px solid #c9cdd4;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.15s ease;
}

.campaign-row--selected .campaign-row__radio {
  border-color: rgb(var(--primary-6));
}

.campaign-row__radio-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgb(var(--primary-6));
}

.campaign-row__media {
  flex: 0 0 auto;
  border-radius: 8px;
  overflow: hidden;
}

.campaign-row__media--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: #f2f3f5;
  color: #c9cdd4;
}

.campaign-row__info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.campaign-row__head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.campaign-row__name {
  font-size: 13px;
  font-weight: 600;
  color: #1d2129;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.campaign-row__loc {
  font-size: 12px;
  color: #86909c;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 120px;
  flex-shrink: 0;
}

.campaign-row__status {
  flex-shrink: 0;
}

.campaign-row__platforms {
  display: flex;
  align-items: center;
  gap: 6px;
}

.campaign-row__desc {
  font-size: 12px;
  color: #86909c;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-all;
}

.campaign-row__time {
  flex: 0 0 auto;
  font-size: 11px;
  color: #a9aeb8;
}

.campaign-picker__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px 0;
}

.campaign-picker__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
  border-top: 1px solid #f2f3f5;
  padding-top: 12px;
}

.campaign-picker__footer-actions {
  display: flex;
  gap: 8px;
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
    .content-left {
      max-width: 100%;
    }
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
  .campaign-picker__search,
  .campaign-picker__platform,
  .campaign-picker__status {
    width: 100%;
  }
}

.material-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 260px;
}
.material-picker__filters {
  display: flex;
}
.material-picker__search {
  width: 100%;
}
.material-picker__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 180px;
  color: #86909c;
  font-size: 13px;
}
.material-picker__grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}
.material-picker__card {
  position: relative;
  display: flex;
  flex-direction: column;
  border: 1px solid #e5e6eb;
  border-radius: 8px;
  overflow: hidden;
  background: #f7f8fa;
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    box-shadow 0.15s ease;
}
.material-picker__card:hover {
  border-color: rgb(var(--primary-6));
}
.material-picker__card--active {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 2px rgba(var(--primary-6), 0.2);
}
.material-picker__img {
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
  display: block;
  background: #f2f3f5;
}
.material-picker__fileicon {
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 4 / 3;
  color: #86909c;
  background: #f2f3f5;
}
.material-picker__meta {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 8px 10px;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.45));
  pointer-events: none;
}
.material-picker__name {
  display: block;
  color: #ffffff;
  font-size: 12px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.material-picker__check {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgb(var(--primary-6));
  color: #ffffff;
}
.material-picker__more {
  display: flex;
  justify-content: center;
}
</style>
