package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type ClientRepository struct {
	db *pgxpool.Pool
}

func NewClientRepository(db *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{db: db}
}

func (r *ClientRepository) Create(ctx context.Context, client *domain.Client) error {
	query := `
		INSERT INTO auth_clients (
			id, tenant_id, client_id, client_secret_hash, name, active, scopes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		client.ID, client.TenantID, client.ClientID, client.ClientSecretHash,
		client.Name, client.Active, client.Scopes, client.CreatedAt, client.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return nil
}

func (r *ClientRepository) GetByClientID(ctx context.Context, clientID string) (*domain.Client, error) {
	query := `
		SELECT id, tenant_id, client_id, client_secret_hash, name, active, scopes, created_at, updated_at
		FROM auth_clients
		WHERE client_id = $1
	`

	client := &domain.Client{}
	err := r.db.QueryRow(ctx, query, clientID).Scan(
		&client.ID, &client.TenantID, &client.ClientID, &client.ClientSecretHash,
		&client.Name, &client.Active, &client.Scopes, &client.CreatedAt, &client.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to get client by ID: %w", err)
	}

	return client, nil
}

func (r *ClientRepository) Update(ctx context.Context, client *domain.Client) error {
	query := `
		UPDATE auth_clients
		SET name = $1, active = $2, scopes = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := r.db.Exec(ctx, query,
		client.Name, client.Active, client.Scopes, client.UpdatedAt, client.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}

	return nil
}

func (r *ClientRepository) Delete(ctx context.Context, clientID string) error {
	query := `DELETE FROM auth_clients WHERE client_id = $1`
	_, err := r.db.Exec(ctx, query, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}
	return nil
}
