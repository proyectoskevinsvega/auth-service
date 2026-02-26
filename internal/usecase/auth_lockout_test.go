package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
)

func TestAuthUseCase_Login_Lockout(t *testing.T) {
	m := setupAuthUseCase(t)
	ctx := context.Background()

	// Update config for test
	m.uc.config.Security.Lockout = struct {
		MaxAttempts      int
		BaseDuration     time.Duration
		EscalationFactor float64
		MaxDuration      time.Duration
	}{
		MaxAttempts:      3,
		BaseDuration:     time.Minute * 1,
		EscalationFactor: 2.0,
		MaxDuration:      time.Hour,
	}

	userID := uuid.New().String()
	userEmail := "test@example.com"
	password := "SecurePass123!"
	passwordHash := "$argon2id$hashed"

	input := LoginInput{
		Identifier: userEmail,
		Password:   password,
		IPAddress:  "1.1.1.1",
		UserAgent:  "TestAgent",
	}

	t.Run("Should lock account after multiple failed attempts", func(t *testing.T) {
		user := &domain.User{
			ID:           userID,
			Email:        userEmail,
			PasswordHash: passwordHash,
			Active:       true,
		}

		// Mock common expectations
		m.rateLimiter.On("CheckLimit", ctx, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		m.rateLimiter.On("Increment", ctx, mock.Anything, mock.Anything).Return(1, nil)
		m.passwordHasher.On("Verify", "wrong", passwordHash).Return(false, nil)
		m.auditRepo.On("Create", ctx, mock.Anything).Return(nil)
		m.redisNotifier.On("Publish", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil)

		// 1st failure
		m.userRepo.On("GetByEmailOrUsername", ctx, input.TenantID, userEmail).Return(user, nil).Once()
		m.userRepo.On("Update", ctx, user).Return(nil).Once()
		_, err := m.uc.Login(ctx, LoginInput{TenantID: input.TenantID, Identifier: userEmail, Password: "wrong"})
		assert.Equal(t, domain.ErrInvalidCredentials, err)
		assert.Equal(t, 1, user.FailedLoginAttempts)

		// 2nd failure
		m.userRepo.On("GetByEmailOrUsername", ctx, input.TenantID, userEmail).Return(user, nil).Once()
		m.userRepo.On("Update", ctx, user).Return(nil).Once()
		_, err = m.uc.Login(ctx, LoginInput{TenantID: input.TenantID, Identifier: userEmail, Password: "wrong"})
		assert.Equal(t, domain.ErrInvalidCredentials, err)
		assert.Equal(t, 2, user.FailedLoginAttempts)

		// 3rd failure - SHOULD LOCK
		m.userRepo.On("GetByEmailOrUsername", ctx, input.TenantID, userEmail).Return(user, nil).Once()
		m.userRepo.On("Update", ctx, user).Return(nil).Once()
		_, err = m.uc.Login(ctx, LoginInput{TenantID: input.TenantID, Identifier: userEmail, Password: "wrong"})
		assert.Equal(t, domain.ErrAccountLocked, err)
		assert.True(t, user.IsLocked())

		// Verify further attempts are blocked immediately
		m.userRepo.On("GetByEmailOrUsername", ctx, input.TenantID, userEmail).Return(user, nil).Once()
		_, err = m.uc.Login(ctx, LoginInput{TenantID: input.TenantID, Identifier: userEmail, Password: "any"})
		assert.Equal(t, domain.ErrAccountLocked, err)

		m.userRepo.AssertExpectations(t)
	})

	t.Run("Should reset attempts on success", func(t *testing.T) {
		m := setupAuthUseCase(t) // Fresh mocks

		user := &domain.User{
			ID:                  userID,
			Email:               userEmail,
			PasswordHash:        passwordHash,
			Active:              true,
			FailedLoginAttempts: 2,
		}

		m.rateLimiter.On("CheckLimit", ctx, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		m.rateLimiter.On("Increment", ctx, mock.Anything, mock.Anything).Return(1, nil)
		m.userRepo.On("GetByEmailOrUsername", ctx, input.TenantID, userEmail).Return(user, nil)
		m.passwordHasher.On("Verify", password, passwordHash).Return(true, nil)

		// Expected reset update
		m.userRepo.On("Update", ctx, user).Return(nil).Once()

		// Mock others for successful flow
		m.riskService.On("AssessLoginRisk", ctx, user, input.IPAddress).Return(domain.NewRiskAssessment(), &domain.Geolocation{Country: "US"}, nil)
		m.sessionRepo.On("Create", ctx, mock.Anything).Return(nil)
		m.tokenGen.On("GenerateSecureToken", mock.Anything).Return("refresh", nil)
		m.refreshRepo.On("Create", ctx, mock.Anything).Return(nil)
		m.jwtService.On("Generate", ctx, mock.Anything).Return("access", nil)
		m.auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
		m.userRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Maybe()

		response, err := m.uc.Login(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 0, user.FailedLoginAttempts)
		assert.Nil(t, user.LockedUntil)
	})
}
