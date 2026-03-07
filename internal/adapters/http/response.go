package http

import (
	"encoding/json"
	"net/http"
	"time"
)

// ErrorCode es el código de error estándar HTTP
type ErrorCode string

const (
	// Client errors (4xx)
	ErrorBadRequest      ErrorCode = "BAD_REQUEST"
	ErrorUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrorForbidden       ErrorCode = "FORBIDDEN"
	ErrorNotFound        ErrorCode = "NOT_FOUND"
	ErrorConflict        ErrorCode = "CONFLICT"
	ErrorValidationError ErrorCode = "VALIDATION_ERROR"

	// Auth-specific errors
	ErrorInvalidCredentials    ErrorCode = "INVALID_CREDENTIALS"
	ErrorAccountLocked         ErrorCode = "ACCOUNT_LOCKED"
	ErrorPasswordExpired       ErrorCode = "PASSWORD_EXPIRED"
	ErrorPasswordResetRequired ErrorCode = "PASSWORD_RESET_REQUIRED"
	ErrorEmailExists           ErrorCode = "EMAIL_EXISTS"
	ErrorUsernameExists        ErrorCode = "USERNAME_EXISTS"
	ErrorInvalidToken          ErrorCode = "INVALID_TOKEN"
	ErrorInvalid2FACode        ErrorCode = "INVALID_2FA_CODE"
	Error2FAAlreadyEnabled     ErrorCode = "2FA_ALREADY_ENABLED"
	Error2FANotEnabled         ErrorCode = "2FA_NOT_ENABLED"
	ErrorSessionNotFound       ErrorCode = "SESSION_NOT_FOUND"
	ErrorOAuthDisabled         ErrorCode = "OAUTH_DISABLED"

	// Server errors (5xx)
	ErrorInternalServer ErrorCode = "INTERNAL_SERVER_ERROR"
)

// StandardErrorResponse es la estructura estándar de respuesta de error
type StandardErrorResponse struct {
	Status    int       `json:"status"`
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// WriteError escribe una respuesta de error formateada de manera estándar
func WriteError(w http.ResponseWriter, statusCode int, code ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	errorResp := &StandardErrorResponse{
		Status:    statusCode,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(errorResp)
}

// WriteBadRequest escribe error de request inválido
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, ErrorBadRequest, message)
}

// WriteUnauthorized escribe error de no autorizado
func WriteUnauthorized(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnauthorized, ErrorUnauthorized, message)
}

// WriteForbidden escribe error de acceso prohibido
func WriteForbidden(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, ErrorForbidden, message)
}

// WriteNotFound escribe error 404
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, ErrorNotFound, message)
}

// WriteConflict escribe error de conflicto
func WriteConflict(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, ErrorConflict, message)
}

// WriteInternalError escribe error interno del servidor
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, ErrorInternalServer, message)
}

// Auth-specific error writers

// WriteInvalidCredentials escribe error de credenciales inválidas
func WriteInvalidCredentials(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnauthorized, ErrorInvalidCredentials, message)
}

// WriteAccountLocked escribe error de cuenta bloqueada
func WriteAccountLocked(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, ErrorAccountLocked, message)
}

// WritePasswordExpired escribe error de contraseña expirada
func WritePasswordExpired(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, ErrorPasswordExpired, message)
}

// WritePasswordResetRequired escribe error de reset de contraseña requerido
func WritePasswordResetRequired(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, ErrorPasswordResetRequired, message)
}

// WriteEmailExists escribe error de email duplicado
func WriteEmailExists(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, ErrorEmailExists, message)
}

// WriteInvalidToken escribe error de token inválido
func WriteInvalidToken(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnauthorized, ErrorInvalidToken, message)
}

// WriteValidationError escribe error de validación
func WriteValidationError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, ErrorValidationError, message)
}

// WriteUsernameExists escribe error de username duplicado
func WriteUsernameExists(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, ErrorUsernameExists, message)
}

// WriteInvalid2FACode escribe error de código 2FA inválido
func WriteInvalid2FACode(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, ErrorInvalid2FACode, message)
}

// Write2FAAlreadyEnabled escribe error de 2FA ya habilitado
func Write2FAAlreadyEnabled(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, Error2FAAlreadyEnabled, message)
}

// Write2FANotEnabled escribe error de 2FA no habilitado
func Write2FANotEnabled(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, Error2FANotEnabled, message)
}

// WriteSessionNotFound escribe error de sesión no encontrada
func WriteSessionNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, ErrorSessionNotFound, message)
}

// WriteOAuthDisabled escribe error de OAuth deshabilitado
func WriteOAuthDisabled(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusServiceUnavailable, ErrorOAuthDisabled, message)
}
