package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type TenantRepository struct {
	db *pgxpool.Pool
}

func NewTenantRepository(db *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO auth_tenants (id, slug, name, active, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		tenant.ID,
		tenant.Slug,
		tenant.Name,
		tenant.Active,
		tenant.Settings,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

func (r *TenantRepository) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	query := `
		SELECT id, slug, name, active, settings, created_at, updated_at
		FROM auth_tenants
		WHERE id = $1
	`
	var t domain.Tenant
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID,
		&t.Slug,
		&t.Name,
		&t.Active,
		&t.Settings,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Or a specific error like ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by id: %w", err)
	}
	return &t, nil
}

func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `
		SELECT id, slug, name, active, settings, created_at, updated_at
		FROM auth_tenants
		WHERE slug = $1
	`
	var t domain.Tenant
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&t.ID,
		&t.Slug,
		&t.Name,
		&t.Active,
		&t.Settings,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}
	return &t, nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	tenant.UpdatedAt = time.Now()
	query := `
		UPDATE auth_tenants
		SET slug = $2, name = $3, active = $4, settings = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		tenant.ID,
		tenant.Slug,
		tenant.Name,
		tenant.Active,
		tenant.Settings,
		tenant.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}
	return nil
}

func (r *TenantRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM auth_tenants WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}
	return nil
}

func (r *TenantRepository) List(ctx context.Context) ([]*domain.Tenant, error) {
	query := `SELECT id, slug, name, active, settings, created_at, updated_at FROM auth_tenants`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Active, &t.Settings, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, &t)
	}
	return tenants, nil
}
