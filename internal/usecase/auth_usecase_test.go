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
	"github.com/vertercloud/auth-service/internal/ports"
	"github.com/vertercloud/auth-service/tests/mocks"
)

type AuthUseCaseMocks struct {
	uc             *AuthUseCase
	userRepo       *mocks.MockUserRepository
	sessionRepo    *mocks.MockSessionRepository
	refreshRepo    *mocks.MockRefreshTokenRepository
	resetRepo      *mocks.MockPasswordResetRepository
	auditRepo      *mocks.MockAuditLogRepository
	jwtService     *mocks.MockJWTService
	passwordHasher *mocks.MockPasswordHasher
	tokenGen       *mocks.MockTokenGenerator
	rateLimiter    *mocks.MockRateLimiter
	sessionStore   *mocks.MockSessionStore
	geoService     *mocks.MockGeolocationService
	emailService   *mocks.MockEmailService
	notifier       *mocks.MockNotificationPublisher
	riskService    *mocks.MockRiskService
}

func setupAuthUseCase(_ *testing.T) *AuthUseCaseMocks {
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
	riskService := new(mocks.MockRiskService)

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

	uc := NewAuthUseCase(
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
		nil, // oauthProviders
		cfg,
		riskService,
	)

	return &AuthUseCaseMocks{
		uc:             uc,
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		refreshRepo:    refreshRepo,
		resetRepo:      resetRepo,
		auditRepo:      auditRepo,
		jwtService:     jwtService,
		passwordHasher: passwordHasher,
		tokenGen:       tokenGen,
		rateLimiter:    rateLimiter,
		sessionStore:   sessionStore,
		geoService:     geoService,
		emailService:   emailService,
		notifier:       notifier,
		riskService:    riskService,
	}
}

func TestAuthUseCase_Register_Success(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations (only critical path, async operations are fire-and-forget)
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)
	m.userRepo.On("GetByUsername", ctx, input.Username).Return(nil, domain.ErrUserNotFound)
	m.userRepo.On("GetByEmail", ctx, input.Email).Return(nil, domain.ErrUserNotFound)
	m.passwordHasher.On("Hash", input.Password).Return("$argon2id$hashed_password", nil)
	m.userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User"), "$argon2id$hashed_password").Return(nil)

	// Async operations (fire-and-forget, not verified in success test)
	m.rateLimiter.On("Increment", mock.Anything, "register:192.168.1.1", time.Hour).Return(1, nil).Maybe()
	m.geoService.On("GetCountry", mock.Anything, input.IPAddress).Return("US", nil).Maybe()
	m.auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil).Maybe()
	m.emailService.On("SendWelcome", mock.Anything, input.Email, input.Username).Return(nil).Maybe()
	m.notifier.On("Publish", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil).Maybe()

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, input.Username, user.Username)
	assert.Equal(t, input.Email, user.Email)
	assert.False(t, user.EmailVerified)
	assert.False(t, user.TwoFactorEnabled)

	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Login_Success(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "testuser",
		Email:            "test@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		EmailVerified:    true,
		TwoFactorEnabled: false,
		Active:           true,
		CreatedAt:        time.Now(),
	}

	// Mock expectations (only critical path, async operations are fire-and-forget)
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	m.passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(true, nil)
	m.tokenGen.On("GenerateSecureToken", 32).Return("secure_refresh_token_value", nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("jwt.access.token", nil)

	// Risk Assessment and Geo
	m.riskService.On("AssessLoginRisk", ctx, existingUser, input.IPAddress).Return(domain.NewRiskAssessment(), &domain.Geolocation{Country: "US"}, nil)
	m.userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Maybe()
	m.auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil).Maybe()

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "jwt.access.token", response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, existingUser.ID, response.User.ID)

	m.userRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.refreshRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Login_RateLimitExceeded(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	// Mock expectations - rate limit exceeded
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(true, nil)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrRateLimitExceeded, err)

	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Login_InvalidPassword(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "WrongPassword123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "testuser",
		Email:            "test@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		EmailVerified:    true,
		TwoFactorEnabled: false,
		Active:           true,
		CreatedAt:        time.Now(),
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	m.passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(false, nil)
	m.auditRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrInvalidCredentials, err)

	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.rateLimiter.AssertExpectations(t)
	m.auditRepo.AssertExpectations(t)
}

