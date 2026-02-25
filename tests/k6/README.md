# Pruebas de Carga con k6 - Auth Service

Suite completa de pruebas de rendimiento para validar la capacidad del servicio de autenticación bajo diferentes escenarios de carga.

## Tabla de Contenidos

- [¿Qué es k6?](#qué-es-k6)
- [Instalación](#instalación)
- [Inicio Rápido](#inicio-rápido)
- [Scripts de Prueba](#scripts-de-prueba)
- [Interpretación de Resultados](#interpretación-de-resultados)
- [Configuración Avanzada](#configuración-avanzada)
- [Benchmarks Esperados](#benchmarks-esperados)
- [Troubleshooting](#troubleshooting)

## ¿Qué es k6?

k6 es una herramienta moderna de pruebas de carga de código abierto desarrollada por Grafana Labs. A diferencia de otras herramientas como JMeter o Apache Bench, k6 está diseñado para:

- Performance testing (pruebas de rendimiento)
- Load testing (pruebas de carga)
- Stress testing (pruebas de estrés)
- Soak testing (pruebas de duración prolongada)

### ¿Por qué k6?

1. Scripts en JavaScript/ES6: Fácil de escribir y mantener
2. Lightweight: Consume pocos recursos (CPU/RAM)
3. Alto rendimiento: Miles de VUs (usuarios virtuales) en una sola máquina
4. CLI first: Perfecto para CI/CD
5. Métricas detalladas: Análisis profundo del rendimiento

## Instalación

### Windows

#### Opción 1: Chocolatey (Recomendado)
```bash
choco install k6
```

#### Opción 2: Scoop
```bash
scoop install k6
```

#### Opción 3: Descarga Manual
1. Ir a: https://github.com/grafana/k6/releases
2. Descargar: `k6-v0.48.0-windows-amd64.zip` (última versión)
3. Extraer a: `C:\k6\`
4. Agregar al PATH:
   ```powershell
   # PowerShell (como administrador)
   [Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\k6", "Machine")
   ```

#### Opción 4: Usando el Makefile
```bash
make install-k6
```

### macOS
```bash
brew install k6
```

### Linux (Ubuntu/Debian)
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69

echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list

sudo apt-get update
sudo apt-get install k6
```

### Verificar Instalación
```bash
k6 version
# Output: k6 v0.48.0 (go1.21.4, windows/amd64)
```

## Inicio Rápido

### Preparar el Servicio

Asegúrate de que el servicio esté corriendo:

```bash
# Iniciar el servicio
make run

# En otra terminal, verificar que esté activo
curl http://localhost:8080/health
# Output: {"status":"healthy"}
```

### Ejecutar Primera Prueba

La prueba más importante es validate-token (mide el cache hit):

```bash
# Desde la raíz del proyecto
make load-test-validate

# O directamente
cd tests/k6
k6 run validate-token.js
```

### Ver Resultados en Tiempo Real

Durante la ejecución verás:

```
          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: validate-token.js
     output: -

  scenarios: (100.00%) 1 scenario, 1000 max VUs, 5m0s max duration
           * default: Up to 1000 looping VUs for 4m30s over 5 stages

     ✓ status is 200
     ✓ response time < 50ms
     ✓ response time < 100ms
     ✓ has user data

     checks.........................: 100.00% ✓ 450234      ✗ 0
     data_received..................: 135 MB  507 kB/s
     data_sent......................: 55 MB   207 kB/s
     http_req_blocked...............: avg=45µs   min=0s     med=0s      max=12.5ms  p(90)=0s      p(95)=0s
     http_req_connecting............: avg=22µs   min=0s     med=0s      max=12.1ms  p(90)=0s      p(95)=0s
   ✓ http_req_duration..............: avg=12.3ms min=3.2ms  med=10.8ms  max=234ms   p(90)=18.5ms  p(95)=24.2ms
     http_req_failed................: 0.00%   ✓ 0           ✗ 112559
     http_req_receiving.............: avg=1.2ms  min=14µs   med=98µs    max=89ms    p(90)=2.5ms   p(95)=4.8ms
     http_req_sending...............: avg=45µs   min=0s     med=15µs    max=25ms    p(90)=98µs    p(95)=145µs
     http_req_tls_handshaking.......: avg=0s     min=0s     med=0s      max=0s      p(90)=0s      p(95)=0s
     http_req_waiting...............: avg=11.1ms min=2.8ms  med=9.5ms   max=198ms   p(90)=17.2ms  p(95)=22.1ms
     http_reqs......................: 112559  422.47/s
     iteration_duration.............: avg=112ms  min=103ms  med=110ms   max=342ms   p(90)=118ms   p(95)=124ms
     iterations.....................: 112559  422.47/s
     vus............................: 47      min=0         max=1000
     vus_max........................: 1000    min=1000      max=1000

running (4m26.5s), 0000/1000 VUs, 112559 complete and 0 interrupted iterations

========================================
VALIDATE TOKEN TEST RESULTS
========================================
Requests per second: 422.47
Avg response time: 12.30ms
P95 response time: 24.20ms
P99 response time: 38.50ms
Error rate: 0.00%
========================================
```

## Scripts de Prueba

### validate-token.js - Validación de Tokens (Cache Hit)

Objetivo: Medir el rendimiento de validación de tokens JWT con cache hit >95%

Carga:
```
30s → 100 VUs
1m  → 500 VUs
2m  → 1000 VUs (pico)
1m  → 1000 VUs (sostenido)
30s → 0 VUs (ramp down)
```

Umbrales de Éxito:
- `http_req_duration p(95) < 50ms`
- `http_req_duration p(99) < 100ms`
- `errors < 1%`
- `RPS: 8,000-12,000` (en VPS 4 vCPU + 8GB RAM)

Ejecutar:
```bash
make load-test-validate
# o
k6 run validate-token.js
```

Qué mide:
- Rendimiento del cache Redis
- Latencia de validación JWT
- Throughput del endpoint `/auth/me`
- Comportamiento bajo alta concurrencia

### login.js - Login de Usuarios

Objetivo: Medir el rendimiento del proceso de login con Argon2id

Carga:
```
30s → 50 VUs
1m  → 200 VUs
2m  → 500 VUs (pico)
1m  → 500 VUs (sostenido)
30s → 0 VUs
```

Umbrales de Éxito:
- `http_req_duration p(95) < 100ms`
- `http_req_duration p(99) < 200ms`
- `errors < 5%`
- `RPS: 800-1,300` (limitado por Argon2id CPU)

Ejecutar:
```bash
make load-test-login
# o
k6 run login.js
```

Qué mide:
- Performance de Argon2id (password hashing)
- Generación de JWT y refresh tokens
- Creación de sesiones
- Rate limiting

Nota: Este test crea 100 usuarios en el setup. Cada VU hace login con usuarios aleatorios.

### register.js - Registro de Usuarios

Objetivo: Medir el rendimiento del proceso de registro completo

Carga:
```
30s → 20 VUs
1m  → 100 VUs
2m  → 300 VUs (pico)
1m  → 300 VUs (sostenido)
30s → 0 VUs
```

Umbrales de Éxito:
- `http_req_duration p(95) < 150ms`
- `http_req_duration p(99) < 300ms`
- `errors < 5%`
- `RPS: 600-950`

Ejecutar:
```bash
make load-test-register
# o
k6 run register.js
```

Qué mide:
- Argon2id password hashing
- Inserción en PostgreSQL
- Email sending (async)
- Event publishing (async)
- Rate limiting

Nota: Cada request crea un usuario único. Pueden generarse miles de usuarios de prueba.

### mixed-load.js - Carga Mixta Realista

Objetivo: Simular tráfico real con múltiples operaciones simultáneas

Mix de Tráfico (4 escenarios en paralelo):
- 70% Validate Token → 700 VUs → Endpoint: `GET /auth/me`
- 20% Refresh Token → 200 VUs → Endpoint: `POST /auth/refresh`
- 8% Login → 80 VUs → Endpoint: `POST /auth/login`
- 2% Register → 20 VUs → Endpoint: `POST /auth/register`

Umbrales Globales:
- `http_req_duration p(95) < 100ms`
- `http_req_duration p(99) < 200ms`
- `errors < 5%`
- `RPS Total: 6,000-9,000`

Ejecutar:
```bash
make load-test-mixed
# o
k6 run mixed-load.js
```

Qué mide:
- Rendimiento del sistema bajo carga realista
- Comportamiento de diferentes endpoints en paralelo
- Contención de recursos (CPU, memoria, DB connections)
- Degradación gradual del sistema

Recomendación: Esta es la prueba más importante para validar el sistema en producción.

### Ejecutar Todas las Pruebas

```bash
# Ejecutar todas las pruebas secuencialmente
make load-test-all

# O manualmente
cd tests/k6
bash run-all-tests.sh       # Linux/Mac
run-all-tests.bat           # Windows
```

Los resultados se guardan en `tests/k6/results/` con timestamp.

## Interpretación de Resultados

### Métricas Clave de k6

#### http_req_duration
Tiempo total de respuesta HTTP (lo más importante)

```
http_req_duration..............: avg=12.3ms  p(95)=24.2ms  p(99)=38.5ms
```

- avg: Tiempo promedio de respuesta
- p(95): 95% de requests están por debajo de este valor
- p(99): 99% de requests están por debajo de este valor

Interpretación:
- Excelente: P95 < 50ms, P99 < 100ms
- Bueno: P95 < 100ms, P99 < 200ms
- Necesita optimización: P95 > 150ms, P99 > 300ms
- Crítico: P99 > 500ms (usuarios notarán lentitud)

#### http_reqs
Requests totales y throughput (RPS)

```
http_reqs......................: 112559  422.47/s
```

- **Total**: 112,559 requests ejecutados
- **Rate**: 422.47 requests/segundo (RPS)

**Interpretación**:
- Validate Token: **8,000-12,000 RPS** = Excelente
- Login: **800-1,300 RPS** = Excelente
- Register: **600-950 RPS** = Excelente
- Mixed Load: **6,000-9,000 RPS** = Excelente

#### http_req_failed
**Tasa de errores**

```
http_req_failed................: 0.00%   0   112559
```

- **0%** = Perfecto
- **< 1%** = Excelente
- **< 5%** = Aceptable (puede ser rate limiting)
- **> 10%** = Problema crítico

#### checks
**Validaciones personalizadas**

```
checks.........................: 100.00% 450234  0
```

Los checks validan:
- Status code correcto (200, 201)
- Body tiene los campos esperados
- Response time cumple SLA

#### vus y vus_max
**Usuarios virtuales concurrentes**

```
vus............................: 47      min=0    max=1000
vus_max........................: 1000    min=1000 max=1000
```

- **vus**: Usuarios activos en ese momento (47)
- **vus_max**: Máximo alcanzado durante la prueba (1000)

#### data_received / data_sent
**Tráfico de red**

```
data_received..................: 135 MB  507 kB/s
data_sent......................: 55 MB   207 kB/s
```

Útil para calcular ancho de banda necesario.

### Ejemplo: Análisis de Resultados

```
http_req_duration..............: avg=12.3ms  p(95)=24.2ms  p(99)=38.5ms
http_reqs......................: 112559  422.47/s
http_req_failed................: 0.00%
checks.........................: 100.00%
```

**Interpretación**:
- **P99 = 38.5ms** → Excelente (< 50ms)
- **422 RPS** → En este momento tiene 47 VUs activos
- **0% errores** → Sistema estable
- **100% checks** → Todas las validaciones pasaron

**Conclusión**: El sistema maneja la carga perfectamente. Puede escalar más.

## Configuración Avanzada

### Cambiar URL del Servicio

```bash
# Opción 1: Variable de entorno
export BASE_URL=http://192.168.1.100:8080
k6 run validate-token.js

# Opción 2: Flag -e
k6 run -e BASE_URL=http://192.168.1.100:8080 validate-token.js

# Opción 3: Producción
k6 run -e BASE_URL=https://auth.miapp.com validate-token.js
```

### Ajustar Carga (VUs y Duración)

Edita el script `.js` y modifica el campo `stages`:

```javascript
export const options = {
  stages: [
    { duration: '1m', target: 50 },    // Ramp up a 50 usuarios
    { duration: '5m', target: 200 },   // Ramp up a 200 usuarios
    { duration: '10m', target: 200 },  // Sostener 200 usuarios por 10 minutos
    { duration: '1m', target: 0 },     // Ramp down
  ],
};
```

### Guardar Resultados en JSON

```bash
k6 run --out json=results.json validate-token.js
```

El archivo JSON contiene todas las métricas para análisis posterior.

### Integración con Grafana Cloud (Gratis)

1. Crear cuenta en: https://grafana.com/auth/sign-up
2. Obtener token de Grafana Cloud k6
3. Ejecutar:

```bash
K6_CLOUD_TOKEN=tu_token_aqui k6 cloud run validate-token.js
```

Verás dashboards en tiempo real en el navegador.

### Pruebas de Estrés (Encontrar el Límite)

Para saber cuándo el sistema colapsa:

```javascript
export const options = {
  stages: [
    { duration: '2m', target: 500 },
    { duration: '2m', target: 1000 },
    { duration: '2m', target: 2000 },
    { duration: '2m', target: 3000 },   // Aumentar hasta que falle
    { duration: '2m', target: 4000 },
    { duration: '2m', target: 5000 },
    { duration: '1m', target: 0 },
  ],
};
```

Monitorea CPU, memoria y DB connections durante la prueba.

## Benchmarks Esperados

### Hardware: VPS 4 vCPU + 8GB RAM

| Operación | P95 | P99 | RPS Sostenido | RPS Pico | Cuello de Botella |
|-----------|-----|-----|---------------|----------|-------------------|
| **Validate Token** | <30ms | <50ms | 8,000-10,000 | 12,000-15,000 | CPU (JWT verify) |
| **Login** | <80ms | <150ms | 800-1,000 | 1,200-1,500 | CPU (Argon2id) |
| **Register** | <120ms | <250ms | 600-800 | 900-1,100 | CPU (Argon2id) + Email |
| **Refresh Token** | <60ms | <100ms | 1,500-2,000 | 2,500-3,000 | PostgreSQL + Redis |
| **Mixed Load** | <80ms | <150ms | 6,000-8,000 | 9,000-12,000 | Mix |

### Factores que Afectan el Rendimiento

**Mejoran**:
- Redis cache hit ratio >95%
- Índices de base de datos optimizados
- Connection pooling configurado
- Goroutines async para operaciones no-críticas

**Limitan**:
- Argon2id (CPU intensivo por diseño)
- PostgreSQL sin replicas (operaciones sin cache)
- Rate limiting activo

## Troubleshooting

### Error: "k6: command not found"

**Causa**: k6 no está en el PATH

**Solución**:
```bash
# Verificar instalación
where k6  # Windows
which k6  # Linux/Mac

# Reinstalar
choco install k6
```

### Error: "connection refused"

**Causa**: El servicio no está corriendo

**Solución**:
```bash
# Verificar que esté activo
curl http://localhost:8080/health

# Iniciar si no está corriendo
make run
```

### Error: "too many open files"

**Causa**: Límite de archivos abiertos del sistema

**Solución Linux/Mac**:
```bash
ulimit -n 10000
```

**Solución Windows**: Generalmente no es necesario, Windows maneja más file descriptors.

### Muchos Errores 429 (Rate Limiting)

**Causa**: Rate limiting activado en pruebas intensivas

**Solución temporal para testing**:
```bash
# Desactivar rate limiting temporalmente
# Editar .env o config
RATE_LIMIT_ENABLED=false
```

**No recomendado en producción**.

### PostgreSQL Connection Errors

**Causa**: Pool de conexiones agotado

**Solución**:
```bash
# Aumentar max connections
# En .env
POSTGRES_MAX_CONNS=80  # Aumentar de 40 a 80
```

### Alta Latencia en Validate Token (>100ms P99)

**Posibles causas**:
1. **Redis lento**: Verificar con `redis-cli --latency`
2. **Cache miss alto**: Revisar logs de cache hit ratio
3. **CPU saturado**: Reducir VUs o escalar hardware

**Diagnóstico**:
```bash
# Monitorear durante la prueba
top  # CPU usage
redis-cli info stats | grep keyspace_hits  # Cache hits
```

## Recursos Adicionales

- **k6 Documentación Oficial**: https://k6.io/docs/
- **k6 Ejemplos**: https://k6.io/docs/examples/
- **Grafana Cloud k6** (gratis): https://grafana.com/products/cloud/k6/
- **k6 Slack Community**: https://k6.io/slack
- **Auth Service Repo**: [../../../README.md](../../README.md)

## Contribuir

Para agregar nuevos tests:

1. Crear archivo `.js` en `tests/k6/`
2. Seguir la estructura de los tests existentes
3. Agregar entrada en `Makefile`
4. Documentar en este README

## Licencia

Este proyecto es parte del Auth Service. Ver LICENSE en la raíz del proyecto.

---
