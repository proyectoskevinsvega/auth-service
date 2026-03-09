package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type AuthUseCase struct {
	userRepo       ports.UserRepository
	sessionRepo    ports.SessionRepository
	refreshRepo    ports.RefreshTokenRepository
	resetRepo      ports.PasswordResetRepository
	auditRepo      ports.AuditLogRepository
	jwtService     ports.JWTService
	passwordHasher ports.PasswordHasher
	tokenGen       ports.TokenGenerator
	rateLimiter    ports.RateLimiter
	sessionStore   ports.SessionStore
	geoService     ports.GeolocationService
	emailService   ports.EmailService
	notifier       ports.NotificationPublisher
	oauthProviders map[string]ports.OAuthProvider
	config         *config.Config
	riskService    ports.RiskService
	roleRepo       ports.RoleRepository // Added for RBAC
	backupCodeRepo ports.BackupCodeRepository
	totpService    ports.TOTPService
	tenantRepo     ports.TenantRepository
}

func NewAuthUseCase(
	userRepo ports.UserRepository,
	sessionRepo ports.SessionRepository,
	refreshRepo ports.RefreshTokenRepository,
	resetRepo ports.PasswordResetRepository,
	auditRepo ports.AuditLogRepository,
	jwtService ports.JWTService,
	passwordHasher ports.PasswordHasher,
	tokenGen ports.TokenGenerator,
	rateLimiter ports.RateLimiter,
	sessionStore ports.SessionStore,
	geoService ports.GeolocationService,
	emailService ports.EmailService,
	notifier ports.NotificationPublisher,
	oauthProviders map[string]ports.OAuthProvider,
	cfg *config.Config,
	riskService ports.RiskService,
	roleRepo ports.RoleRepository,
	backupCodeRepo ports.BackupCodeRepository,
	totpService ports.TOTPService,
	tenantRepo ports.TenantRepository,
) *AuthUseCase {
	return &AuthUseCase{
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
		oauthProviders: oauthProviders,
		config:         cfg,
		riskService:    riskService,
		roleRepo:       roleRepo,
		backupCodeRepo: backupCodeRepo,
		totpService:    totpService,
		tenantRepo:     tenantRepo,
	}
}

type LoginInput struct {
	TenantID   string
	Identifier string // Email o username
	Password   string
	TwoFACode  string
	IPAddress  string
	UserAgent  string
	Device     string
}

type LoginResponse struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
}

