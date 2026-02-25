package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func NewAppError(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Common errors
var (
	ErrBadRequest          = NewAppError("BAD_REQUEST", "Bad request", http.StatusBadRequest)
	ErrUnauthorized        = NewAppError("UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized)
	ErrForbidden           = NewAppError("FORBIDDEN", "Forbidden", http.StatusForbidden)
	ErrNotFound            = NewAppError("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrInternalServer      = NewAppError("INTERNAL_SERVER_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrRateLimitExceeded   = NewAppError("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests)
)
