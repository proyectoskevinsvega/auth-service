package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vertercloud/auth-service/internal/domain"
)

const tokenCachePrefix = "auth:token:"

type TokenCache struct {
	client redis.UniversalClient
}

func NewTokenCache(client redis.UniversalClient) *TokenCache {
	return &TokenCache{
		client: client,
	}
}

func (c *TokenCache) Set(ctx context.Context, jti string, token *domain.Token, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := tokenCachePrefix + jti

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache token: %w", err)
	}

	return nil
}

func (c *TokenCache) Get(ctx context.Context, jti string) (*domain.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := tokenCachePrefix + jti

	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get token from cache: %w", err)
	}

	var token domain.Token
	if err := json.Unmarshal(val, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

func (c *TokenCache) Delete(ctx context.Context, jti string) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := tokenCachePrefix + jti

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete token from cache: %w", err)
	}

	return nil
}

func (c *TokenCache) Exists(ctx context.Context, jti string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	key := tokenCachePrefix + jti

	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token existence: %w", err)
	}

	return count > 0, nil
}