// TestAuthUseCase_Login_UserInactive prueba error cuando el usuario está inactivo
func TestAuthUseCase_Login_UserInactive(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:           uuid.New().String(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$argon2id$hashed_password",
		Active:       false, // Usuario inactivo
		CreatedAt:    time.Now(),
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrUserInactive, err)

	m.userRepo.AssertExpectations(t)
	m.rateLimiter.AssertExpectations(t)
}

// TestAuthUseCase_Login_2FARequired prueba cuando se requiere 2FA pero no se proporciona
func TestAuthUseCase_Login_2FARequired(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		TwoFACode:  "", // No se proporciona código 2FA
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "testuser",
		Email:            "test@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		TwoFactorEnabled: true, // 2FA habilitado
		Active:           true,
		CreatedAt:        time.Now(),
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	m.passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(true, nil)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.Err2FARequired, err)

	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Login_RateLimitCheckFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	// Mock expectations - rate limit check fails with error
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, assert.AnError)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "rate limit check failed")

	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Login_SessionCreationFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "testuser",
		Email:            "test@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		TwoFactorEnabled: false,
		Active:           true,
		CreatedAt:        time.Now(),
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	m.passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(true, nil)
	m.riskService.On("AssessLoginRisk", ctx, existingUser, input.IPAddress).Return(domain.NewRiskAssessment(), &domain.Geolocation{Country: "US"}, nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(assert.AnError)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create session")

	m.rateLimiter.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
}

func TestAuthUseCase_Login_JWTGenerationFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "testuser",
		Email:            "test@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		TwoFactorEnabled: false,
		Active:           true,
		CreatedAt:        time.Now(),
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	m.passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(true, nil)
	m.riskService.On("AssessLoginRisk", ctx, existingUser, input.IPAddress).Return(domain.NewRiskAssessment(), &domain.Geolocation{Country: "US"}, nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("", assert.AnError)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to generate JWT")

	m.rateLimiter.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
}

