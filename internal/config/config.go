package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	JWT          JWTConfig
	RateLimit    RateLimitConfig
	Email        EmailConfig
	OAuth        OAuthConfig
	Security     SecurityConfig
	Geolocation  GeolocationConfig
	Internal     InternalConfig
	Notification NotificationConfig
	Telemetry    TelemetryConfig
	WebAuthn     WebAuthnConfig
	ThreatIntel  ThreatIntelConfig
	GRPCTLS      GRPCTLSConfig
}

type ServerConfig struct {
	HTTPPort       string
	GRPCPort       string
	LogLevel       string
	Environment    string
	BaseDomain     string
	AllowedOrigins []string
	DisableCSRF    bool
}

type DatabaseConfig struct {
	URL             string // Construida internamente
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	MaxConnections  int // Pool size adicional
	MinConnections  int // Pool size adicional
}

type RedisConfig struct {
	Addrs       []string // Construida internamente
	Host        string
	Port        string
	Password    string
	DB          int
	PoolSize    int
	ClusterMode bool
}

type JWTConfig struct {
	PrivateKey    *rsa.PrivateKey
	PublicKey     *rsa.PublicKey
	Issuer        string
	AccessExpiry  time.Duration // JWT access token expiry (default: 15m)
	RefreshExpiry time.Duration // Refresh token expiry (default: 7 days)
}

type RateLimitConfig struct {
	LoginAttempts      int
	LoginWindow        time.Duration
	LoginBlockDuration time.Duration
	RegisterAttempts   int
	RegisterWindow     time.Duration
	RefreshAttempts    int
	RefreshWindow      time.Duration
	ResendAttempts     int
	ResendWindow       time.Duration
	VerifyAttempts     int
	VerifyWindow       time.Duration
}

type EmailConfig struct {
	Enabled      bool
	ResendAPIKey string
	From         string
	FromName     string
}

type OAuthConfig struct {
	Google GoogleOAuthConfig
	GitHub GitHubOAuthConfig
}

type GoogleOAuthConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GitHubOAuthConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type SecurityConfig struct {
	SessionInactivityDays int           // Default session inactivity in days (default: 30)
	Lockout               LockoutConfig // Account lockout configuration (P0)
	PasswordExpiry        PasswordExpiryConfig
}

type PasswordExpiryConfig struct {
	ExpiryDays  int // Days until password expires (default: 90)
	WarningDays int // Days before expiry to start showing warnings (default: 7)
}

type LockoutConfig struct {
	MaxAttempts      int           // Maximum failed attempts before lockout
	BaseDuration     time.Duration // Initial lockout duration
	EscalationFactor float64       // Multiply duration by this for each subsequent failure
	MaxDuration      time.Duration // Maximum possible lockout duration
}

type GeolocationConfig struct {
	Enabled           bool
	MaxMindLicenseKey string
	MaxMindDBPath     string
}

type InternalConfig struct {
}

type NotificationConfig struct {
	Enabled    bool
	RedisQueue string
}

type GRPCTLSConfig struct {
	Enabled    bool
	CACertPath string
	CertPath   string
	KeyPath    string
}

type TelemetryConfig struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExporterType   string // "jaeger", "otlp", "stdout"
	JaegerEndpoint string
	OTLPEndpoint   string
	OTLPInsecure   bool
	SamplingRate   float64 // 0.0 to 1.0
	TraceHTTP      bool
	TraceGRPC      bool
	TraceDatabase  bool
	TraceRedis     bool
}

type WebAuthnConfig struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
}

type ThreatIntelConfig struct {
	Enabled           bool
	Provider          string
	APIKey            string
	BlockThreshold    int // 0-100
	MFAScoreThreshold int // 0-100
}

