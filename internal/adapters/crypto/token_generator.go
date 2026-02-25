package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

type SecureTokenGenerator struct{}

func NewSecureTokenGenerator() *SecureTokenGenerator {
	return &SecureTokenGenerator{}
}

func (g *SecureTokenGenerator) GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
