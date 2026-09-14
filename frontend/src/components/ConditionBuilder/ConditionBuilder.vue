<template>
  <div class="condition-builder" :class="{ 'is-disabled': disabled }">
    <ConditionGroupEditor
      :group="currentGroup"
      :path="ROOT_PATH"
      :depth="0"
      :field-options="fieldOptions"
      :field-groups="fieldGroups"
      :scoped-groups="scopedGroups"
      :rule-context="ruleContext"
      :disabled="disabled"
      :logic-editable="logicEditable"
      :max-depth="maxDepth"
      :max-items="maxItems"
      :logic-mode="logicMode"
      :is-root="true"
      :add-text="addText"
      :add-group-text="addGroupText"
      :clear-text="clearText"
      :empty-text="emptyText"
      :ungroup-text="ungroupText"
      @command="handleCommand"
    >
      <template #header-actions>
        <el-tooltip placement="bottom-end">
          <el-button link size="small" class="cb-help" :icon="HelpCircle"> 可用变量 </el-button>
          <template #content>
            <div class="cb-var-tip">
              <div v-for="group in varGroups" :key="group.key" class="cb-var-tip__group">
                <div v-if="group.label" class="cb-var-tip__group-head">
                  <span class="cb-var-tip__group-name">{{ group.label }}</span>
                  <span class="cb-var-tip__group-count">{{ group.fields.length }}</span>
                </div>
                <div v-if="group.description" class="cb-var-tip__group-desc">
                  {{ group.description }}
                </div>
                <div v-for="field in group.fields" :key="field.value" class="cb-var-tip__row">
                  <div class="cb-var-tip__head">
                    <span class="cb-var-tip__name">{{ conditionFieldLabel(field) }}</span>
                    <code class="cb-var-tip__code">{{ field.value }}</code>
                    <span class="cb-var-tip__type">{{ valueTypeLabel(field.type) }}</span>
                    <span v-if="field.queryable === false" class="cb-var-tip__demo">不可查询</span>
                  </div>
                  <div v-if="field.description" class="cb-var-tip__desc">
                    {{ field.description }}
                  </div>
                </div>
              </div>
              <div v-if="!varGroups.length" class="cb-var-tip__empty">未配置可用变量</div>
            </div>
          </template>
        </el-tooltip>
      </template>
    </ConditionGroupEditor>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { HelpCircle } from 'lucide-vue-next'
import type {
  ConditionBuilderEmits,
  ConditionBuilderProps,
  ConditionCommand,
  ConditionFieldGroup,
  ConditionGroup,
  ConditionLogic,
  ConditionNode,
  ConditionRuleContext,
  ConditionValueType,
} from './ConditionBuilder.types'
import ConditionGroupEditor from './ConditionGroupEditor.vue'
import {
  CONDITION_MAX_DEPTH,
  CONDITION_VALUE_TYPE_LABELS,
  conditionFieldLabel,
  createConditionGroup,
  createConditionItem,
  createScopedConditionGroup,
  groupConditionFields,
  isConditionGroup,
  patchChildItem,
  patchChildLogic,
  pruneInactiveScopedGroups,
  regroupLockedConditions,
  removeNodeWithCollapse,
  setGroupLevelLogic,
  setGroupTreeLevelLogicFromFirst,
  ungroupNodeAt,
  updateGroupAt,
} from './conditionOperator'
import { resolveConditionRules } from './conditionRules'

const props = withDefaults(defineProps<ConditionBuilderProps>(), {
  modelValue: undefined,
  fieldOptions: () => [],
  fieldGroups: () => [],
  scopedGroups: () => [],
  ruleFieldOptions: undefined,
  rules: () => [],
  disabled: false,
  maxItems: 0,
  maxDepth: CONDITION_MAX_DEPTH,
  logicEditable: true,
  logicMode: 'uniform',
  addText: '添加条件',
  addGroupText: '添加子条件组',
  clearText: '清空',
  emptyText: '暂无判断条件，点击下方按钮添加',
  ungroupText: '解组',
})

const emit = defineEmits<ConditionBuilderEmits>()

const ROOT_PATH: number[] = []

const EMPTY_GROUP: ConditionGroup = {
  id: 'condition-root',
  nodeType: 'group',
  logic: 'and',
  children: [],
}

const currentGroup = computed<ConditionGroup>(() => props.modelValue ?? EMPTY_GROUP)

