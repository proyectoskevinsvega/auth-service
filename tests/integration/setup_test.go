package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	cryptoadapter "github.com/vertercloud/auth-service/internal/adapters/crypto"
	httpadapter "github.com/vertercloud/auth-service/internal/adapters/http"
	postgresadapter "github.com/vertercloud/auth-service/internal/adapters/postgres"
	redisadapter "github.com/vertercloud/auth-service/internal/adapters/redis"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/ports"
	"github.com/vertercloud/auth-service/internal/usecase"
)

// TestServer represents a test server with all dependencies
type TestServer struct {
	Server                *httptest.Server
	DB                    *pgxpool.Pool
	Redis                 *redis.Client
	Config                *config.Config
	UserRepo              ports.UserRepository
	SessionRepo           ports.SessionRepository
	TokenRepo             ports.RefreshTokenRepository
	ResetRepo             ports.PasswordResetRepository
	AuditRepo             ports.AuditLogRepository
	EmailVerificationRepo ports.EmailVerificationRepository
	JWTService            ports.JWTService
	TokenCache            ports.TokenCache
	Blacklist             ports.TokenBlacklist
	RateLimiter           ports.RateLimiter
	SessionStore          ports.SessionStore
	Cleanup               func()
}

// SetupTestServer creates a new test server with all dependencies
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()
	ctx := context.Background()

	// Load configuration from environment or use test defaults
	cfg, err := loadTestConfig()
	require.NoError(t, err, "failed to load test config")

	// Setup logger (silent in tests)
	logger := zerolog.New(os.Stdout).Level(zerolog.ErrorLevel)

	// Connect to PostgreSQL test database
	dbPool, err := pgxpool.New(ctx, cfg.Database.URL)
	require.NoError(t, err, "failed to connect to test database")
	require.NoError(t, dbPool.Ping(ctx), "failed to ping test database")

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addrs[0],
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})
	require.NoError(t, redisClient.Ping(ctx).Err(), "failed to connect to Redis")

	// Initialize repositories
	userRepo := postgresadapter.NewUserRepository(dbPool)
	tokenRepo := postgresadapter.NewRefreshTokenRepository(dbPool)
	resetRepo := postgresadapter.NewPasswordResetRepository(dbPool)
	sessionRepo := postgresadapter.NewSessionRepository(dbPool)
	auditRepo := postgresadapter.NewAuditLogRepository(dbPool)
	emailVerificationRepo := postgresadapter.NewEmailVerificationRepository(dbPool)
	roleRepo := postgresadapter.NewRoleRepository(dbPool)
	webauthnRepo := postgresadapter.NewWebAuthnRepository(dbPool)
	tenantRepo := postgresadapter.NewTenantRepository(dbPool)

	// Initialize crypto adapters
	jwtService := cryptoadapter.NewJWTService(
		cfg.JWT.PrivateKey,
		cfg.JWT.PublicKey,
		cfg.JWT.Issuer,
	)
	passwordHasher := cryptoadapter.NewArgon2Hasher()
	totpService := cryptoadapter.NewTOTPService(cfg.JWT.Issuer)
	tokenGenerator := cryptoadapter.NewSecureTokenGenerator()

	// Initialize Redis adapters
	tokenCache := redisadapter.NewTokenCache(redisClient)
	blacklist := redisadapter.NewTokenBlacklist(redisClient)
	rateLimiter := redisadapter.NewRateLimiter(redisClient)
	sessionStore := redisadapter.NewSessionStore(redisClient)
	webauthnSessionStore := redisadapter.NewWebAuthnSessionStore(redisClient)

	// Mock notification services (don't send real emails in tests)
	emailService := &mockEmailService{}
	redisNotifier := &mockNotificationPublisher{}
	riskService := usecase.NewRiskService(nil, tenantRepo, nil, cfg) // Geolocation and Threat Intel disabled in tests

	// Initialize use cases
	tokenUC := usecase.NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		tokenRepo,
		sessionRepo,
		riskService,
		tenantRepo,
		postgresadapter.NewClientRepository(dbPool),
		passwordHasher,
		redisNotifier,
		cfg,
	)

	authUC := usecase.NewAuthUseCase(
		userRepo,
		sessionRepo,
		tokenRepo,
		resetRepo,
		auditRepo,
		jwtService,
		passwordHasher,
		tokenGenerator,
		rateLimiter,
		sessionStore,
		nil, // geolocation service (optional)
		emailService,
		redisNotifier,
		make(map[string]ports.OAuthProvider), // no OAuth in basic tests
		cfg,
		riskService,
		roleRepo,
	)

	sessionUC := usecase.NewSessionUseCase(
		sessionRepo,
		userRepo,
		logger,
	)

	twofaUC := usecase.NewTwoFAUseCase(
		userRepo,
		totpService,
		logger,
	)

	emailVerificationUC := usecase.NewEmailVerificationUseCase(
		userRepo,
		emailVerificationRepo,
		emailService,
		logger,
		cfg.Server.BaseDomain,
		cfg.Server.Environment,
	)

	webauthnUC, _ := usecase.NewWebAuthnUseCase(
		userRepo,
		webauthnRepo,
		webauthnSessionStore,
		authUC,
		cfg,
	)

	// Initialize HTTP handler
	httpHandler := httpadapter.NewHandler(
		authUC,
		tokenUC,
		sessionUC,
		twofaUC,
		emailVerificationUC,
		userRepo,
		nil, // Google OAuth
		nil, // GitHub OAuth
		jwtService,
		webauthnUC,
		logger,
		cfg.Server.AllowedOrigins,
		cfg.Server.Environment,
		cfg.JWT.Issuer,
		cfg.Server.BaseDomain,
	)

	// Create test server (disable telemetry for tests)
	router := httpHandler.SetupRoutes(false)
	server := httptest.NewServer(router)

	cleanup := func() {
		// Clean up test data
		cleanupTestData(t, dbPool, redisClient)

		// Close connections
		server.Close()
		redisClient.Close()
		dbPool.Close()
	}

	return &TestServer{
		Server:                server,
		DB:                    dbPool,
		Redis:                 redisClient,
		Config:                cfg,
		UserRepo:              userRepo,
		SessionRepo:           sessionRepo,
		TokenRepo:             tokenRepo,
		ResetRepo:             resetRepo,
		AuditRepo:             auditRepo,
		EmailVerificationRepo: emailVerificationRepo,
		JWTService:            jwtService,
		TokenCache:            tokenCache,
		Blacklist:             blacklist,
		RateLimiter:           rateLimiter,
		SessionStore:          sessionStore,
		Cleanup:               cleanup,
	}
}

