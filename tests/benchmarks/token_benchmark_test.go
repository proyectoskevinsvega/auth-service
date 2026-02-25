package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/usecase"
	"github.com/vertercloud/auth-service/tests/mocks"
)

// BenchmarkTokenValidation benchmarks the token validation flow
// Target: <5ms p99 latency
func BenchmarkTokenValidation_CacheHit(b *testing.B) {
	// Setup
	jwtService := new(mocks.MockJWTService)
	tokenCache := new(mocks.MockTokenCache)
	blacklist := new(mocks.MockTokenBlacklist)
	userRepo := new(mocks.MockUserRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	sessionRepo := new(mocks.MockSessionRepository)
	riskService := new(mocks.MockRiskService)
	tenantRepo := new(mocks.MockTenantRepository)
	notifier := new(mocks.MockNotificationPublisher)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
	}

	clientRepo := new(mocks.MockClientRepository)
	passwordHasher := new(mocks.MockPasswordHasher)

	uc := usecase.NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		refreshRepo,
		sessionRepo,
		riskService,
		tenantRepo,
		clientRepo,
		passwordHasher,
		notifier,
		cfg,
	)

	ctx := context.Background()
	tokenString := "valid.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	now := time.Now()
	parsedToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "bench@example.com",

		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock setup for cache hit (hot path)
	jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil)
	blacklist.On("IsBlacklisted", ctx, jti).Return(false, nil)
	tokenCache.On("Get", ctx, jti).Return(parsedToken, nil)

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_, _ = uc.ValidateToken(ctx, tokenString)
	}
}

func BenchmarkTokenValidation_CacheMiss(b *testing.B) {
	// Setup
	jwtService := new(mocks.MockJWTService)
	tokenCache := new(mocks.MockTokenCache)
	blacklist := new(mocks.MockTokenBlacklist)
	userRepo := new(mocks.MockUserRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	sessionRepo := new(mocks.MockSessionRepository)
	riskService := new(mocks.MockRiskService)
	tenantRepo := new(mocks.MockTenantRepository)
	notifier := new(mocks.MockNotificationPublisher)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
	}

	clientRepo := new(mocks.MockClientRepository)
	passwordHasher := new(mocks.MockPasswordHasher)

	uc := usecase.NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		refreshRepo,
		sessionRepo,
		riskService,
		tenantRepo,
		clientRepo,
		passwordHasher,
		notifier,
		cfg,
	)

	ctx := context.Background()
	tokenString := "valid.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	now := time.Now()
	parsedToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "bench@example.com",

		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock setup for cache miss (cold path)
	jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil)
	blacklist.On("IsBlacklisted", ctx, jti).Return(false, nil)
	tokenCache.On("Get", ctx, jti).Return(nil, domain.ErrTokenNotFound)
	tokenCache.On("Set", ctx, jti, parsedToken, mock.AnythingOfType("time.Duration")).Return(nil)

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_, _ = uc.ValidateToken(ctx, tokenString)
	}
}

func BenchmarkTokenValidation_Blacklist(b *testing.B) {
	// Setup
	jwtService := new(mocks.MockJWTService)
	tokenCache := new(mocks.MockTokenCache)
	blacklist := new(mocks.MockTokenBlacklist)
	userRepo := new(mocks.MockUserRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	sessionRepo := new(mocks.MockSessionRepository)
	riskService := new(mocks.MockRiskService)
	tenantRepo := new(mocks.MockTenantRepository)
	notifier := new(mocks.MockNotificationPublisher)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
	}

	clientRepo := new(mocks.MockClientRepository)
	passwordHasher := new(mocks.MockPasswordHasher)

	uc := usecase.NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		refreshRepo,
		sessionRepo,
		riskService,
		tenantRepo,
		clientRepo,
		passwordHasher,
		notifier,
		cfg,
	)

	ctx := context.Background()
	tokenString := "blacklisted.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	now := time.Now()
	parsedToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "bench@example.com",

		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock setup for blacklisted token
	jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil)
	blacklist.On("IsBlacklisted", ctx, jti).Return(true, nil)

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_, _ = uc.ValidateToken(ctx, tokenString)
	}
}

// BenchmarkTokenRevocation benchmarks the token revocation operation
func BenchmarkTokenRevocation(b *testing.B) {
	// Setup
	jwtService := new(mocks.MockJWTService)
	tokenCache := new(mocks.MockTokenCache)
	blacklist := new(mocks.MockTokenBlacklist)
	userRepo := new(mocks.MockUserRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	sessionRepo := new(mocks.MockSessionRepository)
	riskService := new(mocks.MockRiskService)
	tenantRepo := new(mocks.MockTenantRepository)
	notifier := new(mocks.MockNotificationPublisher)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
	}

	clientRepo := new(mocks.MockClientRepository)
	passwordHasher := new(mocks.MockPasswordHasher)

	uc := usecase.NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		refreshRepo,
		sessionRepo,
		riskService,
		tenantRepo,
		clientRepo,
		passwordHasher,
		notifier,
		cfg,
	)

	ctx := context.Background()
	tokenString := "valid.jwt.token"
	jti := uuid.New().String()
	userID := uuid.New().String()

	now := time.Now()
	parsedToken := &domain.Token{
		JTI:    jti,
		UserID: userID,
		Email:  "bench@example.com",

		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}

	// Mock setup
	jwtService.On("Verify", ctx, tokenString).Return(parsedToken, nil)
	blacklist.On("Add", ctx, jti, mock.AnythingOfType("time.Duration")).Return(nil)
	tokenCache.On("Delete", ctx, jti).Return(nil)

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_ = uc.RevokeToken(ctx, tokenString)
	}
}