function isScopedActive(scope: string): boolean {
  const group = props.scopedGroups.find((entry) => entry.key === scope)
  return group ? group.active !== false : true
}

function isCollapsibleGroup(target: ConditionGroup): boolean {
  return !target.scope
}

function findScopedGroup(
  scope: string,
  groups: ConditionFieldGroup[],
): ConditionFieldGroup | undefined {
  return groups.find((entry) => entry.key === scope)
}

function isRetainedScopedNode(node: ConditionNode): boolean {
  if (!isConditionGroup(node) || !node.scope) return false
  const entry = findScopedGroup(node.scope, props.scopedGroups)
  return !!entry && entry.active === false
}

let previousActiveKeys: string[] = []

function activeScopedKeys(groups: ConditionFieldGroup[]): string[] {
  return groups.filter((entry) => entry.active !== false).map((entry) => entry.key)
}

function shouldPruneUnassigned(groups: ConditionFieldGroup[]): boolean {
  const activeKeys = activeScopedKeys(groups)
  const prune = previousActiveKeys.length === 1 && activeKeys.length >= 2
  previousActiveKeys = activeKeys
  return prune
}

function normalizeScopedModel(
  group: ConditionGroup,
  groups: ConditionFieldGroup[],
): ConditionGroup {
  const pruneUnassigned = shouldPruneUnassigned(groups)
  if (!groups.length) return flattenScopedGroups(group, groups)
  const active = groups.filter((entry) => entry.active !== false)
  if (active.length <= 1) return flattenScopedGroups(group, groups)
  const source = pruneUnassigned ? pruneUnassignedConditions(group, groups, active) : group
  return groupScopedModel(source, groups, active)
}

function pruneUnassignedConditions(
  group: ConditionGroup,
  groups: ConditionFieldGroup[],
  active: ConditionFieldGroup[],
): ConditionGroup {
  const ownerIndex = fieldOwnerIndex(groups)
  const activeKeys = active.map((entry) => entry.key)
  const children = pruneUnassignedNodes(group.children, ownerIndex, activeKeys)
  return children === group.children ? group : { ...group, children }
}

function pruneUnassignedNodes(
  nodes: ConditionNode[],
  ownerIndex: Map<string, string>,
  activeKeys: string[],
): ConditionNode[] {
  const children: ConditionNode[] = []
  let changed = false
  for (const node of nodes) {
    const kept = pruneUnassignedNode(node, ownerIndex, activeKeys)
    if (kept === null) {
      changed = true
      continue
    }
    if (kept !== node) changed = true
    children.push(kept)
  }
  return changed ? children : nodes
}

function pruneUnassignedNode(
  node: ConditionNode,
  ownerIndex: Map<string, string>,
  activeKeys: string[],
): ConditionNode | null {
  if (!isConditionGroup(node)) {
    if (!node.field) return null
    const owner = ownerIndex.get(node.field)
    if (!owner || !activeKeys.includes(owner)) return null
    return node
  }
  if (node.scope) return node
  const children = pruneUnassignedNodes(node.children, ownerIndex, activeKeys)
  if (!children.length) return null
  return children === node.children ? node : { ...node, children }
}

function isBlankItem(node: ConditionNode): boolean {
  return !isConditionGroup(node) && !node.field && !node.value.trim()
}

function unwrapScopedNode(node: ConditionGroup): ConditionNode[] {
  return node.children
    .filter((child) => !isBlankItem(child))
    .map((child, index) => (index === 0 ? { ...child, logic: node.logic } : child))
}

function unwrapRestGroup(node: ConditionGroup): ConditionNode[] {
  return node.children.map((child, index) =>
    index === 0 ? { ...child, logic: node.logic } : child,
  )
}

function flattenScopedGroups(group: ConditionGroup, groups: ConditionFieldGroup[]): ConditionGroup {
  let changed = false
  const children: ConditionNode[] = []
  for (const child of group.children) {
    if (isConditionGroup(child) && child.scope) {
      const entry = findScopedGroup(child.scope, groups)
      if (entry && entry.active === false) {
        children.push(child)
        continue
      }
      changed = true
      children.push(...unwrapScopedNode(child))
      continue
    }
    children.push(child)
  }
  if (!changed) return group
  return { ...group, children: children.length ? children : [createConditionItem()] }
}

function fieldOwnerIndex(groups: ConditionFieldGroup[]): Map<string, string> {
  const index = new Map<string, string>()
  for (const group of groups) {
    for (const field of group.fields) {
      if (!index.has(field.value)) index.set(field.value, group.key)
    }
  }
  return index
}

