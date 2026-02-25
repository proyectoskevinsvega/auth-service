package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

// MockTokenCache is a mock implementation of ports.TokenCache
type MockTokenCache struct {
	mock.Mock
}

func (m *MockTokenCache) Set(ctx context.Context, jti string, token *domain.Token, ttl time.Duration) error {
	args := m.Called(ctx, jti, token, ttl)
	return args.Error(0)
}

func (m *MockTokenCache) Get(ctx context.Context, jti string) (*domain.Token, error) {
	args := m.Called(ctx, jti)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Token), args.Error(1)
}

func (m *MockTokenCache) Delete(ctx context.Context, jti string) error {
	args := m.Called(ctx, jti)
	return args.Error(0)
}

func (m *MockTokenCache) Exists(ctx context.Context, jti string) (bool, error) {
	args := m.Called(ctx, jti)
	return args.Bool(0), args.Error(1)
}

// MockTokenBlacklist is a mock implementation of ports.TokenBlacklist
type MockTokenBlacklist struct {
	mock.Mock
}

func (m *MockTokenBlacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	args := m.Called(ctx, jti, ttl)
	return args.Error(0)
}

func (m *MockTokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	args := m.Called(ctx, jti)
	return args.Bool(0), args.Error(1)
}

func (m *MockTokenBlacklist) Remove(ctx context.Context, jti string) error {
	args := m.Called(ctx, jti)
	return args.Error(0)
}

// MockRateLimiter is a mock implementation of ports.RateLimiter
type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	args := m.Called(ctx, key, limit, window)
	return args.Bool(0), args.Error(1)
}

func (m *MockRateLimiter) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	args := m.Called(ctx, key, window)
	return args.Int(0), args.Error(1)
}

func (m *MockRateLimiter) Reset(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockRateLimiter) GetCount(ctx context.Context, key string) (int, error) {
	args := m.Called(ctx, key)
	return args.Int(0), args.Error(1)
}

// MockSessionStore is a mock implementation of ports.SessionStore
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Set(ctx context.Context, sessionID string, session *domain.Session, ttl time.Duration) error {
	args := m.Called(ctx, sessionID, session, ttl)
	return args.Error(0)
}

func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	args := m.Called(ctx, sessionID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSessionStore) UpdateLastUsed(ctx context.Context, sessionID string, ttl time.Duration) error {
	args := m.Called(ctx, sessionID, ttl)
	return args.Error(0)
}
