package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vertercloud/auth-service/internal/domain"
)

const sessionPrefix = "auth:session:"

type SessionStore struct {
	client redis.UniversalClient
}

func NewSessionStore(client redis.UniversalClient) *SessionStore {
	return &SessionStore{
		client: client,
	}
}

func (s *SessionStore) Set(ctx context.Context, sessionID string, session *domain.Session, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := sessionPrefix + sessionID

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}

func (s *SessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := sessionPrefix + sessionID

	val, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal(val, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := sessionPrefix + sessionID

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *SessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := sessionPrefix + sessionID

	count, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return count > 0, nil
}

// UpdateLastUsed atomically updates the last_used timestamp using a Lua script
// to avoid race conditions when multiple requests update the same session
func (s *SessionStore) UpdateLastUsed(ctx context.Context, sessionID string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := sessionPrefix + sessionID
	now := time.Now().Unix()

	// Lua script to atomically update last_used field in JSON
	script := `
		local key = KEYS[1]
		local now = ARGV[1]
		local ttl = ARGV[2]

		local data = redis.call('GET', key)
		if not data then
			return redis.error_reply('session not found')
		end

		local session = cjson.decode(data)
		session.last_used_at = now

		local updated = cjson.encode(session)
		redis.call('SET', key, updated, 'EX', ttl)
		return 1
	`

	err := s.client.Eval(ctx, script, []string{key}, now, int(ttl.Seconds())).Err()
	if err != nil {
		if err.Error() == "session not found" {
			return domain.ErrSessionNotFound
		}
		return fmt.Errorf("failed to update last used: %w", err)
	}

	return nil
}
