package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID           string
	TenantID     string
	UserID       string
	JTI          string // JWT ID for tracking current session
	IPAddress    string
	Country      string
	Device       string
	UserAgent    string
	CreatedAt    time.Time
	LastUsedAt   time.Time
	ExpiresAt    time.Time
	Revoked      bool
	RevokedAt    *time.Time
	RevokedBy    string // "user", "system", "security"
	RevokeReason string
	IsCurrent    bool `db:"-"` // Not persisted, used for API responses
}

type NewSessionInput struct {
	UserID        string
	IPAddress     string
	Country       string
	Device        string
	UserAgent     string
	InactivityTTL time.Duration
}

func NewSession(input NewSessionInput) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:         uuid.New().String(),
		TenantID:   "", // Should be set by caller or during initialization
		UserID:     input.UserID,
		IPAddress:  input.IPAddress,
		Country:    input.Country,
		Device:     input.Device,
		UserAgent:  input.UserAgent,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  now.Add(input.InactivityTTL),
		Revoked:    false,
	}
}

func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

func (s *Session) IsActive() bool {
	return !s.Revoked && !s.IsExpired()
}

func (s *Session) UpdateLastUsed(inactivityTTL time.Duration) {
	now := time.Now().UTC()
	s.LastUsedAt = now
	s.ExpiresAt = now.Add(inactivityTTL)
}

func (s *Session) Revoke(revokedBy, reason string) {
	now := time.Now().UTC()
	s.Revoked = true
	s.RevokedAt = &now
	s.RevokedBy = revokedBy
	s.RevokeReason = reason
}

func (s *Session) IsInactiveFor(duration time.Duration) bool {
	return time.Since(s.LastUsedAt) > duration
}

func (s *Session) IsNewCountry(previousCountry string) bool {
	return previousCountry != "" && s.Country != "" && s.Country != previousCountry
}

func (s *Session) IsNewIP(previousIP string) bool {
	return previousIP != "" && s.IPAddress != "" && s.IPAddress != previousIP
}

type AnomalyDetection struct {
	NewCountry    bool
	NewIP         bool
	SuspiciousUA  bool
	RapidRequests bool
}

func (s *Session) DetectAnomalies(previousSessions []*Session) AnomalyDetection {
	anomalies := AnomalyDetection{}

	if len(previousSessions) == 0 {
		return anomalies
	}

	// Check for new country
	lastSession := previousSessions[0]
	if s.IsNewCountry(lastSession.Country) {
		anomalies.NewCountry = true
	}

	// Check for new IP (simple check, can be enhanced)
	if s.IsNewIP(lastSession.IPAddress) {
		anomalies.NewIP = true
	}

	return anomalies
}
