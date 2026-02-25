package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains all Prometheus metrics for the auth service
type Metrics struct {
	// HTTP Metrics
	HTTPRequestsTotal          *prometheus.CounterVec
	HTTPRequestDuration        *prometheus.HistogramVec
	HTTPRequestsInFlight       prometheus.Gauge
	HTTPResponseSizeBytes      *prometheus.HistogramVec

	// Authentication Metrics
	AuthLoginTotal             *prometheus.CounterVec
	AuthLoginDuration          *prometheus.HistogramVec
	AuthRegisterTotal          *prometheus.CounterVec
	AuthRegisterDuration       *prometheus.HistogramVec
	AuthLogoutTotal            *prometheus.CounterVec
	AuthTokenValidationTotal   *prometheus.CounterVec
	AuthTokenRefreshTotal      *prometheus.CounterVec
	AuthPasswordResetTotal     *prometheus.CounterVec
	AuthOAuthLoginTotal        *prometheus.CounterVec

	// 2FA Metrics
	Auth2FAEnableTotal         *prometheus.CounterVec
	Auth2FAVerifyTotal         *prometheus.CounterVec
	Auth2FADisableTotal        *prometheus.CounterVec

	// Session Metrics
	SessionsActive             prometheus.Gauge
	SessionsCreatedTotal       prometheus.Counter
	SessionsRevokedTotal       *prometheus.CounterVec

	// Token Metrics
	TokensCacheHitTotal        prometheus.Counter
	TokensCacheMissTotal       prometheus.Counter
	TokensBlacklistedTotal     prometheus.Counter
	TokensGeneratedTotal       *prometheus.CounterVec

	// Rate Limiting Metrics
	RateLimitExceededTotal     *prometheus.CounterVec

	// Database Metrics
	DBConnectionsActive        prometheus.Gauge
	DBConnectionsIdle          prometheus.Gauge
	DBQueryDuration            *prometheus.HistogramVec
	DBQueriesTotal             *prometheus.CounterVec

	// Redis Metrics
	RedisCommandsTotal         *prometheus.CounterVec
	RedisCommandDuration       *prometheus.HistogramVec
	RedisConnectionErrors      prometheus.Counter

	// Email Metrics
	EmailsSentTotal            *prometheus.CounterVec
	EmailSendDuration          *prometheus.HistogramVec

	// Business Metrics
	UsersTotal                 prometheus.Gauge
	UsersActiveTotal           prometheus.Gauge
	UsersRegisteredLast24h     prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "auth_service"
	}

	m := &Metrics{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint", "status"},
		),

		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being served",
			},
		),

		HTTPResponseSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),

		// Authentication Metrics
		AuthLoginTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "login_total",
				Help:      "Total number of login attempts",
			},
			[]string{"status", "method"}, // method: password, google, github
		),

		AuthLoginDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "login_duration_seconds",
				Help:      "Login operation duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2, 5},
			},
			[]string{"method"},
		),

		AuthRegisterTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "register_total",
				Help:      "Total number of registration attempts",
			},
			[]string{"status"},
		),

		AuthRegisterDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "register_duration_seconds",
				Help:      "Registration operation duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2, 5},
			},
			[]string{"status"},
		),

		AuthLogoutTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "logout_total",
				Help:      "Total number of logout operations",
			},
			[]string{"status"},
		),

		AuthTokenValidationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "token_validation_total",
				Help:      "Total number of token validations",
			},
			[]string{"status", "cache_hit"},
		),

		AuthTokenRefreshTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "token_refresh_total",
				Help:      "Total number of token refresh operations",
			},
			[]string{"status"},
		),

		AuthPasswordResetTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "password_reset_total",
				Help:      "Total number of password reset operations",
			},
			[]string{"operation", "status"}, // operation: request, complete
		),

		AuthOAuthLoginTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "oauth_login_total",
				Help:      "Total number of OAuth login attempts",
			},
			[]string{"provider", "status"},
		),

		// 2FA Metrics
		Auth2FAEnableTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "twofa_enable_total",
				Help:      "Total number of 2FA enable operations",
			},
			[]string{"status"},
		),

		Auth2FAVerifyTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "twofa_verify_total",
				Help:      "Total number of 2FA verification attempts",
			},
			[]string{"status"},
		),

		Auth2FADisableTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "twofa_disable_total",
				Help:      "Total number of 2FA disable operations",
			},
			[]string{"status"},
		),

		// Session Metrics
		SessionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "sessions",
				Name:      "active",
				Help:      "Number of currently active sessions",
			},
		),

		SessionsCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "sessions",
				Name:      "created_total",
				Help:      "Total number of sessions created",
			},
		),

		SessionsRevokedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "sessions",
				Name:      "revoked_total",
				Help:      "Total number of sessions revoked",
			},
			[]string{"reason"}, // reason: logout, password_reset, admin, expired
		),

		// Token Metrics
		TokensCacheHitTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "tokens",
				Name:      "cache_hit_total",
				Help:      "Total number of token cache hits",
			},
		),

		TokensCacheMissTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "tokens",
				Name:      "cache_miss_total",
				Help:      "Total number of token cache misses",
			},
		),

		TokensBlacklistedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "tokens",
				Name:      "blacklisted_total",
				Help:      "Total number of tokens added to blacklist",
			},
		),

		TokensGeneratedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "tokens",
				Name:      "generated_total",
				Help:      "Total number of tokens generated",
			},
			[]string{"type"}, // type: access, refresh
		),

		// Rate Limiting Metrics
		RateLimitExceededTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "rate_limit",
				Name:      "exceeded_total",
				Help:      "Total number of rate limit violations",
			},
			[]string{"endpoint", "identifier"}, // identifier: ip, email
		),

		// Database Metrics
		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "database",
				Name:      "connections_active",
				Help:      "Number of active database connections",
			},
		),

		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "database",
				Name:      "connections_idle",
				Help:      "Number of idle database connections",
			},
		),

		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "database",
				Name:      "query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation", "table"},
		),

		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "database",
				Name:      "queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),

		// Redis Metrics
		RedisCommandsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis",
				Name:      "commands_total",
				Help:      "Total number of Redis commands",
			},
			[]string{"command", "status"},
		),

		RedisCommandDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "redis",
				Name:      "command_duration_seconds",
				Help:      "Redis command duration in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
			},
			[]string{"command"},
		),

		RedisConnectionErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "redis",
				Name:      "connection_errors_total",
				Help:      "Total number of Redis connection errors",
			},
		),

		// Email Metrics
		EmailsSentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "email",
				Name:      "sent_total",
				Help:      "Total number of emails sent",
			},
			[]string{"template", "status"},
		),

		EmailSendDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "email",
				Name:      "send_duration_seconds",
				Help:      "Email send operation duration in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2, 5, 10},
			},
			[]string{"template"},
		),

		// Business Metrics
		UsersTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "users_total",
				Help:      "Total number of registered users",
			},
		),

		UsersActiveTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "users_active_total",
				Help:      "Number of users active in the last 24 hours",
			},
		),

		UsersRegisteredLast24h: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "users_registered_last_24h",
				Help:      "Number of users registered in the last 24 hours",
			},
		),
	}

	return m
}

