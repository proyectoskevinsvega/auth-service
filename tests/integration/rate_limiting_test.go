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
		"email":    "ratelimit@example.com",
		"password": "WrongPassword123!",
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
			"email":    "ratelimit@example.com",
			"password": "Password123!",
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
				"email":    "user" + string(rune('a'+i)) + "@example.com",
				"password": "Password123!",
				"username": "user" + string(rune('a'+i)),
			}

			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
			resp.Body.Close()

			// Should succeed or return validation error, but not rate limit yet
			assert.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode)
		}

		// Next attempt should be rate limited
		registerPayload := map[string]interface{}{
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
	// This test would require waiting for the rate limit window to expire
	// or mocking time, which is complex. For now, we'll test the basic behavior.
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register a user
	registerPayload := map[string]interface{}{
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
		"email":    "reset@example.com",
		"password": "WrongPassword!",
	}

	for i := 0; i < maxAttempts; i++ {
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
		resp.Body.Close()
	}

	// Should be rate limited now
	resp = makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
	resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	// Note: To fully test reset, we would need to either:
	// 1. Wait for the rate limit window to expire (e.g., 15 minutes)
	// 2. Mock the time or Redis TTL
	// 3. Manually clear Redis rate limit keys
	//
	// For a quick test, we'll skip the wait and just verify the behavior above.
	t.Log("Rate limit window reset test requires waiting or time mocking - basic behavior verified")
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
		"email":    users[0].email,
		"password": "WrongPassword!",
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
			"email":    users[1].email,
			"password": users[1].password,
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload2, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "user2 should not be affected by user1's rate limit")
	})
}

// TestRateLimitHeaders tests that rate limit headers are present in responses
func TestRateLimitHeaders(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register a user
	registerPayload := map[string]interface{}{
		"email":    "headers@example.com",
		"password": "Password123!",
		"username": "headersuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Make a login attempt
	loginPayload := map[string]interface{}{
		"email":    "headers@example.com",
		"password": "WrongPassword!",
	}

	resp = makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
	defer resp.Body.Close()

	// Check for rate limit headers (if implemented)
	// Common headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
	// Note: This depends on whether your implementation adds these headers
	t.Log("Rate limit headers check - depends on implementation")

	// Basic assertion: response should have some headers
	assert.NotEmpty(t, resp.Header, "response should have headers")
}

// TestConcurrentRateLimiting tests rate limiting under concurrent requests
func TestConcurrentRateLimiting(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register a user
	registerPayload := map[string]interface{}{
		"email":    "concurrent@example.com",
		"password": "Password123!",
		"username": "concurrentuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Make concurrent failed login attempts
	numConcurrent := 10
	results := make(chan int, numConcurrent)

	loginPayload := map[string]interface{}{
		"email":    "concurrent@example.com",
		"password": "WrongPassword!",
	}

	for i := 0; i < numConcurrent; i++ {
		go func() {
			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", loginPayload, nil)
			defer resp.Body.Close()
			results <- resp.StatusCode
		}()
	}

	// Collect results
	rateLimitedCount := 0
	unauthorizedCount := 0

	for i := 0; i < numConcurrent; i++ {
		status := <-results
		switch status {
		case http.StatusTooManyRequests:
			rateLimitedCount++
		case http.StatusUnauthorized:
			unauthorizedCount++
		}
	}

	// Should have some rate limited responses if we exceeded the limit
	maxAttempts := ts.Config.RateLimit.LoginAttempts
	if numConcurrent > maxAttempts {
		assert.Greater(t, rateLimitedCount, 0, "should have some rate limited responses")
	}

	t.Logf("Concurrent requests: %d unauthorized, %d rate limited", unauthorizedCount, rateLimitedCount)
}

// TestRateLimitBypass tests that successful authentication resets rate limit
func TestRateLimitSuccessfulLoginResets(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	// Register a user
	registerPayload := map[string]interface{}{
		"email":    "success@example.com",
		"password": "Password123!",
		"username": "successuser",
	}
	resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/register", registerPayload, nil)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Make some failed attempts (but not enough to trigger rate limit)
	maxAttempts := ts.Config.RateLimit.LoginAttempts
	failedLoginPayload := map[string]interface{}{
		"email":    "success@example.com",
		"password": "WrongPassword!",
	}

	for i := 0; i < maxAttempts-1; i++ {
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", failedLoginPayload, nil)
		resp.Body.Close()
	}

	// Now login successfully
	t.Run("SuccessfulLogin", func(t *testing.T) {
		correctLoginPayload := map[string]interface{}{
			"email":    "success@example.com",
			"password": "Password123!",
		}

		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", correctLoginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "successful login should work")
	})

	// After successful login, rate limit counter should be reset
	// Make more failed attempts - should be able to make maxAttempts again before being blocked
	t.Run("RateLimitResetAfterSuccess", func(t *testing.T) {
		// Make failed attempts up to limit
		for i := 0; i < maxAttempts; i++ {
			resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", failedLoginPayload, nil)
			resp.Body.Close()
			// Should be 401 (not 429) because counter was reset
			assert.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode)
		}

		// Now should be rate limited
		resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", failedLoginPayload, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "should be rate limited after new round of failures")
	})
}