func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*LoginResponse, error) {
	// Auto-Translate Slug to UUID
	tenant, err := uc.tenantRepo.GetBySlug(ctx, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup tenant: %w", err)
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant with slug %s not found", input.TenantID)
	}
	// Override the textual slug with the database UUID before doing queries
	input.TenantID = tenant.ID

	// Rate limiting
	// Rate limit check (consolidated)
	rateLimitKey := fmt.Sprintf("login:%s:%s", input.TenantID, input.Identifier)
	exceeded, err := uc.rateLimiter.CheckLimit(ctx, rateLimitKey, uc.config.RateLimit.LoginAttempts, uc.config.RateLimit.LoginWindow)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}
	if exceeded {
		return nil, domain.ErrRateLimitExceeded
	}

	// Increment rate limit counter (non-blocking, log error only)
	if _, err := uc.rateLimiter.Increment(ctx, rateLimitKey, uc.config.RateLimit.LoginWindow); err != nil {
		// Log but don't fail - rate limiting tracking is not critical for request flow
		fmt.Printf("warning: failed to increment login rate limit for %s: %v\n", input.IPAddress, err)
	}

	// Get user by email or username
	user, err := uc.userRepo.GetByEmailOrUsername(ctx, input.TenantID, input.Identifier)
	if err != nil {
		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(input.TenantID, "", "login", input.IPAddress, input.UserAgent, "", false, "invalid credentials", nil))
		return nil, domain.ErrInvalidCredentials
	}

	// Check if user is active
	if !user.Active {
		return nil, domain.ErrUserInactive
	}

	// Check if account is locked (P0)
	if user.IsLocked() {
		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "login_attempt_locked", input.IPAddress, input.UserAgent, "", false, "login attempt on locked account", nil))
		return nil, domain.ErrAccountLocked
	}

	// Verify password
	valid, err := uc.passwordHasher.Verify(input.Password, user.PasswordHash)
	if err != nil || !valid {
		// Increment failed attempts and potentially lock account
		lockout := uc.config.Security.Lockout
		user.IncrementFailedAttempts(
			lockout.MaxAttempts,
			lockout.BaseDuration,
			lockout.EscalationFactor,
			lockout.MaxDuration,
		)
		_ = uc.userRepo.Update(ctx, user)

		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "login_failed", input.IPAddress, input.UserAgent, "", false, "invalid credentials", map[string]interface{}{
			"failed_attempts": user.FailedLoginAttempts,
			"locked_until":    user.LockedUntil,
		}))

		if user.IsLocked() {
			event := domain.NewEvent(user.TenantID, domain.EventAccountLocked, user.ID, user.Email, map[string]interface{}{
				"reason":       "too_many_failed_attempts",
				"ip":           input.IPAddress,
				"locked_until": user.LockedUntil,
			})
			_ = uc.notifier.Publish(ctx, event)
			return nil, domain.ErrAccountLocked
		}

		event := domain.NewEvent(user.TenantID, domain.EventLoginFailed, user.ID, user.Email, map[string]interface{}{
			"reason":          "invalid_credentials",
			"ip":              input.IPAddress,
			"failed_attempts": user.FailedLoginAttempts,
		})
		_ = uc.notifier.Publish(ctx, event)

		return nil, domain.ErrInvalidCredentials
	}

	// Password valid - reset failed attempts
	user.ResetFailedAttempts()
	// Update user immediate or in the background? Better immediate for security sync.
	if err := uc.userRepo.Update(ctx, user); err != nil {
		fmt.Printf("warning: failed to reset failed login attempts: %v\n", err)
	}

	// Email Verification check (P0)
	if !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	// Check password expiration (P0)
	if user.IsPasswordExpired(uc.config.Security.PasswordExpiry.ExpiryDays) {
		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "password_expired", input.IPAddress, input.UserAgent, "", false, "password has expired", nil))
		return nil, domain.ErrPasswordExpired
	}

	// Check if password reset is required by admin (P0)
	if user.PasswordResetRequired {
		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "password_reset_required_by_admin", input.IPAddress, input.UserAgent, "", false, "admin forced password reset", nil))
		return nil, domain.ErrPasswordResetRequired
	}

	// Check 2FA if enabled
	if user.TwoFactorEnabled && input.TwoFACode == "" {
		return nil, domain.Err2FARequired
	}

	// Risk Assessment (P0)
	risk, currentLoc, err := uc.riskService.AssessLoginRisk(ctx, user, input.IPAddress)
	if err != nil {
		fmt.Printf("warning: risk assessment failed: %v\n", err)
	}

	if risk != nil && risk.IsBlocked {
		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "login_blocked", input.IPAddress, input.UserAgent, "", false, "high risk login blocked", map[string]interface{}{"reasons": risk.Reasons}))
		return nil, domain.ErrForbidden
	}

	// Geofencing Check (P2)
	if currentLoc != nil {
		if err := uc.riskService.VerifyGeofencing(ctx, user.TenantID, currentLoc.Country); err != nil {
			_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "login_geofenced", input.IPAddress, input.UserAgent, "", false, "geofencing restriction", map[string]interface{}{"country": currentLoc.Country}))
			return nil, err
		}
	}

	// 2FA requirement (if risk is medium)
	isMFAForced := risk != nil && risk.RequireMFA

	// Check 2FA if enabled or forced by risk
	if (user.TwoFactorEnabled || isMFAForced) && input.TwoFACode == "" {
		return nil, domain.Err2FARequired
	}

	// Verify 2FA code (TOTP or Backup Code)
	if (user.TwoFactorEnabled || isMFAForced) && input.TwoFACode != "" {
		// 1. Try TOTP first
		valid, err := uc.totpService.Verify(input.TwoFACode, user.TwoFactorSecret)
		if err == nil && valid {
			// TOTP valid - proceed
		} else {
			// 2. Try Backup Codes
			activeCodes, err := uc.backupCodeRepo.GetActiveByUserID(ctx, user.TenantID, user.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to verify backup code: %w", err)
			}

			found := false
			for _, bc := range activeCodes {
				valid, err := uc.passwordHasher.Verify(input.TwoFACode, bc.CodeHash)
				if err == nil && valid {
					// Mark as used
					if err := uc.backupCodeRepo.MarkAsUsed(ctx, bc.ID); err != nil {
						return nil, fmt.Errorf("failed to use backup code: %w", err)
					}
					found = true
					break
				}
			}

			if !found {
				_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "login_failed_2fa", input.IPAddress, input.UserAgent, "", false, "invalid 2FA code", nil))

				event := domain.NewEvent(user.TenantID, domain.EventLoginFailed, user.ID, user.Email, map[string]interface{}{
					"reason": "invalid_2fa_code",
					"ip":     input.IPAddress,
				})
				_ = uc.notifier.Publish(ctx, event)

				return nil, domain.ErrInvalid2FACode
			}
		}
	}

	country := "XX"
	if currentLoc != nil {
		country = currentLoc.Country
	}

	// Create session
	inactivityTTL := time.Duration(uc.config.Security.SessionInactivityDays) * 24 * time.Hour
	session := domain.NewSession(domain.NewSessionInput{
		TenantID:      user.TenantID,
		UserID:        user.ID,
		IPAddress:     input.IPAddress,
		Country:       country,
		Device:        input.Device,
		UserAgent:     input.UserAgent,
		InactivityTTL: inactivityTTL,
	})

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate JWT
	token := domain.NewToken(user.TenantID, user.ID, user.Email, uc.config.JWT.AccessExpiry)
	token.JTI = session.ID // Correlate token with session

	// Map roles and permissions to token
	for _, role := range user.Roles {
		token.Roles = append(token.Roles, role.Name)
		for _, perm := range role.Permissions {
			token.Permissions = append(token.Permissions, perm.Name)
		}
	}

	accessToken, err := uc.jwtService.Generate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Generate refresh token
	refreshTokenStr, _ := uc.tokenGen.GenerateSecureToken(32)
	refreshTokenHash := hashToken(refreshTokenStr)

	refreshToken := domain.NewRefreshToken(user.TenantID, user.ID, session.ID, refreshTokenHash, uc.config.JWT.RefreshExpiry)
	if err := uc.refreshRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	// Save previous country before updating
	previousCountry := user.LastLoginCountry

	// Prepare response before async operations
	response := &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		User:         user,
	}

	// Async operations (non-critical, fire-and-forget)
	// These operations don't block the response for better performance
	go func() {
		// Use background context to avoid cancellation when request ends
		bgCtx := context.Background()

		// Update user last login
		var lat, lon *float64
		if currentLoc != nil {
			lat = &currentLoc.Latitude
			lon = &currentLoc.Longitude
		}
		user.UpdateLastLogin(input.IPAddress, country, lat, lon)
		_ = uc.userRepo.Update(bgCtx, user)

		// Audit log
		_ = uc.auditRepo.Create(bgCtx, domain.NewAuditLogEntry(user.TenantID, user.ID, "login", input.IPAddress, input.UserAgent, country, true, "", nil))

		// Login Success Event
		successEvent := domain.NewEvent(user.TenantID, domain.EventLoginSuccess, user.ID, user.Email, map[string]interface{}{
			"ip":      input.IPAddress,
			"country": country,
			"device":  input.Device,
		})
		_ = uc.notifier.Publish(bgCtx, successEvent)

		// Check for new country and send notification
		if previousCountry != "" && country != previousCountry {
			event := domain.NewEvent(user.TenantID, domain.EventLoginNewCountry, user.ID, user.Email, map[string]interface{}{
				"ip":      input.IPAddress,
				"country": country,
			})
			_ = uc.notifier.Publish(bgCtx, event)
		}
	}()

	return response, nil
}

