package usecase

import (
	"context"
	"fmt"

	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type TokenUseCase struct {
	jwtService  ports.JWTService
	tokenCache  ports.TokenCache
	blacklist   ports.TokenBlacklist
	userRepo    ports.UserRepository
	refreshRepo ports.RefreshTokenRepository
	sessionRepo ports.SessionRepository
	notifier    ports.NotificationPublisher
	config      *config.Config
}

func NewTokenUseCase(
	jwtService ports.JWTService,
	tokenCache ports.TokenCache,
	blacklist ports.TokenBlacklist,
	userRepo ports.UserRepository,
	refreshRepo ports.RefreshTokenRepository,
	sessionRepo ports.SessionRepository,
	notifier ports.NotificationPublisher,
	cfg *config.Config,
) *TokenUseCase {
	return &TokenUseCase{
		jwtService:  jwtService,
		tokenCache:  tokenCache,
		blacklist:   blacklist,
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		sessionRepo: sessionRepo,
		notifier:    notifier,
		config:      cfg,
	}
}

func (uc *TokenUseCase) ValidateToken(ctx context.Context, tokenString string) (*domain.Token, error) {
	// Parse token to get JTI
	token, err := uc.jwtService.Verify(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Check blacklist first (fast path)
	isBlacklisted, err := uc.blacklist.IsBlacklisted(ctx, token.JTI)
	if err == nil && isBlacklisted {
		return nil, domain.ErrTokenRevoked
	}

	// Check cache (hot path - <2ms)
	cachedToken, err := uc.tokenCache.Get(ctx, token.JTI)
	if err == nil && cachedToken != nil {
		return cachedToken, nil
	}

	// Cold path: verify cryptographically
	token, err = uc.jwtService.Verify(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Check expiration
	if token.IsExpired() {
		return nil, domain.ErrTokenExpired
	}

	// Cache for next validation
	_ = uc.tokenCache.Set(ctx, token.JTI, token, token.TimeToLive())

	return token, nil
}

func (uc *TokenUseCase) RefreshToken(ctx context.Context, refreshTokenStr string) (*LoginResponse, error) {
	// Hash the refresh token
	refreshTokenHash := hashToken(refreshTokenStr)

	// Get refresh token from database
	refreshToken, err := uc.refreshRepo.GetByTokenHash(ctx, refreshTokenHash)
	if err != nil {
		return nil, domain.ErrRefreshTokenInvalid
	}

	// Check if token is valid
	if !refreshToken.IsValid() {
		// Check if token was rotated (possible theft)
		if refreshToken.Revoked {
			// Revoke all user sessions
			_ = uc.sessionRepo.RevokeAllByUserID(ctx, refreshToken.UserID, "security", "token_theft_detected")
			_ = uc.refreshRepo.RevokeByUserID(ctx, refreshToken.UserID)

			// Get user for event
			user, _ := uc.userRepo.GetByID(ctx, refreshToken.UserID)
			if user != nil {
				event := domain.NewEvent(domain.EventTokenStolen, user.ID, user.Email, map[string]interface{}{
					"session_id": refreshToken.SessionID,
				})
				_ = uc.notifier.Publish(ctx, event)
			}

			return nil, domain.ErrTokenStolen
		}

		return nil, domain.ErrRefreshTokenExpired
	}

	// Get user
	user, err := uc.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if !user.Active {
		return nil, domain.ErrUserInactive
	}

	// Get session
	session, err := uc.sessionRepo.GetByID(ctx, refreshToken.SessionID)
	if err != nil || !session.IsActive() {
		return nil, domain.ErrSessionExpired
	}

	// Generate new JWT
	token := domain.NewToken(user.ID, user.Email, uc.config.JWT.AccessExpiry)

	accessToken, err := uc.jwtService.Generate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Rotate refresh token
	newRefreshTokenStr, _ := (&SecureTokenGenerator{}).GenerateSecureToken(32)
	newRefreshTokenHash := hashToken(newRefreshTokenStr)

	newRefreshToken := refreshToken.Rotate(newRefreshTokenHash)

	// Save both tokens
	_ = uc.refreshRepo.Update(ctx, refreshToken) // Mark old as revoked
	if err := uc.refreshRepo.Create(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenStr,
		User:         user,
	}, nil
}

func (uc *TokenUseCase) RevokeToken(ctx context.Context, tokenString string) error {
	// Parse token
	token, err := uc.jwtService.Verify(ctx, tokenString)
	if err != nil {
		return err
	}

	// Add to blacklist with remaining TTL
	if err := uc.blacklist.Add(ctx, token.JTI, token.TimeToLive()); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	// Remove from cache
	_ = uc.tokenCache.Delete(ctx, token.JTI)

	return nil
}

type SecureTokenGenerator struct{}

func (g *SecureTokenGenerator) GenerateSecureToken(length int) (string, error) {
	// Simplified - use crypto adapter in production
	return "generated_token", nil
}
