package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type WebhookRepository struct {
	db *pgxpool.Pool
}

func NewWebhookRepository(db *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) Create(ctx context.Context, sub *domain.WebhookSubscription) error {
	query := `
		INSERT INTO auth_webhook_subscriptions (tenant_id, url, secret, event_types, active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		sub.TenantID,
		sub.URL,
		sub.Secret,
		sub.EventTypes,
		sub.Active,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create webhook subscription: %w", err)
	}
	return nil
}

func (r *WebhookRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.WebhookSubscription, error) {
	query := `
		SELECT id, tenant_id, url, secret, event_types, active, created_at, updated_at
		FROM auth_webhook_subscriptions
		WHERE tenant_id = $1 AND id = $2
	`
	sub := &domain.WebhookSubscription{}
	err := r.db.QueryRow(ctx, query, tenantID, id).Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.URL,
		&sub.Secret,
		&sub.EventTypes,
		&sub.Active,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get webhook subscription: %w", err)
	}
	return sub, nil
}

func (r *WebhookRepository) GetByTenantID(ctx context.Context, tenantID string) ([]*domain.WebhookSubscription, error) {
	query := `
		SELECT id, tenant_id, url, secret, event_types, active, created_at, updated_at
		FROM auth_webhook_subscriptions
		WHERE tenant_id = $1
	`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhook subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*domain.WebhookSubscription
	for rows.Next() {
		sub := &domain.WebhookSubscription{}
		err := rows.Scan(
			&sub.ID,
			&sub.TenantID,
			&sub.URL,
			&sub.Secret,
			&sub.EventTypes,
			&sub.Active,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (r *WebhookRepository) GetSubscriptionsForEvent(ctx context.Context, tenantID string, eventType domain.EventType) ([]*domain.WebhookSubscription, error) {
	query := `
		SELECT id, tenant_id, url, secret, event_types, active, created_at, updated_at
		FROM auth_webhook_subscriptions
		WHERE tenant_id = $1 AND active = TRUE AND $2 = ANY(event_types)
	`
	rows, err := r.db.Query(ctx, query, tenantID, string(eventType))
	if err != nil {
		return nil, fmt.Errorf("failed to search webhook subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*domain.WebhookSubscription
	for rows.Next() {
		sub := &domain.WebhookSubscription{}
		err := rows.Scan(
			&sub.ID,
			&sub.TenantID,
			&sub.URL,
			&sub.Secret,
			&sub.EventTypes,
			&sub.Active,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (r *WebhookRepository) Update(ctx context.Context, sub *domain.WebhookSubscription) error {
	query := `
		UPDATE auth_webhook_subscriptions
		SET url = $1, secret = $2, event_types = $3, active = $4, updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $5 AND id = $6
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		sub.URL,
		sub.Secret,
		sub.EventTypes,
		sub.Active,
		sub.TenantID,
		sub.ID,
	).Scan(&sub.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update webhook subscription: %w", err)
	}
	return nil
}

func (r *WebhookRepository) Delete(ctx context.Context, tenantID, id string) error {
	query := `DELETE FROM auth_webhook_subscriptions WHERE tenant_id = $1 AND id = $2`
	result, err := r.db.Exec(ctx, query, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook subscription not found")
	}
	return nil
}
