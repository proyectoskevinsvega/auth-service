package ports

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/vertercloud/auth-service/internal/domain"
)

type NotificationPublisher interface {
	// Publish publishes an event to the notification queue
	Publish(ctx context.Context, event *domain.Event) error
}

type EmailService interface {
	// SendVerificationEmail sends an email verification link
	SendVerificationEmail(ctx context.Context, to, name string, data map[string]interface{}) error

	// SendPasswordReset sends a password reset email with code and URL
	SendPasswordReset(ctx context.Context, to, code, resetURL string) error

	// SendWelcome sends a welcome email
	SendWelcome(ctx context.Context, to, name string) error

	// Send2FAEnabled sends an email when 2FA is enabled
	Send2FAEnabled(ctx context.Context, to string) error

	// SendSecurityAlert sends a security alert email
	SendSecurityAlert(ctx context.Context, to, subject, message string) error

	// SendPasswordExpiryWarning sends a warning email about password expiration
	SendPasswordExpiryWarning(ctx context.Context, to, name string, daysRemaining int) error
}

type WebhookSender interface {
	Send(ctx context.Context, sub *domain.WebhookSubscription, payload *domain.WebhookPayload) error
}

// WebhookDistributor distributes webhook tasks to a queue for background processing
type WebhookDistributor interface {
	Distribute(ctx context.Context, tenantID string, eventType string, payload map[string]interface{}) error
}

// TaskDistributor is the interface for enqueuing background tasks.
// Defined here (in ports) to avoid circular imports between usecase and worker packages.
type TaskDistributor interface {
	DistributeTaskDeliverWebhook(ctx context.Context, sub *domain.WebhookSubscription, payload *domain.WebhookPayload, opts ...asynq.Option) error
}
