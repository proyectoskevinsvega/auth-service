#!/bin/bash

# Configuration
KEY_DIR="./keys"
CA_NAME="VerterCloud_Root_CA"
SERVER_NAME="auth-service"
CLIENT_NAME="external-platform-01"

mkdir -p $KEY_DIR

echo "--- Generating Root CA ---"
# 1. Generate CA private key and self-signed certificate
openssl genrsa -out $KEY_DIR/ca-key.pem 4096
openssl req -new -x509 -sha256 -key $KEY_DIR/ca-key.pem -out $KEY_DIR/ca.pem -days 3650 -subj "/CN=$CA_NAME"

echo "--- Generating Server Certificate ---"
# 2. Generate Server private key
openssl genrsa -out $KEY_DIR/server-key.pem 2048
# 3. Generate Server CSR
openssl req -new -key $KEY_DIR/server-key.pem -out $KEY_DIR/server.csr -subj "/CN=$SERVER_NAME"
# 4. Sign Server Certificate with CA
openssl x509 -req -sha256 -in $KEY_DIR/server.csr -CA $KEY_DIR/ca.pem -CAkey $KEY_DIR/ca-key.pem -CAcreateserial -out $KEY_DIR/server.pem -days 365

echo "--- Generating Client Certificate ---"
# 5. Generate Client private key
openssl genrsa -out $KEY_DIR/client-key.pem 2048
# 6. Generate Client CSR
openssl req -new -key $KEY_DIR/client-key.pem -out $KEY_DIR/client.csr -subj "/CN=$CLIENT_NAME"
# 7. Sign Client Certificate with CA
openssl x509 -req -sha256 -in $KEY_DIR/client.csr -CA $KEY_DIR/ca.pem -CAkey $KEY_DIR/ca-key.pem -CAcreateserial -out $KEY_DIR/client.pem -days 365

# Cleanup CSRs
rm $KEY_DIR/*.csr
rm $KEY_DIR/*.srl

echo "--- Success! ---"
echo "Files generated in $KEY_DIR:"
echo "  - ca.pem (Root CA Certificate - Share with clients)"
echo "  - server.pem / server-key.pem (Auth Service TLS Certs)"
echo "  - client.pem / client-key.pem (Example Client Certs - Give to authorized platform)"
