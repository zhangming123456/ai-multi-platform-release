export interface PermissionAccess {
  read: boolean
  write: boolean
}

export interface UserInfo {
  id: string
  username: string
  email: string | null
  nickname: string
  role: string
  avatar_url: string | null
  created_at: string
  permissions?: Record<string, PermissionAccess>
}

export interface Account {
  id: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  nickname: string
  avatar: string
  status: 'active' | 'inactive' | 'error'
  followers: number
  createdAt: string
}

export interface Content {
  id: string
  title: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  status: 'draft' | 'review' | 'approved' | 'published'
  content: string
  hashtags: string[]
  createdAt: string
  updatedAt: string
}

export interface PublishTask {
  id: string
  contentId: string
  contentTitle: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  accountId: string
  accountName: string
  status: 'pending' | 'publishing' | 'published' | 'failed'
  scheduledAt: string
  publishedAt: string | null
}

export interface Template {
  id: string
  name: string
  description: string
  platform: 'wechat_mp' | 'xiaohongshu' | 'douyin' | 'wechat_video'
  thumbnail: string
  category: string
  usageCount: number
  createdAt: string
}
