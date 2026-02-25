package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/usecase"
)

type EventWorker struct {
	redisClient redis.UniversalClient
	queue       string
	webhookUC   *usecase.WebhookUseCase
	logger      zerolog.Logger
}

func NewEventWorker(redisClient redis.UniversalClient, queue string, webhookUC *usecase.WebhookUseCase, logger zerolog.Logger) *EventWorker {
	return &EventWorker{
		redisClient: redisClient,
		queue:       queue,
		webhookUC:   webhookUC,
		logger:      logger,
	}
}

func (w *EventWorker) Start(ctx context.Context) {
	w.logger.Info().Str("queue", w.queue).Msg("starting event worker")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("stopping event worker")
			return
		default:
			// BLPop bloquea hasta que haya un elemento o expire el timeout
			result, err := w.redisClient.BLPop(ctx, 5*time.Second, w.queue).Result()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				w.logger.Error().Err(err).Msg("failed to pop event from redis")
				time.Sleep(time.Second) // Evitar bucle infinito de errores
				continue
			}

			if len(result) < 2 {
				continue
			}

			eventJSON := result[1]
			var event domain.Event
			if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
				w.logger.Error().Err(err).Str("event", eventJSON).Msg("failed to unmarshal event")
				continue
			}

			w.logger.Debug().Str("event_id", event.ID).Str("type", string(event.Type)).Msg("processing event")

			if err := w.webhookUC.ProcessEvent(ctx, &event); err != nil {
				w.logger.Error().Err(err).Str("event_id", event.ID).Msg("failed to process event")
			}
		}
	}
}
