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
	UserContextKey   contextKey = "user"
	TokenContextKey  contextKey = "token"
	TenantContextKey contextKey = "tenant"
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
		ctx = context.WithValue(ctx, TenantContextKey, token.TenantID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserContextKey).(string)
	return userID, ok
}

// GetTenantIDFromContext extracts tenant ID from context
func GetTenantIDFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(TenantContextKey).(string)
	return tenantID, ok
}

// GetTokenFromContext extracts token from context
func GetTokenFromContext(ctx context.Context) (*domain.Token, bool) {
	token, ok := ctx.Value(TokenContextKey).(*domain.Token)
	return token, ok
}

func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetTokenFromContext(r.Context())
			if !ok {
				respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
				return
			}

			if !token.HasRole(role) {
				respondWithError(w, http.StatusForbidden, "insufficient permissions: role "+role+" required", "FORBIDDEN")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetTokenFromContext(r.Context())
			if !ok {
				respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
				return
			}

			if !token.HasPermission(permission) {
				respondWithError(w, http.StatusForbidden, "insufficient permissions: permission "+permission+" required", "FORBIDDEN")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *AuthMiddleware) RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetTokenFromContext(r.Context())
			if !ok {
				respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
				return
			}

			if !token.HasScope(scope) {
				respondWithError(w, http.StatusForbidden, "insufficient scope: "+scope+" required", "FORBIDDEN")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
