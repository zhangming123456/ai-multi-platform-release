<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconStar, IconCopy, IconSave, IconSettings } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import { useTokenPlanStore } from '@/stores/tokenPlan'
import api from '@/utils/api'

const router = useRouter()
const store = useTokenPlanStore()

const topic = ref('')
const keywords = ref('')
const selectedPlatforms = ref<string[]>(['wechat_mp', 'xiaohongshu'])
const isGenerating = ref(false)
const isSaving = ref(false)
const hasGenerated = ref(false)
const activePreview = ref('')

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
  if (!store.activePlan) {
    router.push('/settings/token-plan')
    return
  }
  if (!topic.value) return

  isGenerating.value = true
  hasGenerated.value = false

  try {
    const keywordsArray = keywords.value
      ? keywords.value
          .split(',')
          .map((k) => k.trim())
          .filter(Boolean)
      : undefined

    const planId = store.activePlan.id

    const requests = selectedPlatforms.value.map((platform) =>
      api.post('/contents/ai-generate', {
        topic: topic.value,
        platform,
        count: 1,
        plan_id: planId,
        ...(keywordsArray && keywordsArray.length > 0 ? { keywords: keywordsArray } : {}),
      }),
    )

    const responses = await Promise.all(requests)

    const variants: Record<string, { title: string; body: string; hashtags: string[] }> = {}
    responses.forEach((res, idx) => {
      const platform = selectedPlatforms.value[idx]
      const variant = res.data?.variants?.[0]
      if (variant) {
        variants[platform] = {
          title: variant.title || '',
          body: variant.body || '',
          hashtags: Array.isArray(variant.hashtags) ? variant.hashtags : [],
        }
      }
    })

    if (Object.keys(variants).length === 0) {
      Message.error('未获取到生成结果，请重试')
      return
    }

    generatedVariants.value = variants
    activePreview.value = selectedPlatforms.value[0]
    hasGenerated.value = true
  } catch (error: unknown) {
    const err = error as { response?: { status?: number; data?: { detail?: unknown } }; message?: string }
    const status = err.response?.status
    const detail = err.response?.data?.detail

    if (status === 502 && detail && typeof detail === 'object') {
      const detailObj = detail as { message?: string; available_plans?: Array<{ id: string; name: string; display_name?: string; provider: string; model: string }> }
      Message.error(detailObj.message || 'AI 生成失败')
      if (Array.isArray(detailObj.available_plans) && detailObj.available_plans.length > 0) {
        const planNames = detailObj.available_plans
          .map((p) => `${p.display_name || p.name}（${p.provider}/${p.model}）`)
          .join('、')
        Message.warning(`检测到当前模型不可用，可前往设置切换至：${planNames}`)
      }
    } else if (detail && typeof detail === 'object') {
      const detailObj = detail as { message?: string }
      Message.error(detailObj.message || 'AI 生成失败，请重试')
    } else if (typeof detail === 'string') {
      Message.error(detail)
    } else {
      Message.error(err.message || 'AI 生成失败，请重试')
    }
  } finally {
    isGenerating.value = false
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
    Message.success('内容已保存为草稿')
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
  <div>
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

    <a-row :gutter="24">
      <a-col :xs="24" :sm="24" :lg="10">
        <a-card :bordered="false" title="创作输入" style="padding: 20px">
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
                            choice.value as 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
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
      </a-col>

      <a-col :xs="24" :sm="24" :lg="14">
        <a-card :bordered="false" title="生成预览" style="padding: 20px">
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
              <span class="text-[12px]">填写左侧主题与平台，点击生成按钮</span>
            </template>
          </a-empty>

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
                :auto-size="{ minRows: 4 }"
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
      </a-col>
    </a-row>
  </div>
</template>
