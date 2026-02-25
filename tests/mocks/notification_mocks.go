package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

// MockNotificationPublisher is a mock implementation of ports.NotificationPublisher
type MockNotificationPublisher struct {
	mock.Mock
}

func (m *MockNotificationPublisher) Publish(ctx context.Context, event *domain.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockEmailService is a mock implementation of ports.EmailService
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendVerificationEmail(ctx context.Context, to, name string, data map[string]interface{}) error {
	args := m.Called(ctx, to, name, data)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordReset(ctx context.Context, to, code, resetURL string) error {
	args := m.Called(ctx, to, code, resetURL)
	return args.Error(0)
}

func (m *MockEmailService) SendWelcome(ctx context.Context, to, name string) error {
	args := m.Called(ctx, to, name)
	return args.Error(0)
}

func (m *MockEmailService) Send2FAEnabled(ctx context.Context, to string) error {
	args := m.Called(ctx, to)
	return args.Error(0)
}

func (m *MockEmailService) SendSecurityAlert(ctx context.Context, to, subject, message string) error {
	args := m.Called(ctx, to, subject, message)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordExpiryWarning(ctx context.Context, to, name string, daysRemaining int) error {
	args := m.Called(ctx, to, name, daysRemaining)
	return args.Error(0)
}
