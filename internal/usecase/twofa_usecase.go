package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type TwoFAUseCase struct {
	userRepo       ports.UserRepository
	backupCodeRepo ports.BackupCodeRepository
	totpService    ports.TOTPService
	hasher         ports.PasswordHasher
	tokenGen       ports.TokenGenerator
	logger         zerolog.Logger
}

func NewTwoFAUseCase(
	userRepo ports.UserRepository,
	backupCodeRepo ports.BackupCodeRepository,
	totpService ports.TOTPService,
	hasher ports.PasswordHasher,
	tokenGen ports.TokenGenerator,
	logger zerolog.Logger,
) *TwoFAUseCase {
	return &TwoFAUseCase{
		userRepo:       userRepo,
		backupCodeRepo: backupCodeRepo,
		totpService:    totpService,
		hasher:         hasher,
		tokenGen:       tokenGen,
		logger:         logger,
	}
}

type Enable2FAResponse struct {
	Secret string
	QRCode string
}

// Enable2FA generates a TOTP secret and QR code for the user
func (uc *TwoFAUseCase) Enable2FA(ctx context.Context, tenantID, userID string) (*Enable2FAResponse, error) {
	// Get user
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get user")
		return nil, err
	}

	if user.TwoFactorEnabled {
		return nil, fmt.Errorf("2FA already enabled")
	}

	// Generate TOTP secret and QR code
	secret, qrCode, err := uc.totpService.Generate(user.Email)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to generate TOTP secret")
		return nil, err
	}

	// Store secret temporarily (user needs to verify it before enabling)
	user.TwoFactorSecret = secret
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to update user with 2FA secret")
		return nil, err
	}

	uc.logger.Info().Str("user_id", userID).Msg("2FA secret generated")

	return &Enable2FAResponse{
		Secret: secret,
		QRCode: qrCode,
	}, nil
}

// Verify2FA verifies the TOTP code and enables 2FA for the user
func (uc *TwoFAUseCase) Verify2FA(ctx context.Context, tenantID, userID, code string) error {
	// Get user
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get user")
		return err
	}

	if user.TwoFactorEnabled {
		return fmt.Errorf("2FA already enabled")
	}

	if user.TwoFactorSecret == "" {
		return fmt.Errorf("2FA not initialized. Call enable endpoint first")
	}

	// Verify TOTP code
	valid, err := uc.totpService.Verify(code, user.TwoFactorSecret)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to validate TOTP code")
		return err
	}

	if !valid {
		uc.logger.Warn().Str("user_id", userID).Msg("invalid 2FA code during verification")
		return domain.ErrInvalidCredentials
	}

	// Enable 2FA
	user.TwoFactorEnabled = true
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to enable 2FA")
		return err
	}

	uc.logger.Info().Str("user_id", userID).Msg("2FA enabled successfully")

	return nil
}

// Disable2FA disables 2FA for the user after verifying the code
func (uc *TwoFAUseCase) Disable2FA(ctx context.Context, tenantID, userID, code string) error {
	// Get user
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get user")
		return err
	}

	if !user.TwoFactorEnabled {
		return fmt.Errorf("2FA not enabled")
	}

	// Verify TOTP code before disabling
	valid, err := uc.totpService.Verify(code, user.TwoFactorSecret)
	if err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to validate TOTP code")
		return err
	}

	if !valid {
		uc.logger.Warn().Str("user_id", userID).Msg("invalid 2FA code during disable")
		return domain.ErrInvalidCredentials
	}

	// Disable 2FA and clear secret
	user.TwoFactorEnabled = false
	user.TwoFactorSecret = ""
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		uc.logger.Error().Err(err).Str("user_id", userID).Msg("failed to disable 2FA")
		return err
	}

	uc.logger.Info().Str("user_id", userID).Msg("2FA disabled successfully")

	return nil
}

// GenerateBackupCodes generates 10 new backup codes for the user
func (uc *TwoFAUseCase) GenerateBackupCodes(ctx context.Context, tenantID, userID string) ([]string, error) {
	// Verify user exists
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if !user.TwoFactorEnabled {
		return nil, domain.Err2FANotEnabled
	}

	// Generate 10 codes
	plainCodes := make([]string, 10)
	var backupCodes []*domain.BackupCode

	for i := 0; i < 10; i++ {
		// Create a 10-char alphanumeric code
		code, err := uc.tokenGen.GenerateSecureToken(10)
		if err != nil {
			return nil, fmt.Errorf("failed to generate secure token: %w", err)
		}
		plainCodes[i] = code

		// Hash code for storage
		hash, err := uc.hasher.Hash(code)
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}

		backupCodes = append(backupCodes, domain.NewBackupCode(tenantID, userID, hash))
	}

	// Delete existing codes first (only 10 active at a time)
	if err := uc.backupCodeRepo.DeleteAllByUserID(ctx, tenantID, userID); err != nil {
		return nil, err
	}

	// Save new ones
	if err := uc.backupCodeRepo.CreateMany(ctx, backupCodes); err != nil {
		return nil, err
	}

	uc.logger.Info().Str("user_id", userID).Msg("Backup codes regenerated")

	return plainCodes, nil
}
