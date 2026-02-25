package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User, passwordHash string) error {
	query := `
		INSERT INTO auth_users (
			id, username, email, password_hash, active, email_verified,
			two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			created_at, updated_at, password_changed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	// Convert empty strings to NULL for optional fields
	var twoFactorSecret, oauthProvider, oauthProviderID interface{}

	if user.TwoFactorSecret == "" {
		twoFactorSecret = nil
	} else {
		twoFactorSecret = user.TwoFactorSecret
	}

	if user.OAuthProvider == "" {
		oauthProvider = nil
	} else {
		oauthProvider = user.OAuthProvider
	}

	if user.OAuthProviderID == "" {
		oauthProviderID = nil
	} else {
		oauthProviderID = user.OAuthProviderID
	}

	_, err := r.db.Exec(ctx, query,
		user.ID, user.Username, user.Email, passwordHash, user.Active, user.EmailVerified,
		user.TwoFactorEnabled, twoFactorSecret, oauthProvider, oauthProviderID,
		user.CreatedAt, user.UpdatedAt, user.CreatedAt, // password_changed_at starts at creation time
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   last_login_latitude, last_login_longitude, password_changed_at
		FROM auth_users
		WHERE id = $1
	`

	user := &domain.User{}
	var lastLoginAt *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string
	var lastLoginLatitude, lastLoginLongitude *float64

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
		&lastLoginLatitude, &lastLoginLongitude, &user.PasswordChangedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	// Map optional fields
	user.LastLoginAt = lastLoginAt
	if twoFactorSecret != nil {
		user.TwoFactorSecret = *twoFactorSecret
	}
	if oauthProvider != nil {
		user.OAuthProvider = *oauthProvider
	}
	if oauthProviderID != nil {
		user.OAuthProviderID = *oauthProviderID
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginCountry != nil {
		user.LastLoginCountry = *lastLoginCountry
	}
	user.LastLoginLatitude = lastLoginLatitude
	user.LastLoginLongitude = lastLoginLongitude

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   password_changed_at
		FROM auth_users
		WHERE email = $1
	`

	user := &domain.User{}
	var lastLoginAt *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Map optional fields
	user.LastLoginAt = lastLoginAt
	if twoFactorSecret != nil {
		user.TwoFactorSecret = *twoFactorSecret
	}
	if oauthProvider != nil {
		user.OAuthProvider = *oauthProvider
	}
	if oauthProviderID != nil {
		user.OAuthProviderID = *oauthProviderID
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginCountry != nil {
		user.LastLoginCountry = *lastLoginCountry
	}

	return user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   password_changed_at
		FROM auth_users
		WHERE username = $1
	`

	user := &domain.User{}
	var lastLoginAt *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string

	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	// Map optional fields
	user.LastLoginAt = lastLoginAt
	if twoFactorSecret != nil {
		user.TwoFactorSecret = *twoFactorSecret
	}
	if oauthProvider != nil {
		user.OAuthProvider = *oauthProvider
	}
	if oauthProviderID != nil {
		user.OAuthProviderID = *oauthProviderID
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginCountry != nil {
		user.LastLoginCountry = *lastLoginCountry
	}

	return user, nil
}

func (r *UserRepository) GetByEmailOrUsername(ctx context.Context, identifier string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, active, email_verified,
			two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			last_login_latitude, last_login_longitude, failed_login_attempts, locked_until,
			password_changed_at
		FROM auth_users
		WHERE email = $1 OR username = $1
	`

	user := &domain.User{}
	var lastLoginAt, lockedUntil *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string
	var lastLoginLatitude, lastLoginLongitude *float64

	err := r.db.QueryRow(ctx, query, identifier).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
		&lastLoginLatitude, &lastLoginLongitude, &user.FailedLoginAttempts, &lockedUntil, &user.PasswordChangedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email or username: %w", err)
	}

	// Map optional fields
	user.LastLoginAt = lastLoginAt
	user.LockedUntil = lockedUntil
	if twoFactorSecret != nil {
		user.TwoFactorSecret = *twoFactorSecret
	}
	if oauthProvider != nil {
		user.OAuthProvider = *oauthProvider
	}
	if oauthProviderID != nil {
		user.OAuthProviderID = *oauthProviderID
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginCountry != nil {
		user.LastLoginCountry = *lastLoginCountry
	}
	user.LastLoginLatitude = lastLoginLatitude
	user.LastLoginLongitude = lastLoginLongitude

	return user, nil
}

func (r *UserRepository) GetByOAuthProvider(ctx context.Context, provider, providerID string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   password_changed_at
		FROM auth_users
		WHERE oauth_provider = $1 AND oauth_provider_id = $2
	`

	user := &domain.User{}
	var lastLoginAt *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string

	err := r.db.QueryRow(ctx, query, provider, providerID).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by OAuth provider: %w", err)
	}

	// Map optional fields
	user.LastLoginAt = lastLoginAt
	if twoFactorSecret != nil {
		user.TwoFactorSecret = *twoFactorSecret
	}
	if oauthProvider != nil {
		user.OAuthProvider = *oauthProvider
	}
	if oauthProviderID != nil {
		user.OAuthProviderID = *oauthProviderID
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}
	if lastLoginCountry != nil {
		user.LastLoginCountry = *lastLoginCountry
	}

	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE auth_users
		SET username = $2, email = $3, active = $4, email_verified = $5,
			two_factor_enabled = $6, two_factor_secret = $7,
			updated_at = $8, last_login_at = $9, last_login_ip = $10, last_login_country = $11,
			last_login_latitude = $12, last_login_longitude = $13,
			failed_login_attempts = $14, locked_until = $15
		WHERE id = $1
	`

	// Convert empty strings to NULL for optional fields
	var twoFactorSecret, lastLoginIP, lastLoginCountry interface{}

	if user.TwoFactorSecret == "" {
		twoFactorSecret = nil
	} else {
		twoFactorSecret = user.TwoFactorSecret
	}

	if user.LastLoginIP == "" {
		lastLoginIP = nil
	} else {
		lastLoginIP = user.LastLoginIP
	}

	if user.LastLoginCountry == "" {
		lastLoginCountry = nil
	} else {
		lastLoginCountry = user.LastLoginCountry
	}

	_, err := r.db.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.Active, user.EmailVerified,
		user.TwoFactorEnabled, twoFactorSecret,
		user.UpdatedAt, user.LastLoginAt, lastLoginIP, lastLoginCountry,
		user.LastLoginLatitude, user.LastLoginLongitude,
		user.FailedLoginAttempts, user.LockedUntil,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	query := `
		UPDATE auth_users
		SET password_hash = $2, password_changed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (r *UserRepository) GetExpiringPasswords(ctx context.Context, thresholdDays int) ([]*domain.User, error) {
	// Query users whose password will expire in thresholdDays or less
	query := `
		SELECT id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   last_login_latitude, last_login_longitude, failed_login_attempts, locked_until,
			   password_changed_at
		FROM auth_users
		WHERE oauth_provider IS NULL
		  AND password_changed_at + ($1 * interval '1 day') <= NOW()
	`

	rows, err := r.db.Query(ctx, query, thresholdDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring passwords: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		var lastLoginAt, lockedUntil *time.Time
		var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string
		var lastLoginLatitude, lastLoginLongitude *float64

		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
			&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
			&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
			&lastLoginLatitude, &lastLoginLongitude, &user.FailedLoginAttempts, &lockedUntil, &user.PasswordChangedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Map optional fields
		user.LastLoginAt = lastLoginAt
		user.LockedUntil = lockedUntil
		if twoFactorSecret != nil {
			user.TwoFactorSecret = *twoFactorSecret
		}
		if oauthProvider != nil {
			user.OAuthProvider = *oauthProvider
		}
		if oauthProviderID != nil {
			user.OAuthProviderID = *oauthProviderID
		}
		if lastLoginIP != nil {
			user.LastLoginIP = *lastLoginIP
		}
		if lastLoginCountry != nil {
			user.LastLoginCountry = *lastLoginCountry
		}
		user.LastLoginLatitude = lastLoginLatitude
		user.LastLoginLongitude = lastLoginLongitude

		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM auth_users WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
