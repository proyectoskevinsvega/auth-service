package integration

import (
	"context"

	"github.com/vertercloud/auth-service/internal/domain"
)

// mockEmailService is a no-op email service for tests
type mockEmailService struct{}

func (m *mockEmailService) SendWelcome(ctx context.Context, to, username string) error {
	return nil
}

func (m *mockEmailService) SendPasswordReset(ctx context.Context, to, code, resetURL string) error {
	return nil
}

func (m *mockEmailService) SendPasswordChanged(ctx context.Context, to string) error {
	return nil
}

func (m *mockEmailService) SendVerificationEmail(ctx context.Context, to, verificationURL string, metadata map[string]interface{}) error {
	return nil
}

func (m *mockEmailService) SendLoginAlert(ctx context.Context, to, ipAddress, country, device string) error {
	return nil
}

func (m *mockEmailService) Send2FAEnabled(ctx context.Context, to string) error {
	return nil
}

func (m *mockEmailService) Send2FADisabled(ctx context.Context, to string) error {
	return nil
}

func (m *mockEmailService) SendSecurityAlert(ctx context.Context, to, alertType, ipAddress string) error {
	return nil
}

// mockNotificationPublisher is a no-op notification publisher for tests
type mockNotificationPublisher struct{}

func (m *mockNotificationPublisher) Publish(ctx context.Context, event *domain.Event) error {
	return nil
}
