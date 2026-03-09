package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type NATSPublisher struct {
	js jetstream.JetStream
}

func NewNATSPublisher(js jetstream.JetStream) ports.NotificationPublisher {
	return &NATSPublisher{
		js: js,
	}
}

func (p *NATSPublisher) Publish(ctx context.Context, event *domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// We use users.created instead of auth.events.user.created because the user
	// created the stream with "users.*" subject in the CLI.
	var subject string
	switch event.Type {
	case domain.EventUserRegistered:
		subject = "users.created"
	default:
		// Fallback for other events
		subject = fmt.Sprintf("users.%s", event.Type)
	}

	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish to nats: %w", err)
	}

	return nil
}
