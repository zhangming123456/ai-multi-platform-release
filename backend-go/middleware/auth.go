package middleware

import (
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web/context"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
	"ai-multi-platform-release/backend-go/utils"
)

const (
	ContextUserKey  = "current_user"
	ContextPermsKey = "current_permissions"
)

func CurrentUser(ctx *context.Context) *models.User {
	v := ctx.Input.GetData(ContextUserKey)
	if v == nil {
		return nil
	}
	u, ok := v.(*models.User)
	if !ok {
		return nil
	}
	return u
}

func CurrentPermissions(ctx *context.Context) map[string]string {
	v := ctx.Input.GetData(ContextPermsKey)
	if v == nil {
		return map[string]string{}
	}
	m, ok := v.(map[string]string)
	if !ok {
		return map[string]string{}
	}
	return m
}

func ExtractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func AuthRequired(ctx *context.Context) {
	path := ctx.Input.URL()
	if isPublicPath(path) {
		return
	}

	token := ExtractBearerToken(ctx.Input.Header("Authorization"))
	if token == "" {
		ctx.Abort(http.StatusUnauthorized, `{"detail":"未提供认证凭证"}`)
		return
	}

	claims, err := utils.DecodeAccessToken(token)
	if err != nil || claims.UserID == "" {
		ctx.Abort(http.StatusUnauthorized, `{"detail":"无法验证凭据"}`)
		return
	}

	user := LoadUser(claims.UserID)
	if user == nil {
		ctx.Abort(http.StatusUnauthorized, `{"detail":"用户不存在"}`)
		return
	}
	ctx.Input.SetData(ContextUserKey, user)
	ctx.Input.SetData(ContextPermsKey, LoadPermissions(user.ID))
}

func isPublicPath(path string) bool {
	if path == "/" || path == "/healthz" {
		return true
	}
	if strings.HasPrefix(path, "/uploads/") {
		return true
	}
	if path == "/api/auth/login" || path == "/api/auth/register" {
		return true
	}
	if path == "/api/notifications/stream" {
		return true
	}
	return false
}

func LoadUser(userID string) *models.User {
	o := services.GetOrm()
	user := &models.User{ID: userID}
	if err := o.Read(user); err != nil {
		return nil
	}
	return user
}

func LoadPermissions(userID string) map[string]string {
	perms, err := services.GetUserEffectiveFlatPermissions(userID, nil)
	if err != nil {
		return map[string]string{}
	}
	return perms
}
