from __future__ import annotations

import re
from dataclasses import dataclass, field
from enum import Enum, auto
from typing import Any, Callable, Optional


class TokenType(Enum):
    PERM = auto()
    FUNC = auto()
    IDENT = auto()
    OR = auto()
    AND = auto()
    NOT = auto()
    LPAREN = auto()
    RPAREN = auto()
    EOF = auto()


@dataclass
class Token:
    type: TokenType
    value: str
    pos: int


_EXPRESSION_CHARS = {"|", "&", "(", ")", "!"}


def is_expression(source: str) -> bool:
    for ch in source:
        if ch in _EXPRESSION_CHARS:
            return True
    return False


_IDENT_RE = re.compile(r"[a-zA-Z_][a-zA-Z0-9_]*")


class Tokenizer:
    def __init__(self, source: str) -> None:
        self.source = source
        self.pos = 0

    def _skip_ws(self) -> None:
        while self.pos < len(self.source) and self.source[self.pos] == " ":
            self.pos += 1

    def _read_ident(self) -> str:
        m = _IDENT_RE.match(self.source, self.pos)
        if m is None:
            return ""
        self.pos = m.end()
        return m.group()

    def _read_perm_key(self) -> str:
        start = self.pos
        while self.pos < len(self.source) and (
            self.source[self.pos].isalnum() or self.source[self.pos] in {":", "_"}
        ):
            self.pos += 1
        return self.source[start : self.pos]

    def next(self) -> Token:
        self._skip_ws()
        if self.pos >= len(self.source):
            return Token(TokenType.EOF, "", self.pos)

        ch = self.source[self.pos]

        if ch == "(":
            self.pos += 1
            return Token(TokenType.LPAREN, "(", self.pos - 1)
        if ch == ")":
            self.pos += 1
            return Token(TokenType.RPAREN, ")", self.pos - 1)
        if ch == "&":
            self.pos += 1
            return Token(TokenType.AND, "&", self.pos - 1)
        if ch == "|":
            start = self.pos
            self.pos += 1
            if self.pos < len(self.source) and self.source[self.pos] == "|":
                self.pos += 1
                return Token(TokenType.OR, "||", start)
            raise SyntaxError(f"位置 {start}: 期望 '||'，仅有单个 '|'")
        if ch == "!":
            self.pos += 1
            return Token(TokenType.NOT, "!", self.pos - 1)

        if ch.isalpha() or ch == "_":
            start = self.pos
            ident = self._read_ident()

            self._skip_ws()

            if self.pos < len(self.source) and self.source[self.pos] == ":":
                perm_key = ident + self._read_perm_key()
                return Token(TokenType.PERM, perm_key, start)

            if self.pos < len(self.source) and self.source[self.pos] == "(":
                return Token(TokenType.FUNC, ident, start)

            builtin_names = {"isSelf", "isAdmin", "isSuperAdmin", "isBuiltInAdmin", "isOwnAccount", "isOwnContent", "hasSameRole"}
            if ident in builtin_names:
                return Token(TokenType.FUNC, ident, start)

            return Token(TokenType.PERM, ident, start)

        raise SyntaxError(f"位置 {self.pos}: 意外字符 '{ch}'")


BuiltinFunc = Callable[[list[str], dict[str, Any], dict[str, str], set[str]], bool]


def _eval_is_self(args: list[str], ctx: dict[str, Any], _perm_keys: dict[str, str], _all_role_ids: set[str]) -> bool:
    target_id = ctx.get(args[0]) if args else ctx.get("user_id")
    if not target_id:
        return False
    return ctx["current_user"].id == target_id


def _eval_is_admin(_args: list[str], ctx: dict[str, Any], _perm_keys: dict[str, str], _all_role_ids: set[str]) -> bool:
    user = ctx["current_user"]
    return user.role in {"admin", "manager"}


def _eval_is_super_admin(_args: list[str], ctx: dict[str, Any], _perm_keys: dict[str, str], _all_role_ids: set[str]) -> bool:
    user = ctx["current_user"]
    return user.role == "admin" and user.id == "1"


def _eval_is_built_in_admin(args: list[str], ctx: dict[str, Any], _perm_keys: dict[str, str], _all_role_ids: set[str]) -> bool:
    target_id = ctx.get(args[0]) if args else ctx.get("user_id")
    if not target_id:
        return False
    return target_id == "1"


def _eval_is_own_account(args: list[str], ctx: dict[str, Any], _perm_keys: dict[str, str], _all_role_ids: set[str]) -> bool:
    account = ctx.get(args[0]) if args else ctx.get("account")
    if account is None:
        return False
    return account.user_id == ctx["current_user"].id


