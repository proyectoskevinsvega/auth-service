package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) GetRoleByName(ctx context.Context, tenantID, name string) (*domain.Role, error) {
	args := m.Called(ctx, tenantID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Role), args.Error(1)
}

func (m *MockRoleRepository) ListRoles(ctx context.Context, tenantID string) ([]*domain.Role, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Role), args.Error(1)
}

func (m *MockRoleRepository) CreatePermission(ctx context.Context, perm *domain.Permission) error {
	args := m.Called(ctx, perm)
	return args.Error(0)
}

func (m *MockRoleRepository) GetPermissionByName(ctx context.Context, tenantID, name string) (*domain.Permission, error) {
	args := m.Called(ctx, tenantID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Permission), args.Error(1)
}

func (m *MockRoleRepository) ListPermissions(ctx context.Context, tenantID string) ([]*domain.Permission, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Permission), args.Error(1)
}

func (m *MockRoleRepository) AddPermissionToRole(ctx context.Context, tenantID, roleID, permissionID string) error {
	args := m.Called(ctx, tenantID, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRoleRepository) AssignRoleToUser(ctx context.Context, tenantID, userID, roleID string) error {
	args := m.Called(ctx, tenantID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) RemoveRoleFromUser(ctx context.Context, tenantID, userID, roleID string) error {
	args := m.Called(ctx, tenantID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) GetUserRoles(ctx context.Context, tenantID, userID string) ([]domain.Role, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Role), args.Error(1)
}
