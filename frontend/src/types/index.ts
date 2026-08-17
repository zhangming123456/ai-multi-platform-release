export interface UserInfo {
  id: string
  username: string
  email: string | null
  nickname: string
  role: string
  avatar_url: string | null
  created_at: string
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

export interface Store {
  id: string
  name: string
  code: string | null
  address: string | null
  contact: string | null
  phone: string | null
  manager_id: string | null
  status: 'active' | 'inactive'
  created_at: string
  updated_at: string
}

export interface InspectionItem {
  id: string
  name: string
  category: string | null
  max_score: number
  sort_order: number
  is_active: boolean
}

export interface ScoreOption {
  score: number
  label: string
}

export interface InspectionScore {
  item_id: string
  item_name: string
  category: string | null
  standard: string | null
  standard_image: string | null
  score_type: 'score' | 'pass_fail'
  max_score: number
  score_options: ScoreOption[] | null
  score: number
  require_remark: boolean
  require_photo: boolean
  show_remark: boolean
  show_photo: boolean
  comment: string
  ai_generated: boolean
  ai_suggestion?: string
  photos: string[]
}

export interface Inspection {
  id: string
  title: string
  status: 'draft' | 'pending' | 'rectifying' | 'closed'
  store_id: string
  store_name: string
  store_code: string
  store_address: string
  template_id: string | null
  template_name: string | null
  inspector_id: string
  inspector_name: string
  total_score: number
  passed: boolean
  issues: string
  suggestion: string
  ai_summary?: {
    summary: string
    high_risk_problems: { item_id: string; item_name: string; level: string; desc: string }[]
    main_problems: { item_id: string; item_name: string; level: string; desc: string }[]
    priority_suggest: { title: string; desc: string }[]
    business_suggest: { title: string; desc: string }[]
  }
  ai_generated: boolean
  photos: string[]
  scores: InspectionScore[]
  checked_at: string
  created_at: string
  updated_at: string
}

export interface Material {
  id: string
  name: string
  url: string
  type: string
  category: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface InspectionTemplateItem {
  id: string
  category: string | null
  title: string
  standard: string | null
  standard_image: string | null
  standard_images: string[]
  score_type: 'score' | 'pass_fail'
  max_score: number
  score_options: ScoreOption[] | null
  require_remark: boolean
  require_photo: boolean
  show_remark: boolean
  show_photo: boolean
  category_precondition_enabled: boolean
  category_precondition: string | null
}

export interface InspectionMaterial {
  id: string
  category: string | null
  title: string
  standard: string | null
  standard_image: string | null
  standard_images: string[]
  score_type: 'score' | 'pass_fail'
  max_score: number
  score_options: ScoreOption[] | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface InspectionTemplate {
  id: string
  name: string
  description: string | null
  is_active: boolean
  scoring_mode: 'additive' | 'deductive'
  item_count: number
  items?: InspectionTemplateItem[]
  created_at: string
  updated_at: string
}

export interface Paginated<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type InspectionTaskStatus =
  'pending' | 'rechecking' | 'rectified' | 'rectifying' | 'manual_review' | 'confirmed' | 'rejected'

export type InspectionTaskItemStatus =
  'pending' | 'submitted' | 'fixed' | 'not_fixed' | 'manual' | 'confirmed' | 'rejected'

export interface InspectionTask {
  id: string
  inspection_id: string
  store_id: string
  store_name: string
  title: string
  status: InspectionTaskStatus
  inspector_id: string
  inspector_name: string
  responsible_id: string | null
  responsible_name: string | null
  contact: string | null
  phone: string | null
  channel: string
  deadline: string
  closed_at: string
  item_count: number
  fixed_count: number
  overdue: boolean
  created_at: string
  updated_at: string
}

export interface InspectionTaskItem {
  id: string
  task_id: string
  inspection_id: string | null
  score_id: string | null
  item_id: string
  item_name: string
  category: string | null
  standard: string | null
  standard_image: string | null
  score_type: 'score' | 'pass_fail'
  max_score: number
  score: number
  comment: string | null
  ai_suggestion: string | null
  original_photos: string[]
  status: InspectionTaskItemStatus
  rectify_photos: string[]
  rectify_comment: string | null
  recheck_count: number
  ai_result: { fixed: boolean; score: number; reason: string } | null
  submitted_at: string
  rechecked_at: string
}

export interface InspectionTaskLog {
  id: string
  item_id: string | null
  action: string
  content: string
  operator_id: string | null
  operator_name: string | null
  created_at: string
}