def _eval_is_own_content(args: list[str], ctx: dict[str, Any], _perm_keys: dict[str, str], _all_role_ids: set[str]) -> bool:
    content = ctx.get(args[0]) if args else ctx.get("content")
    if content is None:
        return False
    return content.user_id == ctx["current_user"].id


def _eval_has_same_role(args: list[str], ctx: dict[str, Any], _perm_keys: dict[str, str], all_role_ids: set[str]) -> bool:
    target_user = ctx.get(args[0]) if args else ctx.get("target_user")
    if target_user is None:
        return False
    if target_user.id == ctx["current_user"].id:
        return True
    target_role_ids = set(ctx.get("target_role_ids", [])) if ctx.get("target_role_ids") else all_role_ids
    if not target_role_ids:
        return False
    current_role_ids = set(ctx.get("current_role_ids", []))
    return bool(current_role_ids & target_role_ids)


BUILTIN_FUNCTIONS: dict[str, BuiltinFunc] = {
    "isSelf": _eval_is_self,
    "isAdmin": _eval_is_admin,
    "isSuperAdmin": _eval_is_super_admin,
    "isBuiltInAdmin": _eval_is_built_in_admin,
    "isOwnAccount": _eval_is_own_account,
    "isOwnContent": _eval_is_own_content,
    "hasSameRole": _eval_has_same_role,
}


class Parser:
    def __init__(
        self,
        source: str,
        perm_keys: dict[str, str],
        ctx: dict[str, Any],
    ) -> None:
        self.tokenizer = Tokenizer(source)
        self.current = self.tokenizer.next()
        self.perm_keys = perm_keys
        self.ctx = ctx

    def _eat(self, token_type: TokenType) -> Token:
        token = self.current
        if token.type != token_type:
            raise SyntaxError(
                f"位置 {token.pos}: 期望 {token_type.name}，实际得到 {token.type.name} '{token.value}'"
            )
        self.current = self.tokenizer.next()
        return token

    def parse(self) -> bool:
        result = self._expr()
        if self.current.type != TokenType.EOF:
            raise SyntaxError(
                f"位置 {self.current.pos}: 多余的 token '{self.current.value}'"
            )
        return result

    def _expr(self) -> bool:
        left = self._and_expr()
        while self.current.type == TokenType.OR:
            self._eat(TokenType.OR)
            right = self._and_expr()
            left = left or right
        return left

    def _and_expr(self) -> bool:
        left = self._atom()
        while self.current.type == TokenType.AND:
            self._eat(TokenType.AND)
            right = self._atom()
            left = left and right
        return left

    def _atom(self) -> bool:
        if self.current.type == TokenType.NOT:
            self._eat(TokenType.NOT)
            return not self._atom()

        if self.current.type == TokenType.LPAREN:
            self._eat(TokenType.LPAREN)
            result = self._expr()
            self._eat(TokenType.RPAREN)
            return result

        if self.current.type == TokenType.PERM:
            key = self.current.value
            self._eat(TokenType.PERM)
            return key in self.perm_keys

        if self.current.type == TokenType.FUNC:
            return self._function_call()

        raise SyntaxError(
            f"位置 {self.current.pos}: 期望权限 key、函数调用、'!' 或 '('，实际得到 '{self.current.value}'"
        )

    def _function_call(self) -> bool:
        func_name = self.current.value
        self._eat(TokenType.FUNC)

        fn = BUILTIN_FUNCTIONS.get(func_name)
        if fn is None:
            raise ValueError(f"未知的内置函数: {func_name}")

        if self.current.type == TokenType.LPAREN:
            self._eat(TokenType.LPAREN)
            args: list[str] = []
            if self.current.type in (TokenType.IDENT, TokenType.PERM):
                args.append(self.current.value)
                self.current = self.tokenizer.next()
            self._eat(TokenType.RPAREN)
            return fn(args, self.ctx, self.perm_keys, self.ctx.get("all_role_ids", set()))

        return fn([], self.ctx, self.perm_keys, self.ctx.get("all_role_ids", set()))


def extract_perm_keys(expression: str) -> set[str]:
    """Extract all permission keys from an expression string.
    
    Only collects tokens that contain ':' (actual permission keys).
    Identifiers without ':' are treated as function arguments and excluded.
    """
    if not is_expression(expression):
        return {expression}

    tokenizer = Tokenizer(expression)
    keys: set[str] = set()
    while True:
        token = tokenizer.next()
        if token.type == TokenType.EOF:
            break
        if token.type == TokenType.PERM and ":" in token.value:
            keys.add(token.value)
    return keys


def evaluate_permission(
    expression: str,
    perm_keys: dict[str, str],
    context: dict[str, Any],
) -> bool:
    if not is_expression(expression):
        return expression in perm_keys

    parser = Parser(expression, perm_keys, context)
    return parser.parse()
