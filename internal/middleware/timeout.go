package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"pieces-os-go/internal/config"
	"pieces-os-go/internal/model"
	"sync/atomic"
	"time"
)

// 添加超时状态监控
type TimeoutStats struct {
	normalTimeouts int64
	streamTimeouts int64
}

var stats TimeoutStats

// 添加获取统计信息的方法
func GetTimeoutStats() TimeoutStats {
	return TimeoutStats{
		normalTimeouts: atomic.LoadInt64(&stats.normalTimeouts),
		streamTimeouts: atomic.LoadInt64(&stats.streamTimeouts),
	}
}

func TimeoutMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var timeout time.Duration

			// 根据请求类型选择超时时间
			if r.Header.Get("Accept") == "text/event-stream" {
				timeout = cfg.StreamTimeout
			} else {
				timeout = cfg.RequestTimeout
			}

			// 如果超时时间为0，表示不设置超时
			if timeout == 0 {
				next.ServeHTTP(w, r)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			done := make(chan bool)
			go func() {
				next.ServeHTTP(w, r)
				done <- true
			}()

			select {
			case <-ctx.Done():
				w.WriteHeader(http.StatusGatewayTimeout)
				var errCode model.ErrorCode
				if r.Header.Get("Accept") == "text/event-stream" {
					errCode = model.ErrStreamTimeout
					atomic.AddInt64(&stats.streamTimeouts, 1)
				} else {
					errCode = model.ErrRequestTimeout
					atomic.AddInt64(&stats.normalTimeouts, 1)
				}

				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":  errCode,
					"status": http.StatusGatewayTimeout,
				})
				return
			case <-done:
				return
			}
		})
	}
}
