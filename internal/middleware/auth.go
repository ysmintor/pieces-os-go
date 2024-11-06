package middleware

import (
	"encoding/json"
	"net/http"
	"pieces-os-go/internal/model"
	"strings"
)

func Auth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeError(w, model.NewAPIError(model.ErrUnauthorized, "Missing authentication information", http.StatusUnauthorized))
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				writeError(w, model.NewAPIError(model.ErrUnauthorized, "Invalid authentication format", http.StatusUnauthorized))
				return
			}

			token := auth[len(prefix):]
			if token != apiKey {
				writeError(w, model.NewAPIError(model.ErrUnauthorized, "Invalid API key", http.StatusUnauthorized))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AdminAuth 创建管理接口认证中间件
func AdminAuth(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeError(w, model.NewAPIError(model.ErrUnauthorized, "Unauthorized", http.StatusUnauthorized))
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			if token != adminKey || adminKey == "" {
				writeError(w, model.NewAPIError(model.ErrForbidden, "Forbidden", http.StatusForbidden))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// 添加辅助函数
func writeError(w http.ResponseWriter, err *model.APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": err.Message,
			"type":    "error",
			"code":    err.Code,
		},
	})
}
