package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteAuthFlow_WithSetup tests the complete authentication flow:
// Register → Login → GetMe → Refresh Token → Logout → Verify token invalid
func TestCompleteAuthFlow_WithSetup(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Step 1: Register a new user
	t.Run("Register", func(t *testing.T) {
		registerPayload := map[string]interface{}{
			"tenant_id": ts.TenantID,
			"email":     "testuser@example.com",
			"password":  "SecurePassword123!",
			"username":  "testuser",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "register should succeed")

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		// Register returns UserResponse (no tokens)
		assert.Contains(t, response, "id")
		assert.Contains(t, response, "email")
		assert.Contains(t, response, "username")
		assert.Equal(t, "testuser@example.com", response["email"])
		assert.Equal(t, "testuser", response["username"])
	})

	// Step 2: Login with the registered user
	var accessToken, refreshToken string
	t.Run("Login", func(t *testing.T) {
		loginPayload := map[string]interface{}{
			"tenant_id":  ts.TenantID,
			"identifier": "testuser@example.com",
			"password":   "SecurePassword123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "login should succeed")

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		// Login returns access_token, refresh_token, user
		require.Contains(t, response, "access_token")
		require.Contains(t, response, "refresh_token")
		assert.Contains(t, response, "user")

		accessToken = response["access_token"].(string)
		refreshToken = response["refresh_token"].(string)

		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	// Step 3: Validate the access token via GET /auth/me
	t.Run("GetMe", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + accessToken,
		}

		resp := makeRequest(t, ts.Server, "GET", "/api/v1/auth/me", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "GetMe should succeed with valid token")

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "testuser@example.com", response["email"])
		assert.Equal(t, "testuser", response["username"])
	})

	// Step 4: Refresh the access token
	var newAccessToken string
	t.Run("RefreshToken", func(t *testing.T) {
		// Wait a second to ensure IAT is different for the new token
		time.Sleep(time.Second)

		t.Logf("Refreshing token: %s", refreshToken)
		refreshPayload := map[string]interface{}{
			"tenant_id":     ts.TenantID,
			"refresh_token": refreshToken,
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/refresh", refreshPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "token refresh should succeed")

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		require.Contains(t, response, "access_token")
		newAccessToken = response["access_token"].(string)
		assert.NotEmpty(t, newAccessToken)
		assert.NotEqual(t, accessToken, newAccessToken, "new access token should be different")
	})

	// Step 5: Validate new token works via GetMe
	t.Run("GetMeWithNewToken", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + newAccessToken,
		}

		resp := makeRequest(t, ts.Server, "GET", "/api/v1/auth/me", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "GetMe should succeed with refreshed token")
	})

	// Step 6: Logout
	t.Run("Logout", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + newAccessToken,
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/logout", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "logout should succeed")
	})

	// Step 7: Verify token is invalid after logout
	t.Run("GetMeAfterLogout", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + newAccessToken,
		}

		resp := makeRequest(t, ts.Server, "GET", "/api/v1/auth/me", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "token should be invalid after logout")
	})
}

// TestLoginWithInvalidCredentials_WithSetup tests login failure scenarios
func TestLoginWithInvalidCredentials_WithSetup(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// First, register a user
	registerPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":     "validuser@example.com",
		"password":  "ValidPassword123!",
		"username":  "validuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	tests := []struct {
		name           string
		email          string
		password       string
		expectedStatus int
	}{
		{
			name:           "Wrong password",
			email:          "validuser@example.com",
			password:       "WrongPassword123!",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Non-existent user",
			email:          "nonexistent@example.com",
			password:       "SomePassword123!",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty email",
			email:          "",
			password:       "ValidPassword123!",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty password",
			email:          "validuser@example.com",
			password:       "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginPayload := map[string]interface{}{
				"tenant_id":  ts.TenantID,
				"identifier": tt.email,
				"password":   tt.password,
			}

			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestRegisterValidation tests registration input validation
func TestRegisterValidation(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid registration",
			payload: map[string]interface{}{
				"tenant_id": ts.TenantID,
				"email":     "valid@example.com",
				"password":  "SecurePassword123!",
				"username":  "validuser",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid email format",
			payload: map[string]interface{}{
				"tenant_id": ts.TenantID,
				"email":     "notanemail",
				"password":  "SecurePassword123!",
				"username":  "testuser2",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Weak password",
			payload: map[string]interface{}{
				"tenant_id": ts.TenantID,
				"email":     "user@example.com",
				"password":  "weak",
				"username":  "testuser3",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing username",
			payload: map[string]interface{}{
				"tenant_id": ts.TenantID,
				"email":     "user@example.com",
				"password":  "SecurePassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate email",
			payload: map[string]interface{}{
				"tenant_id": ts.TenantID,
				"email":     "valid@example.com",
				"password":  "AnotherPassword123!",
				"username":  "anotheruser",
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", tt.payload, nil)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestConcurrentLogins tests handling of multiple concurrent logins
func TestConcurrentLogins(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register a user
	registerPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":     "concurrent@example.com",
		"password":  "Password123!",
		"username":  "concurrentuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Perform multiple concurrent logins
	numConcurrent := 10
	results := make(chan bool, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func() {
			loginPayload := map[string]interface{}{
				"tenant_id":  ts.TenantID,
				"identifier": "concurrent@example.com",
				"password":   "Password123!",
			}

			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
			defer resp.Body.Close()

			results <- resp.StatusCode == http.StatusOK
		}()
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numConcurrent; i++ {
		if <-results {
			successCount++
		}
	}

	// All concurrent logins should succeed
	assert.Equal(t, numConcurrent, successCount, "all concurrent logins should succeed")
}

// TestSessionManagement tests session creation and retrieval
func TestSessionManagement(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register and login
	registerPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":     "session@example.com",
		"password":  "Password123!",
		"username":  "sessionuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Login to create a session
	loginPayload := map[string]interface{}{
		"tenant_id":  ts.TenantID,
		"identifier": "session@example.com",
		"password":   "Password123!",
	}
	resp = makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var loginResponse map[string]interface{}
	json.Unmarshal(body, &loginResponse)
	accessToken := loginResponse["access_token"].(string)

	// Get active sessions
	t.Run("GetActiveSessions", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + accessToken,
		}

		resp := makeRequest(t, ts.Server, "GET", "/api/v1/auth/sessions", nil, headers)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Logf("Response Status: %d", resp.StatusCode)
			t.Logf("Response Body: %s", string(body))
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		json.Unmarshal(body, &response)

		sessions := response["sessions"].([]interface{})
		assert.GreaterOrEqual(t, len(sessions), 1, "should have at least one active session")
	})

	// Revoke all sessions except current
	t.Run("RevokeAllOtherSessions", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer " + accessToken,
		}

		resp := makeRequest(t, ts.Server, "DELETE", "/api/v1/auth/sessions/all", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestTokenExpiration tests token expiration behavior
func TestTokenExpiration(t *testing.T) {
	// This test would require modifying token expiration times
	// or mocking time, which is complex. Skipping for now.
	t.Skip("Token expiration test requires time manipulation")
}
