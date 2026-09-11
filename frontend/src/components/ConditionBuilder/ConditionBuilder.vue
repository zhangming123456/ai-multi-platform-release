<template>
  <div class="condition-builder" :class="{ 'is-disabled': disabled }">
    <ConditionGroupEditor
      :group="currentGroup"
      :path="ROOT_PATH"
      :depth="1"
      :field-options="fieldOptions"
      :rule-context="ruleContext"
      :disabled="disabled"
      :logic-editable="logicEditable"
      :max-depth="maxDepth"
      :max-items="maxItems"
      :is-root="true"
      :add-text="addText"
      :add-group-text="addGroupText"
      :clear-text="clearText"
      :empty-text="emptyText"
      :ungroup-text="ungroupText"
      @command="handleCommand"
    >
      <template #header-actions>
        <a-tooltip position="br">
          <a-button type="text" size="mini" class="cb-help">
            <template #icon><IconQuestionCircle /></template>
            可用变量
          </a-button>
          <template #content>
            <div class="cb-var-tip">
              <div v-for="field in fieldOptions" :key="field.value" class="cb-var-tip__row">
                <div class="cb-var-tip__head">
                  <span class="cb-var-tip__name">{{ conditionFieldLabel(field) }}</span>
                  <code class="cb-var-tip__code">{{ field.value }}</code>
                  <span class="cb-var-tip__type">{{ valueTypeLabel(field.type) }}</span>
                </div>
                <div v-if="field.description" class="cb-var-tip__desc">{{ field.description }}</div>
              </div>
              <div v-if="!fieldOptions.length" class="cb-var-tip__empty">未配置可用变量</div>
            </div>
          </template>
        </a-tooltip>
      </template>
    </ConditionGroupEditor>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { IconQuestionCircle } from '@arco-design/web-vue/es/icon'
import type {
  ConditionBuilderEmits,
  ConditionBuilderProps,
  ConditionCommand,
  ConditionGroup,
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
  patchChildItem,
  patchChildLogic,
  removeNodeAt,
  ungroupNodeAt,
  updateGroupAt,
} from './conditionOperator'
import { resolveConditionRules } from './conditionRules'

const props = withDefaults(defineProps<ConditionBuilderProps>(), {
  modelValue: undefined,
  fieldOptions: () => [],
  rules: () => [],
  disabled: false,
  maxItems: 0,
  maxDepth: CONDITION_MAX_DEPTH,
  logicEditable: true,
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

const ruleContext = computed<ConditionRuleContext>(() =>
  resolveConditionRules(currentGroup.value, props.rules, props.fieldOptions),
)

function commit(next: ConditionGroup): void {
  emit('update:modelValue', next)
  emit('change', next)
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
    case 'add-item':
      commit(
        updateGroupAt(group, command.path, (target) => ({
          ...target,
          children: [...target.children, createConditionItem()],
        })),
      )
      return
    case 'add-group':
      commit(
        updateGroupAt(group, command.path, (target) => ({
          ...target,
          children: [...target.children, createConditionGroup()],
        })),
      )
      return
    case 'clear-group':
      commit(updateGroupAt(group, command.path, (target) => ({ ...target, children: [] })))
      return
    case 'remove-group':
      commit(removeNodeAt(group, command.path))
      return
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
      commit(
        updateGroupAt(group, command.path, (target) => ({
          ...target,
          children: target.children.filter((_, index) => index !== command.index),
        })),
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

.cb-var-tip__desc {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.6);
}

.cb-var-tip__empty {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}
</style>
