package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/observability/telemetry"
	"github.com/vertercloud/auth-service/internal/ports"
	"github.com/vertercloud/auth-service/internal/usecase"

	_ "github.com/vertercloud/auth-service/docs" // Swagger docs
)

type Handler struct {
	authUC              *usecase.AuthUseCase
	tokenUC             *usecase.TokenUseCase
	sessionUC           *usecase.SessionUseCase
	twofaUC             *usecase.TwoFAUseCase
	emailVerificationUC *usecase.EmailVerificationUseCase
	userRepo            ports.UserRepository
	googleOAuth         ports.OAuthProvider
	githubOAuth         ports.OAuthProvider
	jwtService          ports.JWTService
	logger              zerolog.Logger
	authMiddleware      *AuthMiddleware
	allowedOrigins      []string
	environment         string
	jwtIssuer           string
	baseDomain          string
}

func NewHandler(
	authUC *usecase.AuthUseCase,
	tokenUC *usecase.TokenUseCase,
	sessionUC *usecase.SessionUseCase,
	twofaUC *usecase.TwoFAUseCase,
	emailVerificationUC *usecase.EmailVerificationUseCase,
	userRepo ports.UserRepository,
	googleOAuth ports.OAuthProvider,
	githubOAuth ports.OAuthProvider,
	jwtService ports.JWTService,
	logger zerolog.Logger,
	allowedOrigins []string,
	environment string,
	jwtIssuer string,
	baseDomain string,
) *Handler {
	return &Handler{
		authUC:              authUC,
		tokenUC:             tokenUC,
		sessionUC:           sessionUC,
		twofaUC:             twofaUC,
		emailVerificationUC: emailVerificationUC,
		userRepo:            userRepo,
		googleOAuth:         googleOAuth,
		githubOAuth:         githubOAuth,
		jwtService:          jwtService,
		logger:              logger,
		authMiddleware:      NewAuthMiddleware(tokenUC),
		allowedOrigins:      allowedOrigins,
		environment:         environment,
		jwtIssuer:           jwtIssuer,
		baseDomain:          baseDomain,
	}
}

func (h *Handler) SetupRoutes(telemetryEnabled bool) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Add telemetry middleware BEFORE logger so trace IDs are available in logs
	if telemetryEnabled {
		r.Use(telemetry.HTTPMiddleware("auth-service"))
	}

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   h.allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Security Headers
	r.Use(SecurityHeaders(h.environment))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/refresh", h.RefreshToken)
		r.Post("/auth/forgot-password", h.ForgotPassword)
		r.Post("/auth/reset-password", h.ResetPassword)
		r.Post("/auth/reset-password-code", h.ResetPasswordWithCode)
		r.Post("/auth/verify-email", h.VerifyEmail)
		r.Get("/auth/verify-email", h.VerifyEmailGET) // Support GET for email links

		// OAuth routes
		r.Get("/auth/oauth/google", h.GoogleOAuthStart)
		r.Get("/auth/oauth/google/callback", h.GoogleOAuthCallback)
		r.Get("/auth/oauth/github", h.GitHubOAuthStart)
		r.Get("/auth/oauth/github/callback", h.GitHubOAuthCallback)

		// Public key endpoint
		r.Get("/auth/.well-known/jwks.json", h.GetJWKS)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(h.authMiddleware.RequireAuth)

			r.Post("/auth/logout", h.Logout)
			r.Get("/auth/me", h.GetMe)
			r.Put("/auth/me", h.UpdateMe)

			// Sessions
			r.Get("/auth/sessions", h.ListSessions)
			r.Delete("/auth/sessions/{id}", h.RevokeSession)
			r.Delete("/auth/sessions/all", h.RevokeAllSessions)

			// 2FA
			r.Post("/auth/2fa/enable", h.Enable2FA)
			r.Post("/auth/2fa/verify", h.Verify2FA)
			r.Post("/auth/2fa/disable", h.Disable2FA)

			// Email verification (resend)
			r.Post("/auth/resend-verification", h.ResendVerificationEmail)

			// OIDC UserInfo
			r.Get("/auth/userinfo", h.GetUserInfo)
		})

		// OIDC Discovery (.well-known)
		r.Get("/.well-known/openid-configuration", h.GetOIDCConfiguration)

		// Admin routes (Protected by AuthMiddleware for now, further RBAC can be added)
		r.Group(func(r chi.Router) {
			r.Use(h.authMiddleware.RequireAuth)
			// TODO: Add AdminRoleMiddleware if implemented
			r.Post("/admin/users/{id}/force-reset", h.AdminForcePasswordReset)
		})
	})

	// Health check
	r.Get("/health", h.Health)

	// Swagger documentation with custom auth interceptor
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.BeforeScript(`
			// Automatically prepend "Bearer " to Authorization header
			window.onload = function() {
				const ui = window.ui;
				const OriginalRequestInterceptor = ui.fn.fetch;

				ui.fn.fetch = function(req) {
					if (req.headers && req.headers.Authorization) {
						const authValue = req.headers.Authorization;
						if (!authValue.startsWith('Bearer ')) {
							req.headers.Authorization = 'Bearer ' + authValue;
						}
					}
					return OriginalRequestInterceptor.call(this, req);
				};
			};
		`),
	))

	return r
}

