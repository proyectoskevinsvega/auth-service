# Benchmarks - Performance Testing

## Overview

Los benchmarks miden el rendimiento de operaciones críticas del Auth Service para:
- Identificar bottlenecks
- Validar optimizaciones
- Garantizar SLAs de performance
- Comparar implementaciones alternativas

## Performance Targets

| Operación | Target p99 | Target p50 |
|-----------|-----------|-----------|
| ValidateToken (cache hit) | <2ms | <1ms |
| ValidateToken (cache miss) | <5ms | <3ms |
| Login | <50ms | <30ms |
| Register | <100ms | <50ms |
| RevokeToken | <10ms | <5ms |

## Ejecutar Benchmarks

### Todos los benchmarks
```bash
go test -bench=. -benchmem ./tests/benchmarks
```

### Benchmark específico
```bash
go test -bench=BenchmarkTokenValidation_CacheHit -benchmem ./tests/benchmarks
```

### Con más iteraciones para mayor precisión
```bash
go test -bench=. -benchmem -benchtime=10s ./tests/benchmarks
```

### Guardar resultados
```bash
go test -bench=. -benchmem ./tests/benchmarks > bench_$(date +%Y%m%d_%H%M%S).txt
```

### Comparar resultados (antes y después de optimización)
```bash
# Guardar baseline
go test -bench=. -benchmem ./tests/benchmarks > bench_before.txt

# Hacer cambios de optimización...

# Guardar resultados nuevos
go test -bench=. -benchmem ./tests/benchmarks > bench_after.txt

# Comparar con benchstat
go install golang.org/x/perf/cmd/benchstat@latest
benchstat bench_before.txt bench_after.txt
```

## Benchmarks Implementados

### Token Validation

#### 1. BenchmarkTokenValidation_CacheHit
**Objetivo**: <2ms p99

Mide la validación de tokens cuando están en cache (hot path). Este es el caso más común en producción (>95% de requests).

```bash
go test -bench=BenchmarkTokenValidation_CacheHit -benchmem ./tests/benchmarks
```

**Ejemplo de salida esperada:**
```
BenchmarkTokenValidation_CacheHit-8    500000    2000 ns/op    512 B/op    8 allocs/op
```

#### 2. BenchmarkTokenValidation_CacheMiss
**Objetivo**: <5ms p99

Mide la validación completa con verificación criptográfica cuando el token no está en cache (cold path).

```bash
go test -bench=BenchmarkTokenValidation_CacheMiss -benchmem ./tests/benchmarks
```

#### 3. BenchmarkTokenValidation_Blacklist
**Objetivo**: <1ms p99

Mide la velocidad de detección de tokens en blacklist (debería ser muy rápido).

```bash
go test -bench=BenchmarkTokenValidation_Blacklist -benchmem ./tests/benchmarks
```

#### 4. BenchmarkTokenRevocation
**Objetivo**: <10ms p99

Mide el tiempo para revocar un token (agregar a blacklist + eliminar de cache).

```bash
go test -bench=BenchmarkTokenRevocation -benchmem ./tests/benchmarks
```

### Authentication Operations

#### 5. BenchmarkLogin
**Objetivo**: <50ms p99

Mide el flujo completo de login incluyendo:
- Rate limiting check
- Usuario lookup
- Password verification
- Token generation
- Session creation
- Audit log

```bash
go test -bench=BenchmarkLogin -benchmem ./tests/benchmarks
```

#### 6. BenchmarkRegister
**Objetivo**: <100ms p99

Mide el flujo completo de registro incluyendo:
- Validaciones
- Password hashing (más lento por diseño - Argon2id)
- Usuario creation
- Email sending
- Event publishing

```bash
go test -bench=BenchmarkRegister -benchmem ./tests/benchmarks
```

#### 7. BenchmarkPasswordVerification
**Objetivo**: <30ms p99

Mide específicamente la verificación de password con Argon2id. Esta operación es intencionalmente lenta para resistir ataques de fuerza bruta.

```bash
go test -bench=BenchmarkPasswordVerification -benchmem ./tests/benchmarks
```