type RegisterInput struct {
	TenantID  string
	Username  string
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*domain.User, error) {
	// Auto-Translate Slug to UUID
	tenant, err := uc.tenantRepo.GetBySlug(ctx, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup tenant: %w", err)
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant with slug %s not found", input.TenantID)
	}
	// Override the textual slug with the database UUID before creating users
	input.TenantID = tenant.ID

	// Rate limiting - fail-safe: if rate limiter fails, reject request to prevent abuse
	rateLimitKey := "register:" + input.IPAddress
	exceeded, err := uc.rateLimiter.CheckLimit(ctx, rateLimitKey, uc.config.RateLimit.RegisterAttempts, uc.config.RateLimit.RegisterWindow)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}
	if exceeded {
		return nil, domain.ErrRateLimitExceeded
	}

	// Validate username
	if err := domain.ValidateUsername(input.Username); err != nil {
		return nil, err
	}

	// Validate email
	if err := domain.ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	// Validate password
	if err := domain.ValidatePassword(input.Password); err != nil {
		return nil, err
	}

	// Check if username exists
	existingUsername, _ := uc.userRepo.GetByUsername(ctx, input.TenantID, input.Username)
	if existingUsername != nil {
		return nil, domain.ErrUsernameAlreadyExists
	}

	// Check if email exists
	existingEmail, _ := uc.userRepo.GetByEmail(ctx, input.TenantID, input.Email)
	if existingEmail != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	// Hash password
	passwordHash, err := uc.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := domain.NewUser(domain.NewUserInput{
		TenantID: input.TenantID,
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.userRepo.Create(ctx, user, passwordHash); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// =========================================================================
	// Asignación de rol por defecto (Default Role ID)
	// =========================================================================
	if tenant.Settings.DefaultRoleID != "" {
		// Asignar el rol al usuario que acaba de registrarse
		// ignorando errores silenciosamente para no abortar el registro si el rol no existe
		_ = uc.roleRepo.AssignRoleToUser(ctx, input.TenantID, user.ID, tenant.Settings.DefaultRoleID)

		// Recargar los roles del usuario (para asegurar que el JWT los lleve si hay login post-registro y por logs)
		if roles, err := uc.roleRepo.GetUserRoles(ctx, input.TenantID, user.ID); err == nil {
			user.Roles = roles
		}
	}

	// Async operations (non-critical, fire-and-forget)
	// These operations don't block the response for better performance
	go func() {
		// Use background context to avoid cancellation when request ends
		bgCtx := context.Background()

		// Increment rate limit
		if _, err := uc.rateLimiter.Increment(bgCtx, rateLimitKey, uc.config.RateLimit.RegisterWindow); err != nil {
			fmt.Printf("warning: failed to increment register rate limit for %s: %v\n", input.IPAddress, err)
		}

		// Get location for audit log
		var currentLoc *domain.Geolocation
		if uc.geoService != nil {
			currentLoc, _ = uc.geoService.GetLocation(bgCtx, input.IPAddress)
		}
		country := "XX"
		if currentLoc != nil {
			country = currentLoc.Country
		}

		// Audit log
		_ = uc.auditRepo.Create(bgCtx, domain.NewAuditLogEntry(user.TenantID, user.ID, "register", input.IPAddress, input.UserAgent, country, true, "", nil))

		// Send welcome email (can be slow)
		_ = uc.emailService.SendWelcome(bgCtx, user.Email, user.Username)

		// Publish event
		event := domain.NewEvent(user.TenantID, domain.EventUserRegistered, user.ID, user.Email, nil)
		_ = uc.notifier.Publish(bgCtx, event)
	}()

	return user, nil
}

func (uc *AuthUseCase) ForgotPassword(ctx context.Context, tenantID, email, ipAddress string) error {
	user, err := uc.userRepo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Generate reset token y código de 6 dígitos
	resetTokenStr, _ := uc.tokenGen.GenerateSecureToken(32)

	// Generar código de 6 dígitos
	codeGen := &CodeGenerator{}
	code, _ := codeGen.GenerateNumericCode(6)

	resetToken := domain.NewPasswordResetToken(user.TenantID, user.ID, resetTokenStr, code)

	if err := uc.resetRepo.Create(ctx, resetToken); err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// Async operations (non-critical, fire-and-forget)
	// These operations don't block the response for better performance
	go func() {
		// Use background context to avoid cancellation when request ends
		bgCtx := context.Background()

		// Send reset email con código y URL (can be slow)
		resetURL := BuildURL(uc.config.Server.Environment, uc.config.Server.BaseDomain, fmt.Sprintf("/api/v1/auth/reset-password?token=%s", resetTokenStr))
		_ = uc.emailService.SendPasswordReset(bgCtx, user.Email, code, resetURL)

		// Audit log
		_ = uc.auditRepo.Create(bgCtx, domain.NewAuditLogEntry(user.TenantID, user.ID, "forgot_password", ipAddress, "", "", true, "", nil))
	}()

	return nil
}

type CodeGenerator struct{}

func (g *CodeGenerator) GenerateNumericCode(digits int) (string, error) {
	if digits <= 0 || digits > 10 {
		return "", fmt.Errorf("digits must be between 1 and 10")
	}

	// Calculate maximum value for the given number of digits (e.g., 6 digits = 999999)
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	max.Sub(max, big.NewInt(1)) // 999999 for 6 digits

	// Generate a random number in the range [0, max]
	n, err := rand.Int(rand.Reader, max.Add(max, big.NewInt(1)))
	if err != nil {
		return "", fmt.Errorf("failed to generate random code: %w", err)
	}

	// Format with leading zeros to ensure the correct number of digits
	format := fmt.Sprintf("%%0%dd", digits)
	return fmt.Sprintf(format, n), nil
}

// ResetPasswordWithToken resetea la contraseña usando el token URL
func (uc *AuthUseCase) ResetPasswordWithToken(ctx context.Context, tenantID, token, newPassword, ipAddress string) error {
	// Validate password
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Get reset token
	resetToken, err := uc.resetRepo.GetByToken(ctx, tenantID, token)
	if err != nil {
		return domain.ErrInvalidResetToken
	}

	if !resetToken.IsValid() {
		return domain.ErrResetTokenExpired
	}

	return uc.completePasswordReset(ctx, resetToken, newPassword, ipAddress)
}

// ResetPasswordWithCode resetea la contraseña usando el código de 6 dígitos
func (uc *AuthUseCase) ResetPasswordWithCode(ctx context.Context, tenantID, email, code, newPassword, ipAddress string) error {
	// Validate password
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Get user by email
	user, err := uc.userRepo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		return domain.ErrInvalidResetToken
	}

	// Get reset token by code
	resetToken, err := uc.resetRepo.GetByCode(ctx, user.TenantID, user.ID, code)
	if err != nil {
		return domain.ErrInvalidResetToken
	}

	if !resetToken.IsValid() {
		return domain.ErrResetTokenExpired
	}

	return uc.completePasswordReset(ctx, resetToken, newPassword, ipAddress)
}

