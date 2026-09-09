<template>
  <a-trigger
    class="aia-toolbar-tooltip"
    arrow-class="aia-toolbar-tooltip__arrow"
    trigger="click"
    :default-popup-visible="props.visible"
    :popup-visible="popupVisible"
    @popup-visible-change="handlePopupVisibleChange($event)"
    position="tl"
  >
    <DefineItem v-slot="{ option, optionIndex, isCheck }">
      <template v-if="option.visible === false">
        <div v-show="false">隐藏项</div>
      </template>
      <a-divider v-else-if="option.type === 'divider'" class="aia-attach-menu__divider" />
      <template v-else-if="option.children">
        <a-trigger
          class="aia-toolbar-tooltip"
          arrow-class="aia-toolbar-tooltip__arrow"
          v-model="option.popupVisible"
          position="right"
          :content-style="{
            height: '10px',
            minHeight: '200px',
            maxHeight: '300px',
          }"
          @popup-visible-change="handlePopupVisibleChange($event, option, optionIndex)"
          v-bind="option.triggerProps"
        >
          <template #content>
            <div class="aia-attach-menu aia-attach-menu-right">
              <template v-if="option.showSearch">
                <a-input
                  v-model.trim="option.keyword"
                  autofocus
                  @blur.stop
                  :placeholder="option.searchPlaceholder ?? `搜索当前${option.label}`"
                  @input="handleSearchChange(option.keyword, option, optionIndex)"
                  @press-enter="handleSearch(option.keyword, option, optionIndex)"
                >
                  <template #prefix>
                    <IconSearch :size="option.iconSize" />
                  </template>
                </a-input>
                <a-divider class="aia-attach-menu__divider" />
              </template>
              <div class="aia-attach-menu_main">
                <template v-if="option.request.loading">
                  <ReuseItem
                    :option="
                      {
                        key: '0',
                        type: 'loading',
                        label: props.loadingText,
                      } as Option
                    "
                    :option-index="0"
                  />
                </template>
                <template v-else-if="option.request.empty">
                  <ReuseItem
                    :option="
                      {
                        key: '0',
                        type: 'empty',
                        label: props.emptyText,
                      } as Option
                    "
                    :option-index="0"
                  />
                </template>
                <template
                  v-else
                  v-for="(item, index) in childRender(option, optionIndex)"
                  :key="item.key"
                >
                  <ReuseItem
                    :option="item"
                    :option-index="index"
                    :is-check="option.isCheck?.(item)"
                  />
                </template>
              </div>
            </div>
          </template>
          <button type="button" class="aia-attach-menu__item">
            <template v-if="option.icon">
              <component :is="option.icon" :option="option" :size="option.iconSize ?? 16" />
            </template>
            <span class="aia-attach-menu__item-label flex-1">
              <component
                v-if="isVNode(option.label) || isFunction(option.label)"
                :is="option.label"
                :option="option"
              />
              <template v-else>{{ option.label }}</template>
            </span>
            <IconRight :size="option.iconSize ?? 16" />
          </button>
        </a-trigger>
      </template>
      <div
        class="aia-attach-menu__item"
        :class="{
          [`is-${option.type}`]: option.type,
        }"
        v-else-if="option.readonly"
      >
        <template v-if="option.icon">
          <component :is="option.icon" :option="option" :size="option.iconSize ?? 16" />
        </template>
        <span class="aia-attach-menu__item-label">
          <component
            v-if="isVNode(option.label) || isFunction(option.label)"
            :is="option.label"
            :option="option"
          />
          <template v-else>{{ option.label }}</template>
        </span>
      </div>
      <button
        v-else
        type="button"
        class="aia-attach-menu__item"
        :disabled="option.disabled"
        :readonly="option.readonly"
        @click.stop="handleClickItem(option, optionIndex)"
      >
        <template v-if="option.icon">
          <component :is="option.icon" :size="option.iconSize ?? 16" />
        </template>
        <span class="aia-attach-menu__item-label flex-1">
          <component v-if="isVNode(option.label) || isFunction(option.label)" :is="option.label" />
          <template v-else>{{ option.label }}</template>
        </span>
        <IconCheck
          v-if="isCheck"
          class="aia-attach-menu__item-check"
          :size="option.iconSize ?? 16"
        />
      </button>
    </DefineItem>

    <template #content>
      <div class="aia-attach-menu">
        <template v-for="(option, index) in options" :key="option.key">
          <ReuseItem :option="option" :option-index="index"></ReuseItem>
        </template>
      </div>
    </template>
    <span @click="handleClick">
      <slot name="default" :visible="popupVisible">
        <button
          type="button"
          class="aia-toolbar-btn"
          :disabled="props.disabled"
          @click="handleClick"
        >
          <component :is="IconPlus" :size="props.iconSize" />
        </button>
      </slot>
    </span>
  </a-trigger>