function collectNodeFields(node: ConditionNode): string[] {
  if (!isConditionGroup(node)) return node.field ? [node.field] : []
  return node.children.flatMap(collectNodeFields)
}

function nodeScopeKey(
  node: ConditionNode,
  ownerIndex: Map<string, string>,
  activeKeys: string[],
): string | undefined {
  const fields = collectNodeFields(node).filter(Boolean)
  if (!fields.length) return undefined
  const owners = new Set(fields.map((field) => ownerIndex.get(field)))
  if (owners.size !== 1) return undefined
  const owner = [...owners][0]
  return owner && activeKeys.includes(owner) ? owner : undefined
}

function groupScopedModel(
  group: ConditionGroup,
  groups: ConditionFieldGroup[],
  active: ConditionFieldGroup[],
): ConditionGroup {
  const activeKeys = active.map((entry) => entry.key)
  const ownerIndex = fieldOwnerIndex(groups)
  const unscoped: ConditionNode[] = []
  const retained: ConditionNode[] = []

  for (const child of group.children) {
    if (!isConditionGroup(child) || !child.scope) {
      unscoped.push(child)
      continue
    }
    if (activeKeys.includes(child.scope)) continue
    const entry = findScopedGroup(child.scope, groups)
    if (entry) {
      retained.push(child)
      continue
    }
    unscoped.push(...unwrapScopedNode(child))
  }

  const homed = new Map<string, ConditionNode[]>()
  const rest: ConditionNode[] = []
  for (const node of unscoped) {
    const key = nodeScopeKey(node, ownerIndex, activeKeys)
    if (!key) {
      if (isConditionGroup(node) && !node.scope) {
        rest.push(...unwrapRestGroup(node))
        continue
      }
      rest.push(node)
      continue
    }
    const list = homed.get(key) ?? []
    list.push(node)
    homed.set(key, list)
  }

  const scoped = active.map((entry) => {
    const existing = group.children.find(
      (child): child is ConditionGroup => isConditionGroup(child) && child.scope === entry.key,
    )
    const extra = homed.get(entry.key) ?? []
    if (existing && !extra.length) return existing
    return {
      ...(existing ?? createScopedConditionGroup(entry.key)),
      children: [...(existing?.children ?? []), ...extra],
    }
  })

  const children: ConditionNode[] = [...scoped, ...retained, ...rest]
  const stable =
    children.length === group.children.length &&
    children.every((child, index) => child === group.children[index])
  return stable ? group : { ...group, children }
}

const normalizedGroup = computed(() =>
  regroupLockedConditions(
    normalizeScopedModel(currentGroup.value, props.scopedGroups),
    props.fieldOptions,
    props.scopedGroups,
    props.maxDepth,
  ),
)

const ruleGroup = computed(() =>
  pruneInactiveScopedGroups(currentGroup.value, (scope) => isScopedActive(scope)),
)

const varGroups = computed(() => groupConditionFields(props.fieldOptions, props.fieldGroups))

const ruleFieldOptions = computed(() => props.ruleFieldOptions ?? props.fieldOptions)

const ruleContext = computed<ConditionRuleContext>(() =>
  resolveConditionRules(
    ruleGroup.value,
    props.rules,
    ruleFieldOptions.value,
    props.fieldOptions.map((field) => field.value),
  ),
)

watch(
  normalizedGroup,
  (next) => {
    if (next !== currentGroup.value) commit(next)
  },
  { immediate: true },
)

watch(
  () => props.logicMode,
  (mode, previousMode) => {
    if (mode !== 'uniform' || previousMode === 'uniform') return
    const current = normalizedGroup.value
    const next = setGroupTreeLevelLogicFromFirst(current)
    if (next !== current) commit(next)
  },
  { immediate: true },
)

function commit(next: ConditionGroup): void {
  emit('update:modelValue', next)
  emit('change', next)
}

function canNestGroup(path: number[]): boolean {
  return path.length < props.maxDepth
}

function nextNodeLogic(group: ConditionGroup): ConditionLogic {
  if (props.logicMode !== 'uniform') return 'and'
  const previous = [...group.children].reverse().find((child) => !isRetainedScopedNode(child))
  return previous?.logic ?? group.logic
}

