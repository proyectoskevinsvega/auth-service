#!/bin/bash

# Script to generate RSA key pair for JWT signing

set -e

KEYS_DIR="./keys"

echo "Generating RSA key pair for JWT signing..."

# Create keys directory if it doesn't exist
mkdir -p "$KEYS_DIR"

# Generate private key (4096 bits for maximum security)
openssl genrsa -out "$KEYS_DIR/private.pem" 4096

# Generate public key from private key
openssl rsa -in "$KEYS_DIR/private.pem" -pubout -out "$KEYS_DIR/public.pem"

# Set proper permissions
chmod 600 "$KEYS_DIR/private.pem"
chmod 644 "$KEYS_DIR/public.pem"

echo "✅ RSA key pair generated successfully!"
echo ""
echo "Private key: $KEYS_DIR/private.pem"
echo "Public key:  $KEYS_DIR/public.pem"
echo ""
echo "⚠️  IMPORTANT:"
echo "  - Keep private.pem secure and never commit it to version control"
echo "  - Distribute public.pem to other microservices that need to validate tokens"
echo "  - The public key is also exposed via GET /auth/.well-known/jwks.json"
