<template>
  <div class="panel">
    <div class="sticker-grid">
      <button
        v-for="(item, index) in stickers"
        :key="item"
        type="button"
        class="sticker-item"
        @click="handleAdd(item, index)"
      >
        <img :src="stickerUrls[index]" :alt="item" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { createEmojiSticker } from '../utils/image'
import type { PanelStickerEmits } from './panel-sticker.types'

const emit = defineEmits<PanelStickerEmits>()

const stickers = [
  '😀',
  '😂',
  '🤩',
  '😎',
  '🥰',
  '😭',
  '😡',
  '🤔',
  '👍',
  '👎',
  '👏',
  '🙏',
  '❤️',
  '💔',
  '💯',
  '⭐',
  '🔥',
  '🎉',
  '🎁',
  '🌸',
  '🌈',
  '☀️',
  '☕',
  '🍀',
]

const stickerUrls = computed(() => stickers.map((s) => createEmojiSticker(s)))

function handleAdd(_emoji: string, index: number) {
  emit('add', stickerUrls.value[index])
}
</script>

<style scoped>
.panel {
  display: flex;
}

.sticker-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 6px;
  width: 100%;
  max-height: 180px;
  overflow-y: auto;
}

.sticker-item {
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  padding: 4px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: #f5f5f7;
  cursor: pointer;
  transition: all 0.12s ease;
}

.sticker-item:hover {
  background: rgba(0, 122, 255, 0.08);
  border-color: #007aff;
  transform: scale(1.1);
}

.sticker-item img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
</style>