// Register godoc
// @Summary      Registrar nuevo usuario
// @Description  Crea una nueva cuenta de usuario con username, email y contraseña. Envía un email de verificación de forma asíncrona.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Datos de registro"
// @Success      201  {object}  UserResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse "Email o username ya existe"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "username, email and password are required", "BAD_REQUEST")
		return
	}

	// Get IP address
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	user, err := h.authUC.Register(r.Context(), usecase.RegisterInput{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Send verification email (async, don't block registration if it fails)
	// Use background context to prevent cancellation when request ends
	go func() {
		if err := h.emailVerificationUC.SendVerificationEmail(context.Background(), usecase.SendVerificationInput{
			UserID:    user.ID,
			IPAddress: ipAddress,
			UserAgent: userAgent,
		}); err != nil {
			h.logger.Error().Err(err).Str("user_id", user.ID).Msg("failed to send verification email after registration")
		}
	}()

	respondWithJSON(w, http.StatusCreated, UserResponse{
		ID:               user.ID,
		Username:         user.Username,
		Email:            user.Email,
		Active:           user.Active,
		EmailVerified:    user.EmailVerified,
		TwoFactorEnabled: user.TwoFactorEnabled,
		CreatedAt:        user.CreatedAt.Format(time.RFC3339),
	})
}

// Login godoc
// @Summary      Iniciar sesión
// @Description  Autentica al usuario con email/username y contraseña. Retorna tokens JWT de acceso y actualización. Soporta 2FA si está habilitado.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Credenciales de inicio de sesión"
// @Success      200  {object}  LoginResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Credenciales inválidas"
// @Failure      403  {object}  ErrorResponse "Cuenta inactiva o email no verificado"
// @Failure      429  {object}  ErrorResponse "Demasiados intentos de inicio de sesión"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if req.Identifier == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "identifier and password are required", "BAD_REQUEST")
		return
	}

	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	response, err := h.authUC.Login(r.Context(), usecase.LoginInput{
		Identifier: req.Identifier,
		Password:   req.Password,
		TwoFACode:  req.TwoFACode,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Device:     extractDevice(userAgent),
	})

	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		User: UserResponse{
			ID:               response.User.ID,
			Username:         response.User.Username,
			Email:            response.User.Email,
			Active:           response.User.Active,
			EmailVerified:    response.User.EmailVerified,
			TwoFactorEnabled: response.User.TwoFactorEnabled,
			CreatedAt:        response.User.CreatedAt.Format(time.RFC3339),
		},
	})
}

// RefreshToken handles token refresh
// @Summary      Renovar token de acceso
// @Description  Genera un nuevo access token usando un refresh token válido. Implementa rotación automática de refresh tokens.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest true "Refresh token"
// @Success      200 {object} RefreshTokenResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse "Refresh token inválido o expirado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/refresh [post]
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if req.RefreshToken == "" {
		respondWithError(w, http.StatusBadRequest, "refresh_token is required", "BAD_REQUEST")
		return
	}

	response, err := h.tokenUC.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, RefreshTokenResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
	})
}

