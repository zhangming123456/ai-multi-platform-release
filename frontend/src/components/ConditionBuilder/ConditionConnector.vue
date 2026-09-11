<template>
  <div class="connector">
    <span v-if="index === 0" class="connector__first">IF</span>
    <div v-else-if="editable !== false && !disabled" class="connector__toggle">
      <button
        type="button"
        class="connector__seg"
        :class="{ 'connector__seg--active': logic === 'and' }"
        @click="setLogic('and')"
      >
        且
      </button>
      <button
        type="button"
        class="connector__seg"
        :class="{ 'connector__seg--active': logic === 'or' }"
        @click="setLogic('or')"
      >
        或
      </button>
    </div>
    <span v-else class="connector__static">{{ CONDITION_LOGIC_SYMBOL[logic] }}</span>
  </div>
</template>

<script setup lang="ts">
import type {
  ConditionConnectorEmits,
  ConditionConnectorProps,
  ConditionLogic,
} from './ConditionBuilder.types'
import { CONDITION_LOGIC_SYMBOL } from './conditionOperator'

defineProps<ConditionConnectorProps>()

const emit = defineEmits<ConditionConnectorEmits>()

function setLogic(value: ConditionLogic): void {
  emit('update:logic', value)
}
</script>

<style scoped>
.connector {
  flex: 0 0 28px;
  width: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.connector__first {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 22px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: #007aff;
  background: rgba(0, 122, 255, 0.1);
  border-radius: 6px;
}

.connector__toggle {
  display: flex;
  flex-direction: column;
  width: 28px;
  border-radius: 6px;
  overflow: hidden;
  background: #f2f3f5;
  box-shadow: 0 0 0 1px #e5e6eb;
}

.connector__seg {
  height: 22px;
  padding: 0;
  border: none;
  background: transparent;
  color: #86909c;
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
  cursor: pointer;
  transition:
    background 0.15s ease,
    color 0.15s ease;
}

.connector__seg--active {
  color: #fff;
  background: #00b96b;
}

.connector__static {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 22px;
  font-size: 11px;
  font-weight: 600;
  color: #5856d6;
  background: rgba(88, 86, 214, 0.1);
  border-radius: 6px;
}
</style>
