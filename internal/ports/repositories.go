package ports

import (
	"context"

	"github.com/vertercloud/auth-service/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User, passwordHash string) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByEmailOrUsername(ctx context.Context, identifier string) (*domain.User, error)
	GetByOAuthProvider(ctx context.Context, provider, providerID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID, newPasswordHash string) error
	GetExpiringPasswords(ctx context.Context, thresholdDays int) ([]*domain.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.Session, error)
	GetRecentByUserID(ctx context.Context, userID string, limit int) ([]*domain.Session, error)
	Update(ctx context.Context, session *domain.Session) error
	Revoke(ctx context.Context, sessionID string, revokedBy, reason string) error
	RevokeAllByUserID(ctx context.Context, userID, revokedBy, reason string) error
	DeleteExpired(ctx context.Context) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByID(ctx context.Context, id string) (*domain.RefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	GetBySessionID(ctx context.Context, sessionID string) (*domain.RefreshToken, error)
	Update(ctx context.Context, token *domain.RefreshToken) error
	Revoke(ctx context.Context, tokenID string) error
	RevokeByUserID(ctx context.Context, userID string) error
	RevokeBySessionID(ctx context.Context, sessionID string) error
	DeleteExpired(ctx context.Context) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	GetByCode(ctx context.Context, userID, code string) (*domain.PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, tokenID string) error
	DeleteExpired(ctx context.Context) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, entry *domain.AuditLogEntry) error
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.AuditLogEntry, error)
}

type BlockedIPRepository interface {
	Block(ctx context.Context, ip string, reason string, duration int64) error
	IsBlocked(ctx context.Context, ip string) (bool, error)
	Unblock(ctx context.Context, ip string) error
}

type EmailVerificationRepository interface {
	Create(ctx context.Context, verification *domain.EmailVerification) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerification, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.EmailVerification, error)
	MarkAsVerified(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) (int64, error)
}
