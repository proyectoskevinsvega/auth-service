package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
)

// RabbitMQWebhookDistributor distributes webhook tasks via RabbitMQ
type RabbitMQWebhookDistributor struct {
	channel   *amqp091.Channel
	queueName string
	log       zerolog.Logger
}

// NewRabbitMQWebhookDistributor creates a new webhook distributor using RabbitMQ
func NewRabbitMQWebhookDistributor(channel *amqp091.Channel, log zerolog.Logger) *RabbitMQWebhookDistributor {
	return &RabbitMQWebhookDistributor{
		channel:   channel,
		queueName: "auth.webhooks",
		log:       log.With().Str("component", "rabbitmq_webhook_distributor").Logger(),
	}
}

// DistributeTaskDeliverWebhook sends a webhook task to the RabbitMQ queue (implements TaskDistributor)
func (d *RabbitMQWebhookDistributor) DistributeTaskDeliverWebhook(ctx context.Context, sub *domain.WebhookSubscription, payload *domain.WebhookPayload, opts ...asynq.Option) error {
	task := map[string]interface{}{
		"id":          generateWebhookTaskID(),
		"type":        "webhook.send",
		"webhook_id":  sub.ID,
		"tenant_id":   sub.TenantID,
		"url":         sub.URL,
		"secret":      sub.Secret,
		"event_type":  string(payload.Type),
		"payload":     payload,
		"retries":     0,
		"max_retries": 5,
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook task: %w", err)
	}

	// Ensure queue exists
	if _, err := d.channel.QueueDeclare(
		d.queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	); err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := d.channel.PublishWithContext(ctx,
		"",          // exchange
		d.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp091.Persistent,
			Headers: amqp091.Table{
				"webhook_id":  sub.ID,
				"tenant_id":   sub.TenantID,
				"event_type":  string(payload.Type),
				"retry_count": int32(0),
			},
		},
	); err != nil {
		return fmt.Errorf("failed to publish webhook task: %w", err)
	}

	d.log.Info().
		Str("webhook_id", sub.ID).
		Str("tenant_id", sub.TenantID).
		Str("event_type", string(payload.Type)).
		Msg("Webhook task queued to RabbitMQ")

	return nil
}

func generateWebhookTaskID() string {
	return fmt.Sprintf("wh_%d", time.Now().UnixNano())
}