// Logout godoc
// @Summary      Cerrar sesión
// @Description  Revoca el token de acceso actual agregándolo a la lista negra. El token no podrá ser usado nuevamente.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} MessageResponse
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Extract token from context
	token, ok := GetTokenFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	// Get the actual token string from header
	authHeader := r.Header.Get("Authorization")
	tokenString := strings.Split(authHeader, " ")[1]

	if err := h.tokenUC.RevokeToken(r.Context(), tokenString); err != nil {
		h.logger.Error().Err(err).Str("jti", token.JTI).Msg("failed to revoke token")
		respondWithError(w, http.StatusInternalServerError, "failed to logout", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "logged out successfully",
	})
}

// ForgotPassword godoc
// @Summary      Solicitar restablecimiento de contraseña
// @Description  Envía un email con un código de 6 dígitos y un enlace para restablecer la contraseña. El código expira en 15 minutos. Siempre retorna éxito para evitar enumeración de emails.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body ForgotPasswordRequest true "Email del usuario"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /auth/forgot-password [post]
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if req.Email == "" {
		respondWithError(w, http.StatusBadRequest, "email is required", "BAD_REQUEST")
		return
	}

	ipAddress := r.RemoteAddr

	if err := h.authUC.ForgotPassword(r.Context(), req.Email, ipAddress); err != nil {
		h.logger.Error().Err(err).Msg("forgot password failed")
		// Always return success to prevent email enumeration
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "if the email exists, a reset link will be sent",
	})
}

// ResetPassword godoc
// @Summary      Restablecer contraseña con token
// @Description  Restablece la contraseña usando el token recibido por email (URL). El token es válido por 15 minutos.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordRequest true "Token y nueva contraseña"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse "Token inválido o expirado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/reset-password [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		respondWithError(w, http.StatusBadRequest, "token and new_password are required", "BAD_REQUEST")
		return
	}

	ipAddress := r.RemoteAddr

	if err := h.authUC.ResetPasswordWithToken(r.Context(), req.Token, req.NewPassword, ipAddress); err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "password reset successfully",
	})
}

// ResetPasswordWithCode godoc
// @Summary      Restablecer contraseña con código
// @Description  Restablece la contraseña usando el código de 6 dígitos recibido por email. El código es válido por 15 minutos.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordWithCodeRequest true "Email, código de 6 dígitos y nueva contraseña"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse "Código inválido o expirado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/reset-password-code [post]
func (h *Handler) ResetPasswordWithCode(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordWithCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if req.Email == "" || req.Code == "" || req.NewPassword == "" {
		respondWithError(w, http.StatusBadRequest, "email, code and new_password are required", "BAD_REQUEST")
		return
	}

	ipAddress := r.RemoteAddr

	if err := h.authUC.ResetPasswordWithCode(r.Context(), req.Email, req.Code, req.NewPassword, ipAddress); err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "password reset successfully",
	})
}

// GetMe godoc
// @Summary      Obtener perfil del usuario actual
// @Description  Retorna la información del usuario autenticado actualmente
// @Tags         User Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} UserResponse
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/me [get]
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get user")
		respondWithError(w, http.StatusInternalServerError, "failed to get user", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, UserResponse{
		ID:               user.ID,
		Username:         user.Username,
		Email:            user.Email,
		Active:           user.Active,
		EmailVerified:    user.EmailVerified,
		TwoFactorEnabled: user.TwoFactorEnabled,
		CreatedAt:        user.CreatedAt.Format(time.RFC3339),
	})
}

