package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/usecase"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
	TokenContextKey contextKey = "token"
)

type AuthMiddleware struct {
	tokenUC *usecase.TokenUseCase
}

func NewAuthMiddleware(tokenUC *usecase.TokenUseCase) *AuthMiddleware {
	return &AuthMiddleware{
		tokenUC: tokenUC,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "missing authorization header", "UNAUTHORIZED")
			return
		}

		// Check Bearer format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondWithError(w, http.StatusUnauthorized, "invalid authorization format", "UNAUTHORIZED")
			return
		}

		tokenString := parts[1]

		// Validate token
		token, err := m.tokenUC.ValidateToken(r.Context(), tokenString)
		if err != nil {
			if err == domain.ErrTokenExpired {
				respondWithError(w, http.StatusUnauthorized, "token expired", "TOKEN_EXPIRED")
				return
			}
			if err == domain.ErrTokenRevoked {
				respondWithError(w, http.StatusUnauthorized, "token revoked", "TOKEN_REVOKED")
				return
			}
			respondWithError(w, http.StatusUnauthorized, "invalid token", "INVALID_TOKEN")
			return
		}

		// Add token to context
		ctx := context.WithValue(r.Context(), TokenContextKey, token)
		ctx = context.WithValue(ctx, UserContextKey, token.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserContextKey).(string)
	return userID, ok
}

// GetTokenFromContext extracts token from context
func GetTokenFromContext(ctx context.Context) (*domain.Token, bool) {
	token, ok := ctx.Value(TokenContextKey).(*domain.Token)
	return token, ok
}
