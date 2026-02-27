# Auth Service - Rutas del Cliente (Usuarios, Perfiles y Tenant Admin)

Este documento contiene los comandos cURL correspondientes a la perspectiva del **Cliente**. Esto abarca todas las rutas de uso general para los usuarios finales (autenticación, gestión de perfil, sesiones, 2FA y OAuth) y además las rutas de administración de inquilinos (Tenant Admin), las cuales operan exclusivamente dentro de la organización del administrador.

## Configuración Inicial

Se asume que la URL base de tu API es `http://localhost:8080/api/v1`.
Para rutas protegidas, reemplaza el valor de la variable `$TOKEN` en los comandos por un JWT válido. El ID del Tenant (inquilino) se extrae del token.

```bash
export TOKEN="tu_token_jwt_aqui"
export BASE_URL="http://localhost:8080/api/v1"
```

---

## 1. Autenticación (Rutas Públicas)

### Registro de Usuario

Crea una nueva cuenta de usuario y envía un correo electrónico de verificación.

```bash
curl -X POST "$BASE_URL/auth/register" \
     -H "Content-Type: application/json" \
     -d '{
           "tenant_id": "customer1",
           "username": "johndoe",
           "email": "user@example.com",
           "password": "SecurePass123!"
         }'
```

### Verificar Correo (PIN de 6 dígitos)

Finaliza el registro validando el PIN enviado al correo electrónico. Sin este paso, el inicio de sesión será bloqueado.

```bash
curl -X POST "$BASE_URL/auth/verify-email" \
     -H "Content-Type: application/json" \
     -d '{
           "tenant_id": "customer1",
           "code": "123456"
         }'
```

### Reenviar Código de Verificación

Si el usuario no recibió el correo o el código expiró. **Plaqueado a 4 intentos por hora.**

```bash
curl -X POST "$BASE_URL/auth/resend-verification" \
     -H "Content-Type: application/json" \
     -d '{
           "tenant_id": "customer1",
           "email": "user@example.com"
         }'
```

### Iniciar Sesión (Login)

Autentica al usuario. Retorna tokens JWT (Access y Refresh).
Si 2FA está habilitado, requiere también `"two_fa_code"`.

```bash
curl -X POST "$BASE_URL/auth/login" \
     -H "Content-Type: application/json" \
     -d '{
           "identifier": "user@example.com",
           "password": "SecurePass123!"
         }'
```

### Renovar Token (Refresh Token)

Genera un nuevo token de acceso usando tu Refresh Token válido.

```bash
curl -X POST "$BASE_URL/auth/refresh" \
     -H "Content-Type: application/json" \
     -d '{
           "refresh_token": "tu_refresh_token_aqui"
         }'
```

---

## 2. Perfil de Usuario y Sesiones (Rutas Protegidas)

### Obtener Perfil (Me)

Retorna la información del usuario autenticado actualmente.

```bash
curl -X GET "$BASE_URL/auth/me" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

### Actualizar Perfil

Permite actualizar el nombre de usuario y/o correo electrónico.

```bash
curl -X PUT "$BASE_URL/auth/me" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
           "username": "new_johndoe"
         }'
```

### Listar Sesiones Activas

```bash
curl -X GET "$BASE_URL/auth/sessions" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

### Revocar Todas las Sesiones (Excepto la actual)

Útil para "cerrar sesión en todos los demás dispositivos".

```bash
curl -X DELETE "$BASE_URL/auth/sessions/all" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

### Cerrar Sesión (Logout)

Revoca el token de acceso actual (lo agrega a la lista negra).

```bash
curl -X POST "$BASE_URL/auth/logout" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

---

## 3. Autenticación de Dos Factores (2FA)

### Habilitar 2FA

Genera un secreto TOTP y un código QR en base64. El usuario debe escanear el QR y luego confirmarlo.

```bash
curl -X POST "$BASE_URL/auth/2fa/enable" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

### Verificar y Confirmar 2FA (Después de habilitarlo)

```bash
curl -X POST "$BASE_URL/auth/2fa/verify" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
           "code": "123456"
         }'
```

### Deshabilitar 2FA

Deshabilita el 2FA. Requiere confirmación con un código TOTP por seguridad.

```bash
curl -X POST "$BASE_URL/auth/2fa/disable" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
           "code": "123456"
         }'
```

### Generar Códigos de Respaldo (Backup Codes)

Genera 10 códigos de un solo uso en caso de perder acceso a la app autenticadora.

```bash
curl -X POST "$BASE_URL/auth/2fa/backup-codes" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

---

## 4. Recuperación de Contraseña y Verificación de Correo

### Solicitar Restablecimiento de Contraseña (Forgot Password)

```bash
curl -X POST "$BASE_URL/auth/forgot-password" \
     -H "Content-Type: application/json" \
     -d '{
           "tenant_id": "customer1",
           "email": "user@example.com"
         }'
```

### Restablecer Contraseña (mediante Token URL)

```bash
curl -X POST "$BASE_URL/auth/reset-password" \
     -H "Content-Type: application/json" \
     -d '{
           "tenant_id": "customer1",
           "token": "abc123token...",
           "new_password": "NewSecurePass123!"
         }'
```

