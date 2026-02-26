package crypto

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vertercloud/auth-service/internal/domain"
)

type JWTService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	kid        string // Key ID for key rotation
}

func NewJWTService(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, issuer string) *JWTService {
	// Generate a deterministic Key ID (KID) based on the public key
	// Usually done by hashing the public key modulus or public exponent
	// For simplicity and standard compliance, we'll hash the DER encoded public key
	// Since we don't want to add x509 dependency here if avoiding it,
	// we use a simple SHA-1 (or base64) of the Modulus bytes.
	kidBytes := publicKey.N.Bytes()
	kid := base64.RawURLEncoding.EncodeToString(kidBytes)[:16] // truncated for brevity

	return &JWTService{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
		kid:        kid,
	}
}

type CustomClaims struct {
	Email string   `json:"email"`
	Roles []string `json:"roles,omitempty"`
	Scope string   `json:"scp,omitempty"`
	Perms []string `json:"perms,omitempty"`
	jwt.RegisteredClaims
}

func (s *JWTService) Generate(ctx context.Context, token *domain.Token) (string, error) {
	claims := CustomClaims{
		Email: token.Email,
		Roles: token.Roles,
		Scope: strings.Join(token.Scopes, " "),
		Perms: token.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   token.UserID,
			IssuedAt:  jwt.NewNumericDate(time.Unix(token.IssuedAt, 0)),
			ExpiresAt: jwt.NewNumericDate(time.Unix(token.ExpiresAt, 0)),
			Issuer:    s.issuer,
			ID:        token.JTI,
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	
	// Inyectamos el KID en los Headers del JWT
	jwtToken.Header["kid"] = s.kid

	tokenString, err := jwtToken.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (s *JWTService) Verify(ctx context.Context, tokenString string) (*domain.Token, error) {
	jwtToken, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := jwtToken.Claims.(*CustomClaims)
	if !ok || !jwtToken.Valid {
		return nil, domain.ErrInvalidToken
	}

	// Check expiration
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, domain.ErrTokenExpired
	}

	return &domain.Token{
		JTI:         claims.ID,
		UserID:      claims.Subject,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Scopes:      strings.Split(claims.Scope, " "),
		Permissions: claims.Perms,
		IssuedAt:    claims.IssuedAt.Unix(),
		ExpiresAt:   claims.ExpiresAt.Unix(),
	}, nil
}

func (s *JWTService) GetPublicKeyJWKS() (map[string]interface{}, error) {
	// Convert RSA public key to JWKS format
	n := base64.RawURLEncoding.EncodeToString(s.publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(s.publicKey.E)).Bytes())

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kid": s.kid,
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	}

	return jwks, nil
}