func Load() (*Config, error) {
	// Load RSA keys
	privateKey, err := loadRSAPrivateKey(getEnv("RSA_PRIVATE_KEY_PATH", "./keys/private.pem"))
	if err != nil {
		return nil, fmt.Errorf("failed to load RSA private key: %w", err)
	}

	publicKey, err := loadRSAPublicKey(getEnv("RSA_PUBLIC_KEY_PATH", "./keys/public.pem"))
	if err != nil {
		return nil, fmt.Errorf("failed to load RSA public key: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			HTTPPort:       getEnv("HTTP_PORT", "8080"),
			GRPCPort:       getEnv("GRPC_PORT", "9090"),
			LogLevel:       getEnv("LOG_LEVEL", "info"),
			Environment:    getEnv("ENVIRONMENT", "development"),
			BaseDomain:     getEnv("BASE_DOMAIN", "localhost:8080"),
			AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080"), ","),
			DisableCSRF:    getEnvAsBool("DISABLE_CSRF", false),
		},
		Database: buildDatabaseConfig(),
		Redis:    buildRedisConfig(),
		JWT: JWTConfig{
			PrivateKey:    privateKey,
			PublicKey:     publicKey,
			Issuer:        getEnv("JWT_ISSUER", "localhost:8080"),
			AccessExpiry:  time.Minute * time.Duration(getEnvAsInt("JWT_ACCESS_EXPIRY_MINUTES", 15)),
			RefreshExpiry: time.Hour * 24 * time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRY_DAYS", 7)),
		},
		RateLimit: RateLimitConfig{
			LoginAttempts:      getEnvAsInt("RATE_LIMIT_LOGIN_ATTEMPTS", 5),
			LoginWindow:        time.Second * time.Duration(getEnvAsInt("RATE_LIMIT_LOGIN_WINDOW", 60)),
			LoginBlockDuration: time.Second * time.Duration(getEnvAsInt("RATE_LIMIT_LOGIN_BLOCK_DURATION", 3600)),
			RegisterAttempts:   getEnvAsInt("RATE_LIMIT_REGISTER_ATTEMPTS", 3),
			RegisterWindow:     time.Second * time.Duration(getEnvAsInt("RATE_LIMIT_REGISTER_WINDOW", 3600)),
			RefreshAttempts:    getEnvAsInt("RATE_LIMIT_REFRESH_ATTEMPTS", 10),
			RefreshWindow:      time.Second * time.Duration(getEnvAsInt("RATE_LIMIT_REFRESH_WINDOW", 60)),
			ResendAttempts:     getEnvAsInt("RATE_LIMIT_RESEND_ATTEMPTS", 4),
			ResendWindow:       time.Second * time.Duration(getEnvAsInt("RATE_LIMIT_RESEND_WINDOW", 3600)),
			VerifyAttempts:     getEnvAsInt("RATE_LIMIT_VERIFY_ATTEMPTS", 5),
			VerifyWindow:       time.Second * time.Duration(getEnvAsInt("RATE_LIMIT_VERIFY_WINDOW", 900)), // 15 minutos en segundos
		},
		Email: EmailConfig{
			Enabled:      getEnvAsBool("EMAIL_ENABLED", false),
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
			From:         getEnv("EMAIL_FROM", "noreply@localhost"),
			FromName:     getEnv("EMAIL_FROM_NAME", "Auth Service"),
		},
		OAuth: OAuthConfig{
			Google: GoogleOAuthConfig{
				Enabled:      getEnvAsBool("GOOGLE_OAUTH_ENABLED", false),
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/oauth/google/callback"),
			},
			GitHub: GitHubOAuthConfig{
				Enabled:      getEnvAsBool("GITHUB_OAUTH_ENABLED", false),
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/auth/oauth/github/callback"),
			},
		},

		Geolocation: GeolocationConfig{
			Enabled:           getEnvAsBool("GEOLOCATION_ENABLED", false),
			MaxMindLicenseKey: getEnv("MAXMIND_LICENSE_KEY", ""),
			MaxMindDBPath:     getEnv("MAXMIND_DB_PATH", "./data/GeoLite2-Country.mmdb"),
		},
		Internal: InternalConfig{},
		Notification: NotificationConfig{
			Enabled:    getEnvAsBool("NOTIFICATIONS_ENABLED", false),
			RedisQueue: getEnv("NOTIFICATION_REDIS_QUEUE", "notify:queue"),
		},
		Security: SecurityConfig{
			SessionInactivityDays: getEnvAsInt("SESSION_INACTIVITY_DAYS", 30),
			Lockout: LockoutConfig{
				MaxAttempts:      getEnvAsInt("LOCKOUT_MAX_ATTEMPTS", 5),
				BaseDuration:     time.Minute * time.Duration(getEnvAsInt("LOCKOUT_BASE_DURATION_MINS", 5)),
				EscalationFactor: getEnvAsFloat("LOCKOUT_ESCALATION_FACTOR", 3.0),
				MaxDuration:      time.Hour * 24 * time.Duration(getEnvAsInt("LOCKOUT_MAX_DURATION_DAYS", 1)),
			},
			PasswordExpiry: PasswordExpiryConfig{
				ExpiryDays:  getEnvAsInt("PASSWORD_EXPIRY_DAYS", 90),
				WarningDays: getEnvAsInt("PASSWORD_EXPIRY_WARNING_DAYS", 7),
			},
		},
		Telemetry: TelemetryConfig{
			Enabled:        getEnvAsBool("TELEMETRY_ENABLED", false),
			ServiceName:    getEnv("TELEMETRY_SERVICE_NAME", "auth-service"),
			ServiceVersion: getEnv("TELEMETRY_SERVICE_VERSION", "1.0.0"),
			Environment:    getEnv("TELEMETRY_ENVIRONMENT", getEnv("ENVIRONMENT", "development")),
			ExporterType:   getEnv("TELEMETRY_EXPORTER_TYPE", "jaeger"),
			JaegerEndpoint: getEnv("TELEMETRY_JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
			OTLPEndpoint:   getEnv("TELEMETRY_OTLP_ENDPOINT", "localhost:4317"),
			OTLPInsecure:   getEnvAsBool("TELEMETRY_OTLP_INSECURE", true),
			SamplingRate:   getEnvAsFloat("TELEMETRY_SAMPLING_RATE", 1.0),
			TraceHTTP:      getEnvAsBool("TELEMETRY_TRACE_HTTP", true),
			TraceGRPC:      getEnvAsBool("TELEMETRY_TRACE_GRPC", true),
			TraceDatabase:  getEnvAsBool("TELEMETRY_TRACE_DATABASE", true),
			TraceRedis:     getEnvAsBool("TELEMETRY_TRACE_REDIS", true),
		},
		WebAuthn: WebAuthnConfig{
			RPID:          getEnv("WEBAUTHN_RP_ID", "localhost"),
			RPDisplayName: getEnv("WEBAUTHN_RP_DISPLAY_NAME", "VerterCloud"),
			RPOrigins:     strings.Split(getEnv("WEBAUTHN_RP_ORIGINS", "http://localhost:8080,https://localhost:8080"), ","),
		},
		ThreatIntel: ThreatIntelConfig{
			Enabled:           getEnvAsBool("THREAT_INTEL_ENABLED", false),
			Provider:          getEnv("THREAT_INTEL_PROVIDER", "abuseipdb"),
			APIKey:            getEnv("THREAT_INTEL_API_KEY", ""),
			BlockThreshold:    getEnvAsInt("THREAT_INTEL_BLOCK_THRESHOLD", 80),
			MFAScoreThreshold: getEnvAsInt("THREAT_INTEL_MFA_THRESHOLD", 40),
		},
		GRPCTLS: GRPCTLSConfig{
			Enabled:    getEnvAsBool("GRPC_TLS_ENABLED", false),
			CACertPath: getEnv("GRPC_CA_CERT_PATH", "./keys/ca.pem"),
			CertPath:   getEnv("GRPC_SERVER_CERT_PATH", "./keys/server.pem"),
			KeyPath:    getEnv("GRPC_SERVER_KEY_PATH", "./keys/server-key.pem"),
		},
	}

	// Validate security settings for production
	if err := validateProductionSecurity(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateProductionSecurity ensures security best practices in production
func validateProductionSecurity(cfg *Config) error {
	// In production, only allow HTTPS origins (except localhost for local testing)
	if cfg.Server.Environment == "production" {
		for _, origin := range cfg.Server.AllowedOrigins {
			// Skip localhost/127.0.0.1 for local testing even in prod
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
				continue
			}
			// Check if origin uses http:// (insecure)
			if strings.HasPrefix(origin, "http://") {
				return fmt.Errorf("security error: insecure HTTP origin '%s' not allowed in production (use HTTPS)", origin)
			}
		}

		// Warn if BASE_DOMAIN doesn't look like a production domain
		if strings.Contains(cfg.Server.BaseDomain, "localhost") || strings.Contains(cfg.Server.BaseDomain, "127.0.0.1") {
			return fmt.Errorf("configuration error: BASE_DOMAIN '%s' looks like development, but ENVIRONMENT=production", cfg.Server.BaseDomain)
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key, defaultValue string) time.Duration {
	valueStr := getEnv(key, defaultValue)
	if duration, err := time.ParseDuration(valueStr); err == nil {
		return duration
	}
	// Fallback
	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}
	return 0
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return defaultValue
}

func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return rsaPrivateKey, nil
}

func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPublicKey, nil
}

func buildDatabaseConfig() DatabaseConfig {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")
	name := getEnv("POSTGRES_NAME", "vertercloud")
	sslmode := getEnv("POSTGRES_SSLMODE", "disable")

	// Construir URL de conexión
	url := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, name, sslmode)

	return DatabaseConfig{
		URL:             url,
		Host:            host,
		Port:            port,
		User:            user,
		Password:        password,
		Name:            name,
		SSLMode:         sslmode,
		MaxConns:        getEnvAsInt("POSTGRES_MAX_CONNS", 100),
		MinConns:        getEnvAsInt("POSTGRES_MIN_CONNS", 10),
		MaxConnLifetime: getEnvAsDuration("POSTGRES_MAX_CONN_LIFETIME", "1h"),
		MaxConnIdleTime: getEnvAsDuration("POSTGRES_MAX_CONN_IDLE_TIME", "30m"),
		MaxConnections:  getEnvAsInt("POSTGRES_MAX_CONNECTIONS", 25),
		MinConnections:  getEnvAsInt("POSTGRES_MIN_CONNECTIONS", 5),
	}
}

func buildRedisConfig() RedisConfig {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")

	// Construir dirección
	addr := fmt.Sprintf("%s:%s", host, port)

	return RedisConfig{
		Addrs:       []string{addr},
		Host:        host,
		Port:        port,
		Password:    getEnv("REDIS_PASSWORD", ""),
		DB:          getEnvAsInt("REDIS_DB", 0),
		PoolSize:    getEnvAsInt("REDIS_POOL_SIZE", 10),
		ClusterMode: getEnvAsBool("REDIS_CLUSTER_MODE", false),
	}
}
