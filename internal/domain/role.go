package domain

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func NewRole(name, description string) *Role {
	now := time.Now()
	return &Role{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Permissions: []Permission{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func NewPermission(name, description string) *Permission {
	now := time.Now()
	return &Permission{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
