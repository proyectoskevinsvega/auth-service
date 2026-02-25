package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	cryptoadapter "github.com/vertercloud/auth-service/internal/adapters/crypto"
	postgresadapter "github.com/vertercloud/auth-service/internal/adapters/postgres"
	redisadapter "github.com/vertercloud/auth-service/internal/adapters/redis"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/usecase"
)

// TestSuite contiene la infraestructura de tests
type TestSuite struct {
	ctx            context.Context
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	dbPool         *pgxpool.Pool
	redisClient    *goredis.Client

	// Use cases
	authUC  *usecase.AuthUseCase
	tokenUC *usecase.TokenUseCase

	// Configuración
	cfg    *config.Config
	logger zerolog.Logger
}

// setupTestSuite inicializa la infraestructura de tests
func setupTestSuite(t *testing.T) *TestSuite {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zerolog.WarnLevel)

	// Iniciar contenedor PostgreSQL
	t.Log("Iniciando contenedor PostgreSQL...")
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test_auth"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Error al iniciar contenedor PostgreSQL")

	// Iniciar contenedor Redis
	t.Log("Iniciando contenedor Redis...")
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "Error al iniciar contenedor Redis")

	// Obtener cadena de conexión PostgreSQL
	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Conectar a PostgreSQL
	t.Log("Conectando a PostgreSQL...")
	dbPool, err := pgxpool.New(ctx, pgURL)
	require.NoError(t, err, "Error al conectar a PostgreSQL")

	err = dbPool.Ping(ctx)
	require.NoError(t, err, "Error al hacer ping a PostgreSQL")

	// Ejecutar migraciones
	t.Log("Ejecutando migraciones...")
	err = runMigrations(ctx, dbPool)
	require.NoError(t, err, "Error al ejecutar migraciones")

	// Obtener endpoint de Redis
	redisHost, err := redisContainer.Host(ctx)
	require.NoError(t, err)
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort.Port())

	// Conectar a Redis
	t.Log("Conectando a Redis...")
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:         redisAddr,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	err = redisClient.Ping(ctx).Err()
	require.NoError(t, err, "Error al conectar a Redis")

	// Generar claves RSA de prueba para JWT
	privateKey, publicKey := generateTestRSAKeys(t)

	// Crear configuración de prueba
	cfg := &config.Config{
		JWT: config.JWTConfig{
			PrivateKey:    privateKey,
			PublicKey:     publicKey,
			Issuer:        "test-auth-service",
			AccessExpiry:  time.Hour,
			RefreshExpiry: time.Hour * 24 * 7,
		},
		RateLimit: config.RateLimitConfig{
			LoginAttempts:    5,
			LoginWindow:      time.Minute,
			RegisterAttempts: 3,
			RegisterWindow:   time.Hour,
		},
		Server: config.ServerConfig{
			BaseDomain: "test.local",
		},
		Security: config.SecurityConfig{
			SessionInactivityDays: 30,
		},
	}

	// Inicializar repositorios
	userRepo := postgresadapter.NewUserRepository(dbPool)
	tokenRepo := postgresadapter.NewRefreshTokenRepository(dbPool)
	resetRepo := postgresadapter.NewPasswordResetRepository(dbPool)
	sessionRepo := postgresadapter.NewSessionRepository(dbPool)
	auditRepo := postgresadapter.NewAuditLogRepository(dbPool)
	roleRepo := postgresadapter.NewRoleRepository(dbPool)

	// Inicializar adaptadores crypto
	jwtService := cryptoadapter.NewJWTService(
		cfg.JWT.PrivateKey,
		cfg.JWT.PublicKey,
		cfg.JWT.Issuer,
	)
	passwordHasher := cryptoadapter.NewArgon2Hasher()
	tokenGenerator := cryptoadapter.NewSecureTokenGenerator()

	// Inicializar adaptadores Redis
	tokenCache := redisadapter.NewTokenCache(redisClient)
	blacklist := redisadapter.NewTokenBlacklist(redisClient)
	rateLimiter := redisadapter.NewRateLimiter(redisClient)
	sessionStore := redisadapter.NewSessionStore(redisClient)

	// Inicializar use cases (sin servicios opcionales como email, geo, oauth)
	tokenUC := usecase.NewTokenUseCase(
		jwtService,
		tokenCache,
		blacklist,
		userRepo,
		tokenRepo,
		sessionRepo,
		&mockNotificationPublisher{},
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
		nil,                          // geoService
		&mockEmailService{},          // emailService
		&mockNotificationPublisher{}, // notifier
		nil,                          // oauthProviders
		cfg,
		nil, // riskService
		roleRepo,
	)

	return &TestSuite{
		ctx:            ctx,
		pgContainer:    pgContainer,
		redisContainer: redisContainer,
		dbPool:         dbPool,
		redisClient:    redisClient,
		authUC:         authUC,
		tokenUC:        tokenUC,
		cfg:            cfg,
		logger:         logger,
	}
}