</template>

<script setup lang="ts">
import { isVNode, ref, unref, watch, computed, reactive } from 'vue'
import { createReusableTemplate } from '@vueuse/core'
import { IconPlus, IconSearch, IconRight, IconCheck } from '@arco-design/web-vue/es/icon'
import { debounce, isArray, isBoolean, isFunction, isString, mergeWith, uniqueId } from 'lodash-es'
import { isPromise } from '@arco-design/web-vue/es/_utils/is'

import { type DropdownMenuProps, type Option } from '@/components/DropdownMenu/types'

const props = withDefaults(defineProps<DropdownMenuProps>(), {
  visible: false,
  loadingText: 'Loading...',
  emptyText: '暂无数据',
})
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'option-click', option: Option, index: number, instance?: any): void
}>()

const popupVisible = ref(props.visible)
const options = ref(handleOptions(props.options))

const [DefineItem, ReuseItem] = createReusableTemplate<{
  option: Option
  optionIndex: number
  isCheck?: boolean
}>()

watch(
  () => props.visible,
  (value, oldValue) => {
    if (value === oldValue) return
    popupVisible.value = value
  },
)

function handlePopupVisibleChange(visible: boolean, option?: Option, index?: number) {
  if (option) {
    console.log(visible, 'handlePopupVisibleChange')
    option.keyword = undefined
    option.request.empty = false
    if (visible) {
      handleChildrenRequest(option.keyword, option, index!)
    }
    return
  }
  handleToggle(visible)
  if (visible) {
    options.value = handleOptions(props.options)
  }
}

async function handleClickItem(option: Option, index: number) {
  switch (true) {
    case option.visible:
    case option.readonly:
    case option.disabled: {
      return
    }
  }
  const instance = reactive({
    popupVisible,
  })
  if (isFunction(option.parent?.onOptionClick)) {
    const result = await option.parent.onOptionClick(option, index, instance)
    if (option.parent?.multiple) return
    if (result !== false) {
      instance.popupVisible = false
    }
    return
  }
  if (isFunction(option.onClick)) {
    const result = await option.onClick(option, index, instance)
    if (result !== false) {
      instance.popupVisible = false
    }
    return
  }
  emit('option-click', option, index, instance)
  if (option.parent?.multiple) return
  instance.popupVisible = false
}

function handleClick() {
  handleToggle()
}

function handleToggle(visible?: boolean) {
  if (!isBoolean(visible)) {
    visible = !unref(popupVisible)
  }
  popupVisible.value = visible
  emit('update:visible', visible)
}

function defineOptionRequest(info?: Option['request']): Option['request'] {
  return mergeWith(
    {
      valueKey: 'value',
      labelKey: 'label',
      loading: false,
      page: 1,
      pageSize: 50,
      total: 0,
      data: [],
      keyword: [],
    } as Option['request'],
    info,
  )
}

function handleOptions(
  options: DropdownMenuProps['options'],
  params?: {
    valueKey?: string
    labelKey?: string
    [k: string]: any
  } | null,
  parent?: Option,
): Option[] {
  if (options.length === 0) return [] as Option[]
  const lateralOptions: Option[] = []
  options.forEach((item, index) => {
    if (isArray(item)) {
      if (item.length === 0) {
        return
      }
      const groupOptions = item.map((option, index) => handleOption(option, index, params, parent))
      if (groupOptions.length === 0) {
        return
      }
      lateralOptions.push({
        type: 'divider',
      } as Option)
      lateralOptions.push(...groupOptions)
      return
    }
    lateralOptions.push(handleOption(item, index, params, parent))
  })
  return lateralOptions
}

function handleOption(
  option: Partial<Option>,
  _index: number,
  params?: {
    valueKey?: string
    labelKey?: string
    [k: string]: any
  } | null,
  parent?: Option,
): Option {
  const children = option.children
  delete option.children
  const item = option as Option
  item.value = option[(params?.valueKey ?? 'value') as 'value'] ?? option.value
  item.label = option[(params?.labelKey ?? 'label') as 'label'] ?? option.label
  item.key = uniqueId('option-item-')
  item.keyword = undefined
  item.parent = parent
  item.request = defineOptionRequest()
  item.children = children
  return item
}

