package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOAuthEndpointsExist tests that OAuth endpoints exist and respond appropriately
// Note: These tests verify endpoint existence, not actual OAuth flow (which requires external providers)
func TestOAuthEndpointsExist(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int // When OAuth is disabled
	}{
		{
			name:           "Google OAuth callback",
			method:         "GET",
			path:           "/api/v1/auth/oauth/google/callback",
			expectedStatus: http.StatusBadRequest, // No code param
		},
		{
			name:           "GitHub OAuth callback",
			method:         "GET",
			path:           "/api/v1/auth/oauth/github/callback",
			expectedStatus: http.StatusBadRequest, // No code param
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := makeRequest(t, ts.Server, tt.method, tt.path, nil, nil)
			defer resp.Body.Close()

			// Just verify the endpoint exists (not 404)
			assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "endpoint should exist")

			// With OAuth disabled and no code, should return bad request or service unavailable
			assert.Contains(t, []int{
				http.StatusBadRequest,
				http.StatusServiceUnavailable,
				http.StatusInternalServerError,
			}, resp.StatusCode, "should return error when OAuth disabled or missing code")
		})
	}
}

// TestOAuthCallbackWithoutCode tests OAuth callbacks without authorization code
func TestOAuthCallbackWithoutCode(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	providers := []string{"google", "github"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			path := "/api/v1/auth/oauth/" + provider + "/callback"

			resp := makeRequest(t, ts.Server, "GET", path, nil, nil)
			defer resp.Body.Close()

			// Should return error (bad request or similar)
			assert.NotEqual(t, http.StatusOK, resp.StatusCode, "should fail without authorization code")
		})
	}
}

// TestOAuthCallbackWithInvalidCode tests OAuth callbacks with invalid authorization code
func TestOAuthCallbackWithInvalidCode(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	providers := []string{"google", "github"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			path := "/api/v1/auth/oauth/" + provider + "/callback?code=invalid_code_12345"

			resp := makeRequest(t, ts.Server, "GET", path, nil, nil)
			defer resp.Body.Close()

			// Should return error (OAuth provider will reject invalid code)
			// Or service unavailable if OAuth is disabled
			assert.Contains(t, []int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusServiceUnavailable,
				http.StatusInternalServerError,
			}, resp.StatusCode, "should fail with invalid code")
		})
	}
}

// Note: Full OAuth integration tests would require:
// 1. Mock OAuth provider servers
// 2. Test authorization URL generation
// 3. Test token exchange
// 4. Test user info retrieval
// 5. Test user creation/update from OAuth data
//
// These are better suited for OAuth provider adapter unit tests
// where mocking is easier and more appropriate.

// TestOAuthDisabled verifies behavior when OAuth is disabled (default in tests)
func TestOAuthDisabled(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Verify OAuth is disabled in test config
	assert.False(t, ts.Config.OAuth.Google.Enabled, "Google OAuth should be disabled in tests")
	assert.False(t, ts.Config.OAuth.GitHub.Enabled, "GitHub OAuth should be disabled in tests")

	// Callbacks should return service unavailable or similar
	providers := []string{"google", "github"}

	for _, provider := range providers {
		t.Run(provider+"_disabled", func(t *testing.T) {
			path := "/api/v1/auth/oauth/" + provider + "/callback?code=test_code"

			resp := makeRequest(t, ts.Server, "GET", path, nil, nil)
			defer resp.Body.Close()

			// Should indicate OAuth is not available
			// Exact status depends on implementation (might be 503, 400, or 500)
			assert.NotEqual(t, http.StatusOK, resp.StatusCode, "OAuth callback should fail when disabled")
		})
	}
}
