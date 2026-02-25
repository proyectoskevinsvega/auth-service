package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type BackupCodeRepository struct {
	db *pgxpool.Pool
}

func NewBackupCodeRepository(db *pgxpool.Pool) *BackupCodeRepository {
	return &BackupCodeRepository{db: db}
}

func (r *BackupCodeRepository) CreateMany(ctx context.Context, codes []*domain.BackupCode) error {
	if len(codes) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO auth_2fa_backup_codes (
			id, tenant_id, user_id, code_hash, used, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, code := range codes {
		batch.Queue(query, code.ID, code.TenantID, code.UserID, code.CodeHash, code.Used, code.CreatedAt)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range codes {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch insert: %w", err)
		}
	}

	return nil
}

func (r *BackupCodeRepository) GetActiveByUserID(ctx context.Context, tenantID, userID string) ([]*domain.BackupCode, error) {
	query := `
		SELECT id, tenant_id, user_id, code_hash, used, used_at, created_at
		FROM auth_2fa_backup_codes
		WHERE tenant_id = $1 AND user_id = $2 AND used = false
	`

	rows, err := r.db.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active backup codes: %w", err)
	}
	defer rows.Close()

	var codes []*domain.BackupCode
	for rows.Next() {
		code := &domain.BackupCode{}
		err := rows.Scan(
			&code.ID, &code.TenantID, &code.UserID, &code.CodeHash, &code.Used, &code.UsedAt, &code.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backup code: %w", err)
		}
		codes = append(codes, code)
	}

	return codes, nil
}

func (r *BackupCodeRepository) MarkAsUsed(ctx context.Context, id string) error {
	query := `
		UPDATE auth_2fa_backup_codes
		SET used = true, used_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to mark backup code as used: %w", err)
	}

	return nil
}

func (r *BackupCodeRepository) DeleteAllByUserID(ctx context.Context, tenantID, userID string) error {
	query := `
		DELETE FROM auth_2fa_backup_codes
		WHERE tenant_id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete backup codes: %w", err)
	}

	return nil
}
