<template>
  <div class="page-main">
    <PageHeader
      :title="isEdit ? '编辑活动' : '新增活动'"
      subtitle="维护创作内容关联的活动背景，AI 创作时自动注入活动信息"
    >
      <template #actions>
        <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="goBack">
          <template #icon><IconLeft :size="13" /></template>
          返回列表
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1 pb-24">
      <a-spin :loading="loading" class="w-full">
        <a-card :bordered="false" class="!rounded-xl max-w-[720px]">
          <a-form :model="form" layout="vertical">
            <a-form-item label="活动名称" required>
              <a-input
                v-model="form.name"
                placeholder="如：618 年中大促"
                maxlength="200"
                show-word-limit
              />
            </a-form-item>
            <a-form-item label="活动介绍与宣发图" class="!mb-3">
              <AttachmentInputArea
                v-model="form.description"
                v-model:file-list="form.mediaUrls"
                :file-types="['image']"
                :max-count="9"
                :min-rows="3"
                :max-rows="6"
                placeholder="活动介绍将作为 AI 创作的背景信息注入…"
              />
            </a-form-item>
            <a-form-item label="活动地点">
              <a-input
                v-model="form.location"
                placeholder="如：全国门店 + 线上商城"
                maxlength="200"
              />
            </a-form-item>
            <a-form-item label="面向平台">
              <a-checkbox-group v-model="form.platforms">
                <a-checkbox v-for="p in platformChoices" :key="p.value" :value="p.value">
                  <span class="flex items-center gap-1.5">
                    <PlatformIcon :platform="p.value" />
                    {{ p.label }}
                  </span>
                </a-checkbox>
              </a-checkbox-group>
            </a-form-item>
          </a-form>
        </a-card>
      </a-spin>
    </div>

    <div
      class="sticky bottom-0 z-30 border-t border-[#E5E5EA] bg-white/90 backdrop-blur-xl px-4 md:px-6 lg:px-8 py-3 flex items-center justify-between"
    >
      <a-button @click="goBack">取消</a-button>
      <a-button type="primary" :loading="saving" @click="handleSave">
        <template #icon><IconCheck /></template>
        保存
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconLeft, IconCheck } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import type { PlatformIconType } from '@/components/shared/PlatformIcon.types'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import type { Campaign } from '@/types'
import api, { getApiErrorDetail } from '@/utils/api'

const route = useRoute()
const router = useRouter()

const isEdit = computed(() => !!route.params.id)
const campaignId = computed(() => (route.params.id as string) || '')

const loading = ref(false)
const saving = ref(false)

const platformChoices: { value: PlatformIconType; label: string }[] = [
  { value: 'wechat_mp', label: '公众号' },
  { value: 'xiaohongshu', label: '小红书' },
  { value: 'douyin', label: '抖音' },
  { value: 'wechat_video', label: '视频号' },
  { value: 'wechat_moments', label: '朋友圈' },
  { value: 'weibo', label: '微博' },
]

const form = ref({
  name: '',
  description: '',
  mediaUrls: [] as string[],
  location: '',
  platforms: [] as string[],
})

function parseRawList(raw: string): string[] {
  try {
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? (arr as string[]) : []
  } catch {
    return []
  }
}

function parseMedia(raw: string | string[] | null | undefined): string[] {
  if (!raw) return []
  return Array.isArray(raw) ? raw : parseRawList(raw)
}

function parsePlatforms(raw: string | string[] | null | undefined): PlatformIconType[] {
  if (!raw) return []
  const arr = Array.isArray(raw) ? raw : parseRawList(raw)
  return arr.filter((x): x is PlatformIconType => platformChoices.some((p) => p.value === x))
}

async function uploadDataUrls(urls: string[]): Promise<string[]> {
  const out: string[] = []
  for (const u of urls) {
    if (u.startsWith('data:')) {
      try {
        const blob = await (await fetch(u)).blob()
        const file = new File([blob], 'campaign-media.png', {
          type: blob.type || 'image/png',
        })
        const fd = new FormData()
        fd.append('file', file, file.name)
        const res = await api.post('/uploads', fd)
        const url = res.data?.url || ''
        if (url) out.push(url)
      } catch {
        Message.error('宣发图上传失败，请重试')
        return out
      }
    } else {
      out.push(u)
    }
  }
  return out
}

async function fetchCampaign() {
  loading.value = true
  try {
    const res = await api.get<Campaign>(`/campaigns/${campaignId.value}`)
    const c = res.data
    form.value = {
      name: c.name,
      description: c.description || '',
      mediaUrls: parseMedia(c.media_urls),
      location: c.location || '',
      platforms: parsePlatforms(c.platforms),
    }
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载活动失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!form.value.name.trim()) {
    Message.warning('请填写活动名称')
    return
  }
  const mediaUrls = await uploadDataUrls(form.value.mediaUrls)
  const payload = {
    name: form.value.name.trim(),
    description: form.value.description,
    media_urls: mediaUrls,
    location: form.value.location,
    platforms: form.value.platforms,
  }
  saving.value = true
  try {
    if (isEdit.value) {
      await api.put(`/campaigns/${campaignId.value}`, payload)
      Message.success('更新成功')
    } else {
      await api.post('/campaigns/', payload)
      Message.success('创建成功')
    }
    router.push({ name: 'CampaignManage' })
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '保存失败')
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push({ name: 'CampaignManage' })
}

onMounted(() => {
  if (isEdit.value) fetchCampaign()
})
</script>
