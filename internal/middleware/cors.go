package middleware

import (
	"net/http"
	"pieces-os-go/internal/model"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 允许的来源
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 允许的请求方法
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		// 允许的请求头
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		// 允许凭证
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// 预检请求缓存时间
		w.Header().Set("Access-Control-Max-Age", "3600")

		// 处理 OPTIONS 预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 检查请求方法是否允许
		if r.Method != "GET" && r.Method != "POST" && r.Method != "OPTIONS" {
			writeError(w, model.NewAPIError(model.ErrMethodNotAllowed, "Method not allowed", http.StatusMethodNotAllowed))
			return
		}

		next.ServeHTTP(w, r)
	})
}
