# 🌐 Guía de Despliegue con Nginx y Cloudflare

Esta guía detalla los pasos necesarios para instalar y configurar Nginx como un proxy reverso seguro para el `auth-service` en un VPS Linux (Ubuntu/Debian), optimizado para trabajar con Cloudflare.

## Prerrequisitos

- Un servidor VPS con acceso root o sudo.
- El `auth-service` ejecutándose (por defecto en el puerto `8082`).
- Un dominio apuntando a la IP de tu servidor a través de Cloudflare (Proxy activado 🟠).

---

## Paso 1: Instalación de Nginx

Actualiza los repositorios e instala Nginx:

```bash
sudo apt update
sudo apt install nginx -y
```

## Paso 2: Preparar Directorios de Configuración

Crea los directorios necesarios si no existen:

```bash
sudo mkdir -p /etc/nginx/ssl
sudo mkdir -p /etc/nginx/conf.d
```

## Paso 3: Copiar Archivos de Configuración

Desde la carpeta del proyecto `auth-service`, copia los archivos pre-configurados:

```bash
# Copia la configuración de IPs de Cloudflare
sudo cp nginx/cloudflare_ips.conf /etc/nginx/conf.d/cloudflare_ips.conf

# Copia la configuración principal de Nginx
sudo cp nginx/nginx.conf /etc/nginx/nginx.conf
```

## Paso 4: Configurar SSL (Certificados)

Tienes dos opciones principales:

### Opción A: Certificados de Origen de Cloudflare (Recomendado)

1. Genera un "Origin Certificate" en el panel de Cloudflare (SSL/TLS -> Origin Server).
2. Guarda el certificado en `/etc/nginx/ssl/cert.pem`.
3. Guarda la llave privada en `/etc/nginx/ssl/key.pem`.
4. Edita `/etc/nginx/nginx.conf` y descomenta/ajusta las líneas de SSL:
   ```nginx
   ssl_certificate /etc/nginx/ssl/cert.pem;
   ssl_certificate_key /etc/nginx/ssl/key.pem;
   ```

### Opción B: Let's Encrypt (Certbot)

1. Instala Certbot: `sudo apt install certbot python3-certbot-nginx -y`
2. Genera el certificado: `sudo certbot --nginx -d tu-dominio.com`

## Paso 5: Tuning del Sistema (Alta Escala)

Para soportar millones de solicitudes, aplica los ajustes del kernel definidos en:
👉 [SYSTEM_TUNING.md](SYSTEM_TUNING.md)

## Paso 6: Verificar y Reiniciar

Verifica que no haya errores de sintaxis y reinicia el servicio:

```bash
# Verificar sintaxis
sudo nginx -t

# Reiniciar Nginx
sudo systemctl restart nginx

# Habilitar Nginx en el arranque
sudo systemctl enable nginx
```

---

## Verificación de Seguridad

1. **Test de IP Real**: Revisa los logs de acceso de Nginx (`/var/log/nginx/access.log`). Deberías ver las IPs reales de los usuarios gracias a `cloudflare_ips.conf`.
2. **Acceso Directo**: Intenta acceder a tu IP pública directamente por el puerto 8082. Asegúrate de configurar tu firewall (`ufw`) para permitir solo tráfico en los puertos 80 y 443 desde Cloudflare.

```bash
# Opcional: Firewall hardening con UFW
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw default deny incoming
sudo ufw enable
```

---

> [!TIP]
> Si escalas horizontalmente, solo necesitas añadir más líneas `server IP:PORT;` en el bloque `upstream auth_service` dentro de `nginx.conf`.
