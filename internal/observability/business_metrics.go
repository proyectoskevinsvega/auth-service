package observability

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// BusinessMetricsCollector collects business metrics periodically
type BusinessMetricsCollector struct {
	db      *pgxpool.Pool
	metrics *Metrics
	logger  zerolog.Logger
	done    chan struct{}
}

// NewBusinessMetricsCollector creates a new business metrics collector
func NewBusinessMetricsCollector(db *pgxpool.Pool, metrics *Metrics, logger zerolog.Logger) *BusinessMetricsCollector {
	return &BusinessMetricsCollector{
		db:      db,
		metrics: metrics,
		logger:  logger,
		done:    make(chan struct{}),
	}
}

// Start begins collecting business metrics periodically
func (c *BusinessMetricsCollector) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		// Collect immediately on start
		c.collect()

		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.done:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the metrics collector
func (c *BusinessMetricsCollector) Stop() {
	close(c.done)
}

// collect gathers current business metrics
func (c *BusinessMetricsCollector) collect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Total users
	var totalUsers int64
	err := c.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to collect total users metric")
	} else {
		c.metrics.UsersTotal.Set(float64(totalUsers))
	}

	// Active users (last 24 hours)
	var activeUsers int64
	err = c.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id)
		FROM sessions
		WHERE last_activity > NOW() - INTERVAL '24 hours'
	`).Scan(&activeUsers)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to collect active users metric")
	} else {
		c.metrics.UsersActiveTotal.Set(float64(activeUsers))
	}

	// Users registered in last 24 hours
	var newUsers int64
	err = c.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM users
		WHERE created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&newUsers)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to collect new users metric")
	} else {
		c.metrics.UsersRegisteredLast24h.Set(float64(newUsers))
	}

	// Active sessions
	var activeSessions int64
	err = c.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM sessions
		WHERE expires_at > NOW() AND revoked = false
	`).Scan(&activeSessions)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to collect active sessions metric")
	} else {
		c.metrics.SessionsActive.Set(float64(activeSessions))
	}
}
