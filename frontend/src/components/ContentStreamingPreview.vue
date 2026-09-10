<template>
  <div v-if="stream.text" class="streaming-view">
    <div class="streaming-view__header">
      <PlatformIcon v-if="stream.platform" :platform="iconType" size="sm" />
      <span v-if="stream.platform" class="text-[13px] font-medium">{{ platformLabelText }}</span>
      <span class="streaming-view__badge">AI 生成中</span>
      <span v-if="isJson" class="streaming-view__badge streaming-view__badge--json">JSON</span>
    </div>
    <div ref="textEl" class="streaming-view__text">
      <div v-if="isJson" class="streaming-view__markdown" v-html="markdownHtml"></div>
      <template v-else>{{ stream.text }}</template>
      <span class="streaming-cursor"></span>
    </div>
  </div>
  <a-spin v-else :loading="true" class="w-full py-10">
    <template #icon><IconStar :size="30" :style="{ color: '#007AFF' }" spin /></template>
    <div class="text-center">
      <p class="text-[13px] text-[#86868B]">正在为 {{ platformCount }} 个平台生成适配文案…</p>
    </div>
  </a-spin>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { IconStar } from '@arco-design/web-vue/es/icon'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import type { PlatformIconType } from '@/components/shared/PlatformIcon.types'
import { isJsonLike, renderStreamingJsonMarkdown } from '@/utils/streamPreview'
import type { ContentStreamingPreviewProps } from './ContentStreamingPreview.types'

// stream 以响应式容器传入，父组件渲染时不读取其内部字段，
// 这样逐 chunk 的更新只重渲本组件，不会带动整页 render
const props = defineProps<ContentStreamingPreviewProps>()

const iconType = computed(() => props.stream.platform as PlatformIconType)

const isJson = computed(() => isJsonLike(props.stream.text))

const markdownHtml = computed(() =>
  isJson.value ? renderStreamingJsonMarkdown(props.stream.text) : '',
)

const textEl = ref<HTMLElement | null>(null)

// 把每个 chunk 的滚动合并到一帧内，避免逐 chunk 触发强制重排
let scrollRaf = 0
watch(
  () => props.stream.text,
  () => {
    if (scrollRaf) return
    scrollRaf = requestAnimationFrame(() => {
      scrollRaf = 0
      const el = textEl.value
      if (el) el.scrollTop = el.scrollHeight
    })
  },
)

onUnmounted(() => cancelAnimationFrame(scrollRaf))
</script>

<style scoped lang="scss">
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

.streaming-view__badge--json {
  color: #5856d6;
  background: rgba(88, 86, 214, 0.1);
}

.streaming-view__markdown {
  white-space: normal;
  font-size: 13px;
  line-height: 1.7;
  color: #1d1d1f;

  :deep(p) {
    margin: 0 0 8px;
  }

  :deep(p:last-child) {
    margin-bottom: 0;
  }

  :deep(blockquote) {
    margin: 0 0 8px;
    padding: 6px 10px;
    border-left: 3px solid #007aff;
    background: rgba(0, 122, 255, 0.06);
    border-radius: 0 6px 6px 0;
    color: #4e5969;

    p {
      margin: 0;
    }
  }

  :deep(p code) {
    font-family: 'SF Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 12px;
    padding: 1px 5px;
    border-radius: 4px;
    background: rgba(88, 86, 214, 0.09);
    color: #5856d6;
  }

  :deep(.md-json) {
    margin-top: 4px;
    border: 1px solid rgba(0, 0, 0, 0.07);
    border-radius: 10px;
    overflow: hidden;
    background: #ffffff;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  }

  :deep(.md-json__bar) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    background: #f7f7f9;
    border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  }

  :deep(.md-json__badge) {
    font-family: 'SF Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.08em;
    color: #5856d6;
    background: rgba(88, 86, 214, 0.1);
    padding: 2px 7px;
    border-radius: 4px;
  }

  :deep(.md-json__live) {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 10.5px;
    color: #007aff;
  }

  :deep(.md-json__dot) {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #007aff;
    animation: json-pulse 1.2s ease-in-out infinite;
  }

  :deep(.md-json__pre) {
    margin: 0;
    padding: 12px 14px;
    max-height: 340px;
    overflow: auto;
    background: #ffffff;
  }

  :deep(.md-json__code) {
    display: block;
    font-family: 'SF Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 12.5px;
    line-height: 1.65;
    white-space: pre;
    tab-size: 2;
    color: #1d1d1f;
  }

  :deep(.tok-key) {
    color: #5856d6;
  }

  :deep(.tok-string) {
    color: #248a3d;
  }

  :deep(.tok-number) {
    color: #c25e00;
  }

  :deep(.tok-boolean) {
    color: #007aff;
  }

  :deep(.tok-null) {
    color: #8e8e93;
  }

  :deep(.tok-punct) {
    color: #a6a6ab;
  }
}

@keyframes json-pulse {
  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.35;
  }
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

@keyframes cursor-blink {
  50% {
    opacity: 0;
  }
}

@media (max-width: 768px) {
  .streaming-view__text {
    max-height: 320px;
    padding: 12px 14px;
    font-size: 13.5px;
    line-height: 1.75;
  }

  .streaming-view__markdown :deep(.md-json__pre) {
    padding: 10px 12px;
    max-height: 260px;
  }

  .streaming-view__markdown :deep(.md-json__code) {
    font-size: 11.5px;
  }
}

@media (max-width: 480px) {
  .streaming-view__text {
    max-height: 260px;
    padding: 10px 12px;
    font-size: 13px;
  }
  .streaming-view__header {
    margin-bottom: 10px;
  }
  .streaming-view__badge {
    font-size: 9.5px;
    padding: 2px 6px;
  }

  .streaming-view__markdown :deep(.md-json__pre) {
    max-height: 200px;
  }

  .streaming-view__markdown :deep(.md-json__code) {
    font-size: 11px;
  }
}
</style>
