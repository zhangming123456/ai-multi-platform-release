<template>
  <div class="page-main">
    <PageHeader
      title="日历订阅源"
      subtitle="管理节假日日历的 ICS 订阅源，支持多源合并、自动刷新与手动拉取，数据落库并记录更新时间"
    >
      <template #actions>
        <a-space :size="8">
          <a-button size="mini" :loading="refreshingAll" @click="handleRefreshAll">
            <template #icon><IconRefresh :size="13" /></template>
            全部刷新
          </a-button>
          <a-button
            v-perm="'holiday_source:write'"
            type="text"
            size="mini"
            class="!text-[#007AFF] !px-0 !h-auto"
            @click="openCreate"
          >
            <template #icon><IconPlus :size="13" /></template>
            新增订阅源
          </a-button>
        </a-space>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <div
        class="mb-4 px-4 py-3 rounded-xl bg-[#007AFF]/[0.06] border border-[#007AFF]/[0.12] text-[12px] leading-relaxed text-[#3A3A3C]"
      >
        启用（勾选）多个订阅源后，仪表盘日历会合并展示所有已启用来源的事件；同一日期来自多个来源的名称会合并标注（如
        「国庆节 / 秋分」）。自动刷新支持：手动、每小时、每天、每周、每月、每年。
      </div>

      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <a-table
          :columns="columns"
          :data="sources"
          :bordered="false"
          :hoverable="true"
          :pagination="false"
        >
          <template #enabled="{ record }">
            <a-switch
              :model-value="record.enabled"
              size="small"
              :disabled="!canWrite"
              @change="(value: string | number | boolean) => toggleEnabled(record, Boolean(value))"
            />
          </template>

          <template #source="{ record }">
            <div class="flex items-center gap-2 min-w-0">
              <span
                class="inline-block w-2.5 h-2.5 rounded-full shrink-0 border border-black/10"
                :style="{ backgroundColor: record.color }"
              />
              <div class="flex flex-col gap-0.5 min-w-0">
                <span class="font-medium text-[13px] text-[#1D1D1F] truncate max-w-[320px]">
                  {{ record.name }}
                </span>
                <a-typography-text
                  class="text-[11px] text-[#86868B]"
                  style="max-width: 360px"
                  :ellipsis="{ showTooltip: true }"
                  copyable
                >
                  {{ record.url }}
                </a-typography-text>
              </div>
            </div>
          </template>

          <template #interval="{ record }">
            <a-tag size="small" color="arcoblue">
              {{ intervalLabel(record.refresh_interval) }}
            </a-tag>
          </template>

          <template #refreshedAt="{ record }">
            <div class="flex flex-col gap-0.5">
              <span class="text-[13px] text-[#1D1D1F]">
                {{ formatDateTime(record.last_refreshed_at) }}
              </span>
              <span class="text-[11px] text-[#86868B]">
                {{ formatRelativeTime(record.last_refreshed_at) }}
              </span>
            </div>
          </template>

          <template #eventCount="{ record }">
            <span class="text-[13px] text-[#86868B]">{{ record.event_count }}</span>
          </template>

          <template #status="{ record }">
            <a-tooltip
              v-if="record.last_status === 'failed' && record.last_error"
              :content="record.last_error"
            >
              <a-tag color="red" size="small">拉取失败</a-tag>
            </a-tooltip>
            <a-tag v-else-if="!record.enabled" color="gray" size="small">已停用</a-tag>
            <a-tag v-else-if="record.last_status === 'success'" color="green" size="small">
              正常
            </a-tag>
            <a-tag v-else color="orange" size="small">未同步</a-tag>
          </template>

          <template #actions="{ record }">
            <a-space :size="2">
              <a-tooltip content="立即刷新">
                <a-button
                  type="text"
                  size="small"
                  :loading="isRefreshing(record.id)"
                  :disabled="!canWrite"
                  @click="refreshOne(record)"
                >
                  <template #icon><IconRefresh /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip content="编辑">
                <a-button type="text" size="small" :disabled="!canWrite" @click="openEdit(record)">
                  <template #icon><IconEdit /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除">
                <a-button
                  type="text"
                  size="small"
                  status="danger"
                  :disabled="!canWrite"
                  @click="handleDelete(record)"
                >
                  <template #icon><IconDelete /></template>
                </a-button>
              </a-tooltip>
            </a-space>
          </template>

          <template #empty>
            <a-empty description="暂无订阅源" />
          </template>
        </a-table>
      </a-spin>
    </div>

    <a-modal
      v-model:visible="modalVisible"
      :title="editingId ? '编辑订阅源' : '新增订阅源'"
      :width="520"
      :on-before-ok="handleBeforeOk"
      modal-class="holiday-source-modal"
      ok-text="好"
      cancel-text="取消"
      @cancel="modalVisible = false"
    >
      <div class="source-form">
        <div class="source-row">
          <span class="source-label">名称</span>
          <div class="flex items-center gap-2 flex-1 min-w-0">
            <a-input v-model="form.name" placeholder="如 中国大陆节假日（iCloud）" />
            <a-popover
              trigger="click"
              position="br"
              :content-style="{ padding: '10px' }"
              :unmount-on-close="false"
            >
              <button
                type="button"
                class="source-swatch"
                :style="{ backgroundColor: form.color }"
                title="选择颜色"
              />
              <template #content>
                <div class="source-palette">
                  <button
                    v-for="color in SOURCE_COLOR_PRESETS"
                    :key="color"
                    type="button"
                    class="source-palette-dot"
                    :class="{ 'is-active': color.toUpperCase() === form.color.toUpperCase() }"
                    :style="{ backgroundColor: color }"
                    @click="form.color = color"
                  />
                </div>
              </template>
            </a-popover>
          </div>
        </div>

        <div class="source-row">
          <span class="source-label">订阅</span>
          <div class="flex-1 min-w-0">
            <a-input
              v-model="form.url"
              placeholder="https://calendars.icloud.com/holidays/cn_zh.ics"
            />
          </div>
        </div>

        <div class="source-row">
          <span class="source-label">自动刷新</span>
          <div class="flex-1 min-w-0">
            <a-select v-model="form.refresh_interval" :options="intervalChoices" />
          </div>
        </div>

        <div v-if="editingId" class="source-row">
          <span class="source-label">上次更新</span>
          <span class="source-value">{{ editingRefreshedAt }}</span>
        </div>

        <div class="source-row">
          <span class="source-label">启用</span>
          <div class="flex items-center gap-2 flex-1 min-w-0">
            <a-switch v-model="form.enabled" />
            <span class="text-[12px] text-[#86868B]">{{ form.enabled ? '启用' : '停用' }}</span>
            <span class="text-[11px] text-[#C7C7CC]">启用的订阅源会合并展示在仪表盘日历中</span>
          </div>
        </div>
      </div>

      <p class="source-tip mt-3 mb-0">订阅地址保存时会先试拉取校验，无法解析出事件将不予保存</p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete, IconRefresh } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api, { getApiErrorDetail } from '@/utils/api'
