package worker

import (
	"encoding/json"

	"github.com/hibiken/asynq"
	"github.com/vertercloud/auth-service/internal/domain"
)

const (
	TaskDeliverWebhook = "task:deliver_webhook"
)

type DeliverWebhookPayload struct {
	Subscription *domain.WebhookSubscription `json:"subscription"`
	EventPayload *domain.WebhookPayload      `json:"event_payload"`
}

func NewDeliverWebhookTask(sub *domain.WebhookSubscription, payload *domain.WebhookPayload) (*asynq.Task, error) {
	jsonPayload, err := json.Marshal(DeliverWebhookPayload{
		Subscription: sub,
		EventPayload: payload,
	})
	if err != nil {
		return nil, err
	}

	// Default options: max 10 retries, standard queue
	return asynq.NewTask(TaskDeliverWebhook, jsonPayload, asynq.MaxRetry(10)), nil
}