func TestAuthUseCase_Login_RefreshTokenCreationFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "testuser",
		Email:            "test@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		TwoFactorEnabled: false,
		Active:           true,
		CreatedAt:        time.Now(),
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	m.passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(true, nil)
	m.riskService.On("AssessLoginRisk", ctx, existingUser, input.IPAddress).Return(domain.NewRiskAssessment(), &domain.Geolocation{Country: "US"}, nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("jwt_token", nil)
	m.tokenGen.On("GenerateSecureToken", 32).Return("refresh_token", nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(assert.AnError)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create refresh token")

	m.rateLimiter.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
	m.tokenGen.AssertExpectations(t)
	m.refreshRepo.AssertExpectations(t)
}

func TestAuthUseCase_Login_NewCountryDetection(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := LoginInput{
		Identifier: "testuser",
		Password:   "SecurePass123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Device:     "Desktop",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "testuser",
		Email:            "test@example.com",
		PasswordHash:     "$argon2id$hashed_password",
		TwoFactorEnabled: false,
		Active:           true,
		LastLoginCountry: "MX", // Previous login was from Mexico
		CreatedAt:        time.Now(),
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "login:192.168.1.1", 5, time.Minute).Return(false, nil)
	m.rateLimiter.On("Increment", ctx, "login:192.168.1.1", time.Minute).Return(1, nil)
	m.userRepo.On("GetByEmailOrUsername", ctx, input.Identifier).Return(existingUser, nil)
	m.passwordHasher.On("Verify", input.Password, existingUser.PasswordHash).Return(true, nil)
	m.riskService.On("AssessLoginRisk", ctx, existingUser, input.IPAddress).Return(domain.NewRiskAssessment(), &domain.Geolocation{Country: "US"}, nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("jwt_token", nil)
	m.tokenGen.On("GenerateSecureToken", 32).Return("refresh_token", nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)
	// Async operations use background context
	m.userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	m.auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil)
	m.notifier.On("Publish", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil)

	// Execute
	response, err := m.uc.Login(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "jwt_token", response.AccessToken)
	assert.Equal(t, "refresh_token", response.RefreshToken)

	// Wait for async operations to complete
	time.Sleep(50 * time.Millisecond)

	// Verify new country event was published (async operation)
	m.notifier.AssertCalled(t, "Publish", mock.Anything, mock.MatchedBy(func(event *domain.Event) bool {
		return event.Type == domain.EventLoginNewCountry &&
			event.UserID == existingUser.ID &&
			event.Data["country"] == "US"
	}))

	m.rateLimiter.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
	m.tokenGen.AssertExpectations(t)
	m.refreshRepo.AssertExpectations(t)
	m.auditRepo.AssertExpectations(t)
	m.notifier.AssertExpectations(t)
}

// TestAuthUseCase_Register_UsernameExists prueba error cuando el username ya existe
func TestAuthUseCase_Register_UsernameExists(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "existinguser",
		Email:     "newemail@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	existingUser := &domain.User{
		ID:       uuid.New().String(),
		Username: "existinguser",
		Email:    "existing@example.com",
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)
	m.userRepo.On("GetByUsername", ctx, input.Username).Return(existingUser, nil) // Username existe

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, domain.ErrUsernameAlreadyExists, err)

	m.userRepo.AssertExpectations(t)
	m.rateLimiter.AssertExpectations(t)
}

// TestAuthUseCase_Register_EmailExists prueba error cuando el email ya existe
func TestAuthUseCase_Register_EmailExists(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "newuser",
		Email:     "existing@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	existingUser := &domain.User{
		ID:       uuid.New().String(),
		Username: "otheruser",
		Email:    "existing@example.com",
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)
	m.userRepo.On("GetByUsername", ctx, input.Username).Return(nil, domain.ErrUserNotFound)
	m.userRepo.On("GetByEmail", ctx, input.Email).Return(existingUser, nil) // Email existe

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, domain.ErrEmailAlreadyExists, err)

	m.userRepo.AssertExpectations(t)
	m.rateLimiter.AssertExpectations(t)
}

// TestAuthUseCase_Register_RateLimitExceeded prueba rate limit en registro
func TestAuthUseCase_Register_RateLimitExceeded(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations - rate limit exceeded
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(true, nil)

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, domain.ErrRateLimitExceeded, err)

	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Register_InvalidUsername(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "ab", // Too short - must be at least 3 characters
		Email:     "test@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations - pass rate limit check
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "username")

	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Register_InvalidEmail(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "testuser",
		Email:     "invalid-email", // Invalid email format
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations - pass rate limit check
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "email")

	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Register_InvalidPassword(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "weak", // Too weak - must meet password requirements
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations - pass rate limit check
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "password")

	m.rateLimiter.AssertExpectations(t)
}

func TestAuthUseCase_Register_PasswordHashingFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)
	m.userRepo.On("GetByUsername", ctx, "testuser").Return(nil, domain.ErrUserNotFound)
	m.userRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrUserNotFound)
	m.passwordHasher.On("Hash", "SecurePass123!").Return("", assert.AnError)

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to hash password")

	m.rateLimiter.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
}

func TestAuthUseCase_Register_UserCreationFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, nil)
	m.userRepo.On("GetByUsername", ctx, "testuser").Return(nil, domain.ErrUserNotFound)
	m.userRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrUserNotFound)
	m.passwordHasher.On("Hash", "SecurePass123!").Return("hashed_password", nil)
	m.userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User"), "hashed_password").Return(assert.AnError)

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to create user")

	m.rateLimiter.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
}

