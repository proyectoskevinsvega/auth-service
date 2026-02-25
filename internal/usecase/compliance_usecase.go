package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type ComplianceUseCase struct {
	userRepo    ports.UserRepository
	auditRepo   ports.AuditLogRepository
	sessionRepo ports.SessionRepository
	logger      zerolog.Logger
}

func NewComplianceUseCase(
	userRepo ports.UserRepository,
	auditRepo ports.AuditLogRepository,
	sessionRepo ports.SessionRepository,
	logger zerolog.Logger,
) *ComplianceUseCase {
	return &ComplianceUseCase{
		userRepo:    userRepo,
		auditRepo:   auditRepo,
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

func (uc *ComplianceUseCase) GenerateGDPRReport(ctx context.Context, tenantID, userID string) (*domain.GDPRDataExport, error) {
	uc.logger.Info().Str("user_id", userID).Msg("generating GDPR report")

	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Logs for the user (last 100 entries)
	logs, _, err := uc.auditRepo.Search(ctx, domain.AuditSearchFilter{
		TenantID: tenantID,
		UserID:   userID,
		Limit:    100,
	})
	if err != nil {
		uc.logger.Warn().Err(err).Msg("failed to fetch audit logs for GDPR report")
	}

	sessions, err := uc.sessionRepo.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		uc.logger.Warn().Err(err).Msg("failed to fetch sessions for GDPR report")
	}

	// Transform []AuditLogEntry to match domain.GDPRDataExport if needed
	// AuditSearch returns []*AuditLogEntry, GDPRDataExport wants []AuditLogEntry
	var auditEntries []domain.AuditLogEntry
	for _, l := range logs {
		auditEntries = append(auditEntries, *l)
	}

	var sessionEntries []domain.Session
	for _, s := range sessions {
		sessionEntries = append(sessionEntries, *s)
	}

	return &domain.GDPRDataExport{
		User:           user,
		AuditLogs:      auditEntries,
		ActiveSessions: sessionEntries,
		ExportedAt:     time.Now(),
	}, nil
}

func (uc *ComplianceUseCase) GenerateSOC2Report(ctx context.Context, tenantID string, startDate, endDate time.Time) (*domain.SOC2AuditReport, error) {
	uc.logger.Info().Str("tenant_id", tenantID).Msg("generating SOC2 report")

	// Filter for administrative actions
	adminLogs, _, err := uc.auditRepo.Search(ctx, domain.AuditSearchFilter{
		TenantID:  tenantID,
		StartDate: &startDate,
		EndDate:   &endDate,
		Action:    "admin_", // Simplificación
		Limit:     500,
	})
	if err != nil {
		return nil, err
	}

	// Security summary
	failedLogins, count, _ := uc.auditRepo.Search(ctx, domain.AuditSearchFilter{
		TenantID:  tenantID,
		Action:    "auth_login_failed",
		StartDate: &startDate,
		EndDate:   &endDate,
		Success:   func() *bool { b := false; return &b }(),
		Limit:     1,
	})
	_ = failedLogins // just to get the count

	var adminEntries []domain.AuditLogEntry
	for _, l := range adminLogs {
		adminEntries = append(adminEntries, *l)
	}

	return &domain.SOC2AuditReport{
		TenantID: tenantID,
		Period:   fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		Summary: map[string]interface{}{
			"failed_login_attempts": count,
		},
		AdminLogs:   adminEntries,
		GeneratedAt: time.Now(),
	}, nil
}

func (uc *ComplianceUseCase) GenerateHIPAAReport(ctx context.Context, tenantID string, startDate, endDate time.Time) (*domain.HIPAAReport, error) {
	uc.logger.Info().Str("tenant_id", tenantID).Msg("generating HIPAA report")

	// 1. Security events (critical)
	securityLogs, _, err := uc.auditRepo.Search(ctx, domain.AuditSearchFilter{
		TenantID:  tenantID,
		StartDate: &startDate,
		EndDate:   &endDate,
		Success:   func() *bool { b := false; return &b }(), // Failed actions
		Limit:     500,
	})
	if err != nil {
		return nil, err
	}

	// 2. Access logs (critical data access / login)
	accessLogs, _, err := uc.auditRepo.Search(ctx, domain.AuditSearchFilter{
		TenantID:  tenantID,
		StartDate: &startDate,
		EndDate:   &endDate,
		Action:    "auth_login_success",
		Limit:     500,
	})
	if err != nil {
		uc.logger.Warn().Err(err).Msg("failed to fetch access logs for HIPAA report")
	}

	var securityEntries []domain.AuditLogEntry
	for _, l := range securityLogs {
		securityEntries = append(securityEntries, *l)
	}

	var accessEntries []domain.AuditLogEntry
	for _, l := range accessLogs {
		accessEntries = append(accessEntries, *l)
	}

	return &domain.HIPAAReport{
		TenantID:       tenantID,
		SecurityEvents: securityEntries,
		AccessLogs:     accessEntries,
		GeneratedAt:    time.Now(),
	}, nil
}
