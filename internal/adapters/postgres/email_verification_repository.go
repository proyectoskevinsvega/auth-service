package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type EmailVerificationRepository struct {
	db *pgxpool.Pool
}

func NewEmailVerificationRepository(db *pgxpool.Pool) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

func (r *EmailVerificationRepository) Create(ctx context.Context, verification *domain.EmailVerification) error {
	query := `
		INSERT INTO auth_email_verifications (id, tenant_id, user_id, token_hash, expires_at, created_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(ctx, query,
		verification.ID, verification.TenantID, verification.UserID, verification.TokenHash, verification.ExpiresAt,
		verification.CreatedAt, verification.IPAddress, verification.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to create email verification: %w", err)
	}

	return nil
}

func (r *EmailVerificationRepository) GetByTokenHash(ctx context.Context, tenantID, tokenHash string) (*domain.EmailVerification, error) {
	query := `
		SELECT id, tenant_id, user_id, token_hash, expires_at, verified_at, created_at, ip_address, user_agent
		FROM auth_email_verifications
		WHERE tenant_id = $1 AND token_hash = $2
		LIMIT 1
	`

	var verification domain.EmailVerification
	var verifiedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, tenantID, tokenHash).Scan(
		&verification.ID, &verification.TenantID, &verification.UserID, &verification.TokenHash, &verification.ExpiresAt,
		&verifiedAt, &verification.CreatedAt, &verification.IPAddress, &verification.UserAgent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrVerificationTokenNotFound
		}
		return nil, fmt.Errorf("failed to get email verification: %w", err)
	}

	if verifiedAt.Valid {
		verification.VerifiedAt = &verifiedAt.Time
	}

	return &verification, nil
}

func (r *EmailVerificationRepository) GetByUserID(ctx context.Context, tenantID, userID string) ([]*domain.EmailVerification, error) {
	query := `
		SELECT id, tenant_id, user_id, token_hash, expires_at, verified_at, created_at, ip_address, user_agent
		FROM auth_email_verifications
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 10
	`

	rows, err := r.db.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get verifications: %w", err)
	}
	defer rows.Close()

	var verifications []*domain.EmailVerification
	for rows.Next() {
		var verification domain.EmailVerification
		var verifiedAt sql.NullTime

		err := rows.Scan(
			&verification.ID, &verification.TenantID, &verification.UserID, &verification.TokenHash, &verification.ExpiresAt,
			&verifiedAt, &verification.CreatedAt, &verification.IPAddress, &verification.UserAgent,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan verification: %w", err)
		}

		if verifiedAt.Valid {
			verification.VerifiedAt = &verifiedAt.Time
		}

		verifications = append(verifications, &verification)
	}

	return verifications, nil
}

func (r *EmailVerificationRepository) MarkAsVerified(ctx context.Context, tenantID, tokenHash string) error {
	query := `
		UPDATE auth_email_verifications
		SET verified_at = NOW()
		WHERE tenant_id = $1 AND token_hash = $2 AND verified_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, tenantID, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to mark verification as verified: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVerificationTokenNotFound
	}

	return nil
}

func (r *EmailVerificationRepository) DeleteByUserID(ctx context.Context, tenantID, userID string) error {
	query := `DELETE FROM auth_email_verifications WHERE tenant_id = $1 AND user_id = $2`

	_, err := r.db.Exec(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete verifications: %w", err)
	}

	return nil
}

func (r *EmailVerificationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth_email_verifications WHERE expires_at < NOW() AND verified_at IS NULL`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired verifications: %w", err)
	}

	return result.RowsAffected(), nil
}
