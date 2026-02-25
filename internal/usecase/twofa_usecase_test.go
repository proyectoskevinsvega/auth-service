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

type TwoFAUseCaseMocks struct {
	uc             *TwoFAUseCase
	userRepo       *mocks.MockUserRepository
	backupCodeRepo *mocks.MockBackupCodeRepository
	totpService    *mocks.MockTOTPService
	hasher         *mocks.MockPasswordHasher
	tokenGen       *mocks.MockTokenGenerator
}

func setupTwoFAUseCase(_ *testing.T) TwoFAUseCaseMocks {
	userRepo := new(mocks.MockUserRepository)
	backupCodeRepo := new(mocks.MockBackupCodeRepository)
	totpService := new(mocks.MockTOTPService)
	hasher := new(mocks.MockPasswordHasher)
	tokenGen := new(mocks.MockTokenGenerator)
	logger := zerolog.New(os.Stdout).Level(zerolog.Disabled)

	uc := NewTwoFAUseCase(userRepo, backupCodeRepo, totpService, hasher, tokenGen, logger)

	return TwoFAUseCaseMocks{
		uc:             uc,
		userRepo:       userRepo,
		backupCodeRepo: backupCodeRepo,
		totpService:    totpService,
		hasher:         hasher,
		tokenGen:       tokenGen,
	}
}

