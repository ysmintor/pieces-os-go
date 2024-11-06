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
			isStreamRequest := r.Header.Get("Accept") == "text/event-stream"

			if isStreamRequest {
				timeout = cfg.StreamTimeout
			} else {
				timeout = cfg.RequestTimeout
			}

			if timeout == 0 {
				next.ServeHTTP(w, r)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// 使用自定义的 ResponseWriter 来捕获状态码
			rw := newTimeoutResponseWriter(w)

			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(rw, r.WithContext(ctx))
			}()

			select {
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					// 只有在还没有写入响应时才写入超时错误
					if !rw.written {
						if isStreamRequest {
							atomic.AddInt64(&stats.streamTimeouts, 1)
							writeTimeoutError(w, model.ErrStreamTimeout, "Stream timeout")
						} else {
							atomic.AddInt64(&stats.normalTimeouts, 1)
							writeTimeoutError(w, model.ErrRequestTimeout, "Request timeout")
						}
					}
				}
			case <-done:
				// 请求正常完成
			}
		})
	}
}

type timeoutResponseWriter struct {
	http.ResponseWriter
	written bool
	status  int
}

func newTimeoutResponseWriter(w http.ResponseWriter) *timeoutResponseWriter {
	return &timeoutResponseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (w *timeoutResponseWriter) WriteHeader(status int) {
	w.written = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *timeoutResponseWriter) Write(b []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(b)
}

func writeTimeoutError(w http.ResponseWriter, code model.ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusGatewayTimeout)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "timeout_error",
			"code":    code,
		},
	})
}
