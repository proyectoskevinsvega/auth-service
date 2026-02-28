package domain

import (
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)

type User struct {
	ID                    string
	TenantID              string
	Username              string
	Email                 string
	PasswordHash          string
	Active                bool
	EmailVerified         bool
	TwoFactorEnabled      bool
	TwoFactorSecret       string
	OAuthProvider         string // "google", "github", or empty for email/password
	OAuthProviderID       string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	LastLoginAt           *time.Time
	LastLoginIP           string
	LastLoginCountry      string
	LastLoginLatitude     *float64
	LastLoginLongitude    *float64
	FailedLoginAttempts   int
	LockedUntil           *time.Time
	PasswordChangedAt     time.Time
	PasswordResetRequired bool
	Roles                 []Role                 // Added for RBAC
	Attributes            map[string]interface{} // Added for ABAC
	WebAuthnID            []byte                 // Added for WebAuthn identifier
}

type NewUserInput struct {
	TenantID        string
	Username        string
	Email           string
	Password        string
	OAuthProvider   string
	OAuthProviderID string
}

func NewUser(input NewUserInput) (*User, error) {
	if err := ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	if err := ValidateUsername(input.Username); err != nil {
		return nil, err
	}

	if input.OAuthProvider == "" && input.Password == "" {
		return nil, ErrInvalidPassword
	}

	now := time.Now()
	return &User{
		ID:              uuid.Must(uuid.NewV7()).String(),
		TenantID:        input.TenantID,
		Username:        input.Username,
		Email:           input.Email,
		Active:          true,
		EmailVerified:   input.OAuthProvider != "", // Auto-verify OAuth users
		OAuthProvider:   input.OAuthProvider,
		OAuthProviderID: input.OAuthProviderID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Roles:           []Role{},
		Attributes:      make(map[string]interface{}),
	}, nil
}

func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func ValidateUsername(username string) error {
	if username == "" {
		return ErrInvalidInput
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must be 3-30 characters (letters, numbers, hyphens and underscores)")
	}
	return nil
}

func ValidatePassword(password string) error {
	// Trim spaces for validation
	trimmedPassword := password

	// Check for leading/trailing spaces
	if password != trimmedPassword {
		return fmt.Errorf("password cannot have leading or trailing spaces")
	}

	// Check minimum length
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Check for at least one uppercase letter
	hasUpper := false
	for _, c := range password {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	// Check for at least one lowercase letter
	hasLower := false
	for _, c := range password {
		if c >= 'a' && c <= 'z' {
			hasLower = true
			break
		}
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	// Check for at least one digit
	hasDigit := false
	for _, c := range password {
		if c >= '0' && c <= '9' {
			hasDigit = true
			break
		}
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one number")
	}

	// Check for at least one special character
	hasSpecial := false
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?/"
	for _, c := range password {
		for _, s := range specialChars {
			if c == s {
				hasSpecial = true
				break
			}
		}
		if hasSpecial {
			break
		}
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character (!@#$%%^&*()_+-=[]{}|;:,.<>?/)")
	}

	return nil
}

func (u *User) UpdateLastLogin(ip, country string, lat, lon *float64) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
	u.LastLoginCountry = country
	u.LastLoginLatitude = lat
	u.LastLoginLongitude = lon
	u.UpdatedAt = now
}

func (u *User) Enable2FA(secret string) {
	u.TwoFactorEnabled = true
	u.TwoFactorSecret = secret
	u.UpdatedAt = time.Now()
}

func (u *User) Disable2FA() {
	u.TwoFactorEnabled = false
	u.TwoFactorSecret = ""
	u.UpdatedAt = time.Now()
}

func (u *User) Deactivate() {
	u.Active = false
	u.UpdatedAt = time.Now()
}

func (u *User) Activate() {
	u.Active = true
	u.UpdatedAt = time.Now()
}

func (u *User) IncrementFailedAttempts(maxAttempts int, baseDuration time.Duration, factor float64, maxDuration time.Duration) {
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= maxAttempts {
		// Calculate progressive delay: base * factor ^ (attempts - max)
		// e.g. 5m * 3^0 = 5m, 5m * 3^1 = 15m, 5m * 3^2 = 45m...
		extraAttempts := float64(u.FailedLoginAttempts - maxAttempts)
		durationSeconds := baseDuration.Seconds() * math.Pow(factor, extraAttempts)
		duration := time.Duration(durationSeconds) * time.Second

		if duration > maxDuration {
			duration = maxDuration
		}

		lockout := time.Now().Add(duration)
		u.LockedUntil = &lockout
	}
	u.UpdatedAt = time.Now()
}

func (u *User) ResetFailedAttempts() {
	if u.FailedLoginAttempts > 0 || u.LockedUntil != nil {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
		u.UpdatedAt = time.Now()
	}
}

func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

func (u *User) IsPasswordExpired(expiryDays int) bool {
	if expiryDays <= 0 {
		return false
	}
	// Password is considered expired if it has been more than expiryDays since PasswordChangedAt
	return time.Now().After(u.PasswordChangedAt.Add(time.Hour * 24 * time.Duration(expiryDays)))
}

func (u *User) HasPermission(permissionName string) bool {
	for _, role := range u.Roles {
		for _, perm := range role.Permissions {
			if perm.Name == permissionName {
				return true
			}
		}
	}
	return false
}

func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}
