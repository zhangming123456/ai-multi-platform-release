package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web/context"

	"ai-multi-platform-release/backend-go/services"
)

func inferModeFromKey(permissionKey string) string {
	if services.IsExpression(permissionKey) {
		return "read"
	}
	parts := strings.Split(permissionKey, ":")
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		if last == "read" || last == "write" {
			return last
		}
	}
	return "read"
}

func describeMode(mode string) string {
	if mode == "write" {
		return "写入"
	}
	return "查看"
}

// RequirePermission checks the current user against the given permission key or
// expression. Returns true if authorized, false if a response has already been
// written (401/403/500).
func RequirePermission(ctx *context.Context, permissionKey string) bool {
	user := CurrentUser(ctx)
	if user == nil {
		ctx.Abort(http.StatusUnauthorized, `{"detail":"无法验证凭据"}`)
		return false
	}
	perms := CurrentPermissions(ctx)

	if services.IsExpression(permissionKey) {
		evalCtx := &services.EvalContext{
			CurrentUser: user,
			Data:        map[string]interface{}{},
		}
		hasAccess, err := services.EvaluatePermission(permissionKey, perms, evalCtx)
		if err != nil {
			ctx.Abort(http.StatusInternalServerError, fmt.Sprintf(`{"detail":"权限表达式错误: %s"}`, err))
			return false
		}
		if hasAccess {
			return true
		}
		ctx.Abort(http.StatusForbidden, `{"detail":"无访问权限"}`)
		return false
	}

	mode := inferModeFromKey(permissionKey)
	if perms[permissionKey] != "" {
		return true
	}
	ctx.Abort(http.StatusForbidden, fmt.Sprintf(`{"detail":"无%s权限"}`, describeMode(mode)))
	return false
}

// RequireWritePermission is an alias used by write handlers; the key itself
// carries the write mode.
func RequireWritePermission(ctx *context.Context, permissionKey string) bool {
	return RequirePermission(ctx, permissionKey)
}
