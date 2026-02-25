package http

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
)

type WebhookClient struct {
	client *http.Client
	logger zerolog.Logger
}

func NewWebhookClient(logger zerolog.Logger) *WebhookClient {
	return &WebhookClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (c *WebhookClient) Send(ctx context.Context, sub *domain.WebhookSubscription, payload *domain.WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sub.URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Vertercloud-Webhook/1.0")
	req.Header.Set("X-Vertercloud-Event", string(payload.Type))
	req.Header.Set("X-Vertercloud-Delivery", payload.ID)

	// HMAC Signature
	signature := c.calculateSignature(body, sub.Secret)
	req.Header.Set("X-Vertercloud-Signature", signature)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	c.logger.Debug().
		Str("url", sub.URL).
		Str("event", string(payload.Type)).
		Int("status", resp.StatusCode).
		Msg("webhook sent successfully")

	return nil
}

func (c *WebhookClient) calculateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