import { formatDateTime, formatRelativeTime } from '@/utils/time'
import { usePermissionStore } from '@/stores/permission'
import type {
  HolidayRefreshInterval,
  HolidayRefreshOption,
  HolidaySourceFormValue,
  HolidaySourceItem,
  HolidaySourceListResponse,
  HolidaySourceSavePayload,
} from './HolidaySources.types'

const FALLBACK_INTERVALS: HolidayRefreshOption[] = [
  { value: 'manual', label: '手动' },
  { value: 'hourly', label: '每小时' },
  { value: 'daily', label: '每天' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' },
  { value: 'yearly', label: '每年' },
]

const DEFAULT_SOURCE_COLOR = '#007AFF'

const SOURCE_COLOR_PRESETS = [
  '#007AFF',
  '#FF3B30',
  '#FF9500',
  '#FFCC00',
  '#34C759',
  '#00C7BE',
  '#5856D6',
  '#FF2D55',
  '#AF52DE',
  '#8E8E93',
]

const DEFAULT_FORM: HolidaySourceFormValue = {
  name: '',
  color: DEFAULT_SOURCE_COLOR,
  url: '',
  enabled: true,
  refresh_interval: 'daily',
}

const columns = [
  { title: '启用', dataIndex: 'enabled', slotName: 'enabled', width: 70 },
  { title: '订阅源', dataIndex: 'name', slotName: 'source', width: 320 },
  { title: '自动刷新', dataIndex: 'refresh_interval', slotName: 'interval', width: 110 },
  { title: '最近更新', dataIndex: 'last_refreshed_at', slotName: 'refreshedAt', width: 170 },
  { title: '事件数', dataIndex: 'event_count', slotName: 'eventCount', width: 90 },
  { title: '状态', dataIndex: 'last_status', slotName: 'status', width: 110 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 150 },
]

const permStore = usePermissionStore()

const loading = ref(false)
const refreshingAll = ref(false)
const modalVisible = ref(false)
const editingId = ref<string | null>(null)
const sources = ref<HolidaySourceItem[]>([])
const intervalOptions = ref<HolidayRefreshOption[]>([])
const refreshingIds = ref<string[]>([])
const form = ref<HolidaySourceFormValue>({ ...DEFAULT_FORM })
const editingRefreshedAt = ref('')

const canWrite = computed(() => permStore.hasPermission('holiday_source:write'))

const intervalChoices = computed(() =>
  intervalOptions.value.length ? intervalOptions.value : FALLBACK_INTERVALS,
)

const intervalLabelMap = computed(
  () => new Map(intervalChoices.value.map((option) => [option.value, option.label])),
)

function intervalLabel(value: HolidayRefreshInterval): string {
  return intervalLabelMap.value.get(value) ?? value
}

function isRefreshing(id: string): boolean {
  return refreshingIds.value.includes(id)
}

async function fetchSources(): Promise<void> {
  loading.value = true
  try {
    const { data } = await api.get<HolidaySourceListResponse>('/holiday-sources')
    sources.value = data.items ?? []
    intervalOptions.value = data.interval_options ?? []
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '加载订阅源失败')
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editingId.value = null
  form.value = { ...DEFAULT_FORM }
  editingRefreshedAt.value = ''
  modalVisible.value = true
}

function openEdit(record: HolidaySourceItem): void {
  editingId.value = record.id
  form.value = {
    name: record.name,
    color: record.color || DEFAULT_SOURCE_COLOR,
    url: record.url,
    enabled: record.enabled,
    refresh_interval: record.refresh_interval,
  }
  editingRefreshedAt.value = record.last_refreshed_at
    ? `${formatDateTime(record.last_refreshed_at)}（${formatRelativeTime(record.last_refreshed_at)}）`
    : '尚未同步'
  modalVisible.value = true
}

async function handleBeforeOk(): Promise<boolean> {
  const name = form.value.name.trim()
  const url = form.value.url.trim()
  if (!name) {
    Message.warning('请填写订阅源名称')
    return false
  }
  if (!url) {
    Message.warning('请填写订阅地址')
    return false
  }

  const payload: HolidaySourceSavePayload = {
    name,
    color: form.value.color,
    url,
    enabled: form.value.enabled,
    refresh_interval: form.value.refresh_interval,
  }

  try {
    if (editingId.value) {
      await api.put(`/holiday-sources/${editingId.value}`, payload)
      Message.success('保存成功')
    } else {
      await api.post('/holiday-sources', payload)
      Message.success('新增成功，已完成首次拉取')
    }
    await fetchSources()
    return true
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '保存失败')
    return false
  }
}