### Restablecer Contraseña (mediante Código PIN 6-dígitos)

```bash
curl -X POST "$BASE_URL/auth/reset-password-code" \
     -H "Content-Type: application/json" \
     -d '{
           "tenant_id": "customer1",
           "email": "user@example.com",
           "code": "123456",
           "new_password": "NewSecurePass123!"
         }'
```

### Reenviar Correo de Verificación

Requiere estar logueado (sesión iniciada) pero con estado de correo NO verificado.

```bash
curl -X POST "$BASE_URL/auth/resend-verification" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

---

## 5. WebAuthn y Biometría (Passkeys / FIDO2)

_Nota: Estos flujos típicamente dependen de la API del navegador `navigator.credentials`, pero aquí tienes la referencia de los endpoints REST subyacentes._

### Iniciar Registro de Dispositivo

```bash
curl -X POST "$BASE_URL/auth/webauthn/register/begin" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json"
```

### Iniciar Login con Dispositivo (Público)

```bash
curl -X POST "$BASE_URL/auth/webauthn/login/begin" \
     -H "Content-Type: application/json"
```

---

## 6. OAuth 2.0 y OIDC (Redirecciones Sociales)

Estos endpoints están diseñados para ser abiertos directamente desde el navegador, ya que retornan redirecciones HTTP `307` hacia los proveedores de identidad correspondientes.

- **Iniciar sesión con Google:** `GET /api/v1/auth/oauth/google`
- **Iniciar sesión con GitHub:** `GET /api/v1/auth/oauth/github`
- **Información de Usuario OIDC:**
  ```bash
  curl -X GET "$BASE_URL/auth/userinfo" \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json"
  ```
- **Discovery Endpoint (Público):** `GET /api/v1/.well-known/openid-configuration`
- **Public JSON Web Keys (JWKS):** `GET /api/v1/auth/.well-known/jwks.json`

  > _Nota B2B:_ Nuestro endpoint JWKS devuelve un `kid` (Key ID) único asociado a la llave pública actual. Tus JWT también incluyen este `kid` en los Headers. Utilízalo en tu backend para implementar validación robusta y **Zero-Downtime Key Rotation**. Si rotamos nuestras llaves, tu sistema automáticamente descargará y cacheará la nueva llave basándose en el nuevo `kid`, sin cerrar la sesión de usuarios legítimos.

  > _Telemetría:_ Cada servidor (Tenant ID) que consulta este endpoint genera un incremento en la métrica expuesta de Prometheus `auth_service_tokens_jwks_hits_total`. Para mayor escalabilidad interna, confía más en la validación por firma matemática offline y no abuses de la red interna consultando en masa la ruta HTTP JWKS.

---

## 7. Consumo de Eventos B2B (Webhooks)

El Auth-Service entrega notificaciones asíncronas en tiempo real a tu servidor externo mediante **HTTP POST Webhooks** firmados con **HMAC-SHA256**.

> **Como funciona internamente:** Cuando ocurre un evento de seguridad (login, revocación, etc.), el Auth-Service lo encola en Redis vía `asynq`. Un pool de workers de alto rendimiento procesa las colas y hace el HTTP POST a tu endpoint. El sistema reintenta automáticamente con **Exponential Backoff** si tu servidor no está disponible (hasta 10 reintentos).

### Paso 1: Registra tu Webhook Endpoint

Crea una suscripción de Webhook desde el Panel Administrativo o la API:

```json
POST /api/v1/webhooks
{
  "url": "https://tu-servidor.com/webhooks/auth",
  "events": ["auth_session_revoked", "auth_account_locked", "auth_login_failed"],
  "secret": "tu-webhook-secret-aleatorio"
}
```

### Paso 2: Verifica la Firma HMAC en tu Servidor

Cada POST incluye el header `X-Auth-Signature: sha256=<hex_hash>`. **Debes validarlo** antes de procesar el evento:

```go
// Go: Verificación en tu servidor receptor
func VerifyWebhookSignature(body []byte, secret, signatureHeader string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signatureHeader))
}
```

```javascript
// Node.js: Verificación equivalente
const crypto = require("crypto");
function verifySignature(body, secret, signatureHeader) {
  const expected =
    "sha256=" + crypto.createHmac("sha256", secret).update(body).digest("hex");
  return crypto.timingSafeEqual(
    Buffer.from(expected),
    Buffer.from(signatureHeader),
  );
}
```

### Payload de Evento

```json
{
  "id": "evt_abc123",
  "type": "auth_session_revoked",
  "tenant_id": "t-your-tenant-id",
  "user_id": "usr_xyz",
  "timestamp": "2026-02-26T14:00:00Z",
  "data": { "reason": "forced_logout", "ip": "192.168.1.1" }
}
```

---

## 8. Integración gRPC (Backend to Backend)

Para validación de alta velocidad de los tokens JWT desde servidores de aplicaciones (Sistemas B2B), consulta la **[Guía de Integración gRPC](./grpc-integration.md)**. No utilices HTTP/REST para comunicación inter-microservicios si tienes acceso a la red interna.
