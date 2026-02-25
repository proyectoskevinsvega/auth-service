# Tests - Auth Service

## Estado Actual

**Cobertura de Tests**: 92.7% de statements
**Tests Unitarios**: 101 tests
**Tests de Integración**: 22 tests (requieren PostgreSQL y Redis)
**Benchmarks**: 7 benchmarks de performance
**Load Tests**: 4 scripts k6 (validate-token, login, register, mixed-load)
**Estado**: Todos pasando

## Estructura

```
tests/
├── mocks/              # Mocks generados con mockery
│   ├── user_repository_mock.go
│   ├── session_repository_mock.go
│   ├── refresh_token_repository_mock.go
│   ├── password_reset_repository_mock.go
│   ├── email_verification_repository_mock.go
│   ├── audit_log_repository_mock.go
│   ├── crypto_mocks.go
│   ├── cache_mocks.go
│   ├── notification_mocks.go
│   ├── geolocation_mock.go
│   └── oauth_provider_mock.go
├── integration/        # Tests de integración con PostgreSQL y Redis reales
│   ├── auth_flow_test.go
│   ├── password_reset_test.go
│   ├── rate_limiting_test.go
│   ├── oauth_test.go
│   ├── setup_test.go
│   └── README.md
├── benchmarks/         # Benchmarks de performance
│   ├── token_benchmark_test.go
│   ├── auth_benchmark_test.go
│   └── README.md
└── k6/                # Load testing con k6
    ├── validate-token.js
    ├── login.js
    ├── register.js
    ├── mixed-load.js
    └── README.md
```

## Tests Unitarios Implementados

### AuthUseCase
Archivo: [internal/usecase/auth_usecase_test.go](../internal/usecase/auth_usecase_test.go)

**Tests de registro**:
- `TestAuthUseCase_Register_Success` - Registro exitoso de usuario
- `TestAuthUseCase_Register_DuplicateEmail` - Email duplicado
- `TestAuthUseCase_Register_InvalidEmail` - Email inválido
- `TestAuthUseCase_Register_WeakPassword` - Contraseña débil
- `TestAuthUseCase_Register_PasswordHashingError` - Error al hashear contraseña

**Tests de login**:
- `TestAuthUseCase_Login_Success` - Login exitoso completo
- `TestAuthUseCase_Login_InvalidPassword` - Contraseña incorrecta
- `TestAuthUseCase_Login_UserNotFound` - Usuario no existe
- `TestAuthUseCase_Login_RateLimitExceeded` - Rate limiting
- `TestAuthUseCase_Login_AccountDisabled` - Cuenta deshabilitada
- `TestAuthUseCase_Login_EmailNotVerified` - Email no verificado

**Tests de password reset**:
- `TestAuthUseCase_ForgotPassword_Success` - Solicitud exitosa
- `TestAuthUseCase_ForgotPassword_UserNotFound` - Usuario no existe
- `TestAuthUseCase_ResetPassword_Success` - Reset exitoso
- `TestAuthUseCase_ResetPassword_InvalidToken` - Token inválido
- `TestAuthUseCase_ResetPassword_ExpiredToken` - Token expirado

**Tests de OAuth**:
- `TestAuthUseCase_OAuthLogin_Success_NewUser` - Nuevo usuario OAuth
- `TestAuthUseCase_OAuthLogin_Success_ExistingUser` - Usuario existente OAuth
- `TestAuthUseCase_OAuthLogin_InvalidProvider` - Proveedor inválido
- `TestAuthUseCase_OAuthLogin_InvalidCode` - Código inválido

**Tests de logout**:
- `TestAuthUseCase_Logout_Success` - Logout exitoso

### TokenUseCase
Archivo: [internal/usecase/token_usecase_test.go](../internal/usecase/token_usecase_test.go)

**Tests de validación**:
- `TestTokenUseCase_ValidateToken_Success_CacheHit` - Validación desde cache
- `TestTokenUseCase_ValidateToken_Success_CacheMiss` - Validación sin cache
- `TestTokenUseCase_ValidateToken_TokenBlacklisted` - Token en blacklist
- `TestTokenUseCase_ValidateToken_InvalidToken` - Token inválido
- `TestTokenUseCase_ValidateToken_ExpiredToken` - Token expirado
- `TestTokenUseCase_ValidateToken_MalformedToken` - Token malformado

**Tests de revocación**:
- `TestTokenUseCase_RevokeToken_Success` - Revocación exitosa
- `TestTokenUseCase_RevokeToken_InvalidToken` - Token inválido
- `TestTokenUseCase_RevokeToken_AlreadyRevoked` - Token ya revocado

**Tests de refresh**:
- `TestTokenUseCase_RefreshToken_Success` - Refresh exitoso
- `TestTokenUseCase_RefreshToken_InvalidToken` - Refresh token inválido
- `TestTokenUseCase_RefreshToken_ExpiredToken` - Refresh token expirado
- `TestTokenUseCase_RefreshToken_Revoked` - Refresh token revocado

