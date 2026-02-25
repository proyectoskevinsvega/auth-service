package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/tests/mocks"
)

type TokenUseCaseMocks struct {
	uc          *TokenUseCase
	jwtService  *mocks.MockJWTService
	tokenCache  *mocks.MockTokenCache
	blacklist   *mocks.MockTokenBlacklist
	userRepo    *mocks.MockUserRepository
	refreshRepo *mocks.MockRefreshTokenRepository
	sessionRepo *mocks.MockSessionRepository
	notifier    *mocks.MockNotificationPublisher
}

func setupTokenUseCase(_ *testing.T) *TokenUseCaseMocks {
	jwtService := new(mocks.MockJWTService)
	tokenCache := new(mocks.MockTokenCache)
	blacklist := new(mocks.MockTokenBlacklist)
	userRepo := new(mocks.MockUserRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	sessionRepo := new(mocks.MockSessionRepository)
	notifier := new(mocks.MockNotificationPublisher)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
	}

	uc := NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		refreshRepo,
		sessionRepo,
		notifier,
		cfg,
	)

	return &TokenUseCaseMocks{
		uc:          uc,
		jwtService:  jwtService,
		tokenCache:  tokenCache,
		blacklist:   blacklist,
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		sessionRepo: sessionRepo,
		notifier:    notifier,
	}
}

func TestTokenUseCase_ValidateToken_Success_CacheHit(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	tokenString := "valid.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	// Token from JWT verification
	now := time.Now()
	parsedToken := &domain.Token{
		JTI:       jti,
		UserID:    userID,
		Email:     "test@example.com",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock expectations - cache hit path
	m.jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil).Once()
	m.blacklist.On("IsBlacklisted", ctx, jti).Return(false, nil)
	m.tokenCache.On("Get", ctx, jti).Return(parsedToken, nil)

	// Execute
	token, err := m.uc.ValidateToken(ctx, tokenString)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, jti, token.JTI)
	assert.Equal(t, userID, token.UserID)

	m.jwtService.AssertExpectations(t)
	m.blacklist.AssertExpectations(t)
	m.tokenCache.AssertExpectations(t)
}

func TestTokenUseCase_ValidateToken_Success_CacheMiss(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	tokenString := "valid.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	// Token from JWT verification
	now := time.Now()
	parsedToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "test@example.com",

		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock expectations - cache miss, needs full verification
	m.jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil).Twice() // Called twice
	m.blacklist.On("IsBlacklisted", ctx, jti).Return(false, nil)
	m.tokenCache.On("Get", ctx, jti).Return(nil, domain.ErrTokenNotFound)
	m.tokenCache.On("Set", ctx, jti, parsedToken, mock.AnythingOfType("time.Duration")).Return(nil)

	// Execute
	token, err := m.uc.ValidateToken(ctx, tokenString)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, jti, token.JTI)
	assert.Equal(t, userID, token.UserID)

	m.jwtService.AssertExpectations(t)
	m.blacklist.AssertExpectations(t)
	m.tokenCache.AssertExpectations(t)
}

func TestTokenUseCase_ValidateToken_TokenBlacklisted(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	tokenString := "blacklisted.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	now := time.Now()
	parsedToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "test@example.com",

		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock expectations - token is blacklisted
	m.jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil).Once()
	m.blacklist.On("IsBlacklisted", ctx, jti).Return(true, nil)

	// Execute
	token, err := m.uc.ValidateToken(ctx, tokenString)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, domain.ErrTokenRevoked, err)

	m.jwtService.AssertExpectations(t)
	m.blacklist.AssertExpectations(t)
}

func TestTokenUseCase_ValidateToken_InvalidToken(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	tokenString := "invalid.jwt.token"

	// Mock expectations - JWT verification fails
	m.jwtService.On("Verify", ctx, tokenString).Return(nil, domain.ErrInvalidToken)

	// Execute
	token, err := m.uc.ValidateToken(ctx, tokenString)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, domain.ErrInvalidToken, err)

	m.jwtService.AssertExpectations(t)
}

func TestTokenUseCase_ValidateToken_ExpiredToken(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	tokenString := "expired.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	// Expired token
	now := time.Now()
	expiredToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "test@example.com",

		IssuedAt:  now.Add(-2 * time.Hour).Unix(),
		ExpiresAt: now.Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}

	// Mock expectations
	m.jwtService.On("Verify", ctx, tokenString).Return(expiredToken, nil).Twice()
	m.blacklist.On("IsBlacklisted", ctx, jti).Return(false, nil)
	m.tokenCache.On("Get", ctx, jti).Return(nil, domain.ErrTokenNotFound)

	// Execute
	token, err := m.uc.ValidateToken(ctx, tokenString)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, domain.ErrTokenExpired, err)

	m.jwtService.AssertExpectations(t)
	m.blacklist.AssertExpectations(t)
	m.tokenCache.AssertExpectations(t)
}

func TestTokenUseCase_RevokeToken_Success(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	tokenString := "valid.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	now := time.Now()
	parsedToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "test@example.com",

		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock expectations
	m.jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil)
	m.blacklist.On("Add", ctx, jti, mock.AnythingOfType("time.Duration")).Return(nil)
	m.tokenCache.On("Delete", ctx, jti).Return(nil)

	// Execute
	err := m.uc.RevokeToken(ctx, tokenString)

	// Assert
	assert.NoError(t, err)

	m.jwtService.AssertExpectations(t)
	m.blacklist.AssertExpectations(t)
	m.tokenCache.AssertExpectations(t)
}

