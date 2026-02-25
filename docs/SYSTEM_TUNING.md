# 🚀 High-Scale System Tuning Guide (Linux)

Para manejar millones de usuarios y miles de peticiones por segundo, la configuración de Nginx no es suficiente; el sistema operativo (Kernel) también debe estar optimizado.

Aplica estos ajustes en tu archivo `/etc/sysctl.conf` para liberar el potencial de tu servidor.

## 1. Tuning del Kernel (sysctl)

Añade o modifica estas líneas en `/etc/sysctl.conf`:

```conf
# Aumenta el límite de conexiones pendientes (Backlog)
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535

# Permite reutilizar sockets en estado TIME_WAIT (Crítico para alta carga)
net.ipv4.tcp_tw_reuse = 1

# Aumenta el rango de puertos efímeros disponibles
net.ipv4.ip_local_port_range = 1024 65535

# Aumenta el tamaño de los buffers de red para manejar ráfagas
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# Desactiva el escalado lento de TCP
net.ipv4.tcp_slow_start_after_idle = 0

# Aumenta el límite de archivos abiertos para todo el sistema
fs.file-max = 1000000
```

**Aplicar cambios:**

```bash
sudo sysctl -p
```

---

## 2. Límites de Usuario (ulimit)

Nginx necesita permiso para abrir miles de archivos (sockets). Modifica `/etc/security/limits.conf`:

```conf
* soft nofile 65535
* hard nofile 65535
www-data soft nofile 65535
www-data hard nofile 65535
```

---

## 3. Estrategias de Escalado (Próximos Pasos)

Si un solo servidor llega al 80% de CPU con estos ajustes, debes considerar:

1. **Escalado Horizontal**: Meter un Load Balancer de Cloudflare (o HAProxy) delante de **múltiples** instancias del `auth-service`.
2. **Base de Datos**: PostgreSQL puede convertirse en el cuello de botella. Asegúrate de usar `pgBouncer` para el pooling de conexiones si el número de conexiones simultáneas supera las 500.
3. **Redis**: Úsalo para sesiones y rate-limiting distribuido.

---

> [!IMPORTANT]
> Estos ajustes son agresivos y están diseñados para servidores dedicados de alto rendimiento. Pruébalos primero en un entorno de staging.