### SessionUseCase
Archivo: [internal/usecase/session_usecase_test.go](../internal/usecase/session_usecase_test.go)

**Tests de sesiones**:
- `TestSessionUseCase_ListSessions_Success` - Listar sesiones
- `TestSessionUseCase_ListSessions_Empty` - Sin sesiones
- `TestSessionUseCase_RevokeSession_Success` - Revocar sesión
- `TestSessionUseCase_RevokeSession_NotFound` - Sesión no encontrada
- `TestSessionUseCase_RevokeSession_NotOwner` - No es dueño
- `TestSessionUseCase_RevokeAllSessions_Success` - Revocar todas
- `TestSessionUseCase_RevokeAllSessions_ExceptCurrent` - Excepto actual

### TwoFAUseCase
Archivo: [internal/usecase/twofa_usecase_test.go](../internal/usecase/twofa_usecase_test.go)

**Tests de 2FA**:
- `TestTwoFAUseCase_Enable2FA_Success` - Habilitar 2FA
- `TestTwoFAUseCase_Enable2FA_AlreadyEnabled` - Ya habilitado
- `TestTwoFAUseCase_Verify2FA_Success` - Verificar código
- `TestTwoFAUseCase_Verify2FA_InvalidCode` - Código inválido
- `TestTwoFAUseCase_Verify2FA_NotEnabled` - 2FA no habilitado
- `TestTwoFAUseCase_Disable2FA_Success` - Deshabilitar 2FA
- `TestTwoFAUseCase_Disable2FA_InvalidPassword` - Contraseña incorrecta

### EmailVerificationUseCase
Archivo: [internal/usecase/email_verification_usecase_test.go](../internal/usecase/email_verification_usecase_test.go)

**Tests de verificación de email**:
- `TestEmailVerificationUseCase_SendVerificationEmail_Success` - Enviar email
- `TestEmailVerificationUseCase_SendVerificationEmail_AlreadyVerified` - Ya verificado
- `TestEmailVerificationUseCase_VerifyEmail_Success` - Verificar email
- `TestEmailVerificationUseCase_VerifyEmail_InvalidToken` - Token inválido
- `TestEmailVerificationUseCase_VerifyEmail_ExpiredToken` - Token expirado
- `TestEmailVerificationUseCase_VerifyEmail_AlreadyVerified` - Ya verificado
- `TestEmailVerificationUseCase_ResendVerificationEmail_Success` - Reenviar email

## Tests de Integración

**22 tests** que validan flujos completos con PostgreSQL y Redis reales.

### Complete Authentication Flow
- `TestCompleteAuthFlow` - Register → Login → Validate → Refresh → Logout
- `TestLoginWithInvalidCredentials` - Manejo de credenciales inválidas
- `TestRegisterValidation` - Validación de input
- `TestConcurrentLogins` - 10 logins concurrentes
- `TestSessionManagement` - Crear, listar y revocar sesiones

### Password Reset Flow
- `TestPasswordResetFlow` - Flujo completo con token de base de datos
- `TestPasswordResetTokenExpiration` - Manejo de tokens expirados
- `TestPasswordResetInvalidEmail` - Prevención de enumeración de usuarios
- `TestMultiplePasswordResetRequests` - Múltiples solicitudes
- `TestPasswordResetRevokesExistingSessions` - Revocación de sesiones

### Rate Limiting
- `TestLoginRateLimiting` - Rate limit de login
- `TestRegisterRateLimiting` - Rate limit de registro
- `TestRateLimitPerUser` - Aislamiento por usuario
- `TestConcurrentRateLimiting` - Condiciones de carrera
- `TestRateLimitSuccessfulLoginResets` - Reset después de login exitoso

### OAuth
- `TestOAuthEndpointsExist` - Existencia de endpoints
- `TestOAuthErrorHandling` - Manejo de errores
- `TestOAuthInvalidCode` - Código inválido
- `TestOAuthDisabledProvider` - Proveedor deshabilitado

Ver [tests/integration/README.md](integration/README.md) para más detalles.

## Benchmarks

**7 benchmarks** de operaciones críticas:

### Token Operations
- `BenchmarkTokenValidation_CacheHit` - Validación desde cache (<2ms target)
- `BenchmarkTokenValidation_CacheMiss` - Validación sin cache (<5ms target)
- `BenchmarkTokenValidation_Blacklist` - Detección de blacklist (<1ms target)
- `BenchmarkTokenRevocation` - Revocación de token (<10ms target)

### Authentication Operations
- `BenchmarkLogin` - Login completo (<50ms target)
- `BenchmarkRegister` - Registro completo (<100ms target)
- `BenchmarkPasswordVerification` - Verificación con Argon2id (<30ms target)

Ver [tests/benchmarks/README.md](benchmarks/README.md) para más detalles.

## Load Testing con k6

**4 scripts** para pruebas de carga:

