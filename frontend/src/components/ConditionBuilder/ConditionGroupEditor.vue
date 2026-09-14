<template>
  <div class="cge" :class="containerClass">
    <div v-if="!flat" class="cge-header">
      <span v-if="isRoot" class="cge-header__label">条件组合</span>
      <span v-else class="cge-header__badge" :class="{ 'cge-header__badge--scoped': lockGroup }">
        {{ groupTitle || '条件组' }}
      </span>

      <span v-if="lockedFieldLabel" class="cge-header__locked">
        <Lock :size="12" />
        变量锁定：{{ lockedFieldLabel }}
      </span>

      <el-tooltip v-if="lockedExpression" placement="bottom-end">
        <span class="cge-header__expression">{{ lockedExpression }}</span>
        <template #content>
          <div class="cge-expression-tip">{{ lockedExpression }}</div>
        </template>
      </el-tooltip>

      <span v-if="isRoot" class="cge-header__hint">
        {{
          logicMode === 'uniform'
            ? '同一层级共用一个 且 / 或（且 优先级高于 或）'
            : '每行左侧可切换 且 / 或（且 优先级高于 或）'
        }}
      </span>

      <div class="cge-header__spacer" />

      <slot v-if="isRoot" name="header-actions" />
      <span v-else-if="lockGroup" class="cge-header__lock">
        {{ fieldOptions.length }} 个可用变量
      </span>
      <template v-else>
        <el-button link size="small" class="cge-header__action" @click="requestUngroup">
          {{ ungroupText }}
        </el-button>
        <el-button
          link
          type="danger"
          size="small"
          class="cge-header__action"
          :icon="Trash2"
          @click="requestRemove"
        />
      </template>
    </div>

    <div
      class="cge-body"
      :class="{
        'cge-body--connected': logicMode === 'mixed' && renderNodes.length > 1,
        'cge-body--bracket': showLevelLogic,
      }"
    >
      <template v-for="node in renderNodes" :key="node.child.id">
        <div v-if="node.kind === 'scoped'" class="cge-node cge-node--scoped">
          <ConditionConnector
            :index="node.index"
            :logic="node.child.logic"
            :disabled="disabled"
            :editable="logicEditable"
            :logic-mode="logicMode"
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
            :logic-mode="logicMode"
            :locked-field="effectiveLockedField"
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
          :logic-mode="logicMode"
          :locked-field="effectiveLockedField"
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
            :logic-mode="logicMode"
            @update:logic="onNodeLogicChange(node.index, $event)"
          />
          <ConditionGroupEditor
            :group="asConditionGroup(node.child)"
            :path="childPath(node.index)"
            :depth="depth + 1"
            :field-options="fieldOptions"
            :field-groups="fieldGroups"
            :scoped-groups="[]"
            :logic-mode="logicMode"
            :locked-field="effectiveLockedField"
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
          :locked-field="effectiveLockedField"
          :can-add-group="canAddGroup"
          :max-depth="maxDepth"
          :logic-mode="logicMode"
          :rule-context="ruleContext"
          :disabled="disabled"
          :logic-editable="logicEditable"
          @command="emitCommand"
        />
      </template>

      <div v-if="showLevelLogic" class="cge-level-logic">
        <button
          type="button"
          class="cge-level-seg"
          :class="{ 'cge-level-seg--active': levelLogic === 'and' }"
          :disabled="disabled"
          @click="requestLevelLogic('and')"
        >
          且
        </button>
        <button
          type="button"
          class="cge-level-seg"
          :class="{ 'cge-level-seg--active': levelLogic === 'or' }"
          :disabled="disabled"
          @click="requestLevelLogic('or')"
        >
          或
        </button>
      </div>

      <div v-if="!renderNodes.length" class="cge-empty">{{ emptyText }}</div>
    </div>

    <div v-if="showFooter" class="cge-footer">
      <el-button
        v-if="showAddItem"
        link
        size="small"
        class="cge-footer__add"
        :disabled="disabled || atMaxItems"
        :icon="Plus"
        @click="requestAddItem"
      >
        {{ addText }}
      </el-button>
      <el-button
        v-if="showAddGroup && canAddGroup"
        link
        size="small"
        class="cge-footer__add-group"
        :disabled="disabled"
        :icon="Plus"
        @click="requestAddGroup"
      >
        {{ addGroupText }}
      </el-button>
      <div class="cge-footer__spacer" />
      <el-button
        v-if="showClear"
        link
        size="small"
        class="cge-footer__clear"
        :disabled="disabled"
        @click="emitCommand({ type: 'clear-group', path })"
      >
        {{ clearText }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Lock, Plus, Trash2 } from 'lucide-vue-next'
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
import {
  CONDITION_MAX_DEPTH,
  buildConditionExpression,
  conditionFieldLabel,
  isConditionGroup,
  resolveGroupFieldLock,
} from './conditionOperator'

