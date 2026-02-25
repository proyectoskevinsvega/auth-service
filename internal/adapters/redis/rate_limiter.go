package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitPrefix = "auth:ratelimit:"
const blockedIPPrefix = "auth:blocked:"

type RateLimiter struct {
	client redis.UniversalClient
}

func NewRateLimiter(client redis.UniversalClient) *RateLimiter {
	return &RateLimiter{
		client: client,
	}
}

func (r *RateLimiter) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	fullKey := rateLimitPrefix + key

	count, err := r.client.Get(ctx, fullKey).Int()
	if err != nil {
		if err == redis.Nil {
			// No previous attempts, allow
			return false, nil
		}
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}

	return count >= limit, nil
}

func (r *RateLimiter) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	fullKey := rateLimitPrefix + key

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	count, err := incr.Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get incremented value: %w", err)
	}

	return int(count), nil
}

func (r *RateLimiter) Reset(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	fullKey := rateLimitPrefix + key

	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

func (r *RateLimiter) GetCount(ctx context.Context, key string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	fullKey := rateLimitPrefix + key

	count, err := r.client.Get(ctx, fullKey).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get rate limit count: %w", err)
	}

	return count, nil
}