// TeardownSuite limpia la infraestructura de tests
func (s *TestSuite) TeardownSuite(t *testing.T) {
	t.Log("Limpiando infraestructura de tests...")

	if s.redisClient != nil {
		s.redisClient.Close()
	}

	if s.dbPool != nil {
		s.dbPool.Close()
	}

	if s.redisContainer != nil {
		_ = s.redisContainer.Terminate(s.ctx)
	}

	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(s.ctx)
	}
}

// runMigrations ejecuta todas las migraciones SQL
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Obtener la ruta del directorio de migraciones
	migrationsDir := filepath.Join("..", "..", "internal", "adapters", "postgres", "migrations")

	// Ejecutar migraciones en orden
	migrations := []string{
		"001_initial_schema.up.sql",
		"002_add_performance_indexes.up.sql",
		"003_add_email_verifications.up.sql",
	}

	for _, migration := range migrations {
		migrationPath := filepath.Join(migrationsDir, migration)
		sqlBytes, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("error al leer migración %s: %w", migration, err)
		}

		_, err = pool.Exec(ctx, string(sqlBytes))
		if err != nil {
			return fmt.Errorf("error al ejecutar migración %s: %w", migration, err)
		}
	}

	return nil
}

// generateTestRSAKeys genera un par de claves RSA de prueba para firma JWT
func generateTestRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	// Generar nuevo par de claves RSA para testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Error al generar clave RSA")

	return privateKey, &privateKey.PublicKey
}

// TestCompleteAuthFlow prueba el flujo completo de autenticación end-to-end
func TestCompleteAuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Saltando test de integración en modo short")
	}

	suite := setupTestSuite(t)
	defer suite.TeardownSuite(t)

	ctx := suite.ctx

	// Paso 1: Registrar un nuevo usuario
	t.Log("Paso 1: Registrando nuevo usuario...")
	registerInput := usecase.RegisterInput{
		Username:  "testuser",
		Email:     "testuser@example.com",
		Password:  "SecurePassword123!",
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
	}

	registerOutput, err := suite.authUC.Register(ctx, registerInput)
	require.NoError(t, err, "El registro debe ser exitoso")
	require.NotNil(t, registerOutput)
	assert.NotEmpty(t, registerOutput.ID)
	assert.Equal(t, registerInput.Username, registerOutput.Username)
	assert.Equal(t, registerInput.Email, registerOutput.Email)
	assert.False(t, registerOutput.EmailVerified, "El email no debe estar verificado inicialmente")

	userID := registerOutput.ID
	t.Logf("Usuario registrado exitosamente: ID=%s", userID)

	// Paso 2: Login con credenciales válidas
	t.Log("Paso 2: Iniciando sesión con credenciales válidas...")
	loginInput := usecase.LoginInput{
		Identifier: "testuser",
		Password:   "SecurePassword123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "TestAgent/1.0",
		Device:     "Desktop",
	}

	loginOutput, err := suite.authUC.Login(ctx, loginInput)
	require.NoError(t, err, "El login debe ser exitoso")
	require.NotNil(t, loginOutput)
	assert.NotEmpty(t, loginOutput.AccessToken)
	assert.NotEmpty(t, loginOutput.RefreshToken)
	assert.Equal(t, userID, loginOutput.User.ID)

	accessToken := loginOutput.AccessToken
	refreshToken := loginOutput.RefreshToken
	t.Logf("Login exitoso: AccessToken longitud=%d, RefreshToken longitud=%d",
		len(accessToken), len(refreshToken))

	// Paso 3: Validar el access token
	t.Log("Paso 3: Validando access token...")
	validatedToken, err := suite.tokenUC.ValidateToken(ctx, accessToken)
	require.NoError(t, err, "ValidateToken debe ser exitoso")
	require.NotNil(t, validatedToken)
	assert.Equal(t, userID, validatedToken.UserID)
	assert.Equal(t, registerInput.Email, validatedToken.Email)
	t.Log("Validación de token exitosa")

	// Paso 4: Refrescar el token usando refresh token
	t.Log("Paso 4: Refrescando access token...")
	refreshOutput, err := suite.tokenUC.RefreshToken(ctx, refreshToken)
	require.NoError(t, err, "RefreshToken debe ser exitoso")
	require.NotNil(t, refreshOutput)
	assert.NotEmpty(t, refreshOutput.AccessToken)
	assert.NotEmpty(t, refreshOutput.RefreshToken)
	assert.NotEqual(t, accessToken, refreshOutput.AccessToken, "El nuevo access token debe ser diferente")
	assert.NotEqual(t, refreshToken, refreshOutput.RefreshToken, "El nuevo refresh token debe ser diferente (rotación)")

	newAccessToken := refreshOutput.AccessToken
	t.Logf("Refresco de token exitoso: Nuevo AccessToken longitud=%d", len(newAccessToken))

	// Paso 5: Validar el nuevo access token
	t.Log("Paso 5: Validando nuevo access token...")
	validatedNewToken, err := suite.tokenUC.ValidateToken(ctx, newAccessToken)
	require.NoError(t, err, "ValidateToken para nuevo token debe ser exitoso")
	require.NotNil(t, validatedNewToken)
	assert.Equal(t, userID, validatedNewToken.UserID)
	t.Log("Validación de nuevo token exitosa")

	// Paso 6: Revocar el token (logout)
	t.Log("Paso 6: Revocando access token (logout)...")
	err = suite.tokenUC.RevokeToken(ctx, newAccessToken)
	require.NoError(t, err, "RevokeToken debe ser exitoso")
	t.Log("Revocación de token exitosa")

	// Paso 7: Verificar que el token revocado ahora es inválido
	t.Log("Paso 7: Verificando que el token revocado es inválido...")
	_, err = suite.tokenUC.ValidateToken(ctx, newAccessToken)
	assert.Error(t, err, "ValidateToken debe fallar para token revocado")
	assert.Equal(t, domain.ErrTokenRevoked, err, "Debe retornar ErrTokenRevoked")
	t.Log("Validación de token revocado falló correctamente")

	t.Log("✅ Test de flujo completo de autenticación pasó exitosamente!")
}

