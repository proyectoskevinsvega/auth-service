package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vertercloud/auth-service/internal/usecase"
)

// TestCompleteAuthFlow prueba el flujo completo de autenticación end-to-end
func TestCompleteAuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Saltando test de integración en modo short")
	}

	ts := SetupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	// Paso 1: Registrar un nuevo usuario
	t.Log("Paso 1: Registrando nuevo usuario...")
	tenantID := ts.TenantID
	registerInput := usecase.RegisterInput{
		TenantID:  tenantID,
		Username:  "testuser_int",
		Email:     "testuser_int@example.com",
		Password:  "SecurePassword123!",
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
	}

	registerOutput, err := ts.AuthUC.Register(ctx, registerInput)
	require.NoError(t, err, "El registro debe ser exitoso")
	require.NotNil(t, registerOutput)
	assert.NotEmpty(t, registerOutput.ID)

	userID := registerOutput.ID
	t.Logf("Usuario registrado exitosamente: ID=%s", userID)

	// Paso 2: Login con credenciales válidas
	t.Log("Paso 2: Iniciando sesión con credenciales válidas...")
	loginInput := usecase.LoginInput{
		TenantID:   tenantID,
		Identifier: "testuser_int",
		Password:   "SecurePassword123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "TestAgent/1.0",
		Device:     "Desktop",
	}

	loginOutput, err := ts.AuthUC.Login(ctx, loginInput)
	require.NoError(t, err, "El login debe ser exitoso")
	require.NotNil(t, loginOutput)
	assert.NotEmpty(t, loginOutput.AccessToken)
	assert.NotEmpty(t, loginOutput.RefreshToken)

	accessToken := loginOutput.AccessToken
	refreshToken := loginOutput.RefreshToken

	// Paso 3: Validar el access token
	t.Log("Paso 3: Validando access token...")
	validatedToken, err := ts.TokenUC.ValidateToken(ctx, accessToken)
	require.NoError(t, err, "ValidateToken debe ser exitoso")
	require.NotNil(t, validatedToken)
	assert.Equal(t, userID, validatedToken.UserID)

	// Paso 4: Refrescar el token usando refresh token
	t.Log("Paso 4: Refrescando access token...")
	time.Sleep(time.Second) // Ensure different IAT
	refreshOutput, err := ts.TokenUC.RefreshToken(ctx, tenantID, refreshToken)
	require.NoError(t, err, "RefreshToken debe ser exitoso")
	require.NotNil(t, refreshOutput)
	assert.NotEmpty(t, refreshOutput.AccessToken)
	assert.NotEmpty(t, refreshOutput.RefreshToken)
	assert.NotEqual(t, accessToken, refreshOutput.AccessToken)

	newAccessToken := refreshOutput.AccessToken

	// Paso 5: Validar el nuevo access token
	t.Log("Paso 5: Validando nuevo access token...")
	validatedNewToken, err := ts.TokenUC.ValidateToken(ctx, newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, validatedNewToken.UserID)

	// Paso 6: Revocar el token (logout)
	t.Log("Paso 6: Revocando access token (logout)...")
	err = ts.TokenUC.RevokeToken(ctx, newAccessToken)
	require.NoError(t, err)

	// Paso 7: Verificar que el token revocado ahora es inválido
	t.Log("Paso 7: Verificando que el token revocado es inválido...")
	_, err = ts.TokenUC.ValidateToken(ctx, newAccessToken)
	assert.Error(t, err)

	t.Log("✅ Test de flujo completo de autenticación pasó exitosamente!")
}
