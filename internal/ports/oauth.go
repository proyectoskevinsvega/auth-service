package ports

import (
	"context"
)

type OAuthProvider interface {
	// GetAuthURL returns the OAuth authorization URL
	GetAuthURL(state string) string

	// Exchange exchanges an authorization code for user info
	Exchange(ctx context.Context, code string) (*OAuthUserInfo, error)
}

type OAuthUserInfo struct {
	ProviderID    string
	Email         string
	Name          string
	Picture       string
	EmailVerified bool
}
