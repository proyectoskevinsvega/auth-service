package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/ports"
)

// MockOAuthProvider is a mock implementation of ports.OAuthProvider
type MockOAuthProvider struct {
	mock.Mock
}

func (m *MockOAuthProvider) GetAuthURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *MockOAuthProvider) Exchange(ctx context.Context, code string) (*ports.OAuthUserInfo, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.OAuthUserInfo), args.Error(1)
}