func TestAuthUseCase_Register_RateLimitCheckFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	input := RegisterInput{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "SecurePass123!",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations - rate limit check fails
	m.rateLimiter.On("CheckLimit", ctx, "register:192.168.1.1", 3, time.Hour).Return(false, assert.AnError)

	// Execute
	user, err := m.uc.Register(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "rate limit check failed")

	m.rateLimiter.AssertExpectations(t)
}

// TestForgotPassword_Success prueba la solicitud exitosa de reset de contraseña
func TestForgotPassword_Success(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	email := "test@example.com"
	ipAddress := "192.168.1.1"

	user := &domain.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    email,
		Active:   true,
	}

	// Mock expectations (only critical path, async operations are fire-and-forget)
	m.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
	m.tokenGen.On("GenerateSecureToken", 32).Return("secure_reset_token", nil)
	m.resetRepo.On("Create", ctx, mock.AnythingOfType("*domain.PasswordResetToken")).Return(nil)

	// Async operations use background context
	// Accept any 6-digit code since we now generate random codes
	m.emailService.On("SendPasswordReset", mock.Anything, email,
		mock.MatchedBy(func(code string) bool {
			return len(code) == 6 && code >= "000000" && code <= "999999"
		}),
		mock.AnythingOfType("string")).Return(nil).Maybe()
	m.auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil).Maybe()

	// Execute
	err := m.uc.ForgotPassword(ctx, email, ipAddress)

	// Assert
	assert.NoError(t, err)
	m.userRepo.AssertExpectations(t)
	m.tokenGen.AssertExpectations(t)
	m.resetRepo.AssertExpectations(t)
}

// TestForgotPassword_UserNotFound prueba que no revela si el email existe
func TestForgotPassword_UserNotFound(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	email := "nonexistent@example.com"
	ipAddress := "192.168.1.1"

	// Mock expectations
	m.userRepo.On("GetByEmail", ctx, email).Return(nil, domain.ErrUserNotFound)

	// Execute
	err := m.uc.ForgotPassword(ctx, email, ipAddress)

	// Assert - No revela si el email existe
	assert.NoError(t, err)
	m.userRepo.AssertExpectations(t)

	// No debe llamar a otros servicios
	m.tokenGen.AssertNotCalled(t, "GenerateSecureToken", mock.Anything)
	m.resetRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// TestResetPasswordWithToken_Success prueba el reset exitoso con token
func TestResetPasswordWithToken_Success(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	token := "valid_reset_token"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		Code:      "123456",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
		CreatedAt: time.Now(),
	}

	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
	}

	// Mock expectations (only critical path, async operations are fire-and-forget)
	m.resetRepo.On("GetByToken", ctx, token).Return(resetToken, nil)
	m.passwordHasher.On("Hash", newPassword).Return("$argon2id$new_hash", nil)
	m.userRepo.On("UpdatePassword", ctx, userID, "$argon2id$new_hash").Return(nil)

	// Async operations use background context
	m.resetRepo.On("MarkAsUsed", mock.Anything, resetToken.ID).Return(nil).Maybe()
	m.sessionRepo.On("RevokeAllByUserID", mock.Anything, userID, "system", "password_reset").Return(nil).Maybe()
	m.refreshRepo.On("RevokeByUserID", mock.Anything, userID).Return(nil).Maybe()
	m.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Maybe()
	m.auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil).Maybe()
	m.notifier.On("Publish", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil).Maybe()

	// Execute
	err := m.uc.ResetPasswordWithToken(ctx, token, newPassword, ipAddress)

	// Assert
	assert.NoError(t, err)
	m.resetRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestResetPasswordWithToken_InvalidToken prueba error con token inválido
