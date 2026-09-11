<template>
  <div class="cge" :class="isRoot ? 'cge--root' : 'cge--nested'">
    <div class="cge-header">
      <span v-if="isRoot" class="cge-header__label">条件组合</span>
      <span v-else class="cge-header__badge">条件组</span>

      <span v-if="isRoot" class="cge-header__hint">
        每行左侧可切换 且 / 或（且 优先级高于 或）
      </span>

      <div class="cge-header__spacer" />

      <slot v-if="isRoot" name="header-actions" />
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

    <div class="cge-body" :class="{ 'cge-body--connected': group.children.length > 1 }">
      <template v-for="(child, index) in group.children" :key="child.id">
        <div v-if="isConditionGroup(child)" class="cge-node">
          <ConditionConnector
            :index="index"
            :logic="child.logic"
            :disabled="disabled"
            :editable="logicEditable"
            @update:logic="onNodeLogicChange(index, $event)"
          />
          <ConditionGroupEditor
            :group="asConditionGroup(child)"
            :path="childPath(index)"
            :depth="depth + 1"
            :field-options="fieldOptions"
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
          :item="asConditionItem(child)"
          :path="path"
          :index="index"
          :field-options="fieldOptions"
          :rule-context="ruleContext"
          :disabled="disabled"
          :logic-editable="logicEditable"
          @command="emitCommand"
        />
      </template>

      <div v-if="!group.children.length" class="cge-empty">{{ emptyText }}</div>
    </div>

    <div class="cge-footer">
      <a-button
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
        v-if="canAddGroup"
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
        v-if="isRoot && group.children.length"
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
  addText: '添加条件',
  addGroupText: '添加子条件组',
  clearText: '清空',
  emptyText: '暂无判断条件，点击下方按钮添加',
  ungroupText: '解组',
})

const emit = defineEmits<ConditionGroupEditorEmits>()

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
