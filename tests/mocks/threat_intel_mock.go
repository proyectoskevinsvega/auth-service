package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

type MockThreatIntelligenceService struct {
	mock.Mock
}

func (m *MockThreatIntelligenceService) CheckIP(ctx context.Context, ip string) (*domain.ThreatIntelligence, error) {
	args := m.Called(ctx, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ThreatIntelligence), args.Error(1)
}