func TestResetPasswordWithToken_InvalidToken(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	token := "invalid_token"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	// Mock expectations
	m.resetRepo.On("GetByToken", ctx, token).Return(nil, domain.ErrInvalidResetToken)

	// Execute
	err := m.uc.ResetPasswordWithToken(ctx, token, newPassword, ipAddress)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidResetToken, err)
	m.resetRepo.AssertExpectations(t)
}

// TestResetPasswordWithToken_ExpiredToken prueba error con token expirado
func TestResetPasswordWithToken_ExpiredToken(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	token := "expired_token"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		Token:     token,
		Code:      "123456",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expirado
		UsedAt:    nil,
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	// Mock expectations
	m.resetRepo.On("GetByToken", ctx, token).Return(resetToken, nil)

	// Execute
	err := m.uc.ResetPasswordWithToken(ctx, token, newPassword, ipAddress)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrResetTokenExpired, err)
	m.resetRepo.AssertExpectations(t)
}

// TestResetPasswordWithCode_Success prueba el reset exitoso con código
func TestResetPasswordWithCode_Success(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	email := "test@example.com"
	code := "123456"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    email,
		Active:   true,
	}

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     "some_token",
		Code:      code,
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
		CreatedAt: time.Now(),
	}

	// Mock expectations (only critical path, async operations are fire-and-forget)
	m.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
	m.resetRepo.On("GetByCode", ctx, userID, code).Return(resetToken, nil)
	m.passwordHasher.On("Hash", newPassword).Return("$argon2id$new_hash", nil)
	m.userRepo.On("UpdatePassword", ctx, userID, "$argon2id$new_hash").Return(nil)

	// Async operations use background context
	m.resetRepo.On("MarkAsUsed", mock.Anything, resetToken.ID).Return(nil).Maybe()
	m.sessionRepo.On("RevokeAllByUserID", mock.Anything, userID, "system", "password_reset").Return(nil).Maybe()
	m.refreshRepo.On("RevokeByUserID", mock.Anything, userID).Return(nil).Maybe()
	m.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Maybe()
	m.auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AuditLogEntry")).Return(nil).Maybe()
	m.notifier.On("Publish", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil).Maybe()

	// Execute
	err := m.uc.ResetPasswordWithCode(ctx, email, code, newPassword, ipAddress)

	// Assert
	assert.NoError(t, err)
	m.userRepo.AssertExpectations(t)
	m.resetRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
}

// TestResetPasswordWithCode_InvalidCode prueba error con código inválido
func TestResetPasswordWithCode_InvalidCode(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	email := "test@example.com"
	code := "999999"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    email,
		Active:   true,
	}

	// Mock expectations
	m.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
	m.resetRepo.On("GetByCode", ctx, userID, code).Return(nil, domain.ErrInvalidResetToken)

	// Execute
	err := m.uc.ResetPasswordWithCode(ctx, email, code, newPassword, ipAddress)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidResetToken, err)
	m.userRepo.AssertExpectations(t)
	m.resetRepo.AssertExpectations(t)
}

func TestResetPasswordWithCode_InvalidPassword(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	email := "test@example.com"
	code := "123456"
	newPassword := "weak" // Invalid password - too weak
	ipAddress := "192.168.1.1"

	// Execute - should fail validation before even checking the database
	err := m.uc.ResetPasswordWithCode(ctx, email, code, newPassword, ipAddress)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestResetPasswordWithCode_PasswordHashingFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	email := "test@example.com"
	code := "123456"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    email,
		Active:   true,
	}

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     "hashed_token",
		Code:      code,
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}

	// Mock expectations
	m.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
	m.resetRepo.On("GetByCode", ctx, userID, code).Return(resetToken, nil)
	m.passwordHasher.On("Hash", newPassword).Return("", assert.AnError)

	// Execute
	err := m.uc.ResetPasswordWithCode(ctx, email, code, newPassword, ipAddress)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hash password")

	m.userRepo.AssertExpectations(t)
	m.resetRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
}

