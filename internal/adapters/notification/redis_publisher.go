package notification

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/vertercloud/auth-service/internal/domain"
)

type RedisPublisher struct {
	client redis.UniversalClient
	queue  string
}

func NewRedisPublisher(client redis.UniversalClient, queue string) *RedisPublisher {
	return &RedisPublisher{
		client: client,
		queue:  queue,
	}
}

func (p *RedisPublisher) Publish(ctx context.Context, event *domain.Event) error {
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	if err := p.client.RPush(ctx, p.queue, data).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
