# 🗺 Auth Service — Roadmap

Hoja de ruta para la evolución del microservicio de autenticación.
Prioridades basadas en impacto de seguridad, demanda de funcionalidad y complejidad de implementación.

---

## 🔴 P0 — Crítico (Seguridad / Deuda técnica)

| Feature                           | Estado        | Descripción                                                                                                                                                                                        |
| --------------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Adaptive Authentication**       | ✅ Completado | Análisis de riesgo en tiempo real basado en: "Impossible travel" (cambio de ubicación geográfico imposible en tiempo real), IP de alto riesgo (TOR, VPNs públicas), y biometría de comportamiento. |
| **Account lockout configurable**  | ✅ Completado | Políticas de bloqueo de cuenta por intentos fallidos. Configurable por tenant: número de intentos, duración del bloqueo, escalamiento progresivo.                                                  |
| **Password expiration policies**  | ✅ Completado | Forzar cambio de contraseña después de N días. Notificación previa por email. Configurable por entorno.                                                                                            |
| **Forced password reset (admin)** | ✅ Completado | Capacidad de forzar reset de contraseña a un usuario desde el admin API. Invalida todas las sesiones activas.                                                                                      |

---

## 🟠 P1 — Alta prioridad (Features core)

| Feature                                | Estado        | Descripción                                                                                                      |
| -------------------------------------- | ------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Admin Web Console & Management API** | ❌ Pendiente  | Interfaz gráfica y API de administración para gestionar tenants, llaves RSA, y políticas de seguridad global.    |
| **OIDC Discovery & UserInfo**          | ✅ Completado | Implementar endpoints `/.well-known/openid-configuration` y `/userinfo` para cumplimiento total con OIDC.        |
| **Dynamic Scopes & RBAC/ABAC**         | ✅ Completado | Soporte para scopes dinámicos (OAuth2) y Modelos de Control de Acceso basado en Roles (RBAC) y Atributos (ABAC). |
| **Passwordless (FIDO2/WebAuthn)**      | ❌ Pendiente  | Autenticación biométrica (FaceID/TouchID) y hardware keys como factor primario o secundario.                     |

---

## 🟡 P2 — Media prioridad (Mejoras)

| Feature                         | Estado       | Descripción                                                                                                         |
| ------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------- |
| **User Self-Service Portal**    | ❌ Pendiente | Portal para usuarios: gestión de MFA, sesiones activas, descarga de datos (GDPR) e historial de seguridad personal. |
| **Enterprise Federation (SSO)** | ❌ Pendiente | Integración vía SAML 2.0 y OIDC con Azure AD, Okta, Ping Identity y Google Workspace.                               |
| **Multi-tenant isolation**      | ❌ Pendiente | Aislamiento lógico (User Pools) con configuración independiente por cliente.                                        |
| **Client Credentials Flow**     | ❌ Pendiente | Autenticación Machine-to-Machine para microservicios mediante Client ID / Client Secret.                            |
| **Webhook Lifecycle Events**    | ❌ Pendiente | Notificaciones en tiempo real hacia servicios externos sobre eventos críticos (ej: UserDeletion, RiskDetected).     |
| **Developer SDKs & CLI**        | ❌ Pendiente | SDKs oficiales (Go, TS, Python) y herramienta de comandos para automatización de la plataforma.                     |

---

## 🟢 P3 — Baja prioridad (Nice-to-have)

| Feature                          | Estado       | Descripción                                                                                          |
| -------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------- |
| **Session geofencing**           | ❌ Pendiente | Restricción de acceso basada en perímetros geográficos y prevención de saltos de IP sospechosos.     |
| **Compliance & Audit Reporting** | ❌ Pendiente | Generación automática de reportes de cumplimiento para normativas GDPR, SOC2 e HIPAA.                |
| **Advanced Threat Intelligence** | ❌ Pendiente | Integración con bases de datos de amenazas externas para bloquear IPs maliciosas de forma proactiva. |
| **Backup codes regeneration**    | ❌ Pendiente | Workflow de recuperación de emergencia ante pérdida de dispositivo MFA.                              |
| **Login analytics dashboard**    | ❌ Pendiente | Analíticas avanzadas de uso, retención y patrones de autenticación por aplicación.                   |

---

## 🔧 Infraestructura & Operaciones

| Feature                               | Estado       | Descripción                                                                                                                                                  |
| ------------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **HA / Failover documentation**       | ❌ Pendiente | Guía de deploy multi-instancia con load balancer. Configuración de PostgreSQL replication y Redis Sentinel/Cluster para alta disponibilidad.                 |
| **Backup & recovery procedures**      | ❌ Pendiente | Scripts de backup automatizado para PostgreSQL y Redis. Runbook de recovery ante desastres. RPO/RTO documentados.                                            |
| **Alert rules & runbooks**            | ❌ Pendiente | Reglas de alerta en Prometheus/Alertmanager para: error rate >5%, latencia p99 >500ms, login failures spike, rate limit violations. Runbook por cada alerta. |
| **Kubernetes Helm chart**             | ❌ Pendiente | Chart de Helm con values configurables para deploy en Kubernetes. Incluye: HPA, PDB, NetworkPolicy, ServiceMonitor.                                          |
| **Database query optimization guide** | ❌ Pendiente | Documentar explain plans de queries frecuentes. Cache warming strategies. Connection pool tuning para diferentes cargas.                                     |

---

## ✅ Completado

| Feature                                          | Versión |
| ------------------------------------------------ | ------- |
| Account lockout configurable                     | v1.1    |
| Adaptive Authentication (Impossible Travel)      | v1.1    |
| Password expiration policies                     | v1.1    |
| Forced password reset (admin)                    | v1.1    |
| Dynamic Scopes & RBAC/ABAC                       | v1.2    |
| OIDC Discovery & UserInfo                        | v1.2    |
| JWT RSA-256 + Refresh Token Rotation             | v1.0    |
| OAuth 2.0 (Google, GitHub)                       | v1.0    |
| 2FA/TOTP (Google Authenticator)                  | v1.0    |
| Email verification (Resend API)                  | v1.0    |
| Password reset (token + 6-digit code)            | v1.0    |
| Session management multi-device                  | v1.0    |
| Rate limiting distribuido (Redis)                | v1.0    |
| Argon2id password hashing                        | v1.0    |
| Audit logging completo                           | v1.0    |
| gRPC inter-service API                           | v1.0    |
| Swagger/OpenAPI documentation                    | v1.0    |
| 50+ Prometheus metrics                           | v1.0    |
| 3 Grafana dashboards (34 paneles)                | v1.0    |
| OpenTelemetry tracing (HTTP, gRPC, DB, Redis)    | v1.0    |
| CI/CD (GitHub Actions: CI, Security, Release)    | v1.0    |
| Docker multi-stage (scratch, ~15MB)              | v1.0    |
| k6 load testing (4 escenarios)                   | v1.0    |
| Security headers (CSP, HSTS, Permissions-Policy) | v1.0    |
| Production security validation                   | v1.0    |
| Systemd deployment                               | v1.0    |

---

## Criterios de Priorización

| Criterio                          | Peso  |
| --------------------------------- | ----- |
| **Impacto en seguridad**          | Alto  |
| **Demanda de usuarios/servicios** | Alto  |
| **Reducción de deuda técnica**    | Medio |
| **Complejidad de implementación** | Medio |
| **Valor para portafolio**         | Bajo  |

---

<p align="center">
  <em>Última actualización: Febrero 2026</em>
</p>