async function toggleEnabled(record: HolidaySourceItem, value: boolean): Promise<void> {
  const previous = record.enabled
  record.enabled = value
  try {
    await api.put(`/holiday-sources/${record.id}`, {
      name: record.name,
      color: record.color,
      url: record.url,
      enabled: value,
      refresh_interval: record.refresh_interval,
    })
    Message.success(value ? '已启用' : '已停用')
  } catch (error) {
    record.enabled = previous
    Message.error(getApiErrorDetail(error) || '切换失败')
  }
}

async function refreshOne(record: HolidaySourceItem): Promise<void> {
  if (isRefreshing(record.id)) return
  refreshingIds.value = [...refreshingIds.value, record.id]
  try {
    await api.post(`/holiday-sources/${record.id}/refresh`)
    Message.success('刷新完成')
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '刷新失败')
  } finally {
    refreshingIds.value = refreshingIds.value.filter((id) => id !== record.id)
    await fetchSources()
  }
}

async function handleRefreshAll(): Promise<void> {
  refreshingAll.value = true
  try {
    const { data } = await api.post<{ refreshed: number }>('/holiday-sources/refresh-all')
    Message.success(`已刷新 ${data.refreshed} 个订阅源`)
    await fetchSources()
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '全部刷新失败')
  } finally {
    refreshingAll.value = false
  }
}

function handleDelete(record: HolidaySourceItem): void {
  Modal.warning({
    title: '确认删除',
    content: `确定删除订阅源「${record.name}」吗？其已拉取的事件缓存会一并清除。`,
    hideCancel: false,
    okText: '确认删除',
    cancelText: '取消',
    onOk: async () => {
      try {
        await api.delete(`/holiday-sources/${record.id}`)
        Message.success('删除成功')
        await fetchSources()
      } catch (error) {
        Message.error(getApiErrorDetail(error) || '删除失败')
      }
    },
  })
}

onMounted(() => {
  void fetchSources()
})
</script>

<style scoped>
.source-form {
  border: 1px solid #e5e5ea;
  border-radius: 10px;
  overflow: hidden;
  background: #ffffff;
}

.source-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  padding: 8px 14px;
  border-bottom: 1px solid #e5e5ea;
}

.source-row:last-child {
  border-bottom: none;
}

.source-label {
  flex: 0 0 64px;
  font-size: 13px;
  color: #1d1d1f;
}

.source-value {
  font-size: 13px;
  color: #86868b;
}

.source-swatch {
  flex: 0 0 22px;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.15s ease;
}

.source-swatch:hover {
  transform: scale(1.08);
}

.source-palette {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 8px;
}

.source-palette-dot {
  width: 18px;
  height: 18px;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08);
  transition: transform 0.15s ease;
}

.source-palette-dot:hover {
  transform: scale(1.12);
}

.source-palette-dot.is-active {
  border-color: #ffffff;
  box-shadow: 0 0 0 2px #007aff;
}

.source-tip {
  font-size: 12px;
  line-height: 1.6;
  color: #86868b;
}
</style>

<style>
.holiday-source-modal .arco-modal-footer .arco-btn-primary {
  background-color: #007aff;
  border-color: #007aff;
}

.holiday-source-modal .arco-modal-footer .arco-btn-primary:hover {
  background-color: #0071e3;
  border-color: #0071e3;
}
</style>
