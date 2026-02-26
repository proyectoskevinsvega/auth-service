package worker

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type RedisTaskDistributor struct {
	client *asynq.Client
	logger zerolog.Logger
}

func NewRedisTaskDistributor(client *asynq.Client, logger zerolog.Logger) ports.TaskDistributor {
	return &RedisTaskDistributor{
		client: client,
		logger: logger,
	}
}

func (distributor *RedisTaskDistributor) DistributeTaskDeliverWebhook(ctx context.Context, sub *domain.WebhookSubscription, payload *domain.WebhookPayload, opts ...asynq.Option) error {
	task, err := NewDeliverWebhookTask(sub, payload)
	if err != nil {
		return fmt.Errorf("failed to create deliver webhook task: %w", err)
	}

	info, err := distributor.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue deliver webhook task: %w", err)
	}

	distributor.logger.Debug().
		Str("type", task.Type()).
		Bytes("payload", task.Payload()).
		Str("queue", info.Queue).
		Int("max_retry", info.MaxRetry).
		Msg("enqueued webhook delivery task")

	return nil
}
