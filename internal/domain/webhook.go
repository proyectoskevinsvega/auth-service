package domain

import (
	"time"
)

// WebhookSubscription representa una suscripción a eventos vía webhook
type WebhookSubscription struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	URL        string    `json:"url"`
	Secret     string    `json:"secret"` // Secreto para firmar el payload (HMAC-SHA256)
	EventTypes []string  `json:"event_types"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WebhookPayload es el formato estándar enviado a los webhooks
type WebhookPayload struct {
	ID        string                 `json:"id"`        // ID del evento
	Type      EventType              `json:"type"`      // Tipo de evento
	TenantID  string                 `json:"tenant_id"` // ID del tenant
	Timestamp time.Time              `json:"timestamp"` // Cuándo ocurrió
	Data      map[string]interface{} `json:"data"`      // Datos específicos del evento
}
