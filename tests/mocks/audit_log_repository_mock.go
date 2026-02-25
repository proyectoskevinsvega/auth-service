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

func (m *MockAuditLogRepository) GetByUserID(ctx context.Context, tenantID, userID string, limit, offset int) ([]*domain.AuditLogEntry, error) {
	args := m.Called(ctx, tenantID, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AuditLogEntry), args.Error(1)
}

func (m *MockAuditLogRepository) Search(ctx context.Context, filter domain.AuditSearchFilter) ([]*domain.AuditLogEntry, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.AuditLogEntry), args.Int(1), args.Error(2)
}
