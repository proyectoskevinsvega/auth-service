# 🔐 gRPC Mutual TLS (mTLS) Setup

Para garantizar que solo plataformas y servicios autorizados puedan comunicarse con el `auth-service` vía gRPC, hemos implementado **Mutual TLS (mTLS)**. Esto significa que tanto el servidor como el cliente deben presentar certificados válidos firmados por una Autoridad de Certificación (CA) común.

## 1. Conceptos Clave

- **Root CA**: La entidad de confianza que firma todos los certificados.
- **Server Certificate**: El certificado que el `auth-service` presenta a los clientes.
- **Client Certificate**: El certificado que tu otra plataforma debe presentar al `auth-service`.

---

## 2. Generación de Certificados

Hemos incluido un script para facilitar la creación de estos archivos en entornos de desarrollo o privados:

```bash
# Otorgar permisos
chmod +x scripts/generate_mtls_certs.sh

# Ejecutar el script
./scripts/generate_mtls_certs.sh
```

Esto generará en la carpeta `keys/`:

- `ca.pem`: El certificado de la CA (necesario para todos).
- `server.pem` y `server-key.pem`: Para el Auth Service.
- `client.pem` y `client-key.pem`: Para un cliente de ejemplo.

---

## 3. Configuración del Servidor (Auth Service)

En tu archivo `.env`, activa el mTLS:

```env
GRPC_TLS_ENABLED=true
GRPC_CA_CERT_PATH=./keys/ca.pem
GRPC_SERVER_CERT_PATH=./keys/server.pem
GRPC_SERVER_KEY_PATH=./keys/server-key.pem
```

---

## 4. Cómo conectar desde otro servicio (Cliente)

Para que otra plataforma se conecte, debe cargar el `ca.pem` para verificar al servidor y sus propios `client.pem`/`client-key.pem` para identificarse.

### Ejemplo en Go:

```go
// 1. Cargar el certificado de la CA
caCert, _ := os.ReadFile("path/to/ca.pem")
caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)

// 2. Cargar el par de llaves del cliente
clientCert, _ := tls.LoadX509KeyPair("path/to/client.pem", "path/to/client-key.pem")

// 3. Crear configuración TLS
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{clientCert},
    RootCAs:      caCertPool,
}

// 4. Conectar al gRPC
creds := credentials.NewTLS(tlsConfig)
conn, _ := grpc.Dial("auth.yourdomain.com:443", grpc.WithTransportCredentials(creds))
```

---

## 5. gRPC a través de Cloudflare

Si usas Cloudflare con el proxy activado (🟠):

1. Ve a **SSL/TLS** -> **Edge Certificates**.
2. Asegúrate de que **gRPC** esté activado en la pestaña **Network**.
3. **Importante**: Cloudflare romperá el mTLS si no usas [Cloudflare Authenticated Origin Pulls](https://developers.cloudflare.com/ssl/origin-configuration/authenticated-origin-pull/) o si esperas que el certificado del cliente llegue intacto al servidor.

> [!WARNING]
> En producciones con Cloudflare, el mTLS suele terminarse en Cloudflare o requerir configuraciones avanzadas. Sin Cloudflare, Nginx puede manejar el mTLS directamente.
