package domain

import (
	"time"
)

// AuditSearchFilter define los criterios para buscar en el log de auditoría
type AuditSearchFilter struct {
	TenantID  string     `json:"tenant_id"`
	UserID    string     `json:"user_id,omitempty"`
	Action    string     `json:"action,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Success   *bool      `json:"success,omitempty"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// GDPRDataExport representa toda la información que el sistema tiene sobre un usuario
type GDPRDataExport struct {
	User           *User           `json:"user"`
	AuditLogs      []AuditLogEntry `json:"audit_logs"`
	ActiveSessions []Session       `json:"active_sessions"`
	ExportedAt     time.Time       `json:"exported_at"`
}

// SOC2AuditReport representa un reporte de seguridad para cumplimiento SOC2
type SOC2AuditReport struct {
	TenantID    string                 `json:"tenant_id"`
	Period      string                 `json:"period"` // ej: "2024-01-01 to 2024-01-31"
	Summary     map[string]interface{} `json:"summary"`
	AdminLogs   []AuditLogEntry        `json:"admin_logs"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// HIPAAReport se enfoca en la integridad y el acceso a funciones críticas
type HIPAAReport struct {
	TenantID       string          `json:"tenant_id"`
	SecurityEvents []AuditLogEntry `json:"security_events"`
	AccessLogs     []AuditLogEntry `json:"access_logs"`
	GeneratedAt    time.Time       `json:"generated_at"`
}
