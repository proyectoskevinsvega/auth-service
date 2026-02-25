# Integration Tests - Auth Service

End-to-end integration tests that validate complete flows with real PostgreSQL and Redis instances.

## Overview

These tests differ from unit tests in that they:
- Use **real PostgreSQL** database (not mocks)
- Use **real Redis** instance (not mocks)
- Test **complete HTTP flows** from request to response
- Validate **data persistence** across operations
- Test **concurrent operations** and race conditions

## Test Coverage

### 1. Complete Authentication Flow (`auth_flow_test.go`)
Tests the entire user authentication lifecycle:
- User registration with validation
- Login with credentials
- Token validation
- Token refresh
- Logout and token invalidation
- Concurrent login handling
- Session management
- Input validation and error cases

**Key Tests:**
- `TestCompleteAuthFlow` - Full cycle: Register → Login → Validate → Refresh → Logout
- `TestLoginWithInvalidCredentials` - Wrong password, non-existent user, empty fields
- `TestRegisterValidation` - Email format, weak passwords, duplicate users
- `TestConcurrentLogins` - 10 concurrent login requests
- `TestSessionManagement` - Create, list, and revoke sessions

### 2. Password Reset Flow (`password_reset_test.go`)
Tests the password reset functionality:
- Request password reset
- Token generation and storage
- Password reset with valid token
- Token expiration handling
- Token reuse prevention
- Session revocation after reset
- Multiple reset requests handling

**Key Tests:**
- `TestPasswordResetFlow` - Complete flow with database token retrieval
- `TestPasswordResetTokenExpiration` - Expired tokens rejection
- `TestPasswordResetInvalidEmail` - User enumeration prevention
- `TestMultiplePasswordResetRequests` - Multiple tokens handling
- `TestPasswordResetRevokesExistingSessions` - Security: sessions invalidated

### 3. Rate Limiting (`rate_limiting_test.go`)
Tests rate limiting across all endpoints:
- Login rate limiting (per user/IP)
- Registration rate limiting
- Rate limit reset after successful auth
- Concurrent request handling
- Per-user isolation

