package http

// Este archivo contiene todas las anotaciones de Swagger/OpenAPI para los handlers HTTP
// Las anotaciones están separadas de handler.go para mantener el código más limpio

// RefreshToken godoc
// @Summary      Renovar token de acceso
// @Description  Renueva el token de acceso JWT usando un refresh token válido. Implementa rotación de tokens: devuelve un nuevo access token y un nuevo refresh token.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest true "Refresh token"
// @Success      200  {object}  RefreshTokenResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Refresh token inválido o expirado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/refresh [post]

// Logout godoc
// @Summary      Cerrar sesión
// @Description  Cierra la sesión actual del usuario, revoca los tokens y la sesión activa.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/logout [post]

// ForgotPassword godoc
// @Summary      Solicitar recuperación de contraseña
// @Description  Envía un email con un código de 6 dígitos y un link para restablecer la contraseña. El código/link expira en 15 minutos.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body ForgotPasswordRequest true "Email de la cuenta"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse "Usuario no encontrado"
// @Failure      429  {object}  ErrorResponse "Demasiadas solicitudes"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/forgot-password [post]

// ResetPassword godoc
// @Summary      Restablecer contraseña con token
// @Description  Restablece la contraseña usando el token recibido por email (opción 1: link directo).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordRequest true "Token y nueva contraseña"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/reset-password [post]

// ResetPasswordWithCode godoc
// @Summary      Restablecer contraseña con código
// @Description  Restablece la contraseña usando el código de 6 dígitos recibido por email (opción 2: más fácil en móvil).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordWithCodeRequest true "Email, código y nueva contraseña"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Código inválido o expirado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/reset-password-code [post]

// VerifyEmail godoc
// @Summary      Verificar correo electrónico
// @Description  Verifica el correo electrónico del usuario usando el token recibido por email. El token es de un solo uso y expira en 24 horas.
// @Tags         Email Verification
// @Accept       json
// @Produce      json
// @Param        request body VerifyEmailRequest true "Token de verificación"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      409  {object}  ErrorResponse "Email ya verificado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/verify-email [post]

// ResendVerificationEmail godoc
// @Summary      Reenviar email de verificación
// @Description  Reenvía el email de verificación al usuario autenticado. Elimina tokens anteriores y genera uno nuevo.
// @Tags         Email Verification
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o no autenticado"
// @Failure      409  {object}  ErrorResponse "Email ya verificado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/resend-verification [post]

// GetMe godoc
// @Summary      Obtener perfil del usuario actual
// @Description  Retorna la información del usuario autenticado actualmente.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/me [get]

// UpdateMe godoc
// @Summary      Actualizar perfil del usuario
// @Description  Actualiza el email y/o username del usuario autenticado. Los campos son opcionales.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateUserRequest true "Datos a actualizar"
// @Success      200  {object}  UserResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      409  {object}  ErrorResponse "Email o username ya existe"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/me [put]

// ListSessions godoc
// @Summary      Listar sesiones activas
// @Description  Retorna todas las sesiones activas del usuario autenticado, incluyendo IP, país, dispositivo y fecha de último uso.
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  SessionsResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/sessions [get]

// RevokeSession godoc
// @Summary      Revocar sesión específica
// @Description  Revoca una sesión específica del usuario autenticado. Invalida todos los tokens asociados a esa sesión.
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Session ID"
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      404  {object}  ErrorResponse "Sesión no encontrada"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/sessions/{id} [delete]

// RevokeAllSessions godoc
// @Summary      Revocar todas las sesiones
// @Description  Revoca todas las sesiones del usuario autenticado excepto la actual. Útil si el usuario sospecha que su cuenta está comprometida.
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/sessions/all [delete]

// Enable2FA godoc
// @Summary      Habilitar autenticación de dos factores
// @Description  Genera un nuevo secret TOTP y retorna un código QR para configurar en Google Authenticator u otra app compatible. No activa 2FA hasta que se verifique el código.
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  Enable2FAResponse
// @Failure      401  {object}  ErrorResponse "Token inválido o expirado"
// @Failure      409  {object}  ErrorResponse "2FA ya está habilitado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/2fa/enable [post]

// Verify2FA godoc
// @Summary      Verificar y activar 2FA
// @Description  Verifica el código TOTP y activa 2FA en la cuenta. A partir de este momento, todos los logins requerirán el código 2FA.
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body Verify2FARequest true "Código TOTP"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Token inválido, expirado, o código 2FA incorrecto"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/2fa/verify [post]

// Disable2FA godoc
// @Summary      Deshabilitar autenticación de dos factores
// @Description  Desactiva 2FA en la cuenta. Requiere un código TOTP válido para confirmar.
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body Disable2FARequest true "Código TOTP"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "Token inválido, expirado, o código 2FA incorrecto"
// @Failure      404  {object}  ErrorResponse "2FA no está habilitado"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/2fa/disable [post]

// GetJWKS godoc
// @Summary      Obtener claves públicas JWKS
// @Description  Retorna las claves públicas en formato JWKS (JSON Web Key Set) para validar JWTs firmados por este servicio. Usado por otros microservicios.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  JWKSResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/.well-known/jwks.json [get]

// GoogleOAuthStart godoc
// @Summary      Iniciar login con Google
// @Description  Redirige al usuario a la página de autorización de Google OAuth. Después de autorizar, Google redirige al callback.
// @Tags         OAuth
// @Produce      html
// @Success      302  {string}  string "Redirige a Google OAuth"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/oauth/google [get]

// GoogleOAuthCallback godoc
// @Summary      Callback de Google OAuth
// @Description  Procesa el callback de Google OAuth, intercambia el código por tokens, y crea o actualiza el usuario. Retorna tokens JWT.
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Param        code   query     string  true  "Authorization code"
// @Param        state  query     string  true  "State parameter"
// @Success      200  {object}  LoginResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "OAuth error"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/oauth/google/callback [get]

// GitHubOAuthStart godoc
// @Summary      Iniciar login con GitHub
// @Description  Redirige al usuario a la página de autorización de GitHub OAuth. Después de autorizar, GitHub redirige al callback.
// @Tags         OAuth
// @Produce      html
// @Success      302  {string}  string "Redirige a GitHub OAuth"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/oauth/github [get]

// GitHubOAuthCallback godoc
// @Summary      Callback de GitHub OAuth
// @Description  Procesa el callback de GitHub OAuth, intercambia el código por tokens, obtiene info del usuario (incluido email privado), y crea o actualiza el usuario. Retorna tokens JWT.
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Param        code   query     string  true  "Authorization code"
// @Param        state  query     string  true  "State parameter"
// @Success      200  {object}  LoginResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse "OAuth error"
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/oauth/github/callback [get]

// Health godoc
// @Summary      Health check del servicio
// @Description  Retorna el estado de salud del auth service. Usado por Kubernetes liveness/readiness probes.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /health [get]
