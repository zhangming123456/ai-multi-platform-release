<template>
  <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">{{ title }}</h3>
        <span class="text-[12px] text-[#86868B] font-medium">{{ count }}</span>
      </div>
      <div class="flex items-center gap-4">
        <a-checkbox
          v-if="readableItems.length > 0"
          :model-value="allReadChecked"
          :indeterminate="allReadIndeterminate"
          :disabled="readonly"
          @change="onSelectAllRead"
        >
          <span class="text-[12px] text-[#1D1D1F]">全选读</span>
        </a-checkbox>
        <a-checkbox
          v-if="writableItems.length > 0"
          :model-value="allWriteChecked"
          :indeterminate="allWriteIndeterminate"
          :disabled="readonly"
          @change="onSelectAllWrite"
        >
          <span class="text-[12px] text-[#1D1D1F]">全选写</span>
        </a-checkbox>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <div
        v-for="item in items"
        :key="item.id"
        class="flex items-center justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
      >
        <div class="min-w-0 mr-3">
          <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">{{ item.title }}</p>
          <p class="text-[11px] text-[#86868B] m-0 truncate">{{ item.subtitle }}</p>
        </div>
        <div class="flex items-center gap-3 shrink-0">
          <a-switch
            v-if="item.readKey"
            :model-value="hasRead(item)"
            :disabled="isReadDisabled(item)"
            size="small"
            type="line"
            checked-text="读"
            unchecked-text="读"
            @change="(v: boolean | string | number) => onReadChange(item, v)"
          />
          <a-switch
            v-if="item.writeKeys.length > 0"
            :model-value="hasWrite(item)"
            :disabled="isWriteDisabled(item)"
            size="small"
            type="line"
            checked-text="写"
            unchecked-text="写"
            @change="(v: boolean | string | number) => onWriteChange(item, v)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface ModuleItem {
  id: string
  title: string
  subtitle: string
  readKey?: string
  writeKeys: string[]
}

interface Props {
  title: string
  count: number
  items: ModuleItem[]
  effectiveKeys: Set<string>
  inheritedKeys: Set<string>
  readonly?: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'toggle-read', key: string): void
  (e: 'toggle-write', keys: string[]): void
  (e: 'select-all-read'): void
  (e: 'deselect-all-read', keys: string[]): void
  (e: 'select-all-write'): void
  (e: 'deselect-all-write', keys: string[]): void
}>()

function hasRead(item: ModuleItem): boolean {
  if (!item.readKey) return false
  return props.effectiveKeys.has(item.readKey)
}

function hasWrite(item: ModuleItem): boolean {
  if (item.writeKeys.length === 0) return false
  return item.writeKeys.some((key) => props.effectiveKeys.has(key))
}

function isReadDisabled(item: ModuleItem): boolean {
  return props.readonly || !item.readKey || props.inheritedKeys.has(item.readKey)
}

function isWriteDisabled(item: ModuleItem): boolean {
  if (props.readonly) return true
  if (item.writeKeys.length === 0) return true
  return item.writeKeys.some((key) => props.inheritedKeys.has(key))
}

function onReadChange(item: ModuleItem, checked: boolean | string | number) {
  if (!item.readKey || isReadDisabled(item)) return
  if (checked !== hasRead(item)) {
    emit('toggle-read', item.readKey)
  }
}

function onWriteChange(item: ModuleItem, checked: boolean | string | number) {
  if (item.writeKeys.length === 0 || isWriteDisabled(item)) return
  if (checked !== hasWrite(item)) {
    emit('toggle-write', item.writeKeys)
  }
}

const readableItems = computed(() =>
  props.items.filter((i) => i.readKey && !props.inheritedKeys.has(i.readKey)),
)
const writableItems = computed(() =>
  props.items.filter(
    (i) => i.writeKeys.length > 0 && !i.writeKeys.some((k) => props.inheritedKeys.has(k)),
  ),
)

const allReadChecked = computed(() => {
  if (readableItems.value.length === 0) return false
  return readableItems.value.every((i) => hasRead(i))
})

const allReadIndeterminate = computed(() => {
  if (readableItems.value.length === 0) return false
  const checkedCount = readableItems.value.filter((i) => hasRead(i)).length
  return checkedCount > 0 && checkedCount < readableItems.value.length
})

const allWriteChecked = computed(() => {
  if (writableItems.value.length === 0) return false
  return writableItems.value.every((i) => hasWrite(i))
})

const allWriteIndeterminate = computed(() => {
  if (writableItems.value.length === 0) return false
  const checkedCount = writableItems.value.filter((i) => hasWrite(i)).length
  return checkedCount > 0 && checkedCount < writableItems.value.length
})

function onSelectAllRead(checked: boolean | (string | number | boolean)[]) {
  if (props.readonly) return
  if (checked === true) {
    emit('select-all-read')
  } else if (checked === false) {
    const keys = readableItems.value.map((i) => i.readKey as string).filter(Boolean)
    emit('deselect-all-read', keys)
  }
}

function onSelectAllWrite(checked: boolean | (string | number | boolean)[]) {
  if (props.readonly) return
  if (checked === true) {
    emit('select-all-write')
  } else if (checked === false) {
    const keys = writableItems.value.flatMap((i) => i.writeKeys)
    emit('deselect-all-write', keys)
  }
}
</script>
