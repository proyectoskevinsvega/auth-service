package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPasswordResetFlow tests the complete password reset flow:
// Register → Request Reset → Get token from DB → Reset Password → Login with New Password
func TestPasswordResetFlow(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	// Step 1: Register a user
	registerPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":     "reset@example.com",
		"password":  "OldPassword123!",
		"username":  "resetuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Step 2: Request password reset
	var resetToken string
	t.Run("RequestPasswordReset", func(t *testing.T) {
		resetPayload := map[string]interface{}{
			"tenant_id": ts.TenantID,
			"email":     "reset@example.com",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/forgot-password", resetPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "password reset request should succeed")

		// Get the token from the auth_password_resets table via user_id
		err := ts.DB.QueryRow(ctx, `
			SELECT pr.token FROM auth_password_resets pr
			JOIN auth_users u ON pr.user_id = u.id
			WHERE u.tenant_id = $1 AND u.email = $2 AND pr.used = false AND pr.expires_at > NOW()
			ORDER BY pr.created_at DESC LIMIT 1
		`, ts.TenantID, "reset@example.com").Scan(&resetToken)
		require.NoError(t, err, "should find reset token in database")
		assert.NotEmpty(t, resetToken)
	})

	// Step 3: Reset password with token
	t.Run("ResetPassword", func(t *testing.T) {
		resetPayload := map[string]interface{}{
			"tenant_id":    ts.TenantID,
			"token":        resetToken,
			"new_password": "NewPassword123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/reset-password", resetPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "password reset should succeed")
	})

	// Step 4: Try to login with old password (should fail)
	t.Run("LoginWithOldPassword", func(t *testing.T) {
		loginPayload := map[string]interface{}{
			"tenant_id":  ts.TenantID,
			"identifier": "reset@example.com",
			"password":   "OldPassword123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "login with old password should fail")
	})

	// Step 5: Login with new password (should succeed)
	t.Run("LoginWithNewPassword", func(t *testing.T) {
		loginPayload := map[string]interface{}{
			"tenant_id":  ts.TenantID,
			"identifier": "reset@example.com",
			"password":   "NewPassword123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "login with new password should succeed")

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)

		assert.Contains(t, response, "access_token")
		assert.Contains(t, response, "refresh_token")
	})

	// Step 6: Try to use the same reset token again (should fail)
	t.Run("ReuseResetToken", func(t *testing.T) {
		resetPayload := map[string]interface{}{
			"tenant_id":    ts.TenantID,
			"token":        resetToken,
			"new_password": "AnotherPassword123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/reset-password", resetPayload, nil)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "reusing reset token should fail")
	})
}

// TestPasswordResetInvalidEmail tests password reset request with invalid email
func TestPasswordResetInvalidEmail(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	tests := []struct {
		name           string
		email          string
		expectedStatus int
	}{
		{
			name:           "Non-existent email",
			email:          "nonexistent@example.com",
			expectedStatus: http.StatusOK, // Should return 200 to prevent user enumeration
		},
		{
			name:           "Empty email",
			email:          "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPayload := map[string]interface{}{
				"tenant_id": ts.TenantID,
				"email":     tt.email,
			}

			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/forgot-password", resetPayload, nil)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestPasswordResetInvalidToken tests password reset with invalid tokens
func TestPasswordResetInvalidToken(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	tests := []struct {
		name           string
		token          string
		password       string
		expectedStatus int
	}{
		{
			name:           "Empty token",
			token:          "",
			password:       "NewPassword123!",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty password",
			token:          "some-token",
			password:       "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPayload := map[string]interface{}{
				"tenant_id":    ts.TenantID,
				"token":        tt.token,
				"new_password": tt.password,
			}

			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/reset-password", resetPayload, nil)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestPasswordResetRevokesExistingSessions tests that password reset revokes all sessions
func TestPasswordResetRevokesExistingSessions(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	ctx := context.Background()

	// Register a user
	registerPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":     "revoke@example.com",
		"password":  "OldPassword123!",
		"username":  "revokeuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Login to get a token
	loginPayload := map[string]interface{}{
		"tenant_id":  ts.TenantID,
		"identifier": "revoke@example.com",
		"password":   "OldPassword123!",
	}
	resp = makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResponse map[string]interface{}
	json.Unmarshal(body, &loginResponse)
	accessToken := loginResponse["access_token"].(string)

	// Verify token works
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	resp = makeRequest(t, ts.Server, "GET", "/api/v1/auth/me", nil, headers)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "token should work before reset")

	// Request and perform password reset
	resetRequestPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":     "revoke@example.com",
	}
	resp = makeRequest(t, ts.Server, "POST", "/api/v1/auth/forgot-password", resetRequestPayload, nil)
	resp.Body.Close()

	// Get reset token from DB
	var resetToken string
	err := ts.DB.QueryRow(ctx, `
		SELECT pr.token FROM auth_password_resets pr
		JOIN auth_users u ON pr.user_id = u.id
		WHERE u.tenant_id = $1 AND u.email = $2 AND pr.used = false
		ORDER BY pr.created_at DESC LIMIT 1
	`, ts.TenantID, "revoke@example.com").Scan(&resetToken)
	require.NoError(t, err)

	// Reset password
	resetPayload := map[string]interface{}{
		"tenant_id":    ts.TenantID,
		"token":        resetToken,
		"new_password": "NewPassword123!",
	}
	resp = makeRequest(t, ts.Server, "POST", "/api/v1/auth/reset-password", resetPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Old token should now be invalid
	t.Run("OldTokenInvalidAfterReset", func(t *testing.T) {
		resp := makeRequest(t, ts.Server, "GET", "/api/v1/auth/me", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "old token should be invalid after password reset")
	})

	// Should be able to login with new password
	t.Run("LoginWithNewPasswordAfterReset", func(t *testing.T) {
		loginPayload := map[string]interface{}{
			"tenant_id":  ts.TenantID,
			"identifier": "revoke@example.com",
			"password":   "NewPassword123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
