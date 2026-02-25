package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BlockedIPRepository struct {
	db *pgxpool.Pool
}

func NewBlockedIPRepository(db *pgxpool.Pool) *BlockedIPRepository {
	return &BlockedIPRepository{db: db}
}

func (r *BlockedIPRepository) Block(ctx context.Context, ip string, reason string, duration int64) error {
	query := `
		INSERT INTO auth_blocked_ips (ip_address, reason, blocked_at, expires_at)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (ip_address)
		DO UPDATE SET reason = $2, blocked_at = NOW(), expires_at = $3
	`

	expiresAt := time.Now().Add(time.Duration(duration) * time.Second)

	_, err := r.db.Exec(ctx, query, ip, reason, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to block IP: %w", err)
	}

	return nil
}

func (r *BlockedIPRepository) IsBlocked(ctx context.Context, ip string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM auth_blocked_ips
		WHERE ip_address = $1 AND expires_at > NOW()
	`

	var count int
	err := r.db.QueryRow(ctx, query, ip).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check blocked IP: %w", err)
	}

	return count > 0, nil
}

func (r *BlockedIPRepository) Unblock(ctx context.Context, ip string) error {
	query := `DELETE FROM auth_blocked_ips WHERE ip_address = $1`

	_, err := r.db.Exec(ctx, query, ip)
	if err != nil {
		return fmt.Errorf("failed to unblock IP: %w", err)
	}

	return nil
}
