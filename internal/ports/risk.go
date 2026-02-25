package ports

import (
	"context"

	"github.com/vertercloud/auth-service/internal/domain"
)

type RiskService interface {
	AssessLoginRisk(ctx context.Context, user *domain.User, currentIP string) (*domain.RiskAssessment, *domain.Geolocation, error)
}
