package ports

import (
	"context"

	"github.com/vertercloud/auth-service/internal/domain"
)

type ThreatIntelligenceService interface {
	CheckIP(ctx context.Context, ip string) (*domain.ThreatIntelligence, error)
}
