<script setup lang="ts">
defineProps<{
  status:
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
}>()

const statusColorMap: Record<string, string> = {
  active: 'green',
  inactive: 'gray',
  error: 'red',
  pending: 'orange',
  publishing: 'blue',
  published: 'green',
  failed: 'red',
  draft: 'gray',
  review: 'orange',
  approved: 'cyan',
  ready: 'orange',
}

const statusLabelMap: Record<string, string> = {
  active: '正常',
  inactive: '已停用',
  error: '异常',
  pending: '待发布',
  publishing: '发布中',
  published: '已发布',
  failed: '失败',
  draft: '草稿',
  review: '审核中',
  approved: '已通过',
  ready: '待发布',
}
</script>

<template>
  <a-space :size="4" align="center">
    <a-badge
      :status="status === 'publishing' ? 'processing' : undefined"
      :color="statusColorMap[status]"
      dot
    />
    <a-tag :color="statusColorMap[status]" size="small">
      {{ statusLabelMap[status] || status }}
    </a-tag>
  </a-space>
</template>
