package services

import (
	"strings"

	"ai-multi-platform-release/backend-go/models"
)

func IsExpression(source string) bool {
	return strings.ContainsAny(source, "|&()!")
}

func ExtractPermKeys(expression string) []string {
	if !IsExpression(expression) {
		return []string{expression}
	}
	var keys []string
	seen := map[string]bool{}
	for _, token := range tokenize(expression) {
		if strings.Contains(token, ":") && !seen[token] {
			seen[token] = true
			keys = append(keys, token)
		}
	}
	return keys
}

func tokenize(source string) []string {
	var tokens []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for i := 0; i < len(source); i++ {
		ch := source[i]
		switch {
		case ch == ' ':
			flush()
		case ch == '(' || ch == ')' || ch == '!' || ch == '&':
			flush()
			if ch == '&' && i+1 < len(source) && source[i+1] == '&' {
				tokens = append(tokens, "&&")
				i++
			} else {
				tokens = append(tokens, string(ch))
			}
		case ch == '|':
			flush()
			if i+1 < len(source) && source[i+1] == '|' {
				tokens = append(tokens, "||")
				i++
			} else {
				tokens = append(tokens, string(ch))
			}
		default:
			current.WriteByte(ch)
		}
	}
	flush()
	return tokens
}

type EvalContext struct {
	CurrentUser *models.User
	Data        map[string]interface{}
}

func (c *EvalContext) get(key string) interface{} {
	if c == nil || c.Data == nil {
		return nil
	}
	return c.Data[key]
}

type permParser struct {
	tokens     []string
	pos        int
	permKeys   map[string]string
	ctx        *EvalContext
	allRoleIDs map[string]bool
}

func (p *permParser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

func (p *permParser) next() string {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *permParser) parseExpr() (bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		return false, err
	}
	for p.peek() == "||" {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return false, err
		}
		left = left || right
	}
	return left, nil
}

func (p *permParser) parseAnd() (bool, error) {
	left, err := p.parseAtom()
	if err != nil {
		return false, err
	}
	for p.peek() == "&" || p.peek() == "&&" {
		p.next()
		right, err := p.parseAtom()
		if err != nil {
			return false, err
		}
		left = left && right
	}
	return left, nil
}

func (p *permParser) parseAtom() (bool, error) {
	tok := p.peek()
	if tok == "" {
		return false, errSyntax
	}
	if tok == "!" {
		p.next()
		val, err := p.parseAtom()
		if err != nil {
			return false, err
		}
		return !val, nil
	}
	if tok == "(" {
		p.next()
		val, err := p.parseExpr()
		if err != nil {
			return false, err
		}
		if p.peek() != ")" {
			return false, errSyntax
		}
		p.next()
		return val, nil
	}
	if tok == ")" || tok == "||" || tok == "&" || tok == "&&" {
		return false, errSyntax
	}
	// function call?
	if strings.HasSuffix(tok, "(") || (tok == "isSelf" || tok == "isAdmin" || tok == "isSuperAdmin" || tok == "isBuiltInAdmin" || tok == "isOwnAccount" || tok == "isOwnContent" || tok == "hasSameRole") {
		return p.parseFuncCall()
	}
	// permission key or bare identifier
	p.next()
	_, ok := p.permKeys[tok]
	return ok, nil
}

func (p *permParser) parseFuncCall() (bool, error) {
	name := p.next()
	clean := strings.TrimSuffix(name, "(")
	arg := ""
	if strings.HasSuffix(name, "(") {
		next := p.peek()
		if next != "" && next != ")" && next != "(" && !isOperator(next) {
			arg = p.next()
		}
		if p.peek() == ")" {
			p.next()
		}
	}
	return evalBuiltin(clean, arg, p.ctx, p.allRoleIDs), nil
}

func isOperator(tok string) bool {
	switch tok {
	case ")", "(", "||", "&", "&&", "!":
		return true
	}
	return false
}

func evalBuiltin(name, arg string, ctx *EvalContext, allRoleIDs map[string]bool) bool {
	if ctx == nil || ctx.CurrentUser == nil {
		return false
	}
	user := ctx.CurrentUser
	switch name {
	case "isSelf":
		targetID := arg
		if targetID == "" {
			if v, ok := ctx.Data["target_user_id"]; ok {
				targetID, _ = v.(string)
			}
		}
		return targetID != "" && user.ID == targetID
	case "isAdmin":
		return user.Role == "admin" || user.Role == "manager"
	case "isSuperAdmin":
		return user.Role == "admin" && user.ID == "1"
	case "isBuiltInAdmin":
		targetID := arg
		if targetID == "" {
			if v, ok := ctx.Data["target_user_id"]; ok {
				targetID, _ = v.(string)
			}
		}
		return targetID == "1"
	case "isOwnAccount":
		if v, ok := ctx.Data["account"].(*models.Account); ok && v != nil {
			return v.UserID == user.ID
		}
		return false
	case "isOwnContent":
		if v, ok := ctx.Data["content"].(*models.Content); ok && v != nil {
			return v.UserID == user.ID
		}
		return false
	case "hasSameRole":
		if v, ok := ctx.Data["target_user_id"]; ok {
			targetID, _ := v.(string)
			if targetID == user.ID {
				return true
			}
		}
		if v, ok := ctx.Data["target_role_ids"].([]string); ok {
			targetRoles := map[string]bool{}
			for _, id := range v {
				targetRoles[id] = true
			}
			if len(targetRoles) == 0 {
				for id := range allRoleIDs {
					targetRoles[id] = true
				}
			}
			currentRoles := map[string]bool{}
			if v, ok := ctx.Data["current_role_ids"].([]string); ok {
				for _, id := range v {
					currentRoles[id] = true
				}
			}
			for id := range currentRoles {
				if targetRoles[id] {
					return true
				}
			}
		}
		return false
	}
	return false
}

var errSyntax = errSyntaxType{}

type errSyntaxType struct{}

func (errSyntaxType) Error() string { return "权限表达式语法错误" }

func EvaluatePermission(expression string, permKeys map[string]string, context *EvalContext) (bool, error) {
	if !IsExpression(expression) {
		_, ok := permKeys[expression]
		return ok, nil
	}
	tokens := tokenize(expression)
	p := &permParser{
		tokens:     tokens,
		permKeys:   permKeys,
		ctx:        context,
		allRoleIDs: map[string]bool{},
	}
	if context != nil {
		if v, ok := context.Data["all_role_ids"].([]string); ok {
			for _, id := range v {
				p.allRoleIDs[id] = true
			}
		}
	}
	return p.parseExpr()
}
