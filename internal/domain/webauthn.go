package domain

import "time"

// WebAuthnCredential representa una llave de seguridad registrada por un usuario
type WebAuthnCredential struct {
	ID              []byte    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	UserID          string    `json:"user_id"`
	PublicKey       []byte    `json:"public_key"`
	AttestationType string    `json:"attestation_type"`
	Transport       []string  `json:"transport,omitempty"`
	AAGUID          []byte    `json:"aaguid"`
	SignCount       uint32    `json:"sign_count"`
	CloneWarning    bool      `json:"clone_warning"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WebAuthnSessionData almacena los datos temporales del desafío de WebAuthn
type WebAuthnSessionData struct {
	Challenge            string    `json:"challenge"`
	UserID               string    `json:"user_id"`
	AllowedCredentialIDs [][]byte  `json:"allowed_credentials,omitempty"`
	ExpiresAt            time.Time `json:"expires_at"`
	UserVerification     string    `json:"user_verification"` // preferred, required, discouraged
}
