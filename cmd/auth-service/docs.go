package main

// @title           Vertercloud Auth Service API
// @version         1.0
// @description     Microservicio central de autenticación para la plataforma Vertercloud.
// @description     Gestiona JWT, sesiones, 2FA, OAuth, verificación de email y validación de tokens.

// @contact.name   Vertercloud Support
// @contact.email  support@vertercloud.com

// @license.name  Proprietary
// @license.url   https://vertercloud.com/license

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name Authentication
// @tag.description Endpoints para registro, login, y gestión de tokens

// @tag.name User
// @tag.description Endpoints para gestión de perfil de usuario

// @tag.name Sessions
// @tag.description Endpoints para gestión de sesiones activas

// @tag.name 2FA
// @tag.description Endpoints para autenticación de dos factores (TOTP)

// @tag.name Email Verification
// @tag.description Endpoints para verificación de correo electrónico

// @tag.name OAuth
// @tag.description Endpoints para autenticación con Google y GitHub

// @tag.name Health
// @tag.description Endpoints de salud y monitoreo del servicio

// @schemes http https
// @produce json
// @accept json
