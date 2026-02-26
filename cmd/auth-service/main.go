package main

// @title           Vertercloud Auth Service API
// @version         1.0
// @description     Microservicio central de autenticación para la plataforma Vertercloud.
// @description     Gestiona JWT, sesiones, 2FA, OAuth, verificación de email y validación de tokens.

// @contact.name   Vertercloud Support
// @contact.email  support@vertercloud.com

// @license.name  Proprietary
// @license.url   https://vertercloud.com/license

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @tag.name Authentication
// @tag.description Endpoints para registro, login, y gestión de tokens

// @tag.name User Profile
// @tag.description Endpoints para gestión de perfil de usuario

// @tag.name Sessions
// @tag.description Endpoints para gestión de sesiones activas

// @tag.name Two-Factor Authentication
// @tag.description Endpoints para autenticación de dos factores (TOTP)

// @tag.name Email Verification
// @tag.description Endpoints para verificación de correo electrónico

// @tag.name OAuth
// @tag.description Endpoints para autenticación con Google y GitHub

// @tag.name System
// @tag.description Endpoints de salud y monitoreo del servicio

// @schemes http https
// @produce json
// @accept json

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	cryptoadapter "github.com/vertercloud/auth-service/internal/adapters/crypto"
	geolocationadapter "github.com/vertercloud/auth-service/internal/adapters/geolocation"
	grpcadapter "github.com/vertercloud/auth-service/internal/adapters/grpc"
	pb "github.com/vertercloud/auth-service/internal/adapters/grpc/proto"
	httpadapter "github.com/vertercloud/auth-service/internal/adapters/http"
	"github.com/vertercloud/auth-service/internal/adapters/notification"
	oauthadapter "github.com/vertercloud/auth-service/internal/adapters/oauth"
	postgresadapter "github.com/vertercloud/auth-service/internal/adapters/postgres"
	redisadapter "github.com/vertercloud/auth-service/internal/adapters/redis"
	threatinteladapter "github.com/vertercloud/auth-service/internal/adapters/threatintel"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/observability"
	"github.com/vertercloud/auth-service/internal/observability/telemetry"
	"github.com/vertercloud/auth-service/internal/ports"
	"github.com/vertercloud/auth-service/internal/usecase"
	"github.com/vertercloud/auth-service/internal/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	// Initialize telemetry
	telemetryConfig := telemetry.Config{
		Enabled:        cfg.Telemetry.Enabled,
		ServiceName:    cfg.Telemetry.ServiceName,
		ServiceVersion: cfg.Telemetry.ServiceVersion,
		Environment:    cfg.Telemetry.Environment,
		ExporterType:   cfg.Telemetry.ExporterType,
		JaegerEndpoint: cfg.Telemetry.JaegerEndpoint,
		OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
		OTLPInsecure:   cfg.Telemetry.OTLPInsecure,
		SamplingRate:   cfg.Telemetry.SamplingRate,
		TraceHTTP:      cfg.Telemetry.TraceHTTP,
		TraceGRPC:      cfg.Telemetry.TraceGRPC,
		TraceDatabase:  cfg.Telemetry.TraceDatabase,
		TraceRedis:     cfg.Telemetry.TraceRedis,
	}

	telemetryProvider, err := telemetry.Initialize(telemetryConfig, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize telemetry")
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryProvider.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("failed to shutdown telemetry")
		}
	}()

	// Initialize dependencies
	deps, cleanup, err := initializeDependencies(cfg, logger, telemetryConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize dependencies")
	}
	defer cleanup()

	// Start servers
	httpServer := startHTTPServer(cfg, deps.httpHandler, logger, telemetryConfig)
	grpcServer := startGRPCServer(cfg, deps.grpcServer, logger, telemetryConfig)

	// Wait for shutdown signal
	waitForShutdown(httpServer, grpcServer, logger)

	logger.Info().Msg("auth service stopped gracefully")
}

