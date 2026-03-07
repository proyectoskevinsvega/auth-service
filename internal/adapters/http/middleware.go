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
		// Extract token from Authorization header or Cookie
		var tokenString string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			// Fallback to cookie
			cookie, err := r.Cookie("access_token")
			if err == nil {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			WriteUnauthorized(w, "missing authorization token")
			return
		}

		// Validate token
		token, err := m.tokenUC.ValidateToken(r.Context(), tokenString)
		if err != nil {
			if err == domain.ErrTokenExpired {
				WriteInvalidToken(w, "token expired")
				return
			}
			if err == domain.ErrTokenRevoked {
				WriteInvalidToken(w, "token revoked")
				return
			}
			WriteUnauthorized(w, "invalid token")
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
				WriteUnauthorized(w, "unauthorized")
				return
			}

			if !token.HasRole(role) {
				WriteForbidden(w, "insufficient permissions: role "+role+" required")
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
				WriteUnauthorized(w, "unauthorized")
				return
			}

			if !token.HasPermission(permission) {
				WriteForbidden(w, "insufficient permissions: "+permission+" required")
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
				WriteUnauthorized(w, "unauthorized")
				return
			}

			if !token.HasScope(scope) {
				WriteForbidden(w, "insufficient scope: "+scope+" required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