// UpdateMe godoc
// @Summary      Actualizar perfil del usuario
// @Description  Actualiza el email y/o username del usuario autenticado. Si se cambia el email, se requiere nueva verificación.
// @Tags         User Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateUserRequest true "Datos a actualizar (email y/o username)"
// @Success      200 {object} UserResponse
// @Failure      400 {object} ErrorResponse "Datos inválidos"
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      409 {object} ErrorResponse "Email o username ya existe"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/me [put]
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	// Get current user
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get user")
		respondWithError(w, http.StatusInternalServerError, "failed to get user", "INTERNAL_ERROR")
		return
	}

	// Update fields if provided
	if req.Email != "" && req.Email != user.Email {
		// Check if email already exists
		existingUser, _ := h.userRepo.GetByEmail(r.Context(), req.Email)
		if existingUser != nil {
			respondWithError(w, http.StatusConflict, "email already exists", "EMAIL_EXISTS")
			return
		}
		user.Email = req.Email
		user.EmailVerified = false // Reset verification when email changes
	}

	if req.Username != "" && req.Username != user.Username {
		// Validate username
		if err := domain.ValidateUsername(req.Username); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error(), "INVALID_USERNAME")
			return
		}
		// Check if username already exists
		existingUser, _ := h.userRepo.GetByUsername(r.Context(), req.Username)
		if existingUser != nil {
			respondWithError(w, http.StatusConflict, "username already exists", "USERNAME_EXISTS")
			return
		}
		user.Username = req.Username
	}

	user.UpdatedAt = time.Now()

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to update user")
		respondWithError(w, http.StatusInternalServerError, "failed to update user", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, UserResponse{
		ID:               user.ID,
		Username:         user.Username,
		Email:            user.Email,
		Active:           user.Active,
		EmailVerified:    user.EmailVerified,
		TwoFactorEnabled: user.TwoFactorEnabled,
		CreatedAt:        user.CreatedAt.Format(time.RFC3339),
	})
}

// GoogleOAuthStart godoc
// @Summary      Iniciar autenticación con Google
// @Description  Redirige al usuario a la página de autenticación de Google OAuth
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Success      307 {string} string "Redirección a Google OAuth"
// @Router       /auth/oauth/google [get]
func (h *Handler) GoogleOAuthStart(w http.ResponseWriter, r *http.Request) {
	authURL := h.googleOAuth.GetAuthURL("")
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// GoogleOAuthCallback godoc
// @Summary      Callback de Google OAuth
// @Description  Procesa el callback de Google OAuth. Crea usuario si no existe o inicia sesión si ya existe. Retorna tokens JWT.
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Param        code  query string true "Código de autorización de Google"
// @Param        state query string false "Estado de la solicitud OAuth"
// @Success      200 {object} LoginResponse
// @Failure      400 {object} ErrorResponse "Código de autorización faltante"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/oauth/google/callback [get]
func (h *Handler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "missing authorization code", "BAD_REQUEST")
		return
	}

	state := r.URL.Query().Get("state")
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	response, err := h.authUC.OAuthLogin(
		r.Context(),
		"google",
		code,
		state,
		ipAddress,
		userAgent,
		extractDevice(userAgent),
	)

	if err != nil {
		h.logger.Error().Err(err).Msg("google oauth callback failed")
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		User: UserResponse{
			ID:               response.User.ID,
			Username:         response.User.Username,
			Email:            response.User.Email,
			Active:           response.User.Active,
			EmailVerified:    response.User.EmailVerified,
			TwoFactorEnabled: response.User.TwoFactorEnabled,
			CreatedAt:        response.User.CreatedAt.Format(time.RFC3339),
		},
	})
}

// GitHubOAuthStart godoc
// @Summary      Iniciar autenticación con GitHub
// @Description  Redirige al usuario a la página de autenticación de GitHub OAuth
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Success      307 {string} string "Redirección a GitHub OAuth"
// @Router       /auth/oauth/github [get]
func (h *Handler) GitHubOAuthStart(w http.ResponseWriter, r *http.Request) {
	authURL := h.githubOAuth.GetAuthURL("")
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// GitHubOAuthCallback godoc
// @Summary      Callback de GitHub OAuth
// @Description  Procesa el callback de GitHub OAuth. Crea usuario si no existe o inicia sesión si ya existe. Retorna tokens JWT.
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Param        code  query string true "Código de autorización de GitHub"
// @Param        state query string false "Estado de la solicitud OAuth"
// @Success      200 {object} LoginResponse
// @Failure      400 {object} ErrorResponse "Código de autorización faltante"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/oauth/github/callback [get]
func (h *Handler) GitHubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "missing authorization code", "BAD_REQUEST")
		return
	}

	state := r.URL.Query().Get("state")
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	response, err := h.authUC.OAuthLogin(
		r.Context(),
		"github",
		code,
		state,
		ipAddress,
		userAgent,
		extractDevice(userAgent),
	)

	if err != nil {
		h.logger.Error().Err(err).Msg("github oauth callback failed")
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		User: UserResponse{
			ID:               response.User.ID,
			Username:         response.User.Username,
			Email:            response.User.Email,
			Active:           response.User.Active,
			EmailVerified:    response.User.EmailVerified,
			TwoFactorEnabled: response.User.TwoFactorEnabled,
			CreatedAt:        response.User.CreatedAt.Format(time.RFC3339),
		},
	})
}

