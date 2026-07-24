import { ref, computed } from 'vue'
import {
  REGION_OPTIONS,
  STORAGE_KEY,
  BACKEND_TZ,
  syncTimezone,
} from '@/utils/time'

const selectedTz = ref<string>(
  (() => {
    try {
      return localStorage.getItem(STORAGE_KEY) || BACKEND_TZ
    } catch {
      return BACKEND_TZ
    }
  })(),
)

export function useRegionStore() {
  function switchRegion(value: string) {
    selectedTz.value = value
    syncTimezone(value)
  }

  const currentRegion = computed(() => {
    return (
      REGION_OPTIONS.find((r) => r.value === selectedTz.value) ||
      REGION_OPTIONS[0]
    )
  })

  return {
    selectedTz,
    regions: REGION_OPTIONS,
    switchRegion,
    currentRegion,
  }
}
