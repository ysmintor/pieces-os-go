package middleware

import (
	"net/http"
)

// WhitelistOnly 创建只允许白名单IP访问的中间件
func WhitelistOnly(whitelist []string) func(http.Handler) http.Handler {
	whitelistMap := make(map[string]bool)
	for _, ip := range whitelist {
		whitelistMap[ip] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetRealIP(r)
			if !whitelistMap[ip] {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