// ListSessions godoc
// @Summary      Listar sesiones activas
// @Description  Retorna todas las sesiones activas del usuario autenticado, marcando la sesión actual
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} SessionsResponse
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/sessions [get]
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	token, ok := GetTokenFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	sessions, err := h.sessionUC.ListUserSessions(r.Context(), userID, token.JTI)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list sessions")
		respondWithError(w, http.StatusInternalServerError, "failed to list sessions", "INTERNAL_ERROR")
		return
	}

	// Convert to response format
	sessionResponses := make([]SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, SessionResponse{
			ID:         session.ID,
			IPAddress:  session.IPAddress,
			Country:    session.Country,
			Device:     session.Device,
			UserAgent:  session.UserAgent,
			CreatedAt:  session.CreatedAt.Format(time.RFC3339),
			LastUsedAt: session.LastUsedAt.Format(time.RFC3339),
			IsCurrent:  session.IsCurrent,
		})
	}

	respondWithJSON(w, http.StatusOK, SessionsResponse{
		Sessions: sessionResponses,
	})
}

// RevokeSession godoc
// @Summary      Revocar una sesión específica
// @Description  Revoca una sesión específica por su ID. No permite revocar la sesión actual (usar logout).
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID de la sesión a revocar"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse "ID de sesión requerido"
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      404 {object} ErrorResponse "Sesión no encontrada"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/sessions/{id} [delete]
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		respondWithError(w, http.StatusBadRequest, "session id is required", "BAD_REQUEST")
		return
	}

	if err := h.sessionUC.RevokeSession(r.Context(), userID, sessionID); err != nil {
		if err == domain.ErrSessionNotFound {
			respondWithError(w, http.StatusNotFound, "session not found", "SESSION_NOT_FOUND")
			return
		}
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to revoke session")
		respondWithError(w, http.StatusInternalServerError, "failed to revoke session", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "session revoked successfully",
	})
}

// RevokeAllSessions godoc
// @Summary      Revocar todas las sesiones excepto la actual
// @Description  Revoca todas las sesiones del usuario excepto la sesión actual. Útil para cerrar sesión en todos los dispositivos.
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Retorna mensaje y cantidad de sesiones revocadas"
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/sessions/all [delete]
func (h *Handler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	token, ok := GetTokenFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	revokedCount, err := h.sessionUC.RevokeAllSessions(r.Context(), userID, token.JTI)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to revoke all sessions")
		respondWithError(w, http.StatusInternalServerError, "failed to revoke sessions", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "all other sessions revoked successfully",
		"revoked_count": revokedCount,
	})
}

// Enable2FA godoc
// @Summary      Habilitar autenticación de dos factores
// @Description  Genera un secreto TOTP y un código QR para habilitar 2FA. El usuario debe escanear el QR y verificar con Verify2FA.
// @Tags         Two-Factor Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} Enable2FAResponse "Retorna secret y qr_code (base64)"
// @Failure      400 {object} ErrorResponse "2FA ya habilitado"
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/2fa/enable [post]
func (h *Handler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	response, err := h.twofaUC.Enable2FA(r.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to enable 2FA")
		if err.Error() == "2FA already enabled" {
			respondWithError(w, http.StatusBadRequest, "2FA already enabled", "2FA_ALREADY_ENABLED")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to enable 2FA", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, Enable2FAResponse{
		Secret: response.Secret,
		QRCode: response.QRCode,
	})
}

