package ports

import (
	"context"

	"github.com/vertercloud/auth-service/internal/domain"
)

type GeolocationService interface {
	// GetLocation returns the geolocation info for an IP address
	GetLocation(ctx context.Context, ip string) (*domain.Geolocation, error)
}
