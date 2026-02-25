package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

type MockBackupCodeRepository struct {
	mock.Mock
}

func (m *MockBackupCodeRepository) CreateMany(ctx context.Context, codes []*domain.BackupCode) error {
	args := m.Called(ctx, codes)
	return args.Error(0)
}

func (m *MockBackupCodeRepository) GetActiveByUserID(ctx context.Context, tenantID, userID string) ([]*domain.BackupCode, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.BackupCode), args.Error(1)
}

func (m *MockBackupCodeRepository) MarkAsUsed(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBackupCodeRepository) DeleteAllByUserID(ctx context.Context, tenantID, userID string) error {
	args := m.Called(ctx, tenantID, userID)
	return args.Error(0)
}