// TestEnable2FA_Success prueba la generación exitosa del secret y QR code
func TestEnable2FA_Success(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:       userID,
		TenantID: tenantID,
		Username: "testuser",
		Email:    "test@example.com",

		TwoFactorEnabled: false,
		TwoFactorSecret:  "",
		Active:           true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	secret := "JBSWY3DPEHPK3PXP"
	qrCode := "data:image/png;base64,iVBORw0KGgoAAAANS..."

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Generate", user.Email).Return(secret, qrCode, nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Execute
	response, err := m.uc.Enable2FA(ctx, tenantID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, secret, response.Secret)
	assert.Equal(t, qrCode, response.QRCode)

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestEnable2FA_UserNotFound prueba error cuando el usuario no existe
func TestEnable2FA_UserNotFound(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(nil, domain.ErrUserNotFound)

	// Execute
	response, err := m.uc.Enable2FA(ctx, tenantID, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrUserNotFound, err)
	assert.Nil(t, response)

	m.userRepo.AssertExpectations(t)
}

// TestEnable2FA_AlreadyEnabled prueba error cuando 2FA ya está habilitado
func TestEnable2FA_AlreadyEnabled(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: true, // Ya está habilitado
		TwoFactorSecret:  "EXISTING_SECRET",
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)

	// Execute
	response, err := m.uc.Enable2FA(ctx, tenantID, userID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2FA already enabled")
	assert.Nil(t, response)

	// No debe llamar a Generate ni Update
	m.totpService.AssertNotCalled(t, "Generate", mock.Anything)
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
}

// TestEnable2FA_GenerateError prueba error al generar el secret
func TestEnable2FA_GenerateError(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Generate", user.Email).Return("", "", fmt.Errorf("failed to generate secret"))

	// Execute
	response, err := m.uc.Enable2FA(ctx, tenantID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestVerify2FA_Success prueba la verificación y activación exitosa de 2FA
func TestVerify2FA_Success(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false,
		TwoFactorSecret:  secret, // Secret ya generado por Enable2FA
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(true, nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Execute
	err := m.uc.Verify2FA(ctx, tenantID, userID, code)

	// Assert
	assert.NoError(t, err)

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestVerify2FA_InvalidCode prueba error con código inválido
func TestVerify2FA_InvalidCode(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "999999"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false,
		TwoFactorSecret:  secret,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(false, nil) // Código inválido

	// Execute
	err := m.uc.Verify2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidCredentials, err)

	// No debe llamar a Update
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestVerify2FA_NotInitialized prueba error cuando no se llamó Enable2FA primero
func TestVerify2FA_NotInitialized(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false,
		TwoFactorSecret:  "", // Secret vacío - no se llamó Enable2FA
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)

	// Execute
	err := m.uc.Verify2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2FA not initialized")

	// No debe llamar a Verify ni Update
	m.totpService.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything)
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
}

// TestVerify2FA_AlreadyEnabled prueba error cuando 2FA ya está habilitado
func TestVerify2FA_AlreadyEnabled(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: true, // Ya está habilitado
		TwoFactorSecret:  "EXISTING_SECRET",
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)

	// Execute
	err := m.uc.Verify2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2FA already enabled")

	m.userRepo.AssertExpectations(t)
}

// TestDisable2FA_Success prueba la desactivación exitosa de 2FA
func TestDisable2FA_Success(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: true,
		TwoFactorSecret:  secret,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(true, nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Execute
	err := m.uc.Disable2FA(ctx, tenantID, userID, code)

	// Assert
	assert.NoError(t, err)

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestDisable2FA_InvalidCode prueba error con código inválido
func TestDisable2FA_InvalidCode(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "999999"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: true,
		TwoFactorSecret:  secret,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(false, nil)

	// Execute
	err := m.uc.Disable2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidCredentials, err)

	// No debe llamar a Update
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestDisable2FA_NotEnabled prueba error cuando 2FA no está habilitado
func TestDisable2FA_NotEnabled(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false, // No está habilitado
		TwoFactorSecret:  "",
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)

	// Execute
	err := m.uc.Disable2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2FA not enabled")

	// No debe llamar a Verify ni Update
	m.totpService.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything)
	m.userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	m.userRepo.AssertExpectations(t)
}

// TestVerify2FA_VerificationError prueba error al verificar el código TOTP
func TestVerify2FA_VerificationError(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false,
		TwoFactorSecret:  secret,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(false, fmt.Errorf("totp service error"))

	// Execute
	err := m.uc.Verify2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "totp service error")

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestDisable2FA_VerificationError prueba error al verificar el código durante deshabilitación
func TestDisable2FA_VerificationError(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: true,
		TwoFactorSecret:  secret,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(false, fmt.Errorf("totp verification failed"))

	// Execute
	err := m.uc.Disable2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "totp verification failed")

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestEnable2FA_UpdateError prueba error al actualizar el usuario
func TestEnable2FA_UpdateError(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false,
		TwoFactorSecret:  "",
		Active:           true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	secret := "JBSWY3DPEHPK3PXP"
	qrCode := "data:image/png;base64,iVBORw0KGgoAAAANS..."

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Generate", user.Email).Return(secret, qrCode, nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(fmt.Errorf("database error"))

	// Execute
	response, err := m.uc.Enable2FA(ctx, tenantID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "database error")

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestVerify2FA_UpdateError prueba error al actualizar el usuario después de verificar
func TestVerify2FA_UpdateError(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: false,
		TwoFactorSecret:  secret,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(true, nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(fmt.Errorf("update failed"))

	// Execute
	err := m.uc.Verify2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}

// TestDisable2FA_UpdateError prueba error al actualizar el usuario durante deshabilitación
func TestDisable2FA_UpdateError(t *testing.T) {
	m := setupTwoFAUseCase(t)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := "test-tenant"
	secret := "JBSWY3DPEHPK3PXP"
	code := "123456"

	user := &domain.User{
		ID:               userID,
		Username:         "testuser",
		Email:            "test@example.com",
		TwoFactorEnabled: true,
		TwoFactorSecret:  secret,
		Active:           true,
	}

	// Mock expectations
	m.userRepo.On("GetByID", ctx, tenantID, userID).Return(user, nil)
	m.totpService.On("Verify", code, secret).Return(true, nil)
	m.userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(fmt.Errorf("update error"))

	// Execute
	err := m.uc.Disable2FA(ctx, tenantID, userID, code)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update error")

	m.userRepo.AssertExpectations(t)
	m.totpService.AssertExpectations(t)
}
