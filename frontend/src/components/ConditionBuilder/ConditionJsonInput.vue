<template>
  <div
    ref="hostEl"
    class="cji-editor"
    :class="{ 'cji-editor--error': props.error }"
    :style="{ '--cji-height': `${props.height ?? 300}px` }"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditorState, Prec } from '@codemirror/state'
import type { Extension } from '@codemirror/state'
import {
  EditorView,
  crosshairCursor,
  drawSelection,
  dropCursor,
  highlightActiveLine,
  highlightActiveLineGutter,
  highlightSpecialChars,
  keymap,
  lineNumbers,
  placeholder as cmPlaceholder,
  rectangularSelection,
  tooltips,
} from '@codemirror/view'
import type { Command } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, insertNewlineAndIndent } from '@codemirror/commands'
import {
  HighlightStyle,
  bracketMatching,
  foldGutter,
  indentOnInput,
  indentUnit,
  syntaxHighlighting,
} from '@codemirror/language'
import {
  autocompletion,
  closeBrackets,
  closeBracketsKeymap,
  completionKeymap,
} from '@codemirror/autocomplete'
import type { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import { highlightSelectionMatches, searchKeymap } from '@codemirror/search'
import { json, jsonParseLinter } from '@codemirror/lang-json'
import { lintGutter, lintKeymap, linter } from '@codemirror/lint'
import { tags } from '@lezer/highlight'
import dayjs from 'dayjs'
import {
  conditionFieldGroupMap,
  flattenFieldGroups,
  resolveConditionFieldType,
} from './conditionOperator'
import type {
  ConditionFieldGroup,
  ConditionFieldOption,
  ConditionJsonInputEmits,
  ConditionJsonInputProps,
  ConditionValueType,
} from './ConditionBuilder.types'

const props = defineProps<ConditionJsonInputProps>()
const emit = defineEmits<ConditionJsonInputEmits>()

const INDENT_UNIT = '  '

const VALUE_TYPE_LABELS: Record<ConditionValueType, string> = {
  string: '文本',
  number: '数值',
  boolean: '布尔',
  date: '日期',
  datetime: '日期时间',
  time: '时间',
  select: '枚举',
}

const hostEl = ref<HTMLDivElement | null>(null)
let view: EditorView | null = null

const fields = computed<ConditionFieldOption[]>(() =>
  props.fieldOptions && props.fieldOptions.length
    ? props.fieldOptions
    : flattenFieldGroups(props.fieldGroups ?? []),
)

const fieldGroupMap = computed<Map<string, ConditionFieldGroup>>(() =>
  conditionFieldGroupMap(props.fieldGroups ?? []),
)

function fieldOf(keyValue: string): ConditionFieldOption | undefined {
  return fields.value.find((field) => field.value === keyValue)
}

function fieldGroupSuffix(keyValue: string): string {
  const group = fieldGroupMap.value.get(keyValue)
  return group ? ` · ${group.label}` : ''
}

function applyCompletion(
  target: EditorView,
  completion: Completion,
  from: number,
  to: number,
): void {
  const following = target.state.doc.sliceString(to, Math.min(to + 1, target.state.doc.length))
  const end = following === '"' ? to + 1 : to
  target.dispatch({
    changes: { from, to: end, insert: completion.label },
    selection: { anchor: from + completion.label.length },
  })
}

function withApply(options: Completion[]): Completion[] {
  return options.map((option) => ({ ...option, apply: applyCompletion }))
}

function fieldCompletions(): Completion[] {
  return withApply(
    fields.value.map((field) => ({
      label: `"${field.value}"`,
      type: 'property',
      detail: `${field.label ?? field.value} · ${VALUE_TYPE_LABELS[resolveConditionFieldType(field)]}${fieldGroupSuffix(field.value)}`,
    })),
  )
}

function valueCompletions(field?: ConditionFieldOption): Completion[] {
  if (!field) return []
  if (field.options && field.options.length > 0) {
    return withApply(
      field.options.map((option) => ({
        label: `"${option.value}"`,
        type: 'constant',
        detail: `取值 · ${option.label}`,
      })),
    )
  }
  const type = resolveConditionFieldType(field)
  if (type === 'boolean') {
    return withApply([
      { label: 'true', type: 'constant', detail: '取值 · 是 / 已认证' },
      { label: 'false', type: 'constant', detail: '取值 · 否 / 未认证' },
    ])
  }
  if (type === 'date') {
    return withApply([
      { label: `"${dayjs().format('YYYY-MM-DD')}"`, type: 'constant', detail: '取值 · 今天' },
    ])
  }
  if (type === 'datetime') {
    return withApply([
      {
        label: `"${dayjs().format('YYYY-MM-DD')} 00:00:00"`,
        type: 'constant',
        detail: '取值 · 今天 00:00:00',
      },
    ])
  }
  if (type === 'time') {
    return withApply([
      { label: '"09:30:00"', type: 'constant', detail: '取值 · 示例时刻 HH:mm:ss' },
    ])
  }
  return []
}

const KEYWORD_COMPLETIONS: Completion[] = withApply([
  { label: 'true', type: 'keyword', detail: '布尔真值' },
  { label: 'false', type: 'keyword', detail: '布尔假值' },
  { label: 'null', type: 'keyword', detail: '空值' },
])

function findOpenQuote(before: string): number {
  let inString = false
  let start = -1
  for (let i = 0; i < before.length; i += 1) {
    const char = before[i]
    if (char === '\\') {
      i += 1
      continue
    }
    if (char === '"') {
      inString = !inString
      start = inString ? i : -1
    }
  }
  return inString ? start : -1
}

function findKeyBeforeColon(segment: string): string {
  const trimmed = segment.replace(/\s+$/, '')
  if (!trimmed.endsWith('"')) return ''
  const rest = trimmed.slice(0, -1)
  const start = rest.lastIndexOf('"')
  if (start < 0) return ''
  return rest.slice(start + 1)
}

function closingStringStart(text: string, endQuote: number): number {
  let i = endQuote - 1
  while (i >= 0) {
    if (text[i] === '"') {
      let backslashes = 0
      let j = i - 1
      while (j >= 0 && text[j] === '\\') {
        backslashes += 1
        j -= 1
      }
      if (backslashes % 2 === 0) return i
    }
    i -= 1
  }
  return -1
}

function isValueEnd(before: string): boolean {
  const trimmed = before.replace(/\s+$/, '')
  if (!trimmed) return false
  const last = trimmed.slice(-1)
  if (last === '}' || last === ']') return true
  if (last === '"') {
    const start = closingStringStart(trimmed, trimmed.length - 1)
    if (start < 0) return false
    return trimmed.slice(0, start).replace(/\s+$/, '').endsWith(':')
  }
  return /[0-9el]/.test(last)
}

interface JsonContext {
  kind: 'field' | 'value' | 'plain'
  from: number
  frag: string
  field?: ConditionFieldOption
}

function resolveContext(text: string, pos: number): JsonContext {
  const before = text.slice(0, pos)
  const openQuote = findOpenQuote(before)

  if (openQuote >= 0) {
    const frag = before.slice(openQuote + 1)
    const trimmed = before.slice(0, openQuote).replace(/\s+$/, '')
    if (trimmed.endsWith(':')) {
      return {
        kind: 'value',
        from: openQuote,
        frag,
        field: fieldOf(findKeyBeforeColon(trimmed.slice(0, -1))),
      }
    }
    return { kind: 'field', from: openQuote, frag }
  }

  const trimmed = before.replace(/\s+$/, '')
  if (trimmed.endsWith(':')) {
    return {
      kind: 'value',
      from: pos,
      frag: '',
      field: fieldOf(findKeyBeforeColon(trimmed.slice(0, -1))),
    }
  }
  if (trimmed.endsWith('{') || trimmed.endsWith(',') || trimmed.endsWith('[')) {
    return { kind: 'field', from: pos, frag: '' }
  }

  const word = before.match(/[A-Za-z_][A-Za-z0-9_]*$/)
  return { kind: 'plain', from: word ? pos - word[0].length : pos, frag: word ? word[0] : '' }
}

function jsonCompletionSource(context: CompletionContext): CompletionResult | null {
  const text = context.state.doc.toString()
  const jsonContext = resolveContext(text, context.pos)

  let options: Completion[]
  if (jsonContext.kind === 'field') {
    options = fieldCompletions()
  } else if (jsonContext.kind === 'value') {
    options = valueCompletions(jsonContext.field)
  } else {
    if (!jsonContext.frag) return null
    options = KEYWORD_COMPLETIONS
  }

  const frag = jsonContext.frag.toLowerCase()
  if (frag) {
    options = options.filter((option) => {
      const plain = option.label.replace(/^"|"$/g, '').toLowerCase()
      return plain !== frag && plain.startsWith(frag)
    })
  }

  if (options.length === 0) return null
  return { from: jsonContext.from, options: options.slice(0, 12) }
}

const smartEnter: Command = (target) => {
  const { state } = target
  const selection = state.selection.main
  if (!selection.empty) return false

  const pos = selection.head
  const text = state.doc.toString()
  const before = text.slice(0, pos)

  if (findOpenQuote(before) >= 0) return false
  if (!isValueEnd(before)) return false

  const head = before.replace(/\s+$/, '')
  const after = text.slice(pos)
  const nextChar = after.replace(/^\s+/, '').charAt(0)
  const needComma = nextChar !== ',' && nextChar !== '}' && nextChar !== ']'

  const comma = needComma ? ',' : ''
  target.dispatch({
    changes: { from: head.length, to: pos, insert: comma },
    selection: { anchor: head.length + comma.length },
  })
  return insertNewlineAndIndent(target)
}

const jsonHighlightStyle = HighlightStyle.define([
  { tag: tags.propertyName, color: '#5856d6' },
  { tag: tags.string, color: '#1a7f37' },
  { tag: tags.number, color: '#c2410c' },
  { tag: tags.bool, color: '#007aff' },
  { tag: tags.null, color: '#8e8e93' },
  { tag: tags.punctuation, color: '#86868b' },
  { tag: tags.invalid, color: '#ff3b30' },
])

const editorTheme = EditorView.theme({
  '&.cm-editor': {
    height: 'var(--cji-height, 300px)',
    fontSize: '12px',
    border: '1px solid rgba(0, 0, 0, 0.07)',
    borderRadius: '10px',
    backgroundColor: '#f5f5f7',
    color: '#1d1d1f',
    overflow: 'hidden',
    transition: 'border-color 0.18s, box-shadow 0.18s, background-color 0.18s',
  },
  '&.cm-editor.cm-focused': {
    outline: 'none',
    borderColor: '#007aff',
    boxShadow: '0 0 0 3px rgba(0, 122, 255, 0.16)',
    backgroundColor: '#ffffff',
  },
  '.cm-scroller': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
    lineHeight: '1.65',
    overflow: 'auto',
  },
  '.cm-content': {
    padding: '10px 0',
    caretColor: '#007aff',
  },
  '.cm-line': {
    padding: '0 12px 0 6px',
  },
  '.cm-gutters': {
    backgroundColor: 'transparent',
    border: 'none',
    color: '#aeaeb2',
    paddingLeft: '4px',
  },
  '.cm-activeLine': { backgroundColor: 'rgba(0, 122, 255, 0.04)' },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: '#007aff' },
  '.cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'rgba(0, 122, 255, 0.15)',
  },
  '&.cm-focused .cm-selectionBackground, &.cm-focused .cm-content ::selection': {
    backgroundColor: 'rgba(0, 122, 255, 0.22)',
  },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: '#007aff', borderLeftWidth: '2px' },
  '.cm-matchingBracket': {
    backgroundColor: 'rgba(0, 122, 255, 0.14)',
    outline: 'none',
  },
  '.cm-nonmatchingBracket': { color: '#ff3b30' },
  '.cm-tooltip': {
    border: '1px solid rgba(0, 0, 0, 0.06)',
    borderRadius: '10px',
    backgroundColor: 'rgba(255, 255, 255, 0.98)',
    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.12)',
    backdropFilter: 'blur(20px) saturate(180%)',
    overflow: 'hidden',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul': {
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
    maxHeight: '240px',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li': { padding: '4px 10px', fontSize: '12px' },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: 'rgba(0, 122, 255, 0.08)',
    color: '#1d1d1f',
  },
  '.cm-completionLabel': { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' },
  '.cm-completionDetail': {
    color: '#86868b',
    fontStyle: 'normal',
    marginLeft: '8px',
    fontSize: '11px',
  },
  '.cm-lintRange-error': {
    backgroundImage: 'none',
    borderBottom: '1.5px dashed #ff3b30',
    paddingBottom: '1px',
  },
  '.cm-lintPoint-error::after': {
    content: '""',
    position: 'absolute',
    top: '0',
    left: '0',
    width: '6px',
    height: '6px',
    transform: 'translate(-50%, -50%) rotate(45deg)',
    borderRadius: '1px',
    backgroundColor: '#ff3b30',
    border: '1px solid #ff3b30',
  },
  '.cm-lint-marker-error': { color: '#ff3b30' },
  '.cm-placeholder': { color: '#86868b' },
})

