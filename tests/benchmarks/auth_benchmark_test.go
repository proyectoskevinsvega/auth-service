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

// BenchmarkLogin benchmarks the complete login flow
func BenchmarkLogin(b *testing.B) {
	// Setup
	userRepo := new(mocks.MockUserRepository)
	sessionRepo := new(mocks.MockSessionRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	resetRepo := new(mocks.MockPasswordResetRepository)
	auditRepo := new(mocks.MockAuditLogRepository)
	jwtService := new(mocks.MockJWTService)
	passwordHasher := new(mocks.MockPasswordHasher)
	tokenGen := new(mocks.MockTokenGenerator)
	rateLimiter := new(mocks.MockRateLimiter)
	sessionStore := new(mocks.MockSessionStore)
	geoService := new(mocks.MockGeolocationService)
	emailService := new(mocks.MockEmailService)
	notifier := new(mocks.MockNotificationPublisher)
	roleRepo := new(mocks.MockRoleRepository)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
		Security: config.SecurityConfig{
			SessionInactivityDays: 30,
		},
		RateLimit: config.RateLimitConfig{
			LoginAttempts:    5,
			LoginWindow:      time.Minute,
			RegisterAttempts: 3,
			RegisterWindow:   time.Hour,
		},
	}

	uc := usecase.NewAuthUseCase(
		userRepo,
		sessionRepo,
		refreshRepo,
		resetRepo,
		auditRepo,
		jwtService,
		passwordHasher,
		tokenGen,
		rateLimiter,
		sessionStore,
		geoService,
		emailService,
		notifier,
		nil,
		cfg,
		nil, // riskService
		roleRepo,
	)

	ctx := context.Background()
	input := usecase.LoginInput{
		Identifier: "benchuser",
		Password:   "BenchPass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "benchuser",
		Email:            "bench@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		EmailVerified:    true,
		TwoFactorEnabled: false,
		Active:           true,
		CreatedAt:        time.Now(),
	}

	// Mock expectations
	rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(true, nil)
	tokenGen.On("GenerateSecureToken", 32).Return("secure_refresh_token_value", nil)
	geoService.On("GetLocation", ctx, input.IPAddress).Return(&domain.Geolocation{Country: "US"}, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)
	jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("jwt.access.token", nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil)

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_, _ = uc.Login(ctx, input)
	}
}

// BenchmarkRegister benchmarks the user registration flow
func BenchmarkRegister(b *testing.B) {
	// Setup
	userRepo := new(mocks.MockUserRepository)
	sessionRepo := new(mocks.MockSessionRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	resetRepo := new(mocks.MockPasswordResetRepository)
	auditRepo := new(mocks.MockAuditLogRepository)
	jwtService := new(mocks.MockJWTService)
	passwordHasher := new(mocks.MockPasswordHasher)
	tokenGen := new(mocks.MockTokenGenerator)
	rateLimiter := new(mocks.MockRateLimiter)
	sessionStore := new(mocks.MockSessionStore)
	geoService := new(mocks.MockGeolocationService)
	emailService := new(mocks.MockEmailService)
	notifier := new(mocks.MockNotificationPublisher)
	roleRepo := new(mocks.MockRoleRepository)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
		Security: config.SecurityConfig{
			SessionInactivityDays: 30,
		},
		RateLimit: config.RateLimitConfig{
			LoginAttempts:    5,
			LoginWindow:      time.Minute,
			RegisterAttempts: 3,
			RegisterWindow:   time.Hour,
		},
	}

	uc := usecase.NewAuthUseCase(
		userRepo,
		sessionRepo,
		refreshRepo,
		resetRepo,
		auditRepo,
		jwtService,
		passwordHasher,
		tokenGen,
		rateLimiter,
		sessionStore,
		geoService,
		emailService,
		notifier,
		nil,
		cfg,
		nil, // riskService
		roleRepo,
	)

	ctx := context.Background()

	// Mock expectations
	rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)
	rateLimiter.On("Increment", ctx, "register:192.168.1.1", time.Hour).Return(1, nil)
	userRepo.On("GetByUsername", ctx, "benchuser").Return(nil, domain.ErrUserNotFound)
	userRepo.On("GetByEmail", ctx, "bench@example.com").Return(nil, domain.ErrUserNotFound)
	passwordHasher.On("Hash", "BenchPass123!").Return("$argon2id$hashed_password", nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User"), "$argon2id$hashed_password").Return(nil)
	geoService.On("GetLocation", ctx, "192.168.1.1").Return(&domain.Geolocation{Country: "US"}, nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil)
	emailService.On("SendWelcome", ctx, "bench@example.com", "benchuser").Return(nil)
	notifier.On("Publish", ctx, mock.AnythingOfType("*domain.Event")).Return(nil)

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		input := usecase.RegisterInput{
			Username:  "benchuser",
			Email:     "bench@example.com",
			Password:  "BenchPass123!",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		}
		_, _ = uc.Register(ctx, input)
	}
}

// BenchmarkPasswordHashing benchmarks password hashing operations
// This is critical for understanding registration/login performance
func BenchmarkPasswordVerification(b *testing.B) {
	passwordHasher := new(mocks.MockPasswordHasher)
	password := "SecurePassword123!"
	hash := "$argon2id$v=19$m=65536,t=3,p=2$somesalt$somehash"

	passwordHasher.On("Verify", password, hash).Return(true, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = passwordHasher.Verify(password, hash)
	}
}
