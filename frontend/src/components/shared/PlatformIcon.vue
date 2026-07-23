<script setup lang="ts">
defineProps<{
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  size?: 'sm' | 'md' | 'lg'
}>()

const platformConfig: Record<string, { name: string; gradient: string; letter: string }> = {
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
}

const sizeMap = {
  sm: 24,
  md: 32,
  lg: 44,
}
</script>

<template>
  <a-avatar
    :size="sizeMap[size || 'md']"
    :style="{
      background: platformConfig[platform]?.gradient,
      boxShadow: '0 1px 3px rgba(0,0,0,0.16)',
      fontWeight: 600,
      fontSize: size === 'lg' ? '15px' : size === 'sm' ? '10px' : '12px',
    }"
    :title="platformConfig[platform]?.name || platform"
  >
    {{ platformConfig[platform]?.letter || '?' }}
  </a-avatar>
</template>
