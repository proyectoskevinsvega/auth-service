package domain

import (
	"time"

	"github.com/google/uuid"
)

type Token struct {
	JTI         string
	TenantID    string
	UserID      string
	Email       string
	IssuedAt    int64
	ExpiresAt   int64
	Roles       []string // Added for RBAC
	Scopes      []string // Added for OAuth2 scopes
	Permissions []string // Added for granular RBAC
}

func NewToken(tenantID, userID, email string, expiry time.Duration) *Token {
	now := time.Now()
	return &Token{
		JTI:         uuid.Must(uuid.NewV7()).String(),
		TenantID:    tenantID,
		UserID:      userID,
		Email:       email,
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(expiry).Unix(),
		Roles:       []string{},
		Scopes:      []string{},
		Permissions: []string{},
	}
}

func (t *Token) IsExpired() bool {
	return time.Now().Unix() > t.ExpiresAt
}

func (t *Token) TimeToLive() time.Duration {
	if t.IsExpired() {
		return 0
	}
	return time.Until(time.Unix(t.ExpiresAt, 0))
}

func (t *Token) IssuedAtTime() time.Time {
	return time.Unix(t.IssuedAt, 0)
}

func (t *Token) ExpiresAtTime() time.Time {
	return time.Unix(t.ExpiresAt, 0)
}

func (t *Token) HasRole(role string) bool {
	for _, r := range t.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (t *Token) HasScope(scope string) bool {
	for _, s := range t.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func (t *Token) HasPermission(permission string) bool {
	for _, p := range t.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

type RefreshToken struct {
	ID            string
	TenantID      string
	UserID        string
	SessionID     string
	TokenHash     string
	PreviousToken string // For rotation detection
	ExpiresAt     time.Time
	CreatedAt     time.Time
	Revoked       bool
	RevokedAt     *time.Time
}

func NewRefreshToken(tenantID, userID, sessionID, tokenHash string, expiry time.Duration) *RefreshToken {
	now := time.Now().UTC()
	return &RefreshToken{
		ID:        uuid.Must(uuid.NewV7()).String(),
		TenantID:  tenantID,
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(expiry),
		CreatedAt: now,
		Revoked:   false,
	}
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().UTC().After(rt.ExpiresAt)
}

func (rt *RefreshToken) IsValid() bool {
	return !rt.Revoked && !rt.IsExpired()
}

func (rt *RefreshToken) Revoke() {
	now := time.Now().UTC()
	rt.Revoked = true
	rt.RevokedAt = &now
}

func (rt *RefreshToken) Rotate(newTokenHash string) *RefreshToken {
	newToken := &RefreshToken{
		ID:            uuid.Must(uuid.NewV7()).String(),
		TenantID:      rt.TenantID,
		UserID:        rt.UserID,
		SessionID:     rt.SessionID,
		TokenHash:     newTokenHash,
		PreviousToken: rt.ID,
		ExpiresAt:     rt.ExpiresAt, // Keep same expiry
		CreatedAt:     time.Now().UTC(),
		Revoked:       false,
	}
	rt.Revoke()
	return newToken
}

type PasswordResetToken struct {
	ID        string
	TenantID  string
	UserID    string
	Token     string
	Code      string // Código de 6 dígitos
	ExpiresAt time.Time
	CreatedAt time.Time
	Used      bool
	UsedAt    *time.Time
}

func NewPasswordResetToken(tenantID, userID, token, code string) *PasswordResetToken {
	now := time.Now().UTC() // Use UTC to avoid timezone issues
	return &PasswordResetToken{
		ID:        uuid.Must(uuid.NewV7()).String(),
		TenantID:  tenantID,
		UserID:    userID,
		Token:     token,
		Code:      code,
		ExpiresAt: now.Add(15 * time.Minute), // 15 minutos de expiración
		CreatedAt: now,
		Used:      false,
	}
}

func (prt *PasswordResetToken) IsExpired() bool {
	return time.Now().UTC().After(prt.ExpiresAt)
}

func (prt *PasswordResetToken) IsValid() bool {
	return !prt.Used && !prt.IsExpired()
}

func (prt *PasswordResetToken) MarkAsUsed() {
	now := time.Now().UTC()
	prt.Used = true
	prt.UsedAt = &now
}
