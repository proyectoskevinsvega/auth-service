package observability

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBMetricsCollector collects database metrics periodically
type DBMetricsCollector struct {
	pool    *pgxpool.Pool
	metrics *Metrics
	done    chan struct{}
}

// NewDBMetricsCollector creates a new database metrics collector
func NewDBMetricsCollector(pool *pgxpool.Pool, metrics *Metrics) *DBMetricsCollector {
	return &DBMetricsCollector{
		pool:    pool,
		metrics: metrics,
		done:    make(chan struct{}),
	}
}

// Start begins collecting database metrics periodically
func (c *DBMetricsCollector) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
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
func (c *DBMetricsCollector) Stop() {
	close(c.done)
}

// collect gathers current database connection metrics
func (c *DBMetricsCollector) collect() {
	stats := c.pool.Stat()

	// Update connection metrics
	c.metrics.DBConnectionsActive.Set(float64(stats.AcquiredConns()))
	c.metrics.DBConnectionsIdle.Set(float64(stats.IdleConns()))
}

// RecordQuery records a database query execution
func (c *DBMetricsCollector) RecordQuery(ctx context.Context, operation, table string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
	}

	c.metrics.RecordDBQuery(operation, table, status, duration)

	return err
}

// RedisMetricsWrapper wraps Redis client to record metrics
type RedisMetricsWrapper struct {
	metrics *Metrics
}

// NewRedisMetricsWrapper creates a new Redis metrics wrapper
func NewRedisMetricsWrapper(metrics *Metrics) *RedisMetricsWrapper {
	return &RedisMetricsWrapper{
		metrics: metrics,
	}
}

// RecordCommand records a Redis command execution
func (w *RedisMetricsWrapper) RecordCommand(command string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
		if command == "ping" || command == "connect" {
			w.metrics.RedisConnectionErrors.Inc()
		}
	}

	w.metrics.RecordRedisCommand(command, status, duration)

	return err
}
