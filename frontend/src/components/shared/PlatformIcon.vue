<template>
  <a-avatar
    class="platform-icon"
    :style="{
      background: platformConfig[props.platform]?.gradient,
      boxShadow: '0 1px 3px rgba(0,0,0,0.16)',
      fontWeight: 600,
      width: platformIconSize,
      height: platformIconSize,
      fontSize: platformIconSize,
    }"
    :title="platformConfig[platform]?.name || platform"
  >
    <span class="platform-icon-text">{{ platformConfig[platform]?.letter || '?' }}</span>
  </a-avatar>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { PlatformIconType } from '@/components/shared/PlatformIcon.ts'
import { isNumber } from 'lodash-es'

type PlatformIconProps = {
  platform: PlatformIconType
  size?: 'sm' | 'md' | 'lg' | number | `${string}px`
}

const props = withDefaults(defineProps<PlatformIconProps>(), {
  size: 'sm',
})
const platformIconSize = computed(() => {
  switch (props.size) {
    case 'sm':
      return `10px`
    case 'md':
      return `12px`
    case 'lg':
      return `15px`
  }
  if (isNumber(props.size)) {
    return `${props.size}px`
  }
  return props.size
})

const platformConfig = computed<
  Record<PlatformIconType, { name: string; gradient: string; letter: string }>
>(() => {
  return {
    wechat_mp: {
      name: '微信公众号',
      gradient: 'linear-gradient(135deg, #2DC100 0%, #07C160 100%)',
      letter: '微',
    },
    xiaohongshu: {
      name: '小红书',
      gradient: 'linear-gradient(135deg, #FF5A6E 0%, #E6002D 100%)',
      letter: '红',
    },
    douyin: {
      name: '抖音',
      gradient: 'linear-gradient(135deg, #3A465C 0%, #161823 100%)',
      letter: '抖',
    },
    wechat_video: {
      name: '视频号',
      gradient: 'linear-gradient(135deg, #FA9D3B 0%, #FA5151 100%)',
      letter: '视',
    },
    wechat_moments: {
      name: '朋友圈',
      gradient: 'linear-gradient(135deg, #F5E0A9 0%, #E8B554 100%)',
      letter: '圈',
    },
    weibo: {
      name: '微博',
      gradient: 'linear-gradient(135deg, #FF7A45 0%, #E6162D 100%)',
      letter: '微',
    },
  }
})
</script>

<style lang="scss" scoped>
.platform-icon {
  :deep(.arco-avatar) {
    line-height: 0;
  }
  :deep(.arco-avatar-text) {
    line-height: 0;
    width: 0;
    height: 0;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%) scale(1) !important;
  }
  .platform-icon-text {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%) scale(0.6);
  }
}
</style>
