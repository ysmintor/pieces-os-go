package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"pieces-os-go/internal/config"
	"pieces-os-go/internal/handler"
)

var (
	logWriter io.Writer = os.Stdout
	bufWriter *bufio.Writer
	logMutex  sync.Mutex
)

// InitLogger 初始化日志输出
func InitLogger(logFile string) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile == "" {
		logWriter = os.Stdout
		bufWriter = bufio.NewWriter(logWriter)
		return nil
	}

	// 以截断模式打开文件(O_TRUNC),这样会清除原有内容
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	logWriter = io.MultiWriter(os.Stdout, file)
	bufWriter = bufio.NewWriter(logWriter)
	log.SetOutput(bufWriter)
	return nil
}

func Logger(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := wrapResponseWriter(w)

			// 获取真实IP
			realIP := GetRealIP(r)

			// 增加请求计数
			handler.IncrementCounter(strings.HasPrefix(r.URL.Path, cfg.APIPrefix))

			// 获取请求体中的模型信息
			var modelInfo string
			if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/completions") {
				var requestBody struct {
					Model string `json:"model"`
				}
				if body, err := io.ReadAll(r.Body); err == nil {
					r.Body = io.NopCloser(bytes.NewBuffer(body)) // 重新设置 body 以供后续读取
					if err := json.Unmarshal(body, &requestBody); err == nil {
						modelInfo = fmt.Sprintf(" model=%s", requestBody.Model)
					}
				}
			}

			next.ServeHTTP(wrapped, r)

			logMutex.Lock()
			logger := log.New(bufWriter, "", log.LstdFlags)
			logger.Printf(
				"%s %s%s IP=%s Status=%d Duration=%s",
				r.Method,
				r.RequestURI,
				modelInfo,
				realIP,
				wrapped.status,
				time.Since(start),
			)
			bufWriter.Flush()
			logMutex.Unlock()
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool // 添加此字段来追踪是否已经写入头部
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK, false}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader { // 只在第一次调用时写入
		rw.status = code
		rw.ResponseWriter.WriteHeader(code)
		rw.wroteHeader = true
	}
}

// 添加 Flush 方法实现 http.Flusher 接口
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// 添加 Push 方法实现 http.Pusher 接口 (可选，用于 HTTP/2)
// func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
// 	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
// 		return p.Push(target, opts)
// 	}
// 	return http.ErrNotSupported
// }

// 添加 Hijack 方法实现 http.Hijacker 接口 (可选，用于 WebSocket)
// func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
// 	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
// 		return h.Hijack()
// 	}
// 	return nil, nil, http.ErrNotSupported
// }