type dependencies struct {
	httpHandler *httpadapter.Handler
	grpcServer  *grpcadapter.AuthServer
}

func initializeDependencies(cfg *config.Config, logger zerolog.Logger, telemetryConfig telemetry.Config) (*dependencies, func(), error) {
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

	// Add telemetry tracer if enabled
	if telemetryConfig.Enabled && telemetryConfig.TraceDatabase {
		pgConfig.ConnConfig.Tracer = telemetry.NewPgxTracer()
		logger.Info().Msg("PostgreSQL tracing enabled")
	}

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
	// Use first address if multiple are provided
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

	// Add telemetry hook if enabled
	if telemetryConfig.Enabled && telemetryConfig.TraceRedis {
		redisClient.AddHook(telemetry.NewRedisHook())
		logger.Info().Msg("Redis tracing enabled")
	}

	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		dbPool.Close()
		return nil, nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	logger.Info().Msg("connected to Redis successfully")

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
	clientRepo := postgresadapter.NewClientRepository(dbPool)
	backupCodeRepo := postgresadapter.NewBackupCodeRepository(dbPool)
	webhookRepo := postgresadapter.NewWebhookRepository(dbPool)

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
	webhookClient := httpadapter.NewWebhookClient(logger)

	// Initialize Redis adapters
	tokenCache := redisadapter.NewTokenCache(redisClient)
	blacklist := redisadapter.NewTokenBlacklist(redisClient)
	rateLimiter := redisadapter.NewRateLimiter(redisClient)
	sessionStore := redisadapter.NewSessionStore(redisClient)
	webauthnSessionStore := redisadapter.NewWebAuthnSessionStore(redisClient)

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

	// Initialize threat intelligence service
	var threatIntelService ports.ThreatIntelligenceService
	if cfg.ThreatIntel.Enabled && cfg.ThreatIntel.APIKey != "" {
		threatIntelService = threatinteladapter.NewAbuseIPDBAdapter(cfg)
		logger.Info().Str("provider", cfg.ThreatIntel.Provider).Msg("Threat intelligence service initialized")
	} else {
		logger.Info().Msg("Threat intelligence service disabled")
	}

	// Initialize Risk service (P0)
	riskService := usecase.NewRiskService(geoService, tenantRepo, threatIntelService, cfg)
	logger.Info().Msg("Risk assessment service initialized")

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
		riskService,
		tenantRepo,
		clientRepo,
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
		geoService,
		emailService,
		redisNotifier,
		oauthProviders,
		cfg,
		riskService,
		roleRepo,
		backupCodeRepo,
		totpService,
	)

	sessionUC := usecase.NewSessionUseCase(
		sessionRepo,
		userRepo,
		logger,
	)

	twofaUC := usecase.NewTwoFAUseCase(
		userRepo,
		backupCodeRepo,
		totpService,
		passwordHasher,
		tokenGenerator,
		logger,
	)

	emailVerificationUC := usecase.NewEmailVerificationUseCase(
		userRepo,
		emailVerificationRepo,
		emailService,
		redisNotifier, // Added notifier
		logger,
		cfg.Server.BaseDomain,
		cfg.Server.Environment,
	)
	logger.Info().Msg("Email verification use case initialized")

	webhookUC := usecase.NewWebhookUseCase(
		webhookRepo,
		webhookClient,
		logger,
	)
	logger.Info().Msg("Webhook use case initialized")

	webauthnUC, err := usecase.NewWebAuthnUseCase(
		userRepo,
		webauthnRepo,
		webauthnSessionStore,
		authUC,
		cfg,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize webauthn use case: %w", err)
	}
	logger.Info().Msg("WebAuthn use case initialized")

	metrics := observability.NewMetrics("auth_service")

	// Initialize CertificateManager for mTLS and M2M
	certManager := cryptoadapter.NewCertificateManager("./keys")

	m2mUC := usecase.NewM2MUseCase(certManager, logger)
	logger.Info().Msg("M2M use case initialized")

	complianceUC := usecase.NewComplianceUseCase(
		userRepo,
		auditRepo,
		sessionRepo,
		logger,
	)
	logger.Info().Msg("Compliance use case initialized")

	// Initialize HTTP handler
	httpHandler := httpadapter.NewHandler(
		authUC,
		tokenUC,
		sessionUC,
		twofaUC,
		emailVerificationUC,
		webhookUC,
		userRepo,
		oauthProviders,
		jwtService,
		webauthnUC,
		m2mUC,
		complianceUC,
		logger,
		metrics,
		cfg.Server.AllowedOrigins,
		cfg.Server.Environment,
		cfg.JWT.Issuer,
		cfg.Server.BaseDomain,
	)

	// Start event worker if notifications are enabled
	if cfg.Notification.Enabled {
		eventWorker := worker.NewEventWorker(redisClient, cfg.Notification.RedisQueue, webhookUC, logger)
		go eventWorker.Start(ctx)
		logger.Info().Msg("Event worker started")
	}

	// Initialize gRPC server
	grpcServer := grpcadapter.NewAuthServer(
		tokenUC,
		userRepo,
		logger,
		metrics,
	)

	// mTLS Certificate Management (Bootstrap)
	if cfg.GRPCTLS.Enabled {
		if err := certManager.GenerateMtlsSetup(cfg.Server.BaseDomain); err != nil {
			logger.Error().Err(err).Msg("failed to boostrap mTLS certificates")
			// We continue anyway, the startGRPCServer will fail later if files are missing
		} else {
			logger.Info().Msg("mTLS certificates verified/generated successfully")
		}
	}

	cleanup := func() {
		logger.Info().Msg("cleaning up resources...")

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
		httpHandler: httpHandler,
		grpcServer:  grpcServer,
	}, cleanup, nil
}

