import { ref } from 'vue'
import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(utc)
dayjs.extend(timezone)
dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

export const BACKEND_TZ = 'Asia/Shanghai'

export interface RegionOption {
  value: string
  label: string
  offset: string
}

export const REGION_OPTIONS: RegionOption[] = [
  { value: 'Asia/Shanghai', label: '中国大陆', offset: 'UTC+8' },
  { value: 'Asia/Taipei', label: '中国台湾', offset: 'UTC+8' },
  { value: 'Asia/Hong_Kong', label: '中国香港', offset: 'UTC+8' },
  { value: 'Asia/Macau', label: '中国澳门', offset: 'UTC+8' },
  { value: 'Asia/Tokyo', label: '日本', offset: 'UTC+9' },
  { value: 'Asia/Seoul', label: '韩国', offset: 'UTC+9' },
  { value: 'Asia/Singapore', label: '新加坡', offset: 'UTC+8' },
  { value: 'America/New_York', label: '美国东部', offset: 'UTC-5' },
  { value: 'America/Los_Angeles', label: '美国西部', offset: 'UTC-8' },
  { value: 'Europe/London', label: '英国', offset: 'UTC+0' },
  { value: 'Europe/Paris', label: '法国', offset: 'UTC+1' },
]

export const STORAGE_KEY = 'app_timezone'

function loadStoredTz(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) || BACKEND_TZ
  } catch {
    return BACKEND_TZ
  }
}

export const displayTimezone = ref(loadStoredTz())

export function syncTimezone(value: string): void {
  displayTimezone.value = value
  try {
    localStorage.setItem(STORAGE_KEY, value)
  } catch {
    // ignore
  }
}

export function refreshTimezone(): void {
  displayTimezone.value = loadStoredTz()
}

function toDisplayTime(isoStr: string) {
  return dayjs.tz(isoStr, BACKEND_TZ).tz(displayTimezone.value)
}

export function formatDateTime(isoStr: string | null | undefined): string {
  if (!isoStr) return '--'
  const d = toDisplayTime(isoStr)
  if (!d.isValid()) return isoStr
  return d.format('YYYY-MM-DD HH:mm')
}

export function formatDateTimeSec(isoStr: string | null | undefined): string {
  if (!isoStr) return '--'
  const d = toDisplayTime(isoStr)
  if (!d.isValid()) return isoStr
  return d.format('YYYY-MM-DD HH:mm:ss')
}

export function formatRelativeTime(dateStr: string | null | undefined): string {
  if (!dateStr) return '--'
  const d = toDisplayTime(dateStr)
  if (!d.isValid()) return '--'
  const now = dayjs().tz(displayTimezone.value)
  const diff = now.diff(d, 'second')
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`
  if (diff < 604800) return `${Math.floor(diff / 86400)}天前`
  if (diff < 2592000) return `${Math.floor(diff / 604800)}周前`
  if (diff < 31536000) return `${Math.floor(diff / 2592000)}个月前`
  return d.format('YYYY-MM-DD HH:mm')
}

export function getCurrentTimezoneLabel(): string {
  return REGION_OPTIONS.find((r) => r.value === displayTimezone.value)?.label || displayTimezone.value
}

export { dayjs }