func (uc *AuthUseCase) completePasswordReset(ctx context.Context, resetToken *domain.PasswordResetToken, newPassword, ipAddress string) error {
	// Hash new password
	passwordHash, err := uc.passwordHasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 3. Update password in repository
	err = uc.userRepo.UpdatePassword(ctx, resetToken.TenantID, resetToken.UserID, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Flag password_reset_required is cleared automatically in UpdatePassword repo method

	// 4. Mark token as used
	_ = uc.resetRepo.MarkAsUsed(ctx, resetToken.TenantID, resetToken.ID)

	// 5. Revoke all sessions (security measure)
	_ = uc.sessionRepo.RevokeAllByUserID(ctx, resetToken.TenantID, resetToken.UserID, "system", "password_reset")
	_ = uc.refreshRepo.RevokeByUserID(ctx, resetToken.TenantID, resetToken.UserID)

	// Async operations (non-critical, fire-and-forget)
	go func() {
		// Use background context to avoid cancellation when request ends
		bgCtx := context.Background()

		// Get user for event
		user, _ := uc.userRepo.GetByID(bgCtx, resetToken.TenantID, resetToken.UserID)

		// Audit log
		_ = uc.auditRepo.Create(bgCtx, domain.NewAuditLogEntry(resetToken.TenantID, resetToken.UserID, "reset_password", ipAddress, "", "", true, "", nil))

		// Publish event
		if user != nil {
			event := domain.NewEvent(user.TenantID, domain.EventPasswordChanged, user.ID, user.Email, nil)
			_ = uc.notifier.Publish(bgCtx, event)
		}
	}()

	return nil
}

func (uc *AuthUseCase) NotifyExpiringPasswords(ctx context.Context) (int, error) {
	// 1. Get configuration
	expiry := uc.config.Security.PasswordExpiry
	if expiry.ExpiryDays <= 0 || expiry.WarningDays <= 0 {
		return 0, nil // Disabled
	}

	// 2. Calculate threshold for query
	// We want users whose password was changed at least (ExpiryDays - WarningDays) ago
	thresholdDays := expiry.ExpiryDays - expiry.WarningDays

	// 3. Get users
	users, err := uc.userRepo.GetExpiringPasswords(ctx, thresholdDays)
	if err != nil {
		return 0, fmt.Errorf("failed to get expiring passwords: %w", err)
	}

	// 4. Send notifications
	notifiedCount := 0
	for _, user := range users {
		// Calculate precise days remaining
		daysUsed := int(time.Since(user.PasswordChangedAt).Hours() / 24)
		daysRemaining := expiry.ExpiryDays - daysUsed

		// Only notify if within context deadline
		select {
		case <-ctx.Done():
			return notifiedCount, ctx.Err()
		default:
			// Send email
			err := uc.emailService.SendPasswordExpiryWarning(ctx, user.Email, user.Username, daysRemaining)
			if err != nil {
				// Log error but continue with others
				fmt.Printf("failed to send expiry warning to %s: %v\n", user.Email, err)
				continue
			}

			// Audit log
			_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "password_expiry_warning_sent", "", "", "", true, "", map[string]interface{}{
				"days_remaining": daysRemaining,
			}))

			notifiedCount++
		}
	}

	return notifiedCount, nil
}

