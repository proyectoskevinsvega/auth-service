package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const blacklistPrefix = "auth:blacklist:"

type TokenBlacklist struct {
	client redis.UniversalClient
}

func NewTokenBlacklist(client redis.UniversalClient) *TokenBlacklist {
	return &TokenBlacklist{
		client: client,
	}
}

func (b *TokenBlacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := blacklistPrefix + jti

	if err := b.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	return nil
}

func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := blacklistPrefix + jti

	count, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return count > 0, nil
}

func (b *TokenBlacklist) Remove(ctx context.Context, jti string) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := blacklistPrefix + jti

	if err := b.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}

	return nil
}
