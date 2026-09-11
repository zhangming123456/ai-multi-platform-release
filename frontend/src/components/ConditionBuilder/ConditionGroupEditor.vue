<template>
  <div class="cge" :class="containerClass">
    <div v-if="!flat" class="cge-header">
      <span v-if="isRoot" class="cge-header__label">条件组合</span>
      <span v-else class="cge-header__badge" :class="{ 'cge-header__badge--scoped': lockGroup }">
        {{ groupTitle || '条件组' }}
      </span>

      <span v-if="isRoot" class="cge-header__hint">
        每行左侧可切换 且 / 或（且 优先级高于 或）
      </span>

      <div class="cge-header__spacer" />

      <slot v-if="isRoot" name="header-actions" />
      <span v-else-if="lockGroup" class="cge-header__lock">
        {{ fieldOptions.length }} 个可用变量
      </span>
      <template v-else>
        <a-button type="text" size="mini" class="cge-header__action" @click="requestUngroup">
          {{ ungroupText }}
        </a-button>
        <a-button
          type="text"
          size="mini"
          status="danger"
          class="cge-header__action"
          @click="requestRemove"
        >
          <template #icon><IconDelete /></template>
        </a-button>
      </template>
    </div>

    <div class="cge-body" :class="{ 'cge-body--connected': renderNodes.length > 1 }">
      <template v-for="node in renderNodes" :key="node.child.id">
        <div v-if="node.kind === 'scoped'" class="cge-node cge-node--scoped">
          <ConditionConnector
            :index="node.index"
            :logic="node.child.logic"
            :disabled="disabled"
            :editable="logicEditable"
            @update:logic="onNodeLogicChange(node.index, $event)"
          />
          <ConditionGroupEditor
            :group="asConditionGroup(node.child)"
            :path="childPath(node.index)"
            :depth="depth + 1"
            :field-options="node.scoped?.fields ?? []"
            :field-groups="node.scoped ? [node.scoped] : []"
            :scoped-groups="[]"
            :group-title="node.scoped?.label ?? ''"
            :lock-group="true"
            :rule-context="ruleContext"
            :disabled="disabled"
            :logic-editable="logicEditable"
            :max-depth="maxDepth"
            :max-items="maxItems"
            :add-text="addText"
            :add-group-text="addGroupText"
            :clear-text="clearText"
            :empty-text="emptyText"
            :ungroup-text="ungroupText"
            @command="emitCommand"
          />
        </div>
        <ConditionGroupEditor
          v-else-if="node.kind === 'flat'"
          :group="asConditionGroup(node.child)"
          :path="childPath(node.index)"
          :depth="depth + 1"
          :field-options="node.scoped?.fields ?? []"
          :field-groups="node.scoped ? [node.scoped] : []"
          :scoped-groups="[]"
          :flat="true"
          :rule-context="ruleContext"
          :disabled="disabled"
          :logic-editable="logicEditable"
          :max-depth="maxDepth"
          :max-items="maxItems"
          :add-text="addText"
          :add-group-text="addGroupText"
          :clear-text="clearText"
          :empty-text="emptyText"
          :ungroup-text="ungroupText"
          @command="emitCommand"
        />
        <div v-else-if="node.kind === 'group'" class="cge-node">
          <ConditionConnector
            :index="node.index"
            :logic="node.child.logic"
            :disabled="disabled"
            :editable="logicEditable"
            @update:logic="onNodeLogicChange(node.index, $event)"
          />
          <ConditionGroupEditor
            :group="asConditionGroup(node.child)"
            :path="childPath(node.index)"
            :depth="depth + 1"
            :field-options="fieldOptions"
            :field-groups="fieldGroups"
            :scoped-groups="[]"
            :rule-context="ruleContext"
            :disabled="disabled"
            :logic-editable="logicEditable"
            :max-depth="maxDepth"
            :max-items="maxItems"
            :add-text="addText"
            :add-group-text="addGroupText"
            :clear-text="clearText"
            :empty-text="emptyText"
            :ungroup-text="ungroupText"
            @command="emitCommand"
          />
        </div>
        <ConditionItemRow
          v-else
          :item="asConditionItem(node.child)"
          :path="path"
          :index="node.index"
          :field-options="fieldOptions"
          :field-groups="fieldGroups"
          :rule-context="ruleContext"
          :disabled="disabled"
          :logic-editable="logicEditable"
          @command="emitCommand"
        />
      </template>

      <div v-if="!renderNodes.length" class="cge-empty">{{ emptyText }}</div>
    </div>

    <div v-if="showFooter" class="cge-footer">
      <a-button
        v-if="showAddItem"
        type="text"
        size="small"
        class="cge-footer__add"
        :disabled="disabled || atMaxItems"
        @click="emitCommand({ type: 'add-item', path })"
      >
        <template #icon><IconPlus /></template>
        {{ addText }}
      </a-button>
      <a-button
        v-if="showAddGroup && canAddGroup"
        type="text"
        size="small"
        class="cge-footer__add-group"
        :disabled="disabled"
        @click="emitCommand({ type: 'add-group', path })"
      >
        <template #icon><IconPlus /></template>
        {{ addGroupText }}
      </a-button>
      <div class="cge-footer__spacer" />
      <a-button
        v-if="showClear"
        type="text"
        size="small"
        class="cge-footer__clear"
        :disabled="disabled"
        @click="emitCommand({ type: 'clear-group', path })"
      >
        {{ clearText }}
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'
import ConditionConnector from './ConditionConnector.vue'
import ConditionItemRow from './ConditionItemRow.vue'
import type {
  ConditionCommand,
  ConditionFieldGroup,
  ConditionGroup,
  ConditionGroupEditorEmits,
  ConditionGroupEditorProps,
  ConditionItem,
  ConditionLogic,
  ConditionNode,
} from './ConditionBuilder.types'
import { CONDITION_MAX_DEPTH, isConditionGroup } from './conditionOperator'