func (uc *AuthUseCase) ForcePasswordReset(ctx context.Context, tenantID, userID string) error {
	// 1. Get user
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	// 2. Set flag
	user.PasswordResetRequired = true
	user.UpdatedAt = time.Now()

	// 3. Update user
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to force password reset: %w", err)
	}

	// 4. Revoke all sessions
	if err := uc.sessionRepo.RevokeAllByUserID(ctx, user.TenantID, userID, "system", "forced password reset by admin"); err != nil {
		fmt.Printf("warning: failed to revoke sessions after forced reset: %v\n", err)
	}

	// 5. Revoke all refresh tokens
	if err := uc.refreshRepo.RevokeByUserID(ctx, user.TenantID, userID); err != nil {
		fmt.Printf("warning: failed to revoke refresh tokens after forced reset: %v\n", err)
	}

	// 6. Audit log
	_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, userID, "forced_password_reset", "", "", "", true, "admin forced password reset", nil))

	return nil
}

func (uc *AuthUseCase) OAuthLogin(ctx context.Context, tenantID, provider, code, state, ipAddress, userAgent, device string) (*LoginResponse, error) {
	oauthProvider, exists := uc.oauthProviders[provider]
	if !exists {
		return nil, domain.ErrOAuthProviderNotFound
	}

	// Exchange code for user info
	userInfo, err := oauthProvider.Exchange(ctx, code)
	if err != nil {
		return nil, domain.ErrOAuthCodeInvalid
	}

	// Check if user exists
	user, err := uc.userRepo.GetByOAuthProvider(ctx, tenantID, provider, userInfo.ProviderID)
	if err != nil {
		// Create new user
		newUser, err := domain.NewUser(domain.NewUserInput{
			TenantID:        tenantID,
			Email:           userInfo.Email,
			OAuthProvider:   provider,
			OAuthProviderID: userInfo.ProviderID,
		})
		if err != nil {
			return nil, err
		}

		if err := uc.userRepo.Create(ctx, newUser, ""); err != nil {
			return nil, fmt.Errorf("failed to create OAuth user: %w", err)
		}

		user = newUser
	}

	// Generar username del email si es OAuth (antes del @ )
	if user.Username == "" && user.OAuthProvider != "" {
		// Extraer la parte antes del @
		parts := strings.Split(userInfo.Email, "@")
		if len(parts) > 0 {
			user.Username = parts[0]
		}
	}

	// Continue with normal login flow
	risk, currentLoc, err := uc.riskService.AssessLoginRisk(ctx, user, ipAddress)
	if err != nil {
		fmt.Printf("warning: risk assessment failed: %v\n", err)
	}

	if risk != nil && risk.IsBlocked {
		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "oauth_login_blocked", ipAddress, userAgent, "", false, "high risk oauth login blocked", map[string]interface{}{"reasons": risk.Reasons}))
		return nil, domain.ErrForbidden
	}

	country := "XX"
	if currentLoc != nil {
		country = currentLoc.Country
	}

	inactivityTTL := time.Duration(uc.config.Security.SessionInactivityDays) * 24 * time.Hour
	session := domain.NewSession(domain.NewSessionInput{
		UserID:        user.ID,
		IPAddress:     ipAddress,
		Country:       country,
		Device:        device,
		UserAgent:     userAgent,
		InactivityTTL: inactivityTTL,
	})

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	token := domain.NewToken(user.TenantID, user.ID, user.Email, uc.config.JWT.AccessExpiry)
	token.JTI = session.ID // Correlate token with session

	// Map roles and permissions to token
	for _, role := range user.Roles {
		token.Roles = append(token.Roles, role.Name)
		for _, perm := range role.Permissions {
			token.Permissions = append(token.Permissions, perm.Name)
		}
	}

	accessToken, err := uc.jwtService.Generate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	refreshTokenStr, _ := uc.tokenGen.GenerateSecureToken(32)
	refreshTokenHash := hashToken(refreshTokenStr)

	refreshToken := domain.NewRefreshToken(user.TenantID, user.ID, session.ID, refreshTokenHash, uc.config.JWT.RefreshExpiry)
	if err := uc.refreshRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	// Save previous country before updating
	previousCountry := user.LastLoginCountry

	// Prepare response before async operations
	response := &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		User:         user,
	}

	// Async operations (non-critical, fire-and-forget)
	go func() {
		// Use background context to avoid cancellation when request ends
		bgCtx := context.Background()

		// Update user last login
		var lat, lon *float64
		if currentLoc != nil {
			lat = &currentLoc.Latitude
			lon = &currentLoc.Longitude
		}
		user.UpdateLastLogin(ipAddress, country, lat, lon)
		_ = uc.userRepo.Update(bgCtx, user)

		// Check for new country and send notification
		if previousCountry != "" && country != previousCountry {
			event := domain.NewEvent(user.TenantID, domain.EventLoginNewCountry, user.ID, user.Email, map[string]interface{}{
				"ip":      ipAddress,
				"country": country,
			})
			_ = uc.notifier.Publish(bgCtx, event)
		}
	}()

	return response, nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GetUserInfo returns user information for OIDC UserInfo endpoint
