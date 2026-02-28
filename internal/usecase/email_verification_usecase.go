package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"time"

	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

const (
	// Email verification token expires after 5 minutes
	EmailVerificationExpiry = 5 * time.Minute
	// Token length in bytes (32 bytes = 256 bits of entropy)
	VerificationTokenLength = 32
)

type EmailVerificationUseCase struct {
	userRepo         ports.UserRepository
	verificationRepo ports.EmailVerificationRepository
	emailService     ports.EmailService
	notifier         ports.NotificationPublisher
	logger           zerolog.Logger
	baseDomain       string
	environment      string
	rateLimiter      ports.RateLimiter
	config           *config.Config
}

func NewEmailVerificationUseCase(
	userRepo ports.UserRepository,
	verificationRepo ports.EmailVerificationRepository,
	emailService ports.EmailService,
	notifier ports.NotificationPublisher,
	logger zerolog.Logger,
	baseDomain string,
	environment string,
	rateLimiter ports.RateLimiter,
	cfg *config.Config,
) *EmailVerificationUseCase {
	return &EmailVerificationUseCase{
		userRepo:         userRepo,
		verificationRepo: verificationRepo,
		emailService:     emailService,
		notifier:         notifier,
		logger:           logger,
		baseDomain:       baseDomain,
		environment:      environment,
		rateLimiter:      rateLimiter,
		config:           cfg,
	}
}

type SendVerificationInput struct {
	TenantID  string
	UserID    string
	IPAddress string
	UserAgent string
}

// SendVerificationEmail generates a verification token and sends email
func (uc *EmailVerificationUseCase) SendVerificationEmail(ctx context.Context, input SendVerificationInput) error {
	// Get user
	user, err := uc.userRepo.GetByID(ctx, input.TenantID, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if email is already verified
	if user.EmailVerified {
		return domain.ErrEmailAlreadyVerified
	}

	// Generate secure random token
	token, tokenHash, err := uc.generateVerificationToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Create verification record
	verification := domain.NewEmailVerification(
		user.TenantID,
		user.ID,
		tokenHash,
		EmailVerificationExpiry,
		input.IPAddress,
		input.UserAgent,
	)

	if err := uc.verificationRepo.Create(ctx, verification); err != nil {
		return fmt.Errorf("failed to create verification: %w", err)
	}

	// Build verification URL
	verificationURL := BuildURL(uc.environment, uc.baseDomain, fmt.Sprintf("/api/v1/auth/verify-email?token=%s", token))

	// Send email (only if email service is enabled)
	if uc.emailService != nil {
		emailData := map[string]interface{}{
			"username":         user.Username,
			"verification_url": verificationURL,
			"token":            token,
			"expires_minutes":  int(EmailVerificationExpiry.Minutes()),
		}

		if err := uc.emailService.SendVerificationEmail(ctx, user.Email, user.Username, emailData); err != nil {
			uc.logger.Error().Err(err).Str("user_id", user.ID).Msg("failed to send verification email")
			// Don't fail the request if email sending fails
			return fmt.Errorf("failed to send verification email: %w", err)
		}

		uc.logger.Info().Str("user_id", user.ID).Str("email", user.Email).Msg("verification email sent")
	} else {
		uc.logger.Warn().Str("user_id", user.ID).Msg("email service disabled, verification token created but not sent")
	}

	return nil
}

// VerifyEmail verifies an email using the provided token, protected by Redis rate limiting
func (uc *EmailVerificationUseCase) VerifyEmail(ctx context.Context, tenantID, token, ipAddress string) error {
	// Rate limiting - Protect verification endpoint against brute-force (e.g. 5 guesses per 15 minutes)
	rateLimitKey := fmt.Sprintf("verify_email_attempt:%s:%s", tenantID, ipAddress)
	if uc.rateLimiter != nil && uc.config != nil {
		exceeded, err := uc.rateLimiter.CheckLimit(ctx, rateLimitKey, uc.config.RateLimit.VerifyAttempts, uc.config.RateLimit.VerifyWindow)
		if err != nil {
			return fmt.Errorf("rate limit check failed: %w", err)
		}
		if exceeded {
			return domain.ErrRateLimitExceeded
		}

		// Unconditionally count this attempt in Redis to prevent massive parallel guessing
		if _, err := uc.rateLimiter.Increment(context.Background(), rateLimitKey, uc.config.RateLimit.VerifyWindow); err != nil {
			fmt.Printf("warning: failed to increment verify rate limit: %v\n", err)
		}
	}

	// Hash the token to match database
	tokenHash := uc.hashToken(token)

	// Get verification record
	verification, err := uc.verificationRepo.GetByTokenHash(ctx, tenantID, tokenHash)
	if err != nil {
		return err // Returns ErrVerificationTokenNotFound if not found
	}

	// Check if already verified
	if verification.IsVerified() {
		return domain.ErrVerificationTokenUsed
	}

	// Check if expired
	if verification.IsExpired() {
		return domain.ErrVerificationTokenExpired
	}

	// Get user
	user, err := uc.userRepo.GetByID(ctx, verification.TenantID, verification.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if email already verified (edge case: multiple tokens)
	if user.EmailVerified {
		return domain.ErrEmailAlreadyVerified
	}

	// Mark verification as used
	if err := uc.verificationRepo.MarkAsVerified(ctx, tenantID, tokenHash); err != nil {
		return fmt.Errorf("failed to mark verification as used: %w", err)
	}

	// Update user email_verified flag
	user.EmailVerified = true
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	uc.logger.Info().Str("user_id", user.ID).Str("email", user.Email).Msg("email verified successfully")

	// Emit webhook event
	event := domain.NewEvent(user.TenantID, domain.EventEmailVerified, user.ID, user.Email, nil)
	_ = uc.notifier.Publish(ctx, event)

	return nil
}

// ResendVerificationEmail deletes old tokens and sends a new verification email
func (uc *EmailVerificationUseCase) ResendVerificationEmail(ctx context.Context, tenantID, userID, ipAddress, userAgent string) error {
	// Get user
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if email is already verified
	if user.EmailVerified {
		return domain.ErrEmailAlreadyVerified
	}

	// Delete old verification tokens for this user
	if err := uc.verificationRepo.DeleteByUserID(ctx, tenantID, userID); err != nil {
		uc.logger.Warn().Err(err).Str("user_id", userID).Msg("failed to delete old verification tokens")
		// Continue anyway
	}

	// Send new verification email
	return uc.SendVerificationEmail(ctx, SendVerificationInput{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

// generateVerificationToken generates a secure random 6-digit PIN and its hash
func (uc *EmailVerificationUseCase) generateVerificationToken() (token string, tokenHash string, err error) {
	// Generate a secure 6-digit numeric PIN
	// 1000000 ensures it's between 000000 and 999999
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random PIN: %w", err)
	}

	// Format as 6-digit string with leading zeros if necessary
	token = fmt.Sprintf("%06d", n.Int64())

	// Hash for storage (prevents token leakage from database dumps)
	tokenHash = uc.hashToken(token)

	return token, tokenHash, nil
}

// hashToken creates a SHA-256 hash of the token
func (uc *EmailVerificationUseCase) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
