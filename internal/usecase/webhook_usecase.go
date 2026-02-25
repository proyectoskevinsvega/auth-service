package usecase

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

// WebhookUseCase manages webhook subscriptions and processes auth events
type WebhookUseCase struct {
	repo   ports.WebhookRepository
	sender ports.WebhookSender
	logger zerolog.Logger
}

// NewWebhookUseCase creates a new WebhookUseCase
func NewWebhookUseCase(repo ports.WebhookRepository, sender ports.WebhookSender, logger zerolog.Logger) *WebhookUseCase {
	return &WebhookUseCase{
		repo:   repo,
		sender: sender,
		logger: logger,
	}
}

// CreateSubscription creates a new webhook subscription
func (uc *WebhookUseCase) CreateSubscription(ctx context.Context, sub *domain.WebhookSubscription) error {
	return uc.repo.Create(ctx, sub)
}

// GetSubscription retrieves a webhook subscription by ID
func (uc *WebhookUseCase) GetSubscription(ctx context.Context, tenantID, id string) (*domain.WebhookSubscription, error) {
	return uc.repo.GetByID(ctx, tenantID, id)
}

// DeleteSubscription deletes a webhook subscription
func (uc *WebhookUseCase) DeleteSubscription(ctx context.Context, tenantID, id string) error {
	return uc.repo.Delete(ctx, tenantID, id)
}

// ListSubscriptions returns all webhook subscriptions for a tenant
func (uc *WebhookUseCase) ListSubscriptions(ctx context.Context, tenantID string) ([]*domain.WebhookSubscription, error) {
	return uc.repo.GetByTenantID(ctx, tenantID)
}

// ProcessEvent finds active subscriptions for the event type and dispatches the webhook
func (uc *WebhookUseCase) ProcessEvent(ctx context.Context, event *domain.Event) error {
	subs, err := uc.repo.GetSubscriptionsForEvent(ctx, event.TenantID, event.Type)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions for event %s: %w", event.Type, err)
	}

	if len(subs) == 0 {
		return nil
	}

	payload := &domain.WebhookPayload{
		ID:        event.ID,
		Type:      event.Type,
		TenantID:  event.TenantID,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	}

	for _, sub := range subs {
		go func(s *domain.WebhookSubscription) {
			if err := uc.sender.Send(context.Background(), s, payload); err != nil {
				uc.logger.Error().
					Err(err).
					Str("url", s.URL).
					Str("event", string(event.Type)).
					Msg("failed to send webhook")
			}
		}(sub)
	}

	return nil
}
