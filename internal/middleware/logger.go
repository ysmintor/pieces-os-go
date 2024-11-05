package middleware

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := wrapResponseWriter(w)

		// 增加请求计数
		handler.IncrementCounter(strings.HasPrefix(r.URL.Path, "/v1"))

		next.ServeHTTP(wrapped, r)

		logMutex.Lock()
		logger := log.New(bufWriter, "", log.LstdFlags)
		logger.Printf(
			"%s %s %s %d %s",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			wrapped.status,
			time.Since(start),
		)
		bufWriter.Flush()
		logMutex.Unlock()
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// 添加 Flush 方法实现 http.Flusher 接口
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// 添加 Push 方法实现 http.Pusher 接口 (可选，用于 HTTP/2)
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// 添加 Hijack 方法实现 http.Hijacker 接口 (可选，用于 WebSocket)
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
