package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO auth_refresh_tokens (
			id, tenant_id, user_id, session_id, token_hash, previous_token, expires_at, created_at, revoked, revoked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	// Convert empty string to NULL for previous_token (UUID field)
	var previousToken interface{}
	if token.PreviousToken == "" {
		previousToken = nil
	} else {
		previousToken = token.PreviousToken
	}

	_, err := r.db.Exec(ctx, query,
		token.ID, token.TenantID, token.UserID, token.SessionID, token.TokenHash, previousToken,
		token.ExpiresAt, token.CreatedAt, token.Revoked, token.RevokedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, token_hash, previous_token, expires_at, created_at, revoked, revoked_at
		FROM auth_refresh_tokens
		WHERE tenant_id = $1 AND id = $2
	`

	token := &domain.RefreshToken{}
	var revokedAt sql.NullTime
	var previousToken sql.NullString

	err := r.db.QueryRow(ctx, query, tenantID, id).Scan(
		&token.ID, &token.TenantID, &token.UserID, &token.SessionID, &token.TokenHash, &previousToken,
		&token.ExpiresAt, &token.CreatedAt, &token.Revoked, &revokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if previousToken.Valid {
		token.PreviousToken = previousToken.String
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tenantID, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, token_hash, previous_token, expires_at, created_at, revoked, revoked_at
		FROM auth_refresh_tokens
		WHERE tenant_id = $1 AND token_hash = $2
	`

	token := &domain.RefreshToken{}
	var revokedAt sql.NullTime
	var previousToken sql.NullString

	err := r.db.QueryRow(ctx, query, tenantID, tokenHash).Scan(
		&token.ID, &token.TenantID, &token.UserID, &token.SessionID, &token.TokenHash, &previousToken,
		&token.ExpiresAt, &token.CreatedAt, &token.Revoked, &revokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if previousToken.Valid {
		token.PreviousToken = previousToken.String
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

func (r *RefreshTokenRepository) GetBySessionID(ctx context.Context, tenantID, sessionID string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, token_hash, previous_token, expires_at, created_at, revoked, revoked_at
		FROM auth_refresh_tokens
		WHERE tenant_id = $1 AND session_id = $2 AND revoked = false
		ORDER BY created_at DESC
		LIMIT 1
	`

	token := &domain.RefreshToken{}
	var revokedAt sql.NullTime
	var previousToken sql.NullString

	err := r.db.QueryRow(ctx, query, tenantID, sessionID).Scan(
		&token.ID, &token.TenantID, &token.UserID, &token.SessionID, &token.TokenHash, &previousToken,
		&token.ExpiresAt, &token.CreatedAt, &token.Revoked, &revokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if previousToken.Valid {
		token.PreviousToken = previousToken.String
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

func (r *RefreshTokenRepository) Update(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		UPDATE auth_refresh_tokens
		SET revoked = $2, revoked_at = $3
		WHERE tenant_id = $4 AND id = $1
	`

	_, err := r.db.Exec(ctx, query, token.ID, token.Revoked, token.RevokedAt, token.TenantID)
	if err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, tenantID, tokenID string) error {
	query := `
		UPDATE auth_refresh_tokens
		SET revoked = true, revoked_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	_, err := r.db.Exec(ctx, query, tenantID, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, tenantID, userID string) error {
	query := `
		UPDATE auth_refresh_tokens
		SET revoked = true, revoked_at = NOW()
		WHERE tenant_id = $1 AND user_id = $2 AND revoked = false
	`

	_, err := r.db.Exec(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user refresh tokens: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) RevokeBySessionID(ctx context.Context, tenantID, sessionID string) error {
	query := `
		UPDATE auth_refresh_tokens
		SET revoked = true, revoked_at = NOW()
		WHERE tenant_id = $1 AND session_id = $2 AND revoked = false
	`

	_, err := r.db.Exec(ctx, query, tenantID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke session refresh tokens: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM auth_refresh_tokens WHERE expires_at < NOW()`

	_, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}

	return nil
}
