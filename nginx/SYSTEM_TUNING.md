# 🚀 Guía de Despliegue: Nginx + Cloudflare + Kernel Tuning (auth-service)

Esta guía explica paso a paso cómo configurar tu servidor VPS (Ubuntu/Debian) para exponer el microservicio `auth-service` de forma segura a través del dominio `auth.bravexcolombia.com` usando Cloudflare, y cómo optimizar la red del servidor para alta carga.

---

## Paso 1: Instalar Nginx

Conéctate a tu servidor VPS por SSH y ejecuta el siguiente comando en la terminal para instalar el servidor web Nginx:

```bash
sudo apt update
sudo apt install nginx -y
```

## Paso 2: Configurar los Certificados de Cloudflare

Para que la conexión entre Cloudflare y el VPS sea cifrada, necesitas instalar los certificados de Origen.

1. Entra a tu panel de **Cloudflare > SSL/TLS > Origin Server** y genera un nuevo certificado.
2. En tu servidor VPS, crea la carpeta donde guardaremos las llaves:

   ```bash
   sudo mkdir -p /etc/nginx/ssl
   ```

3. Crea el archivo del certificado público (`.crt`) y pega el contenido provisto por Cloudflare (Origin Certificate):

   ```bash
   sudo nano /etc/nginx/ssl/cloudflare-origin.crt
   ```

4. Crea el archivo de la llave privada (`.key`) y pega el contenido provisto por Cloudflare (Private Key):

   ```bash
   sudo nano /etc/nginx/ssl/cloudflare-origin.key
   ```

## Paso 3: Aplicar la Configuración Nginx (Virtual Hosts)

En servidores con múltiples aplicaciones corriendo (como `pgadmin` o `minio`), **NUNCA** debemos borrar o sobrescribir el archivo global `nginx.conf`. En su lugar, creamos un archivo dedicado (Virtual Host) en la carpeta `sites-available`.

1. Crea un nuevo bloque de servidor para tu microservicio de autenticación:

   ```bash
   sudo nano /etc/nginx/sites-available/auth-service
   ```

2. Pega esta configuración exacta:

   ```nginx
   # Redirigir el puerto 80 a HTTPS (Seguridad)
   server {
       listen 80;
       server_name auth.bravexcolombia.com;
       return 301 https://$host$request_uri;
   }

   # Escuchar el puerto 443 con los Certificados SSL encriptados
   server {
       listen 443 ssl;
       server_name auth.bravexcolombia.com;

       ssl_certificate /etc/nginx/ssl/cloudflare-origin.crt;
       ssl_certificate_key /etc/nginx/ssl/cloudflare-origin.key;

       # Enviar el tráfico al backend de Go (auth-service)
       location / {
           proxy_pass http://127.0.0.1:8002;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto https;
       }
   }
   ```

   _Guarda y cierra el archivo (`CTRL+O`, `Enter`, `CTRL+X`)._

3. Activa tu configuración creando un enlace simbólico hacia `sites-enabled`:

   ```bash
   sudo ln -s /etc/nginx/sites-available/auth-service /etc/nginx/sites-enabled/
   ```

4. Valida que no haya errores de sintaxis y reinicia Nginx para aplicar los cambios:

   ```bash
   sudo nginx -t
   sudo systemctl reload nginx
   ```

---

## 🏎️ Paso 4: Optimización Extrema del Sistema Operativo (Kernel)

Para soportar miles de conexiones simultáneas y evitar cuellos de botella de red hacia tu microservicio, debes optimizar el Kernel de Linux (Ubuntu).

### Modificando `/etc/sysctl.conf`

Este archivo rige las configuraciones de los recursos y la red de Ubuntu a nivel más profundo. Siempre se ubica en la carpeta de configuraciones global del sistema operativo (`/etc/`).

1. Abre el archivo de configuración del Kernel como administrador:

   ```bash
   sudo nano /etc/sysctl.conf
   ```

2. Ve hasta el final del archivo y **añade (no borres lo demás)** estas directivas:

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

   _Guarda y cierra el archivo (`CTRL+O`, `Enter`, `CTRL+X`)._

3. Aplica los cambios en todo el sistema sin reiniciar el VPS con el siguiente comando:

   ```bash
   sudo sysctl -p
   ```

### Modificando `/etc/security/limits.conf`

Finalmente, necesitamos levantar los límites artificiales que Linux le pone a la aplicación de servidor web para abrir sockets (archivos). Al igual que el anterior, este es un archivo de sistema operativo que se modifica directamente desde cualquier lado en Ubuntu.

1. Abre el archivo de límites:

   ```bash
   sudo nano /etc/security/limits.conf
   ```

2. Añade este bloque exactamente al final:

   ```conf
   * soft nofile 65535
   * hard nofile 65535
   www-data soft nofile 65535
   www-data hard nofile 65535
   ```

   _Guarda y cierra el archivo (`CTRL+O`, `Enter`, `CTRL+X`)._

> ⚠️ **Aviso Importante sobre Systemd**: Modificar `/etc/security/limits.conf` solo funciona si el servicio no está corriendo bajo `systemd` con override, o para sesiones de usuario que se reinicien. Si tu aplicación (`auth-service`) corre como un servicio `systemd`, Linux ignorará `limits.conf` para ese proceso.
>
> En ese caso, debes aplicar el límite directamente al servicio:
>
> ```bash
> sudo systemctl edit tu-servicio
> ```
>
> Y añadir las siguientes líneas:
>
> ```ini
> [Service]
> LimitNOFILE=65535
> ```
>
> Tras guardar, ejecuta `sudo systemctl daemon-reload` y `sudo systemctl restart tu-servicio`.

¡Listo! Para que estos cambios surtan efecto por completo a nivel de kernel, y puesto que Nginx ha sido reiniciado, el microservicio está preparado al 100% para alta demanda.
