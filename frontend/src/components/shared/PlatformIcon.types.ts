export type PlatformIconType =
  'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video' | 'wechat_moments' | 'weibo'

export interface PlatformIconProps {
  platform: PlatformIconType
  size?: 'sm' | 'md' | 'lg' | number | `${string}px`
}

export interface PlatformConfigItem {
  name: string
  gradient: string
  letter: string
}

export type PlatformConfigMap = Record<PlatformIconType, PlatformConfigItem>
