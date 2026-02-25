package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type MockRiskService struct {
	mock.Mock
}

// Ensure MockRiskService implements ports.RiskService
var _ ports.RiskService = (*MockRiskService)(nil)

func (m *MockRiskService) AssessLoginRisk(ctx context.Context, user *domain.User, currentIP string) (*domain.RiskAssessment, *domain.Geolocation, error) {
	args := m.Called(ctx, user, currentIP)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*domain.Geolocation), args.Error(2)
	}
	return args.Get(0).(*domain.RiskAssessment), args.Get(1).(*domain.Geolocation), args.Error(2)
}
