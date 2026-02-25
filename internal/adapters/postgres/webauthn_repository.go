package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type WebAuthnRepository struct {
	db *pgxpool.Pool
}

func NewWebAuthnRepository(db *pgxpool.Pool) *WebAuthnRepository {
	return &WebAuthnRepository{db: db}
}

func (r *WebAuthnRepository) GetCredentialByID(ctx context.Context, tenantID string, credentialID []byte) (*domain.WebAuthnCredential, error) {
	query := `
		SELECT id, tenant_id, user_id, public_key, attestation_type, aaguid, sign_count, clone_warning, created_at, updated_at
		FROM auth_webauthn_credentials
		WHERE tenant_id = $1 AND id = $2
	`

	cred := &domain.WebAuthnCredential{}
	err := r.db.QueryRow(ctx, query, tenantID, credentialID).Scan(
		&cred.ID, &cred.TenantID, &cred.UserID, &cred.PublicKey, &cred.AttestationType, &cred.AAGUID,
		&cred.SignCount, &cred.CloneWarning, &cred.CreatedAt, &cred.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidToken // O un error más específico para WebAuthn
		}
		return nil, fmt.Errorf("failed to get webauthn credential: %w", err)
	}

	return cred, nil
}

func (r *WebAuthnRepository) GetCredentialsByUserID(ctx context.Context, tenantID, userID string) ([]*domain.WebAuthnCredential, error) {
	query := `
		SELECT id, tenant_id, user_id, public_key, attestation_type, aaguid, sign_count, clone_warning, created_at, updated_at
		FROM auth_webauthn_credentials
		WHERE tenant_id = $1 AND user_id = $2
	`

	rows, err := r.db.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webauthn credentials: %w", err)
	}
	defer rows.Close()

	var credentials []*domain.WebAuthnCredential
	for rows.Next() {
		cred := &domain.WebAuthnCredential{}
		err := rows.Scan(
			&cred.ID, &cred.TenantID, &cred.UserID, &cred.PublicKey, &cred.AttestationType, &cred.AAGUID,
			&cred.SignCount, &cred.CloneWarning, &cred.CreatedAt, &cred.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webauthn credential: %w", err)
		}
		credentials = append(credentials, cred)
	}

	return credentials, nil
}

func (r *WebAuthnRepository) CreateCredential(ctx context.Context, tenantID string, cred *domain.WebAuthnCredential) error {
	query := `
		INSERT INTO auth_webauthn_credentials (
			id, tenant_id, user_id, public_key, attestation_type, aaguid, sign_count, clone_warning
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(ctx, query,
		cred.ID, cred.TenantID, cred.UserID, cred.PublicKey, cred.AttestationType, cred.AAGUID,
		cred.SignCount, cred.CloneWarning,
	)

	if err != nil {
		return fmt.Errorf("failed to create webauthn credential: %w", err)
	}

	return nil
}

func (r *WebAuthnRepository) UpdateCredential(ctx context.Context, tenantID string, cred *domain.WebAuthnCredential) error {
	query := `
		UPDATE auth_webauthn_credentials
		SET sign_count = $3, clone_warning = $4, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	_, err := r.db.Exec(ctx, query, tenantID, cred.ID, cred.SignCount, cred.CloneWarning)
	if err != nil {
		return fmt.Errorf("failed to update webauthn credential: %w", err)
	}

	return nil
}

func (r *WebAuthnRepository) DeleteCredential(ctx context.Context, tenantID string, credentialID []byte) error {
	query := `DELETE FROM auth_webauthn_credentials WHERE tenant_id = $1 AND id = $2`

	_, err := r.db.Exec(ctx, query, tenantID, credentialID)
	if err != nil {
		return fmt.Errorf("failed to delete webauthn credential: %w", err)
	}

	return nil
}
