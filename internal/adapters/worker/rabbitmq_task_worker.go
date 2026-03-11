package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// Task represents a background job task
type Task struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Retries   int                    `json:"retries"`
	MaxRetries int                   `json:"max_retries"`
	CreatedAt time.Time              `json:"created_at"`
}

// TaskHandler processes a specific task type
type TaskHandler func(ctx context.Context, task Task) error

// RabbitMQWorker processes background tasks from RabbitMQ queues
type RabbitMQWorker struct {
	channel      *amqp091.Channel
	queueName    string
	handlers     map[string]TaskHandler
	log          zerolog.Logger
	maxRetries   int
	quit         chan struct{}
}

// NewRabbitMQWorker creates a new worker for processing background tasks
func NewRabbitMQWorker(channel *amqp091.Channel, queueName string, log zerolog.Logger) *RabbitMQWorker {
	return &RabbitMQWorker{
		channel:    channel,
		queueName:  queueName,
		handlers:   make(map[string]TaskHandler),
		log:        log.With().Str("worker", "rabbitmq_task_worker").Logger(),
		maxRetries: 3,
		quit:       make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a specific task type
func (w *RabbitMQWorker) RegisterHandler(taskType string, handler TaskHandler) {
	w.handlers[taskType] = handler
	w.log.Info().Str("task_type", taskType).Msg("Registered task handler")
}

// Start begins consuming and processing tasks
func (w *RabbitMQWorker) Start(ctx context.Context) error {
	w.log.Info().Str("queue", w.queueName).Msg("Starting RabbitMQ task worker")

	// Declare queue
	queue, err := w.channel.QueueDeclare(
		w.queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Set QoS (prefetch count)
	if err := w.channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	deliveries, err := w.channel.Consume(
		queue.Name,
		"",    // consumer tag
		false, // autoAck - manual ack
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go w.processDeliveries(ctx, deliveries)
	return nil
}

// processDeliveries processes incoming task messages
func (w *RabbitMQWorker) processDeliveries(ctx context.Context, deliveries <-chan amqp091.Delivery) {
	for {
		select {
		case <-w.quit:
			return
		case <-ctx.Done():
			return
		case msg, ok := <-deliveries:
			if !ok {
				w.log.Warn().Msg("Delivery channel closed")
				return
			}

			if err := w.handleMessage(ctx, msg); err != nil {
				w.log.Error().Err(err).Msg("Failed to handle task")
				
				// Check retry count from headers
				retryCount := 0
				if val, ok := msg.Headers["retry_count"]; ok {
					if count, ok := val.(int32); ok {
						retryCount = int(count)
					}
				}

				if retryCount >= w.maxRetries {
					w.log.Error().Int("retries", retryCount).Msg("Max retries exceeded, discarding task")
					msg.Nack(false, false) // Dead letter
				} else {
					// Retry with backoff
					retryCount++
					msg.Headers["retry_count"] = int32(retryCount)
					msg.Headers["retry_at"] = time.Now().Add(time.Duration(retryCount) * time.Second).Format(time.RFC3339)
					msg.Nack(false, true) // Requeue
				}
			} else {
				msg.Ack(false)
			}
		}
	}
}

// handleMessage processes a single task message
func (w *RabbitMQWorker) handleMessage(ctx context.Context, msg amqp091.Delivery) error {
	var task Task
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	handler, ok := w.handlers[task.Type]
	if !ok {
		return fmt.Errorf("no handler for task type: %s", task.Type)
	}

	w.log.Info().
		Str("task_id", task.ID).
		Str("task_type", task.Type).
		Int("retry", task.Retries).
		Msg("Processing task")

	if err := handler(ctx, task); err != nil {
		return fmt.Errorf("handler failed: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the worker
func (w *RabbitMQWorker) Stop() {
	w.log.Info().Msg("Stopping RabbitMQ task worker")
	close(w.quit)
}

// EnqueueTask publishes a task to the queue
func EnqueueTask(channel *amqp091.Channel, queueName string, taskType string, payload map[string]interface{}) error {
	task := Task{
		ID:         generateTaskID(),
		Type:       taskType,
		Payload:    payload,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return channel.PublishWithContext(context.Background(),
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp091.Persistent,
			Headers: amqp091.Table{
				"task_type":   taskType,
				"retry_count": int32(0),
			},
		},
	)
}

func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
