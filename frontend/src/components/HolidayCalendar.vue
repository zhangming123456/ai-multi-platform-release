<template>
  <div class="holiday-calendar" :class="{ 'is-embedded': embedded }">
    <div class="holiday-calendar__header">
      <div class="holiday-calendar__left min-w-2/12" />
      <div class="holiday-calendar__nav">
        <a-button size="mini" shape="circle" @click="shiftMonth(-1)">
          <template #icon><IconLeft :size="12" /></template>
        </a-button>
        <a-select
          v-model="yearValue"
          size="mini"
          class="holiday-calendar__select"
          :options="yearOptions"
        />
        <a-select
          v-model="monthValue"
          size="mini"
          class="holiday-calendar__select holiday-calendar__select--month"
          :options="monthOptions"
        />
        <a-button size="mini" shape="circle" @click="shiftMonth(1)">
          <template #icon><IconRight :size="12" /></template>
        </a-button>
      </div>
      <div class="holiday-calendar__right min-w-2/12 flex justify-end">
        <a-button size="mini" @click="goToday">今天</a-button>
      </div>
    </div>

    <a-spin :loading="loading" class="holiday-calendar__body">
      <div class="holiday-calendar__weekdays">
        <span
          v-for="(label, index) in weekdayLabels"
          :key="label"
          class="holiday-calendar__weekday"
          :class="{ 'is-weekend': isWeekendColumn(index) }"
        >
          {{ label }}
        </span>
      </div>

      <div class="holiday-calendar__grid">
        <button
          v-for="cell in cells"
          :key="cell.date"
          type="button"
          class="holiday-calendar__cell"
          :class="{
            'is-out': !cell.inMonth,
            'is-selected': dayjs(cell.date).isSame(selectedDate),
            'is-weekend': cell.isWeekend,
            'is-rest': cell.holiday?.rest,
            'is-work': cell.holiday?.work,
          }"
          @click="selectDate(cell)"
        >
          <span class="holiday-calendar__day">{{ cell.day }}</span>
          <span
            v-if="cell.holiday && cell.holiday.kind !== 'solar_term'"
            class="holiday-calendar__name"
            :title="cell.holiday.labels.join('、')"
          >
            {{ cell.holiday.name }}
          </span>
          <span v-else-if="cell.holiday && showSolarTerm" class="holiday-calendar__name is-term">
            {{ cell.holiday.name }}
          </span>
          <span v-if="cell.holiday?.rest" class="holiday-calendar__badge is-rest">休</span>
          <span v-else-if="cell.holiday?.work" class="holiday-calendar__badge is-work">班</span>
          <span v-else-if="cell.isToday" class="holiday-calendar__badge is-today">今</span>
        </button>
      </div>
    </a-spin>

    <div v-if="loadError" class="holiday-calendar__error">{{ loadError }}</div>

    <div v-if="showLegend" class="holiday-calendar__legend">
      <span class="holiday-calendar__legend-item">
        <i class="holiday-calendar__dot is-rest"></i>放假
      </span>
      <span class="holiday-calendar__legend-item">
        <i class="holiday-calendar__dot is-work"></i>调休上班
      </span>
      <span class="holiday-calendar__legend-item">
        <i class="holiday-calendar__dot is-festival"></i>节日
      </span>
      <span class="holiday-calendar__legend-item">
        <i class="holiday-calendar__dot is-term"></i>节气
      </span>
      <span v-if="updatedLabel" class="holiday-calendar__updated">{{ updatedLabel }}</span>
    </div>

    <div v-if="showMonthList && monthGroups.length" class="holiday-calendar__list">
      <div class="holiday-calendar__list-head">
        <span class="holiday-calendar__list-title">本月节假日</span>
        <span class="holiday-calendar__list-count">共 {{ monthItems.length }} 天</span>
      </div>
      <div class="holiday-calendar__list-body">
        <div v-for="group in monthGroups" :key="group.key" class="holiday-calendar__list-row">
          <span class="holiday-calendar__list-date">{{
            formatDateRange(group.start, group.end)
          }}</span>
          <span class="holiday-calendar__list-name">{{ group.name }}</span>
          <span class="holiday-calendar__list-kind" :class="kindClass(group.kind)">
            {{ kindLabel(group.kind) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { IconLeft, IconRight } from '@arco-design/web-vue/es/icon'
import api from '@/utils/api'
import type {
  HolidayCalendarCell,
  HolidayCalendarData,
  HolidayCalendarEmits,
  HolidayCalendarProps,
  HolidayDay,
  HolidayKind,
  HolidayMonthGroup,
  HolidaySelectOption,
} from './HolidayCalendar.types'
import dayjs from 'dayjs'

const props = withDefaults(defineProps<HolidayCalendarProps>(), {
  modelValue: Date.now(),
  month: '',
  weekStart: 0,
  showLegend: true,
  showSolarTerm: true,
  showMonthList: true,
  embedded: false,
})

const emit = defineEmits<HolidayCalendarEmits>()

let cachedData: HolidayCalendarData | null = null
let inflight: Promise<HolidayCalendarData> | null = null

const data = ref<HolidayCalendarData | null>(cachedData)
const loading = ref(false)
const loadError = ref('')

const today = formatDate(new Date())
const cursor = ref(props.month || today.slice(0, 7))

const holidayMap = computed(() => {
  const map = new Map<string, HolidayDay>()
  for (const day of data.value?.days ?? []) {
    map.set(day.date, day)
  }
  return map
})

const selectedDate = ref(props.modelValue || '')

watch(
  () => props.modelValue,
  (value, oldValue) => {
    if (value === oldValue) return
    selectedDate.value = value
  },
)

const weekdayLabels = computed(() =>
  props.weekStart === 1
    ? ['一', '二', '三', '四', '五', '六', '日']
    : ['日', '一', '二', '三', '四', '五', '六'],
)

const yearOptions = computed<HolidaySelectOption[]>(() => {
  const currentYear = Number(today.slice(0, 4))
  const years = new Set<number>()
  for (let offset = -3; offset <= 3; offset += 1) {
    years.add(currentYear + offset)
  }
  for (const day of data.value?.days ?? []) {
    years.add(Number(day.date.slice(0, 4)))
  }
  return Array.from(years)
    .filter((year) => year > 1900)
    .sort((a, b) => a - b)
    .map((year) => ({ label: `${year} 年`, value: year }))
})

const monthOptions = computed<HolidaySelectOption[]>(() =>
  Array.from({ length: 12 }, (_, index) => ({ label: `${index + 1} 月`, value: index + 1 })),
)

const yearValue = computed({
  get: () => Number(cursor.value.slice(0, 4)),
  set: (value: number) => setCursor(value, Number(cursor.value.slice(5, 7))),
})

const monthValue = computed({
  get: () => Number(cursor.value.slice(5, 7)),
  set: (value: number) => setCursor(Number(cursor.value.slice(0, 4)), value),
})

const cells = computed<HolidayCalendarCell[]>(() => {
  const [year, month] = cursor.value.split('-').map(Number)
  const first = new Date(year, month - 1, 1)
  const offset = (first.getDay() - props.weekStart + 7) % 7
  const daysInMonth = new Date(year, month, 0).getDate()
  const total = Math.ceil((offset + daysInMonth) / 7) * 7

  const result: HolidayCalendarCell[] = []
  for (let index = 0; index < total; index += 1) {
    const date = new Date(year, month - 1, 1 - offset + index)
    const dateStr = formatDate(date)
    const weekday = date.getDay()
    result.push({
      date: dateStr,
      day: date.getDate(),
      inMonth: date.getMonth() === month - 1,
      isToday: dateStr === today,
      isWeekend: weekday === 0 || weekday === 6,
      holiday: holidayMap.value.get(dateStr),
    })
  }
  return result
})

const monthItems = computed(() =>
  (data.value?.days ?? [])
    .filter((day) => day.date.startsWith(`${cursor.value}-`))
    .slice()
    .sort((a, b) => a.date.localeCompare(b.date)),
)

const monthGroups = computed<HolidayMonthGroup[]>(() => {
  const result: HolidayMonthGroup[] = []
  for (const day of monthItems.value) {
    const last = result[result.length - 1]
    if (last && last.name === day.name && last.kind === day.kind && isNextDay(last.end, day.date)) {
      last.end = day.date
      last.days += 1
      continue
    }
    result.push({
      key: day.date,
      start: day.date,
      end: day.date,
      days: 1,
      name: day.name,
      kind: day.kind,
    })
  }
  return result
})

const updatedLabel = computed(() => {
  const raw = data.value?.updated_at
  return raw ? `数据更新于 ${raw.slice(0, 10)}` : ''
})

function pad(value: number): string {
  return value < 10 ? `0${value}` : String(value)
}

function formatDate(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function formatMonthDay(date: string): string {
  return `${Number(date.slice(5, 7))} 月 ${Number(date.slice(8, 10))} 日`
}

function formatDateRange(start: string, end: string): string {
  if (start === end) {
    return formatMonthDay(start)
  }
  if (start.slice(0, 7) === end.slice(0, 7)) {
    return `${formatMonthDay(start)} – ${Number(end.slice(8, 10))} 日`
  }
  return `${formatMonthDay(start)} – ${formatMonthDay(end)}`
}

function isNextDay(prevDate: string, nextDate: string): boolean {
  const prev = new Date(`${prevDate}T00:00:00`).getTime()
  const next = new Date(`${nextDate}T00:00:00`).getTime()
  return next - prev === 24 * 60 * 60 * 1000
}

function isWeekendColumn(index: number): boolean {
  const weekday = (index + props.weekStart) % 7
  return weekday === 0 || weekday === 6
}

function setCursor(year: number, month: number): void {
  cursor.value = `${year}-${pad(month)}`
}

function shiftMonth(step: number): void {
  const [year, month] = cursor.value.split('-').map(Number)
  const date = new Date(year, month - 1 + step, 1)
  setCursor(date.getFullYear(), date.getMonth() + 1)
}

function goToday(): void {
  cursor.value = today.slice(0, 7)
  selectDateValue(today)
}

function selectDate(cell: HolidayCalendarCell): void {
  selectDateValue(cell.date)
}

function selectDateValue(value: string): void {
  selectedDate.value = value
  emit('update:modelValue', value)
  emit('change', value)
}

function kindLabel(kind: HolidayKind): string {
  switch (kind) {
    case 'holiday':
      return '放假'
    case 'workday':
      return '调休上班'
    case 'solar_term':
      return '节气'
    default:
      return '节日'
  }
}

function kindClass(kind: HolidayKind): string {
  return `is-${kind}`
}

async function loadHolidays(): Promise<void> {
  if (cachedData) {
    data.value = cachedData
    emit('loaded', cachedData)
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    if (!inflight) {
      inflight = api.get<HolidayCalendarData>('/holidays').then((response) => response.data)
    }
    const result = await inflight
    if ((result.days?.length ?? 0) > 0) {
      cachedData = result
    }
    data.value = result
    emit('loaded', result)
  } catch {
    loadError.value = '节假日数据加载失败，请稍后重试'
  } finally {
    inflight = null
    loading.value = false
  }
}

watch(
  () => props.month,
  (value) => {
    if (value && value !== cursor.value) {
      cursor.value = value
    }
  },
)

watch(cursor, (value) => {
  emit('month-change', value)
})

onMounted(() => {
  void loadHolidays()
})
</script>

<style scoped lang="scss">
.holiday-calendar {
  padding: 16px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 14px;
  background: #ffffff;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.04);
}

.holiday-calendar.is-embedded {
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.holiday-calendar__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  .holiday-calendar__nav {
    display: flex;
    align-items: center;
    gap: 6px;
    :deep(.arco-btn) {
      &.arco-btn-size-mini.arco-btn-shape-circle {
        width: 24px;
        height: 24px;
        min-width: 24px;
        min-height: 24px;
      }
    }
  }
}

.holiday-calendar__select {
  width: 88px;
}

.holiday-calendar__select--month {
  width: 72px;
}

.holiday-calendar__meta {
  font-size: 12px;
  color: #86868b;
}

.holiday-calendar__body {
  display: block;
  width: 100%;
}

.holiday-calendar__weekdays,
.holiday-calendar__grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.holiday-calendar__weekday {
  padding: 6px 0;
  text-align: center;
  font-size: 12px;
  color: #86868b;
}

.holiday-calendar__weekday.is-weekend {
  color: #ff9500;
}

.holiday-calendar__grid {
  gap: 4px;
}

.holiday-calendar__cell {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
  min-height: 58px;
  padding: 6px 4px 5px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: #f7f7f9;
  cursor: pointer;
  transition:
    background 0.15s,
    border-color 0.15s;
}

.holiday-calendar__cell:hover {
  background: rgba(0, 122, 255, 0.12);
}

.holiday-calendar__cell.is-out {
  opacity: 0.4;
}

.holiday-calendar__cell.is-rest {
  background: rgba(255, 59, 48, 0.08);
}

.holiday-calendar__cell.is-work {
  background: rgba(255, 149, 0, 0.1);
}

.holiday-calendar__cell.is-selected {
  border-color: #007aff;
  background: rgba(0, 122, 255, 0.12);
}

.holiday-calendar__day {
  font-size: 13px;
  font-weight: 600;
  line-height: 1.2;
  color: #1d1d1f;
}

.holiday-calendar__cell.is-weekend .holiday-calendar__day {
  color: #ff9500;
}

.holiday-calendar__cell.is-rest .holiday-calendar__day {
  color: #ff3b30;
}

.holiday-calendar__name {
  max-width: 100%;
  font-size: 10.5px;
  line-height: 1.2;
  color: #5856d6;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.holiday-calendar__name.is-term {
  color: #34c759;
}

.holiday-calendar__badge {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 14px;
  height: 14px;
  border-radius: 4px;
  font-size: 9px;
  line-height: 14px;
  text-align: center;
  color: #ffffff;
}

.holiday-calendar__badge.is-rest {
  background: #ff3b30;
}

.holiday-calendar__badge.is-work {
  background: #ff9500;
}

.holiday-calendar__badge.is-today {
  background: #007aff;
}

.holiday-calendar__error {
  margin-top: 10px;
  font-size: 12px;
  color: #ff3b30;
}

.holiday-calendar__legend {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px;
  margin-top: 12px;
  font-size: 11.5px;
  color: #86868b;
}

.holiday-calendar__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.holiday-calendar__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #86868b;
}

.holiday-calendar__dot.is-rest {
  background: #ff3b30;
}

.holiday-calendar__dot.is-work {
  background: #ff9500;
}

.holiday-calendar__dot.is-festival {
  background: #5856d6;
}

.holiday-calendar__dot.is-term {
  background: #34c759;
}

.holiday-calendar__updated {
  margin-left: auto;
}

.holiday-calendar__list {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.holiday-calendar__list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 8px;
}

.holiday-calendar__list-title {
  font-size: 13px;
  font-weight: 600;
  color: #1d1d1f;
}

.holiday-calendar__list-count {
  font-size: 11.5px;
  color: #86868b;
}

.holiday-calendar__list-body {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 6px;
}

.holiday-calendar__list-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 8px;
  background: #f7f7f9;
  font-size: 12px;
}

.holiday-calendar__list-date {
  flex-shrink: 0;
  color: #86868b;
}

.holiday-calendar__list-name {
  flex: 1;
  color: #1d1d1f;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.holiday-calendar__list-kind {
  flex-shrink: 0;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10.5px;
  color: #ffffff;
  background: #86868b;
}

.holiday-calendar__list-kind.is-holiday {
  background: #ff3b30;
}

.holiday-calendar__list-kind.is-workday {
  background: #ff9500;
}

.holiday-calendar__list-kind.is-festival {
  background: #5856d6;
}

.holiday-calendar__list-kind.is-solar_term {
  background: #34c759;
}

@media (max-width: 640px) {
  .holiday-calendar {
    padding: 12px;
  }

  .holiday-calendar__cell {
    min-height: 50px;
  }

  .holiday-calendar__name {
    //display: none;
  }
}
</style>