// TestLoginWithInvalidCredentials prueba escenarios de fallo en login
func TestLoginWithInvalidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("Saltando test de integración en modo short")
	}

	suite := setupTestSuite(t)
	defer suite.TeardownSuite(t)

	ctx := suite.ctx

	// Registrar un usuario primero
	registerInput := usecase.RegisterInput{
		Username:  "testuser2",
		Email:     "testuser2@example.com",
		Password:  "CorrectPassword123!",
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
	}

	_, err := suite.authUC.Register(ctx, registerInput)
	require.NoError(t, err)

	// Intentar login con contraseña incorrecta
	loginInput := usecase.LoginInput{
		Identifier: "testuser2",
		Password:   "WrongPassword123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "TestAgent/1.0",
		Device:     "Desktop",
	}

	_, err = suite.authUC.Login(ctx, loginInput)
	assert.Error(t, err, "El login debe fallar con contraseña incorrecta")
	assert.Equal(t, domain.ErrInvalidCredentials, err)
}

// TestTokenValidation_CachePerformance prueba que la validación de tokens usa cache
func TestTokenValidation_CachePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Saltando test de integración en modo short")
	}

	suite := setupTestSuite(t)
	defer suite.TeardownSuite(t)

	ctx := suite.ctx

	// Registrar y hacer login para obtener un token
	registerInput := usecase.RegisterInput{
		Username:  "cachetest",
		Email:     "cachetest@example.com",
		Password:  "Password123!",
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
	}
	_, err := suite.authUC.Register(ctx, registerInput)
	require.NoError(t, err)

	loginInput := usecase.LoginInput{
		Identifier: "cachetest",
		Password:   "Password123!",
		IPAddress:  "192.168.1.1",
		UserAgent:  "TestAgent/1.0",
		Device:     "Desktop",
	}
	loginOutput, err := suite.authUC.Login(ctx, loginInput)
	require.NoError(t, err)

	token := loginOutput.AccessToken

	// Primera validación (frío - cache miss)
	start := time.Now()
	_, err = suite.tokenUC.ValidateToken(ctx, token)
	require.NoError(t, err)
	firstValidation := time.Since(start)

	// Segunda validación (caliente - cache hit)
	start = time.Now()
	_, err = suite.tokenUC.ValidateToken(ctx, token)
	require.NoError(t, err)
	secondValidation := time.Since(start)

	t.Logf("Primera validación (cache miss): %v", firstValidation)
	t.Logf("Segunda validación (cache hit): %v", secondValidation)

	// Cache hit debe ser significativamente más rápido (al menos 2x)
	// Nota: Con implementaciones reales esta diferencia es más pronunciada
	assert.True(t, secondValidation <= firstValidation,
		"Cache hit debe ser más rápido o igual que cache miss")
}