const props = withDefaults(defineProps<ConditionGroupEditorProps>(), {
  disabled: false,
  logicEditable: true,
  logicMode: 'mixed',
  maxDepth: CONDITION_MAX_DEPTH,
  maxItems: 0,
  isRoot: false,
  scopedGroups: () => [],
  groupTitle: '',
  lockGroup: false,
  lockedField: '',
  flat: false,
  addText: '添加条件',
  addGroupText: '添加子条件组',
  clearText: '清空',
  emptyText: '暂无判断条件，点击下方按钮添加',
  ungroupText: '解组',
})

const emit = defineEmits<ConditionGroupEditorEmits>()

const ownLockedField = computed(() => resolveGroupFieldLock(props.group, props.fieldOptions))

const effectiveLockedField = computed(() => props.lockedField || ownLockedField.value)

const lockedFieldLabel = computed(() => {
  if (!effectiveLockedField.value) return ''
  const field = props.fieldOptions.find((entry) => entry.value === effectiveLockedField.value)
  return field ? conditionFieldLabel(field) : effectiveLockedField.value
})

const lockedExpression = computed(() =>
  lockedFieldLabel.value ? buildConditionExpression(props.group, props.fieldOptions) : '',
)

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

const retainedScopes = computed<Set<string>>(
  () =>
    new Set(props.scopedGroups.filter((group) => group.active === false).map((group) => group.key)),
)

function isRetainedNode(node: ConditionNode): boolean {
  return isConditionGroup(node) && !!node.scope && retainedScopes.value.has(node.scope)
}

const activeChildCount = computed(
  () => props.group.children.filter((child) => !isRetainedNode(child)).length,
)

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
    ? scopedRenderCount.value === 0 && activeChildCount.value > 0
    : props.lockGroup && activeChildCount.value > 0,
)

const showFooter = computed(() => !props.flat && (showAddItem.value || showClear.value))

const canAddGroup = computed(() => props.depth < props.maxDepth)

const levelLogic = computed<ConditionLogic>(() => {
  return props.group.children[0]?.logic ?? props.group.logic
})

const showLevelLogic = computed(
  () => props.logicMode === 'uniform' && props.group.children.length > 1,
)

function requestLevelLogic(logic: ConditionLogic): void {
  emitCommand({ type: 'set-level-logic', path: props.path, logic })
}

const atMaxItems = computed(() => props.maxItems > 0 && activeChildCount.value >= props.maxItems)

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

function requestAddItem(): void {
  emitCommand({ type: 'add-item', path: props.path })
}

function requestAddGroup(): void {
  emitCommand({
    type: 'add-group',
    path: props.path,
    field: effectiveLockedField.value || undefined,
  })
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

.cge-header__locked {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: #d46b08;
  background: rgba(255, 149, 0, 0.14);
  border-radius: 6px;
  padding: 2px 8px;
}

.cge-header__lock {
  font-size: 11px;
  color: #86868b;
}

.cge-header__expression {
  flex: 0 1 auto;
  min-width: 0;
  max-width: min(26vw, 380px);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  padding-left: 10px;
  border-left: 1px solid #e5e5ea;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  color: #6e6e73;
}

.cge-expression-tip {
  max-width: 420px;
  word-break: break-all;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  line-height: 1.6;
}

.cge-header__hint {
  font-size: 12px;
  color: #86868b;
}

.cge-header__spacer {
  flex: 1;
}

.cge-body > .cge-level-logic {
  position: absolute;
  top: 50%;
  left: 5px;
  transform: translateY(-50%);
  z-index: 2;
  display: flex;
  flex-direction: column;
  width: 18px;
  border-radius: 4px;
  overflow: hidden;
}

.cge-level-seg {
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  background: #e9f0f7;
  color: #86909c;
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
  cursor: pointer;
  transition:
    background 0.15s ease,
    color 0.15s ease;
}

.cge-level-seg--active {
  color: #fff;
  background: #00b96b;
}

.cge-level-seg:disabled {
  cursor: not-allowed;
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

.cge-body--bracket::before {
  content: '';
  position: absolute;
  left: 13px;
  top: 0;
  bottom: 0;
  width: 5px;
  border-left: 2px solid #e0e3e8;
  border-top: 2px solid #e0e3e8;
  border-bottom: 2px solid #e0e3e8;
  border-radius: 4px 0 0 4px;
  z-index: 0;
  pointer-events: none;
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
