package domain

import "errors"

var (
	// Authentication errors
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrInvalidEmail          = errors.New("invalid email format")
	ErrInvalidUsername       = errors.New("invalid username format")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrWeakPassword          = errors.New("password is too weak")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserInactive          = errors.New("user account is inactive")
	ErrAccountLocked         = errors.New("user account is blocked due to excessive failed attempts")
	ErrPasswordExpired       = errors.New("password has expired and must be changed")

	// Token errors
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token has expired")
	ErrTokenRevoked        = errors.New("token has been revoked")
	ErrTokenNotFound       = errors.New("token not found")
	ErrRefreshTokenInvalid = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token has expired")
	ErrRefreshTokenRevoked = errors.New("refresh token has been revoked")
	ErrRefreshTokenRotated = errors.New("refresh token has been rotated")
	ErrTokenStolen         = errors.New("refresh token theft detected")

	// Session errors
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session has expired")
	ErrSessionRevoked   = errors.New("session has been revoked")
	ErrInvalidSessionID = errors.New("invalid session ID")

	// 2FA errors
	Err2FANotEnabled     = errors.New("2FA is not enabled")
	Err2FAAlreadyEnabled = errors.New("2FA is already enabled")
	ErrInvalid2FACode    = errors.New("invalid 2FA code")
	Err2FARequired       = errors.New("2FA verification required")

	// Rate limiting errors
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrIPBlocked            = errors.New("IP address is blocked")
	ErrTooManyLoginAttempts = errors.New("too many login attempts")
	ErrTooManyRegistrations = errors.New("too many registration attempts")

	// Password reset errors
	ErrInvalidResetToken = errors.New("invalid password reset token")
	ErrResetTokenExpired = errors.New("password reset token has expired")
	ErrResetTokenUsed    = errors.New("password reset token already used")

	// OAuth errors
	ErrOAuthProviderNotFound = errors.New("OAuth provider not found")
	ErrOAuthStateMismatch    = errors.New("OAuth state mismatch")
	ErrOAuthCodeInvalid      = errors.New("invalid OAuth code")
	ErrOAuthUserInfo         = errors.New("failed to get OAuth user info")

	// General errors
	ErrInternal     = errors.New("internal server error")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
