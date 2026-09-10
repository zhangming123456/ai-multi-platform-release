export type StatusBadgeStatus =
  | 'active'
  | 'inactive'
  | 'error'
  | 'pending'
  | 'publishing'
  | 'published'
  | 'failed'
  | 'draft'
  | 'review'
  | 'approved'
  | 'ready'
  | 'pending_review'
  | 'rejected'
  | 'archived'

export interface StatusBadgeProps {
  status: StatusBadgeStatus
}
