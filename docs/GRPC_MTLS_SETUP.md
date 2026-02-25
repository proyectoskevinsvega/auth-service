# 🔐 gRPC Mutual TLS (mTLS) Setup

Para garantizar que solo plataformas y servicios autorizados puedan comunicarse con el `auth-service` vía gRPC, hemos implementado **Mutual TLS (mTLS)**. Esto significa que tanto el servidor como el cliente deben presentar certificados válidos firmados por una Autoridad de Certificación (CA) común.

## 1. Conceptos Clave

- **Root CA**: La entidad de confianza que firma todos los certificados.
- **Server Certificate**: El certificado que el `auth-service` presenta a los clientes.
- **Client Certificate**: El certificado que tu otra plataforma debe presentar al `auth-service`.
- **Identity (CN)**: El servicio extrae el _Common Name_ de tu certificado para identificarte en los logs.

---

## 2. Generación Automática

**Auth Service** generará automáticamente la CA y sus propios certificados de servidor al iniciar si `GRPC_TLS_ENABLED=true` y los archivos no existen en la carpeta `./keys`.

---

## 3. Emisión de Certificados vía API (M2M)

### Opción A: Flujo Convencional (Conveniente)

El servidor genera la llave privada y el certificado por ti.

**POST** `/api/v1/admin/m2m/certificates`  
**Auth**: Bearer Token (Rol: Admin)

```json
{
  "client_name": "Nombre_Empresa_Aliada"
}
```

### Opción B: Flujo Élite (Zero Knowledge - Recomendado)

**Máxima Seguridad**: Tú generas tu propia llave privada localmente y solo nos envías una solicitud de firma (CSR). **Nosotros nunca vemos ni tenemos acceso a tu llave privada.**

**Pasos**:

1. Genera tu llave privada y CSR localmente:
   ```bash
   openssl req -new -newkey rsa:4096 -nodes -keyout client.key -out client.csr -subj "/CN=Nombre_Empresa_Aliada"
   ```
2. Envía el contenido del archivo `client.csr` al servidor:

**POST** `/api/v1/admin/m2m/certificates/sign`  
**Auth**: Bearer Token (Rol: Admin)

```json
{
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIB... (contenido de client.csr) \n-----END CERTIFICATE REQUEST-----"
}
```

**Respuesta**:
El servidor te devolverá el `certificate` firmado y el `ca_certificate`. Tú ya tienes la `private_key` (client.key) en tu servidor.

---

## 4. Configuración del Servidor (Auth Service)

En tu archivo `.env`, activa el mTLS:

```env
GRPC_TLS_ENABLED=true
GRPC_CA_CERT_PATH=./keys/ca.pem
GRPC_SERVER_CERT_PATH=./keys/server.pem
GRPC_SERVER_KEY_PATH=./keys/server-key.pem
```

---

## 5. Cómo conectar desde otro servicio (Cliente)

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

## 6. gRPC a través de Cloudflare

Si usas Cloudflare con el proxy activado (🟠):

1. Ve a **SSL/TLS** -> **Edge Certificates**.
2. Asegúrate de que **gRPC** esté activado en la pestaña **Network**.
3. **Importante**: Cloudflare romperá el mTLS si no usas [Cloudflare Authenticated Origin Pulls](https://developers.cloudflare.com/ssl/origin-configuration/authenticated-origin-pull/).

> [!WARNING]
> En producciones con Cloudflare, el mTLS suele terminarse en Cloudflare. Sin Cloudflare, Nginx puede manejar el mTLS directamente.