func (uc *AuthUseCase) GetUserInfo(ctx context.Context, tenantID, userID string) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Role Management

func (uc *AuthUseCase) CreateRole(ctx context.Context, tenantID, name, description string) error {
	role := domain.NewRole(name, description)
	role.TenantID = tenantID
	return uc.roleRepo.CreateRole(ctx, role)
}

func (uc *AuthUseCase) ListRoles(ctx context.Context, tenantID string) ([]*domain.Role, error) {
	return uc.roleRepo.ListRoles(ctx, tenantID)
}

func (uc *AuthUseCase) CreatePermission(ctx context.Context, tenantID, name, description string) error {
	perm := domain.NewPermission(name, description)
	perm.TenantID = tenantID
	return uc.roleRepo.CreatePermission(ctx, perm)
}

func (uc *AuthUseCase) ListPermissions(ctx context.Context, tenantID string) ([]*domain.Permission, error) {
	return uc.roleRepo.ListPermissions(ctx, tenantID)
}

func (uc *AuthUseCase) AssignRoleToUser(ctx context.Context, tenantID, userID, roleID string) error {
	return uc.roleRepo.AssignRoleToUser(ctx, tenantID, userID, roleID)
}

func (uc *AuthUseCase) AddPermissionToRole(ctx context.Context, tenantID, roleID, permID string) error {
	return uc.roleRepo.AddPermissionToRole(ctx, tenantID, roleID, permID)
}

