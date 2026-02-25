package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	query := `
		INSERT INTO auth_sessions (
			id, user_id, ip_address, country, device, user_agent,
			created_at, last_used_at, expires_at, revoked, revoked_at, revoked_by, revoke_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Exec(ctx, query,
		session.ID, session.UserID, session.IPAddress, session.Country, session.Device, session.UserAgent,
		session.CreatedAt, session.LastUsedAt, session.ExpiresAt, session.Revoked, session.RevokedAt,
		session.RevokedBy, session.RevokeReason,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *SessionRepository) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, ip_address, country, device, user_agent,
			   created_at, last_used_at, expires_at, revoked, revoked_at, revoked_by, revoke_reason
		FROM auth_sessions
		WHERE id = $1
	`

	session := &domain.Session{}
	var revokedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID, &session.UserID, &session.IPAddress, &session.Country, &session.Device, &session.UserAgent,
		&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt, &session.Revoked, &revokedAt,
		&session.RevokedBy, &session.RevokeReason,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}

	return session, nil
}

func (r *SessionRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	query := `
		SELECT id, user_id, ip_address, country, device, user_agent,
			   created_at, last_used_at, expires_at, revoked, revoked_at, revoked_by, revoke_reason
		FROM auth_sessions
		WHERE user_id = $1 AND revoked = false AND expires_at > NOW()
		ORDER BY last_used_at DESC
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		session := &domain.Session{}
		var revokedAt sql.NullTime

		err := rows.Scan(
			&session.ID, &session.UserID, &session.IPAddress, &session.Country, &session.Device, &session.UserAgent,
			&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt, &session.Revoked, &revokedAt,
			&session.RevokedBy, &session.RevokeReason,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if revokedAt.Valid {
			session.RevokedAt = &revokedAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (r *SessionRepository) GetRecentByUserID(ctx context.Context, userID string, limit int) ([]*domain.Session, error) {
	query := `
		SELECT id, user_id, ip_address, country, device, user_agent,
			   created_at, last_used_at, expires_at, revoked, revoked_at, revoked_by, revoke_reason
		FROM auth_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		session := &domain.Session{}
		var revokedAt sql.NullTime

		err := rows.Scan(
			&session.ID, &session.UserID, &session.IPAddress, &session.Country, &session.Device, &session.UserAgent,
			&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt, &session.Revoked, &revokedAt,
			&session.RevokedBy, &session.RevokeReason,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if revokedAt.Valid {
			session.RevokedAt = &revokedAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (r *SessionRepository) Update(ctx context.Context, session *domain.Session) error {
	query := `
		UPDATE auth_sessions
		SET last_used_at = $2, expires_at = $3, revoked = $4, revoked_at = $5, revoked_by = $6, revoke_reason = $7
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		session.ID, session.LastUsedAt, session.ExpiresAt, session.Revoked, session.RevokedAt,
		session.RevokedBy, session.RevokeReason,
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID string, revokedBy, reason string) error {
	query := `
		UPDATE auth_sessions
		SET revoked = true, revoked_at = NOW(), revoked_by = $2, revoke_reason = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, sessionID, revokedBy, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

func (r *SessionRepository) RevokeAllByUserID(ctx context.Context, userID, revokedBy, reason string) error {
	query := `
		UPDATE auth_sessions
		SET revoked = true, revoked_at = NOW(), revoked_by = $2, revoke_reason = $3
		WHERE user_id = $1 AND revoked = false
	`

	_, err := r.db.Exec(ctx, query, userID, revokedBy, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke all sessions: %w", err)
	}

	return nil
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM auth_sessions WHERE expires_at < NOW()`

	_, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}
