export interface UserInfo {
  id: number
  email: string
  nickname: string
  role: string
  avatar_url: string | null
  created_at: string
}

export interface Account {
  id: number
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  nickname: string
  avatar: string
  status: 'active' | 'inactive' | 'error'
  followers: number
  createdAt: string
}

export interface Content {
  id: number
  title: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  status: 'draft' | 'review' | 'approved' | 'published'
  content: string
  hashtags: string[]
  createdAt: string
  updatedAt: string
}

export interface PublishTask {
  id: number
  contentId: number
  contentTitle: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  accountId: number
  accountName: string
  status: 'pending' | 'publishing' | 'published' | 'failed'
  scheduledAt: string
  publishedAt: string | null
}

export interface Template {
  id: number
  name: string
  description: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  thumbnail: string
  category: string
  usageCount: number
  createdAt: string
}
