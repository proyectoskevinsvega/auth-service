# Integration Example - Prometheus Metrics

Complete example of how to integrate Prometheus metrics into the Auth Service.

## Step 1: Update Handler to Accept Metrics

Modify `internal/adapters/http/handler.go`:

```go
type Handler struct {
	authUC               *usecase.AuthUseCase
	tokenUC              *usecase.TokenUseCase
	sessionUC            *usecase.SessionUseCase
	twofaUC              *usecase.TwoFAUseCase
	emailVerificationUC  *usecase.EmailVerificationUseCase
	userRepo             ports.UserRepository
	googleOAuth          ports.OAuthProvider
	githubOAuth          ports.OAuthProvider
	jwtService           ports.JWTService
	logger               zerolog.Logger
	authMiddleware       *AuthMiddleware
	allowedOrigins       []string
	metrics              *observability.Metrics  // ADD THIS
}

func NewHandler(
	authUC *usecase.AuthUseCase,
	tokenUC *usecase.TokenUseCase,
	sessionUC *usecase.SessionUseCase,
	twofaUC *usecase.TwoFAUseCase,
	emailVerificationUC *usecase.EmailVerificationUseCase,
	userRepo ports.UserRepository,
	googleOAuth ports.OAuthProvider,
	githubOAuth ports.OAuthProvider,
	jwtService ports.JWTService,
	logger zerolog.Logger,
	allowedOrigins []string,
	metrics *observability.Metrics,  // ADD THIS PARAMETER
) *Handler {
	return &Handler{
		authUC:              authUC,
		tokenUC:             tokenUC,
		sessionUC:           sessionUC,
		twofaUC:             twofaUC,
		emailVerificationUC: emailVerificationUC,
		userRepo:            userRepo,
		googleOAuth:         googleOAuth,
		githubOAuth:         githubOAuth,
		jwtService:          jwtService,
		logger:              logger,
		authMiddleware:      NewAuthMiddleware(tokenUC),
		allowedOrigins:      allowedOrigins,
		metrics:             metrics,  // ADD THIS
	}
}

func (h *Handler) SetupRoutes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	// ADD PROMETHEUS MIDDLEWARE
	r.Use(observability.PrometheusMiddleware(h.metrics))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   h.allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
	// ... rest of routes

	// Health check
	r.Get("/health", h.Health)

	// ADD METRICS ENDPOINT
	r.Handle("/metrics", promhttp.Handler())

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	return r
}
```

## Step 2: Update main.go