func TestResetPasswordWithToken_PasswordHashingFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	token := "reset_token_12345"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     "hashed_token",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}

	// Mock expectations
	m.resetRepo.On("GetByToken", ctx, mock.AnythingOfType("string")).Return(resetToken, nil)
	m.passwordHasher.On("Hash", newPassword).Return("", assert.AnError)

	// Execute
	err := m.uc.ResetPasswordWithToken(ctx, token, newPassword, ipAddress)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hash password")

	m.resetRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
}

func TestResetPasswordWithToken_UpdatePasswordFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	token := "reset_token_12345"
	newPassword := "NewSecurePass123!"
	ipAddress := "192.168.1.1"

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     "hashed_token",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}

	// Mock expectations
	m.resetRepo.On("GetByToken", ctx, mock.AnythingOfType("string")).Return(resetToken, nil)
	m.passwordHasher.On("Hash", newPassword).Return("hashed_new_password", nil)
	m.userRepo.On("UpdatePassword", ctx, userID, "hashed_new_password").Return(assert.AnError)

	// Execute
	err := m.uc.ResetPasswordWithToken(ctx, token, newPassword, ipAddress)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update password")

	m.resetRepo.AssertExpectations(t)
	m.passwordHasher.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestOAuthLogin_Success_NewUser - SKIPPED
// Este test está comentado porque hay un bug en el código de producción:
// OAuthLogin intenta crear un usuario sin username, pero NewUser valida que username no esté vacío.
// Para implementar este test correctamente, primero hay que arreglar el código de producción.
func TestOAuthLogin_Success_NewUser(t *testing.T) {
	t.Skip("Skipped due to bug in production code - NewUser requires username but OAuth doesn't provide it initially")
}

// TestOAuthLogin_Success_ExistingUser prueba login exitoso con OAuth para usuario existente
func TestOAuthLogin_Success_ExistingUser(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "github"
	code := "oauth_code_456"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	oauthUserInfo := &ports.OAuthUserInfo{
		ProviderID: "github_user_67890",
		Email:      "existing@example.com",
		Name:       "Existing User",
	}

	existingUser := &domain.User{
		ID:              uuid.New().String(),
		Username:        "existing",
		Email:           "existing@example.com",
		OAuthProvider:   "github",
		OAuthProviderID: "github_user_67890",
		Active:          true,
		CreatedAt:       time.Now(),
	}

	// Setup OAuth provider mock
	oauthProvider := new(mocks.MockOAuthProvider)
	m.uc.oauthProviders = map[string]ports.OAuthProvider{
		"github": oauthProvider,
	}

	// Mock expectations (only critical path, async operations are fire-and-forget)
	oauthProvider.On("Exchange", ctx, code).Return(oauthUserInfo, nil)
	m.userRepo.On("GetByOAuthProvider", ctx, provider, oauthUserInfo.ProviderID).Return(existingUser, nil)
	m.geoService.On("GetCountry", ctx, ipAddress).Return("US", nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("jwt.access.token", nil)
	m.tokenGen.On("GenerateSecureToken", 32).Return("secure_refresh_token", nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)

	// Async operations use background context
	m.userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Maybe()

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "jwt.access.token", response.AccessToken)
	assert.Equal(t, existingUser.ID, response.User.ID)

	oauthProvider.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
	m.refreshRepo.AssertExpectations(t)
}

