package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, entry *domain.AuditLogEntry) error {
	query := `
		INSERT INTO auth_audit_log (
			id, user_id, action, ip_address, user_agent, country, success, error_msg, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		entry.ID, entry.UserID, entry.Action, entry.IPAddress, entry.UserAgent, entry.Country,
		entry.Success, entry.ErrorMsg, metadataJSON, entry.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log entry: %w", err)
	}

	return nil
}

func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.AuditLogEntry, error) {
	query := `
		SELECT id, user_id, action, ip_address, user_agent, country, success, error_msg, metadata, created_at
		FROM auth_audit_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*domain.AuditLogEntry
	for rows.Next() {
		entry := &domain.AuditLogEntry{}
		var metadataJSON []byte

		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Action, &entry.IPAddress, &entry.UserAgent, &entry.Country,
			&entry.Success, &entry.ErrorMsg, &metadataJSON, &entry.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log entry: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &entry.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
