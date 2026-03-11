package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

// ReliableRabbitMQPublisher publica eventos con reintentos automáticos
type ReliableRabbitMQPublisher struct {
	channel      *amqp091.Channel
	exchange     string
	maxRetries   int
	baseDelay    time.Duration
	maxDelay     time.Duration
}

// NewReliableRabbitMQPublisher crea un publisher con reintentos
func NewReliableRabbitMQPublisher(channel *amqp091.Channel, exchange string) ports.NotificationPublisher {
	return &ReliableRabbitMQPublisher{
		channel:    channel,
		exchange:   exchange,
		maxRetries: 5,
		baseDelay:  1 * time.Second,
		maxDelay:   30 * time.Second,
	}
}

// Publish publica un evento con reintentos automáticos
func (p *ReliableRabbitMQPublisher) Publish(ctx context.Context, event *domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	routingKey := string(event.Type)

	// Intentar publicar con reintentos exponenciales
	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if attempt > 0 {
			// Calcular delay exponencial: 1s, 2s, 4s, 8s, 16s...
			delay := p.baseDelay * time.Duration(1<<attempt)
			if delay > p.maxDelay {
				delay = p.maxDelay
			}
			
			fmt.Printf("Retrying RabbitMQ publish in %v (attempt %d/%d)...\n", delay, attempt+1, p.maxRetries)
			time.Sleep(delay)
		}

		// Verificar que el canal esté abierto
		if p.channel == nil || p.channel.IsClosed() {
			lastErr = fmt.Errorf("rabbitmq channel is closed")
			continue
		}

		err = p.channel.PublishWithContext(ctx,
			p.exchange,
			routingKey,
			false,
			false,
			amqp091.Publishing{
				ContentType:  "application/json",
				Body:         data,
				DeliveryMode: amqp091.Persistent,
				Headers: amqp091.Table{
					"tenant_id":  event.TenantID,
					"event_type": string(event.Type),
					"retry_count": attempt,
				},
			},
		)

		if err == nil {
			// Éxito
			if attempt > 0 {
				fmt.Printf("RabbitMQ publish succeeded after %d retries\n", attempt)
			}
			return nil
		}

		lastErr = err
		fmt.Printf("RabbitMQ publish failed (attempt %d/%d): %v\n", attempt+1, p.maxRetries, err)
	}

	return fmt.Errorf("failed to publish after %d retries: %w", p.maxRetries, lastErr)
}
