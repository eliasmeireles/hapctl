#!/bin/bash

set -e

CERT_DIR="/home/ubuntu/traefik/certs"

echo "Generating self-signed SSL certificate..."

mkdir -p "$CERT_DIR"

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$CERT_DIR/key.pem" \
    -out "$CERT_DIR/cert.pem" \
    -subj "/C=BR/ST=State/L=City/O=Development/CN=*.localhost" \
    -addext "subjectAltName=DNS:localhost,DNS:*.localhost,DNS:app1.localhost,DNS:app2.localhost,IP:127.0.0.1"

chmod 644 "$CERT_DIR/cert.pem"
chmod 600 "$CERT_DIR/key.pem"

echo "✅ SSL certificate generated at: $CERT_DIR"
echo "   Certificate: $CERT_DIR/cert.pem"
echo "   Private Key: $CERT_DIR/key.pem"
