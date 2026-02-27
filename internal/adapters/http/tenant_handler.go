package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/usecase"
)

type TenantHandler struct {
	tenantUC *usecase.TenantUseCase
	logger   zerolog.Logger
	validate *validator.Validate
}

func NewTenantHandler(tenantUC *usecase.TenantUseCase, logger zerolog.Logger) *TenantHandler {
	return &TenantHandler{
		tenantUC: tenantUC,
		logger:   logger,
		validate: validator.New(),
	}
}

// RegisterTenant maneja la solicitud POST /api/v1/tenants/register
func (h *TenantHandler) RegisterTenant(w http.ResponseWriter, r *http.Request) {
	var req RegisterTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error().Err(err).Msg("failed to decode register tenant request")
		http.Error(w, `{"error": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.logger.Error().Err(err).Msg("validation failed for register tenant request")
		http.Error(w, `{"error": "Validation failed"}`, http.StatusBadRequest)
		return
	}

	tenant, user, err := h.tenantUC.Register(r.Context(), usecase.RegisterTenantInput{
		TenantSlug:    req.TenantSlug,
		TenantName:    req.TenantName,
		AdminUsername: req.AdminUser.Username,
		AdminEmail:    req.AdminUser.Email,
		AdminPassword: req.AdminUser.Password,
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to register tenant")
		http.Error(w, `{"error": "Failed to register new organization"}`, http.StatusInternalServerError)
		return
	}

	resp := RegisterTenantResponse{
		TenantID: tenant.ID,
		Slug:     tenant.Slug,
		Admin:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode register tenant response")
	}
}
