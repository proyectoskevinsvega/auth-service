package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vertercloud/auth-service/internal/domain"
)

const webauthnSessionPrefix = "auth:webauthn:session:"

type WebAuthnSessionStore struct {
	client redis.UniversalClient
}

func NewWebAuthnSessionStore(client redis.UniversalClient) *WebAuthnSessionStore {
	return &WebAuthnSessionStore{
		client: client,
	}
}

func (s *WebAuthnSessionStore) SaveWebAuthnSession(ctx context.Context, key string, session *domain.WebAuthnSessionData, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	redisKey := webauthnSessionPrefix + key

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal webauthn session: %w", err)
	}

	if err := s.client.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store webauthn session: %w", err)
	}

	return nil
}

func (s *WebAuthnSessionStore) GetWebAuthnSession(ctx context.Context, key string) (*domain.WebAuthnSessionData, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	redisKey := webauthnSessionPrefix + key

	val, err := s.client.Get(ctx, redisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrInvalidToken // O un error más específico para WebAuthn
		}
		return nil, fmt.Errorf("failed to get webauthn session: %w", err)
	}

	var session domain.WebAuthnSessionData
	if err := json.Unmarshal(val, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webauthn session: %w", err)
	}

	return &session, nil
}

func (s *WebAuthnSessionStore) DeleteWebAuthnSession(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	redisKey := webauthnSessionPrefix + key

	if err := s.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to delete webauthn session: %w", err)
	}

	return nil
}
