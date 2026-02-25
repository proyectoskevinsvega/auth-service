# Security Configuration

Security measures implemented in the auth-service.

## Implemented Features

### Security Headers Middleware

All HTTP responses include security headers via `SecurityHeaders` middleware:

**Content Security Policy**
```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'
```

**Clickjacking Protection**
```
X-Frame-Options: DENY
```

**MIME Type Sniffing Prevention**
```
X-Content-Type-Options: nosniff
```

**Referrer Policy**
```
Referrer-Policy: strict-origin-when-cross-origin
```

**Permissions Policy**
```
Permissions-Policy: geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=()
```

**HSTS (Production Only)**
```
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```
Only enabled when `ENVIRONMENT=production`.

### Rate Limiting

**Login**: 5 attempts per 60s per IP, blocks for 1 hour after limit
**Registration**: 3 attempts per hour per IP
**Token Refresh**: 10 attempts per 60s per IP

Configuration:
```env
RATE_LIMIT_LOGIN_ATTEMPTS=5
RATE_LIMIT_LOGIN_WINDOW=60
RATE_LIMIT_LOGIN_BLOCK_DURATION=3600
RATE_LIMIT_REGISTER_ATTEMPTS=3
RATE_LIMIT_REGISTER_WINDOW=3600
RATE_LIMIT_REFRESH_ATTEMPTS=10
RATE_LIMIT_REFRESH_WINDOW=60
```

### Input Validation

**Password**:
- Minimum 8 characters
- One uppercase, one lowercase, one number, one special char
- No leading/trailing spaces
- Special chars: `!@#$%^&*()_+-=[]{}|;:,.<>?/`

**Username**:
- 3-30 characters
- Letters, numbers, hyphens, underscores only
- Regex: `^[a-zA-Z0-9_-]{3,30}$`

**Email**:
- Standard email format validation
- Regex: `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`

### Password Security

- Bcrypt hashing with cost factor 12
- Constant-time comparison
- No plaintext storage

### JWT Security

- RSA-256 signing
- Short expiration (configurable, default 1 hour)
- Refresh token rotation
- Token revocation with blacklist
- JTI tracking prevents replay

### Session Management

- Active session tracking per user
- Device fingerprinting (device + IP)
- Geolocation logging (optional)
- Inactivity timeout (configurable, default 30 days)
- Users can view and revoke sessions

### HTTPS Enforcement

Production validation (`ENVIRONMENT=production`):
- `BASE_DOMAIN` cannot be localhost
- `ALLOWED_ORIGINS` must use HTTPS (except localhost)
- All generated URLs use `https://`

### Email Verification

- 32-byte random tokens (256 bits)
- SHA-256 hashing before storage
- 24-hour expiration
- One-time use
- IP tracking

### Password Reset

- Dual verification: token + 6-digit code
- 15-minute expiration
- One-time use
- Rate limited
- Email enumeration prevention

### 2FA/TOTP

- TOTP compatible with Google Authenticator
- QR code generation
- Required when enabled
- Secure secret storage

### Audit Logging

All auth events logged: login, register, password reset, token refresh, session revocation, 2FA changes.

Fields: user ID, action, IP, user agent, timestamp, success/failure, error details.

## Production Deployment

### Pre-Deployment Checklist

**Environment**:
- `ENVIRONMENT=production`
- `BASE_DOMAIN` is production domain (not localhost)
- `ALLOWED_ORIGINS` HTTPS only

**Database**:
- `POSTGRES_SSLMODE=require` or `verify-full`
- Database user has minimal permissions
- Backups configured and tested

**Redis**:
- `REDIS_PASSWORD` set
- Redis protected from public access

**JWT**:
- 4096-bit RSA keys generated
- Private key secured

**Email**:
- `EMAIL_FROM` verified domain
- `RESEND_API_KEY` production key

**Security**:
- Rate limits appropriate for traffic
- `GRPC_ALLOWED_IPS` restricted to internal network
- TLS 1.2+ at load balancer/reverse proxy
- Security headers validated (securityheaders.com)

### Testing Security

**Security Headers**:
```bash
curl -I http://localhost:8080/health
```

**Rate Limiting**:
```bash
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"identifier":"test@example.com","password":"wrong"}'
done
```
Expected: 429 after 5 attempts

**External Scanners**:
- OWASP ZAP
- SecurityHeaders.com
- SSL Labs
- Mozilla Observatory

## Future Improvements

### Refresh Tokens in HTTP-Only Cookies

Current: Refresh tokens in JSON response
Recommended: HTTP-Only cookies with `SameSite=Strict`

```go
http.SetCookie(w, &http.Cookie{
    Name:     "refresh_token",
    Value:    refreshToken,
    Path:     "/api/v1/auth/refresh",
    HttpOnly: true,
    Secure:   environment == "production",
    SameSite: http.SameSiteStrictMode,
    MaxAge:   int(refreshExpiry.Seconds()),
})
```

### CSRF Protection

If using cookies:
- SameSite cookies (mitigates most CSRF)
- Double-submit cookie pattern
- Synchronizer token pattern

### Account Lockout

Current: IP-based rate limiting
Recommended: Account-level lockout after N failed attempts

### Suspicious Activity Detection

- Login from new device/location
- Multiple failed attempts
- Password reset from unknown IP
- Session hijacking

### Secrets Management

Current: Environment variables
Recommended: HashiCorp Vault, AWS Secrets Manager, Azure Key Vault

### Enhanced Security Headers

```
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Embedder-Policy: require-corp
Cross-Origin-Resource-Policy: same-origin
```

### Logging Enhancements

- PII redaction
- Log aggregation (ELK, Splunk)
- Tamper-proof logging
- Automated alerts

## References

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- OWASP Cheat Sheets: https://cheatsheetseries.owasp.org/
- Mozilla Web Security: https://infosec.mozilla.org/guidelines/web_security

## Reporting Vulnerabilities

Email: security@your-domain.com

Do NOT create public GitHub issues for security vulnerabilities.
