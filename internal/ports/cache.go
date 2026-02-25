package ports

import (
	"context"
	"time"

	"github.com/vertercloud/auth-service/internal/domain"
)

type TokenCache interface {
	// Set caches a token with TTL
	Set(ctx context.Context, jti string, token *domain.Token, ttl time.Duration) error

	// Get retrieves a token from cache
	Get(ctx context.Context, jti string) (*domain.Token, error)

	// Delete removes a token from cache
	Delete(ctx context.Context, jti string) error

	// Exists checks if a token exists in cache
	Exists(ctx context.Context, jti string) (bool, error)
}

type TokenBlacklist interface {
	// Add adds a token to the blacklist with TTL
	Add(ctx context.Context, jti string, ttl time.Duration) error

	// IsBlacklisted checks if a token is blacklisted
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	// Remove removes a token from blacklist
	Remove(ctx context.Context, jti string) error
}

type RateLimiter interface {
	// CheckLimit checks if the limit is exceeded for a given key
	CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)

	// Increment increments the counter for a given key
	Increment(ctx context.Context, key string, window time.Duration) (int, error)

	// Reset resets the counter for a given key
	Reset(ctx context.Context, key string) error

	// GetCount gets the current count for a given key
	GetCount(ctx context.Context, key string) (int, error)
}

type SessionStore interface {
	// Set stores a session with TTL
	Set(ctx context.Context, sessionID string, session *domain.Session, ttl time.Duration) error

	// Get retrieves a session
	Get(ctx context.Context, sessionID string) (*domain.Session, error)

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// Exists checks if a session exists
	Exists(ctx context.Context, sessionID string) (bool, error)

	// UpdateLastUsed updates the last used timestamp
	UpdateLastUsed(ctx context.Context, sessionID string, ttl time.Duration) error
}
