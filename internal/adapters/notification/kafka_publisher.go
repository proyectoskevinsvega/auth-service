package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/segmentio/kafka-go"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

// KafkaPublisher implements NotificationPublisher using Kafka for event sourcing
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher creates a new Kafka publisher for events
func NewKafkaPublisher(brokers []string) ports.NotificationPublisher {
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "auth.events"
	}
	
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
	}

	return &KafkaPublisher{
		writer: writer,
	}
}

// Publish publishes an event to Kafka
func (p *KafkaPublisher) Publish(ctx context.Context, event *domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Key by tenant_id for partitioning (events from same tenant go to same partition)
	key := []byte(event.TenantID)
	if event.TenantID == "" {
		key = []byte("global")
	}

	msg := kafka.Message{
		Key:   key,
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "tenant_id", Value: []byte(event.TenantID)},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write to kafka: %w", err)
	}

	return nil
}

// Close closes the Kafka writer
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
