package http

import (
	"time"
)

// Request DTOs

// RegisterRequest representa la solicitud de registro de un nuevo usuario
type RegisterRequest struct {
	TenantID string `json:"tenant_id" example:"customer1" validate:"required"`           // ID del tenant
	Username string `json:"username" example:"johndoe" validate:"required,min=3,max=30"` // Nombre de usuario único (3-30 caracteres)
	Email    string `json:"email" example:"user@example.com" validate:"required,email"`  // Correo electrónico válido
	Password string `json:"password" example:"SecurePass123!" validate:"required,min=8"` // Contraseña (mínimo 8 caracteres)
}

// LoginRequest representa la solicitud de inicio de sesión
type LoginRequest struct {
	TenantID   string `json:"tenant_id" example:"customer1" validate:"required"`           // ID del tenant
	Identifier string `json:"identifier" example:"johndoe" validate:"required"`      // Email o nombre de usuario
	Password   string `json:"password" example:"SecurePass123!" validate:"required"` // Contraseña
	TwoFACode  string `json:"two_fa_code,omitempty" example:"123456"`                // Código 2FA (opcional, solo si está habilitado)
}

// RefreshTokenRequest representa la solicitud para renovar un token de acceso
type RefreshTokenRequest struct {
	TenantID     string `json:"tenant_id" example:"customer1" validate:"required"`           // ID del tenant
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." validate:"required"` // Token de actualización
}

// ForgotPasswordRequest representa la solicitud de recuperación de contraseña
type ForgotPasswordRequest struct {
	TenantID string `json:"tenant_id" example:"customer1" validate:"required"`          // ID del tenant
	Email    string `json:"email" example:"user@example.com" validate:"required,email"` // Correo electrónico de la cuenta
}

// ResetPasswordRequest representa la solicitud de restablecimiento de contraseña con token
type ResetPasswordRequest struct {
	TenantID    string `json:"tenant_id" example:"customer1" validate:"required"`                  // ID del tenant
	Token       string `json:"token" example:"abc123..." validate:"required"`                      // Token recibido por email
	NewPassword string `json:"new_password" example:"NewSecurePass123!" validate:"required,min=8"` // Nueva contraseña
}

// ResetPasswordWithCodeRequest representa la solicitud de restablecimiento de contraseña con código
type ResetPasswordWithCodeRequest struct {
	TenantID    string `json:"tenant_id" example:"customer1" validate:"required"`                  // ID del tenant
	Email       string `json:"email" example:"user@example.com" validate:"required,email"`         // Correo electrónico de la cuenta
	Code        string `json:"code" example:"123456" validate:"required,len=6"`                    // Código de 6 dígitos recibido por email
	NewPassword string `json:"new_password" example:"NewSecurePass123!" validate:"required,min=8"` // Nueva contraseña
}

// UpdateUserRequest representa la solicitud de actualización de perfil
type UpdateUserRequest struct {
	Email    string `json:"email,omitempty" example:"newemail@example.com"` // Nuevo correo electrónico (opcional)
	Username string `json:"username,omitempty" example:"newusername"`       // Nuevo nombre de usuario (opcional)
}

// Enable2FARequest representa la solicitud para habilitar 2FA
type Enable2FARequest struct {
	// No requiere body
}

// Verify2FARequest representa la solicitud de verificación de código 2FA
type Verify2FARequest struct {
	Code string `json:"code" example:"123456" validate:"required,len=6"` // Código TOTP de 6 dígitos
}

// Disable2FARequest representa la solicitud para deshabilitar 2FA
type Disable2FARequest struct {
	Code string `json:"code" example:"123456" validate:"required,len=6"` // Código TOTP de 6 dígitos para confirmar
}

// VerifyEmailRequest representa la solicitud de verificación de email
type VerifyEmailRequest struct {
	TenantID string `json:"tenant_id" example:"customer1" validate:"required"` // ID del tenant
	Token    string `json:"token" example:"abc123..." validate:"required"`     // Token de verificación recibido por email
}

// Response DTOs

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error   string `json:"error" example:"invalid_credentials"`                     // Identificador del error
	Message string `json:"message,omitempty" example:"Email or password incorrect"` // Mensaje descriptivo del error
	Code    string `json:"code,omitempty" example:"AUTH_001"`                       // Código interno del error
}

// LoginResponse representa la respuesta exitosa de login
type LoginResponse struct {
	AccessToken  string       `json:"access_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`  // JWT token de acceso
	RefreshToken string       `json:"refresh_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."` // Token de actualización
	User         UserResponse `json:"user"`                                                            // Información del usuario
}

// UserResponse representa la información de un usuario
type UserResponse struct {
	ID               string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"` // ID único del usuario (UUID)
	Username         string `json:"username" example:"johndoe"`                        // Nombre de usuario
	Email            string `json:"email" example:"user@example.com"`                  // Correo electrónico
	Active           bool   `json:"active" example:"true"`                             // Estado de la cuenta
	EmailVerified    bool   `json:"email_verified" example:"true"`                     // Indica si el email está verificado
	TwoFactorEnabled bool   `json:"two_factor_enabled" example:"false"`                // Indica si 2FA está habilitado
	CreatedAt        string `json:"created_at" example:"2024-01-15T10:30:00Z"`         // Fecha de creación de la cuenta
}

