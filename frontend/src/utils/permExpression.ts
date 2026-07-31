export interface PermContext {
  currentUser?: { id: string; role: string }
  user_id?: string
  account?: { user_id: string }
  content?: { user_id: string }
  target_user?: { id: string; role: string }
  current_role_ids?: string[]
  target_role_ids?: string[]
  [key: string]: any
}

type TokenType = 'PERM' | 'FUNC' | 'IDENT' | 'OR' | 'AND' | 'NOT' | 'LPAREN' | 'RPAREN' | 'EOF'

interface Token {
  type: TokenType
  value: string
  pos: number
}

const BUILTIN_FUNC_NAMES = new Set([
  'isSelf',
  'isAdmin',
  'isSuperAdmin',
  'isBuiltInAdmin',
  'isOwnAccount',
  'isOwnContent',
  'hasSameRole',
])

function _isExpressionChar(ch: string): boolean {
  return ch === '|' || ch === '&' || ch === '(' || ch === ')' || ch === '!'
}

export function isExpression(source: string): boolean {
  for (let i = 0; i < source.length; i++) {
    if (_isExpressionChar(source[i])) return true
  }
  return false
}

function _isIdentChar(ch: string): boolean {
  return /[a-zA-Z0-9_]/.test(ch)
}

function _isIdentStart(ch: string): boolean {
  return /[a-zA-Z_]/.test(ch)
}

class Tokenizer {
  private input: string
  private pos: number

  constructor(input: string) {
    this.input = input
    this.pos = 0
  }

  private skipWhitespace(): void {
    while (this.pos < this.input.length && this.input[this.pos] === ' ') {
      this.pos++
    }
  }

  private readIdent(): string {
    const start = this.pos
    while (this.pos < this.input.length && _isIdentChar(this.input[this.pos])) {
      this.pos++
    }
    return this.input.slice(start, this.pos)
  }

  private readColons(): void {
    while (this.pos < this.input.length && this.input[this.pos] === ':') {
      this.pos++
    }
  }

  next(): Token {
    this.skipWhitespace()
    if (this.pos >= this.input.length) {
      return { type: 'EOF', value: '', pos: this.pos }
    }

    const ch = this.input[this.pos]

    if (ch === '(') {
      this.pos++
      return { type: 'LPAREN', value: '(', pos: this.pos - 1 }
    }
    if (ch === ')') {
      this.pos++
      return { type: 'RPAREN', value: ')', pos: this.pos - 1 }
    }
    if (ch === '&') {
      this.pos++
      return { type: 'AND', value: '&', pos: this.pos - 1 }
    }
    if (ch === '|') {
      const start = this.pos
      this.pos++
      if (this.pos < this.input.length && this.input[this.pos] === '|') {
        this.pos++
        return { type: 'OR', value: '||', pos: start }
      }
      throw new Error(`期望 '||'，位置 ${start} 仅有单个 '|'`)
    }
    if (ch === '!') {
      this.pos++
      return { type: 'NOT', value: '!', pos: this.pos - 1 }
    }

    if (_isIdentStart(ch)) {
      const start = this.pos
      const ident = this.readIdent()

      this.skipWhitespace()

      if (this.pos < this.input.length && this.input[this.pos] === ':') {
        const permStart = start
        const parts: string[] = [ident]
        while (this.pos < this.input.length && this.input[this.pos] === ':') {
          this.readColons()
          if (this.pos >= this.input.length || !_isIdentStart(this.input[this.pos])) break
          parts.push(this.readIdent())
        }
        return { type: 'PERM', value: parts.join(':'), pos: permStart }
      }

      if (this.pos < this.input.length && this.input[this.pos] === '(') {
        return { type: 'FUNC', value: ident, pos: start }
      }

      if (BUILTIN_FUNC_NAMES.has(ident)) {
        return { type: 'FUNC', value: ident, pos: start }
      }

      return { type: 'PERM', value: ident, pos: start }
    }

    throw new Error(`位置 ${this.pos} 出现意外字符 '${ch}'`)
  }
}

type BuiltinFunc = (args: string[], ctx: PermContext) => boolean