**Key Tests:**
- `TestLoginRateLimiting` - Exhaust limit, verify 429 responses
- `TestRegisterRateLimiting` - Multiple registrations from same IP
- `TestRateLimitPerUser` - User isolation (user1 limit doesn't affect user2)
- `TestConcurrentRateLimiting` - Race conditions in rate limit counters
- `TestRateLimitSuccessfulLoginResets` - Counter reset after successful auth

### 4. OAuth Endpoints (`oauth_test.go`)
Tests OAuth endpoint existence and behavior:
- Endpoint availability
- Error handling without code
- Invalid code handling
- Disabled provider behavior

**Note:** Full OAuth flow testing (with provider mocks) is covered in unit tests. These integration tests verify endpoint wiring and basic error handling.

## Prerequisites

### Required Services

**PostgreSQL** (test database):
```bash
# Using Docker
docker run -d \
  --name auth-postgres-test \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=auth_service_test \
  -p 5432:5432 \
  postgres:16-alpine
```

**Redis** (test cache):
```bash
# Using Docker
docker run -d \
  --name auth-redis-test \
  -p 6379:6379 \
  redis:7-alpine
```

### Database Setup

Run migrations on test database:
```bash
# Set test database URL
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/auth_service_test?sslmode=disable"

# Run migrations
make migrate-up
```

Or create the schema manually:
```bash
psql -U postgres -d auth_service_test -f database/migrations/001_initial_schema.up.sql
psql -U postgres -d auth_service_test -f database/migrations/002_add_performance_indexes.up.sql
psql -U postgres -d auth_service_test -f database/migrations/003_add_email_verifications.up.sql
```

## Running Tests

### Run All Integration Tests

```bash
# From project root
go test -v ./tests/integration/...

# With race detection
go test -race -v ./tests/integration/...

# With coverage
go test -v -coverprofile=coverage.out ./tests/integration/...
go tool cover -html=coverage.out
```

### Run Specific Test Files

```bash
# Auth flow tests only
go test -v ./tests/integration/ -run TestCompleteAuthFlow

# Password reset tests only
go test -v ./tests/integration/ -run TestPasswordReset

# Rate limiting tests only
go test -v ./tests/integration/ -run TestRateLimit
```

### Run Specific Test Cases

```bash
# Single test case
go test -v ./tests/integration/ -run TestCompleteAuthFlow/Register

# Pattern matching
go test -v ./tests/integration/ -run ".*RateLimit.*"
```

### Using Makefile

```bash
# Run integration tests (if added to Makefile)
make test-integration

# Or add this target to Makefile:
# test-integration:
#     @echo "Running integration tests..."
#     go test -v -race ./tests/integration/...
```

## Configuration

Tests use environment variables with fallback defaults:

### Environment Variables

```bash
# PostgreSQL
DATABASE_URL="postgres://postgres:postgres@localhost:5432/auth_service_test?sslmode=disable"

# Redis
REDIS_ADDR="localhost:6379"
REDIS_PASSWORD=""
REDIS_DB="1"  # Use DB 1 for tests (DB 0 for production)

# Rate Limits (for testing)
RATE_LIMIT_LOGIN_MAX_ATTEMPTS="5"
RATE_LIMIT_LOGIN_WINDOW="15m"
RATE_LIMIT_REGISTER_MAX_ATTEMPTS="3"
RATE_LIMIT_REGISTER_WINDOW="60m"
```

### Test Configuration

Test configuration is defined in `setup_test.go`:
- Email service: **disabled** (no real emails sent)
- Notifications: **disabled** (no Redis publishing)
- Geolocation: **disabled** (no MaxMind calls)
- OAuth: **disabled** (no external provider calls)
- JWT: **test RSA keys** (included in setup)

## Test Data Management

### Automatic Cleanup

Each test uses `SetupTestServer()` which:
1. Creates fresh test server with all dependencies
2. Connects to PostgreSQL and Redis
3. Runs the test
4. **Automatically cleans up** all test data via `defer ts.Cleanup()`

Cleanup removes data from:
- `audit_logs`
- `password_reset_tokens`
- `email_verifications`
- `sessions`
- `refresh_tokens`
- `users`
- Redis keys (FLUSHDB on test DB)

### Manual Cleanup

If tests fail and don't clean up:

```bash
# PostgreSQL
psql -U postgres -d auth_service_test -c "TRUNCATE TABLE users CASCADE"

# Redis
redis-cli -n 1 FLUSHDB
```

## Best Practices

### Writing New Integration Tests

1. **Use test infrastructure:**
   ```go
   func TestMyFeature(t *testing.T) {
       ts := SetupTestServer(t)
       defer ts.Cleanup()

       // Your test code here
   }
   ```

2. **Use helper functions:**
   ```go
   resp := makeRequest(t, ts.Server, "POST", "/api/v1/auth/login", payload, headers)
   ```

3. **Clean test data:**
   - Rely on `defer ts.Cleanup()` for automatic cleanup
   - Use unique emails per test: `email-` + test name + `@example.com`

4. **Access real dependencies:**
   ```go
   // Query database directly
   var count int
   ts.DB.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)

   // Check Redis directly
   val, err := ts.Redis.Get(ctx, "some:key").Result()
   ```

5. **Test async operations:**
   ```go
   // Give time for goroutines to complete
   time.Sleep(50 * time.Millisecond)

   // Then verify side effects in database
   ```

### Performance Considerations

Integration tests are **slower** than unit tests:
- Unit tests: ~10-50ms each
- Integration tests: ~100-500ms each (database I/O)

**Optimization tips:**
- Run integration tests separately: `go test -short` skips integration
- Use parallel tests: `t.Parallel()` (be careful with shared state)
- Keep test database small: cleanup after each test

## Troubleshooting

### "Connection refused" errors

If you can't connect to PostgreSQL or Redis:

```bash
# Check if services are running
docker ps

# Start PostgreSQL
docker start auth-postgres-test

# Start Redis
docker start auth-redis-test

# Or create them if they don't exist (see Prerequisites)
```

### "Relation does not exist" errors

If the database schema hasn't been created:

```bash
# Run migrations
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/auth_service_test?sslmode=disable"
make migrate-up
```

### Test timeouts

If tests hang or timeout:
- Check if PostgreSQL/Redis are responsive: `psql`, `redis-cli PING`
- Increase test timeout: `go test -timeout 5m`
- Check for deadlocks in code
- Verify cleanup is called: `defer ts.Cleanup()`

### Flaky tests

If tests pass/fail randomly, check for:
1. **Race conditions**: Run with `-race` flag to detect
2. **Async operations**: Add delays before assertions
3. **Shared state**: Tests affecting each other (ensure cleanup)
4. **Rate limiting**: Tests triggering rate limits (use unique users)

## Continuous Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: auth_service_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run migrations
        run: make migrate-up
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/auth_service_test?sslmode=disable

      - name: Run integration tests
        run: go test -v -race ./tests/integration/...
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/auth_service_test?sslmode=disable
          REDIS_ADDR: localhost:6379
          REDIS_DB: 1
```

## Summary

Integration tests provide high confidence that:
- Complete flows work end-to-end
- Database operations persist correctly
- Redis caching/rate limiting works
- Concurrent operations are handled safely
- Error cases are handled properly

Run these tests before deploying to production!

**Test Coverage:**
- **22 integration tests** covering 4 major flows
- **Complete HTTP request → database** validation
- **Real PostgreSQL + Redis** (no mocks)
- **Production-like scenarios** (concurrent requests, rate limits, etc.)

For **unit tests** (101 tests, 92.7% coverage), see `internal/usecase/*_test.go`

For **load tests** (k6 scenarios), see `tests/k6/README.md`
