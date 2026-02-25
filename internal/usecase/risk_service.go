package usecase

import (
	"context"
	"math"
	"time"

	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type RiskService struct {
	geoService         ports.GeolocationService
	tenantRepo         ports.TenantRepository
	threatIntelService ports.ThreatIntelligenceService
	config             *config.Config
}

func NewRiskService(
	geoService ports.GeolocationService,
	tenantRepo ports.TenantRepository,
	threatIntelService ports.ThreatIntelligenceService,
	cfg *config.Config,
) *RiskService {
	return &RiskService{
		geoService:         geoService,
		tenantRepo:         tenantRepo,
		threatIntelService: threatIntelService,
		config:             cfg,
	}
}

func (s *RiskService) AssessLoginRisk(ctx context.Context, user *domain.User, currentIP string) (*domain.RiskAssessment, *domain.Geolocation, error) {
	risk := domain.NewRiskAssessment()

	// Get current location
	currentLoc, err := s.geoService.GetLocation(ctx, currentIP)
	if err != nil {
		// Log error but don't fail login, maybe add a small risk score for unknown location
		risk.AddReason("Could not determine current location", 10)
		return risk, nil, nil
	}

	// 1. Check Impossible Travel
	if user.LastLoginAt != nil && user.LastLoginLatitude != nil && user.LastLoginLongitude != nil {
		dist := calculateDistance(*user.LastLoginLatitude, *user.LastLoginLongitude, currentLoc.Latitude, currentLoc.Longitude)
		timeDiff := time.Since(*user.LastLoginAt).Hours()

		if timeDiff > 0 {
			speed := dist / timeDiff
			if speed > 800 { // Over 800 km/h is highly suspicious (plane speed)
				risk.AddReason("Impossible travel detected", 80)
			} else if speed > 300 { // Fast travel
				risk.AddReason("Fast travel detected", 30)
			}
		}
	}

	// 2. Check for new country
	if user.LastLoginCountry != "" && currentLoc.Country != user.LastLoginCountry {
		risk.AddReason("Login from a new country", 20)
	}

	// 3. Check Threat Intelligence
	if s.threatIntelService != nil && s.config != nil && s.config.ThreatIntel.Enabled {
		intel, err := s.threatIntelService.CheckIP(ctx, currentIP)
		if err == nil && intel != nil {
			risk.ThreatIntelligence = intel
			if intel.ReputationScore > 0 {
				points := int(float64(intel.ReputationScore) * 1.0) // 1:1 mapping for now
				risk.AddReason("Suspicious IP reputation ("+intel.Provider+")", points)
			}
		}
	}

	return risk, currentLoc, nil
}

func (s *RiskService) VerifyGeofencing(ctx context.Context, tenantID string, countryCode string) error {
	if countryCode == "" {
		return nil // Cannot verify if country is unknown
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil // Assume no restrictions if tenant not found or error
	}

	settings := tenant.Settings

	// 1. Check blocked countries
	for _, blocked := range settings.BlockedCountries {
		if countryCode == blocked {
			return domain.ErrGeofencingRestriction
		}
	}

	// 2. Check allowed countries (if whitelist is not empty)
	if len(settings.AllowedCountries) > 0 {
		allowed := false
		for _, a := range settings.AllowedCountries {
			if countryCode == a {
				allowed = true
				break
			}
		}
		if !allowed {
			return domain.ErrGeofencingRestriction
		}
	}

	return nil
}

// calculateDistance returns distance in km between two coordinates using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
