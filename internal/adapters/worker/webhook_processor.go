package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

// WebhookProcessor handles webhook delivery using RabbitMQ
type WebhookProcessor struct {
	webhookRepo ports.WebhookRepository
	httpClient  *http.Client
	log         zerolog.Logger
}

// NewWebhookProcessor creates a new webhook processor
func NewWebhookProcessor(webhookRepo ports.WebhookRepository, log zerolog.Logger) *WebhookProcessor {
	return &WebhookProcessor{
		webhookRepo: webhookRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log.With().Str("processor", "webhook").Logger(),
	}
}

// ProcessWebhook processes a webhook delivery task
func (p *WebhookProcessor) ProcessWebhook(ctx context.Context, tenantID string, eventType string, payload map[string]interface{}) error {
	// Get tenant's webhook subscriptions
	webhooks, err := p.webhookRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get webhook configs: %w", err)
	}

	if len(webhooks) == 0 {
		p.log.Debug().Str("tenant_id", tenantID).Msg("No webhooks configured, skipping")
		return nil
	}

	// Build webhook payload
	webhookPayload := map[string]interface{}{
		"event_id":    generateEventID(),
		"event_type":  eventType,
		"tenant_id":   tenantID,
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
		"data":        payload,
	}

	body, err := json.Marshal(webhookPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Send to all active webhooks
	var lastErr error
	for _, webhook := range webhooks {
		if !webhook.Active {
			continue
		}

		if err := p.sendWebhook(ctx, webhook, eventType, body); err != nil {
			p.log.Error().Err(err).
				Str("tenant_id", tenantID).
				Str("webhook_id", webhook.ID).
				Msg("Failed to send webhook")
			lastErr = err
		}
	}

	return lastErr
}

// sendWebhook sends a single webhook
func (p *WebhookProcessor) sendWebhook(ctx context.Context, webhook *domain.WebhookSubscription, eventType string, body []byte) error {
	// Calculate signature
	signature := p.calculateSignature(body, webhook.Secret)

	// Send HTTP POST
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Event-Type", eventType)
	req.Header.Set("User-Agent", "VerterCloud-Auth/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Log result
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p.log.Info().
			Str("webhook_id", webhook.ID).
			Str("event_type", eventType).
			Int("status_code", resp.StatusCode).
			Msg("Webhook delivered successfully")
	} else {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// calculateSignature generates HMAC-SHA256 signature for webhook payload
func (p *WebhookProcessor) calculateSignature(body []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}