type PasswordlessLoginInput struct {
	IPAddress string
	UserAgent string
	Device    string
}

func (uc *AuthUseCase) PasswordlessLogin(ctx context.Context, user *domain.User, input PasswordlessLoginInput) (*LoginResponse, error) {
	// Check if user is active
	if !user.Active {
		return nil, domain.ErrUserInactive
	}

	// Check if account is locked
	if user.IsLocked() {
		return nil, domain.ErrAccountLocked
	}

	// Risk Assessment
	risk, currentLoc, err := uc.riskService.AssessLoginRisk(ctx, user, input.IPAddress)
	if err != nil {
		fmt.Printf("warning: risk assessment failed: %v\n", err)
	}

	if risk != nil && risk.IsBlocked {
		_ = uc.auditRepo.Create(ctx, domain.NewAuditLogEntry(user.TenantID, user.ID, "passwordless_login_blocked", input.IPAddress, input.UserAgent, "", false, "high risk login blocked", map[string]interface{}{"reasons": risk.Reasons}))
		return nil, domain.ErrForbidden
	}

	country := "XX"
	if currentLoc != nil {
		country = currentLoc.Country
	}

	// Create session
	inactivityTTL := time.Duration(uc.config.Security.SessionInactivityDays) * 24 * time.Hour
	session := domain.NewSession(domain.NewSessionInput{
		UserID:        user.ID,
		IPAddress:     input.IPAddress,
		Country:       country,
		Device:        input.Device,
		UserAgent:     input.UserAgent,
		InactivityTTL: inactivityTTL,
	})

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate JWT
	token := domain.NewToken(user.TenantID, user.ID, user.Email, uc.config.JWT.AccessExpiry)
	token.JTI = session.ID // Correlate token with session

	// Map roles and permissions to token
	for _, role := range user.Roles {
		token.Roles = append(token.Roles, role.Name)
		for _, perm := range role.Permissions {
			token.Permissions = append(token.Permissions, perm.Name)
		}
	}

	accessToken, err := uc.jwtService.Generate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Generate refresh token
	refreshTokenStr, _ := uc.tokenGen.GenerateSecureToken(32)
	refreshTokenHash := hashToken(refreshTokenStr)

	refreshToken := domain.NewRefreshToken(user.TenantID, user.ID, session.ID, refreshTokenHash, uc.config.JWT.RefreshExpiry)
	if err := uc.refreshRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	// Save previous country
	previousCountry := user.LastLoginCountry

	// Prepare response
	response := &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		User:         user,
	}

	// Async updates
	go func() {
		bgCtx := context.Background()
		var lat, lon *float64
		if currentLoc != nil {
			lat = &currentLoc.Latitude
			lon = &currentLoc.Longitude
		}
		user.UpdateLastLogin(input.IPAddress, country, lat, lon)
		_ = uc.userRepo.Update(bgCtx, user)
		_ = uc.auditRepo.Create(bgCtx, domain.NewAuditLogEntry(user.TenantID, user.ID, "webauthn_login", input.IPAddress, input.UserAgent, country, true, "", nil))

		if previousCountry != "" && country != previousCountry {
			event := domain.NewEvent(user.TenantID, domain.EventLoginNewCountry, user.ID, user.Email, map[string]interface{}{
				"ip":      input.IPAddress,
				"country": country,
			})
			_ = uc.notifier.Publish(bgCtx, event)
		}
	}()

	return response, nil
}

