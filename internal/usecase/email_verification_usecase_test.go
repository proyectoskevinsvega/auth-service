package usecase

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/tests/mocks"
)

type EmailVerificationUseCaseMocks struct {
	uc               *EmailVerificationUseCase
	userRepo         *mocks.MockUserRepository
	verificationRepo *mocks.MockEmailVerificationRepository
	emailService     *mocks.MockEmailService
}

func setupEmailVerificationUseCase(_ *testing.T) EmailVerificationUseCaseMocks {
	userRepo := new(mocks.MockUserRepository)
	verificationRepo := new(mocks.MockEmailVerificationRepository)
	emailService := new(mocks.MockEmailService)
	logger := zerolog.New(os.Stdout).Level(zerolog.Disabled)

	uc := NewEmailVerificationUseCase(userRepo, verificationRepo, emailService, logger, "example.com", "development")

	return EmailVerificationUseCaseMocks{
		uc:               uc,
		userRepo:         userRepo,
		verificationRepo: verificationRepo,
		emailService:     emailService,
	}
}

// TestSendVerificationEmail_Success prueba el envío exitoso de un email de verificación
func TestSendVerificationEmail_Success(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: false,
		Active:        true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	input := SendVerificationInput{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.verificationRepo.On("Create", ctx, mock.AnythingOfType("*domain.EmailVerification")).Return(nil)
	m.emailService.On("SendVerificationEmail", ctx, user.Email, user.Username, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// Execute
	err := m.uc.SendVerificationEmail(ctx, input)

	// Assert
	assert.NoError(t, err)
	m.userRepo.AssertExpectations(t)
	m.verificationRepo.AssertExpectations(t)
	m.emailService.AssertExpectations(t)
}

// TestSendVerificationEmail_UserNotFound prueba error cuando el usuario no existe
func TestSendVerificationEmail_UserNotFound(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	input := SendVerificationInput{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(nil, domain.ErrUserNotFound)

	// Execute
	err := m.uc.SendVerificationEmail(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")
	m.userRepo.AssertExpectations(t)
}

// TestSendVerificationEmail_EmailAlreadyVerified prueba error cuando el email ya está verificado
func TestSendVerificationEmail_EmailAlreadyVerified(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: true, // Ya verificado
		Active:        true,
	}

	input := SendVerificationInput{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)

	// Execute
	err := m.uc.SendVerificationEmail(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrEmailAlreadyVerified, err)

	// No debe llamar a Create ni SendVerificationEmail
	m.verificationRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	m.emailService.AssertNotCalled(t, "SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
}

// TestSendVerificationEmail_EmailServiceFailure prueba falla al enviar el email
func TestSendVerificationEmail_EmailServiceFailure(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: false,
		Active:        true,
	}

	input := SendVerificationInput{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.verificationRepo.On("Create", ctx, mock.AnythingOfType("*domain.EmailVerification")).Return(nil)
	m.emailService.On("SendVerificationEmail", ctx, user.Email, user.Username, mock.AnythingOfType("map[string]interface {}")).Return(fmt.Errorf("email service unavailable"))

	// Execute
	err := m.uc.SendVerificationEmail(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send verification email")
	m.userRepo.AssertExpectations(t)
	m.verificationRepo.AssertExpectations(t)
	m.emailService.AssertExpectations(t)
}

// TestVerifyEmail_Success prueba la verificación exitosa de un email
func TestVerifyEmail_Success(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	token := "valid_token_12345"
	tokenHash := "hashed_token"

	verification := &domain.EmailVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     userID,
		TokenHash:  tokenHash,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		VerifiedAt: nil,
		CreatedAt:  time.Now(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: false,
		Active:        true,
	}

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(verification, nil)
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.verificationRepo.On("MarkAsVerified", ctx, tenantID, mock.AnythingOfType("string")).Return(nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.NoError(t, err)
	m.verificationRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestVerifyEmail_TokenNotFound prueba error cuando el token no existe
func TestVerifyEmail_TokenNotFound(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	tenantID := "test-tenant"
	token := "invalid_token"

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(nil, domain.ErrVerificationTokenNotFound)

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrVerificationTokenNotFound, err)
	m.verificationRepo.AssertExpectations(t)
}

// TestVerifyEmail_TokenAlreadyUsed prueba error cuando el token ya fue usado
func TestVerifyEmail_TokenAlreadyUsed(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	tenantID := "test-tenant"
	token := "used_token"
	verifiedAt := time.Now().Add(-1 * time.Hour)

	verification := &domain.EmailVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     uuid.New().String(),
		TokenHash:  "hashed_token",
		ExpiresAt:  time.Now().Add(23 * time.Hour),
		VerifiedAt: &verifiedAt, // Ya verificado
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(verification, nil)

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrVerificationTokenUsed, err)

	// No debe llamar a MarkAsVerified ni Update
	m.verificationRepo.AssertNotCalled(t, "MarkAsVerified", mock.Anything, mock.Anything)
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.verificationRepo.AssertExpectations(t)
}

// TestVerifyEmail_TokenExpired prueba error cuando el token expiró
func TestVerifyEmail_TokenExpired(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	tenantID := "test-tenant"
	token := "expired_token"

	verification := &domain.EmailVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     uuid.New().String(),
		TokenHash:  "hashed_token",
		ExpiresAt:  time.Now().Add(-1 * time.Hour), // Expirado
		VerifiedAt: nil,
		CreatedAt:  time.Now().Add(-25 * time.Hour),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(verification, nil)

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrVerificationTokenExpired, err)

	// No debe llamar a MarkAsVerified ni Update
	m.verificationRepo.AssertNotCalled(t, "MarkAsVerified", mock.Anything, mock.Anything)
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.verificationRepo.AssertExpectations(t)
}

// TestVerifyEmail_EmailAlreadyVerified prueba error cuando el email ya está verificado (edge case)
func TestVerifyEmail_EmailAlreadyVerified(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	token := "valid_token"

	verification := &domain.EmailVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     userID,
		TokenHash:  "hashed_token",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		VerifiedAt: nil,
		CreatedAt:  time.Now(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: true, // Ya verificado
		Active:        true,
	}

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(verification, nil)
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrEmailAlreadyVerified, err)

	// No debe llamar a MarkAsVerified ni Update
	m.verificationRepo.AssertNotCalled(t, "MarkAsVerified", mock.Anything, mock.Anything)
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.verificationRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestResendVerificationEmail_Success prueba el reenvío exitoso del email de verificación
func TestResendVerificationEmail_Success(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: false,
		Active:        true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil).Times(2) // Called by ResendVerificationEmail and SendVerificationEmail
	m.verificationRepo.On("DeleteByUserID", ctx, tenantID, userID).Return(nil)
	m.verificationRepo.On("Create", ctx, mock.AnythingOfType("*domain.EmailVerification")).Return(nil)
	m.emailService.On("SendVerificationEmail", ctx, user.Email, user.Username, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// Execute
	err := m.uc.ResendVerificationEmail(ctx, tenantID, userID, "192.168.1.1", "Mozilla/5.0")

	// Assert
	assert.NoError(t, err)
	m.userRepo.AssertExpectations(t)
	m.verificationRepo.AssertExpectations(t)
	m.emailService.AssertExpectations(t)
}

// TestResendVerificationEmail_UserNotFound prueba error cuando el usuario no existe
func TestResendVerificationEmail_UserNotFound(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(nil, domain.ErrUserNotFound)

	// Execute
	err := m.uc.ResendVerificationEmail(ctx, tenantID, userID, "192.168.1.1", "Mozilla/5.0")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")

	// No debe llamar a DeleteByUserID
	m.verificationRepo.AssertNotCalled(t, "DeleteByUserID", mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
}

// TestResendVerificationEmail_EmailAlreadyVerified prueba error cuando el email ya está verificado
func TestResendVerificationEmail_EmailAlreadyVerified(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: true, // Ya verificado
		Active:        true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)

	// Execute
	err := m.uc.ResendVerificationEmail(ctx, tenantID, userID, "192.168.1.1", "Mozilla/5.0")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrEmailAlreadyVerified, err)

	// No debe llamar a DeleteByUserID ni Create
	m.verificationRepo.AssertNotCalled(t, "DeleteByUserID", mock.Anything, mock.Anything)
	m.verificationRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
}

// TestSendVerificationEmail_CreateError prueba error al crear el registro de verificación
func TestSendVerificationEmail_CreateError(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: false,
		Active:        true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	input := SendVerificationInput{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.verificationRepo.On("Create", ctx, mock.AnythingOfType("*domain.EmailVerification")).Return(fmt.Errorf("database error"))

	// Execute
	err := m.uc.SendVerificationEmail(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create verification")
	m.userRepo.AssertExpectations(t)
	m.verificationRepo.AssertExpectations(t)
}

// TestVerifyEmail_UpdateError prueba error al actualizar el usuario
func TestVerifyEmail_UpdateError(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	token := "valid_token_12345"

	verification := &domain.EmailVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     userID,
		TokenHash:  "hashed_token",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		VerifiedAt: nil,
		CreatedAt:  time.Now(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: false,
		Active:        true,
	}

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(verification, nil)
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.verificationRepo.On("MarkAsVerified", ctx, tenantID, mock.AnythingOfType("string")).Return(nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(fmt.Errorf("update failed"))

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update user")
	m.verificationRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestVerifyEmail_MarkAsVerifiedError prueba error al marcar como verificado
func TestVerifyEmail_MarkAsVerifiedError(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	token := "valid_token_12345"

	verification := &domain.EmailVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     userID,
		TokenHash:  "hashed_token",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		VerifiedAt: nil,
		CreatedAt:  time.Now(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: false,
		Active:        true,
	}

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(verification, nil)
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.verificationRepo.On("MarkAsVerified", ctx, tenantID, mock.AnythingOfType("string")).Return(fmt.Errorf("mark failed"))

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to mark verification as used")
	m.verificationRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}

// TestVerifyEmail_UserNotFound prueba error cuando el usuario no existe
func TestVerifyEmail_UserNotFound(t *testing.T) {
	m := setupEmailVerificationUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	token := "valid_token_12345"

	verification := &domain.EmailVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     userID,
		TokenHash:  "hashed_token",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		VerifiedAt: nil,
		CreatedAt:  time.Now(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	// Mock expectations
	m.verificationRepo.On("GetByTokenHash", ctx, tenantID, mock.AnythingOfType("string")).Return(verification, nil)
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(nil, domain.ErrUserNotFound)

	// Execute
	err := m.uc.VerifyEmail(ctx, tenantID, token)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")
	m.verificationRepo.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
}
