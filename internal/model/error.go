package model

import "fmt"

type ErrorCode string

const (
	ErrInvalidRequest     ErrorCode = "invalid_request"
	ErrUnauthorized       ErrorCode = "unauthorized"
	ErrRateLimitExceeded  ErrorCode = "rate_limit_exceeded"
	ErrInternalError      ErrorCode = "internal_error"
	ErrServiceUnavailable ErrorCode = "service_unavailable"
)

type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Status  int       `json:"-"` // HTTP 状态码
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewAPIError(code ErrorCode, message string, status int) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}
