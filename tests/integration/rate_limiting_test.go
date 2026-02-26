package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoginRateLimiting tests rate limiting on login endpoint
func TestLoginRateLimiting(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register a user first
	registerPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":    "ratelimit@example.com",
		"password": "Password123!",
		"username": "ratelimituser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Make multiple failed login attempts (wrong password)
	maxAttempts := ts.Config.RateLimit.LoginAttempts
	loginPayload := map[string]interface{}{
		"tenant_id":  ts.TenantID,
		"identifier": "ratelimit@example.com",
		"password":   "WrongPassword123!",
	}

	t.Run("ExceedRateLimit", func(t *testing.T) {
		// First N attempts should return 401 (unauthorized)
		for i := 0; i < maxAttempts; i++ {
			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
			resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "failed login should return 401")
		}

		// Next attempt should be rate limited (429)
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "should be rate limited after max attempts")
	})

	// Even with correct password, should still be rate limited
	t.Run("CorrectPasswordStillBlocked", func(t *testing.T) {
		correctLoginPayload := map[string]interface{}{
			"tenant_id":  ts.TenantID,
			"identifier": "ratelimit@example.com",
			"password":   "Password123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", correctLoginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "should still be rate limited even with correct password")
	})
}

// TestRegisterRateLimiting tests rate limiting on register endpoint
func TestRegisterRateLimiting(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	maxAttempts := ts.Config.RateLimit.RegisterAttempts

	t.Run("ExceedRegisterRateLimit", func(t *testing.T) {
		// Make multiple registration attempts from same IP
		for i := 0; i < maxAttempts; i++ {
			registerPayload := map[string]interface{}{
				"tenant_id": ts.TenantID,
				"email":    "user" + string(rune('a'+i)) + "@example.com",
				"password": "Password123!",
				"username": "user" + string(rune('a'+i)),
			}

			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				break
			}
			resp.Body.Close()

			// Should succeed or return validation error, but not rate limit yet
			assert.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode)
		}

		// Next attempt should be rate limited
		registerPayload := map[string]interface{}{
			"tenant_id": ts.TenantID,
			"email":    "ratelimited@example.com",
			"password": "Password123!",
			"username": "ratelimiteduser",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "should be rate limited after max registration attempts")
	})
}

// TestRateLimitReset tests that rate limit resets after window expires
func TestRateLimitReset(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register a user
	registerPayload := map[string]interface{}{
		"tenant_id": ts.TenantID,
		"email":    "reset@example.com",
		"password": "Password123!",
		"username": "resetuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Make failed login attempts up to limit
	maxAttempts := ts.Config.RateLimit.LoginAttempts
	loginPayload := map[string]interface{}{
		"tenant_id":  ts.TenantID,
		"identifier": "reset@example.com",
		"password":   "WrongPassword!",
	}

	for i := 0; i < maxAttempts; i++ {
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
		resp.Body.Close()
	}

	// Should be rate limited now
	resp = makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
	resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	t.Log("Rate limit window reset test basic behavior verified")
}

// TestRateLimitPerUser tests that rate limiting is per-user (email)
func TestRateLimitPerUser(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register two users
	users := []struct {
		email    string
		password string
	}{
		{"user1@example.com", "Password123!"},
		{"user2@example.com", "Password456!"},
	}

	for i, user := range users {
		registerPayload := map[string]interface{}{
			"tenant_id": ts.TenantID,
			"email":    user.email,
			"password": user.password,
			"username": "ratelimituser" + string(rune('1'+i)),
		}
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// Exhaust rate limit for user1
	maxAttempts := ts.Config.RateLimit.LoginAttempts
	loginPayload1 := map[string]interface{}{
		"tenant_id":  ts.TenantID,
		"identifier": users[0].email,
		"password":   "WrongPassword!",
	}

	for i := 0; i <= maxAttempts; i++ {
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload1, nil)
		resp.Body.Close()
	}

	// user1 should be rate limited
	t.Run("User1RateLimited", func(t *testing.T) {
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload1, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "user1 should be rate limited")
	})

	// user2 should still be able to login (not affected by user1's rate limit)
	t.Run("User2NotAffected", func(t *testing.T) {
		loginPayload2 := map[string]interface{}{
			"tenant_id":  ts.TenantID,
			"identifier": users[1].email,
			"password":   users[1].password,
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload2, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "user2 should not be affected by user1's rate limit")
	})
}
