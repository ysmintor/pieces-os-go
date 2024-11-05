package handler

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

var (
	// 记录所有请求的计数器
	totalRequests uint64
	totalPerSec   uint64
	apiRequests   uint64
	apiPerSec     uint64
)

// ResetMinuteCounters 重置每分钟计数器
func ResetMinuteCounters() {
	atomic.StoreUint64(&totalRequests, 0)
	atomic.StoreUint64(&apiRequests, 0)
}

// ResetSecondCounters 重置每秒计数器
func ResetSecondCounters() {
	atomic.StoreUint64(&totalPerSec, 0)
	atomic.StoreUint64(&apiPerSec, 0)
}

// IncrementCounter 增加计数器
func IncrementCounter(isAPIRequest bool) {
	// 增加每分钟计数
	atomic.AddUint64(&totalRequests, 1)
	atomic.AddUint64(&totalPerSec, 1)

	if isAPIRequest {
		atomic.AddUint64(&apiRequests, 1)
		atomic.AddUint64(&apiPerSec, 1)
	}
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	// 获取当前计数
	totalRPM := atomic.LoadUint64(&totalRequests)
	totalRPS := atomic.LoadUint64(&totalPerSec)
	apiRPM := atomic.LoadUint64(&apiRequests)
	apiRPS := atomic.LoadUint64(&apiPerSec)

	response := map[string]interface{}{
		"status": "ok",
		"metrics": map[string]interface{}{
			"total_rpm": totalRPM,
			"total_rps": totalRPS,
			"api_rpm":   apiRPM,
			"api_rps":   apiRPS,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}