// loadTestConfig loads configuration for integration tests
func loadTestConfig() (*config.Config, error) {
	// Always use test defaults to avoid connecting to production databases
	// Generate test RSA keys
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate test RSA keys: %w", err)
	}
	publicKey := &privateKey.PublicKey

	// Use test defaults if env vars not set
	return &config.Config{
		Server: config.ServerConfig{
			Environment:    "test",
			HTTPPort:       "8080",
			GRPCPort:       "50051",
			LogLevel:       "error",
			AllowedOrigins: []string{"*"},
			BaseDomain:     "localhost:8080",
		},
		Database: config.DatabaseConfig{
			URL:             getEnvOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/auth_service_test?sslmode=disable"),
			MaxConns:        10,
			MinConns:        2,
			MaxConnLifetime: 30 * time.Minute,
			MaxConnIdleTime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			Addrs:    []string{getEnvOrDefault("REDIS_ADDR", "localhost:6379")},
			Password: getEnvOrDefault("REDIS_PASSWORD", "redispassword"),
			DB:       getEnvOrDefaultInt("REDIS_DB", 1), // Use DB 1 for tests
			PoolSize: 10,
		},
		JWT: config.JWTConfig{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			Issuer:     "auth-service-test",
		},
		RateLimit: config.RateLimitConfig{
			LoginAttempts:      5,
			LoginWindow:        15 * time.Minute,
			LoginBlockDuration: 15 * time.Minute,
			RegisterAttempts:   3,
			RegisterWindow:     60 * time.Minute,
			RefreshAttempts:    10,
			RefreshWindow:      15 * time.Minute,
		},
		Email: config.EmailConfig{
			Enabled: false, // Disable email sending in tests
		},
		Notification: config.NotificationConfig{
			Enabled: false, // Disable notifications in tests
		},
		Geolocation: config.GeolocationConfig{
			Enabled: false, // Disable geolocation in tests
		},
		OAuth: config.OAuthConfig{
			Google: config.GoogleOAuthConfig{Enabled: false},
			GitHub: config.GitHubOAuthConfig{Enabled: false},
		},
		Security: config.SecurityConfig{
			SessionInactivityDays: 30,
		},
		WebAuthn: config.WebAuthnConfig{
			RPID:          "localhost",
			RPDisplayName: "Test",
			RPOrigins:     []string{"http://localhost"},
		},
	}, nil
}

// cleanupTestData removes all test data from database and Redis
func cleanupTestData(t *testing.T, db *pgxpool.Pool, redis *redis.Client) {
	t.Helper()
	ctx := context.Background()

	// Clean up PostgreSQL tables (in reverse order of dependencies)
	tables := []string{
		"audit_logs",
		"password_reset_tokens",
		"email_verifications",
		"sessions",
		"refresh_tokens",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("warning: failed to clean table %s: %v", table, err)
		}
	}

	// Clean up Redis
	if err := redis.FlushDB(ctx).Err(); err != nil {
		t.Logf("warning: failed to flush Redis: %v", err)
	}
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		fmt.Sscanf(value, "%d", &intVal)
		return intVal
	}
	return defaultValue
}

// makeRequest is a helper to make HTTP requests to the test server
func makeRequest(t *testing.T, server *httptest.Server, method, path string, body interface{}, headers map[string]string) *http.Response {
	t.Helper()

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest(method, server.URL+path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, server.URL+path, nil)
	}

	require.NoError(t, err, "failed to create request")

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "failed to execute request")

	return resp
}