func TestTokenUseCase_RevokeToken_InvalidToken(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	tokenString := "invalid.jwt.token"

	// Mock expectations - JWT verification fails
	m.jwtService.On("Verify", ctx, tokenString).Return(nil, domain.ErrInvalidToken)

	// Execute
	err := m.uc.RevokeToken(ctx, tokenString)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidToken, err)

	m.jwtService.AssertExpectations(t)
}

// TestRefreshToken_Success prueba la renovación exitosa de tokens
func TestRefreshToken_Success(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "refresh_token_12345"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_refresh_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:     userID,
		Email:  "test@example.com",
		Active: true,
	}

	session := &domain.Session{
		ID:        sessionID,
		UserID:    userID,
		JTI:       uuid.New().String(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	m.sessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("new_access_token", nil)
	m.refreshRepo.On("Update", ctx, refreshToken).Return(nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "new_access_token", response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, userID, response.User.ID)
	m.refreshRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
}

// TestRefreshToken_InvalidToken prueba error con refresh token inválido
func TestRefreshToken_InvalidToken(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	refreshTokenStr := "invalid_refresh_token"

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(nil, domain.ErrRefreshTokenInvalid)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrRefreshTokenInvalid, err)

	m.refreshRepo.AssertExpectations(t)
}

// TestRefreshToken_TokenStolen prueba detección de robo de token
func TestRefreshToken_TokenStolen(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "stolen_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   true, // Token ya revocado (posible robo)
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:    userID,
		Email: "test@example.com",

		Active: true,
	}

	// Mock expectations - detecta robo y revoca todo
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.sessionRepo.On("RevokeAllByUserID", ctx, userID, "security", "token_theft_detected").Return(nil)
	m.refreshRepo.On("RevokeByUserID", ctx, userID).Return(nil)
	m.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	m.notifier.On("Publish", ctx, mock.AnythingOfType("*domain.Event")).Return(nil)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrTokenStolen, err)

	m.refreshRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.notifier.AssertExpectations(t)
}

// TestRefreshToken_TokenExpired prueba error con token expirado
func TestRefreshToken_TokenExpired(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "expired_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		Revoked:   false,
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrRefreshTokenExpired, err)

	m.refreshRepo.AssertExpectations(t)
}

// TestRefreshToken_UserNotFound prueba error cuando el usuario no existe
func TestRefreshToken_UserNotFound(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "valid_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.userRepo.On("GetByID", ctx, userID).Return(nil, domain.ErrUserNotFound)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrUserNotFound, err)

	m.refreshRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestRefreshToken_UserInactive prueba error cuando el usuario está inactivo
func TestRefreshToken_UserInactive(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "valid_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:    userID,
		Email: "test@example.com",

		Active: false, // User is inactive
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.userRepo.On("GetByID", ctx, userID).Return(user, nil)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrUserInactive, err)
	m.refreshRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestRefreshToken_SessionExpired prueba error cuando la sesión ha expirado
func TestRefreshToken_SessionExpired(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "valid_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:    userID,
		Email: "test@example.com",

		Active: true,
	}

	session := &domain.Session{
		ID:        sessionID,
		UserID:    userID,
		JTI:       uuid.New().String(),
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		Revoked:   false,
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	m.sessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrSessionExpired, err)

	m.refreshRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
}

// TestRefreshToken_JWTGenerationFailure prueba error al generar JWT
func TestRefreshToken_JWTGenerationFailure(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "valid_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:    userID,
		Email: "test@example.com",

		Active: true,
	}

	session := &domain.Session{
		ID:        sessionID,
		UserID:    userID,
		JTI:       uuid.New().String(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	m.sessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("", assert.AnError)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to generate JWT")

	m.refreshRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
}

// TestRefreshToken_RotationFailure prueba error al rotar refresh token
func TestRefreshToken_RotationFailure(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "valid_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:    userID,
		Email: "test@example.com",

		Active: true,
	}

	session := &domain.Session{
		ID:        sessionID,
		UserID:    userID,
		JTI:       uuid.New().String(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	m.sessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("new_access_token", nil)
	m.refreshRepo.On("Update", ctx, refreshToken).Return(nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(assert.AnError)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to rotate refresh token")

	m.refreshRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
}

// TestRefreshToken_SessionNotFound prueba error cuando no se encuentra la sesión
func TestRefreshToken_SessionNotFound(t *testing.T) {
	m := setupTokenUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	refreshTokenStr := "valid_refresh_token"

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: "hashed_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:    userID,
		Email: "test@example.com",

		Active: true,
	}

	// Mock expectations
	m.refreshRepo.On("GetByTokenHash", ctx, mock.AnythingOfType("string")).Return(refreshToken, nil)
	m.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	m.sessionRepo.On("GetByID", ctx, sessionID).Return(nil, domain.ErrSessionNotFound)

	// Execute
	response, err := m.uc.RefreshToken(ctx, refreshTokenStr)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrSessionExpired, err)

	m.refreshRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
}
