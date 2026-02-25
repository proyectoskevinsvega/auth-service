# Despliegue con Systemd

Archivos necesarios para ejecutar el Auth Service como un servicio systemd en servidores Linux.

## Requisitos Previos

- Sistema operativo Linux con systemd (Ubuntu 16.04+, Debian 8+, CentOS 7+)
- Acceso root o sudo
- PostgreSQL y Redis instalados y corriendo
- Binario compilado del servicio (`make build`)

## Instalación

### Compilar el Servicio

```bash
cd /ruta/al/proyecto
make build
```

### Configurar Variables de Entorno

Copia y configura tu archivo `.env`:

```bash
cp .env.example .env
nano .env  # Configura tus valores
```

### Instalar como Servicio

```bash
cd deployments/systemd
sudo bash install.sh
```

El script hará lo siguiente:
- Crear usuario y grupo `auth-service`
- Copiar binario a `/opt/auth-service/bin/`
- Copiar claves RSA a `/opt/auth-service/keys/`
- Copiar archivo `.env` a `/opt/auth-service/`
- Instalar servicio systemd
- Habilitar inicio automático

### Iniciar el Servicio

```bash
sudo systemctl start auth-service
```

## Gestión del Servicio

### Comandos Básicos

```bash
# Iniciar el servicio
sudo systemctl start auth-service

# Detener el servicio
sudo systemctl stop auth-service

# Reiniciar el servicio
sudo systemctl restart auth-service

# Ver estado del servicio
sudo systemctl status auth-service

# Habilitar inicio automático al arrancar el sistema
sudo systemctl enable auth-service

# Deshabilitar inicio automático
sudo systemctl disable auth-service
```

### Ver Logs

```bash
# Ver logs en tiempo real
sudo journalctl -u auth-service -f

# Ver logs de las últimas 100 líneas
sudo journalctl -u auth-service -n 100

# Ver logs de hoy
sudo journalctl -u auth-service --since today

# Ver logs de un rango de tiempo específico
sudo journalctl -u auth-service --since "2024-01-01" --until "2024-01-02"
```

## Configuración

### Archivo de Servicio

El archivo `auth-service.service` contiene la configuración del servicio systemd:

```ini
[Unit]
Description=Vertercloud Authentication Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=auth-service
WorkingDirectory=/opt/auth-service
EnvironmentFile=/opt/auth-service/.env
ExecStart=/opt/auth-service/bin/auth-service
Restart=always

[Install]
WantedBy=multi-user.target
```

### Modificar Configuración

Para modificar la configuración:

1. Edita el archivo de servicio:
```bash
sudo nano /etc/systemd/system/auth-service.service
```

2. Recarga systemd:
```bash
sudo systemctl daemon-reload
```

3. Reinicia el servicio:
```bash
sudo systemctl restart auth-service
```

## Seguridad

El servicio se ejecuta con las siguientes medidas de seguridad:

- Usuario dedicado: Se ejecuta como usuario `auth-service` sin privilegios
- NoNewPrivileges: No puede escalar privilegios
- PrivateTmp: Tiene un directorio `/tmp` privado
- ProtectSystem: Sistema de archivos del sistema protegido (solo lectura)
- ProtectHome: Directorios home de otros usuarios protegidos
- ReadWritePaths: Solo puede escribir en `/opt/auth-service/logs`

### Permisos de Archivos

```bash
# Archivo .env (contiene secretos)
-rw------- auth-service:auth-service /opt/auth-service/.env

# Claves RSA (contienen claves privadas)
-rw------- auth-service:auth-service /opt/auth-service/keys/*.pem

# Binario
-rwxr-xr-x auth-service:auth-service /opt/auth-service/bin/auth-service
```

## Actualización

Para actualizar el servicio a una nueva versión:

```bash
# 1. Compilar nueva versión
make build

# 2. Detener el servicio
sudo systemctl stop auth-service

# 3. Reemplazar binario
sudo cp bin/auth-service /opt/auth-service/bin/

# 4. Ajustar permisos
sudo chown auth-service:auth-service /opt/auth-service/bin/auth-service
sudo chmod +x /opt/auth-service/bin/auth-service

# 5. Iniciar el servicio
sudo systemctl start auth-service

# 6. Verificar que todo funciona
sudo systemctl status auth-service
sudo journalctl -u auth-service -n 50
```

## Desinstalación

Para desinstalar completamente el servicio:

```bash
cd deployments/systemd
sudo bash uninstall.sh
```

El script preguntará si deseas:
- Eliminar el directorio `/opt/auth-service`
- Eliminar el usuario `auth-service`

## Troubleshooting

### El servicio no inicia

Ver logs detallados:

```bash
sudo journalctl -u auth-service -n 100 --no-pager
```

Verificar configuración:

```bash
sudo systemctl status auth-service
```

Verificar archivo .env:

```bash
sudo cat /opt/auth-service/.env
```

Probar manualmente:

```bash
sudo -u auth-service /opt/auth-service/bin/auth-service
```

### Error de conexión a base de datos

Verificar que PostgreSQL está corriendo:

```bash
sudo systemctl status postgresql
```

Verificar que Redis está corriendo:

```bash
sudo systemctl status redis
```

Verificar conectividad:

```bash
ping -c 3 <POSTGRES_HOST>
redis-cli -h <REDIS_HOST> ping
```

### El servicio se reinicia constantemente

Ver logs para identificar el problema:

```bash
sudo journalctl -u auth-service -f
```

Verificar que el puerto no está en uso:

```bash
sudo netstat -tlnp | grep 8080
sudo netstat -tlnp | grep 9090
```

### Permisos denegados

Verificar permisos del directorio:

```bash
ls -la /opt/auth-service
```

Corregir permisos si es necesario:

```bash
sudo chown -R auth-service:auth-service /opt/auth-service
sudo chmod 600 /opt/auth-service/.env
sudo chmod 600 /opt/auth-service/keys/*.pem
```

## Estructura de Archivos

```
/opt/auth-service/
├── bin/
│   └── auth-service          # Binario del servicio
├── keys/
│   ├── private.pem           # Clave privada RSA
│   └── public.pem            # Clave pública RSA
├── logs/                     # Logs (si se usan archivos)
└── .env                      # Variables de entorno
```

## Referencias

- systemd Documentation: https://www.freedesktop.org/software/systemd/man/
- journalctl Manual: https://www.freedesktop.org/software/systemd/man/journalctl.html
- Systemd Service Security: https://www.freedesktop.org/software/systemd/man/systemd.exec.html