const BUILTIN_FUNCTIONS: Record<string, BuiltinFunc> = {
  isSelf(args, ctx) {
    const keyName = args[0] ?? 'user_id'
    const targetId = (ctx as any)[keyName] ?? ctx.user_id
    if (!targetId) return false
    return ctx.currentUser.id === targetId
  },

  isAdmin(_args, ctx) {
    const role = ctx.currentUser.role
    return role === 'admin' || role === 'manager'
  },

  isSuperAdmin(_args, ctx) {
    const u = ctx.currentUser
    return u.role === 'admin' && u.id === '1'
  },

  isBuiltInAdmin(args, ctx) {
    const keyName = args[0] ?? 'user_id'
    const targetId = (ctx as any)[keyName] ?? ctx.user_id
    if (!targetId) return false
    return targetId === '1'
  },

  isOwnAccount(args, ctx) {
    const account = ctx.account
    if (!account) return false
    return ctx.currentUser.id === account.user_id
  },

  isOwnContent(args, ctx) {
    const content = ctx.content
    if (!content) return false
    return ctx.currentUser.id === content.user_id
  },

  hasSameRole(args, ctx) {
    const targetUser = ctx.target_user
    if (!targetUser) return false
    if (targetUser.id === ctx.currentUser.id) return true
    const currentRoles = new Set(ctx.current_role_ids || [])
    const targetRoles = new Set(ctx.target_role_ids || [])
    if (currentRoles.size === 0 || targetRoles.size === 0) return false
    for (const r of currentRoles) {
      if (targetRoles.has(r)) return true
    }
    return false
  },
}

class Parser {
  private tokenizer: Tokenizer
  private current: Token
  private permKeys: Set<string>
  private funcs: Record<string, BuiltinFunc>
  private ctx: PermContext

  constructor(
    source: string,
    permKeys: Set<string>,
    ctx: PermContext,
    funcs?: Record<string, BuiltinFunc>,
  ) {
    this.tokenizer = new Tokenizer(source)
    this.current = this.tokenizer.next()
    this.permKeys = permKeys
    this.ctx = ctx
    this.funcs = funcs ?? BUILTIN_FUNCTIONS
  }

  private eat(type: TokenType): Token {
    const token = this.current
    if (token.type !== type) {
      throw new Error(`位置 ${token.pos}: 期望 ${type}，实际得到 ${token.type} '${token.value}'`)
    }
    this.current = this.tokenizer.next()
    return token
  }

  parse(): boolean {
    const result = this.expr()
    if (this.current.type !== 'EOF') {
      throw new Error(`位置 ${this.current.pos}: 多余的 token '${this.current.value}'`)
    }
    return result
  }

  private expr(): boolean {
    let left = this.andExpr()
    while (this.current.type === 'OR') {
      this.eat('OR')
      const right = this.andExpr()
      left = left || right
    }
    return left
  }

  private andExpr(): boolean {
    let left = this.atom()
    while (this.current.type === 'AND') {
      this.eat('AND')
      const right = this.atom()
      left = left && right
    }
    return left
  }

  private atom(): boolean {
    if (this.current.type === 'NOT') {
      this.eat('NOT')
      return !this.atom()
    }

    if (this.current.type === 'LPAREN') {
      this.eat('LPAREN')
      const result = this.expr()
      this.eat('RPAREN')
      return result
    }

    if (this.current.type === 'PERM') {
      const key = this.current.value
      this.eat('PERM')
      return this.permKeys.has(key)
    }

    if (this.current.type === 'FUNC') {
      return this.functionCall()
    }

    throw new Error(
      `位置 ${this.current.pos}: 期望权限 key、函数调用、'!' 或 '('，实际得到 '${this.current.value}'`,
    )
  }

  private functionCall(): boolean {
    const funcName = this.current.value
    this.eat('FUNC')

    const fn = this.funcs[funcName]
    if (!fn) {
      throw new Error(`未知的内置函数: ${funcName}`)
    }

    if (this.current.type === 'LPAREN') {
      this.eat('LPAREN')
      const args: string[] = []
      if (this.current.type === 'IDENT' || this.current.type === 'PERM') {
        args.push(this.current.value)
        this.eat(this.current.type)
      }
      this.eat('RPAREN')
      return fn(args, this.ctx)
    }

    return fn([], this.ctx)
  }
}

function _buildPermKeySet(permissions: Record<string, string>): Set<string> {
  return new Set(Object.keys(permissions))
}

export function evaluatePermission(
  expression: string,
  permissions: Record<string, string>,
  context: PermContext,
): boolean {
  if (!isExpression(expression)) {
    return expression in permissions
  }

  const parser = new Parser(expression, _buildPermKeySet(permissions), context)
  return parser.parse()
}