func startHTTPServer(cfg *config.Config, handler *httpadapter.Handler, logger zerolog.Logger, telemetryConfig telemetry.Config) *http.Server {
	router := handler.SetupRoutes(telemetryConfig.Enabled && telemetryConfig.TraceHTTP)

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

func startGRPCServer(cfg *config.Config, authServer *grpcadapter.AuthServer, logger zerolog.Logger, telemetryConfig telemetry.Config) *grpc.Server {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Server.GRPCPort))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen for gRPC")
	}

	// Build gRPC server options
	var serverOpts []grpc.ServerOption

	// Setup mTLS if enabled
	if cfg.GRPCTLS.Enabled {
		logger.Info().Msg("gRPC mTLS enabled, loading certificates...")

		// Load server's certificate and private key
		serverCert, err := tls.LoadX509KeyPair(cfg.GRPCTLS.CertPath, cfg.GRPCTLS.KeyPath)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to load gRPC server certificate")
		}

		// Load CA certificate to verify clients
		caCert, err := os.ReadFile(cfg.GRPCTLS.CACertPath)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to read gRPC CA certificate")
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			logger.Fatal().Msg("failed to append gRPC CA certificate to pool")
		}

		// Create TLS configuration
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caCertPool,
			MinVersion:   tls.VersionTLS12,
		}

		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		logger.Info().Msg("gRPC mTLS configured successfully")
	}

	if telemetryConfig.Enabled && telemetryConfig.TraceGRPC {
		serverOpts = append(serverOpts,
			grpc.ChainUnaryInterceptor(
				telemetry.UnaryServerInterceptor(),
				grpcadapter.UnaryIdentityInterceptor(),
			),
		)
		logger.Info().Msg("gRPC tracing and identity extraction enabled")
	} else {
		serverOpts = append(serverOpts,
			grpc.UnaryInterceptor(grpcadapter.UnaryIdentityInterceptor()),
		)
		logger.Info().Msg("gRPC identity extraction enabled")
	}

	grpcServer := grpc.NewServer(serverOpts...)
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