Modify `cmd/auth-service/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/config"
	cryptoadapter "github.com/vertercloud/auth-service/internal/adapters/crypto"
	geolocationadapter "github.com/vertercloud/auth-service/internal/adapters/geolocation"
	grpcadapter "github.com/vertercloud/auth-service/internal/adapters/grpc"
	pb "github.com/vertercloud/auth-service/internal/adapters/grpc/proto"
	httpadapter "github.com/vertercloud/auth-service/internal/adapters/http"
	"github.com/vertercloud/auth-service/internal/adapters/notification"
	oauthadapter "github.com/vertercloud/auth-service/internal/adapters/oauth"
	postgresadapter "github.com/vertercloud/auth-service/internal/adapters/postgres"
	redisadapter "github.com/vertercloud/auth-service/internal/adapters/redis"
	"github.com/vertercloud/auth-service/internal/observability"  // ADD THIS IMPORT
	"github.com/vertercloud/auth-service/internal/ports"
	"github.com/vertercloud/auth-service/internal/usecase"
	"google.golang.org/grpc"
)

func main() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Set log level
	level, err := zerolog.ParseLevel(cfg.Server.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	logger.Info().Msg("starting auth service")
	logger.Info().Str("environment", cfg.Server.Environment).Msg("configuration loaded")

	// Initialize dependencies
	deps, cleanup, err := initializeDependencies(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize dependencies")
	}
	defer cleanup()

	// Start servers
	httpServer := startHTTPServer(cfg, deps.httpHandler, logger)
	grpcServer := startGRPCServer(cfg, deps.grpcServer, logger)

	// Wait for shutdown signal
	waitForShutdown(httpServer, grpcServer, logger)

	logger.Info().Msg("auth service stopped gracefully")
}

type dependencies struct {
	httpHandler       *httpadapter.Handler
	grpcServer        *grpcadapter.AuthServer
	dbCollector       *observability.DBMetricsCollector        // ADD THIS
	businessCollector *observability.BusinessMetricsCollector  // ADD THIS
}

func initializeDependencies(cfg *config.Config, logger zerolog.Logger) (*dependencies, func(), error) {
	ctx := context.Background()

	// Connect to PostgreSQL
	logger.Info().Msg("connecting to PostgreSQL...")
	pgConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse postgres URL: %w", err)
	}

	pgConfig.MaxConns = int32(cfg.Database.MaxConns)
	pgConfig.MinConns = int32(cfg.Database.MinConns)
	pgConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	pgConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime

	dbPool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Info().Msg("connected to PostgreSQL successfully")

	// Connect to Redis
	logger.Info().Msg("connecting to Redis...")
	redisAddr := cfg.Redis.Addrs[0]
	if len(cfg.Redis.Addrs) > 0 {
		redisAddr = cfg.Redis.Addrs[0]
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: 10,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		dbPool.Close()
		return nil, nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	logger.Info().Msg("connected to Redis successfully")

	// ADD THIS: Initialize Prometheus metrics
	logger.Info().Msg("initializing Prometheus metrics...")
	metrics := observability.NewMetrics("auth_service")

	// ADD THIS: Start database metrics collector
	dbCollector := observability.NewDBMetricsCollector(dbPool, metrics)
	dbCollector.Start(15 * time.Second)
	logger.Info().Msg("database metrics collector started")

	// ADD THIS: Start business metrics collector
	businessCollector := observability.NewBusinessMetricsCollector(dbPool, metrics, logger)
	businessCollector.Start(60 * time.Second)
	logger.Info().Msg("business metrics collector started")

	// Initialize repositories
	userRepo := postgresadapter.NewUserRepository(dbPool)
	tokenRepo := postgresadapter.NewRefreshTokenRepository(dbPool)
	resetRepo := postgresadapter.NewPasswordResetRepository(dbPool)
	sessionRepo := postgresadapter.NewSessionRepository(dbPool)
	auditRepo := postgresadapter.NewAuditLogRepository(dbPool)
	emailVerificationRepo := postgresadapter.NewEmailVerificationRepository(dbPool)

	// Initialize crypto adapters
	jwtService := cryptoadapter.NewJWTService(
		cfg.JWT.PrivateKey,
		cfg.JWT.PublicKey,
		cfg.JWT.Issuer,
	)
	logger.Info().Msg("JWT service initialized with RSA-256")

	passwordHasher := cryptoadapter.NewArgon2Hasher()
	totpService := cryptoadapter.NewTOTPService(cfg.JWT.Issuer)
	tokenGenerator := cryptoadapter.NewSecureTokenGenerator()

	// Initialize Redis adapters
	tokenCache := redisadapter.NewTokenCache(redisClient)
	blacklist := redisadapter.NewTokenBlacklist(redisClient)
	rateLimiter := redisadapter.NewRateLimiter(redisClient)
	sessionStore := redisadapter.NewSessionStore(redisClient)

	// Initialize notification services
	var emailService ports.EmailService
	if cfg.Email.Enabled {
		emailService = notification.NewResendEmailService(cfg.Email.ResendAPIKey, cfg.Email.From, cfg.Email.FromName)
		logger.Info().Msg("Email service (Resend) initialized")
	} else {
		logger.Info().Msg("Email service disabled")
	}
	var redisNotifier ports.NotificationPublisher
	if cfg.Notification.Enabled {
		redisNotifier = notification.NewRedisPublisher(redisClient, cfg.Notification.RedisQueue)
		logger.Info().Msg("Redis notification publisher initialized")
	} else {
		logger.Info().Msg("Notification service disabled")
	}

	// Initialize geolocation service (optional - can handle nil gracefully)
	var geoService ports.GeolocationService
	if cfg.Geolocation.Enabled && cfg.Geolocation.MaxMindDBPath != "" {
		geoService = geolocationadapter.NewMaxMindService(cfg.Geolocation.MaxMindDBPath)
		logger.Info().Msg("MaxMind geo service initialized")
	} else {
		logger.Info().Msg("Geolocation service disabled")
	}

	// Initialize OAuth providers (conditionally)
	oauthProviders := make(map[string]ports.OAuthProvider)

	var googleOAuth, githubOAuth ports.OAuthProvider

	if cfg.OAuth.Google.Enabled {
		googleOAuth = oauthadapter.NewGoogleProvider(
			cfg.OAuth.Google.ClientID,
			cfg.OAuth.Google.ClientSecret,
			cfg.OAuth.Google.RedirectURL,
		)
		oauthProviders["google"] = googleOAuth
		logger.Info().Msg("Google OAuth provider initialized")
	} else {
		logger.Info().Msg("Google OAuth provider disabled")
	}

	if cfg.OAuth.GitHub.Enabled {
		githubOAuth = oauthadapter.NewGitHubProvider(
			cfg.OAuth.GitHub.ClientID,
			cfg.OAuth.GitHub.ClientSecret,
			cfg.OAuth.GitHub.RedirectURL,
		)
		oauthProviders["github"] = githubOAuth
		logger.Info().Msg("GitHub OAuth provider initialized")
	} else {
		logger.Info().Msg("GitHub OAuth provider disabled")
	}

	// Initialize use cases
	tokenUC := usecase.NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		tokenRepo,
		sessionRepo,
		redisNotifier,
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
		geoService,
		emailService,
		redisNotifier,
		oauthProviders,
		cfg,
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
	)

	// MODIFY THIS: Pass metrics to HTTP handler
	httpHandler := httpadapter.NewHandler(
		authUC,
		tokenUC,
		sessionUC,
		twofaUC,
		emailVerificationUC,
		userRepo,
		googleOAuth,
		githubOAuth,
		jwtService,
		logger,
		cfg.Server.AllowedOrigins,
		metrics,  // ADD THIS PARAMETER
	)

	// Initialize gRPC server
	grpcServer := grpcadapter.NewAuthServer(
		tokenUC,
		userRepo,
		logger,
	)

	cleanup := func() {
		logger.Info().Msg("cleaning up resources...")

		// ADD THIS: Stop collectors
		dbCollector.Stop()
		businessCollector.Stop()
		logger.Info().Msg("metrics collectors stopped")

		if geoService != nil {
			if closer, ok := geoService.(interface{ Close() error }); ok {
				_ = closer.Close()
			}
		}

		if err := redisClient.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close Redis connection")
		} else {
			logger.Info().Msg("Redis connection closed")
		}

		dbPool.Close()
		logger.Info().Msg("PostgreSQL connection closed")
	}

	return &dependencies{
		httpHandler:       httpHandler,
		grpcServer:        grpcServer,
		dbCollector:       dbCollector,
		businessCollector: businessCollector,
	}, cleanup, nil
}

func startHTTPServer(cfg *config.Config, handler *httpadapter.Handler, logger zerolog.Logger) *http.Server {
	router := handler.SetupRoutes()

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.HTTPPort).Msg("starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	return server
}

func startGRPCServer(cfg *config.Config, authServer *grpcadapter.AuthServer, logger zerolog.Logger) *grpc.Server {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Server.GRPCPort))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen for gRPC")
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		logger.Info().Str("port", cfg.Server.GRPCPort).Msg("starting gRPC server")
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	return grpcServer
}

func waitForShutdown(httpServer *http.Server, grpcServer *grpc.Server, logger zerolog.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info().Msg("shutdown signal received, starting graceful shutdown...")

	// Graceful HTTP shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown failed")
	} else {
		logger.Info().Msg("HTTP server stopped gracefully")
	}

	// Graceful gRPC shutdown
	grpcServer.GracefulStop()
	logger.Info().Msg("gRPC server stopped gracefully")
}
```