## Interpretar Resultados

### Formato de Salida
```
BenchmarkName-8    iterations    ns/op    B/op    allocs/op
```

- **iterations**: Número de veces que se ejecutó la operación
- **ns/op**: Nanosegundos por operación (1ms = 1,000,000 ns)
- **B/op**: Bytes allocados por operación
- **allocs/op**: Número de allocaciones de memoria por operación

### Ejemplo de Análisis
```
BenchmarkTokenValidation_CacheHit-8    500000    2000 ns/op    512 B/op    8 allocs/op
```

**Análisis:**
- Tiempo: 2000ns = 2μs (excelente, muy por debajo del target de 2ms)
- Memoria: 512 bytes por operación (aceptable)
- Allocaciones: 8 allocaciones (considerar optimizar si es crítico)

### Red Flags
- Más de 10,000 ns/op para cache hit
- Más de 100,000 ns/op para cache miss
- Crecimiento de memoria con más iteraciones (memory leak)
- Más de 50 allocaciones por operación

## Profiling

### CPU Profiling
```bash
go test -bench=. -cpuprofile=cpu.prof ./tests/benchmarks
go tool pprof cpu.prof
```

En el pprof shell:
```
top10          # Top 10 funciones por CPU time
list FuncName  # Ver código de una función específica
web            # Generar gráfico (requiere graphviz)
```

### Memory Profiling
```bash
go test -bench=. -memprofile=mem.prof ./tests/benchmarks
go tool pprof mem.prof
```

### Trace
```bash
go test -bench=BenchmarkLogin -trace=trace.out ./tests/benchmarks
go tool trace trace.out
```

## Monitoreo de Performance

### Regression Testing
Ejecutar benchmarks regularmente y comparar con baseline:

```bash
# scripts/benchmark.sh
#!/bin/bash

DATE=$(date +%Y%m%d_%H%M%S)
OUTPUT="bench_results/bench_${DATE}.txt"

# Ejecutar benchmarks
go test -bench=. -benchmem ./tests/benchmarks > "$OUTPUT"

# Comparar con baseline si existe
if [ -f "bench_results/baseline.txt" ]; then
    benchstat bench_results/baseline.txt "$OUTPUT"
fi
```

### CI/CD Integration
```yaml
# .github/workflows/benchmark.yml
name: Benchmark
on: [push, pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4

      - name: Run Benchmarks
        run: go test -bench=. -benchmem ./tests/benchmarks

      - name: Compare with baseline
        run: |
          benchstat baseline.txt bench_current.txt || true
```

## Optimization Strategies

### Si ValidateToken es lento (>5ms)
1. Verificar hit rate del cache
2. Optimizar serialización/deserialización
3. Usar connection pooling para Redis
4. Considerar cache en memoria local (sync.Map)

### Si Login es lento (>100ms)
1. Verificar queries de base de datos (EXPLAIN)
2. Agregar índices faltantes
3. Optimizar Argon2id parameters (balance seguridad/performance)
4. Paralelizar operaciones independientes

### Si Memory allocation es alta
1. Usar sync.Pool para objetos reutilizables
2. Precalcular tamaños de slices/maps
3. Evitar string concatenation en loops
4. Reducir conversiones de tipos

## Mejores Prácticas

1. **Ejecutar múltiples veces** para resultados consistentes
   ```bash
   go test -bench=. -count=10 ./tests/benchmarks
   ```

2. **Excluir setup time** con `b.ResetTimer()`
   ```go
   // Setup...
   b.ResetTimer()
   // Benchmark...
   ```

3. **Evitar optimizaciones del compilador**
   ```go
   var result *domain.Token
   b.ResetTimer()
   for i := 0; i < b.N; i++ {
       result, _ = uc.ValidateToken(ctx, token)
   }
   _ = result // Prevenir dead code elimination
   ```

4. **Usar -benchmem** siempre para detectar memory issues

5. **Documentar baselines** en README para referencia

## Recursos

- [Go Benchmark Tutorial](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [benchstat tool](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
