package middleware

import (
	"net/http"
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
				http.Error(w, "未提供认证信息", http.StatusUnauthorized)
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				http.Error(w, "认证格式错误", http.StatusUnauthorized)
				return
			}

			token := auth[len(prefix):]
			if token != apiKey {
				http.Error(w, "无效的API密钥", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