const props = withDefaults(defineProps<ConditionGroupEditorProps>(), {
  disabled: false,
  logicEditable: true,
  maxDepth: CONDITION_MAX_DEPTH,
  maxItems: 0,
  isRoot: false,
  scopedGroups: () => [],
  groupTitle: '',
  lockGroup: false,
  flat: false,
  addText: '添加条件',
  addGroupText: '添加子条件组',
  clearText: '清空',
  emptyText: '暂无判断条件，点击下方按钮添加',
  ungroupText: '解组',
})

const emit = defineEmits<ConditionGroupEditorEmits>()

interface RenderNode {
  child: ConditionNode
  index: number
  kind: 'item' | 'group' | 'scoped' | 'flat'
  scoped?: ConditionFieldGroup
}

const containerClass = computed<string[]>(() => {
  if (props.isRoot) return ['cge--root']
  if (props.flat) return ['cge--flat']
  return props.lockGroup ? ['cge--nested', 'cge--scoped'] : ['cge--nested']
})

const scopedIndex = computed<Map<string, ConditionFieldGroup>>(() => {
  const map = new Map<string, ConditionFieldGroup>()
  for (const group of props.scopedGroups) map.set(group.key, group)
  return map
})

const activeScopedCount = computed(
  () => props.scopedGroups.filter((group) => group.active !== false).length,
)

const flatScoped = computed(() => activeScopedCount.value === 1)

const renderNodes = computed<RenderNode[]>(() => {
  const nodes: RenderNode[] = []
  props.group.children.forEach((child, index) => {
    if (!isConditionGroup(child)) {
      nodes.push({ child, index, kind: 'item' })
      return
    }
    const scoped = child.scope ? scopedIndex.value.get(child.scope) : undefined
    if (scoped) {
      if (scoped.active === false) return
      nodes.push({ child, index, kind: flatScoped.value ? 'flat' : 'scoped', scoped })
      return
    }
    nodes.push({ child, index, kind: 'group' })
  })
  return nodes
})

const scopedRenderCount = computed(
  () => renderNodes.value.filter((node) => node.kind === 'scoped').length,
)

const scopedFeatureEnabled = computed(() => props.scopedGroups.length > 0)

const showAddItem = computed(() => !(props.isRoot && scopedRenderCount.value >= 2))

const showAddGroup = computed(
  () => showAddItem.value && !(props.isRoot && scopedFeatureEnabled.value),
)

const showClear = computed(() =>
  props.isRoot
    ? scopedRenderCount.value === 0 && props.group.children.length > 0
    : props.lockGroup && props.group.children.length > 0,
)

const showFooter = computed(() => !props.flat && (showAddItem.value || showClear.value))

const canAddGroup = computed(() => props.depth < props.maxDepth)
const atMaxItems = computed(
  () => props.maxItems > 0 && props.group.children.length >= props.maxItems,
)

function childPath(index: number): number[] {
  return [...props.path, index]
}

function asConditionGroup(node: ConditionNode): ConditionGroup {
  return node as ConditionGroup
}

function asConditionItem(node: ConditionNode): ConditionItem {
  return node as ConditionItem
}

function emitCommand(command: ConditionCommand): void {
  emit('command', command)
}

function onNodeLogicChange(index: number, logic: ConditionLogic): void {
  emitCommand({ type: 'set-logic', path: childPath(index), logic })
}

function requestRemove(): void {
  emitCommand({ type: 'remove-group', path: props.path })
}

function requestUngroup(): void {
  emitCommand({ type: 'ungroup-group', path: props.path })
}
</script>

<style scoped>
.cge {
  display: flex;
  flex-direction: column;
}

.cge--nested {
  flex: 1;
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid #dcdce1;
  border-left: 3px solid #5856d6;
  border-radius: 12px;
  background: #fbfbfd;
}

.cge--scoped {
  border-left-color: #00b96b;
  background: #f7fcf9;
}

.cge--flat {
  min-width: 0;
}

.cge-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.cge-header__label {
  font-size: 13px;
  font-weight: 600;
  color: #1d1d1f;
}

.cge-header__badge {
  font-size: 11px;
  font-weight: 600;
  color: #5856d6;
  background: rgba(88, 86, 214, 0.1);
  border-radius: 6px;
  padding: 2px 8px;
}

.cge-header__badge--scoped {
  color: #0f766e;
  background: rgba(0, 185, 107, 0.12);
}

.cge-header__lock {
  font-size: 11px;
  color: #86868b;
}

.cge-header__hint {
  font-size: 12px;
  color: #86868b;
}

.cge-header__spacer {
  flex: 1;
}

.cge-header__action {
  color: #86868b;
}

.cge-body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cge-body--connected::before {
  content: '';
  position: absolute;
  left: 13px;
  top: 15px;
  bottom: 15px;
  width: 2px;
  border-radius: 1px;
  background: #e5e6eb;
  z-index: 0;
}

.cge-body > * {
  position: relative;
  z-index: 1;
}

.cge-node {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.cge-empty {
  padding: 14px 0;
  text-align: center;
  font-size: 12px;
  color: #86868b;
}

.cge-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed #e5e5ea;
}

.cge-footer__spacer {
  flex: 1;
}

.cge-footer__add {
  color: #007aff;
}

.cge-footer__add-group {
  color: #5856d6;
}

.cge-footer__clear {
  color: #86868b;
}
</style>