function createExtensions(): Extension[] {
  return [
    lineNumbers(),
    lintGutter(),
    highlightActiveLineGutter(),
    highlightSpecialChars(),
    history(),
    foldGutter(),
    drawSelection(),
    dropCursor(),
    EditorState.allowMultipleSelections.of(true),
    indentOnInput(),
    indentUnit.of(INDENT_UNIT),
    bracketMatching(),
    closeBrackets(),
    autocompletion({ override: [jsonCompletionSource] }),
    rectangularSelection(),
    crosshairCursor(),
    highlightActiveLine(),
    highlightSelectionMatches(),
    json(),
    linter((target) => (target.state.doc.length > 0 ? jsonParseLinter()(target) : [])),
    tooltips({ parent: document.body }),
    syntaxHighlighting(jsonHighlightStyle),
    editorTheme,
    cmPlaceholder(props.placeholder ?? ''),
    Prec.high(keymap.of([{ key: 'Enter', run: smartEnter }])),
    keymap.of([
      ...closeBracketsKeymap,
      ...defaultKeymap,
      ...searchKeymap,
      ...historyKeymap,
      ...completionKeymap,
      ...lintKeymap,
    ]),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        emit('update:modelValue', update.state.doc.toString())
      }
    }),
  ]
}

onMounted(() => {
  if (!hostEl.value) return
  view = new EditorView({
    state: EditorState.create({
      doc: props.modelValue,
      extensions: createExtensions(),
    }),
    parent: hostEl.value,
  })
})

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})

watch(
  () => props.modelValue,
  (value) => {
    if (!view) return
    const current = view.state.doc.toString()
    if (value === current) return
    view.dispatch({ changes: { from: 0, to: current.length, insert: value } })
  },
)
</script>

<style scoped>
.cji-editor {
  position: relative;
}

.cji-editor--error :deep(.cm-editor) {
  border-color: #ff3b30;
}

.cji-editor--error :deep(.cm-editor.cm-focused) {
  border-color: #ff3b30;
  box-shadow: 0 0 0 3px rgba(255, 59, 48, 0.16);
}
</style>
