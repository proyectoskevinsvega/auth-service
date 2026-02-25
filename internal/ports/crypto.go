package ports

import (
	"context"

	"github.com/vertercloud/auth-service/internal/domain"
)

type JWTService interface {
	// Generate creates a new JWT token
	Generate(ctx context.Context, token *domain.Token) (string, error)

	// Verify validates and parses a JWT token
	Verify(ctx context.Context, tokenString string) (*domain.Token, error)

	// GetPublicKeyJWKS returns the public key in JWKS format
	GetPublicKeyJWKS() (map[string]interface{}, error)
}

type PasswordHasher interface {
	// Hash hashes a password using Argon2id
	Hash(password string) (string, error)

	// Verify verifies a password against a hash
	Verify(password, hash string) (bool, error)
}

type TOTPService interface {
	// Generate generates a new TOTP secret
	Generate(email string) (secret, qrCode string, err error)

	// Verify verifies a TOTP code
	Verify(code, secret string) (bool, error)
}

type TokenGenerator interface {
	// GenerateSecureToken generates a cryptographically secure random token
	GenerateSecureToken(length int) (string, error)
}
