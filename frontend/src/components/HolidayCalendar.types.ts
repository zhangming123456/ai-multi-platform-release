export type HolidayKind = 'holiday' | 'workday' | 'festival' | 'solar_term'

export interface HolidayDay {
  date: string
  name: string
  kind: HolidayKind
  rest: boolean
  work: boolean
  labels: string[]
}

export interface HolidayCalendarData {
  source: string
  title: string
  updated_at: string
  start: string
  end: string
  total: number
  days: HolidayDay[]
}

export interface HolidayCalendarProps {
  modelValue?: string | number | Date
  month?: string
  weekStart?: 0 | 1
  showLegend?: boolean
  showSolarTerm?: boolean
  showMonthList?: boolean
  embedded?: boolean
}

export type HolidayCalendarEmits = {
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'month-change', month: string): void
  (e: 'loaded', data: HolidayCalendarData): void
}

export interface HolidayCalendarCell {
  date: string
  day: number
  inMonth: boolean
  isToday: boolean
  isWeekend: boolean
  holiday?: HolidayDay
}

export interface HolidaySelectOption {
  label: string
  value: number
}

export interface HolidayMonthGroup {
  key: string
  start: string
  end: string
  days: number
  name: string
  kind: HolidayKind
}
