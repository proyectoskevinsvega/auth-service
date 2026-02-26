# Integración B2B con Auth Service via gRPC

Este documento está dirigido a los equipos de ingeniería de **empresas cliente (Tenants)** que consumen el `auth-service` como su proveedor centralizado de identidades (Identity-as-a-Service).

Para validar la sesión de tus usuarios de manera ultra-rápida y segura, proporcionamos una interfaz **gRPC** de alto rendimiento en el puerto `50051`.

## 1. El Contrato: Archivo `.proto`

El primer paso para integrarte es compilar los "stubs" (código cliente) en el lenguaje que use el backend de tu empresa (Go, Java, Node.js, Python, etc.) utilizando este contrato exacto:

```protobuf
syntax = "proto3";

package auth.v1;

service AuthService {
  // Valida el JWT y retorna toda la info del usuario
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  // Revoca un token en la base de datos centralizada
  rpc RevokeToken(RevokeTokenRequest) returns (RevokeTokenResponse);
  // Obtiene el perfil de un usuario específico
  rpc GetUserByID(GetUserByIDRequest) returns (GetUserByIDResponse);
}

message ValidateTokenRequest {
  string token = 1;
  string tenant_id = 2;
}

message ValidateTokenResponse {
  bool valid = 1;
  string user_id = 2;
  string email = 3;
  string username = 4;
  string tenant_id = 5;
  bool active = 6;
  bool email_verified = 7;
  bool two_factor_enabled = 8;
  string error_code = 9;
  string error_message = 10;
}

// ... (Obtén el archivo auth.proto completo del equipo de integración)
```

---

## 2. Ejemplo de Implementación: Backend en Go (Golang)

Si tu microservicio o backend está escrito en **Go**, así es como puedes llamar a nuestro Auth Service para proteger tus propias rutas:

```go
package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// Importa el paquete generado a partir del auth.proto
	pb "github.com/tu-empresa/proto-gen/auth/v1"
)

func main() {
	// 1. Establecer conexión gRPC con el Auth Service
	// (En producción usa credenciales TLS)
	conn, err := grpc.Dial("auth.tu-proveedor.com:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("No se pudo conectar: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)

	// 2. Extraer el JWT que te envió tu Frontend (ej. de la cabecera / cookie)
	tokenFromFrontend := "eyJhbGciOiJIUzI1NiIs..."
	myTenantID := "mi-empresa-tenant"

	// 3. Llamar al servicio gRPC con un timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	res, err := client.ValidateToken(ctx, &pb.ValidateTokenRequest{
		Token:    tokenFromFrontend,
		TenantId: myTenantID,
	})

	if err != nil {
		log.Fatalf("Fallo en la llamada gRPC: %v", err)
	}

	// 4. Evaluar la respuesta del servidor central
	if !res.Valid {
		log.Printf("Acceso Denegado. Razón: %s", res.ErrorMessage)
		// Retornar 401 Unauthorized a tu frontend
		return
	}

	log.Printf("¡Login exitoso! Usuario: %s (%s)", res.Username, res.Email)
	// Permitir que el usuario acceda a tus recursos operativos
}
```

---

## 3. Ejemplo de Implementación: Backend en Java (Spring Boot)

Si la arquitectura de tu empresa utiliza **Java / Spring Boot** con interceptores, tu implementación de Cliente gRPC luciría de esta manera:

```java
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import auth.v1.AuthServiceGrpc;
import auth.v1.Auth.ValidateTokenRequest;
import auth.v1.Auth.ValidateTokenResponse;

public class AuthGrpcClient {

    private final AuthServiceGrpc.AuthServiceBlockingStub authStub;

    public AuthGrpcClient(String host, int port) {
        // En producción usa .useTransportSecurity() en lugar de .usePlaintext()
        ManagedChannel channel = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext()
                .build();

        authStub = AuthServiceGrpc.newBlockingStub(channel);
    }

    public boolean isUserVerified(String jwtToken) {
        // 1. Crear el Request de Validación
        ValidateTokenRequest request = ValidateTokenRequest.newBuilder()
                .setToken(jwtToken)
                .setTenantId("mi-empresa-tenant")
                .build();

        try {
            // 2. Enviar petición gRPC síncrona al Auth Service
            ValidateTokenResponse response = authStub.validateToken(request);

            // 3. Comprobar resultado del Auth Service
            if (response.getValid()) {
                System.out.println("Validación OK. Usuario: " + response.getEmail());
                return true;
            } else {
                System.out.println("Token denegado. Código: " + response.getErrorCode());
                return false;
            }
        } catch (Exception e) {
            System.err.println("Fallo al contactar Auth Service: " + e.getMessage());
            return false;
        }
    }

    public static void main(String[] args) {
        AuthGrpcClient client = new AuthGrpcClient("auth.tu-proveedor.com", 50051);
        client.isUserVerified("eyJhbGciOiJIUzI...tu-token");
    }
}
```

---

## 4. Mejores Prácticas: Caché Inteligente (LRU)

Si el sistema de tu empresa procesa cientos de peticiones por segundo (RPS) por usuario, llamar al servidor gRPC **en cada request** saturará la red interna y generará latencia de red innecesaria (Rendimiento Subóptimo).

