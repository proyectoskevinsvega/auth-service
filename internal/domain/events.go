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
)

type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	UserID    string                 `json:"user_id"`
	Email     string                 `json:"email,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

func NewEvent(eventType EventType, userID, email string, data map[string]interface{}) *Event {
	return &Event{
		ID:        uuid.New().String(),
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

func NewAuditLogEntry(userID, action, ipAddress, userAgent, country string, success bool, errorMsg string, metadata map[string]interface{}) *AuditLogEntry {
	return &AuditLogEntry{
		ID:        uuid.New().String(),
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
