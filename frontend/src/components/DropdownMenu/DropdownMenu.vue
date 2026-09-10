<template>
  <a-trigger
    class="aia-toolbar-tooltip"
    arrow-class="aia-toolbar-tooltip__arrow"
    trigger="click"
    :default-popup-visible="props.visible"
    :popup-visible="popupVisible"
    @popup-visible-change="handleRootVisibleChange"
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
          position="right"
          :content-style="{
            height: '10px',
            minHeight: '200px',
            maxHeight: '300px',
          }"
          @popup-visible-change="handleSubmenuVisibleChange($event, option, optionIndex)"
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
                  @input="handleSearchChange(option, optionIndex)"
                  @press-enter="handleSearch(option, optionIndex)"
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
                    :option="createStatusOption('loading', option.request) as MenuItem"
                    :option-index="0"
                  />
                </template>
                <template v-else-if="option.request.empty">
                  <ReuseItem
                    :option="createStatusOption('empty', option.request) as MenuItem"
                    :option-index="0"
                  />
                </template>
                <template v-else v-for="(item, index) in childRender(option)" :key="item.key">
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
        @click.stop="handleClickItem(option)"
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
        <ReuseItem
          v-for="(option, index) in items"
          :key="option.key"
          :option="option"
          :option-index="index"
        />
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
import { isVNode, markRaw, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { createReusableTemplate } from '@vueuse/core'
import { IconPlus, IconSearch, IconRight, IconCheck } from '@arco-design/web-vue/es/icon'
import { debounce, isArray, isBoolean, isFunction, isString } from 'lodash-es'
import { isPromise } from '@arco-design/web-vue/es/_utils/is'

import type {
  DropdownMenuEmits,
  DropdownMenuOptions,
  DropdownMenuProps,
  MenuItem,
  Option,
  RequestParams,
} from './DropdownMenu.types'

const props = withDefaults(defineProps<DropdownMenuProps>(), {
  visible: false,
  loadingText: 'Loading...',
  emptyText: '暂无数据',
})

const emit = defineEmits<DropdownMenuEmits>()

const popupVisible = ref(props.visible)
const items = ref<MenuItem[]>(buildTree())

const [DefineItem, ReuseItem] = createReusableTemplate<{
  option: MenuItem
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

function createRequest(): Option['request'] {
  return {
    valueKey: 'value',
    labelKey: 'label',
    loading: false,
    empty: false,
    page: 1,
    pageSize: 50,
    total: 0,
    keyword: [],
    data: [],
  }
}

function isEmptyString(txt?: unknown): boolean {
  if (isString(txt)) {
    return txt.trim().length === 0
  }
  return true
}

function createDivider(parent: MenuItem | undefined, key: string): MenuItem {
  return {
    key,
    index: -1,
    type: 'divider',
    parent,
    request: createRequest(),
  } as MenuItem
}

function createNode(
  source: Partial<Option>,
  params: RequestParams | null,
  parent: MenuItem | undefined,
  key: string,
  index: number,
): MenuItem {
  const rawChildren = source.children
  const item = {
    ...source,
    key,
    index,
    parent,
    request: createRequest(),
    keyword: undefined,
  } as MenuItem
  item.value = (source as RequestParams)[params?.valueKey ?? 'value'] ?? source.value
  item.label = (source as RequestParams)[params?.labelKey ?? 'label'] ?? source.label
  item.children = rawChildren
  if (item.icon && typeof item.icon === 'object') {
    item.icon = markRaw(item.icon)
  }
  if (item.label && typeof item.label === 'object') {
    item.label = markRaw(item.label)
  }
  if (isArray(rawChildren)) {
    item.childrenNodes = buildList(rawChildren, null, item, `${key}/c`)
  }
  return item
}

function buildList(
  list: DropdownMenuOptions,
  params: RequestParams | null,
  parent: MenuItem | undefined,
  basePath: string,
): MenuItem[] {
  if (!isArray(list) || list.length === 0) return []
  const result: MenuItem[] = []
  list.forEach((entry, groupIndex) => {
    if (isArray(entry)) {
      if (entry.length === 0) return
      const groupPath = `${basePath}#${groupIndex}`
      const group = entry.map((option, index) =>
        createNode(option, params, parent, `${groupPath}.${index}`, index),
      )
      if (group.length === 0) return
      result.push(createDivider(parent, `${groupPath}.divider`))
      result.push(...group)
      return
    }
    result.push(createNode(entry, params, parent, `${basePath}.${groupIndex}`, groupIndex))
  })
  return result
}

function buildTree(): MenuItem[] {
  return buildList(props.options, null, undefined, 'root')
}

function createStatusOption(type: 'loading' | 'empty', request: Option['request']): MenuItem {
  return {
    key: `status-${type}`,
    index: 0,
    type,
    label: type === 'loading' ? props.loadingText : props.emptyText,
    request,
  } as MenuItem
}

function childRender(option: MenuItem): MenuItem[] {
  if (!isEmptyString(option.keyword)) {
    return option.keywordNodes ?? []
  }
  if (isFunction(option.children)) {
    return option.dataNodes ?? []
  }
  if (isArray(option.children)) {
    return option.childrenNodes ?? []
  }
  return []
}

function buildChildrenNodes(list: DropdownMenuOptions, item: MenuItem, suffix: string): MenuItem[] {
  return buildList(list, item.request, item, `${item.key}/${suffix}`)
}

function handleChildrenRequest(option: MenuItem, index: number) {
  if (!isFunction(option.children)) return
  if (isEmptyString(option.keyword) && childRender(option).length > 0) return

  const result = option.children(option.keyword, option, index)

  if (isArray(result)) {
    const nodes = buildChildrenNodes(result, option, 'k')
    option.request.empty = false
    option.request.loading = false
    if (!isEmptyString(option.keyword)) {
      option.request.keyword = result
      option.keywordNodes = nodes
    } else {
      option.request.data = result
      option.dataNodes = nodes
    }
    option.request.empty = result.length === 0
    return
  }

  if (isPromise(result) && !option.request.loading) {
    option.request.empty = false
    option.request.loading = true
    result
      .then((response) => {
        const res = (response ?? {}) as {
          data?: DropdownMenuOptions
          valueKey?: string
          labelKey?: string
          page?: number
          pageSize?: number
          total?: number
        }
        const list = res.data ?? []
        option.request.valueKey = res.valueKey ?? option.request.valueKey
        option.request.labelKey = res.labelKey ?? option.request.labelKey
        option.request.page = res.page ?? option.request.page
        option.request.pageSize = res.pageSize ?? option.request.pageSize
        option.request.total = res.total ?? option.request.total
        const nodes = buildChildrenNodes(list, option, 'd')
        if (!isEmptyString(option.keyword)) {
          option.request.keyword = list
          option.keywordNodes = nodes
        } else {
          option.request.data = list
          option.dataNodes = nodes
        }
        option.request.empty = list.length === 0
      })
      .catch((error) => {
        console.error(error)
      })
      .finally(() => {
        setTimeout(() => {
          option.request.loading = false
        }, 500)
      })
  }
}

function handleSearch(option: MenuItem, index: number) {
  option.request.empty = false
  option.request.loading = false
  option.request.keyword = []
  option.keywordNodes = []
  handleChildrenRequest(option, index)
}

const handleSearchChange = debounce(
  (option: MenuItem, index: number) => handleSearch(option, index),
  500,
)

function handleSubmenuVisibleChange(visible: boolean, option: MenuItem, index: number) {
  option.keyword = undefined
  option.request.empty = false
  if (visible) {
    handleChildrenRequest(option, index)
  }
}

function handleRootVisibleChange(visible: boolean) {
  handleToggle(visible)
  if (visible) {
    items.value = buildTree()
  }
}

async function handleClickItem(option: MenuItem) {
  if (option.visible || option.readonly || option.disabled) return

  const instance = reactive({
    popupVisible,
  })
  const parent = option.parent

  if (isFunction(parent?.onOptionClick)) {
    const result = await parent.onOptionClick(option, option.index, instance)
    if (parent?.multiple) return
    if (result !== false) {
      instance.popupVisible = false
    }
    return
  }

  if (isFunction(option.onClick)) {
    const result = await option.onClick(option, option.index, instance)
    if (result !== false) {
      instance.popupVisible = false
    }
    return
  }

  emit('option-click', option, option.index, instance)
  if (parent?.multiple) return
  instance.popupVisible = false
}

function handleClick() {
  handleToggle()
}

function handleToggle(visible?: boolean) {
  if (!isBoolean(visible)) {
    visible = !popupVisible.value
  }
  popupVisible.value = visible
  emit('update:visible', visible)
}

onBeforeUnmount(() => handleSearchChange.cancel())

defineExpose({
  popupVisible,
})
</script>

<style scoped lang="scss">
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
