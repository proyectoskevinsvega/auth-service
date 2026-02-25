package oauth

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"github.com/vertercloud/auth-service/internal/ports"
)

type GitHubProvider struct {
	config *oauth2.Config
}

func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
	}
}

func (p *GitHubProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *GitHubProvider) Exchange(ctx context.Context, code string) (*ports.OAuthUserInfo, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := p.config.Client(ctx, token)

	// Get user info
	userResp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	var githubUser struct {
		ID        int64  `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// If email is not public, fetch from emails endpoint
	if githubUser.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			return nil, fmt.Errorf("failed to get emails: %w", err)
		}
		defer emailResp.Body.Close()

		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}

		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
			return nil, fmt.Errorf("failed to decode emails: %w", err)
		}

		// Find primary verified email
		for _, email := range emails {
			if email.Primary && email.Verified {
				githubUser.Email = email.Email
				break
			}
		}
	}

	return &ports.OAuthUserInfo{
		ProviderID:    fmt.Sprintf("%d", githubUser.ID),
		Email:         githubUser.Email,
		Name:          githubUser.Name,
		Picture:       githubUser.AvatarURL,
		EmailVerified: true, // GitHub emails are already verified
	}, nil
}