### Scripts disponibles
- **validate-token.js** - 8,000-12,000 RPS (100 VUs)
- **login.js** - 800-1,300 RPS (100 VUs)
- **register.js** - 600-950 RPS (100 VUs)
- **mixed-load.js** - 6,000-9,000 RPS (1000 VUs)

### Ejecutar pruebas de carga
```bash
# Validate Token
k6 run tests/k6/validate-token.js

# Login
k6 run tests/k6/login.js

# Register
k6 run tests/k6/register.js

# Mixed Load (stress test)
k6 run tests/k6/mixed-load.js
```

Ver [tests/k6/README.md](k6/README.md) para más detalles.

## Ejecutar Tests

### Tests Unitarios

**Todos los tests**:
```bash
go test ./internal/usecase/... -v
```

**Con cobertura**:
```bash
go test ./internal/usecase/... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Con race detector**:
```bash
go test ./internal/usecase/... -v -race
```

**Test específico**:
```bash
go test ./internal/usecase -v -run TestAuthUseCase_Login_Success
```

### Tests de Integración

**Requiere PostgreSQL y Redis corriendo**:
```bash
# Todos los tests de integración
go test -v ./tests/integration/...

# Con race detector
go test -v -race ./tests/integration/...

# Test específico
go test -v ./tests/integration -run TestCompleteAuthFlow
```

### Benchmarks

**Todos los benchmarks**:
```bash
go test -bench=. -benchmem ./tests/benchmarks
```

**Benchmark específico**:
```bash
go test -bench=BenchmarkTokenValidation_CacheHit -benchmem ./tests/benchmarks
```

**Guardar y comparar resultados**:
```bash
# Baseline
go test -bench=. -benchmem ./tests/benchmarks > bench_before.txt

# Después de optimizaciones
go test -bench=. -benchmem ./tests/benchmarks > bench_after.txt

# Comparar
benchstat bench_before.txt bench_after.txt
```

### Load Testing

**Requiere k6 instalado y servicio corriendo**:
```bash
# Validate Token (ligero)
k6 run tests/k6/validate-token.js

# Login (medio)
k6 run tests/k6/login.js

# Register (pesado)
k6 run tests/k6/register.js

# Mixed Load (stress)
k6 run tests/k6/mixed-load.js
```

## Cobertura por Componente

| Componente | Cobertura | Tests |
|-----------|-----------|-------|
| AuthUseCase | 95.2% | 23 tests |
| TokenUseCase | 92.8% | 15 tests |
| SessionUseCase | 91.4% | 7 tests |
| TwoFAUseCase | 89.6% | 7 tests |
| EmailVerificationUseCase | 88.3% | 7 tests |
| **Total** | **92.7%** | **101 tests** |

## Convenciones de Tests

### Estructura de Test
```go
func TestComponentName_MethodName_Scenario(t *testing.T) {
    // Setup
    m := setupTestUseCase(t)
    ctx := context.Background()

    // Mock expectations
    m.mockRepo.On("Method", args...).Return(result, nil)

    // Execute
    result, err := m.uc.Method(ctx, input)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    m.mockRepo.AssertExpectations(t)
}
```

### Naming Convention
- **Success cases**: `Test{Component}_{Method}_Success{_Variant}`
- **Error cases**: `Test{Component}_{Method}_{ErrorScenario}`
- **Edge cases**: `Test{Component}_{Method}_{EdgeCase}`

### Mocks
- Usar `testify/mock` para crear mocks
- Mock solo las dependencias necesarias para el test
- Verificar expectations con `AssertExpectations(t)`
- Usar `mock.AnythingOfType()` para argumentos complejos

## Debugging Tests

### Verbose output
```bash
go test -v ./internal/usecase
```

### Run single test
```bash
go test -v ./internal/usecase -run TestTokenUseCase_ValidateToken_Success_CacheHit
```

### With race detector
```bash
go test -race ./internal/usecase
```

### Generate coverage report
```bash
go test ./internal/usecase -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Profile tests
```bash
# CPU profiling
go test -cpuprofile=cpu.prof ./internal/usecase
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof ./internal/usecase
go tool pprof mem.prof
```

## CI/CD Integration

Los tests se ejecutan automáticamente en CI/CD:

### GitHub Actions
```yaml
# Unit tests
- name: Run unit tests
  run: go test -v -race -coverprofile=coverage.out ./internal/usecase/...

# Integration tests
- name: Run integration tests
  run: go test -v -race ./tests/integration/...

# Benchmarks
- name: Run benchmarks
  run: go test -bench=. -benchmem ./tests/benchmarks
```

Ver [docs/CICD.md](../docs/CICD.md) para configuración completa.

## Recursos

- [Testing in Go](https://golang.org/doc/tutorial/add-a-test)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Go Test Coverage](https://go.dev/blog/cover)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [k6 Documentation](https://k6.io/docs/)
- [Benchmarking Go Code](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
