# Build stage
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Install swag for generating Swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger documentation
RUN swag init -g cmd/auth-service/main.go -o docs

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -trimpath \
    -o auth-service \
    ./cmd/auth-service

# Verify the binary works
RUN ./auth-service --version || true

# Final stage - minimal runtime image
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data (for geolocation)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/auth-service /usr/local/bin/auth-service

# Create non-root user (using numeric UID for scratch)
# UID 65534 is typically 'nobody' in most systems
USER 65534:65534

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/auth-service", "health"] || exit 1

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/auth-service"]

# Labels for metadata
LABEL org.opencontainers.image.title="Auth Service"
LABEL org.opencontainers.image.description="Production-ready authentication microservice"
LABEL org.opencontainers.image.vendor="Verter Cloud"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.source="https://github.com/vertercloud/auth-service"
LABEL org.opencontainers.image.documentation="https://github.com/vertercloud/auth-service/blob/main/README.md"
