package domain

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID               string
	TenantID         string
	ClientID         string
	ClientSecretHash string
	Name             string
	Active           bool
	Scopes           []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type NewClientInput struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	Name         string
	Scopes       []string
}

func NewClient(input NewClientInput, secretHash string) *Client {
	now := time.Now()
	return &Client{
		ID:               uuid.New().String(),
		TenantID:         input.TenantID,
		ClientID:         input.ClientID,
		ClientSecretHash: secretHash,
		Name:             input.Name,
		Active:           true,
		Scopes:           input.Scopes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (c *Client) IsValid() bool {
	return c.Active
}

func (c *Client) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
