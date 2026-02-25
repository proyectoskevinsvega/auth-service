package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

// MockAuditLogRepository is a mock implementation of ports.AuditLogRepository
type MockAuditLogRepository struct {
	mock.Mock
}

func (m *MockAuditLogRepository) Create(ctx context.Context, entry *domain.AuditLogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditLogRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.AuditLogEntry, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AuditLogEntry), args.Error(1)
}
