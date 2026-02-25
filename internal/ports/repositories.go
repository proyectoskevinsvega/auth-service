package ports

import (
	"context"
	"time"

	"github.com/vertercloud/auth-service/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User, passwordHash string) error
	GetByID(ctx context.Context, tenantID, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, tenantID, username string) (*domain.User, error)
	GetByEmailOrUsername(ctx context.Context, tenantID, identifier string) (*domain.User, error)
	GetByOAuthProvider(ctx context.Context, tenantID, provider, providerID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, tenantID, userID, newPasswordHash string) error
	GetExpiringPasswords(ctx context.Context, thresholdDays int) ([]*domain.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, tenantID, id string) (*domain.Session, error)
	GetByUserID(ctx context.Context, tenantID, userID string) ([]*domain.Session, error)
	GetRecentByUserID(ctx context.Context, tenantID, userID string, limit int) ([]*domain.Session, error)
	Update(ctx context.Context, session *domain.Session) error
	Revoke(ctx context.Context, tenantID, sessionID string, revokedBy, reason string) error
	RevokeAllByUserID(ctx context.Context, tenantID, userID, revokedBy, reason string) error
	DeleteExpired(ctx context.Context) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByID(ctx context.Context, tenantID, id string) (*domain.RefreshToken, error)
	GetByTokenHash(ctx context.Context, tenantID, tokenHash string) (*domain.RefreshToken, error)
	GetBySessionID(ctx context.Context, tenantID, sessionID string) (*domain.RefreshToken, error)
	Update(ctx context.Context, token *domain.RefreshToken) error
	Revoke(ctx context.Context, tenantID, tokenID string) error
	RevokeByUserID(ctx context.Context, tenantID, userID string) error
	RevokeBySessionID(ctx context.Context, tenantID, sessionID string) error
	DeleteExpired(ctx context.Context) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByToken(ctx context.Context, tenantID, token string) (*domain.PasswordResetToken, error)
	GetByCode(ctx context.Context, tenantID, userID, code string) (*domain.PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, tenantID, tokenID string) error
	DeleteExpired(ctx context.Context) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, entry *domain.AuditLogEntry) error
	GetByUserID(ctx context.Context, tenantID, userID string, limit, offset int) ([]*domain.AuditLogEntry, error)
	Search(ctx context.Context, filter domain.AuditSearchFilter) ([]*domain.AuditLogEntry, int, error)
}

type BlockedIPRepository interface {
	Block(ctx context.Context, ip string, reason string, duration int64) error
	IsBlocked(ctx context.Context, ip string) (bool, error)
	Unblock(ctx context.Context, ip string) error
}

type EmailVerificationRepository interface {
	Create(ctx context.Context, verification *domain.EmailVerification) error
	GetByTokenHash(ctx context.Context, tenantID, tokenHash string) (*domain.EmailVerification, error)
	GetByUserID(ctx context.Context, tenantID, userID string) ([]*domain.EmailVerification, error)
	MarkAsVerified(ctx context.Context, tenantID, tokenHash string) error
	DeleteByUserID(ctx context.Context, tenantID, userID string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type RoleRepository interface {
	CreateRole(ctx context.Context, role *domain.Role) error
	GetRoleByName(ctx context.Context, tenantID, name string) (*domain.Role, error)
	ListRoles(ctx context.Context, tenantID string) ([]*domain.Role, error)
	AddPermissionToRole(ctx context.Context, tenantID, roleID, permissionID string) error
	AssignRoleToUser(ctx context.Context, tenantID, userID, roleID string) error
	RemoveRoleFromUser(ctx context.Context, tenantID, userID, roleID string) error
	GetUserRoles(ctx context.Context, tenantID, userID string) ([]domain.Role, error)
	CreatePermission(ctx context.Context, perm *domain.Permission) error
	GetPermissionByName(ctx context.Context, tenantID, name string) (*domain.Permission, error)
	ListPermissions(ctx context.Context, tenantID string) ([]*domain.Permission, error)
}

type WebAuthnRepository interface {
	GetCredentialByID(ctx context.Context, tenantID string, credentialID []byte) (*domain.WebAuthnCredential, error)
	GetCredentialsByUserID(ctx context.Context, tenantID, userID string) ([]*domain.WebAuthnCredential, error)
	CreateCredential(ctx context.Context, tenantID string, cred *domain.WebAuthnCredential) error
	UpdateCredential(ctx context.Context, tenantID string, cred *domain.WebAuthnCredential) error
	DeleteCredential(ctx context.Context, tenantID string, credentialID []byte) error
}

type WebAuthnSessionStore interface {
	SaveWebAuthnSession(ctx context.Context, key string, session *domain.WebAuthnSessionData, ttl time.Duration) error
	GetWebAuthnSession(ctx context.Context, key string) (*domain.WebAuthnSessionData, error)
	DeleteWebAuthnSession(ctx context.Context, key string) error
}

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*domain.Tenant, error)
}

type ClientRepository interface {
	Create(ctx context.Context, client *domain.Client) error
	GetByClientID(ctx context.Context, clientID string) (*domain.Client, error)
	Update(ctx context.Context, client *domain.Client) error
	Delete(ctx context.Context, clientID string) error
}

type BackupCodeRepository interface {
	CreateMany(ctx context.Context, codes []*domain.BackupCode) error
	GetActiveByUserID(ctx context.Context, tenantID, userID string) ([]*domain.BackupCode, error)
	MarkAsUsed(ctx context.Context, id string) error
	DeleteAllByUserID(ctx context.Context, tenantID, userID string) error
}

type WebhookRepository interface {
	Create(ctx context.Context, subscription *domain.WebhookSubscription) error
	GetByID(ctx context.Context, tenantID, id string) (*domain.WebhookSubscription, error)
	GetByTenantID(ctx context.Context, tenantID string) ([]*domain.WebhookSubscription, error)
	GetSubscriptionsForEvent(ctx context.Context, tenantID string, eventType domain.EventType) ([]*domain.WebhookSubscription, error)
	Update(ctx context.Context, subscription *domain.WebhookSubscription) error
	Delete(ctx context.Context, tenantID, id string) error
}