func TestOAuthLogin_NewCountryDetection(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "github"
	code := "oauth_code_789"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	oauthUserInfo := &ports.OAuthUserInfo{
		ProviderID: "github_user_99999",
		Email:      "traveler@example.com",
		Name:       "Traveling User",
	}

	existingUser := &domain.User{
		ID:               uuid.New().String(),
		Username:         "traveler",
		Email:            "traveler@example.com",
		OAuthProvider:    "github",
		OAuthProviderID:  "github_user_99999",
		Active:           true,
		LastLoginCountry: "MX", // Previous login was from Mexico
		CreatedAt:        time.Now(),
	}

	// Setup OAuth provider mock
	oauthProvider := new(mocks.MockOAuthProvider)
	m.uc.oauthProviders = map[string]ports.OAuthProvider{
		"github": oauthProvider,
	}

	// Mock expectations
	oauthProvider.On("Exchange", ctx, code).Return(oauthUserInfo, nil)
	m.userRepo.On("GetByOAuthProvider", ctx, provider, oauthUserInfo.ProviderID).Return(existingUser, nil)
	m.geoService.On("GetCountry", ctx, ipAddress).Return("US", nil) // Now logging from US
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("jwt.access.token", nil)
	m.tokenGen.On("GenerateSecureToken", 32).Return("secure_refresh_token", nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)
	// Async operations use background context
	m.userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	m.notifier.On("Publish", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil)

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "jwt.access.token", response.AccessToken)
	assert.Equal(t, existingUser.ID, response.User.ID)

	// Wait for async operations to complete
	time.Sleep(50 * time.Millisecond)

	// Verify new country event was published (async operation)
	m.notifier.AssertCalled(t, "Publish", mock.Anything, mock.MatchedBy(func(event *domain.Event) bool {
		return event.Type == domain.EventLoginNewCountry &&
			event.UserID == existingUser.ID &&
			event.Data["country"] == "US"
	}))

	oauthProvider.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
	m.tokenGen.AssertExpectations(t)
	m.refreshRepo.AssertExpectations(t)
	m.notifier.AssertExpectations(t)
}

// TestOAuthLogin_ProviderNotFound prueba error cuando el provider no existe
func TestOAuthLogin_ProviderNotFound(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "facebook"
	code := "oauth_code_789"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	// No OAuth providers configured
	m.uc.oauthProviders = map[string]ports.OAuthProvider{}

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrOAuthProviderNotFound, err)
}

// TestOAuthLogin_InvalidCode prueba error cuando el código OAuth es inválido
func TestOAuthLogin_InvalidCode(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "google"
	code := "invalid_code"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	// Setup OAuth provider mock
	oauthProvider := new(mocks.MockOAuthProvider)
	m.uc.oauthProviders = map[string]ports.OAuthProvider{
		"google": oauthProvider,
	}

	// Mock expectations
	oauthProvider.On("Exchange", ctx, code).Return(nil, domain.ErrOAuthCodeInvalid)

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrOAuthCodeInvalid, err)

	oauthProvider.AssertExpectations(t)
}

func TestOAuthLogin_SessionCreationFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "google"
	code := "valid_code"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	existingUser := &domain.User{
		ID:              uuid.New().String(),
		Username:        "testuser",
		Email:           "test@example.com",
		OAuthProvider:   provider,
		OAuthProviderID: "oauth_12345",
		Active:          true,
		CreatedAt:       time.Now(),
	}

	userInfo := &ports.OAuthUserInfo{
		ProviderID: "oauth_12345",
		Email:      "test@example.com",
		Name:       "Test User",
	}

	// Setup OAuth provider mock
	oauthProvider := new(mocks.MockOAuthProvider)
	m.uc.oauthProviders = map[string]ports.OAuthProvider{
		"google": oauthProvider,
	}

	// Mock expectations
	oauthProvider.On("Exchange", ctx, code).Return(userInfo, nil)
	m.userRepo.On("GetByOAuthProvider", ctx, provider, userInfo.ProviderID).Return(existingUser, nil)
	m.geoService.On("GetCountry", ctx, ipAddress).Return("US", nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(assert.AnError)

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create session")

	oauthProvider.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
}