## Step 3: Test Metrics

```bash
# Start the service
make run

# Check metrics endpoint
curl http://localhost:8080/metrics

# You should see output like:
# auth_service_http_requests_total{endpoint="/auth/login",method="POST",status="200"} 42
# auth_service_auth_login_total{method="password",status="success"} 35
# auth_service_database_connections_active 8
# ...
```

## Step 4: Configure Prometheus

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'auth-service'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

Run Prometheus:

```bash
# Using Docker
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus

# Access Prometheus UI
open http://localhost:9090
```

## Step 5: Verify Metrics in Prometheus

1. Open http://localhost:9090
2. Go to "Status" → "Targets" - should show auth-service as UP
3. Try queries:
   ```promql
   rate(auth_service_http_requests_total[5m])
   auth_service_business_users_total
   histogram_quantile(0.95, rate(auth_service_http_request_duration_seconds_bucket[5m]))
   ```

## Done!

Your Auth Service now has comprehensive Prometheus metrics integration with:
- ✅ Automatic HTTP metrics collection
- ✅ Authentication operation metrics
- ✅ Database connection pool monitoring
- ✅ Business KPIs (users, sessions, etc.)
- ✅ Ready for Grafana dashboards
- ✅ Ready for alerting rules

For more queries and dashboard examples, see [README.md](./README.md)