// RefreshTokenResponse representa la respuesta de renovación de token
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`  // Nuevo JWT token de acceso
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."` // Nuevo token de actualización
}

// MessageResponse representa una respuesta simple con mensaje
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"` // Mensaje de respuesta
}

// Enable2FAResponse representa la respuesta al habilitar 2FA
type Enable2FAResponse struct {
	Secret string `json:"secret" example:"JBSWY3DPEHPK3PXP"`                                   // Secret TOTP para configurar en la app
	QRCode string `json:"qr_code" example:"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."` // Código QR en formato Data URI
}

// BackupCodesResponse representa la respuesta con los códigos de respaldo 2FA
type BackupCodesResponse struct {
	BackupCodes []string `json:"backup_codes" example:"[\"ABC123DEF4\", \"GHI567JKL8\"]"` // Lista de 10 códigos de respaldo
}

// SessionResponse representa la información de una sesión activa
type SessionResponse struct {
	ID         string `json:"id" example:"650e8400-e29b-41d4-a716-446655440001"`     // ID de la sesión
	IPAddress  string `json:"ip_address" example:"192.168.1.100"`                    // Dirección IP de la sesión
	Country    string `json:"country" example:"US"`                                  // Código de país (ISO 3166-1 alpha-2)
	Device     string `json:"device" example:"Chrome on Windows"`                    // Dispositivo/navegador
	UserAgent  string `json:"user_agent" example:"Mozilla/5.0 (Windows NT 10.0...)"` // User agent completo
	CreatedAt  string `json:"created_at" example:"2024-01-15T10:30:00Z"`             // Fecha de creación de la sesión
	LastUsedAt string `json:"last_used_at" example:"2024-01-15T12:45:00Z"`           // Última vez que se usó
	IsCurrent  bool   `json:"is_current" example:"true"`                             // Indica si es la sesión actual
}

// SessionsResponse representa la lista de sesiones activas
type SessionsResponse struct {
	Sessions []SessionResponse `json:"sessions"` // Lista de sesiones activas
}

// JWKSResponse representa el conjunto de claves públicas JWKS
type JWKSResponse struct {
	Keys []JWKResponse `json:"keys"` // Array de claves públicas
}

// JWKResponse representa una clave pública en formato JWK
type JWKResponse struct {
	Kty string `json:"kty" example:"RSA"`       // Tipo de clave (RSA)
	Use string `json:"use" example:"sig"`       // Uso de la clave (signature)
	Alg string `json:"alg" example:"RS256"`     // Algoritmo (RS256)
	N   string `json:"n" example:"xGOr-H3z..."` // Módulo RSA (base64url)
	E   string `json:"e" example:"AQAB"`        // Exponente RSA (base64url)
}

// OIDCConfigurationResponse representa la configuración de OpenID Connect
type OIDCConfigurationResponse struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
	JWKSURI                          string   `json:"jwks_uri"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
}

// UserInfoResponse representa la información detallada del usuario autenticado (OIDC)
type UserInfoResponse struct {
	Sub               string `json:"sub"`
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
}

// HealthResponse representa el estado de salud del servicio
type HealthResponse struct {
	Status  string `json:"status" example:"healthy"`       // Estado del servicio (healthy, unhealthy)
	Service string `json:"service" example:"auth-service"` // Nombre del servicio
	Version string `json:"version" example:"1.0.0"`        // Versión del servicio
}

// RBAC DTOs

type CreateRoleRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=50"`
	Description string `json:"description"`
}

type CreatePermissionRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=50"`
	Description string `json:"description"`
}

type RoleResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Permissions []PermissionResponse `json:"permissions,omitempty"`
}

type PermissionResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// WebAuthn DTOs

type WebAuthnLoginBeginRequest struct {
	Identifier string `json:"identifier" validate:"required"` // Email o username
}

type WebAuthnFinishRequest struct {
	Challenge string      `json:"challenge" validate:"required"`
	Response  interface{} `json:"response" validate:"required"` // El objeto retornado por el navegador (FIDO2)
}

// Webhook DTOs

type CreateWebhookRequest struct {
	URL        string   `json:"url" validate:"required,url"`
	Secret     string   `json:"secret" validate:"required,min=16"`
	EventTypes []string `json:"event_types" validate:"required,min=1"`
}

type WebhookResponse struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	EventTypes []string  `json:"event_types"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

// WebhookListResponse representa el listado de webhooks
type WebhookListResponse struct {
	Webhooks []WebhookResponse `json:"webhooks"`
}

// M2M DTOs

type IssueCertificateRequest struct {
	ClientName string `json:"client_name" validate:"required,min=3,max=50" example:"Empresa_Aliada_A"`
}

type ClientCertificateResponse struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	CACert      string `json:"ca_certificate"`
}

type SignCSRRequest struct {
	CSR string `json:"csr" validate:"required" example:"-----BEGIN CERTIFICATE REQUEST-----\nMIIB... \n-----END CERTIFICATE REQUEST-----"`
}
