package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type PasswordResetRepository struct {
	db *pgxpool.Pool
}

func NewPasswordResetRepository(db *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	query := `
		INSERT INTO auth_password_resets (id, tenant_id, user_id, token, code, expires_at, created_at, used, used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		token.ID, token.TenantID, token.UserID, token.Token, token.Code, token.ExpiresAt, token.CreatedAt, token.Used, token.UsedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	return nil
}

func (r *PasswordResetRepository) GetByToken(ctx context.Context, tenantID, token string) (*domain.PasswordResetToken, error) {
	query := `
		SELECT id, tenant_id, user_id, token, code, expires_at, created_at, used, used_at
		FROM auth_password_resets
		WHERE tenant_id = $1 AND token = $2
	`

	resetToken := &domain.PasswordResetToken{}
	var usedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, tenantID, token).Scan(
		&resetToken.ID, &resetToken.TenantID, &resetToken.UserID, &resetToken.Token, &resetToken.Code,
		&resetToken.ExpiresAt, &resetToken.CreatedAt, &resetToken.Used, &usedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidResetToken
		}
		return nil, fmt.Errorf("failed to get password reset token: %w", err)
	}

	if usedAt.Valid {
		resetToken.UsedAt = &usedAt.Time
	}

	return resetToken, nil
}

func (r *PasswordResetRepository) GetByCode(ctx context.Context, tenantID, userID, code string) (*domain.PasswordResetToken, error) {
	query := `
		SELECT id, tenant_id, user_id, token, code, expires_at, created_at, used, used_at
		FROM auth_password_resets
		WHERE tenant_id = $1 AND user_id = $2 AND code = $3 AND used = false
		ORDER BY created_at DESC
		LIMIT 1
	`

	resetToken := &domain.PasswordResetToken{}
	var usedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, tenantID, userID, code).Scan(
		&resetToken.ID, &resetToken.TenantID, &resetToken.UserID, &resetToken.Token, &resetToken.Code,
		&resetToken.ExpiresAt, &resetToken.CreatedAt, &resetToken.Used, &usedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidResetToken
		}
		return nil, fmt.Errorf("failed to get password reset token by code: %w", err)
	}

	if usedAt.Valid {
		resetToken.UsedAt = &usedAt.Time
	}

	return resetToken, nil
}

func (r *PasswordResetRepository) MarkAsUsed(ctx context.Context, tenantID, tokenID string) error {
	query := `
		UPDATE auth_password_resets
		SET used = true, used_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	_, err := r.db.Exec(ctx, query, tenantID, tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark password reset token as used: %w", err)
	}

	return nil
}

func (r *PasswordResetRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM auth_password_resets WHERE expires_at < NOW()`

	_, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired password reset tokens: %w", err)
	}

	return nil
}
