package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

// MockGeolocationService is a mock implementation of ports.GeolocationService
type MockGeolocationService struct {
	mock.Mock
}

func (m *MockGeolocationService) GetLocation(ctx context.Context, ip string) (*domain.Geolocation, error) {
	args := m.Called(ctx, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Geolocation), args.Error(1)
}
