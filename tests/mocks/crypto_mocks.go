package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

// MockJWTService is a mock implementation of ports.JWTService
type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) Generate(ctx context.Context, token *domain.Token) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) Verify(ctx context.Context, tokenString string) (*domain.Token, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Token), args.Error(1)
}

func (m *MockJWTService) GetPublicKeyJWKS() (map[string]interface{}, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockPasswordHasher is a mock implementation of ports.PasswordHasher
type MockPasswordHasher struct {
	mock.Mock
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordHasher) Verify(password, hash string) (bool, error) {
	args := m.Called(password, hash)
	return args.Bool(0), args.Error(1)
}

// MockTOTPService is a mock implementation of ports.TOTPService
type MockTOTPService struct {
	mock.Mock
}

func (m *MockTOTPService) Generate(email string) (secret, qrCode string, err error) {
	args := m.Called(email)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockTOTPService) Verify(code, secret string) (bool, error) {
	args := m.Called(code, secret)
	return args.Bool(0), args.Error(1)
}

// MockTokenGenerator is a mock implementation of ports.TokenGenerator
type MockTokenGenerator struct {
	mock.Mock
}

func (m *MockTokenGenerator) GenerateSecureToken(length int) (string, error) {
	args := m.Called(length)
	return args.String(0), args.Error(1)
}
