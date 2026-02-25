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
			id, tenant_id, username, email, password_hash, active, email_verified,
			two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			created_at, updated_at, password_changed_at, password_reset_required, attributes, webauthn_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
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
		user.ID, user.TenantID, user.Username, user.Email, passwordHash, user.Active, user.EmailVerified,
		user.TwoFactorEnabled, twoFactorSecret, oauthProvider, oauthProviderID,
		user.CreatedAt, user.UpdatedAt, user.CreatedAt, user.PasswordResetRequired, user.Attributes, user.WebAuthnID,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   last_login_latitude, last_login_longitude, failed_login_attempts, locked_until,
			   password_changed_at, password_reset_required, attributes, webauthn_id
		FROM auth_users
		WHERE tenant_id = $1 AND id = $2
	`

	user := &domain.User{}
	var lastLoginAt, lockedUntil *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string
	var lastLoginLatitude, lastLoginLongitude *float64

	err := r.db.QueryRow(ctx, query, tenantID, id).Scan(
		&user.ID, &user.TenantID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
		&lastLoginLatitude, &lastLoginLongitude, &user.FailedLoginAttempts, &lockedUntil,
		&user.PasswordChangedAt, &user.PasswordResetRequired, &user.Attributes, &user.WebAuthnID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
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

	// Load roles
	roles, err := r.getUserRoles(ctx, user.TenantID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user roles: %w", err)
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   failed_login_attempts, locked_until, password_changed_at, password_reset_required, attributes, webauthn_id
		FROM auth_users
		WHERE tenant_id = $1 AND email = $2
	`

	user := &domain.User{}
	var lastLoginAt, lockedUntil *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string

	err := r.db.QueryRow(ctx, query, tenantID, email).Scan(
		&user.ID, &user.TenantID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
		&user.FailedLoginAttempts, &lockedUntil, &user.PasswordChangedAt, &user.PasswordResetRequired, &user.Attributes, &user.WebAuthnID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
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

	// Load roles
	roles, err := r.getUserRoles(ctx, user.TenantID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user roles: %w", err)
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, tenantID, username string) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   failed_login_attempts, locked_until, password_changed_at, password_reset_required, attributes, webauthn_id
		FROM auth_users
		WHERE tenant_id = $1 AND username = $2
	`

	user := &domain.User{}
	var lastLoginAt, lockedUntil *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string

	err := r.db.QueryRow(ctx, query, tenantID, username).Scan(
		&user.ID, &user.TenantID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
		&user.FailedLoginAttempts, &lockedUntil, &user.PasswordChangedAt, &user.PasswordResetRequired, &user.Attributes, &user.WebAuthnID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
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

	// Load roles
	roles, err := r.getUserRoles(ctx, user.TenantID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user roles: %w", err)
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) GetByEmailOrUsername(ctx context.Context, tenantID, identifier string) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, username, email, password_hash, active, email_verified,
			two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			last_login_latitude, last_login_longitude, failed_login_attempts, locked_until,
			password_changed_at, password_reset_required, attributes, webauthn_id
		FROM auth_users
		WHERE tenant_id = $1 AND (email = $2 OR username = $2)
	`

	user := &domain.User{}
	var lastLoginAt, lockedUntil *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string
	var lastLoginLatitude, lastLoginLongitude *float64

	err := r.db.QueryRow(ctx, query, tenantID, identifier).Scan(
		&user.ID, &user.TenantID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
		&lastLoginLatitude, &lastLoginLongitude, &user.FailedLoginAttempts, &lockedUntil, &user.PasswordChangedAt,
		&user.PasswordResetRequired, &user.Attributes, &user.WebAuthnID,
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

	// Load roles
	roles, err := r.getUserRoles(ctx, user.TenantID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user roles: %w", err)
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) GetByOAuthProvider(ctx context.Context, tenantID, provider, providerID string) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, username, email, password_hash, active, email_verified,
			   two_factor_enabled, two_factor_secret, oauth_provider, oauth_provider_id,
			   created_at, updated_at, last_login_at, last_login_ip, last_login_country,
			   failed_login_attempts, locked_until, password_changed_at, password_reset_required, attributes, webauthn_id
		FROM auth_users
		WHERE tenant_id = $1 AND oauth_provider = $2 AND oauth_provider_id = $3
	`

	user := &domain.User{}
	var lastLoginAt, lockedUntil *time.Time
	var twoFactorSecret, oauthProvider, oauthProviderID, lastLoginIP, lastLoginCountry *string

	err := r.db.QueryRow(ctx, query, tenantID, provider, providerID).Scan(
		&user.ID, &user.TenantID, &user.Username, &user.Email, &user.PasswordHash, &user.Active, &user.EmailVerified,
		&user.TwoFactorEnabled, &twoFactorSecret, &oauthProvider, &oauthProviderID,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastLoginIP, &lastLoginCountry,
		&user.FailedLoginAttempts, &lockedUntil, &user.PasswordChangedAt, &user.PasswordResetRequired, &user.Attributes, &user.WebAuthnID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by OAuth provider: %w", err)
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

	// Load roles
	roles, err := r.getUserRoles(ctx, user.TenantID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user roles: %w", err)
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE auth_users
		SET username = $2, email = $3, active = $4, email_verified = $5,
			two_factor_enabled = $6, two_factor_secret = $7,
			updated_at = $8, last_login_at = $9, last_login_ip = $10, last_login_country = $11,
			last_login_latitude = $12, last_login_longitude = $13,
			failed_login_attempts = $14, locked_until = $15, password_reset_required = $16, attributes = $17, webauthn_id = $18
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
		user.FailedLoginAttempts, user.LockedUntil, user.PasswordResetRequired, user.Attributes, user.WebAuthnID,
		user.ID, user.TenantID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, tenantID, userID, newPasswordHash string) error {
	query := `
		UPDATE auth_users
		SET password_hash = $3, password_changed_at = NOW(), updated_at = NOW(), password_reset_required = FALSE
		WHERE tenant_id = $1 AND id = $2
	`

	_, err := r.db.Exec(ctx, query, tenantID, userID, newPasswordHash)
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
			   password_changed_at, password_reset_required
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
			&lastLoginLatitude, &lastLoginLongitude, &user.FailedLoginAttempts, &lockedUntil,
			&user.PasswordChangedAt, &user.PasswordResetRequired,
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

func (r *UserRepository) getUserRoles(ctx context.Context, tenantID, userID string) ([]domain.Role, error) {
	query := `
		SELECT r.id, r.tenant_id, r.name, r.description, r.created_at, r.updated_at 
		FROM auth_roles r
		JOIN auth_user_roles ur ON r.id = ur.role_id
		WHERE ur.tenant_id = $1 AND ur.user_id = $2
	`
	rows, err := r.db.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		role := domain.Role{}
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	for i := range roles {
		perms, err := r.getRolePermissions(ctx, tenantID, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = perms
	}

	return roles, nil
}

func (r *UserRepository) getRolePermissions(ctx context.Context, tenantID, roleID string) ([]domain.Permission, error) {
	query := `
		SELECT p.id, p.tenant_id, p.name, p.description, p.created_at, p.updated_at 
		FROM auth_permissions p
		JOIN auth_role_permissions rp ON p.id = rp.permission_id
		WHERE rp.tenant_id = $1 AND rp.role_id = $2
	`
	rows, err := r.db.Query(ctx, query, tenantID, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		p := domain.Permission{}
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}
