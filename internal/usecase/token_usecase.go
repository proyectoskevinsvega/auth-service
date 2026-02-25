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
	riskService ports.RiskService
	tenantRepo  ports.TenantRepository
	clientRepo  ports.ClientRepository
	hasher      ports.PasswordHasher
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
	riskService ports.RiskService,
	tenantRepo ports.TenantRepository,
	clientRepo ports.ClientRepository,
	hasher ports.PasswordHasher,
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
		riskService: riskService,
		tenantRepo:  tenantRepo,
		clientRepo:  clientRepo,
		hasher:      hasher,
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

	// Session Geofencing Check (P2)
	// We need the country associated with the current request.
	// In a real production scenario, the country would be passed in the context
	// or determined here via GeoIP if the IP is available in the context.
	// For this implementation, we'll try to get it if possible.
	ip, _ := ctx.Value("client_ip").(string)
	if ip != "" {
		tenant, err := uc.tenantRepo.GetByID(ctx, token.TenantID)
		if err == nil && tenant != nil && tenant.Settings.EnforceSessionGeofencing {
			// Get current location
			_, loc, err := uc.riskService.AssessLoginRisk(ctx, &domain.User{
				ID:       token.UserID,
				TenantID: token.TenantID,
			}, ip)

			if err == nil && loc != nil {
				// Get session to compare country
				session, err := uc.sessionRepo.GetByID(ctx, token.TenantID, token.JTI)
				if err == nil && session != nil {
					if session.Country != "" && loc.Country != session.Country {
						// Significant geographic shift in active session
						_ = uc.sessionRepo.Revoke(ctx, token.TenantID, session.ID, "security", "session_hijacking_suspected_geofence")
						return nil, domain.ErrSessionHijackingSuspected
					}
				}
			}
		}
	}

	// Cache for next validation
	_ = uc.tokenCache.Set(ctx, token.JTI, token, token.TimeToLive())

	return token, nil
}

func (uc *TokenUseCase) RefreshToken(ctx context.Context, tenantID, refreshTokenStr string) (*LoginResponse, error) {
	// Hash the refresh token
	refreshTokenHash := hashToken(refreshTokenStr)

	// Get refresh token from database
	refreshToken, err := uc.refreshRepo.GetByTokenHash(ctx, tenantID, refreshTokenHash)
	if err != nil {
		return nil, domain.ErrRefreshTokenInvalid
	}

	// Check if token is valid
	if !refreshToken.IsValid() {
		// Check if token was rotated (possible theft)
		if refreshToken.Revoked {
			// Revoke all user sessions
			_ = uc.sessionRepo.RevokeAllByUserID(ctx, refreshToken.TenantID, refreshToken.UserID, "security", "token_theft_detected")
			_ = uc.refreshRepo.RevokeByUserID(ctx, refreshToken.TenantID, refreshToken.UserID)

			// Get user for event
			user, _ := uc.userRepo.GetByID(ctx, refreshToken.TenantID, refreshToken.UserID)
			if user != nil {
				event := domain.NewEvent(user.TenantID, domain.EventTokenStolen, user.ID, user.Email, map[string]interface{}{
					"session_id": refreshToken.SessionID,
				})
				_ = uc.notifier.Publish(ctx, event)
			}

			return nil, domain.ErrTokenStolen
		}

		return nil, domain.ErrRefreshTokenExpired
	}

	// Get user
	user, err := uc.userRepo.GetByID(ctx, refreshToken.TenantID, refreshToken.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if !user.Active {
		return nil, domain.ErrUserInactive
	}

	// Get session
	session, err := uc.sessionRepo.GetByID(ctx, refreshToken.TenantID, refreshToken.SessionID)
	if err != nil || !session.IsActive() {
		return nil, domain.ErrSessionExpired
	}

	// Generate new JWT
	token := domain.NewToken(user.TenantID, user.ID, user.Email, uc.config.JWT.AccessExpiry)

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

func (uc *TokenUseCase) IssueClientToken(ctx context.Context, clientID, clientSecret string) (*LoginResponse, error) {
	// Get client
	client, err := uc.clientRepo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, domain.ErrInvalidClient
	}

	// Verify secret
	isValid, err := uc.hasher.Verify(clientSecret, client.ClientSecretHash)
	if err != nil || !isValid {
		return nil, domain.ErrInvalidClient
	}

	if !client.Active {
		return nil, domain.ErrInvalidClient
	}

	// Generate JWT for client
	// For M2M, UserID is the ClientID, and we use a special email or just ClientID
	token := domain.NewToken(client.TenantID, client.ClientID, client.ClientID+"@m2m.local", uc.config.JWT.AccessExpiry)
	token.Scopes = client.Scopes

	accessToken, err := uc.jwtService.Generate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client JWT: %w", err)
	}

	// For Client Credentials, we usually don't issue Refresh Tokens
	// but we return the same response structure for consistency
	return &LoginResponse{
		AccessToken: accessToken,
		User: &domain.User{
			ID:       client.ClientID,
			TenantID: client.TenantID,
			Username: client.Name,
			Active:   true,
		},
	}, nil
}

type SecureTokenGenerator struct{}

func (g *SecureTokenGenerator) GenerateSecureToken(length int) (string, error) {
	// Simplified - use crypto adapter in production
	return "generated_token", nil
}