// Verify2FA godoc
// @Summary      Verificar y confirmar 2FA
// @Description  Verifica el código TOTP de 6 dígitos y confirma la habilitación de 2FA para el usuario. Debe llamarse después de Enable2FA.
// @Tags         Two-Factor Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body Verify2FARequest true "Código TOTP de 6 dígitos"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse "Código inválido o faltante"
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/2fa/verify [post]
func (h *Handler) Verify2FA(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if req.Code == "" {
		respondWithError(w, http.StatusBadRequest, "code is required", "BAD_REQUEST")
		return
	}

	if err := h.twofaUC.Verify2FA(r.Context(), userID, req.Code); err != nil {
		if err == domain.ErrInvalidCredentials {
			respondWithError(w, http.StatusBadRequest, "invalid 2FA code", "INVALID_2FA_CODE")
			return
		}
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to verify 2FA")
		respondWithError(w, http.StatusInternalServerError, "failed to verify 2FA", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "2FA enabled successfully",
	})
}

// Disable2FA godoc
// @Summary      Deshabilitar autenticación de dos factores
// @Description  Deshabilita 2FA para el usuario actual. Requiere proporcionar un código TOTP válido para confirmar.
// @Tags         Two-Factor Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body Disable2FARequest true "Código TOTP de 6 dígitos"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse "Código inválido, faltante o 2FA no habilitado"
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/2fa/disable [post]
func (h *Handler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	var req Disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if req.Code == "" {
		respondWithError(w, http.StatusBadRequest, "code is required", "BAD_REQUEST")
		return
	}

	if err := h.twofaUC.Disable2FA(r.Context(), userID, req.Code); err != nil {
		if err == domain.ErrInvalidCredentials {
			respondWithError(w, http.StatusBadRequest, "invalid 2FA code", "INVALID_2FA_CODE")
			return
		}
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to disable 2FA")
		if err.Error() == "2FA not enabled" {
			respondWithError(w, http.StatusBadRequest, "2FA not enabled", "2FA_NOT_ENABLED")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to disable 2FA", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "2FA disabled successfully",
	})
}

// VerifyEmail godoc
// @Summary      Verificar email (POST)
// @Description  Verifica el email del usuario usando el token recibido por correo. El token es válido por 48 horas.
// @Tags         Email Verification
// @Accept       json
// @Produce      json
// @Param        request body object{token=string} true "Token de verificación"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse "Token inválido, expirado o ya usado"
// @Failure      404 {object} ErrorResponse "Token no encontrado"
// @Failure      409 {object} ErrorResponse "Email ya verificado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/verify-email [post]
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if req.Token == "" {
		respondWithError(w, http.StatusBadRequest, "token is required", "MISSING_TOKEN")
		return
	}

	if err := h.emailVerificationUC.VerifyEmail(r.Context(), req.Token); err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "email verified successfully",
	})
}

