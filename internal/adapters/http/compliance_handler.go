package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// GenerateGDPRReport godoc
// @Summary      Generar reporte GDPR para un usuario
// @Description  Extrae toda la información disponible de un usuario, cumpliendo con el derecho de portabilidad de datos. Requiere privilegios de Admin.
// @Tags         Compliance
// @Produce      json
// @Security     BearerAuth
// @Param        userID path string true "ID del Usuario"
// @Success      200 {object} domain.GDPRDataExport
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /admin/compliance/gdpr/{userID} [get]
func (h *Handler) GenerateGDPRReport(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	tenantID := h.getTenantID(r) // Assuming this helper exists or I need to implement it

	report, err := h.complianceUC.GenerateGDPRReport(r.Context(), tenantID, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to generate GDPR report")
		WriteInternalError(w, "Failed to generate report")
		return
	}

	respondWithJSON(w, http.StatusOK, report)
}

// GenerateSOC2Report godoc
// @Summary      Generar reporte SOC2 de seguridad
// @Description  Genera un resumen de eventos de seguridad y logs administrativos para un periodo dado. Requiere privilegios de Admin.
// @Tags         Compliance
// @Produce      json
// @Security     BearerAuth
// @Param        start_date query string false "Fecha inicio (YYYY-MM-DD)"
// @Param        end_date   query string false "Fecha fin (YYYY-MM-DD)"
// @Success      200 {object} domain.SOC2AuditReport
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /admin/compliance/soc2 [get]
func (h *Handler) GenerateSOC2Report(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")

	startDate := time.Now().AddDate(0, -1, 0) // Default last month
	endDate := time.Now()

	if startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = t
		}
	}

	report, err := h.complianceUC.GenerateSOC2Report(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate SOC2 report")
		WriteInternalError(w, "Failed to generate report")
		return
	}

	respondWithJSON(w, http.StatusOK, report)
}

// GenerateHIPAAReport godoc
// @Summary      Generar reporte HIPAA de integridad
// @Description  Genera un reporte de integridad de datos y eventos de seguridad críticos para cumplimiento HIPAA. Requiere privilegios de Admin.
// @Tags         Compliance
// @Produce      json
// @Security     BearerAuth
// @Param        start_date query string false "Fecha inicio (YYYY-MM-DD)"
// @Param        end_date   query string false "Fecha fin (YYYY-MM-DD)"
// @Success      200 {object} domain.HIPAAReport
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /admin/compliance/hipaa [get]
func (h *Handler) GenerateHIPAAReport(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")

	startDate := time.Now().AddDate(0, -1, 0) // Default last month
	endDate := time.Now()

	if startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = t
		}
	}

	report, err := h.complianceUC.GenerateHIPAAReport(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate HIPAA report")
		WriteInternalError(w, "Failed to generate report")
		return
	}

	respondWithJSON(w, http.StatusOK, report)
}

// Helper to get tenant ID from context/headers (consistent with existing isolation)
func (h *Handler) getTenantID(r *http.Request) string {
	// This usually comes from a middleware that sets it in context
	// For now, extraction from context if available, or fallback to a header/param
	// Checking common patterns in this codebase
	if tenantID, ok := r.Context().Value("tenant_id").(string); ok {
		return tenantID
	}
	return r.Header.Get("X-Tenant-ID")
}
