package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

type TaskProcessor interface {
	Start() error
	ProcessTaskDeliverWebhook(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	server     *asynq.Server
	httpClient *http.Client
	logger     zerolog.Logger
}

func NewRedisTaskProcessor(server *asynq.Server, logger zerolog.Logger) TaskProcessor {
	return &RedisTaskProcessor{
		server: server,
		httpClient: &http.Client{
			Timeout: 15 * time.Second, // Max time to wait for webhook delivery
		},
		logger: logger,
	}
}

func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskDeliverWebhook, processor.ProcessTaskDeliverWebhook)

	return processor.server.Start(mux)
}

func (processor *RedisTaskProcessor) ProcessTaskDeliverWebhook(ctx context.Context, task *asynq.Task) error {
	var payload DeliverWebhookPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %w", asynq.SkipRetry) // Don't retry unmarshal errors
	}

	sub := payload.Subscription
	eventInfo := payload.EventPayload

	processor.logger.Info().
		Str("webhook_id", sub.ID).
		Str("url", sub.URL).
		Str("event_type", string(eventInfo.Type)).
		Msg("processing webhook delivery task")

	// 1. Serialize the event payload
	bodyBytes, err := json.Marshal(eventInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", asynq.SkipRetry)
	}

	// 2. Prepare HTTP Request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", asynq.SkipRetry)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Auth-Service-Webhook/1.0")
	req.Header.Set("X-Auth-Event", string(eventInfo.Type))

	// 3. Compute HMAC Signature if Secret is provided
	if sub.Secret != "" {
		mac := hmac.New(sha256.New, []byte(sub.Secret))
		mac.Write(bodyBytes)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Auth-Signature", "sha256="+signature)
	}

	// 4. Dispatch the HTTP Request
	resp, err := processor.httpClient.Do(req)
	if err != nil {
		processor.logger.Warn().Err(err).Str("url", sub.URL).Msg("webhook request failed, backing off")
		return fmt.Errorf("webhook request failed: %w", err) // asynq will automatically retry with backoff
	}
	defer resp.Body.Close()

	// 5. Check response status codes
	if resp.StatusCode >= 400 {
		processor.logger.Warn().
			Int("status", resp.StatusCode).
			Str("url", sub.URL).
			Msg("webhook returned error status code, will retry")
		return fmt.Errorf("remote API returned status %d", resp.StatusCode)
	}

	processor.logger.Info().
		Str("webhook_id", sub.ID).
		Str("url", sub.URL).
		Msg("webhook successfully delivered")

	return nil
}
