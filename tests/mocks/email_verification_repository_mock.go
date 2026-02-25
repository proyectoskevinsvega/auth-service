package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

type MockEmailVerificationRepository struct {
	mock.Mock
}

func (m *MockEmailVerificationRepository) Create(ctx context.Context, verification *domain.EmailVerification) error {
	args := m.Called(ctx, verification)
	return args.Error(0)
}

func (m *MockEmailVerificationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerification, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EmailVerification), args.Error(1)
}

func (m *MockEmailVerificationRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.EmailVerification, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.EmailVerification), args.Error(1)
}

func (m *MockEmailVerificationRepository) MarkAsVerified(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

func (m *MockEmailVerificationRepository) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockEmailVerificationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