La mejor práctica B2B es implementar una **Caché en Memoria de Vida Corta (Short-Lived LRU Cache)** alrededor de tu cliente gRPC.

### ¿Cómo funciona la arquitectura Avanzada?

1. Guardas en la RAM de tu API el resultado positivo de `ValidateToken` usando una Llave Compuesta: `SHA256(JWT) + TenantID`. **NUNCA uses el UserID** como llave primaria del caché, porque un usuario puede tener su sesión móvil revocada pero la de escritorio activa.
2. Le asignas un Tiempo de Vida (TTL) de **10 a 15 segundos**.
3. Las peticiones comunes (ej. `GET /perfil`) leerán la RAM (0ms de latencia) y evitarán el viaje por red al Auth-Service.
4. **Bypass de Seguridad (Validación en vivo):** Para operaciones críticas como "Transferir Dinero" o "Borrar Cuenta", omites condicionalmente el caché para forzar al gRPC a responder con el estado real del usuario en ese milisegundo.

### Ejemplo Wrapper Avanzado en Go (con `go-cache`)

```go
package authclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	pb "mi-empresa/proto/auth/v1"
)

type CachedAuthClient struct {
	grpcClient pb.AuthServiceClient
	tokenCache *cache.Cache
}

func NewCachedAuthClient(client pb.AuthServiceClient) *CachedAuthClient {
	// Caché que expira en 15 segundos, se limpia cada minuto
	return &CachedAuthClient{
		grpcClient: client,
		tokenCache: cache.New(15*time.Second, 1*time.Minute),
	}
}

// computeCacheKey previene colisiones y reduce el consumo de memoria RAM
func computeCacheKey(token, tenantID string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%s:%s", tenantID, hex.EncodeToString(hash[:]))
}

// ValidateJWT envuelve la llamada gRPC. Configura forceLiveCheck = true SÓLO para operaciones sensibles.
func (c *CachedAuthClient) ValidateJWT(ctx context.Context, tokenString, tenantID string, forceLiveCheck bool) (*pb.ValidateTokenResponse, error) {
	cacheKey := computeCacheKey(tokenString, tenantID)

	// 1. Bypass check - Intentar leer de caché local si NO es una ruta crítica
	if !forceLiveCheck {
		if cachedResp, found := c.tokenCache.Get(cacheKey); found {
			return cachedResp.(*pb.ValidateTokenResponse), nil
		}
	}

	// 2. Cache Miss / Security Bypass: Consultar por gRPC al Auth-Service
	req := &pb.ValidateTokenRequest{Token: tokenString, TenantId: tenantID}
	resp, err := c.grpcClient.ValidateToken(ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. Guardar en caché SÓLO si el token es válido
	if resp.Valid {
		c.tokenCache.Set(cacheKey, resp, cache.DefaultExpiration)
	}

	return resp, nil
}
```

### Ejemplo Wrapper Avanzado en Java (con `Caffeine`)

```java
import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;
import java.util.concurrent.TimeUnit;

public class CachedAuthClient {
    private final AuthServiceGrpc.AuthServiceBlockingStub grpcStub;

    // Caché LRU que expira entradas 15 segundos después de escribirse.
    private final Cache<String, ValidateTokenResponse> tokenCache = Caffeine.newBuilder()
        .expireAfterWrite(15, TimeUnit.SECONDS)
        .maximumSize(50_000)
        .build();

    public CachedAuthClient(AuthServiceGrpc.AuthServiceBlockingStub grpcStub) {
        this.grpcStub = grpcStub;
    }

    private String computeCacheKey(String token, String tenantId) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(token.getBytes(StandardCharsets.UTF_8));
            return tenantId + ":" + HexFormat.of().formatHex(hash);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 no disponible", e);
        }
    }

    public ValidateTokenResponse validateJwt(String tokenString, String tenantId, boolean forceLiveCheck) {
        String cacheKey = computeCacheKey(tokenString, tenantId);

        if (forceLiveCheck) {
            // Bypass de seguridad para operaciones transaccionales (fuerza gRPC real)
            tokenCache.invalidate(cacheKey);
            return callGrpc(tokenString, tenantId);
        }

        // Busca en caché, si no existe o expiró, llama a nuestro gRPC.
        return tokenCache.get(cacheKey, key -> callGrpc(tokenString, tenantId));
    }

    private ValidateTokenResponse callGrpc(String token, String tenantId) {
        ValidateTokenRequest req = ValidateTokenRequest.newBuilder()
            .setToken(token)
            .setTenantId(tenantId)
            .build();
        return grpcStub.validateToken(req);
    }
}
```

### Notas Importantes de Seguridad (Entorno B2B)

1. Estas comunicaciones inter-nodos (gRPC) siempre deberían ocurrir dentro de una **VPC privada** de red, o en su defecto, enrutadas vía Internet acompañadas siempre por **certificados mTLS** provistos por el comando de Sistema (Tenant Owner): `POST /api/v1/admin/m2m/certificates`.
2. Todo token debe considerarse revocado si el parámetro `res.Valid` retorna `false`. No dependas únicamente de la validación matemática local.