function isEmptyString(txt?: any): boolean {
  if (isString(txt)) {
    return txt.trim().length === 0
  }
  return true
}

const handleSearchChange = debounce(handleSearch, 500)
function handleSearch(_keyword: string | undefined, option: Option, index: number) {
  // console.log(keyword, 'renderChild:eventevent')
  option.request.empty = false
  option.request.loading = false
  option.request.keyword = []
  handleChildrenRequest(option.keyword, option, index)
}

function handleChildrenRequest(keyword: string | undefined, option: Option, index: number) {
  if (isFunction(option.children)) {
    const options = renderChild(option, index!)
    if (isEmptyString(keyword) && options.length > 0) {
      return
    }
    const p = option.children(keyword, option, index)
    if (isArray(p)) {
      option.request!.empty = false
      option.request!.loading = false
      if (!isEmptyString(keyword)) {
        option.request!.keyword = p
      } else {
        option.request!.data = p
      }
      option.request!.empty = p.length === 0
    } else if (isPromise(p) && !option.request!.loading) {
      option.request!.empty = false
      option.request!.loading = true
      p.then((result) => {
        let list: Option[] = result.data ?? []
        if (!isEmptyString(option.keyword)) {
          option.request!.keyword = list
        } else {
          option.request!.data = list
        }
        option.request!.valueKey = result.valueKey
        option.request!.labelKey = result.labelKey
        option.request!.page = result.page
        option.request!.pageSize = result.pageSize
        option.request!.total = result.total
        option.request!.empty = list.length === 0
      })
        .catch((e) => {
          console.error(e)
        })
        .finally(() => {
          setTimeout(() => {
            option.request!.loading = false
          }, 500)
        })
    }
  }
}
function renderChild(option: Option, index: number): Option[] {
  return unref(childRender)(option, index)
}
const childRender = computed(() => {
  return (option: Option, _index: number): Option[] => {
    if (!isEmptyString(option.keyword)) {
      const list = option.request.keyword
      console.log(option.label, option.keyword, list, 'renderChild:1')
      return handleOptions(list, option.request, option)
    }
    if (isFunction(option.children)) {
      const list = option.request.data
      console.log(option.label, option.keyword, list, 'renderChild:2')
      return handleOptions(list, option.request, option)
    }
    if (isArray(option.children)) {
      console.log(option.label, option.keyword, option.children, 'renderChild:3')
      return handleOptions(option.children, null, option)
    }
    return []
  }
})

defineExpose({
  popupVisible,
})
</script>

<style scoped lang="scss">
//:global(.aia-toolbar-tooltip .arco-tooltip-content) {
//  padding: 0;
//  background-color: transparent;
//  transform: translateY(8px);
//}
//:global(.aia-toolbar-tooltip.arco-trigger-position-right .arco-tooltip-content) {
//  padding: 0;
//  background-color: transparent;
//  transform: translate(-6px, 0);
//}
//
//:global(.aia-toolbar-tooltip .aia-toolbar-tooltip__arrow) {
//  display: none;
//}

.aia-attach-menu {
  height: 100%;
  position: relative;
  min-width: 168px;
  padding: 4px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 8px;
  background: #2c2c2e;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.8);
  &.aia-attach-menu-right {
    display: flex;
    flex-direction: column;
    :deep(.arco-input-wrapper) {
      padding: 0 6px;
      background-color: transparent;
      border: none;
      color: #ffffff;
      .arco-input-prefix {
        padding-right: 4px;
      }
      .arco-icon {
        color: #ffffff;
      }
      &.arco-input-focus {
        border: none;
      }
    }
    .aia-attach-menu_main {
      flex: 1;
      position: relative;
      overflow-y: auto;
    }
  }
}
.aia-attach-menu__divider {
  padding: 0;
  margin: 4px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
}
.aia-attach-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #e5e5ea;
  font-size: 13px;
  line-height: 1;
  text-align: left;
  transition: background 0.15s ease;
  cursor: initial;
  &.is-loading,
  &.is-empty {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: auto;
  }
  .aia-attach-menu__item-label {
    max-width: 150px;
    line-height: 1.2;
  }
  &:not(:disabled) {
    .aia-attach-menu__item-check {
      color: #6ee7a0;
    }
  }
}
button.aia-attach-menu__item {
  cursor: pointer;
}
button.aia-attach-menu__item:hover:not(:disabled) {
  background: #3a3a3c;
  color: #ffffff;
}
.aia-attach-menu__item:disabled {
  color: #c9cdd4;
  cursor: not-allowed;
}
</style>
