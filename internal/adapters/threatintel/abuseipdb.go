package threatintel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
)

type AbuseIPDBAdapter struct {
	apiKey string
	client *http.Client
}

func NewAbuseIPDBAdapter(cfg *config.Config) *AbuseIPDBAdapter {
	return &AbuseIPDBAdapter{
		apiKey: cfg.ThreatIntel.APIKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type abuseIPDBResponse struct {
	Data struct {
		IPAddress            string     `json:"ipAddress"`
		AbuseConfidenceScore int        `json:"abuseConfidenceScore"`
		LastReportedAt       *time.Time `json:"lastReportedAt"`
	} `json:"data"`
}

func (a *AbuseIPDBAdapter) CheckIP(ctx context.Context, ip string) (*domain.ThreatIntelligence, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("AbuseIPDB API key not configured")
	}

	url := fmt.Sprintf("https://api.abuseipdb.com/api/v2/check?ipAddress=%s", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Key", a.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AbuseIPDB API error: %s", resp.Status)
	}

	var result abuseIPDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &domain.ThreatIntelligence{
		IP:              result.Data.IPAddress,
		ReputationScore: result.Data.AbuseConfidenceScore,
		LastReportedAt:  result.Data.LastReportedAt,
		Provider:        "AbuseIPDB",
	}, nil
}