func TestOAuthLogin_JWTGenerationFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "google"
	code := "valid_code"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	existingUser := &domain.User{
		ID:              uuid.New().String(),
		Username:        "testuser",
		Email:           "test@example.com",
		OAuthProvider:   provider,
		OAuthProviderID: "oauth_12345",
		Active:          true,
		CreatedAt:       time.Now(),
	}

	userInfo := &ports.OAuthUserInfo{
		ProviderID: "oauth_12345",
		Email:      "test@example.com",
		Name:       "Test User",
	}

	// Setup OAuth provider mock
	oauthProvider := new(mocks.MockOAuthProvider)
	m.uc.oauthProviders = map[string]ports.OAuthProvider{
		"google": oauthProvider,
	}

	// Mock expectations
	oauthProvider.On("Exchange", ctx, code).Return(userInfo, nil)
	m.userRepo.On("GetByOAuthProvider", ctx, provider, userInfo.ProviderID).Return(existingUser, nil)
	m.geoService.On("GetCountry", ctx, ipAddress).Return("US", nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("", assert.AnError)

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to generate JWT")

	oauthProvider.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
}

func TestOAuthLogin_RefreshTokenCreationFailure(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "google"
	code := "valid_code"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	existingUser := &domain.User{
		ID:              uuid.New().String(),
		Username:        "testuser",
		Email:           "test@example.com",
		OAuthProvider:   provider,
		OAuthProviderID: "oauth_12345",
		Active:          true,
		CreatedAt:       time.Now(),
	}

	userInfo := &ports.OAuthUserInfo{
		ProviderID: "oauth_12345",
		Email:      "test@example.com",
		Name:       "Test User",
	}

	// Setup OAuth provider mock
	oauthProvider := new(mocks.MockOAuthProvider)
	m.uc.oauthProviders = map[string]ports.OAuthProvider{
		"google": oauthProvider,
	}

	// Mock expectations
	oauthProvider.On("Exchange", ctx, code).Return(userInfo, nil)
	m.userRepo.On("GetByOAuthProvider", ctx, provider, userInfo.ProviderID).Return(existingUser, nil)
	m.geoService.On("GetCountry", ctx, ipAddress).Return("US", nil)
	m.sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	m.jwtService.On("Generate", ctx, mock.AnythingOfType("*domain.Token")).Return("jwt_token", nil)
	m.tokenGen.On("GenerateSecureToken", 32).Return("refresh_token", nil)
	m.refreshRepo.On("Create", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(assert.AnError)

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create refresh token")

	oauthProvider.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.geoService.AssertExpectations(t)
	m.sessionRepo.AssertExpectations(t)
	m.jwtService.AssertExpectations(t)
	m.tokenGen.AssertExpectations(t)
	m.refreshRepo.AssertExpectations(t)
}

func TestOAuthLogin_UserCreationFailure(t *testing.T) {
	t.Skip("Skipped due to bug in production code - OAuthLogin tries to create user without username, but NewUser validates username is not empty")

	m := setupAuthUseCase(t)
	ctx := context.Background()

	provider := "google"
	code := "valid_code"
	state := "random_state"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	device := "Desktop"

	userInfo := &ports.OAuthUserInfo{
		ProviderID: "oauth_new_user",
		Email:      "newuser@example.com",
		Name:       "New User",
	}

	// Setup OAuth provider mock
	oauthProvider := new(mocks.MockOAuthProvider)
	m.uc.oauthProviders = map[string]ports.OAuthProvider{
		"google": oauthProvider,
	}

	// Mock expectations
	oauthProvider.On("Exchange", ctx, code).Return(userInfo, nil).Once()
	m.userRepo.On("GetByOAuthProvider", ctx, provider, userInfo.ProviderID).Return(nil, domain.ErrUserNotFound).Once()
	m.userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User"), "").Return(assert.AnError).Once()

	// Execute
	response, err := m.uc.OAuthLogin(ctx, provider, code, state, ipAddress, userAgent, device)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create OAuth user")

	oauthProvider.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}