// VerifyEmailGET godoc
// @Summary      Verificar email (GET)
// @Description  Verifica el email mediante un enlace GET (para clics desde el correo). Retorna HTML con confirmación visual.
// @Tags         Email Verification
// @Accept       json
// @Produce      html
// @Param        token query string true "Token de verificación desde el email"
// @Success      200 {string} string "HTML con mensaje de éxito"
// @Failure      400 {object} ErrorResponse "Token inválido, expirado o ya usado"
// @Failure      404 {object} ErrorResponse "Token no encontrado"
// @Failure      409 {object} ErrorResponse "Email ya verificado"
// @Router       /auth/verify-email [get]
func (h *Handler) VerifyEmailGET(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondWithError(w, http.StatusBadRequest, "token is required", "MISSING_TOKEN")
		return
	}

	if err := h.emailVerificationUC.VerifyEmail(r.Context(), token); err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Return HTML response for better UX when clicking email links
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Email Verified</title>
    <style>
        body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; background: white; padding: 40px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .success { color: #28a745; font-size: 48px; margin-bottom: 20px; }
        h1 { color: #333; margin-bottom: 10px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="success">✓</div>
        <h1>Email Verified Successfully!</h1>
        <p>Your email has been verified. You can now close this window.</p>
    </div>
</body>
</html>`
	w.Write([]byte(html))
}

// ResendVerificationEmail godoc
// @Summary      Reenviar email de verificación
// @Description  Reenvía el email de verificación al usuario autenticado. Solo disponible si el email no está verificado.
// @Tags         Email Verification
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} MessageResponse
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      409 {object} ErrorResponse "Email ya verificado"
// @Failure      429 {object} ErrorResponse "Demasiados intentos"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/resend-verification [post]
func (h *Handler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	if err := h.emailVerificationUC.ResendVerificationEmail(r.Context(), userID, ipAddress, userAgent); err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "verification email sent successfully",
	})
}

// GetJWKS godoc
// @Summary      Obtener claves públicas JWT
// @Description  Retorna las claves públicas en formato JWKS (JSON Web Key Set) para verificar tokens JWT firmados por este servicio
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Success      200 {object} JWKSResponse
// @Failure      500 {object} ErrorResponse
// @Router       /auth/.well-known/jwks.json [get]
func (h *Handler) GetJWKS(w http.ResponseWriter, r *http.Request) {
	jwk, err := h.jwtService.GetPublicKeyJWKS()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get JWKS")
		respondWithError(w, http.StatusInternalServerError, "failed to get JWKS", "INTERNAL_ERROR")
		return
	}

	respondWithJSON(w, http.StatusOK, JWKSResponse{
		Keys: []JWKResponse{
			{
				Kty: jwk["kty"].(string),
				Use: jwk["use"].(string),
				Alg: jwk["alg"].(string),
				N:   jwk["n"].(string),
				E:   jwk["e"].(string),
			},
		},
	})
}

// @Router       /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, HealthResponse{
		Status:  "healthy",
		Service: "auth-service",
		Version: "1.1.0",
	})
}

// AdminForcePasswordReset godoc
// @Summary      Forzar reset de contraseña (Admin)
// @Description  Marca la contraseña de un usuario como expirada y requiere reset. Invalida todas las sesiones activas. Solo para administradores.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID del usuario"
// @Success      200 {object} MessageResponse
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      403 {object} ErrorResponse "No autorizado"
// @Failure      404 {object} ErrorResponse "Usuario no encontrado"
// @Failure      500 {object} ErrorResponse
// @Router       /admin/users/{id}/force-reset [post]
func (h *Handler) AdminForcePasswordReset(w http.ResponseWriter, r *http.Request) {
	// 1. Get user ID from URL
	targetUserID := chi.URLParam(r, "id")
	if targetUserID == "" {
		respondWithError(w, http.StatusBadRequest, "user id is required", "BAD_REQUEST")
		return
	}

	// 2. Call use case
	if err := h.authUC.ForcePasswordReset(r.Context(), targetUserID); err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "forced password reset successfully",
	})
}

// GetOIDCConfiguration godoc
// @Summary      OIDC Discovery
// @Description  Retorna la configuración de OpenID Connect para el descubrimiento automático
// @Tags         OIDC
// @Accept       json
// @Produce      json
// @Success      200 {object} OIDCConfigurationResponse
// @Router       /.well-known/openid-configuration [get]
func (h *Handler) GetOIDCConfiguration(w http.ResponseWriter, r *http.Request) {
	issuer := h.jwtIssuer
	baseURL := "http://" + h.baseDomain
	if h.environment == "production" {
		baseURL = "https://" + h.baseDomain
	}

	config := OIDCConfigurationResponse{
		Issuer:                           issuer,
		AuthorizationEndpoint:            baseURL + "/auth/authorize",
		TokenEndpoint:                    baseURL + "/auth/login",
		UserinfoEndpoint:                 baseURL + "/auth/userinfo",
		JWKSURI:                          baseURL + "/auth/.well-known/jwks.json",
		ScopesSupported:                  []string{"openid", "profile", "email"},
		ResponseTypesSupported:           []string{"code", "token", "id_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		ClaimsSupported:                  []string{"sub", "iss", "name", "email", "email_verified", "preferred_username"},
	}

	respondWithJSON(w, http.StatusOK, config)
}

// GetUserInfo godoc
// @Summary      UserInfo (OIDC)
// @Description  Retorna los claims del usuario autenticado siguiendo el estándar OIDC
// @Tags         OIDC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} UserInfoResponse
// @Failure      401 {object} ErrorResponse "No autenticado"
// @Failure      500 {object} ErrorResponse
// @Router       /auth/userinfo [get]
func (h *Handler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	user, err := h.authUC.GetUserInfo(r.Context(), userID)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, UserInfoResponse{
		Sub:               user.ID,
		Name:              user.Username, // Mapping username to name as fallback
		PreferredUsername: user.Username,
		Email:             user.Email,
		EmailVerified:     user.EmailVerified,
	})
}

// Helper functions
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message, errorCode string) {
	respondWithJSON(w, code, ErrorResponse{
		Error: message,
		Code:  errorCode,
	})
}

func (h *Handler) handleAuthError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrInvalidCredentials:
		respondWithError(w, http.StatusUnauthorized, "invalid credentials", "INVALID_CREDENTIALS")
	case domain.ErrAccountLocked:
		respondWithError(w, http.StatusForbidden, "account is locked due to too many failed attempts", "ACCOUNT_LOCKED")
	case domain.ErrPasswordExpired:
		respondWithError(w, http.StatusForbidden, "password has expired", "PASSWORD_EXPIRED")
	case domain.ErrPasswordResetRequired:
		respondWithError(w, http.StatusForbidden, "password reset is required by admin", "PASSWORD_RESET_REQUIRED")
	case domain.ErrUserNotFound:
		respondWithError(w, http.StatusNotFound, "user not found", "USER_NOT_FOUND")
	case domain.ErrEmailAlreadyExists:
		respondWithError(w, http.StatusConflict, "email already exists", "EMAIL_EXISTS")
	case domain.ErrUsernameAlreadyExists:
		respondWithError(w, http.StatusConflict, "username already exists", "USERNAME_EXISTS")
	case domain.ErrInvalidEmail:
		respondWithError(w, http.StatusBadRequest, "invalid email format", "INVALID_EMAIL")
	case domain.ErrInvalidUsername:
		respondWithError(w, http.StatusBadRequest, "invalid username format", "INVALID_USERNAME")
	case domain.ErrInvalidPassword:
		respondWithError(w, http.StatusBadRequest, "invalid password", "INVALID_PASSWORD")
	case domain.ErrWeakPassword:
		respondWithError(w, http.StatusBadRequest, "password is too weak", "WEAK_PASSWORD")
	case domain.ErrRateLimitExceeded:
		respondWithError(w, http.StatusTooManyRequests, "rate limit exceeded", "RATE_LIMIT")
	case domain.ErrTokenExpired:
		respondWithError(w, http.StatusUnauthorized, "token expired", "TOKEN_EXPIRED")
	case domain.ErrTokenRevoked:
		respondWithError(w, http.StatusUnauthorized, "token revoked", "TOKEN_REVOKED")
	case domain.ErrInvalidResetToken:
		respondWithError(w, http.StatusBadRequest, "invalid or expired reset token", "INVALID_RESET_TOKEN")
	case domain.Err2FARequired:
		respondWithError(w, http.StatusUnauthorized, "2FA code required", "2FA_REQUIRED")
	case domain.ErrVerificationTokenNotFound:
		respondWithError(w, http.StatusNotFound, "verification token not found", "TOKEN_NOT_FOUND")
	case domain.ErrVerificationTokenExpired:
		respondWithError(w, http.StatusBadRequest, "verification token expired", "TOKEN_EXPIRED")
	case domain.ErrVerificationTokenUsed:
		respondWithError(w, http.StatusBadRequest, "verification token already used", "TOKEN_USED")
	case domain.ErrEmailAlreadyVerified:
		respondWithError(w, http.StatusConflict, "email already verified", "EMAIL_VERIFIED")
	default:
		// Check if it's a validation error with a descriptive message
		errMsg := err.Error()
		if strings.Contains(errMsg, "password") || strings.Contains(errMsg, "username") || strings.Contains(errMsg, "email") {
			// It's a validation error, return the descriptive message
			respondWithError(w, http.StatusBadRequest, errMsg, "VALIDATION_ERROR")
		} else {
			h.logger.Error().Err(err).Msg("unhandled error")
			respondWithError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
		}
	}
}

func extractDevice(userAgent string) string {
	// Simple device extraction - can be enhanced
	if len(userAgent) > 50 {
		return userAgent[:50] + "..."
	}
	return userAgent
}
