package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventLoginNewCountry        EventType = "auth_login_new_country"
	EventPasswordChanged        EventType = "auth_password_changed"
	Event2FAEnabled             EventType = "auth_2fa_enabled"
	Event2FADisabled            EventType = "auth_2fa_disabled"
	EventSessionRevoked         EventType = "auth_session_revoked"
	EventAllSessionsRevoked     EventType = "auth_all_sessions_revoked"
	EventTokenStolen            EventType = "auth_token_stolen"
	EventUserRegistered         EventType = "auth_user_registered"
	EventPasswordResetRequested EventType = "auth_password_reset_requested"
	EventPasswordReset          EventType = "auth_password_reset"
	EventOAuthLinked            EventType = "auth_oauth_linked"
	EventLoginSuccess           EventType = "auth_login_success"
	EventLoginFailed            EventType = "auth_login_failed"
	EventAccountLocked          EventType = "auth_account_locked"
	EventEmailVerified          EventType = "auth_email_verified"
)

type Event struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Type      EventType              `json:"type"`
	UserID    string                 `json:"user_id"`
	Email     string                 `json:"email,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

func NewEvent(tenantID string, eventType EventType, userID, email string, data map[string]interface{}) *Event {
	return &Event{
		ID:        uuid.Must(uuid.NewV7()).String(),
		TenantID:  tenantID,
		Type:      eventType,
		UserID:    userID,
		Email:     email,
		Timestamp: time.Now(),
		Data:      data,
	}
}

func (e *Event) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type AuditLogEntry struct {
	ID        string
	TenantID  string
	UserID    string
	Action    string
	IPAddress string
	UserAgent string
	Country   string
	Success   bool
	ErrorMsg  string
	Metadata  map[string]interface{}
	CreatedAt time.Time
}

func NewAuditLogEntry(tenantID, userID, action, ipAddress, userAgent, country string, success bool, errorMsg string, metadata map[string]interface{}) *AuditLogEntry {
	return &AuditLogEntry{
		ID:        uuid.Must(uuid.NewV7()).String(),
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Country:   country,
		Success:   success,
		ErrorMsg:  errorMsg,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}