// GetUserByEmail abstractly returns the User entity strictly from its Email
func (uc *AuthUseCase) GetUserByEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	return uc.userRepo.GetByEmail(ctx, tenantID, email)
}

// GetResendRateLimitConfig returns the specific Rate Limit config for Resend Email
func (uc *AuthUseCase) GetResendRateLimitConfig() (int, time.Duration) {
	return uc.config.RateLimit.ResendAttempts, uc.config.RateLimit.ResendWindow
}

// ResendVerificationEmail checks rate limits and then returns the targeted user
func (uc *AuthUseCase) ResendVerificationEmail(ctx context.Context, tenantID, email, ipAddress string) (*domain.User, error) {
	// Rate limiting - Protect Resend Endpoint per user email (e.g. 4 emails per Hour)
	rateLimitKey := fmt.Sprintf("resend:%s:%s", tenantID, email)
	exceeded, err := uc.rateLimiter.CheckLimit(ctx, rateLimitKey, uc.config.RateLimit.ResendAttempts, uc.config.RateLimit.ResendWindow)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}
	if exceeded {
		return nil, domain.ErrRateLimitExceeded
	}

	// Fetch user ID disconnected from JWT Context
	user, err := uc.userRepo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// All checks passed, we can Increment the Rate Limiter
	if _, err := uc.rateLimiter.Increment(context.Background(), rateLimitKey, uc.config.RateLimit.ResendWindow); err != nil {
		fmt.Printf("warning: failed to increment resend rate limit: %v\n", err)
	}

	// We return the user struct to the handler so it can trigger the actual verification email
	return user, nil
}
