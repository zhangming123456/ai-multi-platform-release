package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
)

func MaskAPIKey(value string) string {
	if len(value) <= 7 {
		return "****"
	}
	return value[:3] + "****" + value[len(value)-4:]
}

func MaskEmail(value string) string {
	if !strings.Contains(value, "@") {
		return "****"
	}
	parts := strings.SplitN(value, "@", 2)
	username, domain := parts[0], parts[1]
	if len(username) <= 1 {
		return "*@" + domain
	}
	return username[:1] + "***@" + domain
}

var sensitiveFieldHandlers = map[string]func(string) string{
	"api_key":         MaskAPIKey,
	"access_token":    func(v string) string { return "****" },
	"cookie_data":     func(v string) string { return "****" },
	"hashed_password": func(v string) string { return "****" },
	"password":        func(v string) string { return "****" },
	"email":           MaskEmail,
	"phone": func(v string) string {
		if len(v) >= 7 {
			return v[:3] + "****" + v[len(v)-4:]
		}
		return "****"
	},
}

var fullyExcludedPrefixes = []string{
	"/api/db/",
	"/docs",
	"/openapi.json",
	"/redoc",
}

var authPathPrefixes = []string{
	"/api/auth/",
}

func shouldExcludePath(path string) bool {
	for _, p := range fullyExcludedPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func isAuthPath(path string) bool {
	for _, p := range authPathPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func MaskValue(key string, value interface{}, path string) interface{} {
	if key == "access_token" && isAuthPath(path) {
		return value
	}
	if handler, ok := sensitiveFieldHandlers[key]; ok {
		if s, ok := value.(string); ok && s != "" {
			return handler(s)
		}
	}
	switch v := value.(type) {
	case map[string]interface{}:
		return MaskDict(v, path)
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, item := range v {
			result = append(result, MaskElement(item, path))
		}
		return result
	}
	return value
}

func MaskDict(data map[string]interface{}, path string) map[string]interface{} {
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		result[k] = MaskValue(k, v, path)
	}
	return result
}

func MaskElement(item interface{}, path string) interface{} {
	switch v := item.(type) {
	case map[string]interface{}:
		return MaskDict(v, path)
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, sub := range v {
			result = append(result, MaskElement(sub, path))
		}
		return result
	}
	return item
}

func MaskResponseData(data interface{}, path string) interface{} {
	if shouldExcludePath(path) {
		return data
	}
	switch v := data.(type) {
	case map[string]interface{}:
		return MaskDict(v, path)
	case []interface{}:
		return MaskElement(v, path)
	}
	return data
}

// PrivacyMaskFilter wraps the response body to mask sensitive fields.
func PrivacyMaskFilter(ctx *context.Context) {
	path := ctx.Input.URL()
	if shouldExcludePath(path) {
		ctx.Input.SetData("privacy_checked", true)
		return
	}
	if strings.Contains(ctx.Input.Header("Content-Type"), "application/json") || ctx.Input.Header("Accept") != "" {
		ctx.Input.SetData("privacy_checked", true)
	}
}

func WriteMaskedJSON(w http.ResponseWriter, statusCode int, payload interface{}, path string) {
	data := MaskResponseData(payload, path)
	body, err := json.Marshal(data)
	if err != nil {
		utils_WriteError(w, http.StatusInternalServerError, "序列化失败")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(body)
}

func utils_WriteError(w http.ResponseWriter, statusCode int, detail string) {
	body, _ := json.Marshal(map[string]string{"detail": detail})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(body)
}
