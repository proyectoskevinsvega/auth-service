package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

type MockWebAuthnRepository struct {
	mock.Mock
}

func (m *MockWebAuthnRepository) GetCredentialByID(ctx context.Context, tenantID string, credentialID []byte) (*domain.WebAuthnCredential, error) {
	args := m.Called(ctx, tenantID, credentialID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WebAuthnCredential), args.Error(1)
}

func (m *MockWebAuthnRepository) GetCredentialsByUserID(ctx context.Context, tenantID, userID string) ([]*domain.WebAuthnCredential, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WebAuthnCredential), args.Error(1)
}

func (m *MockWebAuthnRepository) CreateCredential(ctx context.Context, tenantID string, cred *domain.WebAuthnCredential) error {
	args := m.Called(ctx, tenantID, cred)
	return args.Error(0)
}

func (m *MockWebAuthnRepository) UpdateCredential(ctx context.Context, tenantID string, cred *domain.WebAuthnCredential) error {
	args := m.Called(ctx, tenantID, cred)
	return args.Error(0)
}

func (m *MockWebAuthnRepository) DeleteCredential(ctx context.Context, tenantID string, credentialID []byte) error {
	args := m.Called(ctx, tenantID, credentialID)
	return args.Error(0)
}
