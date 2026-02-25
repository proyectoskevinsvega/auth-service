# 🛡️ Compliance & Audit Reporting

Este documento describe las capacidades de cumplimiento normativo (Compliance) del Vertercloud Auth Service, enfocándose en la generación automática de reportes para GDPR, SOC2 e HIPAA.

## Introducción

El servicio incluye un motor de auditoría avanzado que registra cada evento crítico del sistema. A partir de esta base de datos de auditoría, se generan reportes estructurados que facilitan la portabilidad de datos y la certificación ante auditores externos.

## Normativas Soportadas

### 1. GDPR (Reglamento General de Protección de Datos)

Enfocado en el **Derecho de Portabilidad** y el **Derecho de Acceso**.

- **Endpoint**: `GET /api/v1/admin/compliance/gdpr/{userID}`
- **Contenido del Reporte**:
  - Perfil completo del usuario (Username, Email, Status).
  - Listado de sesiones activas y dispositivos vinculados.
  - Historial reciente de auditoría (últimas 100 acciones).
  - Marca de tiempo de exportación.

### 2. SOC2 (System and Organization Controls)

Enfocado en los criterios de **Seguridad** y **Disponibilidad**.

- **Endpoint**: `GET /api/v1/admin/compliance/soc2?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD`
- **Contenido del Reporte**:
  - Resumen de intentos fallidos de inicio de sesión (Detección de Brute Force).
  - Log detallado de todas las acciones administrativas (Creación de roles, gestión de permisos).
  - Periodo de auditoría específico.

### 3. HIPAA (Health Insurance Portability and Accountability Act)

Enfocado en la **Integridad** y el **Control de Acceso** a datos sensibles.

- **Endpoint**: `GET /api/v1/admin/compliance/hipaa?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD`
- **Contenido del Reporte**:
  - Registro de accesos exitosos (Logins).
  - Eventos de seguridad (Violaciones de políticas, bloqueos de cuenta).
  - Trazabilidad de cambios de identidad.

## Arquitectura de Auditoría

Los reportes se alimentan del `AuditLogRepository`, el cual utiliza PostgreSQL para almacenamiento persistente. Cada entrada de log incluye:

- **Identity**: TenantID, UserID.
- **Context**: IP Address, User Agent, Country (GeoIP).
- **Action**: Acción ejecutada (ej: `auth_login_success`).
- **Result**: Éxito/Fracaso y mensaje de error si aplica.
- **Metadata**: Información adicional estructurada en formato JSON.

## Seguridad de los Reportes

- **Acceso Restringido**: Solo usuarios con el rol `admin` debidamente autenticados pueden generar estos reportes.
- **Aislamiento (Multi-tenancy)**: Cada reporte está estrictamente filtrado por el `tenant_id` del administrador que realiza la consulta, garantizando que los datos de un cliente no sean visibles para otro.

---

_Vertercloud Auth Service v1.7 — Generando confianza a través de la transparencia._