// RecordHTTPRequest records an HTTP request with timing
func (m *Metrics) RecordHTTPRequest(method, endpoint, status string, duration time.Duration, responseSize int) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
	if responseSize > 0 {
		m.HTTPResponseSizeBytes.WithLabelValues(method, endpoint).Observe(float64(responseSize))
	}
}

// RecordLogin records a login attempt
func (m *Metrics) RecordLogin(method, status string, duration time.Duration) {
	m.AuthLoginTotal.WithLabelValues(status, method).Inc()
	m.AuthLoginDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordRegister records a registration attempt
func (m *Metrics) RecordRegister(status string, duration time.Duration) {
	m.AuthRegisterTotal.WithLabelValues(status).Inc()
	m.AuthRegisterDuration.WithLabelValues(status).Observe(duration.Seconds())
}

// RecordTokenValidation records a token validation
func (m *Metrics) RecordTokenValidation(status string, cacheHit bool) {
	cacheStatus := "miss"
	if cacheHit {
		cacheStatus = "hit"
		m.TokensCacheHitTotal.Inc()
	} else {
		m.TokensCacheMissTotal.Inc()
	}
	m.AuthTokenValidationTotal.WithLabelValues(status, cacheStatus).Inc()
}

// RecordRateLimitExceeded records a rate limit violation
func (m *Metrics) RecordRateLimitExceeded(endpoint, identifier string) {
	m.RateLimitExceededTotal.WithLabelValues(endpoint, identifier).Inc()
}

// RecordDBQuery records a database query
func (m *Metrics) RecordDBQuery(operation, table, status string, duration time.Duration) {
	m.DBQueriesTotal.WithLabelValues(operation, table, status).Inc()
	m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordRedisCommand records a Redis command
func (m *Metrics) RecordRedisCommand(command, status string, duration time.Duration) {
	m.RedisCommandsTotal.WithLabelValues(command, status).Inc()
	m.RedisCommandDuration.WithLabelValues(command).Observe(duration.Seconds())
}

// RecordEmail records an email send operation
func (m *Metrics) RecordEmail(template, status string, duration time.Duration) {
	m.EmailsSentTotal.WithLabelValues(template, status).Inc()
	m.EmailSendDuration.WithLabelValues(template).Observe(duration.Seconds())
}

// IncrementInFlightRequests increments the in-flight requests counter
func (m *Metrics) IncrementInFlightRequests() {
	m.HTTPRequestsInFlight.Inc()
}

// DecrementInFlightRequests decrements the in-flight requests counter
func (m *Metrics) DecrementInFlightRequests() {
	m.HTTPRequestsInFlight.Dec()
}
