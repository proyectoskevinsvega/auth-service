package domain

import (
	"time"
)

type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

type RiskAssessment struct {
	Score              int
	Level              RiskLevel
	Reasons            []string
	IsBlocked          bool
	RequireMFA         bool
	ThreatIntelligence *ThreatIntelligence
}

type ThreatIntelligence struct {
	IP              string
	ReputationScore int // 0-100 (AbuseIPDB style)
	LastReportedAt  *time.Time
	Provider        string
}

func NewRiskAssessment() *RiskAssessment {
	return &RiskAssessment{
		Score:   0,
		Level:   RiskLevelLow,
		Reasons: make([]string, 0),
	}
}

func (r *RiskAssessment) AddReason(reason string, points int) {
	r.Reasons = append(r.Reasons, reason)
	r.Score += points
	r.updateLevel()
}

func (r *RiskAssessment) updateLevel() {
	if r.Score >= 80 {
		r.Level = RiskLevelHigh
		r.IsBlocked = true
	} else if r.Score >= 40 {
		r.Level = RiskLevelMedium
		r.RequireMFA = true
	} else {
		r.Level = RiskLevelLow
	}
}
