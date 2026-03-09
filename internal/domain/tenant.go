package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        string
	Slug      string // e.g. 'acme'
	Name      string // e.g. 'Acme Corp'
	Active    bool
	Settings  TenantSettings
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TenantSettings struct {
	AllowedCountries         []string `json:"allowed_countries"`
	BlockedCountries         []string `json:"blocked_countries"`
	EnforceSessionGeofencing bool     `json:"enforce_session_geofencing"`
	DefaultRoleID            string   `json:"default_role_id"`
}

type NewTenantInput struct {
	Slug string
	Name string
}

func NewTenant(input NewTenantInput) *Tenant {
	now := time.Now()
	return &Tenant{
		ID:     uuid.Must(uuid.NewV7()).String(),
		Slug:   input.Slug,
		Name:   input.Name,
		Active: true,
		Settings: TenantSettings{
			AllowedCountries:         []string{},
			BlockedCountries:         []string{},
			EnforceSessionGeofencing: false,
			DefaultRoleID:            "", // Se debe llenar vía Admin API o Base de datos
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
