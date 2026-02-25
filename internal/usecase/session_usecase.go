package usecase

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type SessionUseCase struct {
	sessionRepo ports.SessionRepository
	userRepo    ports.UserRepository
	logger      zerolog.Logger
}

func NewSessionUseCase(
	sessionRepo ports.SessionRepository,
	userRepo ports.UserRepository,
	logger zerolog.Logger,
) *SessionUseCase {
	return &SessionUseCase{
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// ListUserSessions returns all active sessions for a user
func (uc *SessionUseCase) ListUserSessions(ctx context.Context, userID, currentJTI string) ([]*domain.Session, error) {
	sessions, err := uc.sessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list user sessions")
		return nil, err
	}

	// Mark current session
	for _, session := range sessions {
		if session.JTI == currentJTI {
			session.IsCurrent = true
		}
	}

	return sessions, nil
}

// RevokeSession revokes a specific session
func (uc *SessionUseCase) RevokeSession(ctx context.Context, userID, sessionID string) error {
	// Verify session belongs to user
	session, err := uc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if err == domain.ErrSessionNotFound {
			return domain.ErrSessionNotFound
		}
		uc.logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get session")
		return err
	}

	if session.UserID != userID {
		uc.logger.Warn().
			Str("session_id", sessionID).
			Str("user_id", userID).
			Str("session_user_id", session.UserID).
			Msg("user attempted to revoke session that doesn't belong to them")
		return domain.ErrSessionNotFound
	}

	// Revoke session
	if err := uc.sessionRepo.Revoke(ctx, sessionID, "user", "revoked by user"); err != nil {
		uc.logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to revoke session")
		return err
	}

	uc.logger.Info().
		Str("user_id", userID).
		Str("session_id", sessionID).
		Msg("session revoked")

	return nil
}

// RevokeAllSessions revokes all sessions except the current one
func (uc *SessionUseCase) RevokeAllSessions(ctx context.Context, userID, currentJTI string) (int, error) {
	sessions, err := uc.sessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get user sessions")
		return 0, err
	}

	revokedCount := 0
	for _, session := range sessions {
		// Skip current session
		if session.JTI == currentJTI {
			continue
		}

		if err := uc.sessionRepo.Revoke(ctx, session.ID, "user", "revoked all sessions"); err != nil {
			uc.logger.Error().
				Err(err).
				Str("session_id", session.ID).
				Msg("failed to revoke session")
			continue
		}

		revokedCount++
	}

	uc.logger.Info().
		Str("user_id", userID).
		Int("revoked_count", revokedCount).
		Msg("revoked all other sessions")

	return revokedCount, nil
}

// CleanupExpiredSessions removes expired sessions (can be called periodically)
func (uc *SessionUseCase) CleanupExpiredSessions(ctx context.Context) error {
	if err := uc.sessionRepo.DeleteExpired(ctx); err != nil {
		uc.logger.Error().Err(err).Msg("failed to cleanup expired sessions")
		return err
	}

	uc.logger.Info().Msg("cleaned up expired sessions")
	return nil
}
