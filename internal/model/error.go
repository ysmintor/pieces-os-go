package model

import "fmt"

type ErrorCode string

const (
	// 客户端错误 (400-499)
	ErrInvalidRequest    ErrorCode = "invalid_request"     // 400 请求格式错误
	ErrUnauthorized      ErrorCode = "unauthorized"        // 401 未授权
	ErrForbidden         ErrorCode = "forbidden"           // 403 禁止访问
	ErrRouteNotFound     ErrorCode = "route_not_found"     // 404 路由不存在
	ErrMethodNotAllowed  ErrorCode = "method_not_allowed"  // 405 方法不允许
	ErrTooManyRequests   ErrorCode = "too_many_requests"   // 429 请求过多
	ErrRateLimitExceeded ErrorCode = "rate_limit_exceeded" // 429 超出速率限制

	// 服务端错误 (500-599)
	ErrInternalError      ErrorCode = "internal_error"      // 500 内部错误
	ErrServiceUnavailable ErrorCode = "service_unavailable" // 503 服务不可用
	ErrGatewayTimeout     ErrorCode = "gateway_timeout"     // 504 网关超时

	// 业务逻辑错误
	ErrModelNotFound  ErrorCode = "model_not_found"  // 模型不存在
	ErrInvalidModel   ErrorCode = "invalid_model"    // 无效的模型
	ErrModelOverload  ErrorCode = "model_overload"   // 模型过载
	ErrContextTooLong ErrorCode = "context_too_long" // 上下文过长
	ErrContentFilter  ErrorCode = "content_filter"   // 内容被过滤

	// 超时相关错误
	ErrRequestTimeout    ErrorCode = "request_timeout"    // 请求超时
	ErrStreamTimeout     ErrorCode = "stream_timeout"     // 流式请求超时
	ErrConnectionTimeout ErrorCode = "connection_timeout" // 连接超时

	// 限流相关错误
	ErrIPBlocked       ErrorCode = "ip_blocked"       // IP被封禁
	ErrQuotaExceeded   ErrorCode = "quota_exceeded"   // 配额超限
	ErrConcurrentLimit ErrorCode = "concurrent_limit" // 并发限制

	// 验证相关错误
	ErrInvalidToken     ErrorCode = "invalid_token"     // 无效的令牌
	ErrTokenExpired     ErrorCode = "token_expired"     // 令牌过期
	ErrInvalidSignature ErrorCode = "invalid_signature" // 无效的签名

	// 数据相关错误
	ErrDataNotFound  ErrorCode = "data_not_found" // 数据不存在
	ErrDataConflict  ErrorCode = "data_conflict"  // 数据冲突
	ErrDataCorrupted ErrorCode = "data_corrupted" // 数据损坏

	// 系统相关错误
	ErrSystemOverload    ErrorCode = "system_overload"    // 系统过载
	ErrMaintenanceMode   ErrorCode = "maintenance_mode"   // 维护模式
	ErrResourceExhausted ErrorCode = "resource_exhausted" // 资源耗尽
)

// HTTP状态码映射
var ErrorStatusMap = map[ErrorCode]int{
	ErrInvalidRequest:     400,
	ErrUnauthorized:       401,
	ErrForbidden:          403,
	ErrRouteNotFound:      404,
	ErrMethodNotAllowed:   405,
	ErrTooManyRequests:    429,
	ErrRateLimitExceeded:  429,
	ErrInternalError:      500,
	ErrServiceUnavailable: 503,
	ErrGatewayTimeout:     504,
	ErrRequestTimeout:     504,
	ErrStreamTimeout:      504,
}

type APIError struct {
	Code    ErrorCode `json:"code"`              // 错误代码
	Message string    `json:"message"`           // 错误消息
	Status  int       `json:"-"`                 // HTTP 状态码
	Details any       `json:"details,omitempty"` // 详细错误信息(可选)
}

func (e *APIError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s: %s (details: %v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// 创建新的API错误
func NewAPIError(code ErrorCode, message string, status int) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// 创建带详细信息的API错误
func NewAPIErrorWithDetails(code ErrorCode, message string, status int, details any) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Status:  status,
		Details: details,
	}
}

// 获取错误的HTTP状态码
func GetErrorStatus(code ErrorCode) int {
	if status, ok := ErrorStatusMap[code]; ok {
		return status
	}
	return 500 // 默认返回500
}
