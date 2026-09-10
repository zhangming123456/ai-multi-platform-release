<template>
  <div class="log-terminal animate-fade-up">
    <div class="log-terminal__bar">
      <div class="flex items-center gap-2">
        <span class="log-dot log-dot--r"></span>
        <span class="log-dot log-dot--y"></span>
        <span class="log-dot log-dot--g"></span>
        <IconCode :size="14" class="ml-2" style="color: #8e8e93" />
        <span class="log-terminal__title">API 调用日志</span>
        <span v-if="isGenerating" class="log-live">
          <span class="log-live__pulse"></span>
          实时监听中
        </span>
        <span v-else-if="logs.length > 0" class="log-idle">空闲</span>
      </div>
      <div class="flex items-center gap-1">
        <span class="log-terminal__count">{{ logs.length }} 条</span>
        <button class="log-terminal__btn" title="清空日志" @click="emit('clear')">
          <IconDelete :size="13" />
        </button>
      </div>
    </div>

    <div class="log-terminal__body-wrap">
      <div ref="panelEl" class="log-terminal__body">
        <div v-if="logs.length === 0" class="log-terminal__empty">
          <span class="log-terminal__prompt">➜</span>
          暂无调用记录，点击「AI 生成内容」后这里将实时输出接口日志
        </div>
        <div
          v-for="entry in logs"
          :key="entry.id"
          v-memo="[entry.id]"
          class="log-line"
          :class="`log-line--${entry.level}`"
        >
          <span class="log-line__time">{{ entry.time }}</span>
          <span class="log-line__level">{{ levelText(entry.level) }}</span>
          <span class="log-line__msg">{{ entry.message }}</span>
        </div>
        <div v-if="isGenerating" class="log-line log-line--cursor">
          <span class="log-terminal__prompt">➜</span>
          <span class="log-cursor"></span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { IconCode, IconDelete } from '@arco-design/web-vue/es/icon'
import type {
  LogLevel,
  ContentLogTerminalProps,
  ContentLogTerminalEmits,
} from './ContentLogTerminal.types'

const props = defineProps<ContentLogTerminalProps>()

const emit = defineEmits<ContentLogTerminalEmits>()

const panelEl = ref<HTMLElement | null>(null)

// 用 rAF 合并高频日志的滚动，避免每条日志都触发一次强制重排
let scrollRaf = 0
watch(
  () => props.logs.length,
  () => {
    if (scrollRaf) return
    scrollRaf = requestAnimationFrame(() => {
      scrollRaf = 0
      const panel = panelEl.value
      if (panel) panel.scrollTop = panel.scrollHeight
    })
  },
)

onUnmounted(() => cancelAnimationFrame(scrollRaf))

function levelText(level: LogLevel) {
  switch (level) {
    case 'req':
      return 'REQ '
    case 'ok':
      return 'OK  '
    case 'err':
      return 'ERR '
    default:
      return 'INFO'
  }
}
</script>

<style scoped lang="scss">
.log-terminal {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-radius: 12px;
  overflow: hidden;
  background: #1d1d1f;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.28),
    0 2px 8px rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.log-terminal__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: linear-gradient(180deg, #2c2c2e 0%, #252527 100%);
  user-select: none;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.log-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}
.log-dot--r {
  background: #ff5f57;
}
.log-dot--y {
  background: #febc2e;
}
.log-dot--g {
  background: #28c840;
}

.log-terminal__title {
  font-size: 12px;
  font-weight: 600;
  color: #d1d1d6;
  letter-spacing: 0.04em;
}

.log-live {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
  color: #34c759;
  margin-left: 8px;
  font-weight: 500;
}

.log-live__pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #34c759;
  animation: log-pulse 1.2s ease-in-out infinite;
}

@keyframes log-pulse {
  0%,
  100% {
    opacity: 1;
    box-shadow: 0 0 0 0 rgba(52, 199, 89, 0.5);
  }
  50% {
    opacity: 0.6;
    box-shadow: 0 0 0 4px rgba(52, 199, 89, 0);
  }
}

.log-idle {
  font-size: 10px;
  color: #636366;
  margin-left: 8px;
}

.log-terminal__count {
  font-size: 11px;
  color: #636366;
  font-variant-numeric: tabular-nums;
  margin-right: 6px;
}

.log-terminal__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: #8e8e93;
  cursor: pointer;
  transition:
    background 0.15s ease,
    color 0.15s ease;
}

.log-terminal__btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}

.log-terminal__body-wrap {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.log-terminal__body {
  height: 100%;
  max-height: 100%;
  min-height: 0;
  overflow-y: auto;
  padding: 12px 14px;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, 'Cascadia Code', 'Roboto Mono', monospace;
  font-size: 12px;
  line-height: 1.7;
  background:
    radial-gradient(ellipse 60% 40% at 80% 0%, rgba(0, 122, 255, 0.05), transparent), #1d1d1f;
}

.log-terminal__empty {
  color: #48484a;
  font-size: 12px;
}

.log-terminal__prompt {
  color: #34c759;
  margin-right: 6px;
}

.log-line {
  display: flex;
  align-items: baseline;
  gap: 10px;
  animation: log-in 0.25s cubic-bezier(0.25, 0.1, 0.25, 1) both;
  padding: 1px 0;
}

@keyframes log-in {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.log-line__time {
  color: #48484a;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.log-line__level {
  flex-shrink: 0;
  font-weight: 700;
  font-size: 10px;
  letter-spacing: 0.08em;
  padding: 1px 6px;
  border-radius: 4px;
  white-space: pre;
}

.log-line--info .log-line__level {
  color: #8e8e93;
  background: rgba(142, 142, 147, 0.12);
}
.log-line--info .log-line__msg {
  color: #aeaeb2;
}

.log-line--req .log-line__level {
  color: #ff9f0a;
  background: rgba(255, 159, 10, 0.12);
}
.log-line--req .log-line__msg {
  color: #ffd60a;
}

.log-line--ok .log-line__level {
  color: #30d158;
  background: rgba(48, 209, 88, 0.12);
}
.log-line--ok .log-line__msg {
  color: #6ee7a0;
}

.log-line--err .log-line__level {
  color: #ff453a;
  background: rgba(255, 69, 58, 0.14);
}
.log-line--err .log-line__msg {
  color: #ff6961;
}

.log-line__msg {
  word-break: break-all;
}

.log-cursor {
  display: inline-block;
  width: 7px;
  height: 14px;
  background: #34c759;
  animation: cursor-blink 0.9s steps(1) infinite;
  vertical-align: middle;
}

@keyframes cursor-blink {
  50% {
    opacity: 0;
  }
}

@media (max-width: 768px) {
  .log-terminal__body {
    max-height: 200px;
    font-size: 11px;
    padding: 10px 12px;
  }
  .log-terminal__bar {
    padding: 8px 12px;
  }
  .log-terminal__title {
    font-size: 11px;
  }
  .log-line {
    gap: 6px;
  }
  .log-line__time {
    font-size: 10.5px;
  }
  .log-line__level {
    font-size: 9px;
    padding: 1px 4px;
  }
}

@media (max-width: 480px) {
  .log-terminal__body {
    max-height: 160px;
  }
}
</style>
