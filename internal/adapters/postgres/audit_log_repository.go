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
			id, tenant_id, user_id, action, ip_address, user_agent, country, success, error_msg, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		entry.ID, entry.TenantID, entry.UserID, entry.Action, entry.IPAddress, entry.UserAgent, entry.Country,
		entry.Success, entry.ErrorMsg, metadataJSON, entry.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log entry: %w", err)
	}

	return nil
}

func (r *AuditLogRepository) GetByUserID(ctx context.Context, tenantID, userID string, limit, offset int) ([]*domain.AuditLogEntry, error) {
	query := `
		SELECT id, tenant_id, user_id, action, ip_address, user_agent, country, success, error_msg, metadata, created_at
		FROM auth_audit_log
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(ctx, query, tenantID, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*domain.AuditLogEntry
	for rows.Next() {
		entry := &domain.AuditLogEntry{}
		var metadataJSON []byte

		err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.UserID, &entry.Action, &entry.IPAddress, &entry.UserAgent, &entry.Country,
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

func (r *AuditLogRepository) Search(ctx context.Context, filter domain.AuditSearchFilter) ([]*domain.AuditLogEntry, int, error) {
	baseQuery := `SELECT id, tenant_id, user_id, action, ip_address, user_agent, country, success, error_msg, metadata, created_at FROM auth_audit_log WHERE tenant_id = $1`
	countQuery := `SELECT COUNT(*) FROM auth_audit_log WHERE tenant_id = $1`

	args := []interface{}{filter.TenantID}
	where := ""
	placeholderID := 2

	if filter.UserID != "" {
		where += fmt.Sprintf(" AND user_id = $%d", placeholderID)
		args = append(args, filter.UserID)
		placeholderID++
	}
	if filter.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", placeholderID)
		args = append(args, filter.Action)
		placeholderID++
	}
	if filter.StartDate != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", placeholderID)
		args = append(args, *filter.StartDate)
		placeholderID++
	}
	if filter.EndDate != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", placeholderID)
		args = append(args, *filter.EndDate)
		placeholderID++
	}
	if filter.Success != nil {
		where += fmt.Sprintf(" AND success = $%d", placeholderID)
		args = append(args, *filter.Success)
		placeholderID++
	}

	// Get Total Count
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery+where, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get Paged Results
	limitOffset := fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", placeholderID, placeholderID+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, baseQuery+where+limitOffset, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*domain.AuditLogEntry
	for rows.Next() {
		entry := &domain.AuditLogEntry{}
		var metadataJSON []byte

		err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.UserID, &entry.Action, &entry.IPAddress, &entry.UserAgent, &entry.Country,
			&entry.Success, &entry.ErrorMsg, &metadataJSON, &entry.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log entry: %w", err)
		}

		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &entry.Metadata)
		}
		entries = append(entries, entry)
	}

	return entries, totalCount, nil
}
