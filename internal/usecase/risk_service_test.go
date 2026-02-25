package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/tests/mocks"
)

func TestRiskService_AssessLoginRisk_ThreatIntel(t *testing.T) {
	mockGeo := new(mocks.MockGeolocationService)
	mockTenant := new(mocks.MockTenantRepository)
	mockIntel := new(mocks.MockThreatIntelligenceService)
	cfg := &config.Config{
		ThreatIntel: config.ThreatIntelConfig{
			Enabled: true,
		},
	}

	service := NewRiskService(mockGeo, mockTenant, mockIntel, cfg)

	ctx := context.Background()
	user := &domain.User{
		ID: "user-1",
	}
	ip := "1.2.3.4"

	// Mock geolocation
	mockGeo.On("GetLocation", ctx, ip).Return(&domain.Geolocation{
		Country:   "US",
		Latitude:  40.7128,
		Longitude: -74.0060,
	}, nil)

	// Case 1: High risk IP
	mockIntel.On("CheckIP", ctx, ip).Return(&domain.ThreatIntelligence{
		IP:              ip,
		ReputationScore: 90,
		Provider:        "AbuseIPDB",
	}, nil)

	risk, _, err := service.AssessLoginRisk(ctx, user, ip)

	assert.NoError(t, err)
	assert.True(t, risk.IsBlocked)
	assert.Equal(t, domain.RiskLevelHigh, risk.Level)
	assert.Contains(t, risk.Reasons, "Suspicious IP reputation (AbuseIPDB)")
	assert.Equal(t, 90, risk.ThreatIntelligence.ReputationScore)

	// Case 2: Clean IP
	mockIntel.ExpectedCalls = nil
	mockIntel.On("CheckIP", ctx, ip).Return(&domain.ThreatIntelligence{
		IP:              ip,
		ReputationScore: 0,
		Provider:        "AbuseIPDB",
	}, nil)

	risk, _, err = service.AssessLoginRisk(ctx, user, ip)
	assert.NoError(t, err)
	assert.False(t, risk.IsBlocked)
	assert.Equal(t, domain.RiskLevelLow, risk.Level)
}
