package health

import (
	"context"
	"time"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

type Check struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HealthChecker interface {
	Check(ctx context.Context) Check
}

type DatabaseChecker struct {
	ping func(context.Context) error
}

func NewDatabaseChecker(ping func(context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{ping: ping}
}

func (c *DatabaseChecker) Check(ctx context.Context) Check {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.ping(ctx); err != nil {
		return Check{
			Name:   "database",
			Status: StatusUnhealthy,
			Error:  err.Error(),
		}
	}

	return Check{
		Name:   "database",
		Status: StatusHealthy,
	}
}
