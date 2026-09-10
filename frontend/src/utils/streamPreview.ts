import { Marked } from 'marked'

const STRUCTURE_NOTE =
  '**结构说明**：`items[]` 中每一项对应一个「平台 × 内容形式」任务；每项包含 `variants[]` 文案变体，字段为 `title`（标题）、`body`（正文）、`hashtags`（话题标签）。'

export function escapeHtml(raw: string): string {
  return raw.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function stripFence(text: string): string {
  let out = text.replace(/^\s*```[a-zA-Z]*\s*\n?/, '')
  out = out.replace(/\n?[ \t]*```[ \t]*$/, '')
  return out
}

export function isJsonLike(text: string): boolean {
  const body = stripFence(text).trimStart()
  return body.startsWith('{') || body.startsWith('[')
}

export function formatStreamingJson(text: string): string {
  const raw = stripFence(text)
  const indent = '  '
  const len = raw.length
  let out = ''
  let depth = 0
  let inString = false
  let escaped = false

  for (let i = 0; i < len; i += 1) {
    const ch = raw.charAt(i)

    if (inString) {
      out += ch
      if (escaped) {
        escaped = false
      } else if (ch === '\\') {
        escaped = true
      } else if (ch === '"') {
        inString = false
      }
      continue
    }

    if (ch === '"') {
      inString = true
      out += ch
    } else if (ch === '{' || ch === '[') {
      depth += 1
      out += ch + '\n' + indent.repeat(depth)
    } else if (ch === '}' || ch === ']') {
      depth = Math.max(0, depth - 1)
      if (/\n[ \t]*$/.test(out)) {
        out = out.replace(/\n[ \t]*$/, '')
      } else {
        out += '\n' + indent.repeat(depth)
      }
      out += ch
    } else if (ch === ',') {
      out += ch + '\n' + indent.repeat(depth)
    } else if (ch === ':') {
      out += ': '
    } else if (ch !== ' ' && ch !== '\n' && ch !== '\r' && ch !== '\t') {
      out += ch
    }
  }

  return out.replace(/[ \t]+$/, '')
}

export function highlightJson(json: string): string {
  const len = json.length
  let html = ''
  let i = 0

  while (i < len) {
    const ch = json.charAt(i)

    if (ch === '"') {
      let j = i + 1
      let str = '"'
      while (j < len) {
        const c = json.charAt(j)
        if (c === '\\') {
          str += c + json.charAt(j + 1)
          j += 2
          continue
        }
        str += c
        j += 1
        if (c === '"') break
      }
      let k = j
      while (k < len && (json.charAt(k) === ' ' || json.charAt(k) === '\t')) k += 1
      const isKey = json.charAt(k) === ':'
      html += `<span class="${isKey ? 'tok-key' : 'tok-string'}">${escapeHtml(str)}</span>`
      i = j
      continue
    }

    if (ch === 't' || ch === 'f' || ch === 'n') {
      const rest = json.slice(i)
      const lit = rest.startsWith('true')
        ? 'true'
        : rest.startsWith('false')
          ? 'false'
          : rest.startsWith('null')
            ? 'null'
            : ''
      if (lit) {
        html += `<span class="${lit === 'null' ? 'tok-null' : 'tok-boolean'}">${lit}</span>`
        i += lit.length
        continue
      }
    }

    const isNumberStart =
      (ch >= '0' && ch <= '9') ||
      (ch === '-' && json.charAt(i + 1) >= '0' && json.charAt(i + 1) <= '9')
    if (isNumberStart) {
      let j = i
      while (j < len && '0123456789.eE+-'.includes(json.charAt(j))) j += 1
      html += `<span class="tok-number">${escapeHtml(json.slice(i, j))}</span>`
      i = j
      continue
    }

    if (ch === '\n') {
      html += '\n'
    } else if (ch === '{' || ch === '}' || ch === '[' || ch === ']' || ch === ',' || ch === ':') {
      html += `<span class="tok-punct">${ch}</span>`
    } else {
      html += ch === ' ' ? ' ' : escapeHtml(ch)
    }
    i += 1
  }

  return html
}

function pickFence(content: string): string {
  if (!content.includes('```')) return '```'
  if (!content.includes('````')) return '````'
  return '~~~~'
}

function countKey(pretty: string, key: string): number {
  return (pretty.match(new RegExp(`"${key}"\\s*:`, 'g')) ?? []).length
}

const markdown = new Marked({
  gfm: true,
  breaks: false,
  renderer: {
    code({ text, lang }) {
      if (lang !== 'json') return false
      return (
        '<div class="md-json">' +
        '<div class="md-json__bar">' +
        '<span class="md-json__badge">JSON</span>' +
        '<span class="md-json__live"><i class="md-json__dot"></i>实时生成</span>' +
        '</div>' +
        '<pre class="md-json__pre"><code class="md-json__code">' +
        highlightJson(text) +
        '</code></pre>' +
        '</div>'
      )
    },
    html() {
      return ''
    },
  },
})

export function renderStreamingJsonMarkdown(text: string): string {
  const pretty = formatStreamingJson(text)
  const tasks = countKey(pretty, 'platform')
  const variants = countKey(pretty, 'title')
  const fence = pickFence(pretty)
  const source =
    '> AI 正在实时生成结构化内容（JSON），以下为当前已接收的数据。\n\n' +
    `> 已接收 **${tasks}** 个任务、**${variants}** 条文案变体\n\n` +
    STRUCTURE_NOTE +
    '\n\n' +
    fence +
    'json\n' +
    pretty +
    '\n' +
    fence +
    '\n'
  return markdown.parse(source) as string
}
