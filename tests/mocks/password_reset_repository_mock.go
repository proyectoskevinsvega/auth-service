package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

// MockPasswordResetRepository is a mock implementation of ports.PasswordResetRepository
type MockPasswordResetRepository struct {
	mock.Mock
}

func (m *MockPasswordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockPasswordResetRepository) GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PasswordResetToken), args.Error(1)
}

func (m *MockPasswordResetRepository) GetByCode(ctx context.Context, userID, code string) (*domain.PasswordResetToken, error) {
	args := m.Called(ctx, userID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PasswordResetToken), args.Error(1)
}

func (m *MockPasswordResetRepository) MarkAsUsed(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockPasswordResetRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
