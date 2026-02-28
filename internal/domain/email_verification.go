package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type EmailVerification struct {
	ID         string
	TenantID   string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	VerifiedAt *time.Time
	CreatedAt  time.Time
	IPAddress  string
	UserAgent  string
}

func NewEmailVerification(tenantID, userID, tokenHash string, expiryDuration time.Duration, ipAddress, userAgent string) *EmailVerification {
	now := time.Now().UTC()
	return &EmailVerification{
		ID:        uuid.Must(uuid.NewV7()).String(),
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(expiryDuration),
		CreatedAt: now,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
}

func (ev *EmailVerification) IsExpired() bool {
	return time.Now().UTC().After(ev.ExpiresAt)
}

func (ev *EmailVerification) IsVerified() bool {
	return ev.VerifiedAt != nil
}

func (ev *EmailVerification) MarkAsVerified() {
	now := time.Now().UTC()
	ev.VerifiedAt = &now
}

// Domain errors
var (
	ErrVerificationTokenNotFound = errors.New("verification token not found")
	ErrVerificationTokenExpired  = errors.New("verification token expired")
	ErrVerificationTokenUsed     = errors.New("verification token already used")
	ErrEmailAlreadyVerified      = errors.New("email already verified")
)