function handleCommand(command: ConditionCommand): void {
  const group = currentGroup.value

  switch (command.type) {
    case 'set-logic': {
      const nodePath = command.path
      if (nodePath.length === 0) return
      const index = nodePath[nodePath.length - 1]
      commit(
        updateGroupAt(group, nodePath.slice(0, -1), (target) =>
          patchChildLogic(target, index, command.logic),
        ),
      )
      return
    }
    case 'set-level-logic':
      commit(
        updateGroupAt(group, command.path, (target) => setGroupLevelLogic(target, command.logic)),
      )
      return
    case 'add-item':
      commit(
        updateGroupAt(group, command.path, (target) => {
          const item = { ...createConditionItem(), logic: nextNodeLogic(target) }
          return {
            ...target,
            children: [...target.children, item],
          }
        }),
      )
      return
    case 'add-group':
      if (!canNestGroup(command.path)) return
      commit(
        updateGroupAt(group, command.path, (target) => {
          const groupNode = command.field
            ? createConditionGroup([createConditionItem(command.field)])
            : createConditionGroup()
          return {
            ...target,
            children: [...target.children, { ...groupNode, logic: nextNodeLogic(target) }],
          }
        }),
      )
      return
    case 'clear-group':
      commit(
        updateGroupAt(group, command.path, (target) => ({
          ...target,
          children: target.children.filter((child) => isRetainedScopedNode(child)),
        })),
      )
      return
    case 'remove-group': {
      if (!command.path.length) return
      const index = command.path[command.path.length - 1]
      commit(removeNodeWithCollapse(group, command.path.slice(0, -1), index, isCollapsibleGroup))
      return
    }
    case 'ungroup-group':
      commit(ungroupNodeAt(group, command.path))
      return
    case 'update-item':
      commit(
        updateGroupAt(group, command.path, (target) =>
          patchChildItem(target, command.index, command.patch),
        ),
      )
      return
    case 'remove-item':
      commit(removeNodeWithCollapse(group, command.path, command.index, isCollapsibleGroup))
      return
    case 'wrap-item':
      if (!canNestGroup(command.path)) return
      commit(
        updateGroupAt(group, command.path, (target) => {
          const item = target.children[command.index]
          if (!item || isConditionGroup(item)) return target
          const wrapped: ConditionGroup = {
            ...createConditionGroup([
              { ...item, logic: 'and' },
              createConditionItem(command.field ?? ''),
            ]),
            logic: item.logic,
          }
          const children = target.children.slice()
          children[command.index] = wrapped
          return { ...target, children }
        }),
      )
      return
  }
}

function valueTypeLabel(type?: ConditionValueType): string {
  return CONDITION_VALUE_TYPE_LABELS[type ?? 'string']
}

function addItem(): void {
  handleCommand({ type: 'add-item', path: ROOT_PATH })
}

function clear(): void {
  handleCommand({ type: 'clear-group', path: ROOT_PATH })
}

defineExpose({ addItem, clear })
</script>

<style scoped>
.condition-builder {
  border: 1px solid #e5e5ea;
  border-radius: 14px;
  background: #ffffff;
  padding: 14px;
}

.condition-builder.is-disabled {
  opacity: 0.6;
  pointer-events: none;
}

.cb-help {
  color: #86868b;
}

.cb-help:hover {
  color: #007aff;
}

.cb-var-tip {
  max-height: 280px;
  overflow-y: auto;
  min-width: 220px;
}

.cb-var-tip__group + .cb-var-tip__group {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.14);
}

.cb-var-tip__group-head {
  display: flex;
  align-items: center;
  gap: 6px;
}

.cb-var-tip__group-name {
  font-size: 12px;
  font-weight: 600;
  color: #ffffff;
}

.cb-var-tip__group-count {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.16);
  border-radius: 999px;
  padding: 0 6px;
}

.cb-var-tip__group-desc {
  margin-top: 2px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.5);
}

.cb-var-tip__row {
  padding: 4px 0;
}

.cb-var-tip__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cb-var-tip__name {
  font-size: 12px;
  font-weight: 600;
  color: #ffffff;
}

.cb-var-tip__code {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.7);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.cb-var-tip__type {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.85);
  background: rgba(255, 255, 255, 0.16);
  border-radius: 4px;
  padding: 1px 5px;
}

.cb-var-tip__demo {
  font-size: 10px;
  color: rgba(255, 214, 10, 0.95);
  background: rgba(255, 214, 10, 0.18);
  border-radius: 4px;
  padding: 1px 5px;
}

.cb-var-tip__desc {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.6);
}

.cb-var-tip__empty {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}
</style>
