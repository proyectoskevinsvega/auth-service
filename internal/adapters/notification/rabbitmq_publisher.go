package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

// RabbitMQPublisher implements NotificationPublisher using RabbitMQ
type RabbitMQPublisher struct {
	channel *amqp091.Channel
	exchange string
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher
func NewRabbitMQPublisher(channel *amqp091.Channel, exchange string) ports.NotificationPublisher {
	return &RabbitMQPublisher{
		channel:  channel,
		exchange: exchange,
	}
}

// Publish publishes an event to RabbitMQ
func (p *RabbitMQPublisher) Publish(ctx context.Context, event *domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Routing key based on event type
	routingKey := string(event.Type)

	err = p.channel.PublishWithContext(ctx,
		p.exchange,    // exchange
		routingKey,    // routing key
		false,         // mandatory
		false,         // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp091.Persistent,
			Headers: amqp091.Table{
				"tenant_id": event.TenantID,
				"event_type": string(event.Type),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish to rabbitmq: %w", err)
	}

	return nil
}
